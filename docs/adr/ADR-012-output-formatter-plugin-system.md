# ADR-012: Output Formatter Plugin System

## Status
Accepted

## Context

The initial CSV output was a single hardcoded format (35-column comma-delimited). When iCompta import compatibility was added, the options were:

1. Add format-specific branches inside every parser's write path
2. Add an `--icompta` boolean flag with conditional logic at call sites
3. Introduce a clean formatter abstraction that parsers write *to*, not *around*

Option 1 and 2 would scatter iCompta logic across 6 parser packages and make adding a third format (e.g., YNAB, Banktivity) a multi-file change.

## Decision

Implement an **OutputFormatter plugin system** using the strategy pattern:

```go
// internal/formatter/formatter.go
type OutputFormatter interface {
    WriteHeader(w io.Writer) error
    WriteTransaction(w io.Writer, t models.Transaction) error
    FormatDate(t time.Time) string
    Name() string
}
```

A `FormatterRegistry` maps string names to formatter instances. Parsers receive a formatter via the DI container and call `WriteHeader`/`WriteTransaction` — they never know which format is active.

**Registered formatters:**
- `"standard"` — `StandardFormatter`: 29-column, comma-delimited, RFC3339 dates
- `"icompta"` — `iComptaFormatter`: 10-column, semicolon-delimited, `dd.MM.yyyy` dates, matches existing iCompta CSV import plugin column mapping

**CLI surface:**
```
--format standard   # 29-column CSV (default until v1.4)
--format icompta    # iCompta-compatible semicolon format
```

The `--format` flag is a **root-level persistent flag** — it applies to all 6 parsers automatically without per-command wiring.

## Consequences

**Positive:**
- Adding a new output format = one new file implementing `OutputFormatter` + one `Register()` call
- All parsers automatically gain new formats with zero changes to their code
- iCompta format is fully isolated — changes to it cannot break standard format

**Negative:**
- Every parser must go through the formatter interface — direct `csv.Writer` usage is no longer appropriate
- The formatter must handle all 29 standard columns even if the specific output format uses fewer (unused columns are ignored during write)

## Future Work

- YNAB, Banktivity, or other personal finance app formats as additional registered formatters
- Per-column custom mapping via config (power user feature)
