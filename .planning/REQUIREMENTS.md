# Requirements: camt-csv

**Defined:** 2026-03-23
**Core Value:** Reliable, maintainable financial data conversion with intelligent categorization.

## v1.6 Requirements

Requirements for multi-LLM provider support. Each maps to roadmap phases.

### Provider

- [ ] **PROV-01**: User can select AI provider via `ai.provider` config key (gemini/openrouter)
- [x] **PROV-02**: User can use OpenRouter API for chat-based categorization (tier 4 — AI fallback)
- [ ] **PROV-03**: User can set any OpenAI-compatible base URL via `ai.base_url` config key
- [x] **PROV-04**: OpenRouterClient implements AIClient interface with retry and rate-limiting

### Configuration

- [ ] **CONF-01**: All AI settings consolidated in config.yaml `ai:` section (provider, model, base_url, api_key)
- [ ] **CONF-02**: Single env var `CAMT_AI_API_KEY` replaces provider-specific vars (backward-compat: `GEMINI_API_KEY` still works as fallback)
- [ ] **CONF-03**: Model name in config.yaml accepted as-is for both providers (e.g. `gemini-3.0-flash` or `mistralai/mistral-small-2603`)
- [ ] **CONF-04**: Validation checks provider-specific requirements (api_key present, model non-empty)

### Semantic

- [ ] **SEM-01**: Semantic tier (embeddings) gracefully skips when no embedding-capable provider is available
- [ ] **SEM-02**: When provider is openrouter but `GEMINI_API_KEY` is also set, semantic tier uses Gemini embeddings

## Future Requirements

### Provider Expansion

- **PROV-05**: User can use Ollama as a local AI provider
- **PROV-06**: User can use Anthropic Claude API for categorization
- **PROV-07**: Embedding provider can be configured independently from chat provider

## Out of Scope

| Feature | Reason |
|---------|--------|
| OpenAI Go SDK dependency | Raw HTTP matches existing GeminiClient pattern; no SDK needed |
| OpenRouter embedding support | OpenRouter doesn't expose embedding endpoints |
| Streaming responses | Not needed for single-category classification |
| Provider-specific prompt tuning | Same prompt works across all chat completion APIs |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PROV-01 | Phase 16 | Pending |
| PROV-02 | Phase 16 | Complete |
| PROV-03 | Phase 16 | Pending |
| PROV-04 | Phase 16 | Complete |
| CONF-01 | Phase 16 | Pending |
| CONF-02 | Phase 16 | Pending |
| CONF-03 | Phase 16 | Pending |
| CONF-04 | Phase 16 | Pending |
| SEM-01 | Phase 16 | Pending |
| SEM-02 | Phase 16 | Pending |

**Coverage:**
- v1.6 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0

---
*Requirements defined: 2026-03-23*
*Last updated: 2026-03-23 — traceability mapped to phase 16*
