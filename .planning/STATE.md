# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-15)

**Core value:** Reliable, maintainable financial data conversion with intelligent categorization.

**Current focus:** Phase 7 — Batch Infrastructure

## Current Position

Phase: 8 of 8 (AI Safety Controls)
Plan: 3 of 3 in current phase
Status: Complete
Last activity: 2026-02-16 — Completed 08-03-PLAN.md (Auto-learn flag control)

Progress: [█████████████████░░░] 75% (v1.2 phases 5-8: 9/12 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 18 (v1.1: 11, v1.2: 7)
- Average duration: 424 sec (v1.2)
- Total execution time: ~1 day (v1.1) + 51 min (v1.2)

**By Phase (v1.1 completed):**

| Phase | Plans | Status |
|-------|-------|--------|
| 1. Critical Bugs & Security | 3/3 | Complete |
| 2. Configuration & State Cleanup | 1/1 | Complete |
| 3. Architecture & Error Handling | 3/3 | Complete |
| 4. Test Coverage & Safety | 4/4 | Complete |

**v1.2 Phases:**
Starting fresh with Phase 5.
| Phase 05 P01 | 5 | 2 tasks | 4 files |
| Phase 05 P02 | 113 | 2 tasks | 2 files |
| Phase 05 P03 | 489 | 2 tasks | 8 files |
| Phase 06 P01 | 496 | 2 tasks | 7 files |
| Phase 06 P03 | 192 | 2 tasks | 1 files |
| Phase 07 P01 | 472 | 2 tasks | 4 files |
| Phase 07 P02 | 854 | 2 tasks | 8 files |
| Phase 08 P02 | 1133 | 3 tasks | 4 files |
| Phase 08 P01 | 1135 | 3 tasks | 7 files |
| Phase 08 P03 | 480 | 3 tasks | 10 files |

## Accumulated Context

### Decisions

Full decision log in PROJECT.md Key Decisions table.
Recent decisions affecting v1.2:

- v1.2: Output standardization uses formatter pattern (not parser rewrite)
- v1.2: iCompta format is separate mode, not replacement of standard CSV
- v1.2: Revolut parser upgrade to 35-column format for consistency
- v1.2: Exchange transactions preserved with both currencies visible
- v1.2: AI auto-learn defaults to OFF (user must enable)
- v1.2: Revolut Savings product maps to separate iCompta account ("Revolut CHF Vacances")
- v1.2: Account routing by Product+Currency: Current/CHF→Revolut CHF, Savings/CHF→Revolut CHF Vacances, Current/EUR→Revolut EUR
- v1.2: iCompta import plugins already exist in user's DB (CSV-Revolut-CHF, CSV-Revolut-EUR, CSV-RevInvest) — match their column names and use semicolon separator + dd.MM.yyyy dates
- [Phase 05]: Strategy pattern chosen for formatters (not inheritance or monolithic switch)
- [Phase 05]: Status mapping for iCompta: BOOK/RCVD→cleared, PDNG→pending, REVD/CANC→reverted
- [Phase 05]: ProcessFile() refactored to use Parse() + WriteTransactionsToCSVWithFormatter (not ConvertToCSV)
- [Phase 06-01]: Product field positioned after Currency for logical grouping of account-level fields
- [Phase 06-01]: CSV format expanded from 34 to 35 columns to include Product field
- [Phase 06-01]: No validation on Product field values - accepts any string from source data
- [Phase 06-03]: Exchange transactions preserve metadata in OriginalAmount/OriginalCurrency for future FX handling
- [Phase 06-03]: REVERTED and PENDING transactions are logged when skipped for user visibility
- [Phase 07-01]: Sequential processing chosen over parallel for Phase 7 (simplicity, error isolation)
- [Phase 07-01]: Manifest always written to {outputDir}/.manifest.json before returning
- [Phase 07-01]: Exit code semantics standardized: 0=all success, 1=partial success, 2=all failed or no files
- [Phase 07-02]: PDF parser migrated from stub to BatchProcessor composition pattern
- [Phase 07-02]: PDF command supports both batch mode (--batch flag) and consolidation mode (default)
- [Phase 07-02]: All CLI commands detect directory input and invoke BatchConvert automatically
- [Phase 07-02]: Manifest loading happens in CLI layer for exit code determination
- [Phase 07-02]: Exit code fallback strategy if manifest unreadable: exit based on success count
- [Phase 08-02]: Strict rate limiting with burst=1 (no bursting allowed) for API quota protection
- [Phase 08-02]: Exponential backoff retry (3 attempts, 2x multiplier, ±20% jitter) for transient failures
- [Phase 08-02]: Rate limit check before API call (fail fast on exceeded limit)
- [Phase 08-01]: Heuristic confidence estimation for AI categorizations (0.9 for known categories, 0.8 for unknown)
- [Phase 08-01]: INFO-level logging for audit trail in production (not DEBUG)
- [Phase 08]: Auto-learn defaults to OFF per v1.2 decision D-11 for safety

### Pending Todos

None yet for v1.2.

### Blockers/Concerns

None at roadmap stage.

## Session Continuity

Last session: 2026-02-16
Stopped at: Completed 08-03-PLAN.md (Auto-learn flag control)
Resume file: .planning/phases/08-ai-safety-controls/08-03-SUMMARY.md

---
*State initialized: 2026-02-01 (v1.1)*
*Last updated: 2026-02-16 (v1.2 phase 8 complete: 3/3 plans)*
