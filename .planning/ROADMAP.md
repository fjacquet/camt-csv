# Roadmap: camt-csv

## Overview

This roadmap tracks the evolution of camt-csv from a functional MVP through codebase hardening (v1.1), full feature polish (v1.2), standard CSV format optimization (v1.3), operational simplification (v1.4), Jumpsoft Money export support (v1.5), and multi-LLM provider support (v1.6).

## Milestones

- Completed **v1.1 Hardening** - Phases 1-4 (shipped 2026-02-01)
- Completed **v1.2 Full Polish** - Phases 5-9 (shipped 2026-02-16)
- Completed **v1.3 Standard CSV Trim** - Phases 10-11 (shipped 2026-02-16)
- Completed **v1.4 Simplify** - Phases 12-13 (shipped 2026-02-23)
- Completed **v1.5 Jumpsoft Money Export** - Phases 14-15 (shipped 2026-03-02)
- Active **v1.6 Multi-LLM Provider** - Phase 16 (in progress)

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

<details>
<summary>Completed v1.5 Jumpsoft Money Export (Phases 14-15) - SHIPPED 2026-03-02</summary>

- [x] **Phase 14: JumpsoftFormatter** - Build and register the formatter with full CLI integration (completed 2026-03-02)
- [x] **Phase 15: Verification** - Unit and integration tests confirming correct output (completed 2026-03-02)

Full details: `.planning/milestones/v1.5-ROADMAP.md`

</details>

### v1.6 Multi-LLM Provider (In Progress)

**Milestone Goal:** Make AI categorization provider-agnostic — support OpenRouter alongside Gemini, with unified config and graceful embedding fallback.

- [ ] **Phase 16: Multi-LLM Provider** - OpenRouterClient, unified config, provider selection, and graceful semantic handling

## Phase Details

### Phase 16: Multi-LLM Provider
**Goal**: User can select any OpenAI-compatible AI provider via config and have categorization work end-to-end, with semantic tier gracefully adapting
**Depends on**: Phase 15 (existing AIClient interface and GeminiClient pattern)
**Requirements**: PROV-01, PROV-02, PROV-03, PROV-04, CONF-01, CONF-02, CONF-03, CONF-04, SEM-01, SEM-02
**Success Criteria** (what must be TRUE):
  1. User sets `ai.provider: openrouter` and `CAMT_AI_API_KEY` in config; categorization uses OpenRouter instead of Gemini
  2. User sets `ai.base_url` to any OpenAI-compatible endpoint and categorization calls that URL
  3. User sets `ai.model: mistralai/mistral-small-2603`; model name passes through to API unchanged
  4. Starting with `GEMINI_API_KEY` set (old behavior) still works — backward compatibility preserved
  5. Config validation rejects missing api_key or empty model with a clear error before any API call is made
  6. OpenRouter without `GEMINI_API_KEY` → semantic tier skips safely, chat tier categorizes
  7. OpenRouter with `GEMINI_API_KEY` → semantic tier uses Gemini embeddings, chat tier uses OpenRouter
  8. Log output clearly indicates when the semantic tier is skipped vs. active

Plans:
- [ ] 16-01: Unified AI config (provider, base_url, CAMT_AI_API_KEY, backward-compat, validation)
- [ ] 16-02: OpenRouterClient (AIClient interface, retry, rate-limiting)
- [ ] 16-03: Container wiring and semantic graceful handling

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-4 Hardening | v1.1 | 11/11 | Complete | 2026-02-01 |
| 5-9 Full Polish | v1.2 | 14/14 | Complete | 2026-02-16 |
| 10-11 CSV Trim | v1.3 | 3/3 | Complete | 2026-02-16 |
| 12-13 Simplify | v1.4 | 4/4 | Complete | 2026-02-23 |
| 14-15 Jumpsoft Export | v1.5 | 2/2 | Complete | 2026-03-02 |
| 16. Multi-LLM Provider | v1.6 | 0/3 | Not started | - |

---
*Roadmap created: 2026-02-01*
*Last updated: 2026-03-23 — v1.6 roadmap added (phases 16-17)*
