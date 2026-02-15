# Feature Landscape: Revolut Parser Overhaul + iCompta Import + AI Safety

**Project:** camt-csv milestone — Revolut transaction-type intelligence, iCompta CSV export, AI auto-learn controls
**Researched:** February 15, 2026
**Overall Confidence:** MEDIUM-HIGH (Revolut API verified, iCompta community documentation moderate, AI safeguards patterns high)

## Table Stakes

Features users expect from a financial converter. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Parse all Revolut transaction types** | Real data shows 7+ types (Transfer, Card Payment, Exchange, Deposit, Fee, Charge, Refund). Current parser accepts all but provides no semantic handling. | Medium | Revolut API has been stable for years. Semantic handling (distinguishing pocket transfers from inter-account moves) requires description parsing. |
| **iCompta CSV export** | Target user has iCompta on macOS and needs to import 3+ years of standardized transactions. CSV is universal import format. | Medium | iCompta supports 5 formats; CSV is most portable. Import limitation: iCompta cannot auto-recognize transfers, so linked transactions must be flagged in CSV for post-import manual linking or conversion. |
| **Categorical accuracy** | If user imports 1,000 transactions without categories, they lose time manually fixing wrong categorizations. Auto-learn helps but only if safe. | Low-High (depends on safety gates) | Existing 3-tier categorization works; safety gates determine usability. |
| **Prevent silent categorization errors** | If AI quietly mislabels 50 transactions and saves to YAML, user may not notice until reconciliation. Loss of trust. | High | Requires approval gates and logging. |
| **Multi-currency exchange pairing** | Revolut CSV exports exchange transactions in BOTH currencies: EUR side and CHF side as separate rows. Importer must link them. | High | Real data shows this is common; naive parsing creates duplicate entries. iCompta expects transfer linking. |

## Differentiators

Features that set product apart. Not expected, but valued. These create reason to choose over manual import.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Transaction type semantics** | User understands WHY each transaction happened. "Transfer to pocket" vs "inter-account transfer" vs "payment" are actionable distinctions. | Medium | Requires parsing descriptions (e.g., "To CHF Vacances" for pockets). Resolves ambiguity in Revolut CSV Type field. |
| **Automatic exchange pairing** | Importer automatically recognizes EUR↔CHF exchanges, deduplicates, and flags for iCompta linking. Zero manual work. | High | Requires matching logic: same amount, opposite sign, same date, matching currency pairs. iCompta post-import can auto-link with "Convert to transfer" UI. |
| **AI auto-learn with audit trail** | AI categorizes transaction → user reviews → approves/rejects → YAML updated + decision logged. Full traceability, zero surprises. | High | Differentiates from naive "fire-and-forget" AI. Needs approval command or interactive prompt flow. |
| **Pocket-aware categorization** | "Transfer to CHF Vacances" recognized as savings goal, not payment. Proper category assignment. | Low | Builds on transaction type semantics. Pocket metadata from Revolut description. |
| **Batch import with safety dashboard** | Process 100 files with side-by-side category approval. See which files had high AI confidence vs. low. | High | Advanced UX. First phase: single-file approval. Batch approval second phase. |

## Anti-Features

Features to explicitly NOT build (scope creep guards).

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Auto-resolve iCompta transfers** | iCompta's database is closed; no API to create/link transfers programmatically. Attempting this causes data corruption. | Export CSV with transaction type hints (TRANSFER_PENDING_LINK) + user manual "Convert to Transfer" in iCompta UI. Document process. |
| **Support all Revolut API endpoints** | API access requires OAuth, business account, or special approval. CLI tool works with exported CSV (user-friendly, no keys needed). | Stick to CSV export parsing. If user needs API, recommend Revolut business SDK directly. |
| **Real-time balance tracking** | Revolut balance field in CSV is stale (snapshot at export time). Building a reconciliation system = separate tool. | Only use balance for validation warnings: "Balance dropped 40% — verify reconciliation." |
| **Custom category taxonomies per user** | Each user's iCompta has different categories. Forcing standard taxonomy breaks user workflows. | Output to iCompta standard format (colon-separated hierarchies) + let user map in iCompta. |
| **Transaction splitting** | User uploaded one TRANSFER for 100 CHF. It should become two splits in iCompta (from account A to B). Currently not supported by iCompta import. | Defer to Phase 2. For now, parse as single transaction. Document limitation. |

## Feature Dependencies

```
1. Parse all transaction types (base)
   ↓
2. Transaction type semantics (build on parse)
   ↓
3. Multi-currency exchange pairing (advanced logic on types)

1. 35-column standard CSV (exists)
   ↓
2. iCompta CSV export (format known, extend existing)
   ↓
3. Transfer linking hints (CSV column for iCompta post-import)

1. Three-tier categorization (exists)
   ↓
2. AI categorization (exists, in use)
   ↓
3. Auto-learn approval gates (new safeguard)
   ↓
4. Audit trail logging (enable full traceability)

Rate limiting (independent, enables safe API usage)
   ↓
AI auto-learn (API calls need throttling)
```

## MVP Recommendation

### Phase A: Revolut Transaction Type Semantics + iCompta Export (8–10 days)

**Rationale:** Unblocks iCompta import workflow for user. Provides semantic value without risk.

1. **Revolut Type Parsing** — Parse all 7 types; add semantic enrichment (pocket detection, exchange pairing candidate identification)
2. **iCompta CSV Export** — Extend standard CSV with iCompta-specific fields:
   - `LinkedTransaction` flag for exchange pairs (e.g., "EUR-CHF-2025-02-15-100")
   - Hierarchical category format (colon-separated for iCompta parent:child)
   - Transfer type field for iCompta post-import conversion
3. **Exchange Pairing Detection** — Identify EUR↔CHF pairs with same amount, opposite sign, same date. Log pair IDs in CSV.

**Complexity:** Low-Medium

- Revolut type handling: pattern matching on descriptions
- CSV export: add 3–4 columns to existing template
- Pairing: O(n²) scan, but usually <100 exchange transactions per file

### Phase B: AI Auto-Learn Approval Gates + Rate Limiting (5–7 days)

**Rationale:** Makes AI safe for production use. Prevents silent miscategorization.

1. **Approval Workflow:**
   - Flag high-confidence AI categorizations (>85%) for auto-save
   - Prompt user for review on medium confidence (50–85%)
   - Require explicit approval on low confidence (<50%) or novel categories
   - `--auto-approve` flag for batch runs (explicit opt-in)

2. **Audit Trail:**
   - Log each AI categorization: confidence score, category, decision (approved/rejected), timestamp
   - Store in `.camt-csv/audit.log` (JSON lines format)
   - Include in batch reports

3. **Rate Limiting:**
   - Use `golang.org/x/time/rate.Limiter` for Gemini API calls
   - Default: 10 requests/second (Gemini free tier limit)
   - Configurable via `--ai-rate-limit N` flag
   - Graceful degradation: if rate limit hit, revert to keyword-only categorization

**Complexity:** Medium

- Approval: prompts or flag handling (straightforward CLI logic)
- Audit trail: JSON logging to file
- Rate limiting: standard Go library integration

### Phase C: Batch Conversion + Dashboard (deferred to Phase 2)

**Defer because:** User's immediate need is single-file import. Batch dashboard adds UI complexity without solving critical workflow first.

## Feature Complexity Summary

| Feature | Lines of Code Est. | Risk | Go Expertise Needed |
|---------|-------------------|------|-------------------|
| Revolut type semantics | 150–200 | Low | Low (string parsing) |
| Exchange pairing detection | 100–150 | Low | Low (maps, sorting) |
| iCompta CSV export | 200–250 | Low | Low (template extension) |
| AI approval workflow | 300–400 | Medium | Medium (CLI prompts, state mgmt) |
| Audit trail logging | 150–200 | Low | Low (JSON serialization) |
| Rate limiting integration | 100–150 | Low | Low (standard library) |

## Known Unknowns

- **iCompta category hierarchy depth:** Does iCompta support "Parent:Child:Grandchild" or only one level deep? Need to test or verify in iCompta forums. → **Action:** Test with sample iCompta export before Phase A commit.
- **Revolut field expansion:** Are there undocumented fields in user's Revolut exports? → **Action:** Ask user to provide sample export; scan for Type/Product/Description patterns beyond documented ones.
- **Multi-currency file handling:** User has separate CHF and EUR files. Should the tool merge them for iCompta import or treat separately? → **Action:** Clarify with user on workflow; likely separate files are correct.
- **Gemini rate limiting in production:** Is 10 req/s safe? Should it be lower? → **Action:** Monitor in testing; adjust based on quota feedback.

## Sources

- [Revolut Developer Docs — Transactions](https://developer.revolut.com/docs/business/transactions)
- [iCompta Help — Formats Supported](https://www.icompta-app.com/help.php)
- [iCompta Forums — CSV Import Category Format](https://forums.lyricapps.com/viewtopic.php?t=1822)
- [iCompta Forums — Transfer Recognition Limitation](https://forums.lyricapps.com/viewtopic.php?t=3102)
- [Golang.org/x/time/rate — Token Bucket](https://pkg.go.dev/golang.org/x/time/rate)
- [Go Wiki — Rate Limiting](https://go.dev/wiki/RateLimiting)
- [DocuClipper — Auto-Categorization & User Confirmation Patterns](https://www.docuclipper.com/blog/automatic-transaction-categorization/)
