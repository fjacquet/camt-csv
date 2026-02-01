# Roadmap: camt-csv Codebase Hardening

## Overview

This milestone transforms camt-csv from a feature-complete but fragile tool into a hardened, maintainable foundation for future development. We address all identified concerns across four phases: first resolving critical bugs and security issues that directly impact users, then cleaning up deprecated configuration and global state, next standardizing error handling architecture, and finally closing test coverage gaps and adding safety features. Each phase delivers observable improvements in reliability and code quality.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3, 4): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Critical Bugs & Security** - Fix user-impacting bugs and security vulnerabilities
- [ ] **Phase 2: Configuration & State Cleanup** - Remove deprecated config and global state
- [ ] **Phase 3: Architecture & Error Handling** - Standardize error patterns and cleanup temp files
- [ ] **Phase 4: Test Coverage & Safety** - Close coverage gaps and add data protection

## Phase Details

### Phase 1: Critical Bugs & Security
**Goal**: Users experience no data leaks, predictable behavior, and proper error handling in PDF parsing

**Depends on**: Nothing (first phase)

**Requirements**: BUG-01, BUG-02, BUG-03, BUG-04, SEC-01, SEC-02, SEC-03

**Success Criteria** (what must be TRUE):
  1. No debug files accumulate in working directory after PDF parsing
  2. All temporary files have unpredictable random names and are cleaned up properly
  3. API credentials never appear in any log output at any level
  4. File permissions are appropriate for content type (0644 for non-secrets, 0600 for credentials)
  5. Context cancellation properly propagates through PDF parsing operations

**Plans**: 3 plans in 1 wave

Plans:
- [x] 01-01-PLAN.md — PDF parser bug fixes (debug file, context propagation, temp cleanup)
- [x] 01-02-PLAN.md — MockLogger state isolation for test verification
- [x] 01-03-PLAN.md — Security hardening (credential logging, temp file naming, permissions)

### Phase 2: Configuration & State Cleanup
**Goal**: All configuration flows through DI container with no global state or deprecated functions

**Depends on**: Phase 1

**Requirements**: DEBT-01, ARCH-02, DEBT-02

**Success Criteria** (what must be TRUE):
  1. No deprecated config functions exist in codebase (MustGetEnv, LoadEnv, GetEnv removed)
  2. No global mutable state exists in config package (globalConfig, Logger removed)
  3. Container initialization failures propagate as errors instead of silent fallback creation

**Plans**: 1 plan in 1 wave

Plans:
- [ ] 02-01-PLAN.md — Remove deprecated config functions, global state, and fallback categorizer creation

### Phase 3: Architecture & Error Handling
**Goal**: Error handling is consistent, predictable, and never panics unexpectedly

**Depends on**: Phase 2

**Requirements**: ARCH-01, ARCH-03, DEBT-03

**Success Criteria** (what must be TRUE):
  1. Error handling follows documented patterns across all commands (exit vs retry vs continue)
  2. CLI initialization failures produce clear user-facing error messages instead of panics
  3. PDF parser uses single temp file path instead of creating multiple temp files

**Plans**: TBD

Plans:
- [ ] 03-01: TBD

### Phase 4: Test Coverage & Safety
**Goal**: Test suite thoroughly validates edge cases and safety features protect user data

**Depends on**: Phase 3

**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, SAFE-01

**Success Criteria** (what must be TRUE):
  1. Tests verify correct error behavior when container is nil (not just "doesn't panic")
  2. Concurrent processing tests cover race conditions and context cancellation mid-processing
  3. PDF format detection tests cover ambiguous cases (partial markers, markers in descriptions)
  4. Error messages include file path and field context for all parsers
  5. Category mapping YAML files are backed up with timestamps before auto-learning overwrites

**Plans**: TBD

Plans:
- [ ] 04-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Critical Bugs & Security | 3/3 | Complete | 2026-02-01 |
| 2. Configuration & State Cleanup | 0/1 | Not started | - |
| 3. Architecture & Error Handling | 0/? | Not started | - |
| 4. Test Coverage & Safety | 0/? | Not started | - |

---
*Roadmap created: 2026-02-01*
*Last updated: 2026-02-01 after phase 2 planning*
