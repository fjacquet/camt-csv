---
phase: 12-input-auto-detection
verified: 2026-02-23T00:00:00Z
status: passed
score: 9/9 must-haves verified
re_verification: false
---

# Phase 12: Input Auto-Detection Verification Report

**Phase Goal:** All parser commands accept file or folder input transparently — users never need a separate batch command
**Verified:** 2026-02-23
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                          | Status     | Evidence                                                                                                  |
|----|------------------------------------------------------------------------------------------------|------------|-----------------------------------------------------------------------------------------------------------|
| 1  | User can run `camt-csv camt path/to/file.xml --output out.csv` and it processes the single file | ✓ VERIFIED | `RunConvert` in `cmd/common/convert.go` routes `!IsDir()` to `ProcessFile(...)` unchanged                |
| 2  | User can run `camt-csv revolut path/to/folder/ --output ./out/` and it processes every matching file to individual CSVs | ✓ VERIFIED | `cmd/revolut/convert.go` guard passes, routes to `batchConvert` using `BatchProcessor` + formatter       |
| 3  | User can run `camt-csv pdf path/to/folder/ --output ./out/` and it consolidates all PDFs into one CSV | ✓ VERIFIED | `cmd/pdf/convert.go` folder path calls `consolidatePDFDirectory`, no `--batch` branch present            |
| 4  | When a folder is passed without `--output`, the command exits with a clear error message        | ✓ VERIFIED | All 6 parsers contain guard: `fileInfo.IsDir() && outputPath == ""` → `logger.Fatal(...)` with `--output` and "folder" in message |
| 5  | All 6 parsers (camt, debit, revolut, revolut-investment, selma, pdf) accept both file and folder inputs | ✓ VERIFIED | camt/debit/selma/revolut-investment delegate to `common.RunConvert`; revolut and pdf have own guards     |
| 6  | `FolderConvert` uses BatchProcessor with formatter (not legacy BatchConvert interface)          | ✓ VERIFIED | `cmd/common/convert.go:97` — `batch.NewBatchProcessor(fullParser, logger, outFormatter)`                 |
| 7  | PDF command has no `--batch` flag and no `pdfBatchConvert` function                            | ✓ VERIFIED | `grep -n "batch" cmd/pdf/convert.go` returns 0 matches; `pdfBatchConvert` removed                       |
| 8  | Tests for folder routing exist and pass                                                        | ✓ VERIFIED | `cmd/common/convert_test.go` — 3 tests; `go test ./cmd/common/...` — 15 tests pass                      |
| 9  | CHANGELOG.md updated under [Unreleased] with Input Auto-Detection entries                      | ✓ VERIFIED | `CHANGELOG.md` lines 20-28 describe all 6 parsers, --output requirement, and PDF consolidation-only mode |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact                            | Expected                                                              | Status     | Details                                                                                           |
|-------------------------------------|-----------------------------------------------------------------------|------------|---------------------------------------------------------------------------------------------------|
| `cmd/common/convert.go`             | RunConvert with --output guard + FolderConvert using BatchProcessor   | ✓ VERIFIED | Exports `RunConvert`, `FolderConvert`, `BatchConvertLegacy`; guard at line 58-60; FolderConvert at line 80 |
| `cmd/common/convert_test.go`        | Unit tests for folder detection, output guard, and FolderConvert routing | ✓ VERIFIED | Contains `TestRunConvert_FolderWithoutOutput`, `TestFolderConvert_EmptyDirectory`, `TestFolderConvert_InvalidFormat` |
| `cmd/common/export_test.go`         | SetOsExitFn helper for testable os.Exit injection                     | ✓ VERIFIED | Exports `SetOsExitFn` (test-only file), used by all 3 tests                                       |
| `cmd/revolut/convert.go`            | --output guard before batchConvert call                               | ✓ VERIFIED | Guard at line 60-62: `fileInfo.IsDir() && outputPath == ""` → Fatalf                             |
| `cmd/pdf/convert.go`                | --output guard + consolidation-only folder mode (no --batch)          | ✓ VERIFIED | Guard at line 84-86; folder always calls `consolidatePDFDirectory`; no `--batch` flag or `pdfBatchConvert` |
| `CHANGELOG.md`                      | Unreleased section with Input Auto-Detection changes                  | ✓ VERIFIED | Lines 20-28 contain "Input Auto-Detection", all 6 parsers listed, --batch removal documented      |

### Key Link Verification

| From                        | To                              | Via                                | Status     | Details                                                                         |
|-----------------------------|---------------------------------|------------------------------------|------------|---------------------------------------------------------------------------------|
| `cmd/common/convert.go`     | `internal/batch.BatchProcessor` | `batch.NewBatchProcessor(...)`     | ✓ WIRED    | Line 97: `processor := batch.NewBatchProcessor(fullParser, logger, outFormatter)` |
| `cmd/common/convert.go`     | `internal/formatter.FormatterRegistry` | `registry.Get(format)`       | ✓ WIRED    | Lines 82-87: `formatterReg.Get(format)` with error guard                        |
| `cmd/revolut/convert.go`    | `cmd/common/convert.go`         | Same --output guard pattern        | ✓ WIRED    | Guard pattern `outputPath == ""` at line 60; same error message wording          |
| `cmd/pdf/convert.go`        | `consolidatePDFDirectory`       | folder mode always calls it        | ✓ WIRED    | Lines 88-95: only folder branch calls `consolidatePDFDirectory`; no `--batch` gate |
| `cmd/camt/convert.go`       | `cmd/common.RunConvert`         | delegation                         | ✓ WIRED    | Line 17: `common.RunConvert(cmd, args, container.CAMT, "CAMT.053")`             |
| `cmd/debit/convert.go`      | `cmd/common.RunConvert`         | delegation                         | ✓ WIRED    | Line 17: `common.RunConvert(cmd, args, container.Debit, "Debit")`               |
| `cmd/selma/convert.go`      | `cmd/common.RunConvert`         | delegation                         | ✓ WIRED    | Line 17: `common.RunConvert(cmd, args, container.Selma, "Selma")`               |
| `cmd/revolut-investment/convert.go` | `cmd/common.RunConvert`  | delegation                         | ✓ WIRED    | Line 17: `common.RunConvert(cmd, args, container.RevolutInvestment, "Revolut Investment")` |

### Requirements Coverage

| Requirement | Source Plan | Description                                                                                          | Status       | Evidence                                                                                  |
|-------------|-------------|------------------------------------------------------------------------------------------------------|--------------|-------------------------------------------------------------------------------------------|
| INPUT-01    | 12-01, 12-02 | User can pass a file path or a folder path to any parser command                                    | ✓ SATISFIED  | All 6 parsers handle both; `RunConvert` stat-checks input; revolut/pdf have own guards    |
| INPUT-02    | 12-01        | When input is a file, the command processes that single file (unchanged single-file behavior)        | ✓ SATISFIED  | `RunConvert` routes `!IsDir()` → `ProcessFile(...)` unchanged; PDF routes to `common.ProcessFile` |
| INPUT-03    | 12-01        | When input is a folder, the command processes all matching files (non-recursive, extension filtered) | ✓ SATISFIED  | `FolderConvert` → `BatchProcessor.ProcessDirectory` filters by extension; PDF reads `.pdf` files only |
| INPUT-04    | 12-01, 12-02 | When input is a folder, `--output` flag is required; command exits with clear error if omitted      | ✓ SATISFIED  | All 6 parsers have guard with `"--output flag is required when processing a folder"` message |
| INPUT-05    | 12-01, 12-02 | camt, debit, revolut, revolut-investment, selma: folder mode outputs one CSV per input file         | ✓ SATISFIED  | `FolderConvert` and `batchConvert` both use `BatchProcessor` which outputs per-file CSVs  |
| INPUT-06    | 12-02        | pdf: folder mode consolidates all PDFs in the folder into a single CSV                              | ✓ SATISFIED  | `consolidatePDFDirectory` collects all transactions, sorts chronologically, writes one CSV |

No orphaned requirements — all 6 INPUT-0x IDs declared in PLAN frontmatter are accounted for and verified.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | None found | — | No anti-patterns detected in changed files |

No TODO/FIXME/placeholder comments, no stub implementations, no empty return values found in `cmd/common/convert.go`, `cmd/revolut/convert.go`, or `cmd/pdf/convert.go`.

### Human Verification Required

None — all success criteria are programmatically verifiable through code inspection and test execution.

### Build and Test Results

- `go build ./...` — passes (0 errors)
- `go test ./...` — 3033 tests pass across 30 packages
- `go test ./cmd/common/...` — 15 tests pass (includes 3 new folder-routing tests)

### Summary

Phase 12 fully achieves its goal. All 6 parser commands (camt, debit, revolut, revolut-investment, selma, pdf) now transparently accept either a file path or a folder path:

- **File path**: processed as a single file (unchanged behavior)
- **Folder path with `--output`**: routed to modern `BatchProcessor` (camt/debit/selma/revolut-investment/revolut) or `consolidatePDFDirectory` (pdf)
- **Folder path without `--output`**: immediate fatal error with a clear message specifying the `--output` flag is required

The implementation is clean, well-tested, and has no stub or placeholder code. The `--batch` flag has been cleanly removed from the PDF command. The CHANGELOG is fully updated. All 6 INPUT requirements are marked Complete in REQUIREMENTS.md and confirmed in the codebase.

---

_Verified: 2026-02-23_
_Verifier: Claude (gsd-verifier)_
