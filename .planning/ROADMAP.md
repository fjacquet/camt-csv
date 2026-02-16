# Roadmap: camt-csv

## Overview

This roadmap tracks the evolution of camt-csv from a functional MVP through codebase hardening (v1.1), full feature polish (v1.2), and standard CSV format optimization (v1.3).

## Milestones

- ✅ **v1.1 Hardening** — Phases 1-4 (shipped 2026-02-01)
- ✅ **v1.2 Full Polish** — Phases 5-9 (shipped 2026-02-16)
- 🚧 **v1.3 Standard CSV Trim** — Phases 10-11 (in progress)

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

### 🚧 v1.3 Standard CSV Trim (In Progress)

**Milestone Goal:** Remove 6 redundant/dead fields from 35-column standard CSV format, keeping only fields with actual data.

- [x] **Phase 10: CSV Format Trim** (1 plan) - Remove 6 fields from StandardFormatter and update model (completed 2026-02-16)
- [ ] **Phase 11: Integration Verification** - Verify all parsers and tests pass with new 29-column format

## Phase Details

### Phase 10: CSV Format Trim
**Goal**: Standard CSV format reduced to 29 columns with no redundant fields
**Depends on**: Phase 9 (v1.2 shipped)
**Requirements**: CSV-01, CSV-02, CSV-03, CSV-04, CSV-05, CSV-06, INT-01, INT-02
**Success Criteria** (what must be TRUE):
  1. StandardFormatter header contains exactly 29 columns (6 fields removed)
  2. Removed columns: BookkeepingNumber, IsDebit, Debit, Credit, Recipient, AmountTax
  3. Transaction.MarshalCSV produces 29-column CSV output matching new header
  4. Transaction.UnmarshalCSV correctly parses 29-column CSV input
  5. Example CSV output from any parser shows 29 columns with no empty redundant fields
**Plans**: 1 plan

Plans:
- [ ] 10-01-PLAN.md — Update StandardFormatter and Transaction CSV methods to 29 columns

### Phase 11: Integration Verification
**Goal**: All parsers and tests work correctly with 29-column format
**Depends on**: Phase 10
**Requirements**: INT-03, INT-04, INT-05
**Success Criteria** (what must be TRUE):
  1. All parser unit tests pass (CAMT, PDF, Revolut, Revolut Investment, Selma, Debit)
  2. Integration tests (cross-parser consistency) pass with new column count
  3. iCompta formatter remains unchanged at 10 columns with semicolon separator
  4. End-to-end test: convert sample file from each parser, verify 29 columns in standard format
  5. End-to-end test: convert sample file from each parser, verify 10 columns in iCompta format
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 10 → 11

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Critical Bugs & Security | v1.1 | 3/3 | Complete | 2026-02-01 |
| 2. Configuration & State Cleanup | v1.1 | 1/1 | Complete | 2026-02-01 |
| 3. Architecture & Error Handling | v1.1 | 3/3 | Complete | 2026-02-01 |
| 4. Test Coverage & Safety | v1.1 | 4/4 | Complete | 2026-02-01 |
| 5. Output Framework | v1.2 | 3/3 | Complete | 2026-02-16 |
| 6. Revolut Parsers Overhaul | v1.2 | 3/3 | Complete | 2026-02-16 |
| 7. Batch Infrastructure | v1.2 | 2/2 | Complete | 2026-02-16 |
| 8. AI Safety Controls | v1.2 | 3/3 | Complete | 2026-02-16 |
| 9. Batch-Formatter Integration | v1.2 | 3/3 | Complete | 2026-02-16 |
| 10. CSV Format Trim | v1.3 | Complete    | 2026-02-16 | - |
| 11. Integration Verification | v1.3 | 0/? | Not started | - |

---
*Roadmap created: 2026-02-01*
*Last updated: 2026-02-16 — v1.3 Phase 10 planning complete*
