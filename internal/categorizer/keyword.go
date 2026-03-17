package categorizer

import (
	"context"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// KeywordStrategy implements categorization using keyword pattern matching
// from category configuration loaded from YAML files.
type KeywordStrategy struct {
	categories []models.CategoryConfig
	store      CategoryStoreInterface
	logger     logging.Logger
}

// NewKeywordStrategy creates a new KeywordStrategy instance.
func NewKeywordStrategy(categories []models.CategoryConfig, store CategoryStoreInterface, logger logging.Logger) *KeywordStrategy {
	strategy := &KeywordStrategy{
		categories: categories,
		store:      store,
		logger:     logger,
	}

	return strategy
}

// Name returns the name of this strategy for logging and debugging.
func (s *KeywordStrategy) Name() string {
	return "Keyword"
}

// Categorize attempts to categorize a transaction using keyword pattern matching.
func (s *KeywordStrategy) Categorize(ctx context.Context, tx Transaction) (models.Category, bool, error) {
	// If party name is empty, cannot categorize
	if strings.TrimSpace(tx.PartyName) == "" {
		return models.Category{}, false, nil
	}

	// Performance optimization: Use helper function with strings.Builder to minimize allocations
	// during case conversion in the categorization hot path
	partyName := strings.ToUpper(tx.PartyName)
	description := strings.ToUpper(tx.Info)

	// Try to match against category keywords
	for _, categoryConfig := range s.categories {
		for _, keyword := range categoryConfig.Keywords {
			// Performance optimization: Use helper function to minimize allocations in keyword matching loop
			keywordUpper := strings.ToUpper(keyword)

			// Check if keyword appears in party name or description
			if strings.Contains(partyName, keywordUpper) || strings.Contains(description, keywordUpper) {
				s.logger.WithFields(
					logging.Field{Key: "strategy", Value: s.Name()},
					logging.Field{Key: "party", Value: tx.PartyName},
					logging.Field{Key: "keyword", Value: keyword},
					logging.Field{Key: "category", Value: categoryConfig.Name},
				).Debug("Transaction categorized using keyword matching")

				category := models.Category{
					Name:        categoryConfig.Name,
					Description: categoryDescriptionFromName(categoryConfig.Name),
					Confidence:  0.95, // High confidence for keyword matches
					Source:      "keyword",
				}

				return category, true, nil
			}
		}
	}

	return models.Category{}, false, nil
}

// loadCategories loads category configurations from the store.
func (s *KeywordStrategy) loadCategories() {
	categories, err := s.store.LoadCategories()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to load categories for KeywordStrategy")
	} else {
		s.categories = categories
		s.logger.WithField("count", len(categories)).Debug("Loaded categories for KeywordStrategy")
	}
}

// ReloadCategories reloads the categories from the store.
// This can be called when the underlying YAML files have been updated.
func (s *KeywordStrategy) ReloadCategories() {
	s.loadCategories()
}
