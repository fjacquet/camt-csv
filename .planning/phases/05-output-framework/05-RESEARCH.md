# Phase 5: Output Framework - Research

**Researched:** 2026-02-15
**Domain:** Financial CSV output formatting with iCompta compatibility
**Confidence:** HIGH (source: codebase inspection + schema analysis)

## Summary

Phase 5 standardizes output across all parsers (CAMT, PDF, Revolut, Selma, Debit) into two formats: **standard** (35-column backward-compatible) and **icompta** (iCompta-import-ready). The work implements an output formatter plugin system using the composition pattern already established in the codebase, adds `--format` and `--date-format` CLI flags, and maintains backward compatibility while supporting iCompta's split-based transaction model.

**Primary recommendation:** Implement OutputFormatter interface in `internal/formatter/` with StandardFormatter, iComptaFormatter, and registry pattern. Use existing CSV infrastructure (`common.WriteTransactionsToCSVWithLogger`) as base, accepting formatter parameter. No changes to Transaction model needed.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/gocarina/gocsv` | v0.0.x | CSV marshaling/unmarshaling | Already in use; handles struct→CSV mapping |
| `encoding/csv` | stdlib | CSV writing with custom delimiters | stdlib; used for semicolon separator support (iCompta requirement) |
| `time` | stdlib | Date formatting | Built-in; handles multiple date formats |
| `github.com/shopspring/decimal` | v1.x | Precise currency calculations | Already in use for all amount fields |
| `cobra` | v1.x | CLI framework | Existing CLI infrastructure |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/sirupsen/logrus` | v1.x | Structured logging | Already configured for all CLI output |
| `github.com/spf13/viper` | v1.x | Configuration management | Already handles config hierarchy |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Formatter interface pattern | Template method in parsers | Formatters are more composable; template method would require modifying all parsers |
| Custom date format in config | Only hardcoded formats | Config flexibility matches OUT-04 requirement for user control |
| Semicolon separator (iCompta) | Always comma | iCompta plugins expect semicolon; must be configurable per format |

## Architecture Patterns

### Recommended Project Structure

**New directories and files:**
```
internal/formatter/              # NEW
├── formatter.go                 # OutputFormatter interface + registry
├── standard.go                  # StandardFormatter (35-column)
└── icompta.go                   # iComptaFormatter (iCompta format)

internal/common/
├── csv.go                       # MODIFIED: Accept formatter parameter
└── [existing]

cmd/*/                           # MODIFIED: Add --format and --date-format flags
└── convert.go
```

### Pattern 1: OutputFormatter Interface

**What:** Segregated interface allowing pluggable output formatters.

**When to use:** All CSV output operations; formatters encapsulate format-specific logic.

**Example:**
```go
// Source: internal/formatter/formatter.go
package formatter

import "fjacquet/camt-csv/internal/models"

// OutputFormatter defines a pluggable output format for transactions
type OutputFormatter interface {
    // Header returns the CSV column names for this format
    Header() []string

    // Format converts transactions to rows ready for CSV writing
    // Each inner slice is a row of strings
    Format(transactions []models.Transaction) ([][]string, error)
}

// FormatterRegistry manages available formatters
type FormatterRegistry struct {
    formatters map[string]OutputFormatter
}

func NewFormatterRegistry() *FormatterRegistry {
    return &FormatterRegistry{
        formatters: map[string]OutputFormatter{
            "standard":  &StandardFormatter{},
            "icompta":   &iComptaFormatter{},
        },
    }
}

func (r *FormatterRegistry) Get(name string) (OutputFormatter, error) {
    if f, ok := r.formatters[name]; ok {
        return f, nil
    }
    return nil, fmt.Errorf("unknown format: %s", name)
}

func (r *FormatterRegistry) Register(name string, formatter OutputFormatter) {
    r.formatters[name] = formatter
}
```

### Pattern 2: StandardFormatter (Backward Compatible)

**What:** 35-column format matching current `common.WriteTransactionsToCSV()` output.

**When to use:** Default format; maintains backward compatibility with existing workflows.

**Key fields in order:**
```
BookkeepingNumber, Status, Date, ValueDate, Name, PartyName, PartyIBAN,
Description, RemittanceInfo, Amount, CreditDebit, IsDebit, Debit, Credit, Currency,
AmountExclTax, AmountTax, TaxRate, Recipient, InvestmentType, Number, Category,
Type, Fund, NumberOfShares, Fees, IBAN, EntryReference, Reference,
AccountServicer, BankTxCode, OriginalCurrency, OriginalAmount, ExchangeRate
```

**Date format:** DD.MM.YYYY (hardcoded in Transaction.formatDateForCSV)

**CSV delimiter:** Configurable via config, default comma (from `common.Delimiter`)

**Example:**
```go
// Source: internal/formatter/standard.go
package formatter

import "fjacquet/camt-csv/internal/models"

type StandardFormatter struct{}

func (f *StandardFormatter) Header() []string {
    return []string{
        "BookkeepingNumber", "Status", "Date", "ValueDate", "Name", "PartyName", "PartyIBAN",
        "Description", "RemittanceInfo", "Amount", "CreditDebit", "IsDebit", "Debit", "Credit", "Currency",
        "AmountExclTax", "AmountTax", "TaxRate", "Recipient", "InvestmentType", "Number", "Category",
        "Type", "Fund", "NumberOfShares", "Fees", "IBAN", "EntryReference", "Reference",
        "AccountServicer", "BankTxCode", "OriginalCurrency", "OriginalAmount", "ExchangeRate",
    }
}

func (f *StandardFormatter) Format(transactions []models.Transaction) ([][]string, error) {
    var rows [][]string
    for _, tx := range transactions {
        record, err := tx.MarshalCSV()
        if err != nil {
            return nil, err
        }
        rows = append(rows, record)
    }
    return rows, nil
}
```

### Pattern 3: iComptaFormatter (Plugin-Compatible)

**What:** iCompta-specific format matching user's import plugins (CSV-Revolut-CHF, CSV-Revolut-EUR, CSV-RevInvest).

**When to use:** When importing into iCompta; produces denormalized CSV matching plugin expectations.

**iCompta column mapping (from import plugins reference):**

iCompta splits transactions into ICTransaction (header) + ICTransactionSplit (category detail). Our denormalized format includes both on one row:

| iCompta Schema | Our Column | Source Field | Example |
|---|---|---|---|
| ICTransaction.date | Date | tx.Date | 15.02.2026 |
| ICTransaction.name | Name | tx.Name / tx.PartyName | Vendor ABC |
| ICTransaction.amount | Amount | tx.Amount | 100.00 |
| ICTransaction.comment | Description | tx.Description | Restaurant visit |
| ICTransaction.status | Status | tx.Status | "cleared" (mapped from CAMT status) |
| ICTransaction.payee | Payee | tx.PartyName | (optional, mapped to Name) |
| ICTransaction.type | Type | tx.Type | CARD_PAYMENT, TRANSFER, etc. |
| ICTransactionSplit.category | Category | tx.Category | Food, Transport, etc. |
| ICTransactionSplit.amount | SplitAmount | tx.Amount | (same as Amount in v1) |
| ICTransactionSplit.amountWithoutTaxes | SplitAmountExclTax | tx.AmountExclTax | 100.00 |
| ICTransactionSplit.taxesRate | SplitTaxRate | tx.TaxRate | 0.00 |

**Key constraints from user's iCompta plugins:**
- **CSV-Revolut-CHF**: Semicolon separator (`;`), `dd.MM.yyyy` date format
- **CSV-Revolut-EUR**: Semicolon separator (`;`), `dd.MM.yyyy` date format
- **CSV-RevInvest**: Semicolon separator (`;`), `dd.MM.yyyy` date format
- **All plugins**: Expect category field for split mapping

**Output header (10 columns minimum):**
```
Date;Name;Amount;Description;Status;Category;SplitAmount;SplitAmountExclTax;SplitTaxRate;Type
```

**Denormalized row example:**
```
15.02.2026;Vendor ABC;100.00;Restaurant;cleared;Food;100.00;100.00;0.00;CARD_PAYMENT
```

**Implementation:**
```go
// Source: internal/formatter/icompta.go
package formatter

import (
    "fmt"
    "fjacquet/camt-csv/internal/models"
)

type iComptaFormatter struct{}

func (f *iComptaFormatter) Header() []string {
    return []string{
        "Date", "Name", "Amount", "Description", "Status", "Category",
        "SplitAmount", "SplitAmountExclTax", "SplitTaxRate", "Type",
    }
}

func (f *iComptaFormatter) Format(transactions []models.Transaction) ([][]string, error) {
    var rows [][]string
    for _, tx := range transactions {
        // Map status to iCompta values
        status := "cleared"
        if tx.Status == "PENDING" {
            status = "pending"
        } else if tx.Status == "REVERTED" {
            status = "reverted"
        }

        row := []string{
            formatDateForIConta(tx.Date),        // dd.MM.yyyy
            tx.Name,
            tx.Amount.StringFixed(2),
            tx.Description,
            status,
            tx.Category,
            tx.Amount.StringFixed(2),            // SplitAmount = Amount (v1 single split)
            tx.AmountExclTax.StringFixed(2),
            tx.TaxRate.StringFixed(2),
            tx.Type,
        }
        rows = append(rows, row)
    }
    return rows, nil
}

func formatDateForICompta(date time.Time) string {
    if date.IsZero() {
        return ""
    }
    return date.Format("02.01.2006")  // dd.MM.yyyy
}
```

### Pattern 4: Integration with Existing CSV Writer

**What:** Extend `WriteTransactionsToCSVWithLogger()` to accept optional formatter and format-specific settings.

**When to use:** All CSV export calls; provides single point of control for output formatting.

**Current signature:**
```go
func WriteTransactionsToCSVWithLogger(
    transactions []models.Transaction,
    csvFile string,
    logger logging.Logger,
) error
```

**New signature:**
```go
func WriteTransactionsToCSVWithFormatter(
    transactions []models.Transaction,
    csvFile string,
    logger logging.Logger,
    formatter formatter.OutputFormatter,
    delimiter rune,
    dateFormat string, // Optional: override default date format
) error
```

**Implementation approach:**
1. Keep existing function for backward compatibility
2. Create new function that accepts formatter
3. Use formatter.Header() and formatter.Format()
4. Handle format-specific delimiters (`,` for standard, `;` for iCompta)
5. Log format selection

**Code example:**
```go
// Source: internal/common/csv.go (modified)
import "fjacquet/camt-csv/internal/formatter"

func WriteTransactionsToCSVWithFormatter(
    transactions []models.Transaction,
    csvFile string,
    logger logging.Logger,
    fmt formatter.OutputFormatter,
    delimiter rune,
) error {
    if logger == nil {
        logger = logging.NewLogrusAdapter("info", "text")
    }

    // Prepare transactions
    for i := range transactions {
        transactions[i].UpdateNameFromParties()
        transactions[i].UpdateRecipientFromPayee()
        transactions[i].UpdateDebitCreditAmounts()
    }

    // Format transactions
    rows, err := fmt.Format(transactions)
    if err != nil {
        return fmt.Errorf("formatting failed: %w", err)
    }

    // Write CSV with format-specific delimiter
    file, err := os.Create(csvFile)
    if err != nil {
        return fmt.Errorf("create file: %w", err)
    }
    defer file.Close()

    csvWriter := csv.NewWriter(file)
    csvWriter.Comma = delimiter

    if err := csvWriter.Write(fmt.Header()); err != nil {
        return fmt.Errorf("write header: %w", err)
    }

    for _, row := range rows {
        if err := csvWriter.Write(row); err != nil {
            return fmt.Errorf("write row: %w", err)
        }
    }

    csvWriter.Flush()
    return csvWriter.Error()
}
```

### Anti-Patterns to Avoid

- **Hardcoding formatters in parsers:** Don't make each parser handle output formatting. Use composition instead.
- **Modifying Transaction model for format:** iCompta compatibility should not add fields to Transaction. Use formatter to project existing fields.
- **Format-specific logic in ConvertToCSV():** Keep parsers format-agnostic; pass formatter as parameter.
- **Ignoring delimiters:** iCompta uses semicolon; standard uses comma. Make this configurable per formatter.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CSV writing with different delimiters | Custom CSV formatter | `encoding/csv` + OutputFormatter interface | stdlib handles quoting, escaping, buffering |
| Date format conversion | String replacement with regex | `time.Format()` with Go layout strings | time package handles all format combinations correctly |
| Field mapping for different schemas | Parser-specific export code | OutputFormatter interface with registry | Composition is testable, extensible, DRY |
| iCompta integration logic | Custom denormalization code | OutputFormatter that projects Transaction fields | Keeps concern in one place; easy to maintain plugin compatibility |

**Key insight:** OutputFormatter is the abstraction that prevents format-specific logic from leaking into parsers, common utilities, or CLI commands.

## Common Pitfalls

### Pitfall 1: Formatter Only Exports; Doesn't Validate Format Compatibility

**What goes wrong:** User specifies `--format icompta` but category field is empty, resulting in imports without category assignments.

**Why it happens:** Formatter.Format() doesn't validate that required fields are populated. It just reads what's in Transaction.

**How to avoid:** Add validation in Format():
```go
func (f *iComptaFormatter) Format(transactions []models.Transaction) ([][]string, error) {
    for _, tx := range transactions {
        if tx.Category == "" {
            // Log warning; fill with fallback
            logger.Warnf("Transaction %s has no category", tx.Name)
            tx.Category = "Uncategorized"
        }
    }
    // ...
}
```

**Warning signs:** iCompta imports with no category assignments; user asks why categories disappeared.

### Pitfall 2: Date Format Configuration Applied Inconsistently

**What goes wrong:** Config specifies `--date-format YYYY-MM-DD` but transaction dates still output as DD.MM.YYYY.

**Why it happens:** Date formatting is currently hardcoded in Transaction.formatDateForCSV(). Config changes aren't applied to formatter.

**How to avoid:** Pass dateFormat parameter to formatter:
```go
type iComptaFormatter struct {
    dateFormat string  // e.g., "02.01.2006" for dd.MM.yyyy
}

func (f *iComptaFormatter) formatDate(date time.Time) string {
    return date.Format(f.dateFormat)
}
```

**Warning signs:** User sets `--date-format` flag but it has no effect; dates still in hardcoded format.

### Pitfall 3: Delimiter Configuration Silently Fails

**What goes wrong:** iCompta importer expects semicolon; standard formatter outputs commas.

**Why it happens:** Delimiter is baked into both StandardFormatter and CSV writer separately. If they disagree, CSV is malformed.

**How to avoid:** Formatter declares its delimiter preference:
```go
type OutputFormatter interface {
    Header() []string
    Format(transactions []models.Transaction) ([][]string, error)
    Delimiter() rune  // NEW: Return preferred delimiter
}
```

Then CSV writer uses formatter.Delimiter():
```go
csvWriter.Comma = formatter.Delimiter()
```

**Warning signs:** iCompta can't parse the CSV; columns are misaligned.

### Pitfall 4: Status Mapping Is Lossy

**What goes wrong:** CAMT status values like "BOOK", "PDNG", "RCVD" are mapped to iCompta's "cleared", "pending" but meaning changes.

**Why it happens:** iCompta's status field has specific semantics; arbitrary mapping loses original CAMT data.

**How to avoid:**
1. Document explicit mapping in formatter comments
2. Preserve original status in separate field or comment
3. Add validation that mapped status is reasonable

**Mapping recommended:**
```go
// CAMT status → iCompta status mapping
var statusMap = map[string]string{
    "BOOK":    "cleared",      // Booked = cleared in iCompta
    "PDNG":    "pending",      // Pending = pending
    "RCVD":    "cleared",      // Received = cleared (settled)
    "REVD":    "reverted",     // Reverted = reverted
    "CANC":    "reverted",     // Cancelled ≈ reverted (conservative)
    "":        "cleared",      // Default to cleared
}
```

**Warning signs:** User says iCompta reconciliation shows wrong statuses; importer can't match.

### Pitfall 5: Amount Formatting Loses Precision

**What goes wrong:** Amounts with many decimal places (CHF/EUR exchange rates) get truncated or rounded differently than original.

**Why it happens:** Different formatters use different rounding rules or significant figures.

**How to avoid:** Use decimal.Decimal.StringFixed(2) consistently:
```go
tx.Amount.StringFixed(2)      // Always 2 decimals
tx.TaxRate.StringFixed(2)     // Consistent formatting
```

Never convert to float64 for formatting; always use StringFixed().

**Warning signs:** Reconciliation in iCompta shows tiny discrepancies; audit trail can't match.

## Code Examples

Verified patterns from official sources and codebase inspection:

### Example 1: Create Formatter Registry in Container

```go
// Source: internal/container/container.go (to add)
type Container struct {
    // ... existing fields ...
    formatterRegistry *formatter.FormatterRegistry
}

func (c *Container) GetFormatterRegistry() *formatter.FormatterRegistry {
    if c.formatterRegistry == nil {
        c.formatterRegistry = formatter.NewFormatterRegistry()
    }
    return c.formatterRegistry
}

// Usage in CLI:
registry := container.GetFormatterRegistry()
fmt, err := registry.Get("icompta")
if err != nil {
    log.Fatal(err)
}
```

### Example 2: Update ConvertToCSV to Accept Formatter

```go
// Source: internal/parser/base.go or individual adapters
func (a *Adapter) ConvertToCSV(
    ctx context.Context,
    inputFile, outputFile string,
    formatter formatter.OutputFormatter,
    delimiter rune,
) error {
    transactions, err := a.ParseFromFile(ctx, inputFile)
    if err != nil {
        return err
    }

    return common.WriteTransactionsToCSVWithFormatter(
        transactions,
        outputFile,
        a.GetLogger(),
        formatter,
        delimiter,
    )
}
```

### Example 3: Add --format Flag to CLI Command

```bash
# Source: cmd/camt/convert.go or cmd/revolut/convert.go

var convertCmd = &cobra.Command{
    Use:   "convert --input file.xml --output out.csv --format icompta",
    Short: "Convert to CSV with format selection",
    RunE: func(cmd *cobra.Command, args []string) error {
        format, _ := cmd.Flags().GetString("format")
        inputFile, _ := cmd.Flags().GetString("input")
        outputFile, _ := cmd.Flags().GetString("output")

        registry := container.GetFormatterRegistry()
        formatter, err := registry.Get(format)
        if err != nil {
            return err
        }

        // Get delimiter based on format
        var delimiter rune = ','
        if format == "icompta" {
            delimiter = ';'
        }

        return adapter.ConvertToCSV(ctx, inputFile, outputFile, formatter, delimiter)
    },
}

func init() {
    convertCmd.Flags().StringP("format", "f", "standard",
        "Output format: standard (35-col) or icompta (iCompta-compatible)")
    convertCmd.Flags().StringP("date-format", "d", "DD.MM.YYYY",
        "Date format: DD.MM.YYYY, YYYY-MM-DD, etc.")
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Parser-specific CSV output (e.g., RevolutFormatter in parser) | OutputFormatter interface in separate package | v1.2 Phase 5 | Enables format abstraction without modifying parsers |
| Hardcoded comma delimiter | Configurable via formatter and config | v1.2 Phase 5 | Supports iCompta's semicolon requirement |
| 4-column Revolut output | 35-column standard format with formatter option | v1.2 Phase 6 | Unified format across all parsers; optional iCompta mode |
| iCompta logic sprinkled in categorizer | Dedicated iComptaFormatter in formatter package | v1.2 Phase 5 | Single responsibility; easier to test and maintain |

**Deprecated/outdated:**
- **Direct iCompta database writes:** Use CSV import via iCompta plugins instead (safer, non-fragile)
- **Custom denormalization in parsers:** Move to OutputFormatter pattern (reduces parser complexity)

## Open Questions

1. **Should formatters modify Transaction objects?**
   - What we know: Currently Format() iterates over transactions, doesn't mutate them
   - What's unclear: Should we populate missing fields (e.g., Category="Uncategorized")? Or assume caller pre-processes?
   - Recommendation: Formatters should NOT mutate. Caller (parser) is responsible for ensuring required fields are populated before passing to formatter.

2. **How should we handle split transactions in iCompta format (v2 enhancement)?**
   - What we know: iCompta splits are ICTransactionSplit rows with same transaction ID
   - What's unclear: v1 has no split support; should we reserve room for future splits?
   - Recommendation: Current single-split denormalized format is fine for v1. Future v2 iComptaFormatter can output multiple rows per transaction for splits.

3. **Should date format be per-formatter or global config?**
   - What we know: iCompta requires dd.MM.yyyy; standard format might prefer YYYY-MM-DD
   - What's unclear: Should user set one global `--date-format` or should each format have its own?
   - Recommendation: Global `--date-format` config with formatter override capability. iCompta default is dd.MM.yyyy, standard default is DD.MM.YYYY (backward compatible).

4. **How to validate CSV before writing?**
   - What we know: Write happens after parsing and categorization
   - What's unclear: Should we add a validation step to ensure formatters don't produce invalid output?
   - Recommendation: Add ValidationFormatter wrapper that checks output before write (optional, for safety)

## Sources

### Primary (HIGH confidence)
- **Codebase inspection** - internal/parser/parser.go, internal/models/transaction.go, internal/common/csv.go
- **iCompta schema** - .planning/reference/icompta-schema.sql
- **iCompta import plugins** - .planning/reference/icompta-import-plugins.txt (user's 9 existing plugins)
- **Architecture research** - .planning/research/ARCHITECTURE.md (v1.2 formatter pattern documented)

### Secondary (MEDIUM confidence)
- **Go CSV stdlib** - encoding/csv documentation (standard library; HIGH confidence)
- **time.Format() layout** - Go time package documentation (standard library; HIGH confidence)
- **existing codebase patterns** - gocsv integration, DI container, interface segregation (verified in codebase)

### Tertiary (LOW confidence)
- **iCompta plugin behavior** - Inferred from import plugin JSON mappings; not tested with live iCompta

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** - All libraries already in use; no new dependencies needed
- Architecture: **HIGH** - OutputFormatter interface pattern is established in codebase (parser.FullParser pattern confirmed)
- iCompta format details: **MEDIUM-HIGH** - Schema documented, plugins documented, but actual CSV import untested
- Date formatting: **HIGH** - time.Format() is stdlib; behavior well-defined
- Pitfalls: **MEDIUM** - Based on financial software experience; some specific to this codebase

**Research date:** 2026-02-15
**Valid until:** 2026-03-15 (30 days; architecture stable but iCompta testing may surface new constraints)

**Key unknowns requiring validation:**
1. Actual iCompta CSV import behavior (plugins reference exists but import not tested)
2. Whether iCompta plugins handle missing category gracefully
3. Performance of OutputFormatter.Format() on large transactions (100K+)
4. Whether iCompta splits integration happens in Phase 5 or Phase 6
