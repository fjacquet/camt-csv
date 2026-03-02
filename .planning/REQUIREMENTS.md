# Requirements: camt-csv

**Defined:** 2026-03-02
**Core Value:** Reliable, maintainable financial data conversion with intelligent categorization.

## v1 Requirements (v1.5 Milestone)

Requirements for Jumpsoft Money CSV export support.

### Format

- [x] **FMT-01**: `--format jumpsoft` option available on all parser commands
- [x] **FMT-02**: JumpsoftFormatter outputs comma-delimited CSV with header row
- [x] **FMT-03**: Date field uses `YYYY-MM-DD` format (configurable via `--date-format` flag)
- [x] **FMT-04**: Amount field is signed (negative for debits, positive for credits)
- [x] **FMT-05**: Category field populated from categorizer output

### Integration

- [x] **INT-01**: JumpsoftFormatter registered in FormatterRegistry as `"jumpsoft"`
- [x] **INT-02**: `--format jumpsoft` works in single-file mode across all 6 parsers (camt, pdf, revolut, selma, debit, revolut-investment)
- [x] **INT-03**: `--format jumpsoft` works in folder mode across all parsers
- [x] **INT-04**: `--format jumpsoft` documented in CLI help output

### Testing

- [ ] **TEST-01**: Unit tests for JumpsoftFormatter field mapping and CSV output structure
- [ ] **TEST-02**: Integration test verifying end-to-end output with at least one parser

## v2 Requirements

### Advanced Jumpsoft Features

- **ADV-01**: Configurable column selection for Jumpsoft Money CSV (user picks which columns to include)
- **ADV-02**: Jumpsoft Money import plugin pre-configuration documentation

## Out of Scope

| Feature | Reason |
|---------|--------|
| Auto-detection of Jumpsoft Money account structure | No programmatic API; user configures import in app |
| Jumpsoft Money direct database import | No documented API for direct DB access |
| New parsers (MT940, OFX, QIF) | Separate future milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FMT-01 | Phase 14 | Complete |
| FMT-02 | Phase 14 | Complete |
| FMT-03 | Phase 14 | Complete |
| FMT-04 | Phase 14 | Complete |
| FMT-05 | Phase 14 | Complete |
| INT-01 | Phase 14 | Complete |
| INT-02 | Phase 14 | Complete |
| INT-03 | Phase 14 | Complete |
| INT-04 | Phase 14 | Complete |
| TEST-01 | Phase 15 | Pending |
| TEST-02 | Phase 15 | Pending |

**Coverage:**
- v1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0

---
*Requirements defined: 2026-03-02*
*Last updated: 2026-03-02 — traceability finalized after roadmap creation (TEST-01/02 moved to Phase 15)*
