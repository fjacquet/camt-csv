# ADR-020: Preserve Parser-Internal Categories in Categorization Helper

## Status

**ACCEPTED** - Implemented in v2.3.2

## Context

The shared `ProcessTransactionsWithCategorizationStats` helper always ran the external categorizer (YAML lookup → semantic → AI) for every transaction, regardless of whether the parser had already determined the correct category through domain-specific logic.

For investment parsers (Selma), categories like `Investissements`, `Revenus Financiers`, `Impôts`, and `Frais Bancaires` are deterministic based on transaction type (`trade`, `dividend`, `withholding_tax`, `selma_fee`). Running those through the external categorizer was wasteful and destructive — since `PartyName` was empty, the helper defaulted to `Uncategorized`, overwriting the correct internally-set category.

## Decision

In `ProcessTransactionsWithCategorizationStats`, skip external categorization when `tx.Category` is already set to a non-empty, non-`Uncategorized` value:

```go
if tx.Category != "" && tx.Category != models.CategoryUncategorized {
    stats.IncrementSuccessful()
    continue
}
```

## Rationale

- **Correctness**: Parser-internal logic has domain knowledge the external categorizer lacks (e.g., `withholding_tax` is always `Impôts`)
- **Cost**: Skipping AI calls for deterministic categories reduces API usage
- **Separation of concerns**: Parsers own their domain categorization; the shared helper handles the general case

## Consequences

### Positive

- Investment transactions get correct categories without AI calls
- External categorizer only runs when genuinely needed
- Backward compatible — normal parsers that don't pre-set categories are unaffected

### Negative

- If a parser sets a wrong category internally, it cannot be overridden by the external categorizer — the internal logic must be correct
