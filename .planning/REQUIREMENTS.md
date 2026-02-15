# Requirements: camt-csv

**Defined:** 2026-02-15
**Core Value:** Reliable, maintainable financial data conversion with intelligent categorization.

## v1.2 Requirements

Requirements for v1.2 Full Polish milestone. Each maps to roadmap phases.

### Revolut Parser

- [ ] **REV-01**: Parser correctly identifies and tags all transaction types (Transfer, Card Payment, Exchange, Deposit, Fee, Charge, Card Refund, Charge Refund)
- [ ] **REV-02**: Parser outputs standardized 35-column CSV format (matching CAMT/PDF/Selma/Debit)
- [ ] **REV-03**: Exchange transactions preserve original currency and amount metadata
- [ ] **REV-04**: Product field (Current/Savings) is mapped to Transaction model
- [ ] **REV-05**: REVERTED/PENDING transactions are handled (skip or flag based on state)

### Revolut Investment Parser

- [ ] **RINV-01**: Parser handles SELL transactions
- [ ] **RINV-02**: Parser handles CUSTODY_FEE transactions
- [ ] **RINV-03**: Batch conversion support implemented

### Output Formatting (all parsers)

- [ ] **OUT-01**: `--format` flag selects output format (standard, icompta)
- [ ] **OUT-02**: iCompta format maps fields to ICTransaction columns (date, name, amount, payee, type, status)
- [ ] **OUT-03**: iCompta format includes category in output for ICTransactionSplit mapping
- [ ] **OUT-04**: Configurable date format in output

### AI Categorization Safety (all parsers)

- [ ] **AI-01**: `--auto-learn` flag controls whether AI categorizations are saved to YAML
- [ ] **AI-02**: Gemini API calls have rate limiting to respect quota
- [ ] **AI-03**: Gemini API calls have retry with backoff on failures

### Batch Processing

- [ ] **BATCH-01**: PDF parser supports batch conversion
- [ ] **BATCH-02**: Revolut Investment parser supports batch conversion
- [ ] **BATCH-03**: Batch failures report which files failed without stopping the entire run

## Future Requirements

Deferred to future milestones. Tracked but not in current roadmap.

### New Formats

- **FMT-01**: MT940 bank statement parsing
- **FMT-02**: OFX/QFX import support

### Advanced Features

- **ADV-01**: AI confidence scoring with threshold-based auto-learn
- **ADV-02**: Audit trail for AI categorization decisions
- **ADV-03**: Multi-file exchange transaction pairing (cross-currency matching)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full PDF parser strategy refactor | Deferred to v2 — too large for this milestone |
| Direct iCompta database writes | Too fragile — CSV import via iCompta's plugin is safer |
| UI/web interface | CLI-only tool |
| Database backend | YAML sufficient at current scale |
| Replacing pdftotext | Separate initiative |
| Exchange pair matching across files | Complex (±timing, ±rounding) — deferred to future |
| Real-time Revolut API integration | Out of scope — file-based conversion only |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| OUT-01 | Phase 5 | Pending |
| OUT-02 | Phase 5 | Pending |
| OUT-03 | Phase 5 | Pending |
| OUT-04 | Phase 5 | Pending |
| REV-01 | Phase 6 | Pending |
| REV-02 | Phase 6 | Pending |
| REV-03 | Phase 6 | Pending |
| REV-04 | Phase 6 | Pending |
| REV-05 | Phase 6 | Pending |
| RINV-01 | Phase 6 | Pending |
| RINV-02 | Phase 6 | Pending |
| RINV-03 | Phase 6 | Pending |
| BATCH-02 | Phase 6 | Pending |
| BATCH-01 | Phase 7 | Pending |
| BATCH-03 | Phase 7 | Pending |
| AI-01 | Phase 8 | Pending |
| AI-02 | Phase 8 | Pending |
| AI-03 | Phase 8 | Pending |

**Coverage:**

- v1.2 requirements: 18 total
- Mapped to phases: 18/18 ✓
- Unmapped: 0

**Note:** BATCH-02 (Revolut Investment batch support) and RINV-03 (Batch conversion support) cover the same capability and are both mapped to Phase 6.

---
*Requirements defined: 2026-02-15*
*Last updated: 2026-02-15 after roadmap creation*
