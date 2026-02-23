---
phase: 12-input-auto-detection
plan: "01"
subsystem: cmd/common
tags: [batch, folder-mode, routing, formatter, testing]
dependency_graph:
  requires:
    - internal/batch.BatchProcessor
    - internal/formatter.FormatterRegistry
    - internal/parser.FullParser
  provides:
    - cmd/common.FolderConvert
    - cmd/common.RunConvert (updated: --output guard + FolderConvert routing)
  affects:
    - cmd/camt
    - cmd/debit
    - cmd/selma
    - cmd/revolut-investment
tech_stack:
  added: []
  patterns:
    - osExitFn package variable for testable os.Exit injection
    - FolderConvert with injectable logger and parser for unit testing
key_files:
  created:
    - cmd/common/convert_test.go
    - cmd/common/export_test.go
  modified:
    - cmd/common/convert.go
decisions:
  - "Added osExitFn package-level variable to FolderConvert to enable unit testing without os.Exit killing the test process"
  - "EmptyDirectory test asserts exit code 2 (per BatchManifest contract) rather than 0 — consistent with existing manifest_test.go expectations"
  - "BatchConvertLegacy retained unchanged; RunConvert now calls FolderConvert instead"
metrics:
  duration: "6m"
  completed: "2026-02-23"
  tasks_completed: 2
  files_changed: 4
---

# Phase 12 Plan 01: Input Auto-Detection (RunConvert Guard + FolderConvert) Summary

**One-liner:** Add `--output` guard and `FolderConvert` to `cmd/common/convert.go` so all shared-parser commands route folder input through `BatchProcessor` with formatter support.

## What Was Built

`RunConvert` in `cmd/common/convert.go` now implements the input auto-detection contract:

- **File input** → `ProcessFile(...)` (unchanged single-file behavior)
- **Folder input + no `--output`** → `logger.Fatal("--output flag is required...")` with a clear human-readable message
- **Folder input + `--output`** → `FolderConvert(...)` using `BatchProcessor` + `FormatterRegistry`

### New: `FolderConvert`

`FolderConvert(ctx, p, inputDir, outputDir, logger, format, dateFormat)` is the modern batch path that:

1. Resolves the output formatter via `formatter.NewFormatterRegistry().Get(format)` — fatals on invalid format
2. Asserts the parser to `parser.FullParser` — fatals if not supported
3. Creates `batch.NewBatchProcessor(fullParser, logger, outFormatter)`
4. Calls `processor.ProcessDirectory(ctx, inputDir, outputDir)`
5. Writes (refreshes) the manifest at `outputDir/.manifest.json`
6. Logs `"Batch complete: N/M files succeeded"`
7. Calls `osExitFn(manifest.ExitCode())` only if exit code != 0

### `BatchConvertLegacy` Retained

The legacy function is kept intact (not removed, not changed) for Phase 13 cleanup. Its usage in `RunConvert` is replaced by `FolderConvert`.

### New: `osExitFn` Variable

A package-level `var osExitFn = os.Exit` enables tests to replace `os.Exit` with a no-op capture function. Exposed via `export_test.go` → `SetOsExitFn`.

## Tests Added

File: `cmd/common/convert_test.go` (package `common_test`)

| Test | What it verifies |
|------|-----------------|
| `TestRunConvert_FolderWithoutOutput` | Non-FullParser triggers FATAL "batch conversion" in FolderConvert |
| `TestFolderConvert_EmptyDirectory` | 0-file dir: no FATAL, exit code 2 (BatchManifest contract), manifest written |
| `TestFolderConvert_InvalidFormat` | Format "invalid" triggers FATAL mentioning "invalid" or "format" |

All 15 tests in `cmd/common` pass (3 new + 12 existing).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Testability] Added osExitFn injection point for os.Exit**

- **Found during:** Task 2 (TestFolderConvert_EmptyDirectory)
- **Issue:** `FolderConvert` calls `os.Exit(manifest.ExitCode())` when exit code != 0. An empty directory results in `ExitCode() == 2` (per existing `BatchManifest` contract in `manifest_test.go`), which would kill the test process.
- **Fix:** Added `var osExitFn = os.Exit` and `SetOsExitFn` helper in `export_test.go`. Tests capture the exit code without exiting.
- **Files modified:** `cmd/common/convert.go`, `cmd/common/export_test.go`
- **Commits:** 301d95f

**2. [Clarification] TestFolderConvert_EmptyDirectory asserts exit code 2, not 0**

- **Found during:** Task 2
- **Issue:** Plan said "0 files = valid scenario per BatchProcessor". The BatchProcessor itself does not error, but `BatchManifest.ExitCode()` returns 2 for 0 files (tested and asserted in `internal/batch/manifest_test.go`). Changing `ExitCode()` would break existing tests.
- **Fix:** Test now asserts `capturedExitCode == 2` (consistent with manifest contract) rather than expecting clean return.
- **Files modified:** `cmd/common/convert_test.go`

## Self-Check

Files created/modified:
- `cmd/common/convert.go` — contains `FolderConvert`, `osExitFn`, updated `RunConvert`
- `cmd/common/convert_test.go` — 3 new tests
- `cmd/common/export_test.go` — `SetOsExitFn` helper

Commits:
- `8e1e26c` — Task 1: FolderConvert + --output guard
- `301d95f` — Task 2: tests + export_test.go + CHANGELOG

## Self-Check: PASSED

All files verified present and all commits confirmed in git log.
