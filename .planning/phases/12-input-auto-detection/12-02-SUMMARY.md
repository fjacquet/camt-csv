---
phase: 12-input-auto-detection
plan: "02"
subsystem: cmd/revolut,cmd/pdf
tags: [folder-mode, auto-detection, guard, batch-removal, pdf, revolut]
dependency_graph:
  requires:
    - cmd/common.FolderConvert (from 12-01)
    - internal/formatter.FormatterRegistry
    - internal/parser.FullParser
  provides:
    - cmd/revolut: --output guard before batchConvert
    - cmd/pdf: --output guard + consolidation-only folder mode (no --batch)
  affects:
    - cmd/revolut/convert.go
    - cmd/pdf/convert.go
    - CHANGELOG.md
tech_stack:
  added: []
  patterns:
    - --output guard pattern (fileInfo.IsDir() && outputPath == "") consistent across all 6 parsers
    - PDF folder mode consolidates only — no batch branching
key_files:
  created: []
  modified:
    - cmd/revolut/convert.go
    - cmd/pdf/convert.go
    - CHANGELOG.md
decisions:
  - "PDF folder mode always consolidates to one CSV — removed --batch flag and pdfBatchConvert function entirely"
  - "revolut --output guard added before batchConvert, matching cmd/common pattern exactly"
  - "All 6 parser commands (camt, debit, revolut, revolut-investment, selma, pdf) now implement INPUT-01 through INPUT-06"
metrics:
  duration: "5m"
  completed: "2026-02-23"
  tasks_completed: 3
  files_changed: 3
---

# Phase 12 Plan 02: Input Auto-Detection (revolut + pdf guards) Summary

**One-liner:** Add `--output` guards to revolut and pdf commands and remove pdf's `--batch` flag, completing INPUT-01 through INPUT-06 across all 6 parser commands.

## What Was Built

### Task 1: revolut --output guard

`cmd/revolut/convert.go` now includes the `--output` required guard immediately before the `batchConvert` call:

```go
if fileInfo.IsDir() && outputPath == "" {
    logger.Fatalf("--output flag is required when processing a folder. Use -o or --output to specify the output directory.")
}
```

The existing `batchConvert` function (using `BatchProcessor` with formatter support) is unchanged.

### Task 2: pdf --output guard + --batch removal

`cmd/pdf/convert.go` was refactored to:

1. **Remove** the `--batch` flag from `init()` — no longer registered
2. **Remove** `pdfBatchConvert` function (~30 lines) — no longer called or needed
3. **Add** `--output` guard: `if fileInfo.IsDir() && root.SharedFlags.Output == ""` → Fatalf
4. **Simplify** folder-mode branching: folder always calls `consolidatePDFDirectory`, no `--batch` branch
5. **Remove** now-unused imports (`"fjacquet/camt-csv/internal/batch"`, `"fjacquet/camt-csv/internal/formatter"` — were only used in the removed `pdfBatchConvert`)
6. **Update** the `Long` description: two modes only (single file, directory consolidation)

The `consolidatePDFDirectory` function and `sortTransactionsChronologically` helper are unchanged.

### Task 3: CHANGELOG

Added under `## [Unreleased]`:

- **Changed:** All 6 parser commands now accept file or folder transparently; folder requires `--output`; PDF folder always consolidates
- **Removed:** `--batch` flag from `pdf` command; `pdfBatchConvert` function

## Verification

- `go build ./...` — passes (all packages)
- `go test ./...` — 3033 tests pass across 30 packages
- `grep -n "batch" cmd/pdf/convert.go` — no --batch flag, no pdfBatchConvert

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check

Files modified:
- `cmd/revolut/convert.go` — contains `--output` guard before batchConvert
- `cmd/pdf/convert.go` — --batch removed, pdfBatchConvert removed, guard added, consolidation-only folder mode
- `CHANGELOG.md` — Unreleased section updated with Input Auto-Detection entries

Commits:
- `72fa569` — feat(12-02): add --output guard to revolut command
- `c5bbdec` — feat(12-02): remove --batch flag from pdf command and add --output guard
- `94208e9` — chore(12-02): update CHANGELOG with Input Auto-Detection changes

## Self-Check: PASSED

All files verified present and all commits confirmed in git log.
