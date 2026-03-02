# Product Requirements Document: camt-csv

**Version:** 1.4 (current)
**Last updated:** 2026-02-23
**Status:** Active development

---

## 1. Overview

**camt-csv** is a Go CLI tool that converts financial statement exports from Swiss and European banks into a standardized CSV format, with optional AI-powered transaction categorization. Its primary purpose is to bridge the gap between raw bank exports (XML, PDF, proprietary CSV) and personal finance management in **iCompta** (macOS).

### Problem Statement

Swiss and European bank customers receive financial data in a variety of formats — CAMT.053 XML (ISO 20022 standard), scanned/digital PDFs, and app-specific CSV exports (Revolut, Selma). None of these import directly into iCompta without manual reformatting. Doing this by hand for hundreds of transactions per month is error-prone and time-consuming.

### Solution

A single CLI tool that:

1. Accepts any supported bank export format as input
2. Parses and normalizes transactions into a unified internal model
3. Categorizes transactions using a four-tier strategy (exact match → keyword → semantic embedding → AI)
4. Outputs iCompta-compatible CSV (or standard 29-column CSV for other uses)

---

## 2. Users

### Primary User

**Personal finance manager (solo user)** — technically comfortable, uses macOS, manages personal and household finances in iCompta. Has bank accounts at Swiss banks (CAMT.053), holds Revolut CHF and EUR accounts, and uses Selma for automated investing.

**Workflow today:**

1. Download statements from each bank/app
2. Run `camt-csv` to convert each file
3. Import resulting CSV into iCompta

**Pain points addressed:**

- Multiple bank formats require separate manual handling
- iCompta cannot import raw CAMT.053 XML
- Transaction categories must be manually assigned in iCompta for hundreds of transactions
- Revolut exports mix account types (Current/Savings) and currencies in one file

### Secondary Audience

Go developers and financial data engineers who need a reference implementation of ISO 20022 CAMT.053 parsing or multi-format CSV normalization in Go.

---

## 3. Goals

### v1.x Goals (current milestone family)

| Goal | Status |
|------|--------|
| Parse all 6 input formats reliably | ✅ Shipped (v1.0–v1.2) |
| Output iCompta-compatible CSV | ✅ Shipped (v1.2) |
| Four-tier AI categorization with safety controls | ✅ Shipped (v1.2) |
| Standard CSV trimmed to useful 29 columns | ✅ Shipped (v1.3) |
| Eliminate batch/single-file split (folder detection) | 🔵 v1.4 in progress |
| iCompta format as default (no flag needed) | 🔵 v1.4 in progress |

### Non-Goals

- Web UI or API server
- Mobile app
- Multi-user / collaborative features
- Real-time bank data sync (API-based import)
- Database backend (YAML files are sufficient at personal scale)

---

## 4. Supported Input Formats

| Command | Format | Source |
|---------|--------|--------|
| `camt` | CAMT.053 XML (ISO 20022 v001.02) | Swiss banks (UBS, Raiffeisen, PostFinance, etc.) |
| `pdf` | PDF bank statements | Any bank that exports PDF statements |
| `revolut` | Revolut CSV export | Revolut CHF / EUR current and savings accounts |
| `revolut-investment` | Revolut Investment CSV | Revolut Stocks/ETF portfolio |
| `selma` | Selma CSV export | Selma automated investment accounts |
| `debit` | Generic debit CSV | Generic debit card transaction exports |

### CAMT.053 Specifics

- Namespace: `urn:iso:std:iso:20022:tech:xsd:camt.053.001.02`
- Version tested: 001.02 (newer versions parsed best-effort)
- Swiss bank extensions: partial support

### Revolut Specifics

All 8 transaction types are preserved in output:

| Type | Example |
|------|---------|
| Card Payment | Coffee at Starbucks |
| Transfer | P2P money transfer |
| Exchange | CHF → EUR conversion |
| Deposit | Top-up from bank |
| Fee | Monthly fee |
| Card Refund | Merchant refund |
| Charge | Subscription charge |
| Cashback | Reward cashback |

`Product` field (Current/Savings) is preserved for routing to the correct iCompta account.

---

## 5. Output Formats

### iCompta Format (default from v1.4)

| Property | Value |
|----------|-------|
| Separator | Semicolon (`;`) |
| Date format | `dd.MM.yyyy` |
| Encoding | UTF-8 |
| Header | Yes |
| Columns | 10 |
| Flag | `--format icompta` (default) |

Matches the existing `CSV-Revolut-CHF`, `CSV-Revolut-EUR`, and `CSV-RevInvest` import plugins in the user's iCompta database. No iCompta configuration changes required.

### Standard Format

| Property | Value |
|----------|-------|
| Separator | Comma (`,`) |
| Date format | RFC3339 |
| Encoding | UTF-8 |
| Header | Yes |
| Columns | 29 |
| Flag | `--format standard` |

For custom tooling, spreadsheet analysis, or other finance apps.

---

## 6. Transaction Categorization

### Four-Tier Strategy Chain

Categorization runs each strategy in order, stopping at the first match:

```
DirectStrategy (1.00) → KeywordStrategy (0.95) → SemanticStrategy (0.90) → AIStrategy (0.80)
```

| Tier | Mechanism | Speed | Source |
|------|-----------|-------|--------|
| 1. Direct | Exact string match on party name | Instant | `database/creditors.yaml` / `database/debtors.yaml` |
| 2. Keyword | Regex/substring match | Fast | `database/categories.yaml` |
| 3. Semantic | Cosine similarity on Gemini embeddings | Medium | Computed from `categories.yaml` |
| 4. AI | Gemini generative API | Slow | Google Gemini API |

Each result carries a **confidence score** (1.00 → 0.80) for audit and filtering.

### Auto-Learn Behavior

| Mode | Behavior |
|------|----------|
| Default (no flag) | AI suggestions written to staging files for manual review |
| `--auto-learn` | AI suggestions written directly to live YAML (with backup) |

Staging files: `database/staging_creditors.yaml`, `database/staging_debtors.yaml`

Before any live YAML write, a `.backup` copy is created (one rolling backup per file).

### AI Safety Controls

- **Rate limit:** 1 request/second, burst=1 (prevents Gemini quota exhaustion)
- **Retry:** Exponential backoff on transient errors (3 retries, max 30s delay)
- **Auto-learn default OFF:** Prevents silent data corruption on first use
- **Testability:** `AIClient` interface enables mock injection; `TEST_MODE=true` disables real API calls

---

## 7. CLI Interface (v1.4 Target)

### Single-File Mode

```bash
camt-csv <command> <input-file> --output <output-file>
```

### Folder Mode (v1.4 — replaces batch)

```bash
camt-csv <command> <input-folder> --output <output-folder>
```

Automatically processes all matching files in the folder (non-recursive). `--output` is required for folder input.

**PDF folder mode:** Consolidates all PDFs into one CSV (matching prior `--consolidate` behavior).

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `icompta` | Output format: `icompta` or `standard` |
| `--log-level` | `info` | Logging verbosity |
| `--ai-enabled` | `true` | Enable/disable AI categorization |
| `--auto-learn` | `false` | Write AI results directly to YAML |
| `--date-format` | (per-parser) | Override date format in output |

### Removed in v1.4

- `batch` subcommand → replaced by folder detection in every command
- `--batch` flag → removed from all parser commands

---

## 8. Architecture Decisions Summary

| ADR | Decision |
|-----|----------|
| ADR-001 | Segregated parser interfaces composed into `FullParser` |
| ADR-002 | Four-tier hybrid categorization (direct → keyword → semantic → AI) |
| ADR-003 | Functional programming patterns (pure functions, no global state) |
| ADR-004 | Viper-based hierarchical config (file → env → flags) |
| ADR-005 | Revolut Investment parser with SELL and CUSTODY_FEE support |
| ADR-006 | Gemini HTTP API integration via `AIClient` interface |
| ADR-007/008 | Logging abstraction with DI-injected logger |
| ADR-009 | Semantic routing via Gemini embedding cosine similarity |
| ADR-010 | Three-tier error severity (fatal / retryable / recoverable) |
| ADR-011 | Rolling YAML backup before every auto-learn write |
| ADR-012 | `OutputFormatter` plugin system (strategy pattern) |
| ADR-013 | Universal `BatchProcessor` via composition (superseded v1.4) |
| ADR-014 | iCompta semicolon/dd.MM.yyyy format matching existing import plugins |
| ADR-015 | AI safety: rate limit, backoff, auto-learn OFF default, confidence scores |
| ADR-016 | Standard CSV trimmed to 29 columns (removed 6 unused columns) |
| ADR-017 | Input auto-detection: file → single, folder → multi (v1.4) |

Full ADR files: `docs/adr/`

---

## 9. Configuration

### Hierarchy (later overrides earlier)

1. Config file: `~/.camt-csv/camt-csv.yaml` or `.camt-csv/config.yaml`
2. `.env` file (auto-loaded from current directory)
3. Environment variables
4. CLI flags

### Key Environment Variables

| Variable | Config Key | Purpose |
|----------|-----------|---------|
| `GEMINI_API_KEY` | `ai.api_key` | Google Gemini API key |
| `CAMT_AI_ENABLED` | `ai.enabled` | Enable/disable AI (true/false) |
| `CAMT_AI_MODEL` | `ai.model` | Gemini model name |
| `CAMT_LOG_LEVEL` | `log.level` | Logging level |

---

## 10. Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| **Reliability** | Zero data loss on file conversion; bad input returns clear error, not silent empty output |
| **Safety** | AI auto-learn defaults OFF; YAML backup before every write |
| **Testability** | All dependencies injected; `TEST_MODE=true` for CI without API keys |
| **Backwards compatibility** | CLI flags and YAML file formats preserved across minor versions |
| **Performance** | Personal-scale (hundreds to low thousands of transactions) — throughput not critical |
| **Portability** | macOS (primary), Linux, Windows via Go cross-compilation |

---

## 11. Release History

| Version | Theme | Key Deliverables | Shipped |
|---------|-------|-----------------|---------|
| v1.1 | Hardening | Bug fixes, security hardening, error handling, test coverage | 2026-02-01 |
| v1.2 | Full Polish | iCompta formatter, Revolut overhaul, batch infrastructure, AI safety | 2026-02-16 |
| v1.3 | CSV Trim | Standard format trimmed to 29 columns | 2026-02-16 |
| v1.4 | Simplify | Folder auto-detection, drop batch, icompta default | In progress |

---

## 12. Future Considerations

| Idea | Notes |
|------|-------|
| PDF strategy pattern refactor | Deferred to v2; current implementation works but is not extensible |
| MT940 / OFX / QIF parsers | Requested format support; architecture supports adding parsers |
| Cross-file exchange pairing | Match CHF→EUR and EUR→CHF entries across Revolut files |
| YAML concurrent access protection | Single-threaded per command currently; needed if parallelism added |
| Parametric iCompta column mapping | Allow users with different import plugin configs to map columns via YAML |
| Recursive folder scanning | Not needed for current flat-directory use case |

---

*PRD created: 2026-02-23*
*Owner: fjacquet*
