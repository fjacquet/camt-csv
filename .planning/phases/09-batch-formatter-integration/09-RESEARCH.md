# Phase 9: Batch-Formatter Integration - Research

**Researched:** 2026-02-16
**Domain:** Formatter integration with batch processing, cross-parser output formatting in directory modes
**Confidence:** HIGH

## Summary

Phase 9 closes integration gaps between Phase 5 (Output Framework) and Phase 7 (Batch Infrastructure). Single-file workflows work correctly with formatters, but batch and consolidation code paths ignore format options and use legacy output writers. The technical challenge is **straightforward**: wire formatter options through existing batch/consolidation pipelines. No new abstractions are needed; composition and dependency injection already exist from Phases 5 and 7.

**Current state:**
- ✓ Phase 5: Output formatters (strategy pattern) support multiple formats
- ✓ Phase 7: BatchProcessor provides standardized manifest + exit codes
- ✗ Integration: Batch paths don't use formatters, Revolut uses legacy batch, PDF consolidation ignores format flags

**Three integration points must be updated:**
1. **BatchProcessor API:** Add formatter/dateFormat configuration options
2. **Revolut adapter:** Replace legacy `BatchConvert()` with `BatchProcessor` composition
3. **PDF commands:** Pass format/dateFormat through batch and consolidation functions

---

## User Constraints

No CONTEXT.md exists for this phase. This is a gap closure phase from the v1.2 audit with well-defined requirements.

---

## Standard Stack

### Core (Already Decided in Phases 5-7)

| Component | Purpose | Status | Used In |
|-----------|---------|--------|---------|
| Strategy Pattern (`internal/formatter/formatter.go`) | Multiple output formats without modification to core logic | ✓ Implemented | All parsers, single-file mode |
| `internal/batch/processor.go` | Standardized batch processing with manifest + exit codes | ✓ Implemented | PDF batch (via adapter), but not Revolut |
| Dependency Injection via `BaseParser` | Formatters passed to parsers via DI | ✓ Implemented | Single-file through `ProcessFile()` |
| Context propagation | Cancellation/timeout control in batch operations | ✓ Implemented | `ProcessDirectory()` accepts context |

### Output Formatters (Already in Place)

| Formatter | Version | Purpose | Gap |
|-----------|---------|---------|-----|
| `StandardFormatter` | Phase 5 | 35-column legacy format (all parsers) | Works in single-file mode |
| `IComptaFormatter` | Phase 5 | 10-column iCompta format (semicolon, iCompta date format) | Works in single-file mode |

### Libraries (Already in Use)

| Library | Module | Purpose |
|---------|--------|---------|
| `encoding/csv` | stdlib | CSV writing with custom delimiters |
| `fjacquet/camt-csv/internal/formatter` | Project | OutputFormatter interface + registry |
| `fjacquet/camt-csv/internal/batch` | Project | BatchProcessor, BatchManifest, BatchResult |
| `fjacquet/camt-csv/internal/common` | Project | `WriteTransactionsToCSVWithFormatter()` already exists |

---

## Architecture Patterns

### Current Data Flow: Single File Mode (WORKS)

```
CLI flag --format "icompta"
         ↓
    cmd/pdf/convert.go
         ↓
    common.ProcessFile()
         ↓
    FormatterRegistry.Get("icompta") → IComptaFormatter
         ↓
    WriteTransactionsToCSVWithFormatter(transactions, file, formatter)
         ↓
    CSV file with semicolon delimiter, iCompta date format
```

**Source:** Phase 5 verification in `.planning/phases/05-output-framework/05-VERIFICATION.md`

### Target Data Flow: Batch Mode (GAPS EXIST)

```
CLI flag --format "icompta"
         ↓
    cmd/pdf/convert.go
         ↓
    pdfBatchConvert() — CURRENTLY IGNORES --format FLAG
         ↓
    BatchProcessor.ProcessDirectory() — CURRENTLY LACKS FORMATTER CONFIG
         ↓
    ??? Missing link: how does formatter reach CSV writer in batch?
         ↓
    CSV file written with legacy writer (ignores format)
```

**The gap:** Format options are parsed but not threaded into batch processing.

---

## Architecture Decisions from Prior Phases

### From Phase 5 (Output Framework)
- **D-06:** `--format` flag is cross-parser (works on all parsers, not just Revolut)
- **D-07:** ICompta format uses semicolon delimiter + `dd.MM.yyyy` date format
- **Pattern:** Strategy pattern via `OutputFormatter` interface with registry

### From Phase 7 (Batch Infrastructure)
- **Pattern 1:** `BatchProcessor` handles standardized batch processing
- **Pattern 2:** `BatchManifest` provides exit codes (0=all success, 1=partial, 2=all failed)
- **Decision:** Sequential processing (Phase 7-01) for simplicity and error isolation

### From Audit Findings
- **Gap 1:** Revolut `adapter.BatchConvert()` delegates to legacy `BatchConvert()` function instead of `BatchProcessor`
- **Gap 2:** PDF `pdfBatchConvert()` and `consolidatePDFDirectory()` don't accept/use format/dateFormat parameters
- **Gap 3:** `BatchProcessor` lacks formatter configuration API

---

## Required Integration Points

### 1. BatchProcessor Formatter Configuration (CRITICAL)

**Current state:** `BatchProcessor.ProcessDirectory()` calls `common.ExportTransactionsToCSVWithLogger()` which uses the default writer (line 208 in `internal/batch/processor.go`).

**Required change:** Thread formatter options through `processFile()` and apply `WriteTransactionsToCSVWithFormatter()` instead.

**Design options:**
- **Option A (Recommended):** Add formatter fields to `BatchProcessor` struct, initialized at construction
- **Option B:** Pass formatter as parameter to `ProcessDirectory()`
- **Option C:** Create separate `FormattedBatchProcessor` struct

**Recommendation: Option A** — matches existing dependency injection pattern from Phase 7.

```go
type BatchProcessor struct {
    parser    parser.FullParser
    logger    logging.Logger
    formatter formatter.OutputFormatter  // NEW
    delimiter rune                       // NEW (optional override)
}

// NewBatchProcessor now accepts formatter
func NewBatchProcessor(p parser.FullParser, logger logging.Logger,
    fmt formatter.OutputFormatter, delimiter rune) *BatchProcessor { ... }
```

**Impact:** Single constructor parameter addition, backward compatible if formatter defaults to StandardFormatter.

### 2. Revolut Adapter BatchConvert Refactoring (HIGH PRIORITY)

**Current state (lines 68-71 of `internal/revolutparser/adapter.go`):**
```go
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    return BatchConvert(inputDir, outputDir)  // Delegates to legacy function
}
```

**Required change:** Use `BatchProcessor` composition like PDF adapter does.

**Target implementation (same pattern as PDF at lines 76-98 of `internal/pdfparser/adapter.go`):**
```go
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    processor := batch.NewBatchProcessor(a, a.GetLogger())
    manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
    // ... handle manifest and exit codes as needed
}
```

**Note:** Revolut legacy `BatchConvert()` function will become deprecated but can remain for backward compatibility initially.

### 3. PDF Batch and Consolidation Paths (HIGH PRIORITY)

**Current state:**
- `pdfBatchConvert()` (line 231): Calls `p.BatchConvert()` but ignores format/dateFormat flags
- `consolidatePDFDirectory()` (line 114): Uses `WriteTransactionsToCSVWithLogger()` (legacy writer, no formatter)

**Required changes:**

**3a. Batch path (pdfBatchConvert):**
- Accept `format` and `dateFormat` parameters from CLI
- Pass through to BatchProcessor (once formatter API added in step 1)

**3b. Consolidation path (consolidatePDFDirectory):**
- Accept `format` and `dateFormat` parameters from CLI
- Use `WriteTransactionsToCSVWithFormatter()` instead of `WriteTransactionsToCSVWithLogger()`
- Resolve formatter via `FormatterRegistry` before writing

**Example for consolidation (pseudocode):**
```go
func consolidatePDFDirectory(ctx context.Context, p parser.FullParser,
    inputDir, outputFile string, validate bool, logger logging.Logger,
    format string, dateFormat string) (int, error) {

    // ... collect all transactions into allTransactions ...

    // NEW: Get formatter from registry
    reg := formatter.NewFormatterRegistry()
    fmt, err := reg.Get(format)
    if err != nil {
        return 0, fmt.Errorf("unknown format: %w", err)
    }

    // NEW: Use formatter-aware writer
    delimiter := fmt.Delimiter()
    if err := common.WriteTransactionsToCSVWithFormatter(
        allTransactions, outputFile, logger, fmt, delimiter); err != nil {
        return 0, fmt.Errorf("failed to write CSV: %w", err)
    }

    return processedCount, nil
}
```

### 4. CLI Command Wiring (MEDIUM PRIORITY)

**Current state (`cmd/pdf/convert.go` line 96-100):**
```go
format, _ := cmd.Flags().GetString("format")
dateFormat, _ := cmd.Flags().GetString("date-format")
// ... format and dateFormat are fetched but NOT passed to pdfBatchConvert()

if fileInfo.IsDir() && batchMode {
    pdfBatchConvert(ctx, p, inputPath, root.SharedFlags.Output, logger)  // Missing format args
}
```

**Required change:** Thread format/dateFormat through function signatures.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|------------|-------------|-----|
| Multiple output format support | Custom if/else per format in CSV writer | Strategy pattern + `FormatterRegistry` (Phase 5) | Extensible, testable, avoids monolithic writer |
| Batch error aggregation | Loop with manual error tracking | `BatchProcessor` + `BatchManifest` (Phase 7) | Standardized manifest, exit codes, retries |
| Formatter selection | String-based switch with duplicated formatters | `FormatterRegistry.Get()` (Phase 5) | Single registry, extensible to custom formatters |
| Date format conversion | Manual string parsing per format | Go `time.Format()` with formatter-provided layout | Formatter encapsulates layout; consistent across parsers |

**Key insight:** All infrastructure already exists. Phase 9 is 95% glue work, not new infrastructure.

---

## Common Pitfalls

### Pitfall 1: Bypassing the Formatter Registry
**What goes wrong:** Hardcoding formatter selection in batch functions instead of using registry lookup.
**Why it happens:** "It's just batch mode, we only need one format" thinking leads to special cases that accumulate.
**How to avoid:** Always route through `FormatterRegistry.Get(formatName)` even if only one format is used.
**Warning signs:** Function that accepts `format` string but doesn't call `registry.Get()`.

### Pitfall 2: Not Passing Context for Cancellation
**What goes wrong:** Batch operations can't be cancelled mid-directory processing.
**Why it happens:** Forgetting that context needs to flow through batch functions even though some intermediate functions don't use it.
**How to avoid:** Pass `ctx` through all batch pipeline functions; check `ctx.Done()` in loops.
**Warning signs:** No context parameter in `consolidatePDFDirectory()` or `pdfBatchConvert()`.

### Pitfall 3: Inconsistent Delimiter Handling
**What goes wrong:** Batch output uses formatter's delimiter, but legacy code path uses hard-coded delimiter from `common.Delimiter` constant.
**Why it happens:** Two CSV writers with different delimiter sources causes surprises in format switching.
**How to avoid:** Always get delimiter from formatter; only override if explicitly requested.
**Warning signs:** Constants like `common.Delimiter` being used instead of `formatter.Delimiter()`.

### Pitfall 4: Forgetting to Update Manifest Structure
**What goes wrong:** Batch output is formatted correctly but manifest doesn't reflect which formatter was used.
**Why it happens:** Manifest is orthogonal to formatting; easy to add formatter support without adding it to manifest.
**How to avoid:** If formatter config is added to BatchProcessor, add it to manifest metadata for auditability.
**Warning signs:** No `Formatter` or `DateFormat` field in manifest JSON output.

### Pitfall 5: Breaking Backward Compatibility Silently
**What goes wrong:** Old code that calls `processor.ProcessDirectory()` without formatter parameter silently uses wrong format.
**Why it happens:** Adding required parameters without defaults breaks old callers.
**How to avoid:** Provide sensible defaults (StandardFormatter, standard delimiter) if formatter/delimiter not specified.
**Warning signs:** Constructor changes that are not backward compatible (no default values).

---

## Code Examples

### Example 1: Adding Formatter Support to BatchProcessor

**Source:** Composition of Phase 5 + Phase 7 patterns

```go
package batch

import (
    "fjacquet/camt-csv/internal/formatter"
    "fjacquet/camt-csv/internal/logging"
    "fjacquet/camt-csv/internal/parser"
)

// BatchProcessor with formatter support
type BatchProcessor struct {
    parser    parser.FullParser
    logger    logging.Logger
    formatter formatter.OutputFormatter  // NEW: Strategy pattern
}

// NewBatchProcessor creates a batch processor with optional formatter
func NewBatchProcessor(p parser.FullParser, logger logging.Logger,
    fmt formatter.OutputFormatter) *BatchProcessor {
    // Default to StandardFormatter if not provided
    if fmt == nil {
        fmt = formatter.NewStandardFormatter()
    }
    return &BatchProcessor{
        parser:    p,
        logger:    logger,
        formatter: fmt,
    }
}

// processFile uses formatter instead of legacy writer
func (bp *BatchProcessor) processFile(ctx context.Context, filePath, outputDir string) BatchResult {
    // ... validation and parsing unchanged ...

    delimiter := bp.formatter.Delimiter()
    if err := common.WriteTransactionsToCSVWithFormatter(
        transactions, outputPath, bp.logger, bp.formatter, delimiter); err != nil {
        result.Error = fmt.Sprintf("write_error: %v", err)
        return result
    }

    result.Success = true
    return result
}
```

**Why this works:** Minimal changes to existing BatchProcessor; formatter is already tested in Phase 5.

### Example 2: Wiring Format Flags Through PDF Consolidation

**Source:** Adapt `consolidatePDFDirectory()` from Phase 7 pattern

```go
// consolidatePDFDirectory now accepts format and dateFormat
func consolidatePDFDirectory(ctx context.Context, p parser.FullParser,
    inputDir, outputFile string, validate bool, logger logging.Logger,
    format string, dateFormat string) (int, error) {

    // ... existing code to collect allTransactions ...

    // NEW: Resolve formatter from registry
    formatterReg := formatter.NewFormatterRegistry()
    fmt, err := formatterReg.Get(format)
    if err != nil {
        logger.WithError(err).Error("Invalid format",
            logging.Field{Key: "format", Value: format})
        return 0, err
    }

    // Sort transactions
    sortTransactionsChronologically(allTransactions)

    // NEW: Use formatted writer
    delimiter := fmt.Delimiter()
    if err := internalcommon.WriteTransactionsToCSVWithFormatter(
        allTransactions, outputFile, logger, fmt, delimiter); err != nil {
        return processedCount, fmt.Errorf("failed to write CSV: %w", err)
    }

    return processedCount, nil
}
```

**Note:** `dateFormat` parameter added for signature consistency; may be used by formatter if Phase 5 adds date layout support (currently iCompta is hardcoded to `dd.MM.yyyy`).

### Example 3: Revolut Adapter Using BatchProcessor

**Source:** Replicate PDF adapter pattern (lines 76-98 of `internal/pdfparser/adapter.go`)

```go
// BatchConvert now uses BatchProcessor instead of legacy function
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    // Use BatchProcessor composition (same as PDF)
    processor := batch.NewBatchProcessor(a, a.GetLogger())

    manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
    if err != nil {
        // Config/permission error (not file-level errors)
        return 0, err
    }

    // Log summary
    a.GetLogger().Info("Batch processing completed",
        logging.Field{Key: "total", Value: manifest.TotalFiles},
        logging.Field{Key: "succeeded", Value: manifest.SuccessCount},
        logging.Field{Key: "failed", Value: manifest.FailureCount})

    // Write manifest
    manifestPath := filepath.Join(outputDir, ".manifest.json")
    if err := manifest.WriteManifest(manifestPath); err != nil {
        a.GetLogger().WithError(err).Warn("Failed to write manifest")
    }

    return manifest.SuccessCount, nil
}
```

**Impact:** Legacy `BatchConvert()` function can be deprecated. Revolut gets manifest + exit codes for free.

### Example 4: CLI Wiring (cmd/pdf/convert.go)

**Source:** Current code with additions highlighted

```go
func pdfFunc(cmd *cobra.Command, args []string) {
    ctx := cmd.Context()
    logger := root.GetLogrusAdapter()

    inputPath := root.SharedFlags.Input
    format, _ := cmd.Flags().GetString("format")      // Already present
    dateFormat, _ := cmd.Flags().GetString("date-format")  // Already present

    p, err := appContainer.GetParser(container.PDF)
    if err != nil {
        logger.Fatalf("Error getting PDF parser: %v", err)
    }

    fileInfo, err := os.Stat(inputPath)
    if err != nil {
        logger.Fatalf("Error accessing input path: %v", err)
    }

    batchMode, _ := cmd.Flags().GetBool("batch")

    if fileInfo.IsDir() && batchMode {
        // NEW: Pass format and dateFormat to batch function
        pdfBatchConvert(ctx, p, inputPath, root.SharedFlags.Output,
            logger, format, dateFormat)  // NEW parameters
    } else if fileInfo.IsDir() {
        // NEW: Pass format and dateFormat to consolidation function
        count, err := consolidatePDFDirectory(ctx, p, inputPath,
            root.SharedFlags.Output, root.SharedFlags.Validate, logger,
            format, dateFormat)  // NEW parameters
        if err != nil {
            logger.Fatalf("Error consolidating PDFs: %v", err)
        }
        logger.Infof("Consolidated %d PDF files successfully!", count)
    } else {
        // Existing single-file flow (unchanged)
        common.ProcessFile(ctx, p, inputPath, root.SharedFlags.Output,
            root.SharedFlags.Validate, root.Log, appContainer, format, dateFormat)
        root.Log.Info("PDF to CSV conversion completed successfully!")
    }
}
```

**New function signatures:**
```go
func pdfBatchConvert(ctx context.Context, p parser.FullParser, inputDir, outputDir string,
    logger logging.Logger, format string, dateFormat string) { ... }

func consolidatePDFDirectory(ctx context.Context, p parser.FullParser,
    inputDir, outputFile string, validate bool, logger logging.Logger,
    format string, dateFormat string) (int, error) { ... }
```

---

## State of the Art

| Old Approach | Current Approach (Phase 9) | When Changed | Impact |
|--------------|---------------------------|---|---------|
| Legacy `BatchConvert()` functions in each parser | `BatchProcessor` composition pattern | Phase 7 (PDF), Phase 9 (Revolut) | Standardized manifests, exit codes across all parsers |
| Batch output hardcoded to standard format | Batch respects `--format` flag via formatter integration | Phase 9 | Users can batch-convert to iCompta format |
| Consolidation mode ignores format flag | Consolidation uses formatter pipeline | Phase 9 | Consistent behavior across batch, consolidation, single-file |
| No manifest in Revolut batch | Manifest generated by BatchProcessor | Phase 9 | Users can audit which files succeeded/failed in batch runs |

**Deprecated/outdated:**
- `internal/revolutparser/BatchConvert()` standalone function — still works but Phase 9 routes through adapter's BatchConvert which uses BatchProcessor
- `common.WriteTransactionsToCSVWithLogger()` in batch paths — still works but batch paths switch to `WriteTransactionsToCSVWithFormatter()` for format support

---

## Open Questions

1. **Date format handling in batch mode:**
   - Current: ICompta formatter hardcodes `dd.MM.yyyy` (from Phase 5 decision D-07)
   - Question: Should `--date-format` override formatter's preferred date format?
   - Recommendation: Phase 5 evidence shows `--date-format` is user-provided Go layout; pass to formatter. If formatter doesn't support custom layouts yet, add that in Phase 9 enhancement (LOW priority for MVP).

2. **Backward compatibility of BatchProcessor API:**
   - Current: Phase 7 code doesn't pass formatter
   - Question: Should we provide default formatter if none specified, or make it required?
   - Recommendation: Provide default (StandardFormatter) to ensure old code still works.

3. **Manifest metadata:**
   - Current: `BatchManifest` doesn't record which formatter was used
   - Question: Should manifest include formatter/dateFormat metadata for auditability?
   - Recommendation: Add optional metadata fields for audit trail (LOW priority for MVP).

---

## Sources

### Primary (HIGH confidence)

- **Phase 5 verification:** `.planning/phases/05-output-framework/05-VERIFICATION.md` — Confirms OutputFormatter strategy pattern, iCompta format implementation, registry pattern
- **Phase 7 verification:** `.planning/phases/07-batch-infrastructure/07-VERIFICATION.md` — Confirms BatchProcessor design, manifest structure, exit code semantics
- **v1.2 Audit findings:** `.planning/v1.2-MILESTONE-AUDIT.md` — Documents exact gaps: Revolut legacy batch, PDF batch/consolidation ignore format flags, BatchProcessor lacks formatter API
- **Codebase inspection:**
  - `internal/batch/processor.go` (lines 1-226) — Current BatchProcessor implementation
  - `internal/formatter/formatter.go` (lines 1-100) — OutputFormatter interface + registry
  - `cmd/pdf/convert.go` (lines 1-287) — Batch and consolidation paths
  - `internal/revolutparser/adapter.go` (lines 68-71) — Legacy batch delegation
  - `internal/pdfparser/adapter.go` (lines 76-98) — BatchProcessor usage pattern to replicate
  - `internal/common/csv.go` (lines 72-172) — WriteTransactionsToCSVWithFormatter implementation already exists

### Secondary (MEDIUM confidence)

- **Project decisions:** `.planning/reference/v1.2-decisions.md` (D-05 through D-10) — Confirms cross-parser formatter design, iCompta format specifics
- **CLAUDE.md:** Architecture overview confirms segregated interfaces, dependency injection patterns

---

## Metadata

**Confidence breakdown:**
- **Standard Stack:** HIGH — All infrastructure already implemented in Phases 5 and 7
- **Architecture:** HIGH — Patterns proven in Phase 7 PDF adapter; Revolut just needs to replicate
- **Pitfalls:** HIGH — Based on audit findings and code inspection
- **Integration points:** HIGH — Exact gaps documented in audit report

**Research date:** 2026-02-16
**Valid until:** 2026-02-23 (stable domain, no active upstream changes expected)

**Next step:** Planner will create 3 implementation plans (09-01, 09-02, 09-03) targeting the three integration points with specific tasks and testing strategy.
