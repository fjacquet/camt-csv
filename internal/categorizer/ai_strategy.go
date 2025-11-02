package categorizer

import (
	"context"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// AIStrategy implements categorization using AI services.
// It uses the AIClient interface to interact with external AI services.
type AIStrategy struct {
	aiClient AIClient
	logger   logging.Logger
}

// NewAIStrategy creates a new AIStrategy instance.
func NewAIStrategy(aiClient AIClient, logger logging.Logger) *AIStrategy {
	return &AIStrategy{
		aiClient: aiClient,
		logger:   logger,
	}
}

// Name returns the name of this strategy for logging and debugging.
func (s *AIStrategy) Name() string {
	return "AI"
}

// Categorize attempts to categorize a transaction using AI services.
func (s *AIStrategy) Categorize(ctx context.Context, tx Transaction) (models.Category, bool, error) {
	// If no AI client is available, cannot categorize
	if s.aiClient == nil {
		s.logger.WithFields(
			logging.Field{Key: "strategy", Value: s.Name()},
			logging.Field{Key: "party", Value: tx.PartyName},
		).Debug("AI client not available, skipping AI categorization")
		return models.Category{}, false, nil
	}

	// If party name is empty, cannot categorize effectively
	if strings.TrimSpace(tx.PartyName) == "" {
		return models.Category{}, false, nil
	}

	// Convert Transaction to models.Transaction for AI client
	modelTransaction, err := s.convertToModelTransaction(tx)
	if err != nil {
		s.logger.WithError(err).WithFields(
			logging.Field{Key: "strategy", Value: s.Name()},
			logging.Field{Key: "party", Value: tx.PartyName},
		).Warn("Failed to convert transaction for AI categorization")
		return models.Category{}, false, nil
	}

	// Use the AI client to categorize
	categorizedTransaction, err := s.aiClient.Categorize(ctx, modelTransaction)
	if err != nil {
		s.logger.WithError(err).WithFields(
			logging.Field{Key: "strategy", Value: s.Name()},
			logging.Field{Key: "party", Value: tx.PartyName},
		).Warn("AI categorization failed")
		return models.Category{}, false, nil
	}

	// Check if AI provided a valid category
	if strings.TrimSpace(categorizedTransaction.Category) == "" ||
		categorizedTransaction.Category == models.CategoryUncategorized {
		s.logger.WithFields(
			logging.Field{Key: "strategy", Value: s.Name()},
			logging.Field{Key: "party", Value: tx.PartyName},
			logging.Field{Key: "ai_category", Value: categorizedTransaction.Category},
		).Debug("AI returned uncategorized result")
		return models.Category{}, false, nil
	}

	s.logger.WithFields(
		logging.Field{Key: "strategy", Value: s.Name()},
		logging.Field{Key: "party", Value: tx.PartyName},
		logging.Field{Key: "category", Value: categorizedTransaction.Category},
	).Debug("Transaction categorized using AI")

	// Return the category from the AI response
	category := models.Category{
		Name:        categorizedTransaction.Category,
		Description: categoryDescriptionFromName(categorizedTransaction.Category),
	}

	return category, true, nil
}

// convertToModelTransaction converts a categorizer Transaction to a models.Transaction
// for use with the AI client.
func (s *AIStrategy) convertToModelTransaction(tx Transaction) (models.Transaction, error) {
	// Parse the date string to time.Time
	var parsedDate time.Time
	var err error

	if tx.Date != "" {
		parsedDate, err = dateutils.ParseDateString(tx.Date)
		if err != nil {
			// If parsing fails, use zero time and log warning
			s.logger.WithError(err).WithFields(
				logging.Field{Key: "date_string", Value: tx.Date},
				logging.Field{Key: "party", Value: tx.PartyName},
			).Debug("Failed to parse date for AI categorization, using zero time")
			parsedDate = time.Time{}
		}
	}

	modelTransaction := models.Transaction{
		PartyName:   tx.PartyName,
		Description: tx.Description,
		Amount:      models.ParseAmount(tx.Amount),
		Date:        parsedDate,
		Category:    "", // Will be filled by AI
	}

	// Add additional context from the Info field if available
	if tx.Info != "" {
		// Combine description and info for better AI context
		if modelTransaction.Description != "" {
			modelTransaction.Description += " | " + tx.Info
		} else {
			modelTransaction.Description = tx.Info
		}
	}

	return modelTransaction, nil
}
