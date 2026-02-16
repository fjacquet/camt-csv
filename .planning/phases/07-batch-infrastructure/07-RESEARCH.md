# Phase 7: Batch Infrastructure - Research

**Researched:** 2026-02-16
**Domain:** Go CLI batch processing patterns, error handling, exit codes, manifest generation
**Confidence:** HIGH

## Summary

Phase 7 standardizes batch processing across all parsers with comprehensive error reporting. The project has **partial** batch implementations in CAMT, Revolut, Selma, Debit, and Revolut Investment parsers, but they lack standardized error manifests and unified exit code semantics. The PDF parser has no batch support at all—it only has a single-directory consolidation mode in `cmd/pdf/`.

The core challenge is not inventing batch infrastructure, but **standardizing** what already partially exists. Key patterns are already present:
- Continue-on-failure (skip individual files) ✓
- Chronological transaction sorting ✓
- File validation before processing ✓
- Error logging per file ✓
- Aggregation/consolidation ✓

What's **missing** is standardization:
1. **Exit code semantics** (0=all success, 1=partial, 2=all failed) — not currently implemented
2. **Manifest generation** (JSON/YAML summary showing succeeded/failed files) — not present
3. **PDF batch support** — only consolidation mode exists
4. **Unified error reporting interface** — each parser implements this differently
5. **Batch error types** — no structured batch-specific error classification

**Primary recommendation:** Create a `BatchProcessor` abstraction in `internal/batch/` that all parsers use via composition. This avoids replicating error handling logic across 6 parser adapters. Then standardize exit codes at the CLI command layer.

---

## Standard Stack

### Core Go Patterns for Batch Processing
| Pattern | Version/Detail | Purpose | Why Standard |
|---------|---|---------|--------------|
| `sync/errgroup` | Go 1.13+ stdlib | Parallel task execution with error collection | Prevents task spawn storms; Go idiom |
| Context with cancellation | Go 1.7+ stdlib | Batch cancellation/timeout control | Standard Go concurrency mechanism |
| Struct composition | Go idiom | Reusable batch behavior across parsers | Avoids code duplication; follows DRY |
| Custom error types | Go idiom | Structured error reporting per file | Allows selective error handling |
| JSON marshaling | `encoding/json` stdlib | Manifest serialization | Human-readable + machine-parseable |

### Supporting Libraries (Already in Project)
| Library | Version | Purpose | Used For |
|---------|---------|---------|----------|
| `github.com/sirupsen/logrus` | v1.x | Structured logging | Already used; consistency |
| `github.com/shopspring/decimal` | v1.x | Precise currency amounts | Already in models.Transaction |
| `github.com/spf13/cobra` | v1.x | CLI commands | Already framework for all commands |

### Not Needed (Already Decided)
| Standard Tool | Why Not | What We Use Instead |
|---|---|---|
| Job queues (Bull, RabbitMQ, Kafka) | Overkill for CLI tool | In-memory `errgroup` |
| Distributed transaction coordination | Single process | Sequential/parallel iteration |
| Retry logic with exponential backoff | File format errors are unrecoverable | Single attempt per file |

---

## Architecture Patterns

### Pattern 1: Batch Result Struct (Manifest Structure)

**What:** Structured representation of batch operation outcomes per file, aggregated across all files.

**When to use:** After each batch operation completes; serialized to JSON/YAML for reporting.

**Recommendation:**
```go
// BatchResult represents the result of a single file's processing
type BatchResult struct {
    FilePath    string
    Success     bool
    Error       string      // Only if Success=false
    FileName    string
    RecordCount int         // Transactions extracted
}

// BatchManifest represents the aggregate result of a batch operation
type BatchManifest struct {
    TotalFiles      int
    SuccessCount    int
    FailureCount    int
    Results         []BatchResult   // Per-file details
    Duration        time.Duration
    ProcessedAt     time.Time
    ExitCode        int             // 0=all success, 1=partial, 2=all failed
}

// WriteManifest writes manifest to JSON file
func (bm *BatchManifest) WriteManifest(filePath string) error { ... }

// ExitCode returns the appropriate shell exit code
func (bm *BatchManifest) ExitCode() int {
    if bm.FailureCount == 0 { return 0 }
    if bm.SuccessCount == 0 { return 2 }
    return 1
}
```

**Source:** Go CLI best practices; similar to Terraform, Docker Compose batch operations.

---

### Pattern 2: Reusable BatchProcessor (Composition-Based)

**What:** A pluggable batch processor that wraps any parser and handles standardized error collection/reporting.

**When to use:** For any parser's batch mode, especially PDF which currently has no batch implementation.

**Architecture:**
```go
// BatchProcessor handles standardized batch processing for any parser
type BatchProcessor struct {
    parser   parser.FullParser
    logger   logging.Logger
    manifest *BatchManifest
}

// NewBatchProcessor creates a processor for a given parser
func NewBatchProcessor(p parser.FullParser, logger logging.Logger) *BatchProcessor { ... }

// ProcessDirectory processes all files in a directory, collecting results
// Returns manifest and never stops on first error
func (bp *BatchProcessor) ProcessDirectory(ctx context.Context, inputDir, outputDir string) (*BatchManifest, error) {
    manifest := &BatchManifest{ProcessedAt: time.Now()}

    // Phase 1: Discover and filter files
    files := bp.discoverFiles(inputDir)
    manifest.TotalFiles = len(files)

    // Phase 2: Process each file independently
    for _, file := range files {
        result := bp.processFile(ctx, file, outputDir)
        manifest.Results = append(manifest.Results, result)

        if result.Success {
            manifest.SuccessCount++
        } else {
            manifest.FailureCount++
        }
    }

    manifest.ExitCode = manifest.computeExitCode()
    return manifest, nil  // Never error; success/failure is in manifest
}

// processFile processes a single file, returning a result struct (never panics)
func (bp *BatchProcessor) processFile(ctx context.Context, filePath string, outputDir string) BatchResult {
    result := BatchResult{
        FilePath: filePath,
        FileName: filepath.Base(filePath),
    }

    // Validate
    valid, err := bp.parser.ValidateFormat(filePath)
    if err != nil || !valid {
        result.Success = false
        result.Error = "validation_failed"
        return result
    }

    // Parse
    file, _ := os.Open(filePath)
    transactions, err := bp.parser.Parse(context.Background(), file)
    file.Close()

    if err != nil {
        result.Success = false
        result.Error = err.Error()
        return result
    }

    result.RecordCount = len(transactions)
    result.Success = true
    return result
}
```

**Why this pattern:**
- Eliminates duplicate batch logic across 6 parser adapters
- Allows PDF parser to inherit batch mode for free
- All parsers get standardized manifest generation
- Exit code logic centralized, consistent

**Composition advantage over inheritance:** Each parser adapter remains lightweight; batch concerns isolated from parsing concerns.

---

### Pattern 3: Unified Batch Error Types

**What:** Structured error classification for batch scenarios (distinct from parse errors).

**When to use:** When logging/reporting individual file failures; helps users distinguish root causes.

**Recommendation:**
```go
// BatchFileError represents a failure mode for a single file
type BatchFileError struct {
    FilePath string
    Category string // "validation", "parsing", "io", "write"
    Message  string
}

// Return from processor:
// - Validation failure: "validation_failed"
// - Parse error: "parse_error: <details>"
// - IO error: "io_error: <details>"
// - Write error: "write_error: <details>"
```

---

### Anti-Patterns to Avoid

- **Partial manifest generation:** Don't build manifests incrementally as you process. Collect all results first, then assemble manifest. Reason: Partial writes corrupt manifest on crash.
- **Batch error stops processing:** Never return early from batch loop on first error. Always process all files, collect errors. Reason: Users expect all valid files processed even if some fail.
- **Mixed exit codes with error logs:** Don't use `logger.Fatal()` for batch errors. Exit codes are exit codes; logging is logging. Return manifest with exit code, let CLI layer decide `os.Exit()`.
- **Inconsistent file filtering:** Don't filter files per-parser in batch mode. Standardize patterns: XML files end with `.xml`, CSV files end with `.csv`, PDF files end with `.pdf`. Reason: Predictability.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parallel file processing with error collection | Custom goroutine pools | `sync/errgroup` from stdlib | Handles cancellation, error aggregation, prevents leaks |
| Exit code semantics for batch ops | Home-grown state machine | Pattern (0=ok, 1=partial, 2=fail) + simple logic | Matches Unix conventions; widely recognized |
| Manifest serialization (JSON/CSV) | Custom marshaling code | `encoding/json` (stdlib) + simple struct tags | Type-safe; handles edge cases; standard library |
| Batch state tracking | Custom structs everywhere | Single `BatchManifest` struct + `BatchProcessor` | Single source of truth; prevents sync bugs |

**Key insight:** Batch infrastructure is largely solved in Go's stdlib (`context`, `sync/errgroup`, `encoding/json`). The challenge is composition—reusing proven patterns across multiple parsers without code duplication.

---

## Common Pitfalls

### Pitfall 1: No Differentiation Between Partial and Total Failure

**What goes wrong:** Exit code 1 is used for both "some files succeeded" and "all files failed". User doesn't know if batch is partially or completely broken.

**Why it happens:** Developers treat exit code as binary (error/no-error) instead of a spectrum.

**How to avoid:** Use exit codes as specified in Phase 7 requirements:
- 0 = all files succeeded
- 1 = some succeeded, some failed (partial success)
- 2 = all files failed OR no files found

**Warning signs:** Logs show mixed success/failure but exit code is always 1; user can't script recovery.

---

### Pitfall 2: Manifest Not Written on Failure

**What goes wrong:** Batch process fails halfway through. No manifest generated. User has no record of what was processed.

**Why it happens:** Error handling returns early without writing manifest.

**How to avoid:** Always write manifest before exiting, even on error. Structure like:
```go
manifest, _ := processor.ProcessDirectory(ctx, inputDir, outputDir)  // Never errors
if err := manifest.WriteManifest(manifestPath); err != nil {
    logger.WithError(err).Warn("Failed to write manifest (but batch results stand)")
}
os.Exit(manifest.ExitCode)  // Use manifest exit code
```

**Warning signs:** Batch fails; no `.manifest.json` produced; user must manually audit output dir.

---

### Pitfall 3: Reprocessing Same Directory Creates Duplicate Results

**What goes wrong:** User runs batch twice on same input dir without moving input files. Output files accumulate; old files remain.

**Why it happens:** No deduplication or overwrite handling per parser.

**How to avoid:** Each parser's batch mode should:
1. Document whether files are moved/deleted (Revolut CSV: moves files)
2. Document whether output overwrites (CAMT: overwrites .csv files)
3. Or generate unique output names: `{account}_{start}_{end}_{timestamp}.csv`

**Current state:** CAMT consolidation generates unique names; Revolut moves input files. Standard this pattern.

**Warning signs:** Running batch twice produces different output; no clear documentation of idempotency.

---

### Pitfall 4: File Discovery Race Condition

**What goes wrong:** Between "discover files" and "process file", files are deleted or renamed. Batch silently skips; no manifest record.

**Why it happens:** No locking; loose coupling between discovery and processing.

**How to avoid:**
1. Snapshot file list early
2. Log skipped files (deleted-after-discovery) as warnings
3. Include skipped count in manifest

**Current state:** None of the parsers handle this. Low severity for CLI tool (single process, no concurrent access), but good practice.

---

### Pitfall 5: Memory Exhaustion on Large Batches

**What goes wrong:** Batch loads all transactions into memory, crashes on very large batch.

**Why it happens:** `[]Transaction` accumulation in `BatchManifest.Results` or aggregator.

**How to avoid:** For v1 of Phase 7, acceptable limit is "all files must fit in RAM". Document this. Revisit in v1.2 if needed (streaming CSV writer).

**Current state:** PDF consolidation loads all transactions into memory. CAMT batch uses aggregator which also accumulates. Acceptable for now.

---

## Code Examples

### Example 1: Implementing Batch Mode for PDF Parser

**Current state:** PDF has `cmd/pdf/consolidatePDFDirectory()` but `adapter.BatchConvert()` returns `not implemented`.

**Recommended change:**
```go
// In internal/pdfparser/adapter.go

// BatchConvert implements the batch converter interface using the new standardized processor
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    processor := batch.NewBatchProcessor(a, a.GetLogger())

    manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
    if err != nil {
        // processor never errors; err indicates config/permission issues
        return 0, err
    }

    // Log manifest summary
    a.GetLogger().Infof("Batch processing completed: %d total, %d succeeded, %d failed",
        manifest.TotalFiles, manifest.SuccessCount, manifest.FailureCount)

    return manifest.SuccessCount, nil  // Return count of succeeded files
}
```

**Result:** PDF parser now supports full batch mode with exit codes and manifest.

**Source:** Existing pattern from Revolut/CAMT adapters, elevated to shared abstraction.

---

### Example 2: Manifest Generation

**Pseudocode:**
```go
// In internal/batch/processor.go

// WriteManifest serializes batch results to JSON
func (bm *BatchManifest) WriteManifest(filePath string) error {
    data, err := json.MarshalIndent(bm, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal manifest: %w", err)
    }

    return os.WriteFile(filePath, data, 0644)
}

// Example output:
// {
//   "total_files": 5,
//   "success_count": 4,
//   "failure_count": 1,
//   "exit_code": 1,
//   "processed_at": "2026-02-16T10:30:00Z",
//   "duration_seconds": 45.123,
//   "results": [
//     {
//       "file_path": "/input/camt_1.xml",
//       "file_name": "camt_1.xml",
//       "success": true,
//       "record_count": 152
//     },
//     {
//       "file_path": "/input/camt_2.xml",
//       "file_name": "camt_2.xml",
//       "success": false,
//       "error": "parse_error: invalid CAMT structure"
//     }
//   ]
// }
```

---

### Example 3: Exit Code Handling in CLI

**In cmd commands:**
```go
func batchFunc(cmd *cobra.Command, args []string) {
    // ... setup ...

    manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
    if err != nil {
        logger.Fatalf("Batch failed: %v", err)  // Fatal exits 1
    }

    // Write manifest for user inspection
    manifestPath := filepath.Join(outputDir, ".manifest.json")
    if err := manifest.WriteManifest(manifestPath); err != nil {
        logger.WithError(err).Warn("Could not write manifest")
    }

    // Log summary
    logger.Infof("Batch complete: %d/%d succeeded",
        manifest.SuccessCount, manifest.TotalFiles)

    // Exit with semantic code (from manifest)
    if manifest.ExitCode != 0 {
        os.Exit(manifest.ExitCode)  // 1 or 2
    }
}
```

**Result:** User can shell-script batch operations:
```bash
camt-csv batch -i input/ -o output/
if [ $? -eq 0 ]; then
    echo "All succeeded"
elif [ $? -eq 1 ]; then
    echo "Some failed; see .manifest.json"
elif [ $? -eq 2 ]; then
    echo "All failed"
fi
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| Each parser had own batch loop | Moving to shared BatchProcessor | Phase 7 | Consistency, maintainability |
| No manifest generation | JSON manifest per batch run | Phase 7 | User visibility into failures |
| Exit code always 1 on error | Semantic codes (0/1/2) | Phase 7 | Better shell scripting |
| PDF only had consolidation mode | PDF gets full batch interface | Phase 7 | Feature parity |

**Deprecated/outdated:**
- ~~Custom error handling per parser in batch~~ → Use BatchFileError
- ~~Batch implementation details scattered~~ → Centralize in `internal/batch/`
- ~~No manifest generation~~ → Standard JSON manifest

---

## Open Questions

1. **Should batch use parsers in parallel or sequential mode?**
   - Current: Sequential (one file at a time)
   - Parallel would be faster but requires thread-safe categorizer
   - Recommendation: **Keep sequential for Phase 7 (simpler)**. Parallelize in Phase 8 if needed.

2. **What if output file already exists? Overwrite or error?**
   - CAMT currently: Overwrites silently
   - Revolut currently: Moves input (no collision)
   - PDF currently: Should overwrite (to maintain consolidation semantics)
   - Recommendation: **Standardize to overwrite silently** (idempotent). Document clearly.

3. **Should manifest be mandatory or optional output?**
   - Current: Optional (not generated)
   - Recommended: **Always generated** in `{outputDir}/.manifest.json`
   - Reason: Users always want to know which files succeeded/failed

4. **Should batch process hidden files (.csv, .xml)?**
   - Recommendation: **No**. Only process files with standard extensions. Skip dot-files.

5. **What if no files found in input directory?**
   - Current behavior: Different per parser (some warn, some error)
   - Recommendation: **Exit 2 (all failed), generate manifest with 0 files, don't error**. This is expected for empty directories.

---

## Sources

### Primary (HIGH confidence)
- **Codebase inspection:** Phase 6 existing batch implementations in CAMT, Revolut, Selma, Debit, Revolut Investment adapters
  - `/Users/fjacquet/Projects/camt-csv/internal/camtparser/adapter.go` — BatchConvert implementation
  - `/Users/fjacquet/Projects/camt-csv/internal/revolutparser/revolutparser.go` — BatchConvertWithLogger
  - `/Users/fjacquet/Projects/camt-csv/cmd/batch/batch.go` — Aggregation-based batch processor
  - `/Users/fjacquet/Projects/camt-csv/internal/batch/aggregator.go` — File grouping and consolidation
  - `/Users/fjacquet/Projects/camt-csv/cmd/pdf/convert.go` — PDF consolidation mode (demonstrates pattern)

- **Go stdlib documentation:** `context`, `sync/errgroup`, `encoding/json`, `os.Exit` semantics
- **CLAUDE.md project guidelines:** Parser interface design, error handling patterns, testing conventions

### Secondary (MEDIUM confidence)
- **Unix conventions:** Exit code semantics (0=success, 1=error, 2=misuse)
- **CLI best practices:** Manifest generation, continue-on-failure patterns (Terraform, Docker Compose)

### Tertiary (LOW confidence)
- None; all findings verified in codebase or stdlib.

---

## Metadata

**Confidence breakdown:**
- **Standard stack:** HIGH — Go stdlib patterns well-established; existing codebase implements most
- **Architecture:** HIGH — Multiple working batch implementations in codebase; patterns clear
- **Pitfalls:** HIGH — Extracted from existing code gaps and common CLI mistakes
- **Code examples:** HIGH — Based directly on existing implementations

**Research date:** 2026-02-16
**Valid until:** 2026-03-16 (stable domain; no rapid changes expected)
**Phase:** 7 of ~12
