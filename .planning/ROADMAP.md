# Roadmap: camt-csv

## Overview

This roadmap tracks the evolution of camt-csv from a functional MVP (v1.0) through codebase hardening (v1.1) to full feature polish (v1.2). The v1.2 milestone transforms Revolut parsers from basic CSV extraction into semantically-aware converters, standardizes output across all parsers for iCompta compatibility, adds AI auto-learn safety controls, and completes batch processing infrastructure.

## Milestones

- ✅ **v1.1 Hardening** - Phases 1-4 (shipped 2026-02-01)
- 🚧 **v1.2 Full Polish** - Phases 5-9 (in progress)

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

- [x] **Phase 5: Output Framework** - Standardize CSV output with iCompta compatibility (completed 2026-02-16)
- [x] **Phase 6: Revolut Parsers Overhaul** - Transaction type intelligence and investment support (completed 2026-02-16)
- [x] **Phase 7: Batch Infrastructure** - Universal batch processing with error handling (completed 2026-02-16)
- [x] **Phase 8: AI Safety Controls** - Safe AI auto-learning with rate limiting (completed 2026-02-16)
- [ ] **Phase 9: Batch-Formatter Integration** - Close audit gaps: formatter support in batch/consolidation paths

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

**Plans**: 3 plans in 3 waves

Plans:
- [x] 05-01-PLAN.md — Create OutputFormatter interface and implementations
- [x] 05-02-PLAN.md — Integrate formatters with CSV writing and DI container
- [x] 05-03-PLAN.md — Add CLI flags to all parser commands

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

**Plans**: 3 plans in 2 waves

Plans:
- [x] 06-01-PLAN.md — Add Product field to Transaction model and builder
- [x] 06-02-PLAN.md — Add SELL/CUSTODY_FEE handling and batch support to investment parser
- [x] 06-03-PLAN.md — Enhance Revolut parser to populate all 35 fields with Product and exchange metadata

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

**Plans**: 2 plans in 2 waves

Plans:
- [x] 07-01-PLAN.md — Create shared BatchProcessor infrastructure with manifest generation
- [x] 07-02-PLAN.md — Integrate PDF parser with batch infrastructure and standardize exit codes

### Phase 8: AI Safety Controls

**Goal**: AI categorization has safety gates preventing silent miscategorization
**Depends on**: Phase 7
**Requirements**: AI-01, AI-02, AI-03
**Success Criteria** (what must be TRUE):

  1. User can control AI auto-learning via `--auto-learn` flag (default: review required)
  2. Gemini API calls respect rate limits to avoid quota exhaustion
  3. Gemini API calls retry with exponential backoff on transient failures
  4. AI categorizations are logged with confidence scores before saving

**Plans**: 3 plans in 2 waves

Plans:
- [x] 08-01-PLAN.md — Add confidence metadata and pre-save logging infrastructure
- [x] 08-02-PLAN.md — Implement rate limiting and retry logic in GeminiClient
- [x] 08-03-PLAN.md — Wire --auto-learn flag and gate auto-save behavior

### Phase 9: Batch-Formatter Integration

**Goal**: Batch and consolidation code paths use the formatter pipeline and BatchProcessor infrastructure
**Depends on**: Phase 8
**Requirements**: OUT-01 (full), OUT-04 (full), BATCH-03 (full)
**Gap Closure**: Closes all gaps from v1.2 milestone audit
**Success Criteria** (what must be TRUE):

  1. BatchProcessor supports formatter configuration (format + date-format options)
  2. PDF batch mode (`--batch`) passes format/dateFormat flags through to output
  3. PDF consolidation mode uses formatter pipeline instead of legacy writer
  4. Revolut adapter.BatchConvert uses BatchProcessor (with manifest + exit codes)
  5. `--format icompta` produces correct output in all modes: single file, batch, consolidation

**Plans**: 3 plans in 2 waves

Plans:
- [ ] 09-01-PLAN.md — Add formatter support to BatchProcessor API
- [ ] 09-02-PLAN.md — Migrate Revolut batch to BatchProcessor composition
- [ ] 09-03-PLAN.md — Wire formatters through PDF batch and consolidation paths

## Progress

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
| 9. Batch-Formatter Integration | v1.2 | 0/3 | Not started | - |

---
*Roadmap created: 2026-02-01*
*Last updated: 2026-02-16 — added Phase 9 gap closure from milestone audit*
