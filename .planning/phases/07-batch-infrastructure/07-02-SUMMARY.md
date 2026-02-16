---
phase: 07-batch-infrastructure
plan: 02
subsystem: batch
tags: [cli, batch-processing, exit-codes, manifest, user-experience]
dependency-graph:
  requires: [BatchProcessor, BatchManifest, parser.FullParser, logging.Logger]
  provides: [directory batch support for all parsers, semantic exit codes, manifest-based error reporting]
  affects: [all 6 CLI parser commands, shell scripting workflows]
tech-stack:
  added: [directory detection in CLI commands, manifest loading, exit code semantics]
  patterns: [interface-based batch detection, fallback error handling, standardized CLI behavior]
key-files:
  created: []
  modified:
    - internal/pdfparser/adapter.go
    - internal/pdfparser/pdfparser_test.go
    - cmd/pdf/convert.go
    - cmd/camt/convert.go
    - cmd/revolut/convert.go
    - cmd/selma/convert.go
    - cmd/debit/convert.go
    - cmd/revolut-investment/convert.go
decisions:
  - "PDF parser migrated from stub to BatchProcessor composition pattern"
  - "PDF command supports both batch mode (--batch flag) and consolidation mode (default for directories)"
  - "All CLI commands now detect directory input and invoke BatchConvert automatically"
  - "Manifest loading happens in CLI layer, not parser layer"
  - "Exit code fallback strategy: if manifest unreadable, exit based on success count"
  - "Batch function uses interface assertion for backward compatibility"
metrics:
  duration: 854
  completed: 2026-02-16
---

# Phase 7 Plan 02: CLI Batch Support & Exit Code Standardization Summary

**One-liner:** Unified batch processing with manifest-based exit codes across all 6 parser CLI commands

## What Was Built

Integrated the BatchProcessor infrastructure (from Plan 01) into all CLI commands, providing consistent batch processing behavior with semantic exit codes for shell scripting:

1. **PDF Parser BatchConvert Implementation** (Task 1)
   - Replaced "not implemented" stub with BatchProcessor composition
   - Generates `.manifest.json` with batch results
   - Added comprehensive tests (batch success, empty directory, invalid directory)
   - All tests pass including race detection

2. **CLI Command Batch Support** (Task 2)
   - Added directory detection to all 6 commands (PDF, CAMT, Revolut, Selma, Debit, Revolut Investment)
   - Each command now invokes BatchConvert when given directory input
   - Load manifest after batch processing and exit with semantic codes
   - PDF command gets `--batch` flag to choose between batch and consolidation modes

3. **Standardized Exit Code Behavior**
   - Exit 0: All files succeeded
   - Exit 1: Partial success (some files failed)
   - Exit 2: All files failed or no files processed
   - Fallback: If manifest unreadable, exit based on success count

## Implementation Highlights

**PDF Adapter (Task 1):**
```go
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    processor := batch.NewBatchProcessor(a, a.GetLogger())
    manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
    if err != nil {
        return 0, err
    }
    // Write manifest
    manifestPath := filepath.Join(outputDir, ".manifest.json")
    manifest.WriteManifest(manifestPath)
    return manifest.SuccessCount, nil
}
```

**CLI Batch Function Pattern (Task 2):**
```go
func batchConvert(ctx context.Context, p interface{}, inputDir, outputDir string, logger logging.Logger) {
    batchConverter, ok := p.(interface {
        BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error)
    })
    if !ok {
        logger.Error("Parser does not support batch conversion")
        os.Exit(1)
    }

    count, err := batchConverter.BatchConvert(ctx, inputDir, outputDir)
    if err != nil {
        logger.WithError(err).Error("Batch conversion failed")
        os.Exit(1)
    }

    // Load manifest
    manifestPath := filepath.Join(outputDir, ".manifest.json")
    manifestData, err := os.ReadFile(manifestPath)
    if err != nil {
        logger.WithError(err).Warn("Could not read manifest")
        if count == 0 {
            os.Exit(1)
        }
        return
    }

    var manifest batch.BatchManifest
    json.Unmarshal(manifestData, &manifest)

    // Log summary
    logger.Info(fmt.Sprintf("Batch complete: %d/%d files succeeded",
        manifest.SuccessCount, manifest.TotalFiles))

    if manifest.FailureCount > 0 {
        logger.Warn(fmt.Sprintf("%d files failed (see %s for details)",
            manifest.FailureCount, manifestPath))
    }

    // Exit with semantic code
    if manifest.ExitCode() != 0 {
        os.Exit(manifest.ExitCode())
    }
}
```

**Directory Detection:**
```go
fileInfo, err := os.Stat(inputPath)
if err != nil {
    logger.Fatalf("Error accessing input path: %v", err)
}

if fileInfo.IsDir() {
    batchConvert(ctx, p, inputPath, outputPath, logger)
} else {
    common.ProcessFile(ctx, p, inputPath, outputPath, ...)
}
```

## Deviations from Plan

None - plan executed exactly as written.

## Test Results

All tests pass for affected packages:
- **internal/pdfparser**: 91 tests (includes 3 new BatchConvert tests)
- **cmd/pdf, cmd/camt, cmd/revolut, cmd/selma, cmd/debit, cmd/revolut-investment**: All existing tests pass
- **Build**: Successful (`make build`)
- **Race detector**: No races detected

**New Test Coverage:**
- `TestAdapterBatchConvert`: Multiple files with mock extractor, verifies manifest creation
- `TestAdapterBatchConvert_EmptyDirectory`: Zero files, verifies manifest still created
- `TestAdapterBatchConvert_InvalidInputDirectory`: Error handling for missing directory

## Integration Points

**Shell Scripting Example:**
```bash
#!/bin/bash
# Process directory of PDFs with error handling
./camt-csv pdf --batch -i pdfs/ -o csvs/

case $? in
    0) echo "All files succeeded" ;;
    1) echo "Partial success - check .manifest.json" ;;
    2) echo "All files failed" ;;
esac
```

**Manifest Inspection:**
```bash
# After batch operation
jq '.results[] | select(.success == false)' csvs/.manifest.json
# Shows only failed files with error messages
```

## CLI Command Updates

**Before (single file only):**
```bash
camt-csv pdf -i file.pdf -o output.csv  # Works
camt-csv pdf -i pdfs/ -o output.csv     # Error or consolidation
```

**After (automatic batch detection):**
```bash
camt-csv pdf -i file.pdf -o output.csv      # Single file mode
camt-csv pdf --batch -i pdfs/ -o csvs/      # Batch mode (1 PDF → 1 CSV)
camt-csv pdf -i pdfs/ -o output.csv         # Consolidation mode (N PDFs → 1 CSV)
```

**Applies to all parsers:**
- `camt-csv camt -i camt_files/ -o output_dir/`
- `camt-csv revolut -i revolut_csvs/ -o output_dir/`
- `camt-csv selma -i selma_files/ -o output_dir/`
- `camt-csv debit -i debit_files/ -o output_dir/`
- `camt-csv revolut-investment -i files/ -o output_dir/`

## Benefits

**For Users:**
- 🚀 **One command for multiple files**: No more manual loops
- 📊 **Detailed error reports**: Manifest shows exactly which files failed and why
- 🔧 **Shell scriptable**: Exit codes enable conditional logic in automation
- 📁 **Consistent behavior**: All parsers work the same way

**For Developers:**
- 🎯 **Single source of truth**: BatchProcessor handles all batch logic
- 🧪 **Testable**: Interface-based design allows easy mocking
- 🛡️ **Error isolation**: Individual file failures don't stop batch
- 📦 **Reusable**: Pattern copied across all 6 commands

## Ready for Phase 8

Batch infrastructure now complete. All parsers support:
1. Individual file conversion (existing behavior)
2. Batch directory processing (new)
3. Manifest generation with per-file results
4. Semantic exit codes for automation

**Phase 7 Complete**: 2/2 plans ✅
- 07-01: BatchProcessor + BatchManifest infrastructure ✅
- 07-02: CLI integration with exit codes ✅

**Next Phase**: Phase 8 (output standardization and iCompta format refinement)

## Commits

- `88dcc24`: feat(07-02): implement PDF parser BatchConvert using BatchProcessor
- `1df8402`: feat(07-02): add batch directory support with manifest exit codes to all CLI commands

## Self-Check: PASSED

**Modified files verified:**
```bash
✅ internal/pdfparser/adapter.go (BatchConvert implemented)
✅ internal/pdfparser/pdfparser_test.go (3 new tests added)
✅ cmd/pdf/convert.go (batch mode + manifest exit codes)
✅ cmd/camt/convert.go (batch mode + manifest exit codes)
✅ cmd/revolut/convert.go (batch mode + manifest exit codes)
✅ cmd/selma/convert.go (batch mode + manifest exit codes)
✅ cmd/debit/convert.go (batch mode + manifest exit codes)
✅ cmd/revolut-investment/convert.go (batch mode + manifest exit codes)
```

**Commits verified:**
```bash
✅ 88dcc24 (Task 1: PDF BatchConvert)
✅ 1df8402 (Task 2: CLI batch support)
```

**Test execution:**
```bash
✅ All parser tests pass (internal/pdfparser)
✅ All command tests pass (cmd/*)
✅ Build successful (make build)
✅ No race conditions detected
```

All plan objectives achieved. Batch infrastructure fully integrated into CLI commands.
