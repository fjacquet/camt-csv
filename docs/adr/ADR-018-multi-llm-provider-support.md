# ADR-018: Multi-LLM Provider Support

## Status

**ACCEPTED** - Implemented in v1.6

## Context

The AI categorization tier was tightly coupled to Google Gemini — the only supported LLM provider. Users wanted the ability to switch to cheaper or self-hosted alternatives (e.g., Mistral Small via OpenRouter) without modifying code. Additionally, Gemini is the only provider offering embedding endpoints, so a split architecture is needed when using non-Gemini chat providers.

## Decision

Implement provider-agnostic AI categorization with a split chat/embedding client architecture:

1. **Provider selection via config**: `ai.provider` key (`gemini` or `openrouter`) determines which `AIClient` implementation handles chat-based categorization (tier 4).
2. **OpenAI-compatible client**: `OpenRouterClient` uses the standard `/chat/completions` API format, making it compatible with any OpenAI-compatible endpoint via `ai.base_url`.
3. **Split client wiring**: The DI container creates separate `chatClient` and `embeddingClient`. When provider is `openrouter`, embeddings fall back to Gemini (if `GEMINI_API_KEY` is set) or are skipped gracefully.
4. **Unified API key**: `CAMT_AI_API_KEY` replaces provider-specific env vars, with `GEMINI_API_KEY` retained as backward-compatible fallback.
5. **Constructor injection**: `NewGeminiClient` accepts `apiKey` as a parameter instead of reading from environment, enabling multi-key scenarios.

## Rationale

### Why not a single universal client?

OpenRouter and Gemini use incompatible API formats (OpenAI chat/completions vs. Gemini's native format). A universal adapter would add unnecessary complexity. Instead, each provider has its own client implementing the existing `AIClient` interface.

### Why split chat and embedding clients?

OpenRouter does not expose embedding endpoints. When a user selects OpenRouter for cheaper chat categorization, they may still want Gemini embeddings for the semantic tier. The `SetEmbeddingClient()` method on `Categorizer` allows independent wiring without changing the `NewCategorizer` signature.

### Why raw HTTP instead of an SDK?

Consistent with the existing `GeminiClient` pattern (ADR-006). Both clients use `net/http` directly — no SDK dependency, minimal attack surface, full control over retry/rate-limiting.

## Consequences

### Positive

- Users can switch AI providers without code changes — just config
- OpenRouter provides access to dozens of models at varying price points
- Semantic tier degrades gracefully when embeddings are unavailable
- Backward compatible — existing Gemini-only configs work unchanged

### Negative

- Two separate client implementations to maintain
- Embedding support limited to Gemini (OpenRouter has no embedding API)
- Prompt must work across different model families (no provider-specific tuning)

## Alternatives Considered

| Alternative | Reason Rejected |
|-------------|-----------------|
| Jetify AI SDK | Adds dependency; doesn't match existing raw HTTP pattern |
| LangChain Go | Over-engineered for single-category classification |
| Single OpenAI-format client for all | Gemini's native API format is incompatible |
| Ollama support | Deferred to future — requires local model management |
