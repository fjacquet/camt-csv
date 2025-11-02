package categorizer

import (
	"context"

	"fjacquet/camt-csv/internal/models"
)

// CategorizationStrategy defines a method for categorizing transactions.
// Each strategy implements a specific approach to categorization (direct mapping, keywords, AI, etc.).
type CategorizationStrategy interface {
	// Categorize attempts to categorize a transaction using this strategy.
	// Returns the category, a boolean indicating if categorization was successful,
	// and any error encountered during the process.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request-scoped values
	//   - tx: Transaction to categorize
	//
	// Returns:
	//   - models.Category: The assigned category (only valid if found is true)
	//   - bool: Whether categorization was successful
	//   - error: Any error encountered during categorization
	Categorize(ctx context.Context, tx Transaction) (models.Category, bool, error)

	// Name returns the name of this strategy for logging and debugging purposes.
	Name() string
}