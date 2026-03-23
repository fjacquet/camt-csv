---
phase: 16-multi-llm-provider
plan: 01
subsystem: config
tags: [viper, config, ai, provider, openrouter, gemini, env-vars]

# Dependency graph
requires: []
provides:
  - Config.AI.Provider field (default "gemini", accepts "openrouter")
  - Config.AI.BaseURL field (default empty string for client-level URL defaults)
  - CAMT_AI_API_KEY primary env var with GEMINI_API_KEY backward-compat fallback
  - Provider validation rejects unknown values with clear error
  - Model empty-string validation when AI is enabled
  - Updated config.yaml documenting new fields
affects: [16-02-openrouter-client, 16-03-container-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Unified CAMT_AI_API_KEY env var with named fallback to GEMINI_API_KEY
    - Provider enum validation in validateConfig (extensible: add to validProviders map)

key-files:
  created: []
  modified:
    - internal/config/viper.go
    - internal/config/viper_test.go
    - internal/container/container_test.go
    - .camt-csv/config.yaml
    - CHANGELOG.md

key-decisions:
  - "CAMT_AI_API_KEY is the new unified env var; GEMINI_API_KEY kept as fallback for backward compatibility"
  - "Provider validation done at config load time, not at client construction"
  - "BaseURL defaults to empty string — each client uses its own hardcoded URL when not overridden"

patterns-established:
  - "Provider field validation: map[string]bool{'gemini': true, 'openrouter': true} in validateConfig"

requirements-completed: [CONF-01, CONF-02, CONF-03, CONF-04, PROV-01, PROV-03]

# Metrics
duration: 6min
completed: 2026-03-23
---

# Phase 16 Plan 01: Multi-LLM Config Foundation Summary

**Provider-agnostic Config.AI struct with CAMT_AI_API_KEY env var, provider/base_url fields, and strict startup validation for gemini|openrouter selection**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-23T06:26:00Z
- **Completed:** 2026-03-23T06:32:12Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Extended Config.AI struct with Provider (default "gemini") and BaseURL (default "") fields
- Replaced GEMINI_API_KEY-only binding with CAMT_AI_API_KEY primary + GEMINI_API_KEY fallback
- Added provider validation (gemini|openrouter), model non-empty check, and updated API key error message
- Updated .camt-csv/config.yaml to document all new AI config fields for users
- All 3049 tests pass with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend Config.AI struct with Provider and BaseURL** - `d6773f4` (feat, TDD)
2. **Task 2: Update config.yaml with provider and base_url fields** - `b1a92a0` (feat)

## Files Created/Modified

- `internal/config/viper.go` - Added Provider/BaseURL to AI struct, defaults, env binding, validation
- `internal/config/viper_test.go` - Added tests for defaults, CAMT_AI_API_KEY, GEMINI_API_KEY fallback, provider/model validation
- `internal/container/container_test.go` - Updated inline AI struct literals to match new struct shape
- `.camt-csv/config.yaml` - Added provider: gemini and base_url: '' to ai: section
- `CHANGELOG.md` - Documented new config fields under [Unreleased]

## Decisions Made

- CAMT_AI_API_KEY is the unified env var going forward; GEMINI_API_KEY fallback ensures zero breaking change for existing users
- Provider validation is strict at startup (not lazy at first API call) — fail fast principle
- BaseURL is empty by default, meaning each client uses its own hardcoded URL when not configured

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated container_test.go inline AI struct literals**
- **Found during:** Task 1 (running full test suite post-implementation)
- **Issue:** `internal/container/container_test.go` used inline anonymous struct literals for config.Config.AI that no longer matched the extended struct shape (missing Provider and BaseURL fields)
- **Fix:** Added Provider string and BaseURL string fields to all 5 inline AI struct literals in container_test.go
- **Files modified:** internal/container/container_test.go
- **Verification:** go test ./... passes with 3049 tests
- **Committed in:** d6773f4 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - struct shape update in dependent test file)
**Impact on plan:** Necessary correctness fix caused by struct shape change. No scope creep.

## Issues Encountered

None - the inline struct pattern in container_test.go is expected when the Config struct is an anonymous struct (not a named type), requiring all usages to be updated in lock-step.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Config.AI.Provider and Config.AI.BaseURL fields are ready for container wiring (Plan 16-03)
- CAMT_AI_API_KEY env var is bound and ready for OpenRouterClient construction
- Plan 16-02 (OpenRouterClient) can proceed independently

---
*Phase: 16-multi-llm-provider*
*Completed: 2026-03-23*
