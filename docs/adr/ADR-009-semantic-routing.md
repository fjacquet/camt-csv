# ADR-009: Semantic Routing for Transaction Categorization

## Status
Proposed

## Context
The current transaction categorization system uses a "hybrid" approach consisting of:
1. Direct match (exact string match)
2. Keyword match (regex/substring)
3. AI fallback (LLM generation via Gemini)

While the keyword match is fast, it is brittle and requires constant maintenance of keyword lists. The AI fallback is powerful but slower and more expensive (in terms of tokens/latency) for every single uncategorized transaction.

"Semantic Routing" (or Vector Search) offers a middle ground:
- **Speed**: Faster than LLM generation (once embeddings are cached).
- **Flexibility**: Matches "conceptually similar" transactions without needing exact keyword matches (e.g., matching "Starbucks" to "Alimentation" even if "Starbucks" isn't in the keyword list but "Coffee" is, and the embedding model understands the relationship).
- **Cost**: Cheaper than generative calls.

## Decision
We will implement a **Semantic Categorization Strategy** that uses vector embeddings to categorize transactions.

### Architecture
1.  **Index Generation**:
    - Load categories and their keywords from `database/categories.yaml`.
    - Create a "representative text" for each category (e.g., "CategoryName: keyword1, keyword2, ...").
    - Generate vector embeddings for each category using an embedding model (e.g., Gemini `embedding-001`).
    - Cache these embeddings in memory (and potentially persist them to disk to avoid re-computing on every run).

2.  **Runtime Categorization**:
    - For each transaction, generate an embedding for its `PartyName` + `Description`.
    - Calculate the **Cosine Similarity** between the transaction embedding and all category embeddings.
    - Identify the category with the highest similarity score.
    - **Thresholding**: If the score exceeds a defined threshold (e.g., 0.70), assign the category. Otherwise, return "Uncategorized" and let the next strategy (LLM) handle it.

3.  **Integration**:
    - Insert `SemanticStrategy` into the categorization chain:
      `DirectStrategy` -> `KeywordStrategy` -> `SemanticStrategy` -> `AIStrategy` (LLM).

### Technical Changes
- **`AIClient` Interface**: Add `GetEmbedding(ctx, text) ([]float32, error)`.
- **`GeminiClient`**: Implement `GetEmbedding` using the Gemini Embeddings API.
- **`SemanticStrategy`**: New struct implementing `CategorizationStrategy`.
- **Configuration**: Add `SEMANTIC_MATCH_THRESHOLD` (default 0.7).

## Consequences
- **Positive**: Improved categorization rate without explicit keyword maintenance. Lower latency/cost compared to pure LLM.
- **Negative**: Adds a dependency on an embedding model. Requires managing embedding cache.
- **Risks**: False positives if the threshold is too low.

## Future Work
- Persist the embedding index to a file (e.g., `embeddings.json` or a lightweight vector DB) to speed up startup.
