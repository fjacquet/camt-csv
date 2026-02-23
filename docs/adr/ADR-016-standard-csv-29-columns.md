# ADR-016: Standard CSV Format Trimmed to 29 Columns

## Status
Accepted

## Context

The original `StandardFormatter` output 35 columns per transaction. Analysis of all downstream consumers (iCompta import, manual review in spreadsheets) revealed that 6 columns were never used:

| Removed Column | Reason |
|----------------|--------|
| `BookkeepingNumber` | Internal accounting reference never populated by any parser |
| `IsDebit` | Redundant — derivable from `Amount` sign |
| `Debit` | Redundant — split of `Amount` never used in iCompta |
| `Credit` | Redundant — split of `Amount` never used in iCompta |
| `Recipient` | Duplicate of `PartyName` in practice |
| `AmountTax` | Tax breakdown never populated; always zero |

The 35-column format was also wider than necessary for manual CSV inspection, making it harder to work with in spreadsheets.

## Decision

Trim `StandardFormatter` from 35 to **29 columns** by removing the 6 unused columns listed above.

The column set is now:

```
Date, ValueDate, Amount, Currency, Description, PartyName, PartyAccount,
PartyBankCode, Reference, MandateReference, CreditorID, TransactionID,
EndToEndID, Purpose, Category, Confidence, Account, BankAccount,
TransactionType, Product, OriginalAmount, OriginalCurrency, ExchangeRate,
FeeAmount, Balance, Notes, Status, Source, Format
```

All parser tests and integration tests were updated for the 29-column schema. The iCompta formatter (ADR-014) is unaffected — it has its own 10-column schema.

## Consequences

**Positive:**
- CSV files are ~17% narrower — easier to inspect manually
- Removes confusion from columns that always contained empty values
- All parsers are consistent: no parser produces the removed columns

**Negative:**
- **Breaking change** for any downstream tooling relying on the 35-column layout or column-index parsing
- Users with existing 35-column CSV imports need to update their import configurations

## Compatibility Note

The iCompta import plugins (`CSV-Revolut-CHF`, etc.) use the `icompta` formatter (semicolon, 10-column) — they are **not affected** by this change. The 29-column format is only used when `--format standard` is explicitly passed.

## Future Work

- Further column reduction possible if additional columns prove unused across all parsers
- Column names could be configurable for users mapping to legacy tooling
