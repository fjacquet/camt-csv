# Roadmap: camt-csv

## Overview

This roadmap tracks the evolution of camt-csv from a functional MVP (v1.0) through codebase hardening (v1.1) to full feature polish (v1.2). The v1.2 milestone transforms Revolut parsers from basic CSV extraction into semantically-aware converters, standardizes output across all parsers for iCompta compatibility, adds AI auto-learn safety controls, and completes batch processing infrastructure.

## Milestones

- ✅ **v1.1 Hardening** - Phases 1-4 (shipped 2026-02-01)
- 🚧 **v1.2 Full Polish** - Phases 5-8 (in progress)

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

<details>
<summary>✅ v1.1 Hardening (Phases 1-4) - SHIPPED 2026-02-01</summary>

- [x] **Phase 1: Critical Bugs & Security** (3/3 plans)ss - completed 2026-02-01
- [x] **Phase 2: Configuration & State Cleanup** (1/1 plans) - completed 2026-02-01
- [x] **Phase 3: Architecture & Error Handling** (3/3 plans) - completed 2026-02-01
- [x] **Phase 4: Test Coverage & Safety** (4/4 plans) - completed 2026-02-01

Full details: `.planning/milestones/v1.1-ROADMAP.md`

</details>

### 🚧 v1.2 Full Polish (In Progress)

**Milestone Goal:** Overhaul Revolut parsers with transaction-type intelligence, standardize CSV output across all parsers for iCompta import compatibility, add AI auto-learn safety, and bring batch support everywhere.

- [ ] **Phase 5: Output Framework** - Standardize CSV output with iCompta compatibility
- [ ] **Phase 6: Revolut Parsers Overhaul** - Transaction type intelligence and investment support
- [ ] **Phase 7: Batch Infrastructure** - Universal batch processing with error handling
- [ ] **Phase 8: AI Safety Controls** - Safe AI auto-learning with rate limiting

## Phase Details

### Phase 5: Output Framework

**Goal**: All parsers produce standardized, iCompta-compatible CSV output with configurable formatting
**Depends on**: Phase 4 (v1.1 complete)
**Requirements**: OUT-01, OUT-02, OUT-03, OUT-04
**Success Criteria** (what must be TRUE):

  1. User can select output format via `--format` flag (standard, icompta)
  2. iCompta format produces CSV that imports cleanly into iCompta without data loss
  3. User can configure date format in output (DD.MM.YYYY, YYYY-MM-DD, etc.)
  4. All parsers (CAMT, PDF, Revolut, Selma, Debit) support both standard and iCompta formats
  5. Legacy CSV output remains unchanged when using standard format (backward compatible)
**Plans**: TBD

### Phase 6: Revolut Parsers Overhaul

**Goal**: Revolut parsers understand transaction semantics and output standardized format
**Depends on**: Phase 5
**Requirements**: REV-01, REV-02, REV-03, REV-04, REV-05, RINV-01, RINV-02, RINV-03, BATCH-02
**Success Criteria** (what must be TRUE):

  1. Revolut parser correctly identifies all transaction types (Transfer, Card Payment, Exchange, Deposit, Fee, Charge, Card Refund, Charge Refund)
  2. Revolut parser outputs 35-column standardized CSV format matching other parsers
  3. Exchange transactions preserve both currencies with original amounts visible in output
  4. Product field (Current/Savings) appears in Transaction model and CSV output
  5. REVERTED and PENDING transactions are either skipped or flagged based on user preference
  6. Revolut Investment parser handles SELL transactions correctly
  7. Revolut Investment parser handles CUSTODY_FEE transactions correctly
  8. Revolut Investment parser supports batch conversion mode
**Plans**: TBD

### Phase 7: Batch Infrastructure

**Goal**: All parsers support batch processing with comprehensive error reporting
**Depends on**: Phase 6
**Requirements**: BATCH-01, BATCH-03
**Success Criteria** (what must be TRUE):

  1. PDF parser supports batch conversion mode
  2. Batch mode generates manifest showing succeeded/failed files
  3. Batch processing continues after individual file failures
  4. Failed files are logged with specific error messages
  5. Exit code reflects batch status (0=all success, 1=partial, 2=all failed)
**Plans**: TBD

### Phase 8: AI Safety Controls

**Goal**: AI categorization has safety gates preventing silent miscategorization
**Depends on**: Phase 7
**Requirements**: AI-01, AI-02, AI-03
**Success Criteria** (what must be TRUE):

  1. User can control AI auto-learning via `--auto-learn` flag (default: review required)
  2. Gemini API calls respect rate limits to avoid quota exhaustion
  3. Gemini API calls retry with exponential backoff on transient failures
  4. AI categorizations are logged with confidence scores before saving
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Critical Bugs & Security | v1.1 | 3/3 | Complete | 2026-02-01 |
| 2. Configuration & State Cleanup | v1.1 | 1/1 | Complete | 2026-02-01 |
| 3. Architecture & Error Handling | v1.1 | 3/3 | Complete | 2026-02-01 |
| 4. Test Coverage & Safety | v1.1 | 4/4 | Complete | 2026-02-01 |
| 5. Output Framework | v1.2 | 0/? | Not started | - |
| 6. Revolut Parsers Overhaul | v1.2 | 0/? | Not started | - |
| 7. Batch Infrastructure | v1.2 | 0/? | Not started | - |
| 8. AI Safety Controls | v1.2 | 0/? | Not started | - |

---
*Roadmap created: 2026-02-01*
*Last updated: 2026-02-15 for v1.2 milestone*
