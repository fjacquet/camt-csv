# Roadmap: camt-csv

## Overview

This roadmap tracks the evolution of camt-csv from a functional MVP through codebase hardening (v1.1), full feature polish (v1.2), standard CSV format optimization (v1.3), and operational simplification (v1.4).

## Milestones

- ✅ **v1.1 Hardening** — Phases 1-4 (shipped 2026-02-01)
- ✅ **v1.2 Full Polish** — Phases 5-9 (shipped 2026-02-16)
- ✅ **v1.3 Standard CSV Trim** — Phases 10-11 (shipped 2026-02-16)
- **v1.4 Simplify** — Phases 12-13 (in progress)

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>✅ v1.1 Hardening (Phases 1-4) — SHIPPED 2026-02-01</summary>

- [x] **Phase 1: Critical Bugs & Security** (3/3 plans) — completed 2026-02-01
- [x] **Phase 2: Configuration & State Cleanup** (1/1 plans) — completed 2026-02-01
- [x] **Phase 3: Architecture & Error Handling** (3/3 plans) — completed 2026-02-01
- [x] **Phase 4: Test Coverage & Safety** (4/4 plans) — completed 2026-02-01

Full details: `.planning/milestones/v1.1-ROADMAP.md`

</details>

<details>
<summary>✅ v1.2 Full Polish (Phases 5-9) — SHIPPED 2026-02-16</summary>

- [x] **Phase 5: Output Framework** (3/3 plans) — completed 2026-02-16
- [x] **Phase 6: Revolut Parsers Overhaul** (3/3 plans) — completed 2026-02-16
- [x] **Phase 7: Batch Infrastructure** (2/2 plans) — completed 2026-02-16
- [x] **Phase 8: AI Safety Controls** (3/3 plans) — completed 2026-02-16
- [x] **Phase 9: Batch-Formatter Integration** (3/3 plans) — completed 2026-02-16

Full details: `.planning/milestones/v1.2-ROADMAP.md`

</details>

<details>
<summary>✅ v1.3 Standard CSV Trim (Phases 10-11) — SHIPPED 2026-02-16</summary>

- [x] **Phase 10: CSV Format Trim** (1/1 plans) — completed 2026-02-16
- [x] **Phase 11: Integration Verification** (2/2 plans) — completed 2026-02-16

Full details: `.planning/milestones/v1.3-ROADMAP.md`

</details>

### v1.4 Simplify (Phases 12-13)

- [x] **Phase 12: Input Auto-Detection** — All 6 parser commands detect file vs. folder automatically (completed 2026-02-23)
- [ ] **Phase 13: Batch Removal and Format Default** — Drop batch command/flag, make icompta the default format

---

## Phase Details

### Phase 12: Input Auto-Detection

**Goal**: All parser commands accept file or folder input transparently — users never need a separate batch command
**Depends on**: Phases 1-11 (existing parser infrastructure)
**Requirements**: INPUT-01, INPUT-02, INPUT-03, INPUT-04, INPUT-05, INPUT-06
**Success Criteria** (what must be TRUE):

  1. User can run `camt-csv camt path/to/file.xml --output out.csv` and it processes the single file
  2. User can run `camt-csv revolut path/to/folder/ --output ./out/` and it processes every matching file in that folder to individual CSVs
  3. User can run `camt-csv pdf path/to/folder/ --output ./out/` and it consolidates all PDFs into one CSV
  4. When a folder is passed without `--output`, the command exits immediately with a clear error message explaining the flag is required
  5. All 6 parsers (camt, debit, revolut, revolut-investment, selma, pdf) accept both file and folder inputs

**Plans:** 2/2 plans complete

Plans:
- [x] 12-01-PLAN.md — Core folder detection: --output guard + FolderConvert in cmd/common (camt, debit, selma, revolut-investment)
- [ ] 12-02-PLAN.md — Apply auto-detection to revolut and pdf commands, update CHANGELOG

### Phase 13: Batch Removal and Format Default

**Goal**: CLI surface is clean and defaults match real-world use — iCompta output with no flags required, no obsolete batch machinery
**Depends on**: Phase 12
**Requirements**: BATCH-01, BATCH-02, FORMAT-01, FORMAT-02
**Success Criteria** (what must be TRUE):

  1. `camt-csv batch` no longer exists — running it prints an unknown command error
  2. No parser command accepts a `--batch` flag — passing it produces an unknown flag error
  3. Running any parser command with no `--format` flag produces iCompta-compatible semicolon-delimited output
  4. Running `--format standard` still produces the 29-column comma-delimited CSV
**Plans**: TBD

---

## Progress Table

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Critical Bugs & Security | 3/3 | Done | 2026-02-01 |
| 2. Configuration & State Cleanup | 1/1 | Done | 2026-02-01 |
| 3. Architecture & Error Handling | 3/3 | Done | 2026-02-01 |
| 4. Test Coverage & Safety | 4/4 | Done | 2026-02-01 |
| 5. Output Framework | 3/3 | Done | 2026-02-16 |
| 6. Revolut Parsers Overhaul | 3/3 | Done | 2026-02-16 |
| 7. Batch Infrastructure | 2/2 | Done | 2026-02-16 |
| 8. AI Safety Controls | 3/3 | Done | 2026-02-16 |
| 9. Batch-Formatter Integration | 3/3 | Done | 2026-02-16 |
| 10. CSV Format Trim | 1/1 | Done | 2026-02-16 |
| 11. Integration Verification | 2/2 | Done | 2026-02-16 |
| 12. Input Auto-Detection | 2/2 | Complete   | 2026-02-23 |
| 13. Batch Removal and Format Default | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-01*
*Last updated: 2026-02-23 — Phase 12 Plan 01 complete (1/2)*
