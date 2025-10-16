package categorizer

import (
	"context"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

// GeminiClient implements the AIClient interface for interacting with the Google Gemini API.
type GeminiClient struct {
	// Add fields for Gemini API client, e.g., API key, HTTP client
	// For now, we'll keep it simple as the actual API interaction is out of scope for this task.
	log *logrus.Logger
}

// NewGeminiClient creates a new instance of GeminiClient.
func NewGeminiClient(logger *logrus.Logger) *GeminiClient {
	if logger == nil {
		logger = logrus.New()
	}
	return &GeminiClient{
		log: logger,
	}
}

// Categorize takes a context and a Transaction model, and returns the categorized Transaction
// or an error if categorization fails.
// This is a placeholder implementation. Actual Gemini API calls would go here.
func (c *GeminiClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	c.log.WithFields(logrus.Fields{
		"operation":   "gemini_categorization",
		"description": transaction.Description,
	}).Debug("Attempting to categorize transaction using Gemini API (mocked)")

	// Simulate API call and categorization logic
	// In a real implementation, this would involve:
	// 1. Constructing a prompt for the Gemini API based on transaction details.
	// 2. Making an HTTP request to the Gemini API.
	// 3. Parsing the API response to extract the category.
	// 4. Handling potential API errors or rate limits.

	// For demonstration, we'll assign a dummy category.
	// In a real scenario, you might have more sophisticated logic or fallbacks.
	if transaction.Description == "Coffee Shop" {
		transaction.Category = "Food & Drink"
	} else if transaction.Description == "Online Store" {
		transaction.Category = "Shopping"
	} else {
		transaction.Category = "Uncategorized (AI)"
	}

	c.log.WithFields(logrus.Fields{
		"operation":   "gemini_categorization",
		"description": transaction.Description,
		"category":    transaction.Category,
	}).Debug("Transaction categorized by Gemini API (mocked)")

	return transaction, nil
}
