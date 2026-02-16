---
phase: 10-csv-format-trim
plan: 01
subsystem: formatter
tags: [csv-format, data-model, breaking-change]

dependency_graph:
  requires:
    - internal/models/transaction.go (MarshalCSV/UnmarshalCSV)
    - internal/formatter/standard.go (Header method)
  provides:
    - 29-column CSV format (reduced from 35)
  affects:
    - All parsers using StandardFormatter
    - CSV export functionality
    - CSV import functionality (if implemented)

tech_stack:
  added: []
  patterns:
    - CSV serialization with reduced field set
    - Backward incompatible format change

key_files:
  created: []
  modified:
    - internal/formatter/standard.go (29-column header)
    - internal/models/transaction.go (MarshalCSV/UnmarshalCSV updated)
    - internal/formatter/formatter_test.go (test updates)
    - internal/models/transaction_test.go (test updates)
    - internal/models/builder_test.go (test updates)

decisions:
  - title: "Remove 6 redundant/dead fields from CSV format"
    rationale: "BookkeepingNumber, AmountTax never populated; IsDebit redundant with CreditDebit; Debit/Credit derived from Amount+CreditDebit; Recipient redundant with Name/PartyName"
    impact: "Breaking change - old 35-column CSV files cannot be read by new code without conversion"
    alternatives_considered: "Keep all fields for backward compatibility"
    chosen: "Remove fields to simplify format"

  - title: "Keep iCompta formatter unchanged"
    rationale: "iCompta formatter is user-facing format for specific app, should remain stable"
    impact: "Users can continue importing to iCompta without changes"

metrics:
  duration: 237
  completed_at: "2026-02-16T12:11:10Z"
---

# Phase 10 Plan 01: CSV Format Trim Summary

**Reduced StandardFormatter CSV output from 35 columns to 29 by removing 6 redundant/dead fields (BookkeepingNumber, IsDebit, Debit, Credit, Recipient, AmountTax)**

## Tasks Completed

| Task | Description | Commit | Files Modified |
|------|-------------|--------|----------------|
| 1 | Update StandardFormatter header to 29 columns | 9d65eb7 | internal/formatter/standard.go |
| 2 | Update Transaction.MarshalCSV to output 29 columns | d906744 | internal/models/transaction.go |
| 3 | Update Transaction.UnmarshalCSV to parse 29 columns | ea41955 | internal/models/transaction.go |
| 4 | Update formatter tests for 29-column format | b517e79 | internal/formatter/formatter_test.go |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed model tests for 29-column format**
- **Found during:** Task 4 verification
- **Issue:** TestTransaction_DateFormattingIndirect and TestTransaction_CSVAccuracy tests were using old 35-column format assumptions
- **Fix:**
  - Updated TestTransaction_DateFormattingIndirect to use new column indices (Date at index 1, ValueDate at index 2)
  - Updated CSV test data from 35 columns to 29 columns
  - Removed checks for removed fields (BookkeepingNumber, DebitFlag) in TestTransaction_CSVAccuracy
- **Files modified:** internal/models/transaction_test.go, internal/models/builder_test.go
- **Commit:** 47f57b4

## Removed Columns

The following 6 columns were removed from the CSV format:

| Field | Original Index | Reason for Removal |
|-------|---------------|-------------------|
| BookkeepingNumber | 0 | Never populated by any parser |
| IsDebit | 11 | Redundant with CreditDebit field |
| Debit | 12 | Derived from Amount + CreditDebit |
| Credit | 13 | Derived from Amount + CreditDebit |
| Recipient | 19 | Redundant with Name/PartyName |
| AmountTax | 17 | Never populated by any parser |

## New Column Order (29 columns)

1. Status
2. Date
3. ValueDate
4. Name
5. PartyName
6. PartyIBAN
7. Description
8. RemittanceInfo
9. Amount
10. CreditDebit
11. Currency
12. Product
13. AmountExclTax
14. TaxRate
15. InvestmentType
16. Number
17. Category
18. Type
19. Fund
20. NumberOfShares
21. Fees
22. IBAN
23. EntryReference
24. Reference
25. AccountServicer
26. BankTxCode
27. OriginalCurrency
28. OriginalAmount
29. ExchangeRate

## Test Results

### Formatter Tests
```
go test ./internal/formatter/
PASS
ok  	fjacquet/camt-csv/internal/formatter	0.360s
```

All StandardFormatter tests pass with 29-column expectations.
All iComptaFormatter tests pass unchanged (10 columns, semicolon delimiter).

### Model Tests
```
go test ./internal/models/
PASS
ok  	fjacquet/camt-csv/internal/models	0.289s
```

All model tests pass including CSV marshaling/unmarshaling tests.

### Sample Conversion Verification

**StandardFormatter (29 columns):**
```bash
go run main.go camt --input samples/camt053/camt53-47.xml --output /tmp/test-29col.csv
head -n 1 /tmp/test-29col.csv | awk -F',' '{print NF}'
# Output: 29
```

Header verified:
```
Status,Date,ValueDate,Name,PartyName,PartyIBAN,Description,RemittanceInfo,Amount,CreditDebit,Currency,Product,AmountExclTax,TaxRate,InvestmentType,Number,Category,Type,Fund,NumberOfShares,Fees,IBAN,EntryReference,Reference,AccountServicer,BankTxCode,OriginalCurrency,OriginalAmount,ExchangeRate
```

**iComptaFormatter (10 columns - unchanged):**
```bash
go run main.go camt --input samples/camt053/camt53-47.xml --output /tmp/test-icompta.csv --format icompta
head -n 1 /tmp/test-icompta.csv | awk -F';' '{print NF}'
# Output: 10
```

## Decisions Made

1. **Keep Update methods in MarshalCSV**: Although removed fields (Recipient, Debit, Credit) are no longer serialized, the Update methods (UpdateNameFromParties, UpdateRecipientFromPayee, UpdateDebitCreditAmounts) are still called for internal consistency.

2. **Breaking change accepted**: This is a breaking change - old 35-column CSV files cannot be read by new UnmarshalCSV without conversion. This is acceptable for v1.3 as the focus is on simplifying the format going forward.

3. **iCompta formatter protection**: Explicitly verified that iCompta formatter remains unchanged, protecting users who rely on this format for imports.

## Impact Assessment

### Positive
- Simplified CSV format (17% fewer columns)
- Eliminated dead fields that were never used
- Reduced file size and parsing overhead
- Cleaner data model

### Neutral
- Tests updated to reflect new format
- Documentation needs update (handled in separate docs task)

### Breaking
- Existing 35-column CSV files cannot be imported with new code
- External tools parsing the CSV format will need updates
- Migration path: convert old CSV files using v1.2 code before upgrading

## Self-Check: PASSED

### Created Files
No new files created.

### Modified Files
- ✓ internal/formatter/standard.go (exists, modified)
- ✓ internal/models/transaction.go (exists, modified)
- ✓ internal/formatter/formatter_test.go (exists, modified)
- ✓ internal/models/transaction_test.go (exists, modified)
- ✓ internal/models/builder_test.go (exists, modified)

### Commits
- ✓ 9d65eb7: feat(10-01): reduce StandardFormatter header to 29 columns
- ✓ d906744: feat(10-01): reduce MarshalCSV output to 29 columns
- ✓ ea41955: feat(10-01): update UnmarshalCSV to parse 29 columns
- ✓ b517e79: test(10-01): update formatter tests for 29-column format
- ✓ 47f57b4: fix(10-01): update model tests for 29-column CSV format

All commits verified in git log.
