---
phase: 03-architecture-and-error-handling
plan: 01
subsystem: documentation
tags: [error-handling, conventions, patterns, logging]

# Dependency graph
requires:
  - phase: 01-critical-bugs-and-security
    provides: Context propagation and error logging patterns
  - phase: 02-configuration-and-state-cleanup
    provides: Container-based dependency injection
provides:
  - Comprehensive error handling patterns for three severity levels
  - Guidelines for fatal, retryable, and recoverable errors
  - init() function error handling best practices
  - Integration with custom error types from parsererror package
affects: [04-deprecation-and-cleanup, future-command-implementations]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Three-tier error severity: fatal, retryable, recoverable"
    - "init() error handling: prefer Cobra graceful handling over panic"
    - "Custom error types with Unwrap() for error chain inspection"

key-files:
  created: []
  modified:
    - ".planning/codebase/CONVENTIONS.md"

key-decisions:
  - "Fatal errors exit immediately (config, container, permissions, format)"
  - "Retryable errors log and degrade gracefully (network, API, rate limits)"
  - "Recoverable errors log and continue (single tx failures, cleanup, optional features)"
  - "init() should not panic - let Cobra handle MarkFlagRequired errors"
  - "Error messages must include context (file path, party name, index)"

patterns-established:
  - "Error severity levels guide exit vs retry vs continue decisions"
  - "Future enhancement: exponential backoff retry logic for network errors"
  - "Custom error types integrate with severity levels via wrapping"

# Metrics
duration: 1.4min
completed: 2026-02-01
---

# Phase 3 Plan 01: Error Handling Patterns Documentation

**Comprehensive error handling patterns defining when to exit, retry, or continue with degraded functionality across all CLI commands**

## Performance

- **Duration:** 1.4 min (82 seconds)
- **Started:** 2026-02-01T19:38:42Z
- **Completed:** 2026-02-01T19:40:04Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- Defined three error severity levels with clear usage criteria
- Documented fatal errors (exit immediately) for config, container, permissions failures
- Documented retryable errors (log and degrade) for network, API, rate limit issues
- Documented recoverable errors (log and continue) for single tx failures, cleanup, optional features
- Added concrete code examples for each severity level
- Documented init() function error handling best practices (avoid panic, prefer Cobra)
- Integrated custom error types from parsererror package

## Task Commits

Each task was completed in a single comprehensive commit:

1. **Tasks 1-2: Define error handling hierarchy and document pattern examples** - `fe5c661` (docs)

**Note:** Both tasks completed together as they formed a cohesive documentation section.

## Files Created/Modified

- `.planning/codebase/CONVENTIONS.md` - Added comprehensive Error Handling Patterns section (208 lines)

## Decisions Made

**1. Fatal errors exit immediately**
- Use when application cannot continue: config failures, container init failures, required flags missing, file permission errors, invalid format
- Log level: Fatal (exits with non-zero status)
- Error messages must include specific context and actionable guidance

**2. Retryable errors log and continue with degraded functionality**
- Use for transient failures: network errors, temporary file issues, rate limiting
- Log level: Warn
- Future enhancement: implement exponential backoff retry logic

**3. Recoverable errors log and continue**
- Use for non-critical failures: single transaction parsing, optional categorization, cleanup, metadata extraction
- Log level: Warn (user awareness) or Debug (troubleshooting)
- Continue processing with fallback or skip

**4. init() should not panic**
- Avoid: `panic(err)` causes immediate crash with poor context
- Prefer: Let Cobra handle MarkFlagRequired errors gracefully
- Rationale: Cobra shows clear usage message, MarkFlagRequired errors are programmer errors caught in tests

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Error handling patterns documented and ready for implementation
- Command handlers in Plan 02 can now follow consistent patterns
- Foundation established for predictable error behavior across all CLI commands
- Three severity levels provide clear guidance: exit vs retry vs continue

**Ready for:** Plan 02 (command handler implementations with consistent error patterns)

**Blockers:** None

**Concerns:** None - documentation is comprehensive and actionable

---
*Phase: 03-architecture-and-error-handling*
*Completed: 2026-02-01*
