# camt-csv

## What This Is

A Go CLI tool that converts financial statement formats (CAMT.053 XML, PDF bank statements, Revolut CSV, Selma CSV) into standardized CSV with AI-powered transaction categorization. Supports iCompta-compatible output formatting, universal batch processing with manifest reporting, and AI safety controls for categorization.

## Core Value

Reliable, maintainable financial data conversion with intelligent categorization.

## Requirements

### Validated

- ✓ CAMT.053 XML parsing and CSV conversion — existing
- ✓ PDF bank statement parsing via pdftotext extraction — existing
- ✓ Revolut CSV export parsing — existing
- ✓ Revolut Investment export parsing — existing
- ✓ Selma CSV export parsing — existing
- ✓ Debit statement parsing — existing
- ✓ Three-tier categorization (direct mapping → keyword → AI/Gemini) — existing
- ✓ Auto-learning of AI categorizations back to YAML — existing
- ✓ Batch processing mode for multiple files — existing
- ✓ PDF directory consolidation for multi-file processing — existing
- ✓ DI container for dependency wiring — existing
- ✓ Viper-based hierarchical configuration — existing
- ✓ Context propagation for cancellation support — existing
- ✓ Cobra CLI with subcommands per format — existing
- ✓ Custom error types with context (ParseError, ValidationError, etc.) — existing
- ✓ PDF parser bug fixes (debug file, context, temp cleanup) — v1.1
- ✓ MockLogger state isolation for test verification — v1.1
- ✓ Security hardening (credential logging, temp naming, permissions) — v1.1
- ✓ Deprecated config removal and global state cleanup — v1.1
- ✓ Standardized error handling patterns (fatal/retryable/recoverable) — v1.1
- ✓ PDF parser temp file consolidation — v1.1
- ✓ Comprehensive test coverage (concurrent, PDF format, error messages) — v1.1
- ✓ Category YAML backup system — v1.1
- ✓ OutputFormatter plugin system (StandardFormatter + iComptaFormatter) — v1.2
- ✓ --format flag selects output format (standard, icompta) on all parsers — v1.2
- ✓ iCompta format maps fields to ICTransaction columns with category — v1.2
- ✓ Configurable --date-format flag on all parsers — v1.2
- ✓ Revolut parser identifies all 8 transaction types — v1.2
- ✓ Revolut parser outputs standardized CSV format (29 columns after v1.3 trim) — v1.2
- ✓ Exchange transactions preserve original currency/amount metadata — v1.2
- ✓ Product field (Current/Savings) in Transaction model and CSV output — v1.2
- ✓ REVERTED/PENDING transactions logged when skipped — v1.2
- ✓ Revolut Investment SELL transaction handling — v1.2
- ✓ Revolut Investment CUSTODY_FEE transaction handling — v1.2
- ✓ Revolut Investment batch conversion support — v1.2
- ✓ PDF parser batch conversion with --batch flag — v1.2
- ✓ Batch failures reported per-file with manifest (continue on error) — v1.2
- ✓ AI --auto-learn flag controls YAML auto-save (default OFF) — v1.2
- ✓ Gemini API rate limiting (burst=1) — v1.2
- ✓ Gemini API retry with exponential backoff — v1.2
- ✓ BatchProcessor universal infrastructure with formatter integration — v1.2

### Active

None — ready for next milestone.

### Shipped (v1.3)

- ✓ Standard CSV format trimmed from 35 to 29 columns — v1.3
- ✓ Removed BookkeepingNumber, IsDebit, Debit, Credit, Recipient, AmountTax — v1.3
- ✓ All parser tests and integration tests updated for 29-column format — v1.3
- ✓ End-to-end tests for both standard and iCompta formats — v1.3

### Out of Scope

- Full PDF parser strategy pattern refactor — deferred to v2
- New input format parsers (MT940, OFX, QIF) — future milestone
- UI/web interface — CLI-only tool
- Database backend — YAML file storage is sufficient for current scale
- Replacing pdftotext dependency — separate initiative requiring evaluation of Go PDF libraries
- Cross-file exchange transaction pairing — complex timing/rounding matching, deferred

## Context

Shipped v1.3 Standard CSV Trim (29-column format). Previously shipped v1.2 Full Polish with 43,619 LOC Go across 132 modified files.
Tech stack: Go 1.24.2, Cobra 1.10.2, Viper 1.21.0, Logrus 1.9.4.
External dependency on `pdftotext` (Poppler utils) for PDF parsing.
Optional dependency on Google Gemini API for AI categorization.
Codebase map available at `.planning/codebase/` with 7 analysis documents.

**Target import app:** iCompta (macOS personal finance). Schema at `.planning/reference/icompta-schema.sql`.
Key tables: ICAccount, ICTransaction, ICTransactionSplit, ICCategory, ICCurrency.
iCompta supports CSV import via configurable ICImportPlugin with column mapping.
iCompta already has CSV-Revolut-CHF, CSV-Revolut-EUR, CSV-RevInvest import plugins configured.

**Real Revolut data profile** (from 2022-2026 exports):

- File 1 (CHF): 2,126 txns — Transfer (1,237), Card Payment (683), Exchange (135), Deposit (64), Fee (4), Charge (2), Card Refund (1)
- File 2 (EUR): 184 txns — Exchange (90), Card Payment (84), Transfer (6), Card Refund (3), Deposit (1)
- Products: Current + Savings. States: COMPLETED (2,121), REVERTED (4), PENDING (1)
- One file per currency, all account types mixed

Known technical debt:

- GetLogrusAdapter() creates new logger when mock injected (testing limitation)
- YAML store lacks concurrent access protection (single-threaded per command currently)
- PDF parser would benefit from strategy pattern refactor (deferred)
- dateFormat parameter reserved but unused in PDF convert.go (future dynamic date formatting)

## Constraints

- **Tech stack**: Go — no language changes
- **Backwards compatibility**: CLI interface and YAML file formats must remain compatible
- **External tools**: pdftotext dependency stays (removal is a separate initiative)
- **Testing**: All changes must maintain or improve test coverage, no regressions
- **AI API**: Gemini integration stays with rate limiting and retry patterns

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Minimal PDF parser fixes only | Full strategy pattern refactor too large; fix bugs without restructuring | ✓ Good — bugs fixed, refactor deferred to v2 |
| Bugs & security first priority | Highest user impact, reduces risk before architectural changes | ✓ Good — clean foundation established |
| Include safety features (backup) | Essential for reliability before adding new capabilities | ✓ Good — backup system implemented |
| Keep pdftotext dependency | Replacing external tool is a separate initiative | — Pending |
| Three-tier error severity | Clear categories make error handling predictable | ✓ Good — documented in CONVENTIONS.md |
| Shared pointer pattern for MockLogger | State isolation while maintaining entry collection | ✓ Good — tests can verify specific log messages |
| Category backup enabled by default | Protects user data during auto-learning | ✓ Good — atomic behavior prevents data loss |
| Accept TEST-01 as adequate | os.Exit testing is known Go limitation | ⚠️ Revisit — refactor logger injection if needed |
| Target iCompta as import destination | User's actual finance app; CSV import with configurable field mapping | ✓ Good — iCompta format implemented and tested |
| Include all Revolut transaction types | User wants full visibility; filtering can be done in iCompta | ✓ Good — all 8 types preserved in output |
| Include Exchange transactions as-is | Tagged with type, not hidden; cross-file pairing deferred to future | ✓ Good — metadata preserved for future pairing |
| Separate iCompta account for Revolut Savings | CHF Vacances pocket has 688 txns; needs own account for proper tracking | ✓ Good — Product field enables routing |
| Route by Product+Currency to iCompta account | Current/CHF→Revolut CHF, Savings/CHF→Revolut CHF Vacances, Current/EUR→Revolut EUR | ✓ Good — Product field populated |
| Match existing iCompta import plugins | CSV-Revolut-CHF/EUR already configured in user's DB; use semicolon separator + dd.MM.yyyy | ✓ Good — iComptaFormatter matches |
| Output formatter is cross-parser | --format flag (standard, icompta) works on all parsers, not just Revolut | ✓ Good — all 6 parsers support both formats |
| AI auto-learn defaults to OFF | Prevent silent miscategorization; user must opt-in with --auto-learn flag | ✓ Good — safe default, explicit opt-in |
| No cross-file exchange pairing in v1.2 | Complex timing/rounding matching; each file processed independently | — Deferred to future |
| Strategy pattern for formatters | Extensible plugin system vs. inheritance or switch | ✓ Good — FormatterRegistry supports new formats |
| BatchProcessor composition pattern | Universal batch for all parsers vs. per-parser batch code | ✓ Good — 1 processor, 6 parsers |
| Rate limiting with burst=1 | Strict quota protection vs. burst allowance | ✓ Good — prevents API quota exhaustion |
| Confidence scoring per strategy tier | Audit trail for categorization decisions | ✓ Good — 1.0/0.95/0.90/0.8 per tier |

---
*Last updated: 2026-02-16 — v1.3 shipped*
