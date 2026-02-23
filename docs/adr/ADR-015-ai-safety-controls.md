# ADR-015: AI Categorization Safety Controls

## Status
Accepted

## Context

The Gemini AI integration (ADR-006) could call the Gemini API for every uncategorized transaction. Without guardrails this creates three risks:

1. **API quota exhaustion** — large batch runs could hit Google's rate limits and fail entirely
2. **Silent data corruption** — AI suggestions accepted automatically could miscategorize transactions and write bad entries to the YAML database
3. **No audit trail** — no way to know which categorization came from direct match vs keyword vs AI

Additionally, the AI client was initially only testable via live API calls, making tests slow and quota-expensive.

## Decision

### 1. Auto-learn defaults to OFF

```
--auto-learn   # opt-in flag; default is OFF
```

Without `--auto-learn`:
- AI suggestions are written to **staging files** (`database/staging_creditors.yaml`, `database/staging_debtors.yaml`)
- Staging files are `.gitignore`d — they are for review only
- User manually promotes staging entries to the live YAML files

With `--auto-learn`:
- AI suggestions write directly to live YAML (with backup per ADR-011)

### 2. Rate limiting with burst=1

A token bucket rate limiter wraps all Gemini API calls:
- **Rate**: 1 request/second sustained
- **Burst**: 1 (no bursting — strict quota protection)

This prevents quota exhaustion on large batch runs at the cost of throughput. For typical personal finance use (hundreds of transactions per run), 1 req/s is acceptable.

### 3. Exponential backoff retry

Retryable API errors (429, 503, network timeouts) trigger automatic retry:
- Initial delay: 1s
- Multiplier: 2×
- Max retries: 3
- Max delay: 30s

### 4. Confidence scoring per strategy tier

Each categorization strategy returns a confidence score alongside the category:

| Strategy | Confidence |
|----------|-----------|
| DirectStrategy (exact match) | 1.00 |
| KeywordStrategy (regex match) | 0.95 |
| SemanticStrategy (embedding similarity) | 0.90 |
| AIStrategy (LLM generation) | 0.80 |

Score is included in the transaction output and staging files for audit.

### 5. Testable AIClient interface

```go
type AIClient interface {
    CategorizeTransaction(ctx context.Context, description string) (string, error)
    GetEmbedding(ctx context.Context, text string) ([]float32, error)
}
```

Tests inject a `MockAIClient` via the DI container. `TEST_MODE=true` disables real API calls in integration tests.

## Consequences

**Positive:**
- Safe default: users cannot accidentally corrupt their YAML database on first run
- Rate limiter prevents quota-related failures in batch runs
- Confidence scores enable filtering/auditing of categorizations
- Tests run without API keys or quota costs

**Negative:**
- `--auto-learn` is a non-obvious flag for new users who expect AI to "just work"
- burst=1 makes batch categorization slow for large uncategorized datasets

## Future Work

- Configurable rate limit (per-user quota tiers vary)
- Bulk embedding API to categorize N transactions in one call instead of N calls
- Staging file review UI / approval workflow
