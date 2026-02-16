---
phase: 06-revolut-parsers-overhaul
plan: 02
subsystem: parsers
tags: [revolut, investment, csv, batch-processing]

# Dependency graph
requires:
  - phase: 05-output-framework
    provides: Transaction builder and formatter infrastructure
provides:
  - Complete investment transaction type coverage (BUY, SELL, DIVIDEND, CUSTODY_FEE, CASH TOP-UP)
  - Batch conversion support for investment CSV files
affects: [06-03, batch-operations, investment-reporting]

# Tech tracking
tech-stack:
  added: []
  patterns: [batch-converter-interface, transaction-type-handling]

key-files:
  created: []
  modified:
    - internal/revolutinvestmentparser/revolutinvestmentparser.go
    - internal/revolutinvestmentparser/revolutinvestmentparser_test.go
    - internal/revolutinvestmentparser/adapter.go

key-decisions:
  - "SELL transactions are credit (incoming money from sales)"
  - "CUSTODY_FEE transactions are debit (outgoing fees) with fee tracking"
  - "BatchConvert validates each file before conversion, continues on errors"

patterns-established:
  - "Transaction type handling: SELL (credit, shares, description), CUSTODY_FEE (debit, fees)"
  - "Batch processing: validate→convert→continue on error pattern"

# Metrics
duration: 281s
completed: 2026-02-16
---

# Phase 06 Plan 02: Investment Parser Enhancement Summary

**Revolut Investment parser now handles SELL/CUSTODY_FEE transactions and supports batch directory conversion with graceful error handling**

## Performance

- **Duration:** 4 min 41 sec
- **Started:** 2026-02-16T05:38:41Z
- **Completed:** 2026-02-16T05:43:22Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Added SELL transaction type handling (credit direction, share count, proceeds tracking)
- Added CUSTODY_FEE transaction type handling (debit direction, fee amount tracking)
- Implemented full BatchConvert() with validation, error recovery, and logging
- Added comprehensive test coverage for new transaction types

## Task Commits

Each task was committed atomically:

1. **Task 1: Add SELL and CUSTODY_FEE handling** - `25961cd` (feat)
2. **Task 2: Implement BatchConvert** - `20dd883` (feat)

## Files Created/Modified
- `internal/revolutinvestmentparser/revolutinvestmentparser.go` - Added SELL and CUSTODY_FEE cases in convertRowToTransaction()
- `internal/revolutinvestmentparser/revolutinvestmentparser_test.go` - Added tests for SELL and CUSTODY_FEE types, updated default type test
- `internal/revolutinvestmentparser/adapter.go` - Implemented full BatchConvert() with validation and error handling

## Decisions Made
- **SELL transaction direction:** Credit (money coming in from selling shares) - consistent with financial transaction flow
- **CUSTODY_FEE fee tracking:** Use WithFees() method to record fee amount separately from transaction amount
- **BatchConvert error handling:** Continue processing remaining files when individual file fails - resilient batch operation
- **Builder method selection:** Use WithPayer() for credit (SELL), WithPayee() for debit (CUSTODY_FEE) - matches existing patterns

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed test expectations for SELL transaction**
- **Found during:** Task 1 (initial build after adding SELL case)
- **Issue:** TestConvertRowToTransaction_DefaultType tested SELL as default case, but SELL now has explicit handling
- **Fix:** Changed test to use "OTHER" transaction type for default case testing, maintaining test validity
- **Files modified:** internal/revolutinvestmentparser/revolutinvestmentparser_test.go
- **Verification:** All tests pass with proper SELL behavior
- **Committed in:** 25961cd (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Test correction necessary for proper validation of new SELL implementation. No scope change.

## Issues Encountered
- **Initial builder method error:** First used WithRecipient() for SELL/CUSTODY_FEE, which doesn't exist. Corrected to WithPayer()/WithPayee() following existing transaction patterns in the file.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Investment parser feature-complete for known Revolut transaction types
- Batch conversion capability ready for multi-file processing
- Ready for Phase 06-03 (integration testing and CLI enhancements)
- All existing tests pass, build succeeds

## Self-Check

Verifying implementation claims:

**Files exist:**
- ✓ internal/revolutinvestmentparser/revolutinvestmentparser.go modified (SELL and CUSTODY_FEE cases present)
- ✓ internal/revolutinvestmentparser/adapter.go modified (BatchConvert implemented)

**Commits exist:**
- ✓ 25961cd: feat(06-02): add SELL and CUSTODY_FEE transaction handling
- ✓ 20dd883: feat(06-02): implement BatchConvert for investment parser

**Functionality verified:**
- ✓ SELL case exists at line 233
- ✓ CUSTODY_FEE case exists at line 258
- ✓ BatchConvert() signature correct at line 76
- ✓ All investment parser tests pass (28 tests)
- ✓ Build succeeds

## Self-Check: PASSED

---
*Phase: 06-revolut-parsers-overhaul*
*Completed: 2026-02-16*
