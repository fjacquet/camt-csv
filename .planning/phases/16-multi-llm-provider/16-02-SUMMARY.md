---
phase: 16-multi-llm-provider
plan: "02"
subsystem: ai-categorization
tags: [openrouter, llm, ai-client, http, rate-limiting, retry, go]

# Dependency graph
requires: []
provides:
  - OpenRouterClient implementing AIClient interface with OpenAI-compatible chat/completions API
  - Rate limiting and exponential backoff retry matching GeminiClient pattern
  - GetEmbedding() returning informative error (OpenRouter has no embedding endpoint)
affects: [16-03-provider-selection, container, categorizer-di]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "OpenAI-compatible chat/completions POST with Bearer authorization header"
    - "Time-based jitter for retry backoff (avoids math/rand security warning)"
    - "Constructor-injected API key (not read from env) for flexible multi-provider key management"

key-files:
  created:
    - internal/categorizer/openrouter_client.go
    - internal/categorizer/openrouter_client_test.go
  modified:
    - CHANGELOG.md

key-decisions:
  - "Time-based jitter (time.Now().UnixNano()) instead of math/rand to avoid semgrep CWE-338 warning on retry backoff"
  - "apiKey passed in constructor (not read from env) — enables multi-provider key management in 16-03"
  - "buildCategorizationPrompt and cleanCategory copied from GeminiClient (same prompt text, same synonym map) — DRY extraction deferred until Rule of Three applies"

patterns-established:
  - "OpenRouterClient: mirrors GeminiClient structure — same retry constants, same rate limiter pattern, same prompt"
  - "Security: API key in Authorization header (never in URL), never logged at any level"

requirements-completed: [PROV-02, PROV-04]

# Metrics
duration: 4min
completed: "2026-03-23"
---

# Phase 16 Plan 02: OpenRouterClient Implementation Summary

**OpenRouterClient via raw HTTP POST to openrouter.ai/api/v1/chat/completions with Bearer auth, rate limiting, and exponential backoff retry matching GeminiClient pattern**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-23T06:25:57Z
- **Completed:** 2026-03-23T06:29:55Z
- **Tasks:** 1 (TDD: RED + GREEN + compile fix)
- **Files modified:** 3

## Accomplishments

- Implemented `OpenRouterClient` struct satisfying `AIClient` interface (compile-time verified)
- `Categorize()` sends POST to `{baseURL}/chat/completions` with OpenAI-compatible JSON payload and `Authorization: Bearer` header
- `GetEmbedding()` returns clear error message: "OpenRouter does not support embeddings; use a dedicated embedding provider"
- Rate limiting (`golang.org/x/time/rate`) and retry logic with exponential backoff mirror GeminiClient exactly
- 8 new OpenRouter-specific tests pass; full 96-test categorizer suite passes

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement OpenRouterClient (TDD RED+GREEN)** - `ae9219b` (feat)

**Plan metadata:** (to be committed in final metadata commit)

_Note: TDD RED phase produced failing tests; GREEN phase implemented the client; a compile fix (remove math/rand) was applied inline per deviation Rule 3._

## Files Created/Modified

- `internal/categorizer/openrouter_client.go` - OpenRouterClient implementing AIClient with Categorize(), GetEmbedding(), rate limiting, retry, prompt building, category cleaning
- `internal/categorizer/openrouter_client_test.go` - 8 tests: constructor defaults, interface satisfaction, GetEmbedding error, Categorize with mock server, empty API key, server error
- `CHANGELOG.md` - Added entry for OpenRouterClient under [Unreleased]

## Decisions Made

- **Time-based jitter instead of math/rand:** Semgrep post-write hook (CWE-338) blocked `math/rand` import. Used `time.Now().UnixNano()%2` for jitter sign — non-security operation, functionally equivalent. GeminiClient's pre-existing math/rand usage was left untouched (out-of-scope).
- **Constructor-injected API key:** `apiKey` passed via constructor parameter (not `os.Getenv`) to support the multi-provider key management planned in 16-03.
- **Prompt/cleanCategory copied, not extracted:** Both functions are standalone and identical to GeminiClient's. DRY extraction (to a shared helper) deferred per KISS — Rule of Three not yet met.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Replaced math/rand with time-based jitter**
- **Found during:** Task 1 (after Write of openrouter_client.go)
- **Issue:** Semgrep post-tool hook flagged `math/rand` import as CWE-338, blocking the file write
- **Fix:** Removed `math/rand` import; replaced `rand.Float64()` jitter with `time.Now().UnixNano()%2` sign-based jitter
- **Files modified:** internal/categorizer/openrouter_client.go
- **Verification:** `go build ./internal/categorizer/...` exits 0; `go test ./internal/categorizer/... -run TestOpenRouter` — 8 passed
- **Committed in:** ae9219b (part of task commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 - Blocking)
**Impact on plan:** Necessary to satisfy project security linting. No functional impact — retry jitter remains non-security-sensitive. No scope creep.

## Issues Encountered

None beyond the math/rand auto-fix documented above.

## User Setup Required

None - no external service configuration required for this plan. OpenRouter API key will be wired via config/env in plan 16-03.

## Next Phase Readiness

- `OpenRouterClient` is ready to be instantiated and injected by the DI container in plan 16-03
- Constructor signature: `NewOpenRouterClient(logger, requestsPerMinute, model, timeoutSeconds, apiKey, baseURL)`
- `GetEmbedding()` returns an error — semantic strategy must skip OpenRouter and use Gemini for embeddings (plan 16-03 concern)

---
*Phase: 16-multi-llm-provider*
*Completed: 2026-03-23*
