# camt-csv

## What This Is

A Go CLI tool that converts financial statement formats (CAMT.053 XML, PDF bank statements, Revolut CSV, Selma CSV) into standardized CSV with AI-powered transaction categorization. The codebase has been hardened in v1.1 to resolve all identified concerns — bugs, security, tech debt, architecture inconsistencies, test gaps, and safety features — establishing a clean foundation for future development.

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

### Active

(None yet — next milestone to be defined)

### Out of Scope

- Full PDF parser strategy pattern refactor — too large; minimal bug fixes only in v1.1
- New input format parsers — hardening milestone, not feature development
- UI/web interface — CLI-only tool
- Database backend — YAML file storage is sufficient for current scale
- Replacing pdftotext dependency — separate initiative requiring evaluation of Go PDF libraries

## Context

Shipped v1.1 Hardening with ~40,800 LOC Go across 21 modified files.
Tech stack: Go 1.24.2, Cobra 1.10.2, Viper 1.21.0, Logrus 1.9.4.
External dependency on `pdftotext` (Poppler utils) for PDF parsing.
Optional dependency on Google Gemini API for AI categorization.
Codebase map available at `.planning/codebase/` with 7 analysis documents.

Known technical debt:

- GetLogrusAdapter() creates new logger when mock injected (testing limitation)
- YAML store lacks concurrent access protection (single-threaded per command currently)
- PDF parser would benefit from strategy pattern refactor (deferred)

## Constraints

- **Tech stack**: Go — no language changes
- **Backwards compatibility**: CLI interface and YAML file formats must remain compatible
- **External tools**: pdftotext dependency stays (removal is a separate initiative)
- **Testing**: All changes must maintain or improve test coverage, no regressions
- **AI API**: Gemini integration stays, but add resilience patterns around it

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

---
*Last updated: 2026-02-01 after v1.1 milestone*
