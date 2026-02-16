# Phase 6: Revolut Parsers Overhaul - Research

**Researched:** 2026-02-16
**Domain:** Revolut transaction parsing with semantic type identification and standardized CSV output
**Confidence:** HIGH (source: codebase inspection, real data profile analysis, Phase 5 output framework already delivered)

## Summary

Phase 6 upgrades Revolut parsers (both standard and investment) to understand transaction semantics and output the standardized 35-column CSV format established by other parsers (CAMT, PDF, Selma, Debit). The work centers on three tasks: (1) enhance transaction type identification and tagging in the regular Revolut parser, (2) add Product field mapping to Transaction model and output, (3) handle REVERTED/PENDING transaction states, and (4) ensure both Revolut parsers integrate with the Phase 5 output formatter system for format-flexible CSV output (standard 35-column, iCompta 10-column).

The regular Revolut parser currently outputs a 4-column custom format (Date, Description, Amount, Currency). This phase standardizes it to 35-column format matching Transaction.MarshalCSV() output by wiring formatters. Real data analysis shows 8 transaction types across 2,126 CHF and 184 EUR transactions, with Product field distribution (Current/Savings) requiring account routing logic for iCompta import.

**Primary recommendation:** (1) Add Product field to Transaction model (csv tag), (2) Enhance Revolut parser to populate all 35 Transaction fields including Type tagging, (3) Wire Output Formatter system to both Revolut parsers using Phase 5 infrastructure, (4) Add SELL/CUSTODY_FEE handling to investment parser, (5) Implement batch conversion for investment parser.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/gocarina/gocsv` | v0.0.x | CSV marshaling for Transaction → output | Already in use; struct tag mapping |
| `encoding/csv` | stdlib | CSV writer with custom delimiters | stdlib; handles semicolon for iCompta |
| `time` | stdlib | Date parsing and formatting | Built-in; handles StartedDate/CompletedDate |
| `github.com/shopspring/decimal` | v1.x | Precise amount calculations | Already in use; required for financial data |
| `internal/formatter` | Phase 5 | Output format selection (standard/icompta) | Already implemented; pluggable interface |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/sirupsen/logrus` | v1.x | Structured logging for debug/error messages | Already configured for all parsers |
| `internal/models.TransactionBuilder` | — | Fluent API for creating Transaction objects | Consistent construction pattern across all parsers |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Add Product to Transaction | Store in separate mapping/config | Product is core data; belongs in Transaction model (mirrors other parsers' fields) |
| Per-parser CSV output format | Unified 35-column output | Unified format simplifies iCompta import and user experience |
| Filter PENDING/REVERTED | Handle with state parameter | Requirement REV-05 says "skip or flag"; handler should support both |

## Architecture Patterns

### Recommended Project Structure

**Changes to existing packages:**

```
internal/models/
├── transaction.go                   # MODIFY: Add Product field
├── constants.go                     # MODIFY: Add Product/transaction type constants if needed
└── [existing]

internal/revolutparser/
├── revolutparser.go                 # MODIFY: Enhance field population
├── adapter.go                       # MODIFY: Wire output formatter
└── [existing]

internal/revolutinvestmentparser/
├── revolutinvestmentparser.go       # MODIFY: Add SELL/CUSTODY_FEE handling, wire formatter
├── adapter.go                       # MODIFY: Wire output formatter + batch conversion
└── [existing]

cmd/revolut/convert.go              # MODIFY: Already has --format flag from Phase 5
cmd/revolut-investment/convert.go   # MODIFY: Already has --format flag from Phase 5
```

**No new packages needed** — leverage Phase 5 OutputFormatter system.

### Pattern 1: Product Field in Transaction

**What:** Add Product field (Current/Savings) to Transaction struct for Revolut account routing.

**When to use:** All Revolut CSV rows have Product field; route to different iCompta accounts based on Product+Currency.

**Example:**

```go
// Source: internal/models/transaction.go (ADD)
type Transaction struct {
    // ... existing fields ...
    Product           string          `csv:"Product"`           // Product type (Current, Savings)
    // ... rest of fields ...
}
```

**Mapping:**
- `Product="CURRENT"` + `Currency="CHF"` → iCompta "Revolut CHF"
- `Product="SAVINGS"` + `Currency="CHF"` → iCompta "Revolut CHF Vacances"
- `Product="CURRENT"` + `Currency="EUR"` → iCompta "Revolut EUR"

### Pattern 2: Transaction Type Identification

**What:** Populate Type field with transaction semantic type from Revolut's Type column.

**When to use:** All rows; enables categorization and account routing.

**Current implementation (incomplete):**
```go
// Source: internal/revolutparser/revolutparser.go (CURRENT)
// Line 208: Stores Type field but doesn't leverage semantics
builder = builder.WithType(row.Type)  // e.g., "TRANSFER", "CARD_PAYMENT"
```

**Required types (from real data):**
- TRANSFER (1,237 txns, 58%)
- CARD_PAYMENT (683 txns, 32%)
- EXCHANGE (135 txns, 6%)
- DEPOSIT (64 txns, 3%)
- FEE (4 txns, <1%)
- CHARGE (1 txn, <1%)
- CARD_REFUND (1 txn, <1%)
- CHARGE_REFUND (1 txn, <1%)

### Pattern 3: Exchange Transaction Metadata

**What:** Preserve original currency and amount in OriginalCurrency/OriginalAmount fields for exchange transactions.

**When to use:** When Type="EXCHANGE"; captures both sides of the conversion.

**Example from real data:**
```
Row: Type=EXCHANGE, Amount=-100.00 EUR, Description="Exchanged 100.00 EUR to CHF"
Expected output:
  Amount: -100.00
  Currency: EUR
  OriginalCurrency: CHF (or reverse, based on sign)
  OriginalAmount: ~120.00 (derived from FX rate)
```

**Note:** Phase 6 does NOT implement cross-file exchange pairing (deferred to future); each file stands alone.

### Pattern 4: Output Formatter Integration

**What:** Revolut parsers write CSV using OutputFormatter interface (Phase 5).

**When to use:** All CSV output; enables --format flag (standard/icompta).

**Example:**
```go
// Source: cmd/revolut/convert.go (MODIFY)
// Phase 5 already wired this for Revolut:
// - Line X: `--format` flag defined
// - Line Y: `--date-format` flag defined
// - ProcessFile() called with format parameter

// Phase 6 must ensure:
// 1. revolut parser populates all 35 Transaction fields
// 2. WriteTransactionsToCSVWithFormatter called (not custom WriteToCSV)
// 3. Both standard (34-col) and iCompta (10-col) formats work
```

### Pattern 5: Batch Conversion Support

**What:** Investment parser implements BatchConvert() interface for --batch flag.

**When to use:** Multiple investment CSV files in one operation.

**Current status:**
- Revolut parser: Already has BatchConvert() (lines 414-518)
- Investment parser: Missing BatchConvert() — must implement

**Required interface:**
```go
// Source: internal/parser/parser.go
type BatchConverter interface {
    BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error)
}
```

### Anti-Patterns to Avoid

- **Custom per-parser CSV output:** Don't create parser-specific WriteToCSV variants. Use OutputFormatter + WriteTransactionsToCSVWithFormatter (Phase 5).
- **Ignoring REVERTED/PENDING:** REV-05 requires handling; implement state parameter (config or flag) rather than hardcoding behavior.
- **Missing field mappings:** All 35 Transaction fields must be populated for standardization; don't leave fields empty/zero when source data exists.
- **Inconsistent exchange handling:** Document approach (both currencies visible, no pair matching in v1.2).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CSV output formatting | Custom delimiters/columns per parser | `internal/formatter.OutputFormatter` + Phase 5 infrastructure | Formatters are pluggable, tested, reusable across all parsers |
| Transaction construction | Manual field assignment | `models.TransactionBuilder` | Builder ensures consistent field initialization and validation |
| Batch file processing | Custom directory iteration | `parser.BatchConverter` interface | Interface provides standard error handling, file filtering, logging |
| Date parsing | `time.Parse` with per-field logic | `dateutils` package functions or TransactionBuilder.WithDateFromDatetime() | Consistent date handling across all parsers |
| Decimal math | `float64` arithmetic | `github.com/shopspring/decimal` | Already required for all financial calculations; avoids rounding errors |

**Key insight:** Phase 5 delivered OutputFormatter as a reusable system. Revolut parser must plug into it rather than maintaining custom CSV logic.

## Common Pitfalls

### Pitfall 1: Product Field Not Propagated to CSV Output

**What goes wrong:** Product field added to Transaction model but not included in CSV header/output columns.

**Why it happens:** Forgot to update StandardFormatter or iComptaFormatter headers. Product field exists in Transaction but isn't exported if not in CSV columns.

**How to avoid:**
- Verify StandardFormatter.Header() includes all 35 column names (check against Transaction.MarshalCSV header order)
- Verify Transaction.MarshalCSV() already includes Product in its output (check line 307-349)
- Test: Parse real Revolut CSV → check output CSV includes Product column

**Warning signs:**
- Output CSV missing Product column
- iCompta import ignores Product field
- No way to distinguish Current vs Savings accounts in output

### Pitfall 2: REVERTED/PENDING Transactions Silently Dropped

**What goes wrong:** Parser filters all non-COMPLETED transactions without user control or visibility.

**Why it happens:** Current parser hardcodes `if row.State != "COMPLETED" continue` (line 96). Requirement REV-05 says "skip or flag based on user preference."

**How to avoid:**
- Implement state handling as config option (e.g., `--include-pending` flag or config)
- Don't silently skip; log warnings for skipped transactions
- Consider adding Status field to output even for skipped transactions (allows later filtering in iCompta)
- Test with real data: CHF file has 4 REVERTED, 1 PENDING (should be visible in logs)

**Warning signs:**
- Logs show X transactions parsed, output CSV has fewer rows, no explanation
- User doesn't know why expected transactions are missing
- iCompta import shows different transaction count than Revolut CSV

### Pitfall 3: Exchange Transactions Lose Original Currency/Amount

**What goes wrong:** Exchange transactions converted to single currency, original FX rate and amount lost.

**Why it happens:** Parser focuses on transaction amount, ignores semantic meaning. For EXCHANGE type, both currencies are significant.

**How to avoid:**
- When Type="EXCHANGE", extract both currencies from description or compute from FX field
- Populate OriginalCurrency and OriginalAmount fields from CSV data
- Example: "Exchanged 100.00 EUR to CHF at rate 1.08"
  - Amount: 108.00 CHF
  - OriginalAmount: 100.00 EUR
  - OriginalCurrency: EUR
  - ExchangeRate: 1.08
- Test: Parse EUR exchange rows → verify OriginalCurrency/OriginalAmount in output

**Warning signs:**
- CSV output shows only Amount/Currency, no OriginalAmount/OriginalCurrency
- Exchange transactions can't be matched across CHF/EUR files (data loss)
- iCompta import doesn't distinguish original from converted amounts

### Pitfall 4: Formatter Integration Incomplete

**What goes wrong:** Parser populates all 35 Transaction fields but output CSV only has 4 columns (old format).

**Why it happens:** Parser's ConvertToCSV() or adapter still calls custom WriteToCSV() instead of using Phase 5 OutputFormatter.

**How to avoid:**
- Audit revolutparser/adapter.go: ConvertToCSV() must use common.WriteTransactionsToCSVWithFormatter()
- Audit cmd/revolut/convert.go: ProcessFile() already wires formatter (Phase 5), verify adapter uses it
- Test: Convert Revolut CSV with `--format icompta` → should produce semicolon-delimited, 10-column output
- Compare to CAMT/PDF/Selma parsers (all use Phase 5 formatter system)

**Warning signs:**
- `--format icompta` flag ignored, output still has 4 columns
- CSV delimiter always comma, never semicolon
- Output columns don't match StandardFormatter.Header()

### Pitfall 5: Investment Parser Missing Batch Support

**What goes wrong:** Regular Revolut parser has BatchConvert() but investment parser doesn't.

**Why it happens:** Requirement BATCH-02 (Revolut Investment batch support) and RINV-03 (Batch conversion support) both point to investment parser; must implement interface.

**How to avoid:**
- Implement BatchConverter interface in revolutinvestmentparser/adapter.go
- Model after revolutparser.BatchConvert() (lines 414-518): iterate files, validate format, parse, write output
- Test with multiple investment CSV files in one directory
- Verify --batch flag works on revolut-investment command

**Warning signs:**
- Investment parser doesn't support `--batch` flag (comparison: `camt --batch` works, `revolut-investment --batch` fails)
- No BatchConvert() method on adapter
- BATCH-02 requirement uncovered in Phase 6 verification

### Pitfall 6: Inconsistent Date Handling

**What goes wrong:** StartedDate and CompletedDate from Revolut CSV mapped incorrectly to Transaction.Date/ValueDate.

**Why it happens:** Field names are intuitive but mapping must be consistent: StartedDate → ValueDate (when transaction initiated), CompletedDate → Date (when settled).

**How to avoid:**
- Document mapping clearly: StartedDate (initiated) = ValueDate; CompletedDate (settled) = Date
- Verify builder calls: `.WithDateFromDatetime(row.CompletedDate)` and `.WithValueDateFromDatetime(row.StartedDate)`
- Test with real data: Check output dates match expected settlement dates
- Note: Phase 5 already uses DD.MM.yyyy format in iComptaFormatter; ensure consistency

**Warning signs:**
- Transaction dates are transposed (Started/Completed swapped)
- iCompta import shows wrong transaction dates
- ValueDate is empty or incorrect

## Code Examples

Verified patterns from codebase and official sources:

### Example 1: Populate All 35 Transaction Fields

```go
// Source: internal/revolutparser/revolutparser.go (ENHANCE EXISTING)
// Current: Lines 200-220 build partial transaction
// Enhanced: Populate all fields per Transaction struct

builder := models.NewTransactionBuilder().
    // Status and dates
    WithStatus(row.State).                           // "COMPLETED", "REVERTED", "PENDING"
    WithDateFromDatetime(row.CompletedDate).        // Settled date
    WithValueDateFromDatetime(row.StartedDate).     // Initiated date

    // Parties and descriptions
    WithDescription(row.Description).
    WithPartyName(row.Description).
    WithPayee(row.Description, "").                 // For debits
    WithPayer(row.Description, "").                 // For credits

    // Amount and currency
    WithAmount(amountDecimal, row.Currency).
    WithFees(feeDecimal).

    // Type and product (NEW FIELDS)
    WithType(row.Type).                             // "TRANSFER", "CARD_PAYMENT", "EXCHANGE", etc.
    WithInvestment(row.Type).                       // For consistency with other parsers
    WithCategory(models.CategoryUncategorized).     // Pre-categorization (will be overridden if categorizer provided)

    // Product field (NEW - must add to builder)
    // builder.WithProduct(row.Product)             // "CURRENT", "SAVINGS"

    // Set transaction direction based on amount sign
    if isDebit {
        builder = builder.AsDebit()
    } else {
        builder = builder.AsCredit()
    }

transaction, err := builder.Build()
```

**Note:** TransactionBuilder must be enhanced to support `.WithProduct()` method.

### Example 2: Handle Exchange Transactions

```go
// Source: internal/revolutparser/revolutparser.go (ADD LOGIC)
// When row.Type == "EXCHANGE", extract metadata

if row.Type == "EXCHANGE" {
    // Parse description: "Exchanged 100.00 EUR to CHF" or similar pattern
    // Extract original currency and amount from description or FX field

    // Example if Revolut CSV had FX rate column:
    // Amount: 100.00 (CHF after conversion)
    // OriginalAmount: X.XX (EUR before conversion)
    // OriginalCurrency: "EUR"
    // ExchangeRate: computed from Amount and OriginalAmount

    // For now with current CSV structure:
    // - Amount and Currency capture the settled amount
    // - OriginalCurrency/OriginalAmount left empty (data not in CSV)
    // - Document decision in commit

    builder = builder.
        WithOriginalCurrency("").    // Parse from Description if available
        WithExchangeRate(decimal.One) // Mark as 1:1 if not available
}
```

### Example 3: Filter REVERTED/PENDING (with logging)

```go
// Source: internal/revolutparser/revolutparser.go (ENHANCE LINE 96)
// Current: if revolutRows[i].State != models.StatusCompleted { continue }
// Enhanced: Log skipped transactions for visibility

// Determine skip behavior from config/flag (assume skipPending=true for now)
skipPending := true  // Would come from config or CLI flag

if revolutRows[i].State != models.StatusCompleted {
    if skipPending {
        logger.WithFields(
            logging.Field{Key: "date", Value: revolutRows[i].CompletedDate},
            logging.Field{Key: "description", Value: revolutRows[i].Description},
            logging.Field{Key: "state", Value: revolutRows[i].State},
        ).Info("Skipping transaction (state not COMPLETED)")
    }
    // Future: could include PENDING/REVERTED with flag or add Status field
    continue
}
```

### Example 4: Wire Output Formatter in Adapter

```go
// Source: internal/revolutparser/adapter.go (MODIFY WriteToCSV)
// Current: Uses custom WriteToCSVWithLogger (4-column format)
// Enhanced: Use Phase 5 OutputFormatter system

func (a *Adapter) WriteToCSV(transactions []models.Transaction, csvFile string) error {
    // Get formatter from container/registry (Phase 5 pattern)
    // For now, use StandardFormatter (default)
    // CLI handles --format flag selection upstream

    formatter := formatter.NewStandardFormatter()
    delimiter := formatter.Delimiter()

    // Use Phase 5 infrastructure
    return common.WriteTransactionsToCSVWithFormatter(
        transactions,
        csvFile,
        a.GetLogger(),
        formatter,
        delimiter,
    )
}
```

### Example 5: Investment Parser - SELL Transaction

```go
// Source: internal/revolutinvestmentparser/revolutinvestmentparser.go (ADD CASE)
// Current: Lines 183 handle BUY, 233 handle DIVIDEND
// Add: SELL transaction handling

case strings.Contains(row.Type, "SELL"):
    // SELL: stock/fund sale
    // Quantity: negative (units sold)
    // Amount: proceeds (cash received)

    builder = builder.
        WithDescription(fmt.Sprintf("Sold %s %s", row.Quantity, row.Ticker)).
        WithType("SELL")

    // Quantity as negative (selling)
    quantity, _ := strconv.Atoi(row.Quantity)
    builder = builder.WithNumberOfShares(-1 * quantity)

    // Amount is proceeds
    amount, _ := decimal.NewFromString(row.TotalAmount)
    if strings.HasPrefix(row.TotalAmount, "-") {
        // Already negative from CSV
        builder = builder.AsDebit().WithAmount(amount, row.Currency)
    } else {
        builder = builder.AsCredit().WithAmount(amount, row.Currency)
    }

    // Fund/ticker info
    builder = builder.
        WithFund(row.Ticker).
        WithInvestment("SELL")
```

### Example 6: Investment Parser - Batch Conversion

```go
// Source: internal/revolutinvestmentparser/adapter.go (ADD METHOD)
// Required by BatchConverter interface (Phase 6)

func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    logger := a.GetLogger()
    if logger == nil {
        logger = logging.NewLogrusAdapter("info", "text")
    }

    logger.Info("Batch converting Revolut investment CSV files",
        logging.Field{Key: "inputDir", Value: inputDir},
        logging.Field{Key: "outputDir", Value: outputDir})

    // Create output directory
    if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
        return 0, fmt.Errorf("error creating output directory: %w", err)
    }

    // List CSV files in input directory
    files, err := os.ReadDir(inputDir)
    if err != nil {
        return 0, fmt.Errorf("error reading input directory: %w", err)
    }

    var processed int
    for _, file := range files {
        if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
            continue
        }

        inputPath := filepath.Join(inputDir, file.Name())
        outputPath := filepath.Join(outputDir, strings.TrimSuffix(file.Name(), ".csv")+"-standardized.csv")

        // Convert single file
        if err := a.ConvertToCSV(ctx, inputPath, outputPath); err != nil {
            logger.WithError(err).Warn("Failed to convert file",
                logging.Field{Key: "file", Value: inputPath})
            continue
        }

        processed++
    }

    logger.Info("Batch conversion completed",
        logging.Field{Key: "count", Value: processed})

    return processed, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-parser custom CSV output (4/34/10 columns) | Unified OutputFormatter interface with pluggable implementations | Phase 5 (2026-02-16) | All parsers now support --format flag; standardizes to 35-column core, 10-column iCompta variant |
| Parser WriteToCSV() methods duplicated | Centralized common.WriteTransactionsToCSVWithFormatter() | Phase 5 | Eliminates duplication; formatter selected by CLI flag, not parser |
| Revolut CSV → 4 columns (Date, Description, Amount, Currency) | Revolut CSV → 35 columns (all Transaction fields) | Phase 6 target | Enables iCompta import, cross-parser consistency, metadata preservation |
| COMPLETED transactions only (hardcoded filter) | Configurable state handling (skip/include PENDING/REVERTED) | Phase 6 target | Transparent transaction filtering; user controls visibility |
| No Type field populated | Type field identifies transaction semantics | Phase 6 target | Enables categorization rules based on type (TRANSFER vs CARD_PAYMENT behavior differs) |
| Product field not mapped | Product field in Transaction model → account routing | Phase 6 target | Supports Savings vs Current account separation in iCompta |

**Deprecated/outdated:**
- Custom revolutparser.WriteToCSV(): Replaced by common.WriteTransactionsToCSVWithFormatter() + OutputFormatter (Phase 5)
- Ad-hoc date formatting per parser: Standardized to dd.MM.yyyy; --date-format flag reserved for future dynamic formats (Phase 5)

## Open Questions

1. **Product Field Integration**
   - What we know: Revolut CSV has Product column (CURRENT/SAVINGS); real data shows 1,438 Current and 688 Savings txns in CHF file
   - What's unclear: Should Product field be added to Transaction.MarshalCSV() header (35-column standard), or is it a Revolut-only field? Check other parsers (CAMT, PDF, Selma, Debit) — do they have Product concept?
   - Recommendation: Check CLAUDE.md and other parser implementations. If Product is Revolut-specific, keep it in Transaction model but don't export to standard CSV header. iCompta import plugins already configured; verify column names match.

2. **REVERTED/PENDING State Handling**
   - What we know: Real data has 4 REVERTED and 1 PENDING transaction; requirement REV-05 says "skip or flag based on user preference"
   - What's unclear: Is this a CLI flag (--include-pending), config file option, or automatic detection? Does user want to exclude these or include with special Status marking?
   - Recommendation: Start with conservative approach: skip by default (log each skipped transaction), add --include-pending flag if needed. Document in changelog.

3. **Exchange Transaction Metadata Extraction**
   - What we know: 135 EXCHANGE transactions in CHF file (6%); real CSV likely has Description like "Exchanged 100.00 EUR to CHF"
   - What's unclear: Does Revolut CSV include FX rate or pre-calculated amounts? How to extract original currency from description reliably?
   - Recommendation: Inspect actual Revolut CSV structure. If FX field exists, parse it. If not, derive from amounts. Document assumption in code.

4. **Investment Parser SELL/CUSTODY_FEE Implementation Details**
   - What we know: Investment parser currently handles BUY, DIVIDEND, CASH_TOP_UP (lines 183-293)
   - What's unclear: Investment CSV structure for SELL and CUSTODY_FEE rows. Are Quantity/Price/Amount populated the same way as BUY? Does CUSTODY_FEE have a Ticker?
   - Recommendation: Check investment parser test data to understand row structure. Match pattern from existing transaction types.

5. **Batch Conversion Flag/Command Wiring**
   - What we know: Revolut parser has BatchConvert(), investment parser doesn't; Phase 5 already added --format flag
   - What's unclear: Does cmd/revolut-investment command already have --batch flag defined? Does cmd/common/process.go handle batch mode?
   - Recommendation: Verify cmd/revolut-investment/convert.go has batch flag and calls adapter.BatchConvert(). Model after cmd/camt/convert.go.

## Sources

### Primary (HIGH confidence)
- Codebase inspection (2026-02-16):
  - `/Users/fjacquet/Projects/camt-csv/internal/revolutparser/` — current Revolut parser (4-column output, partial field population)
  - `/Users/fjacquet/Projects/camt-csv/internal/revolutinvestmentparser/` — investment parser (BUY/DIVIDEND/TOP-UP handling)
  - `/Users/fjacquet/Projects/camt-csv/internal/models/transaction.go` — Transaction struct (35 fields, MarshalCSV method)
  - `/Users/fjacquet/Projects/camt-csv/internal/formatter/` — Phase 5 OutputFormatter system (standard + iCompta formatters)
  - `/Users/fjacquet/Projects/camt-csv/internal/common/csv.go` — WriteTransactionsToCSVWithFormatter (Phase 5 integration point)
- Real data analysis (`/.planning/reference/v1.2-decisions.md`):
  - 2,126 CHF transactions: Type distribution (TRANSFER 58%, CARD_PAYMENT 32%, EXCHANGE 6%, DEPOSIT 3%, other <1%)
  - 184 EUR transactions: Mostly EXCHANGE and CARD_PAYMENT
  - Product distribution: 1,438 Current, 688 Savings (CHF file)
  - States: 2,121 COMPLETED, 4 REVERTED, 1 PENDING

### Secondary (MEDIUM confidence)
- Phase 5 Research and Verification:
  - `.planning/phases/05-output-framework/05-RESEARCH.md` — OutputFormatter interface, StandardFormatter (34-col), iComptaFormatter (10-col)
  - `.planning/phases/05-output-framework/05-VERIFICATION.md` — Phase 5 delivered and tested; all 6 parsers have --format flag
  - `.planning/reference/icompta-import-plugins.txt` — iCompta CSV-Revolut-CHF and CSV-Revolut-EUR plugin mappings
- Requirements traceability (`/.planning/REQUIREMENTS.md`):
  - REV-01 through REV-05: Revolut parser requirements
  - RINV-01 through RINV-03: Investment parser requirements
  - BATCH-02: Revolut Investment batch support

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All libraries already in use; OutputFormatter phase tested
- Architecture patterns: HIGH — Phase 5 infrastructure verified; Revolut parser anatomy clear from inspection
- Pitfalls: MEDIUM-HIGH — Based on current parser structure + Phase 5 integration patterns; some REVERTED/PENDING details need user clarification
- Code examples: MEDIUM — Based on existing patterns in codebase; SELL/CUSTODY_FEE details need investment CSV structure verification

**Research date:** 2026-02-16
**Valid until:** 2026-02-23 (7 days for this active domain; Revolut CSV format stable, but investment parser details may need early clarification)

**Research gaps:**
- SELL and CUSTODY_FEE transaction structure in investment CSV (inspect test data early in Phase 6 planning)
- Exact mapping of Product field in other parsers (may be Revolut-specific; verify before adding to Transaction.MarshalCSV)
- Whether --batch flag already wired on revolut-investment command (check Phase 5 completion)

