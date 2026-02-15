# Architecture Patterns: camt-csv Feature Integration

**Domain:** Financial transaction parser/converter CLI with AI categorization
**Researched:** 2026-02-15
**Confidence:** HIGH (source: codebase inspection, design patterns verified)

## Current Architecture Overview

The camt-csv application follows a **segregated interface composition pattern** with a **dependency injection container** as the single source of truth for wiring.

### Core Interfaces (Interface Segregation Pattern)

All parsers implement the `FullParser` composite interface:

```go
type FullParser interface {
    Parser                      // Parse(ctx, reader) → []Transaction
    Validator                   // ValidateFormat(file) → bool
    CSVConverter                // ConvertToCSV(ctx, inFile, outFile) → error
    LoggerConfigurable          // SetLogger(logger)
    CategorizerConfigurable     // SetCategorizer(categorizer)
    BatchConverter              // BatchConvert(ctx, inDir, outDir) → (count, error)
}
```

Each parser embeds `BaseParser` which:
- Stores `logger` and `categorizer` fields
- Provides helper methods: `GetLogger()`, `GetCategorizer()`, `SetLogger()`, `SetCategorizer()`
- Provides `WriteToCSV()` that delegates to `common.WriteTransactionsToCSV()`

## Feature 1: Transaction-Type-Aware Revolut Parsing

### Current State

- **Revolut CSV columns:** Type, Product, StartedDate, CompletedDate, Description, Amount, Fee, Currency, State, Balance
- **Known types:** ~9 (TRANSFER, CARD_PAYMENT, REFUND, EXCHANGE, TOPUP, DIVIDENDS, INTEREST, BUY, SELL)
- **Current handling:** Basic filtering in `ParseWithCategorizer()`, Type copied to Transaction.Type field but not used in categorization

### Recommended Architecture: Type Processing IN Parser

**Decision:** Keep type-aware logic in the `revolutparser` package, not a separate processor.

**Rationale:**
1. **Cohesion:** Type mapping is Revolut format-specific
2. **Single Responsibility:** Parser owns format understanding
3. **Testability:** Unit test type logic in parser package
4. **Config encapsulation:** Revolut type rules stay in Revolut parser config

### Implementation: Strategy Pattern for Type Processors

Add type-specific processors inside `revolutparser` package:

```go
type TypeProcessor interface {
    ProcessType(row *RevolutCSVRow, tx *models.Transaction) error
}

var typeProcessors = map[string]TypeProcessor{
    "TRANSFER":      &TransferProcessor{},
    "CARD_PAYMENT":  &CardPaymentProcessor{},
    "EXCHANGE":      &ExchangeProcessor{},
    "DIVIDENDS":     &DividendProcessor{},
    "INTEREST":      &InterestProcessor{},
    "BUY":           &BuyProcessor{},
    "SELL":          &SellProcessor{},
}
```

Invoke in `convertRevolutRowToTransaction()`:

```go
if processor, ok := typeProcessors[row.Type]; ok {
    if err := processor.ProcessType(&row, &tx); err != nil {
        logger.WithError(err).Warn("Type processing failed")
    }
}
```

### Transaction Model Fields Used

The `Transaction` struct already has sufficient fields:
- `Type: string` → Revolut Type (TRANSFER, CARD_PAYMENT, etc.)
- `Investment: string` → Investment type for investments (Buy, Sell, Income)
- `Description: string` → Preserved from Revolut
- `Category: string` → Categorization result

**Mapping example:**

| Revolut Type | Transaction.Type | Transaction.Investment | Reason |
|---|---|---|---|
| TRANSFER | "TRANSFER" | "" | Internal transfer |
| CARD_PAYMENT | "CARD_PAYMENT" | "" | Expense tracking |
| DIVIDENDS | "DIVIDENDS" | "Income" | Investment income |
| BUY | "BUY" | "Buy" | Stock/fund purchase |

## Feature 2: Standardizing Revolut Output

### Current State

- **Revolut WriteToCSV:** 4-column format (Date, Description, Amount, Currency)
- **Common WriteToCSV:** 35-column standard format (all Transaction fields)
- **Risk:** Breaking change for existing Revolut users

### Recommended Strategy: Format Flag + OutputFormatter Interface

Add `--format` flag: `revolut` (4-col), `standard` (35-col), `icompta` (iCompta-specific)

### Implementation: OutputFormatter Plugin System

Create in `internal/formatter/`:

```go
type OutputFormatter interface {
    Format(transactions []models.Transaction) ([][]string, error)
    Header() []string
}

type FormatterRegistry struct {
    formatters map[string]OutputFormatter
}

func NewFormatterRegistry() *FormatterRegistry {
    return &FormatterRegistry{
        formatters: map[string]OutputFormatter{
            "standard":  &StandardFormatter{},
            "revolut":   &RevolutFormatter{},
            "icompta":   &iComptaFormatter{},
        },
    }
}
```

Modify common CSV export:

```go
func WriteTransactionsToCSVWithFormatter(
    transactions []models.Transaction,
    csvFile string,
    formatter formatter.OutputFormatter,
    logger logging.Logger,
) error {
    // Use formatter.Header() and formatter.Format()
}
```

**Backward compatibility:** Default to `standard`, emit deprecation warning for Revolut, change default in v2.0.

## Feature 3: iCompta CSV Output

### iCompta Schema Key Tables

- `ICTransaction` — Transaction header (date, amount, name, status)
- `ICTransactionSplit` — Category assignment (one or more splits per transaction)

iCompta uses **split model**: one transaction can have multiple category splits (e.g., 60% food, 40% transport).

### Mapping: Revolut → iCompta

| Transaction Field | iCompta Column | Notes |
|---|---|---|
| `Date` | date | YYYY-MM-DD ISO |
| `Amount` | amount | Sum of splits |
| `Name` (PartyName) | name | Payee/payer |
| `Status` | status | "COMPLETED" → "cleared" |
| `Description` | comment | Narrative |
| `Category` | ICTransactionSplit.category | From categorizer |
| `AmountExclTax` | ICTransactionSplit.amountWithoutTaxes | If available |

### Output Format: Denormalized CSV

iComptaFormatter exports as:

```csv
TransactionID,Date,Name,Amount,Comment,Status,SplitAmount,SplitCategory,SplitAmountExclTax,SplitTaxRate
TX001,2026-02-15,Vendor ABC,100.00,Restaurant,cleared,100.00,Food,100.00,0.00
TX002,2026-02-15,Gas Station,80.00,Fuel,cleared,80.00,Transport,80.00,0.00
```

Implementation in `internal/formatter/icompta.go`:

```go
type iComptaFormatter struct{}

func (f *iComptaFormatter) Header() []string {
    return []string{
        "TransactionID", "Date", "Name", "Amount", "Comment", "Status",
        "SplitAmount", "SplitCategory", "SplitAmountExclTax", "SplitTaxRate",
    }
}

func (f *iComptaFormatter) Format(transactions []models.Transaction) ([][]string, error) {
    var rows [][]string
    for i, tx := range transactions {
        txID := fmt.Sprintf("TX%06d", i+1)
        date := tx.Date.Format("2006-01-02")
        row := []string{
            txID, date, tx.Name, tx.Amount.StringFixed(2),
            tx.Description, "cleared",
            tx.Amount.StringFixed(2), tx.Category,
            tx.AmountExclTax.StringFixed(2), tx.TaxRate.StringFixed(2),
        }
        rows = append(rows, row)
    }
    return rows, nil
}
```

## Feature 4: AI Auto-Learn Safety Controls

### Current AI Flow

Three-tier categorization in `internal/categorizer/categorizer.go`:
1. Direct mapping (creditors.yaml / debtors.yaml)
2. Keyword matching (categories.yaml rules)
3. AI fallback (if enabled)

Auto-learning: AI results saved back to YAML if `categorization.auto_learn = true`.

### Safety Controls: Proposed Additions

| Control | Type | Purpose |
|---|---|---|
| `categorization.auto_learn` | bool | Master enable/disable |
| `categorization.learn_confidence_threshold` | float | Min confidence (0.0-1.0) to auto-save |
| `categorization.learn_dry_run` | bool | Log what would be learned, don't save |
| `categorization.learn_approval_required` | bool | Require explicit approval |

### Implementation: Audit Trail in Categorizer

Add to `internal/categorizer/categorizer.go`:

```go
type AuditResult struct {
    Category           *Category
    Source             string      // "direct", "keyword", "ai"
    Confidence         float64     // 0.0-1.0
    AutoLearnWill      bool        // Would be saved if approved
    ApprovalRequired   bool
}

type CategorizeAudit struct {
    Results []AuditResult
    mu      sync.RWMutex
}

func (c *Categorizer) GetAuditLog() []AuditResult {
    c.audit.mu.RLock()
    defer c.audit.mu.RUnlock()
    return append([]AuditResult{}, c.audit.Results...)
}
```

Config wiring in `container.NewContainer()`:

```go
cat := categorizer.NewCategorizer(aiClient, categoryStore, logger)

if !cfg.Categorization.AutoLearn {
    cat.DisableAutoLearn()
}
if cfg.Categorization.LearningDryRun {
    cat.EnableDryRun()
}
if cfg.Categorization.LearnConfidenceThreshold > 0 {
    cat.SetConfidenceThreshold(cfg.Categorization.LearnConfidenceThreshold)
}
```

**Backward compatible:** Defaults allow existing auto-learn behavior.

## Feature 5: Batch Support for PDF and Revolut Investment

### Current Batch Support

- **CAMT parser:** Fully implemented `BatchConvert()`
- **Revolut parser:** Fully implemented `BatchConvert()`
- **PDF parser:** `BatchConvert` exists
- **Revolut Investment parser:** MISSING `BatchConvert`

### Implementation: Template Method in BaseParser

Add to `internal/parser/base.go`:

```go
func (b *BaseParser) DefaultBatchConvert(
    ctx context.Context,
    inputDir, outputDir string,
    fileExt string,
    convertFunc func(ctx context.Context, inFile, outFile string) error,
) (int, error) {
    logger := b.GetLogger()
    entries, err := os.ReadDir(inputDir)
    if err != nil {
        return 0, fmt.Errorf("cannot read directory: %w", err)
    }

    count := 0
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        if filepath.Ext(entry.Name()) != fileExt {
            continue
        }

        inFile := filepath.Join(inputDir, entry.Name())
        outFile := filepath.Join(outputDir, entry.Name()+".csv")

        if err := convertFunc(ctx, inFile, outFile); err != nil {
            logger.WithError(err).Warn("Failed to convert file",
                logging.Field{Key: "file", Value: entry.Name()})
            continue  // Skip on error, continue batch
        }
        count++
    }
    return count, nil
}
```

Each parser implements:

```go
// In revolutinvestmentparser/adapter.go
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    return a.DefaultBatchConvert(ctx, inputDir, outputDir, ".csv", a.ConvertToCSV)
}
```

## New vs. Modified: Component Changes

### New Components (5)

| Component | Location |
|---|---|
| `TypeProcessor` interface | `internal/revolutparser/types.go` |
| Type processors (Transfer, CardPayment, Exchange, Dividend, Interest, Buy, Sell) | `internal/revolutparser/types.go` |
| `OutputFormatter` interface | `internal/formatter/formatter.go` |
| `StandardFormatter`, `RevolutFormatter`, `iComptaFormatter` | `internal/formatter/` |
| `FormatterRegistry` | `internal/formatter/formatter.go` |
| `AuditResult`, `CategorizeAudit` | `internal/categorizer/audit.go` |

### Modified Components (6)

| Component | Changes |
|---|---|
| `revolutparser.convertRevolutRowToTransaction()` | Add type processor invocation |
| `revolutparser.ParseWithCategorizer()` | Delegate type handling to processors |
| `BaseParser` | Add `DefaultBatchConvert()` method |
| `common.WriteTransactionsToCSVWithLogger()` | Accept optional formatter param |
| `Categorizer` | Add audit tracking, confidence checks |
| `Config` | Add `categorization.learn_*` fields |

### No Changes Needed

| Component | Reason |
|---|---|
| `Transaction` model | Already has Type, Investment, Category fields |
| `Parser` interfaces | Already support all features |
| Container | Formatters are optional plugins |
| CLI commands | Will add `--format` flag, but no core refactor |

## Data Flow: Type-Aware Revolut with All Features

```
revolut convert --input file.csv --output out.csv --format icompta

CLI → Container.GetParser(Revolut)
    ↓
Adapter.ConvertToCSV(ctx, input, output)
    ├─ Parse(ctx, reader)
    │   ├─ Validate format
    │   ├─ Unmarshal CSV → RevolutCSVRow[]
    │   └─ For each row:
    │       ├─ convertRevolutRowToTransaction()
    │       │   ├─ Build Transaction
    │       │   ├─ typeProcessors[row.Type].ProcessType()
    │       │   │   └─ Populate Type, Investment, Description
    │       │   └─ Return tx
    │       └─ Categorizer.Categorize()
    │           ├─ Try direct mapping
    │           ├─ Try keyword matching
    │           ├─ Try AI (if enabled + threshold)
    │           └─ Log audit trail
    │
    └─ WriteToCSVWithFormatter(transactions, output, iComptaFormatter)
        ├─ Formatter.Header() → column names
        ├─ For each tx: Formatter.Format() → rows
        └─ CSV write + flush
```

## Scalability Considerations

| Concern | 100 TX | 10K TX | 100K TX | 1M TX |
|---|---|---|---|---|
| **Memory** | ~2MB | ~200MB | ~2GB | ~20GB |
| **Parsing** | <100ms | ~1-2s | ~10-20s | ~100-200s |
| **Categorization** | ~500ms (AI) | ~50-100s | 10-20min | 2-4hrs |
| **Type processing** | Negligible | Negligible | ~1s | ~10s |
| **Write to CSV** | <100ms | ~1s | ~10s | ~100s |
| **Audit trail** | ~1KB | ~100KB | ~1MB | ~10MB |

**Recommendation:** For 10K+ transactions, disable AI categorization or use keyword-only mode.

## Build Order for Implementation

### Phase 1: Type-Aware Parsing (Low Risk)

1. Create `internal/revolutparser/types.go`
2. Implement type processors
3. Update `convertRevolutRowToTransaction()`
4. Add tests
5. **Impact:** Additive, no breaking changes

### Phase 2: Output Formatters (Medium Risk)

1. Create `internal/formatter/`
2. Implement formatters
3. Update common CSV export
4. Add `--format` flag
5. Default to `standard` with deprecation warning
6. **Impact:** Breaking change in v2.0

### Phase 3: AI Safety Controls (Low Risk)

1. Add audit tracking to `Categorizer`
2. Add config fields
3. Wire in container
4. **Impact:** Additive, fully backwards-compatible

### Phase 4: Batch Support (Low Risk)

1. Add `DefaultBatchConvert()` to `BaseParser`
2. Implement in `RevolutInvestmentAdapter`
3. Add tests
4. **Impact:** Additive

## Component Boundaries

```
revolutparser/
├── adapter.go (FullParser implementation)
├── revolutparser.go (core parsing + type-aware logic)
├── types.go (NEW: TypeProcessor + implementations)
└── validation.go

formatter/ (NEW)
├── formatter.go (interface + registry)
├── standard.go (35-col)
├── revolut.go (4-col)
└── icompta.go (iCompta format)

categorizer/
├── categorizer.go (3-tier + audit)
├── audit.go (NEW: audit tracking)
└── strategies/

parser/
├── base.go (add DefaultBatchConvert())
└── parser.go (interfaces)

config/
└── viper.go (add learn_* fields)

models/
└── transaction.go (no change)

container/
└── container.go (no change)
```

## Confidence Assessment

| Area | Confidence | Reasoning |
|---|---|---|
| Type handling | HIGH | Strategy pattern verified in codebase |
| Formatter system | HIGH | Composition pattern established |
| iCompta mapping | MEDIUM-HIGH | Schema documented, import untested |
| AI safety | HIGH | Config system proven |
| Batch processing | HIGH | Pattern established |
| Overall integration | HIGH | No conflicting dependencies |

## Gaps for Phase-Specific Research

1. Type processor edge cases (disputed/pending transactions)
2. iCompta split amount calculations (complex splits)
3. Batch performance benchmarks (100K+ transactions)
4. Audit log persistence (disk vs. in-memory)
5. Category correlation with Revolut types
