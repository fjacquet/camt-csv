package categorizer

import (
	"context"

	"fjacquet/camt-csv/internal/models"
)

// AIClient defines the interface for AI-based categorization services.
// This abstraction allows the core categorization logic to be tested independently
// of external API calls and provides flexibility in choosing AI providers.
type AIClient interface {
	// Categorize takes a context and a Transaction model, and returns the categorized Transaction
	// or an error if categorization fails.
	// Implementations will interact with an external AI service (e.g., Google Gemini).
	Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error)

	// GetEmbedding returns the vector embedding for the given text.
	GetEmbedding(ctx context.Context, text string) ([]float32, error)
}

// BatchAIClient is optionally implemented by AI clients that can categorize
// many transactions in one request (the Gemini and OpenRouter clients do).
// Keys of the result are lowercased, trimmed party names; values are cleaned
// category answers.
type BatchAIClient interface {
	CategorizeBatch(ctx context.Context, transactions []models.Transaction) (map[string]string, error)
}
