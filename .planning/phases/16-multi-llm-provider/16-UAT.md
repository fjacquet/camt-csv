---
status: complete
phase: 16-multi-llm-provider
source: [16-01-SUMMARY.md, 16-02-SUMMARY.md, 16-03-SUMMARY.md]
started: 2026-03-23T12:00:00Z
updated: 2026-03-23T12:30:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Default provider is gemini (backward compat)
expected: Run with only GEMINI_API_KEY set. App starts without errors, provider defaults to "gemini".
result: pass

### 2. Config validation rejects invalid provider
expected: Set `ai.provider: invalid` in config.yaml, run any command. App exits with clear error message mentioning valid providers (gemini, openrouter).
result: pass

### 3. Config validation rejects empty model when AI enabled
expected: Set `ai.model: ""` with `ai.enabled: true` in config. App exits with error about missing model name.
result: pass

### 4. CAMT_AI_API_KEY overrides GEMINI_API_KEY
expected: Set both env vars with different values. Run with `--log-level debug`. The CAMT_AI_API_KEY value is used (verify via debug log showing "AI provider: gemini" without API key warnings).
result: pass

### 5. OpenRouter provider selection
expected: Set `ai.provider: openrouter` and `CAMT_AI_API_KEY=<your-openrouter-key>`. Run a CAMT conversion with `--ai-enabled`. Log output shows "AI provider: openrouter" at startup.
result: pass

### 6. Semantic tier skips without GEMINI_API_KEY
expected: Set `ai.provider: openrouter` with only CAMT_AI_API_KEY (no GEMINI_API_KEY). Log output shows "Semantic tier: skipped (no embedding provider available)".
result: pass

### 7. Semantic tier active with both keys
expected: Set `ai.provider: openrouter` with CAMT_AI_API_KEY for OpenRouter AND GEMINI_API_KEY for embeddings. Log output shows "Semantic tier: active (Gemini embeddings)".
result: pass

### 8. OpenRouter categorization works end-to-end
expected: With openrouter provider configured and a valid API key, run a small CAMT file conversion with `--ai-enabled`. Transactions get categorized (categories appear in output CSV, not all "Uncategorized").
result: pass

### 9. All tests still pass
expected: `make test` (or `go test ./...`) passes all tests with 0 failures.
result: pass

### 10. Lint is clean
expected: `make lint` shows 0 issues.
result: pass

## Summary

total: 10
passed: 10
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]
