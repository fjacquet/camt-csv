# camt-csv: Codebase Hardening

## What This Is

A Go CLI tool that converts financial statement formats (CAMT.053 XML, PDF bank statements, Revolut CSV, Selma CSV) into standardized CSV with AI-powered transaction categorization. This milestone focuses on hardening the existing codebase by addressing all identified concerns — bugs, security issues, tech debt, architecture inconsistencies, test gaps, and missing safety features — to establish a clean foundation for future feature development.

## Core Value

Every identified codebase concern is resolved, making the tool reliable and maintainable enough to confidently build new features on top of.

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

### Active

- [ ] Fix all known bugs (debug file accumulation, MockLogger verification, context loss)
- [ ] Resolve all security concerns (credential logging, predictable temp files, file permissions)
- [ ] Clean up deprecated configuration system (remove legacy env functions, global state)
- [ ] Remove fallback categorizer creation that bypasses DI
- [ ] Fix MockLogger state management for proper test verification
- [ ] Fix PDF parser bugs (debug file, double temp files, context loss) without full refactor
- [ ] Standardize error handling patterns across codebase
- [ ] Add file-level locking for concurrent YAML store access
- [ ] Replace panic in CLI init with proper error handling
- [ ] Fix temp file cleanup ordering in PDF parser
- [ ] Close test coverage gaps (nil container, concurrency edge cases, PDF format detection, error wrapping)
- [ ] Add dry-run mode for categorization preview
- [ ] Add batch error recovery with failure reporting
- [ ] Add category mapping backup before auto-learning overwrites

### Out of Scope

- Full PDF parser strategy pattern refactor — too large for this milestone, minimal bug fixes only
- New input format parsers — this milestone is about hardening, not new features
- UI/web interface — CLI-only tool
- Database backend — YAML file storage is sufficient for current scale
- Raising concurrent processing threshold — needs benchmarking, defer to performance milestone

## Context

- Brownfield Go project with ~27 internal packages, mature CLI structure
- Go 1.24.2, Cobra 1.10.2, Viper 1.21.0, Logrus 1.9.4
- External dependency on `pdftotext` (Poppler utils) for PDF parsing
- Optional dependency on Google Gemini API for AI categorization
- Codebase map available at `.planning/codebase/` with 7 analysis documents
- CONCERNS.md identified 26 items across 8 categories
- Some deprecated config code marked for removal in v3.0.0 still present
- Category store YAML files lack concurrent access protection
- Error handling is inconsistent: mix of return error, log.Fatal, log.Warn-and-continue

## Constraints

- **Tech stack**: Go — no language changes
- **Backwards compatibility**: CLI interface and YAML file formats must remain compatible
- **External tools**: pdftotext dependency stays (removal is a separate initiative)
- **Testing**: All changes must maintain or improve test coverage, no regressions
- **AI API**: Gemini integration stays, but add resilience patterns around it

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Minimal PDF parser fixes only | Full strategy pattern refactor too large; fix bugs without restructuring | — Pending |
| Bugs & security first priority | Highest user impact, reduces risk before architectural changes | — Pending |
| Include safety features (dry-run, backup, error recovery) | Essential for reliability before adding new capabilities | — Pending |
| Keep pdftotext dependency | Replacing external tool is a separate initiative | — Pending |

---
*Last updated: 2026-02-01 after initialization*
