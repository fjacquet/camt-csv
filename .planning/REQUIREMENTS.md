# Requirements: camt-csv Codebase Hardening

**Defined:** 2026-02-01
**Core Value:** Every identified codebase concern is resolved, making the tool reliable and maintainable enough to confidently build new features on top of.

## v1 Requirements

Requirements for this hardening milestone. Grouped by concern category.

### Bug Fixes

- [x] **BUG-01**: PDF parser stops writing `debug_pdf_extract.txt` to current working directory; debug output uses OS temp directory or is removed entirely
- [x] **BUG-02**: MockLogger properly isolates state so `WithError()` and `WithFields()` create independent instances; tests can verify specific log messages were emitted at correct levels
- [x] **BUG-03**: `ParseWithExtractor()` uses the passed context instead of discarding it for `context.Background()`
- [x] **BUG-04**: PDF parser temp file cleanup uses single defer block with correct close-then-remove ordering

### Security

- [x] **SEC-01**: No API credentials (GEMINI_API_KEY or its value) appear in any log output at any level; only log presence/absence of key
- [x] **SEC-02**: All temporary files use `os.CreateTemp()` with random naming; no predictable temp file paths in any parser
- [x] **SEC-03**: File permissions use 0644 for non-secret files (CSV output, debug) and 0600 only for files containing credentials or sensitive data

### Tech Debt

- [x] **DEBT-01**: All deprecated config functions (`MustGetEnv()`, `LoadEnv()`, `GetEnv()`) removed; all configuration flows through Viper/Container
- [x] **DEBT-02**: Fallback categorizer creation in `PersistentPostRun` removed; if container is nil, error propagates instead of silently creating unmanaged objects
- [x] **DEBT-03**: PDF parser double temp file creation consolidated into single temp file handling path

### Architecture

- [x] **ARCH-01**: Error handling follows documented pattern: define which errors exit immediately vs retry vs log-and-continue; all command handlers follow the pattern consistently
- [x] **ARCH-02**: Global mutable state (`globalConfig`, `Logger` variables) removed from `internal/config/config.go`; all state flows through DI container
- [x] **ARCH-03**: `panic(err)` in `cmd/categorize/categorize.go` `init()` replaced with graceful error handling that produces clear user-facing error message

### Test Coverage

- [ ] **TEST-01**: Commands verify correct behavior when container is nil (not just "doesn't panic" but validates error output)
- [ ] **TEST-02**: Concurrent processor tests cover race conditions, context cancellation mid-processing, and partial result handling
- [ ] **TEST-03**: PDF format detection tests cover edge cases: partial Viseca markers, markers in transaction descriptions, ambiguous formats
- [ ] **TEST-04**: Error wrapping tests verify error chain depth and message clarity for each parser; user-facing errors include file path and field context

### Safety Features

- [ ] **SAFE-01**: Category YAML files are backed up (timestamped copy) before auto-learn overwrites; backup location is configurable or defaults to same directory

## v2 Requirements

Deferred to future milestones. Tracked but not in current roadmap.

### Safety Features

- **SAFE-02**: Dry-run mode for categorization that previews changes without saving to YAML
- **SAFE-03**: Batch error recovery with per-file failure reporting, retry option, and summary of skipped files

### Resilience

- **RESIL-01**: Circuit breaker pattern for Gemini API calls with exponential backoff
- **RESIL-02**: Helpful error message when pdftotext is not installed (detect and guide user)

### Performance

- **PERF-01**: Concurrent processing threshold tuned via benchmarking (currently 100, likely better at 500+)
- **PERF-02**: Result channel streaming for large files instead of full buffering

### Refactoring

- **REFAC-01**: PDF parser refactored into strategy pattern with separate implementations per format
- **REFAC-02**: YAML store concurrent access protection via file-level locking (sync.RWMutex or atomic writes)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full PDF strategy pattern refactor | Too large for this milestone; minimal bug fixes only |
| New input format parsers | Hardening milestone, not feature development |
| YAML store file locking | Deferred to v2; current usage pattern is single-threaded per command |
| Dry-run mode | Deferred to v2; backup provides sufficient safety net |
| Batch error recovery | Deferred to v2; current skip behavior is acceptable for now |
| Replacing pdftotext dependency | Separate initiative requiring evaluation of Go PDF libraries |
| Concurrent threshold tuning | Needs dedicated benchmarking effort |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| BUG-01 | Phase 1 | Complete |
| BUG-02 | Phase 1 | Complete |
| BUG-03 | Phase 1 | Complete |
| BUG-04 | Phase 1 | Complete |
| SEC-01 | Phase 1 | Complete |
| SEC-02 | Phase 1 | Complete |
| SEC-03 | Phase 1 | Complete |
| DEBT-01 | Phase 2 | Complete |
| DEBT-02 | Phase 2 | Complete |
| ARCH-02 | Phase 2 | Complete |
| ARCH-01 | Phase 3 | Complete |
| ARCH-03 | Phase 3 | Complete |
| DEBT-03 | Phase 3 | Complete |
| TEST-01 | Phase 4 | Pending |
| TEST-02 | Phase 4 | Pending |
| TEST-03 | Phase 4 | Pending |
| TEST-04 | Phase 4 | Pending |
| SAFE-01 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 18 total
- Mapped to phases: 18
- Unmapped: 0

---
*Requirements defined: 2026-02-01*
*Last updated: 2026-02-01 after phase 3 completion*
