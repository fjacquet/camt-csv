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
}
