# Requirements: camt-csv

**Defined:** 2026-02-16
**Core Value:** Reliable, maintainable financial data conversion with intelligent categorization.

## v1.3 Requirements

Requirements for Standard CSV Trim. Each maps to roadmap phases.

### CSV Format

- [ ] **CSV-01**: Standard CSV format removes BookkeepingNumber column (never populated by parsers)
- [ ] **CSV-02**: Standard CSV format removes IsDebit/DebitFlag column (redundant with CreditDebit)
- [ ] **CSV-03**: Standard CSV format removes Debit column (derived from Amount + CreditDebit)
- [ ] **CSV-04**: Standard CSV format removes Credit column (derived from Amount + CreditDebit)
- [ ] **CSV-05**: Standard CSV format removes Recipient column (redundant with Name/PartyName)
- [ ] **CSV-06**: Standard CSV format removes AmountTax column (never populated by parsers)

### Integrity

- [ ] **INT-01**: StandardFormatter header reflects 29 columns after removal
- [ ] **INT-02**: MarshalCSV/UnmarshalCSV updated for 29-column format
- [ ] **INT-03**: All parser tests pass with new column count
- [ ] **INT-04**: Integration tests (cross-parser consistency) pass
- [ ] **INT-05**: iCompta formatter remains unchanged (10 columns, semicolon)

## Future Requirements

None — this is a focused cleanup milestone.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Removing Transaction struct fields | Fields may still be used internally; only CSV output changes |
| Per-parser column sets | Different scope; each parser would need its own formatter |
| iCompta format changes | iCompta format is already minimal at 10 columns |
| Removing Payee/Payer fields | Internal fields not in CSV output, still used for Name derivation |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CSV-01 | — | Pending |
| CSV-02 | — | Pending |
| CSV-03 | — | Pending |
| CSV-04 | — | Pending |
| CSV-05 | — | Pending |
| CSV-06 | — | Pending |
| INT-01 | — | Pending |
| INT-02 | — | Pending |
| INT-03 | — | Pending |
| INT-04 | — | Pending |
| INT-05 | — | Pending |

**Coverage:**
- v1.3 requirements: 11 total
- Mapped to phases: 0
- Unmapped: 11 ⚠️

---
*Requirements defined: 2026-02-16*
*Last updated: 2026-02-16 after initial definition*
