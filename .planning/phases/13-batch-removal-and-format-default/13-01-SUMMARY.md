---
phase: 13-batch-removal-and-format-default
plan: "01"
subsystem: cli
tags: [go, cobra, batch, cleanup, dead-code]

# Dependency graph
requires:
  - phase: 12-input-auto-detection
    provides: FolderConvert replaces BatchConvertLegacy; every parser has folder-mode path
provides:
  - cmd/batch/ package deleted
  - batch subcommand removed from main.go
  - BatchConvertLegacy removed from cmd/common/convert.go
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - main.go
    - cmd/common/convert.go

key-decisions:
  - "batch subcommand removed with no deprecation period — Cobra's unknown command error is the desired UX"
  - "BatchConvertLegacy deleted entirely; FolderConvert is the sole folder-mode path"
  - "encoding/json import removed from cmd/common/convert.go since it was only used by BatchConvertLegacy"

patterns-established: []

requirements-completed: [BATCH-01, BATCH-02]

# Metrics
duration: 6min
completed: 2026-02-23
---

# Phase 13 Plan 01: Batch Removal Summary

**cmd/batch/ package and BatchConvertLegacy function deleted — `camt-csv batch` now returns Cobra's "unknown command" error**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-02-23T06:58:53Z
- **Completed:** 2026-02-23T07:05:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Deleted entire `cmd/batch/` directory (batch.go and batch_test.go, ~300 lines of dead code)
- Removed `batch` import and `root.Cmd.AddCommand(batch.Cmd)` from `main.go`
- Removed `BatchConvertLegacy` function (~50 lines) and its `encoding/json` import from `cmd/common/convert.go`
- All 3021 existing tests continue to pass after removals

## Task Commits

Each task was committed atomically:

1. **Task 1: Delete cmd/batch package and deregister from main.go** - `a91f67f` (feat)
2. **Task 2: Remove BatchConvertLegacy from cmd/common/convert.go** - `a235294` (feat)

## Files Created/Modified

- `main.go` - Removed batch import and AddCommand call
- `cmd/common/convert.go` - Removed BatchConvertLegacy function and encoding/json import

## Decisions Made

- No deprecation message added — Cobra's built-in "unknown command 'batch'" is the correct UX per BATCH-01
- `FolderConvert` retained as the sole folder-mode path; only the legacy function was removed

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- BATCH-01 and BATCH-02 requirements satisfied
- Phase 13 Plan 02 (format default change) is unblocked and ready to execute

---
*Phase: 13-batch-removal-and-format-default*
*Completed: 2026-02-23*
