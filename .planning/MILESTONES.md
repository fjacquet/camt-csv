# Project Milestones: camt-csv

## v1.1 Hardening (Shipped: 2026-02-01)

**Delivered:** Resolved all 18 identified codebase concerns across bugs, security, tech debt, architecture, test coverage, and safety features — establishing a clean, maintainable foundation for future development.

**Phases completed:** 1-4 (11 plans total)

**Key accomplishments:**

- Fixed all PDF parser bugs: debug file leak, context propagation loss, double temp file creation, cleanup ordering
- Hardened security: no credential logging, random temp file names, standardized file permissions
- Removed all deprecated config functions and global mutable state; DI container is sole config pathway
- Standardized error handling with documented three-tier pattern (fatal/retryable/recoverable)
- Added comprehensive test coverage: concurrent processing edge cases, PDF format detection, error message validation
- Implemented category YAML backup system with timestamped copies before auto-learn overwrites

**Stats:**

- 21 Go files modified, 59 files total
- 8,494 lines added, 399 removed
- 4 phases, 11 plans, 49 commits
- 1 day (2026-02-01)

**Git range:** `bc674cb` (codebase map) → `0b4b63b` (phase 4 complete)

**What's next:** Future development on hardened foundation — resilience patterns, performance tuning, or new features.

---

## v1.2 Full Polish (Shipped: 2026-02-16)

**Delivered:** Overhauled Revolut parsers with transaction-type intelligence, standardized CSV output across all parsers for iCompta import compatibility, added AI auto-learn safety controls, and built universal batch processing infrastructure with formatter integration.

**Phases completed:** 5-9 (14 plans total)

**Key accomplishments:**

- OutputFormatter plugin system with strategy pattern — StandardFormatter (35-col CSV) and iComptaFormatter (10-col semicolon) for iCompta import compatibility
- Revolut parser intelligence — all 8 transaction types, Product field routing (Current/Savings), exchange metadata preservation, 35-column standardized output
- Revolut Investment parser completions — SELL and CUSTODY_FEE handlers, batch conversion support
- BatchProcessor universal infrastructure — manifest generation, semantic exit codes (0/1/2), error isolation per file
- AI safety controls — rate limiting (burst=1), exponential backoff retry, confidence scoring per strategy, --auto-learn flag (default OFF)
- End-to-end formatter integration — --format flag works in all modes (single file, batch, consolidation) across all 6 parsers

**Stats:**

- 132 files modified
- 16,423 lines added, 177 removed
- 5 phases, 14 plans, 78 commits
- 1 day (2026-02-15 → 2026-02-16)
- Total codebase: 43,619 LOC Go

**Git range:** `v1.1` → `v1.2`

**Audit:** 18/18 requirements satisfied, 5/5 phases verified, 6/6 integration points, 5/5 E2E flows

**What's next:** Future development — cross-file exchange pairing, new format parsers (MT940, OFX), or UI improvements.

---

