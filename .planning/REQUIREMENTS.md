# Requirements: camt-csv

**Defined:** 2026-02-23
**Core Value:** Reliable, maintainable financial data conversion with intelligent categorization.

## v1.4 Requirements

Requirements for the Simplify milestone. Each maps to roadmap phases.

### Input Handling

- [x] **INPUT-01**: User can pass a file path or a folder path to any parser command (camt, debit, revolut, revolut-investment, selma, pdf)
- [x] **INPUT-02**: When input is a file, the command processes that single file (unchanged single-file behavior)
- [x] **INPUT-03**: When input is a folder, the command processes all matching files in that folder (non-recursive, file extension filtered per parser)
- [x] **INPUT-04**: When input is a folder, `--output` flag is required; command exits with a clear error if omitted
- [x] **INPUT-05**: camt, debit, revolut, revolut-investment, selma: folder mode outputs one CSV per input file to the `--output` directory
- [x] **INPUT-06**: pdf: folder mode consolidates all PDFs in the folder into a single CSV in the `--output` directory

### Batch Removal

- [ ] **BATCH-01**: `batch` subcommand is removed from the CLI entirely
- [ ] **BATCH-02**: `--batch` flag is removed from all parser commands

### Format Default

- [ ] **FORMAT-01**: Default output format is `icompta` (no `--format` flag required for typical iCompta use)
- [ ] **FORMAT-02**: `--format` flag remains available to override (e.g., `--format standard` for 29-column CSV)

## Future Requirements

### Potential v1.5+

- Cross-file exchange transaction pairing (complex timing/rounding matching)
- Recursive folder processing (not needed for current use case)
- MT940/OFX/QIF format support

## Out of Scope

| Feature | Reason |
|---------|--------|
| Recursive folder scanning | Not needed — all files are in flat directories per use case |
| Merging non-PDF formats | Only PDF consolidation makes sense (one statement per file for others) |
| Auto-discover output path | Explicit `--output` required for clarity and safety |
| Removing --format flag entirely | User may still need standard format for other tools |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| INPUT-01 | Phase 12 | Complete |
| INPUT-02 | Phase 12 | Complete |
| INPUT-03 | Phase 12 | Complete |
| INPUT-04 | Phase 12 | Complete |
| INPUT-05 | Phase 12 | Complete |
| INPUT-06 | Phase 12 | Complete |
| BATCH-01 | Phase 13 | Pending |
| BATCH-02 | Phase 13 | Pending |
| FORMAT-01 | Phase 13 | Pending |
| FORMAT-02 | Phase 13 | Pending |

**Coverage:**
- v1.4 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-23*
*Last updated: 2026-02-23 after initial definition*
