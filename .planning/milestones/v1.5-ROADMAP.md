# Roadmap: camt-csv

## Overview

This roadmap tracks the evolution of camt-csv from a functional MVP through codebase hardening (v1.1), full feature polish (v1.2), standard CSV format optimization (v1.3), operational simplification (v1.4), and Jumpsoft Money export support (v1.5).

## Milestones

- Completed **v1.1 Hardening** - Phases 1-4 (shipped 2026-02-01)
- Completed **v1.2 Full Polish** - Phases 5-9 (shipped 2026-02-16)
- Completed **v1.3 Standard CSV Trim** - Phases 10-11 (shipped 2026-02-16)
- Completed **v1.4 Simplify** - Phases 12-13 (shipped 2026-02-23)
- Active **v1.5 Jumpsoft Money Export** - Phases 14-15 (in progress)

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>Completed v1.1 Hardening (Phases 1-4) - SHIPPED 2026-02-01</summary>

- [x] **Phase 1: Critical Bugs & Security** (3/3 plans) - completed 2026-02-01
- [x] **Phase 2: Configuration & State Cleanup** (1/1 plans) - completed 2026-02-01
- [x] **Phase 3: Architecture & Error Handling** (3/3 plans) - completed 2026-02-01
- [x] **Phase 4: Test Coverage & Safety** (4/4 plans) - completed 2026-02-01

Full details: `.planning/milestones/v1.1-ROADMAP.md`

</details>

<details>
<summary>Completed v1.2 Full Polish (Phases 5-9) - SHIPPED 2026-02-16</summary>

- [x] **Phase 5: Output Framework** (3/3 plans) - completed 2026-02-16
- [x] **Phase 6: Revolut Parsers Overhaul** (3/3 plans) - completed 2026-02-16
- [x] **Phase 7: Batch Infrastructure** (2/2 plans) - completed 2026-02-16
- [x] **Phase 8: AI Safety Controls** (3/3 plans) - completed 2026-02-16
- [x] **Phase 9: Batch-Formatter Integration** (3/3 plans) - completed 2026-02-16

Full details: `.planning/milestones/v1.2-ROADMAP.md`

</details>

<details>
<summary>Completed v1.3 Standard CSV Trim (Phases 10-11) - SHIPPED 2026-02-16</summary>

- [x] **Phase 10: CSV Format Trim** (1/1 plans) - completed 2026-02-16
- [x] **Phase 11: Integration Verification** (2/2 plans) - completed 2026-02-16

Full details: `.planning/milestones/v1.3-ROADMAP.md`

</details>

<details>
<summary>Completed v1.4 Simplify (Phases 12-13) - SHIPPED 2026-02-23</summary>

- [x] **Phase 12: Input Auto-Detection** - All 6 parser commands detect file vs. folder automatically (completed 2026-02-23)
- [x] **Phase 13: Batch Removal and Format Default** - Drop batch command/flag, make icompta the default format (completed 2026-02-23)

Full details: `.planning/milestones/v1.4-ROADMAP.md`

</details>

### Active v1.5 Jumpsoft Money Export (In Progress)

**Milestone Goal:** Add `--format jumpsoft` output option across all parsers, producing clean comma-delimited CSV for Jumpsoft Money import.

- [x] **Phase 14: JumpsoftFormatter** - Build and register the formatter with full CLI integration (completed 2026-03-02)
- [x] **Phase 15: Verification** - Unit and integration tests confirming correct output (completed 2026-03-02)

## Phase Details

### Phase 14: JumpsoftFormatter
**Goal**: Users can produce Jumpsoft Money-compatible CSV output from any parser using `--format jumpsoft`
**Depends on**: Nothing (first phase of v1.5; existing FormatterRegistry and --format infrastructure already in place)
**Requirements**: FMT-01, FMT-02, FMT-03, FMT-04, FMT-05, INT-01, INT-02, INT-03, INT-04
**Success Criteria** (what must be TRUE):
  1. Running any parser command with `--format jumpsoft` produces a comma-delimited CSV file with header row containing columns: Date, Description, Amount, Currency, Category, Type, Notes
  2. Date column contains YYYY-MM-DD formatted dates, overridable with the existing --date-format flag
  3. Amount column is signed: negative value for debit transactions, positive value for credit transactions
  4. Category column is populated from categorizer output when the categorizer runs
  5. `--help` on any parser command lists `jumpsoft` as a valid --format choice
**Plans**: 1 plan

Plans:
- [ ] 14-01-PLAN.md — Implement JumpsoftFormatter struct, register in FormatterRegistry, update CLI help text, and verify full test suite

### Phase 15: Verification
**Goal**: JumpsoftFormatter behavior is validated by automated tests covering field mapping and end-to-end output
**Depends on**: Phase 14
**Requirements**: TEST-01, TEST-02
**Success Criteria** (what must be TRUE):
  1. `go test ./internal/formatter/...` passes with tests asserting each Jumpsoft CSV column maps to the correct Transaction field (Date, Description, Amount sign, Currency, Category, Type, Notes)
  2. `go test ./...` passes with at least one integration test that exercises a complete parse-then-format pipeline through one parser and asserts a valid Jumpsoft CSV file is produced
  3. `go test -race ./...` passes with no data races introduced by the new formatter
**Plans**: 1 plan

Plans:
- [ ] 15-01-PLAN.md — Unit tests for JumpsoftFormatter (12 subtests covering all 7 columns + edge cases) and integration test via CAMT parse→format pipeline

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-4 Hardening | v1.1 | 11/11 | Complete | 2026-02-01 |
| 5-9 Full Polish | v1.2 | 14/14 | Complete | 2026-02-16 |
| 10-11 CSV Trim | v1.3 | 3/3 | Complete | 2026-02-16 |
| 12-13 Simplify | v1.4 | 4/4 | Complete | 2026-02-23 |
| 14. JumpsoftFormatter | v1.5 | 1/1 | Complete | 2026-03-02 |
| 15. Verification | 1/1 | Complete    | 2026-03-02 | - |

---
*Roadmap created: 2026-02-01*
*Last updated: 2026-03-02 — phase 15 planned (1 plan)*
