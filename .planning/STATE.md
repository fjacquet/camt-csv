# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-15)

**Core value:** Reliable, maintainable financial data conversion with intelligent categorization.

**Current focus:** Phase 6 — Revolut Parsers Overhaul

## Current Position

Phase: 6 of 8 (Revolut Parsers Overhaul)
Plan: 2 of 3 in current phase
Status: In Progress
Last activity: 2026-02-16 — Completed 06-02-PLAN.md (Investment parser enhancement)

Progress: [████████░░░░░░░░░░░░] 33% (v1.2 phases 5-8: 4/12 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 14 (v1.1: 11, v1.2: 3)
- Average duration: 202 sec (v1.2)
- Total execution time: ~1 day (v1.1) + 10 min (v1.2)

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
| Phase 06 P02 | 281 | 2 tasks | 3 files |

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
- [Phase 06-02]: SELL transactions are credit (incoming money from sales)
- [Phase 06-02]: CUSTODY_FEE transactions are debit (outgoing fees) with fee tracking
- [Phase 06-02]: BatchConvert validates each file before conversion, continues on errors

### Pending Todos

None yet for v1.2.

### Blockers/Concerns

None at roadmap stage.

## Session Continuity

Last session: 2026-02-16
Stopped at: Completed 06-02-PLAN.md (Investment parser enhancement)
Resume file: .planning/phases/06-revolut-parsers-overhaul/06-02-SUMMARY.md

---
*State initialized: 2026-02-01 (v1.1)*
*Last updated: 2026-02-16 (v1.2 phase 6 in progress: 2/3 plans)*
