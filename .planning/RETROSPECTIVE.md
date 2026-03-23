# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.6 — Multi-LLM Provider

**Shipped:** 2026-03-23
**Phases:** 1 | **Plans:** 3 | **Sessions:** 2

### What Was Built
- OpenRouterClient implementing AIClient interface with OpenAI-compatible chat/completions API
- Provider-agnostic config (`ai.provider`, `ai.base_url`, `CAMT_AI_API_KEY`)
- Split chat/embedding client architecture with graceful semantic tier degradation
- ADR-018 documenting the multi-LLM architecture decision

### What Worked
- Wave-based parallel execution: Plans 16-01 and 16-02 ran in parallel (independent config + client work)
- Existing AIClient interface abstraction made adding OpenRouterClient straightforward — no interface changes needed
- Fail-fast config validation catches errors at startup, not at API call time
- Semgrep hook caught CWE-338 (math/rand) during development — security linting as guardrail

### What Was Inefficient
- Context window reset mid-phase required session continuation (summary-based handoff)
- 16-03 SUMMARY used wrong frontmatter key (`requirements_satisfied` vs `requirements_completed`) — tooling mismatch
- Prompt/cleanCategory code duplicated between GeminiClient and OpenRouterClient instead of extracted to shared helper

### Patterns Established
- Constructor-injected API keys (no os.Getenv in client constructors) — cleaner testing, multi-key support
- SetEmbeddingClient setter pattern for post-construction dependency wiring (matches SetStagingStore)
- Time-based jitter for retry backoff to satisfy security linters

### Key Lessons
1. When adding a second implementation of an interface, the interface abstraction quality becomes immediately visible — AIClient was well-designed
2. Split client architecture (chat vs embedding) is a good pattern when providers have asymmetric capabilities
3. Backward-compatible env var fallbacks (GEMINI_API_KEY -> CAMT_AI_API_KEY) avoid breaking existing setups

### Cost Observations
- Model mix: budget profile (haiku for researchers, sonnet for planning)
- Sessions: 2 (1 for planning + Wave 1 execution, 1 for Wave 2 + verification + completion)
- Notable: Wave-based parallelism saved time on plans 01+02; plan 03 was sequential (dependency)

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.6 | 2 | 1 | Wave-based parallel execution, budget model profile |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.6 | 3,049 | - | 1 (OpenRouterClient uses net/http only) |

### Top Lessons (Verified Across Milestones)

1. Well-designed interfaces make extension straightforward — invest in interface design upfront
2. Fail-fast validation at boundaries (config load, constructor) prevents cascading errors
