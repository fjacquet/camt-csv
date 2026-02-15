# Project Research Summary

**Project:** camt-csv — Go CLI financial converter with Revolut enhancements
**Domain:** Financial data format conversion and categorization
**Researched:** 2026-02-15
**Confidence:** HIGH (codebase inspection) / MEDIUM-HIGH (feature/architecture patterns) / MEDIUM (pitfalls require field validation)

## Executive Summary

The camt-csv project is a mature Go CLI tool for converting financial statements across multiple formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV) into standardized CSV with AI-powered transaction categorization. The codebase demonstrates solid architecture: a dependency injection container, segregated interfaces (Parser, Categorizer, OutputFormatter), and factory patterns for extensibility. The upcoming milestone focuses on enhancing the Revolut parser with semantic transaction-type intelligence, adding iCompta export capability, implementing AI auto-learn safety controls, and enabling batch processing.

The research confirms that minimal new dependencies are needed. Core technologies (Go, Cobra, Viper, decimal.Decimal) are proven; only file locking (`fslock`) is a justified new addition for Phase 3. The primary risk is **breaking changes to CSV output format** — moving from 4 columns to 35 columns will silently corrupt downstream user scripts unless versioned carefully with a 2-release deprecation path. Secondary risks include multi-currency rounding cascades, concurrent YAML corruption, and type detection over-eagerness that breaks special-case handling.

The recommended approach prioritizes backward compatibility: Phase 1 establishes CSV versioning and iCompta format as a separate output mode (not a replacement). Phase 2 adds transaction-type semantics and multi-currency exchange pairing. Phase 3 implements batch processing with manifests and file locking. Phase 4 adds AI approval workflows. This ordering prevents data loss, isolates breaking changes, and allows validation at each stage.

## Key Findings

### Recommended Stack

The technology stack is **minimal and proven**. All core dependencies already exist in go.mod:

**Core technologies:**
- **Go 1.21+** — Compiled binary, excellent stdlib (csv, sync, errors), no runtime dependencies
- **Cobra v1.7+** — CLI framework, already integrated and battle-tested
- **Viper v1.16+** — Hierarchical config (file → env → flags), already in place
- **decimal.Decimal (shopspring)** — Financial precision (already used), essential to prevent float rounding errors
- **gocsv (gocarina)** — CSV struct unmarshal, simplifies Revolut/Selma format parsing
- **logrus v1.9+** — Structured logging, already integrated

**New dependency (Phase 3+):**
- **fslock (theckman) v0.8.1+** — Portable file locking for YAML writes during batch processing. No stdlib equivalent; prevents concurrent write corruption.

**Supporting libraries already in place:**
- `sync.RWMutex` and `sync.Mutex` — In-process synchronization (codebase pattern established)
- `encoding/csv` — CSV I/O from stdlib
- `time` — Date parsing with custom format support (DD.MM.YYYY)
- `google/uuid` — Transaction IDs for compliance reports

No external ML libraries needed; Gemini API used via HTTP client for AI categorization with rate limiting via `golang.org/x/time/rate` (stdlib).

**Rationale:** The "minimal deps" approach reduces risk, keeps the binary small, and maintains developer velocity. The only gap is file locking for concurrent YAML writes in batch scenarios — justified because it prevents data loss and has no stdlib alternative.

### Expected Features

Three categories of features identified from domain research:

**Must have (table stakes):**
- Parse all Revolut transaction types (7+ types: Transfer, Card Payment, Exchange, Deposit, Fee, Charge, Refund, Investment). Users expect semantic understanding, not just CSV passthrough.
- Categorical accuracy with safety gates. Current 3-tier system (direct → keyword → AI) works; safety gates prevent silent miscategorization.
- Prevent silent categorization errors. If AI mislabels 50 transactions and saves to YAML without user review, trust is lost. Requires approval gates and audit logs.
- Multi-currency exchange pairing for iCompta linking. Revolut exports EUR↔CHF exchanges as separate rows; importer must link them or risk duplicates.
- iCompta CSV export. Target user has iCompta on macOS; CSV is standard import format but requires format compliance (hierarchy, decimal separator, date format).

**Should have (competitive differentiators):**
- Transaction type semantics. Distinguish "Transfer to CHF Vacances" (savings goal) from inter-account transfers (bookkeeping). Requires description parsing.
- Automatic exchange pairing. Detect EUR↔CHF pairs (same amount, opposite sign, same date), flag for iCompta post-import linking. Zero manual work.
- AI auto-learn with audit trail. Categorizations logged with confidence scores, decisions (approved/rejected), timestamps. Full traceability.
- Pocket-aware categorization. Revolut pocket transfers recognized as savings, not payments.
- Batch import with safety dashboard. Process 100+ files with category approval; see which had high AI confidence vs. low (Phase 2 deferral).

**Defer to v2+ (out of scope):**
- Auto-resolve iCompta transfers. iCompta API closed; no way to create/link transfers programmatically. Document manual process instead.
- Real-time balance tracking. Revolut balance is stale snapshot; reconciliation system is separate tool.
- Custom category taxonomies. Each user's iCompta has different categories; force standard taxonomy → breaks workflows. Output to iCompta format, let user map.
- Transaction splitting. iCompta doesn't support split imports; defer to v2.

### Architecture Approach

The codebase uses **segregated interface composition with dependency injection** — proven patterns that minimize coupling and enable testing.

**Core architecture patterns:**
- **FullParser composite interface** — All parsers implement: Parse, Validator, CSVConverter, LoggerConfigurable, CategorizerConfigurable, BatchConverter. Wired via Container.
- **Factory + DI Container** — Parsers registered in factory; CLI commands get parser from container, not direct factory call. Ensures categorizer injected correctly.
- **BaseParser template** — Shared functionality (logging, CSV write, batch template method). New parsers extend BaseParser.
- **Three-tier categorizer** — Direct mapping (YAML) → Keyword matching (rules) → AI fallback (Gemini). Each tier has confidence score.

**Critical pitfalls addressed:**
1. CSV breaking change without migration path — moving 4-column to 35-column breaks user scripts
2. Multi-currency rounding cascades — 0.01-0.05 CHF drifts accumulate across paired exchanges
3. YAML concurrent write corruption — batch processes overwrite each other's category learning
4. Type detection breaking special cases — new type parsing breaks "To CHF Vacances" special handling
5. iCompta silent data loss — wrong decimal/date format causes silent skips in import

## Implications for Roadmap

Research suggests a 4-phase structure organized by risk and dependencies:

### Phase 1: CSV Format Versioning & Output Framework
**Rationale:** CSV is user-facing; breaking it corrupts downstream workflows. Must establish migration path FIRST.

**Delivers:**
- CSV format versioning in headers
- OutputFormatter interface with `standard|revolut|icompta` modes
- Migration script for legacy data
- Deprecation warnings

**Avoids:** CSV breaking change, iCompta format mismatches

**Research needed:** No (formatter architecture established in codebase)

---

### Phase 2: Revolut Type Intelligence & Multi-Currency Handling
**Rationale:** Type semantics improve categorization; must complete before AI auto-learn (Phase 4).

**Delivers:**
- Parse all 7+ Revolut transaction types with semantic enrichment
- Exchange pair detection (EUR↔CHF by date/amount/currency)
- Multi-currency rounding strategy with audit fields
- Type detection confidence scoring

**Avoids:** Type detection over-eagerness, rounding cascades, special case breakage

**Research needed:** Type edge cases (disputed/pending from real data), exchange matching on 3+ currencies, iCompta split handling

---

### Phase 3: Batch Processing, File Locking & iCompta Output
**Rationale:** Batch and iCompta features need infrastructure (locking, manifests) to prevent data loss.

**Delivers:**
- Batch conversion with manifests (succeeded/failed summary)
- File-level locking for YAML writes (fslock)
- iCompta category hierarchy mapping
- Exit codes and error reporting

**Avoids:** YAML corruption, silent batch failures, iCompta data loss

**Research needed:** Batch performance (100K+ tx), file locking on APFS/ext4, iCompta wizard edge cases

---

### Phase 4: AI Auto-Learn Safety Controls & Approval Workflow
**Rationale:** AI is powerful but risky without gates. Complete Phases 1-3 first to isolate safety logic.

**Delivers:**
- Confidence thresholds for auto-save (85%+ auto, 50-85% review, <50% approve)
- Audit logging with decision trail
- Rate limiting for Gemini API
- Config snapshots for batch reproducibility

**Avoids:** Silent miscategorization, feature flag inconsistency, learning from bad data

**Research needed:** Approval workflow UX preference, confidence calibration, audit retention policy

---

**Dependency flow:** Phase 1 (CSV) → Phase 2 (Types) → Phase 3 (Batch infrastructure) → Phase 4 (AI safety)

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| **Stack** | HIGH | All core deps in go.mod; only new dep (fslock) well-tested |
| **Features** | MEDIUM-HIGH | Revolut/iCompta documented; edge cases (split handling, type conflicts) unknown |
| **Architecture** | HIGH | Codebase patterns mature; no conflicts; components follow conventions |
| **Pitfalls** | MEDIUM | Critical risks identified; field testing will refine thresholds |

**Overall: HIGH** for stack/architecture, **MEDIUM-HIGH** for features/pitfalls pending field validation.

### Gaps to Address

- iCompta category hierarchy depth limits (test with real export)
- Revolut type edge cases (disputed, pending, reversed states)
- iCompta import wizard silent skip behavior (manual test)
- Batch approval workflow UX preference (flag vs. interactive)
- AI confidence calibration (real Gemini output distribution)
- Multi-currency processing order (sequential vs. in-memory)

---

## Sources

**Primary (HIGH confidence):**
- Codebase inspection (parser interfaces, container, BaseParser patterns)
- Go stdlib docs (sync, encoding/csv, time)
- CLAUDE.md project instructions (user requirements)

**Secondary (MEDIUM confidence):**
- Revolut Developer Docs (transaction types)
- iCompta Help + Forums (import format, limitations)
- shopspring/decimal docs (precision patterns)

**Tertiary (LOW confidence — needs validation):**
- iCompta user experience (forums, documentation, some speculation)
- Gemini confidence score distribution (not tested in real batch scenarios)

---

*Research completed: 2026-02-15*
*Ready for roadmap: yes*
