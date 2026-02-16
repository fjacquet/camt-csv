# v1.2 Milestone: Cross-Phase Integration Audit Report

**Audit Date**: 2026-02-16
**Auditor**: Integration Verification System
**Status**: CRITICAL WIRING GAPS IDENTIFIED

---

## Executive Summary

**Finding**: The v1.2 milestone has **all 4 phases passing individual verification** (unit tests, build, component tests), but **cross-phase integration has 3 critical gaps** that violate the E2E flow requirements.

### Impact
- ✓ **Single file conversion with formatters** → WORKS
- ✗ **Batch PDF conversion with formatters** → BROKEN (flags ignored)
- ✗ **Batch Revolut conversion with formatters** → BROKEN (no formatter support, no manifest)
- ✗ **PDF consolidation with formatters** → BROKEN (flags ignored)
- ✓ **AI categorization with rate limits** → WORKS (but bypassed in batch flows)

### Root Cause
Phases 5-7 were executed sequentially with different scopes. Phase 7 (Batch Infrastructure) was not retrofitted into Phase 6 (Revolut), and formatter configuration cannot flow through BatchProcessor API.

---

## Critical Gaps

### GAP 1: Revolut Batch Uses Pre-Phase-7 Legacy Logic

**Severity**: CRITICAL  
**Files**: 
- `/Users/fjacquet/Projects/camt-csv/internal/revolutparser/adapter.go:70`
- `/Users/fjacquet/Projects/camt-csv/internal/revolutparser/revolutparser.go:442-536`

**Problem**:
```go
// adapter.go line 70 - delegates to legacy function
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
    return BatchConvert(inputDir, outputDir)  // ← NOT using BatchProcessor
}

// revolutparser.go line 442-536 - custom batch logic with no formatter support
func BatchConvertWithLogger(...) {
    // Loops through files manually
    WriteToCSVWithLogger(transactions, outputPath, logger)  // ← Hardcoded format
    // No manifest.json generation
}
```

**Impact**: 
- User command: `camt-csv revolut -i csv_dir/ -o out_dir/ --format icompta`
- Expected: iCompta format (10 cols, semicolon), manifest.json with exit codes
- Actual: Standard format (35 cols, comma), NO manifest, exit code always 0 on success
- User experience: Flags are silently ignored

**Violates**: Phase 7 contract (BatchManifest standardization)

---

### GAP 2: PDF Batch Ignores Formatter Flags

**Severity**: HIGH  
**Files**: 
- `/Users/fjacquet/Projects/camt-csv/cmd/pdf/convert.go:92-96`
- `/Users/fjacquet/Projects/camt-csv/cmd/pdf/convert.go:231-268`

**Problem**:
```go
// Line 92-96: Flags retrieved but not passed to batch function
batchMode, _ := cmd.Flags().GetBool("batch")
if fileInfo.IsDir() && batchMode {
    pdfBatchConvert(ctx, p, inputPath, root.SharedFlags.Output, logger)
    // ↑ format/dateFormat NOT passed here
}

// Line 231: pdfBatchConvert signature has no formatter params
func pdfBatchConvert(ctx context.Context, p parser.FullParser, inputDir, outputDir string, logger logging.Logger) {
    // Cannot apply formatter
}
```

**Impact**:
- User command: `camt-csv pdf -i pdf_dir/ -o out_dir/ --batch --format icompta`
- Expected: Each PDF → individual CSV in iCompta format + manifest
- Actual: Each PDF → individual CSV in standard format + manifest
- User experience: Formatter flag silently ignored

**Root Cause**: Phase 7's BatchProcessor doesn't have formatter configuration API

---

### GAP 3: PDF Consolidation Ignores Formatter Flags

**Severity**: HIGH  
**Files**: `/Users/fjacquet/Projects/camt-csv/cmd/pdf/convert.go:98-228`

**Problem**:
```go
// Line 98-103: Flags retrieved but not passed
count, err := consolidatePDFDirectory(ctx, p, inputPath,
    root.SharedFlags.Output, root.SharedFlags.Validate, logger)
// ↑ format/dateFormat NOT passed

// Line 218: Uses legacy function without formatter
if err := internalcommon.WriteTransactionsToCSVWithLogger(allTransactions, outputFile, logger); err != nil {
    // ↑ Hardcoded format, ignores --format flag
}
```

**Impact**:
- User command: `camt-csv pdf -i pdf_dir/ -o out.csv --format icompta`
- Expected: iCompta format (10 cols, semicolon, dd.MM.yyyy)
- Actual: Standard format (35 cols, comma)
- User experience: Formatter flag silently ignored

---

## E2E Flows Status Matrix

| Flow | Command | Expected | Actual | Status |
|------|---------|----------|--------|--------|
| Single file | `revolut -i file.csv -o out.csv --format icompta` | iCompta format | iCompta format | ✓ PASS |
| Batch Revolut | `revolut -i csv_dir/ -o out_dir/ --format icompta` | iCompta + manifest | Standard format, NO manifest | ✗ FAIL |
| Batch PDF | `pdf -i pdf_dir/ -o out_dir/ --batch --format icompta` | iCompta + manifest | Standard format + manifest | ✗ FAIL |
| PDF Consolidate | `pdf -i pdf_dir/ -o out.csv --format icompta` | iCompta format | Standard format | ✗ FAIL |
| AI Categorization | `camt -i file.xml --ai-enabled --auto-learn` | Rate limited, logged | Rate limited, logged | ✓ PASS |

---

## Phase Integration Assessment

### Phase 5: Output Formatters
- **Individual Status**: ✓ PASSED (unit tests, 20+ test cases)
- **Integration Status**: ⚠️ PARTIAL
  - Single file flows: ✓ Works correctly
  - Batch flows: ✗ Not wired through
  - Consolidation: ✗ Legacy code path bypasses formatter
- **Issue**: Formatter adoption incomplete in batch/consolidation code paths

### Phase 6: Revolut Overhaul
- **Individual Status**: ✓ PASSED (Product field added, 35 columns)
- **Integration Status**: ✗ INCOMPLETE
  - Single file conversion: ✓ Works
  - Batch conversion: ✗ Uses pre-Phase-7 legacy logic
- **Issue**: Batch logic not refactored after Phase 7 delivered BatchProcessor

### Phase 7: Batch Infrastructure
- **Individual Status**: ✓ PASSED (BatchProcessor, manifest, 17 test cases)
- **Integration Status**: ⚠️ PARTIAL
  - PDF batch: ✓ Uses BatchProcessor correctly
  - Revolut batch: ✗ Not updated to use BatchProcessor
  - Formatter support: ✗ No API for formatter configuration
- **Issue**: PDF integration complete, but Revolut not retrofitted; BatchProcessor lacks formatter config

### Phase 8: AI Safety
- **Individual Status**: ✓ PASSED (confidence metadata, rate limiting, auto-learn)
- **Integration Status**: ✓ COMPLETE
  - CLI flags: ✓ --auto-learn wired through config to categorizer
  - Rate limiting: ✓ 10 RPM default enforced
  - Retry logic: ✓ Exponential backoff implemented
  - Confidence logging: ✓ Pre-save audit logging active
- **Issue**: None; fully wired but only benefits single-file flows due to gaps in 5-7

---

## Data Flow Analysis

### Single File Path (WORKING)
```
CLI --format flag
    ↓
ProcessFile()
    ↓
GetFormatterRegistry().Get(format)
    ↓
WriteTransactionsToCSVWithFormatter()
    ↓
OutputFormatter.Format()
    ↓
CSV File (correct format)
```

### Batch Revolut Path (BROKEN)
```
CLI --format flag (retrieved but not used)
    ↓
batchConvert() (doesn't receive format param)
    ↓
adapter.BatchConvert() → legacy BatchConvert()
    ↓
WriteToCSVWithLogger() (hardcoded format)
    ↓
CSV File (WRONG format - always standard)
    ↓
(No manifest.json generated - violates Phase 7)
```

### Batch PDF Path (BROKEN)
```
CLI --format flag (retrieved but not used)
    ↓
pdfBatchConvert() (doesn't receive format param)
    ↓
BatchProcessor.ProcessDirectory()
    ↓
parser.ConvertToCSV() (no formatter parameter)
    ↓
CSV File (WRONG format - always standard)
    ↓
manifest.json (correct, but format wrong)
```

---

## Architecture Mismatch

**Problem**: Phase 7 designed BatchProcessor without formatter configuration interface.

```go
// Current interface (Phase 7):
type BatchProcessor struct {
    parser parser.FullParser  // Has no formatter config
    logger logging.Logger
}

// Called via:
func (bp *BatchProcessor) ProcessDirectory(ctx context.Context, inputDir, outputDir string) (*BatchManifest, error) {
    // Calls parser.ConvertToCSV which doesn't support formatters
    result := bp.processFile(ctx, filePath, outputDir)
    // ↑ No way to pass formatter through
}
```

**Alternative approaches (for future fix)**:
1. Add formatter to BatchProcessor constructor
2. Create FormattedBatchProcessor variant
3. Make parser.ConvertToCSV formatter-aware (requires breaking change)

---

## Recommendations

### MUST-FIX (Blocking for v1.2 release)

1. **Retrofit Revolut to Use BatchProcessor** (Est. 1-2 hours)
   - Change `revolutparser/adapter.go:BatchConvert()` to use `batch.NewBatchProcessor(a, a.GetLogger())`
   - Remove legacy `BatchConvert()` function from `revolutparser.go`
   - Ensure categorizer is wired (already happens via adapter's SetCategorizer)
   - Write manifest.json at end

2. **Add Formatter Support to PDF Consolidation** (Est. 1 hour)
   - Modify `cmd/pdf/convert.go` to pass `format` and `dateFormat` to `consolidatePDFDirectory()`
   - Replace `WriteTransactionsToCSVWithLogger` call with `WriteTransactionsToCSVWithFormatter`
   - Get formatter from registry

3. **Add Formatter Config to BatchProcessor** (Est. 2-3 hours)
   - Extend BatchProcessor to accept formatter in constructor
   - Modify `ProcessDirectory()` to use formatter in file output
   - Update PDF adapter's BatchConvert to pass formatter
   - Update pdfBatchConvert to pass formatter through

### SHOULD-FIX (Consistency improvements)

4. **Standardize --batch Flag** (Est. 1-2 hours)
   - Add explicit `--batch` flag to CAMT, Revolut, Revolut Investment, Selma, Debit
   - Current state: auto-detects directory without flag (unclear UX)
   - Expected: explicit opt-in with `--batch` (consistent with PDF)

5. **Standardize Batch Naming** (Est. 1 hour)
   - Revolut: `{name}-standardized.csv`
   - PDF: `{name}.csv`
   - Choose one convention

---

## Verification Checklist

After fixes:

- [ ] Build succeeds: `make build`
- [ ] All tests pass: `make test`
- [ ] Test single file: `./camt-csv revolut -i test.csv -o /tmp/out.csv --format icompta`
  - [ ] Output is iCompta format (10 cols, semicolon)
- [ ] Test Revolut batch: `./camt-csv revolut -i csv_dir/ -o out_dir/ --format icompta`
  - [ ] Each output file is iCompta format
  - [ ] `.manifest.json` exists in out_dir/
  - [ ] Exit code correctly reflects success (0) or failure (1/2)
- [ ] Test PDF batch: `./camt-csv pdf -i pdf_dir/ -o out_dir/ --batch --format icompta`
  - [ ] Each output file is iCompta format
  - [ ] `.manifest.json` exists
  - [ ] Exit code 0 (success), 1 (partial), 2 (all failed)
- [ ] Test PDF consolidation: `./camt-csv pdf -i pdf_dir/ -o /tmp/out.csv --format icompta`
  - [ ] Output is iCompta format (10 cols, semicolon)
- [ ] Linter passes: `make lint`

---

## Document References

- Phase 5 SUMMARY: `.planning/phases/05-output-framework/05-01-SUMMARY.md`
- Phase 6 SUMMARY: `.planning/phases/06-revolut-parsers-overhaul/06-01-SUMMARY.md`
- Phase 7 SUMMARY: `.planning/phases/07-batch-infrastructure/07-01-SUMMARY.md`
- Phase 8 SUMMARY: `.planning/phases/08-ai-safety-controls/08-01-SUMMARY.md`

---

## Conclusion

**v1.2 Milestone Phase Achievement**: All 4 phases completed individual objectives successfully.

**v1.2 Milestone Integration Status**: 3 out of 5 critical E2E flows are broken due to incomplete cross-phase wiring, particularly the failure to retrofit Phase 6 (Revolut) to use Phase 7 (Batch Infrastructure) and the lack of formatter configuration support in BatchProcessor.

**Estimated Fix Effort**: 4-6 hours for must-fix items, 2-3 hours for should-fix items.

**Release Readiness**: **NOT READY** - Must fix formatter integration in batch flows before release. Single-file flows are production-ready.
