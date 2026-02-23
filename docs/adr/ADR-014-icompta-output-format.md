# ADR-014: iCompta as Target Import Format

## Status
Accepted

## Context

camt-csv's output was originally a generic 29-column CSV with no specific import target. The user's actual workflow ends with importing transactions into **iCompta** (macOS personal finance app, SQLite-backed). iCompta supports CSV import via configurable `ICImportPlugin` records with explicit column mapping.

The user's iCompta database already had three import plugins configured:
- `CSV-Revolut-CHF` — Revolut CHF transactions
- `CSV-Revolut-EUR` — Revolut EUR transactions
- `CSV-RevInvest` — Revolut investment transactions

These plugins expected a specific format. Rather than creating new plugins, we should match the existing ones.

## Decision

Add `icompta` as a named output formatter (see ADR-012) that produces output matching the existing iCompta import plugins:

| Property | Value |
|----------|-------|
| Separator | Semicolon (`;`) |
| Date format | `dd.MM.yyyy` |
| Encoding | UTF-8 |
| Header row | Yes |
| Columns | 10 (Date, Description, Amount, Currency, Category, Account, Reference, Balance, Notes, Type) |

**Revolut account routing** is handled by the `Product` field in the `Transaction` model:
- `Product=Current` + `Currency=CHF` → import with `CSV-Revolut-CHF`
- `Product=Savings` + `Currency=CHF` → import with a separate "Revolut CHF Vacances" account
- `Product=Current` + `Currency=EUR` → import with `CSV-Revolut-EUR`

The routing decision is left to the user (iCompta import UI) — camt-csv populates the `Product` and `Currency` fields so the user can route correctly.

**v1.4 change:** `icompta` becomes the default `--format` value. The typical user always wants iCompta output; `--format standard` is the explicit override.

## Consequences

**Positive:**
- Zero iCompta configuration changes required — existing import plugins work immediately
- Semicolon separator avoids conflicts with European decimal notation in amounts
- `dd.MM.yyyy` matches Swiss date conventions used in source bank statements

**Negative:**
- `icompta` format is tied to the user's specific import plugin column mapping — it is not a generic "iCompta format" that works for all iCompta users
- A different user with different iCompta plugins would need to configure their own formatter

## Future Work

- Parametric formatter config (column mapping driven by a YAML file) for users with different iCompta plugin setups
- CAMT and PDF parsers should also populate `Product` field for completeness
