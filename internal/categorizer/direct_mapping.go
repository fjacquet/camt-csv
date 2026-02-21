package categorizer

import (
	"context"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// DirectMappingStrategy implements categorization using exact name matches
// from creditor and debtor mapping databases.
type DirectMappingStrategy struct {
	creditorMappings map[string]string // Maps creditor names to categories
	debtorMappings   map[string]string // Maps debtor names to categories
	store            CategoryStoreInterface
	logger           logging.Logger
	mu               sync.RWMutex // Protects the mappings
}

// NewDirectMappingStrategy creates a new DirectMappingStrategy instance.
func NewDirectMappingStrategy(creditorMappings, debtorMappings map[string]string, store CategoryStoreInterface, logger logging.Logger) *DirectMappingStrategy {
	strategy := &DirectMappingStrategy{
		creditorMappings: creditorMappings,
		debtorMappings:   debtorMappings,
		store:            store,
		logger:           logger,
	}

	return strategy
}

// Name returns the name of this strategy for logging and debugging.
func (s *DirectMappingStrategy) Name() string {
	return "DirectMapping"
}

// Categorize attempts to categorize a transaction using direct name mapping.
func (s *DirectMappingStrategy) Categorize(ctx context.Context, tx Transaction) (models.Category, bool, error) {
	// If party name is empty, cannot categorize
	if strings.TrimSpace(tx.PartyName) == "" {
		return models.Category{}, false, nil
	}

	// Performance optimization: Use helper function to minimize allocations during party name normalization
	partyNameLower := strings.ToLower(tx.PartyName)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var categoryName string
	var found bool

	// Check appropriate mapping based on transaction direction
	if tx.IsDebtor {
		// For debtor transactions, check debtor mappings
		categoryName, found = s.debtorMappings[partyNameLower]
		if found {
			s.logger.WithFields(
				logging.Field{Key: "strategy", Value: s.Name()},
				logging.Field{Key: "party", Value: tx.PartyName},
				logging.Field{Key: "category", Value: categoryName},
				logging.Field{Key: "mapping_type", Value: "debtor"},
			).Debug("Transaction categorized using direct debtor mapping")
		}
	} else {
		// For creditor transactions, check creditor mappings
		categoryName, found = s.creditorMappings[partyNameLower]
		if found {
			s.logger.WithFields(
				logging.Field{Key: "strategy", Value: s.Name()},
				logging.Field{Key: "party", Value: tx.PartyName},
				logging.Field{Key: "category", Value: categoryName},
				logging.Field{Key: "mapping_type", Value: "creditor"},
			).Debug("Transaction categorized using direct creditor mapping")
		}
	}

	// If not found or if the found category is a failed AI attempt, allow other strategies to try
	if !found || categoryName == "Uncategorized (AI)" {
		if found && categoryName == "Uncategorized (AI)" {
			s.logger.WithFields(
				logging.Field{Key: "strategy", Value: s.Name()},
				logging.Field{Key: "party", Value: tx.PartyName},
				logging.Field{Key: "category", Value: categoryName},
			).Debug("Found previous failed AI attempt, allowing retry with other strategies")
		}
		return models.Category{
			Confidence: 0.0,
			Source:     "none",
		}, false, nil
	}

	// Create category with name and description
	category := models.Category{
		Name:        categoryName,
		Description: categoryDescriptionFromName(categoryName),
		Confidence:  1.0, // Highest confidence for direct mappings
		Source:      "direct_mapping",
	}

	return category, true, nil
}

// ReloadMappings reloads the mappings from the store.
// This can be called when the underlying YAML files have been updated.
func (s *DirectMappingStrategy) ReloadMappings() {
	// Load data from store FIRST (outside lock)
	creditorMappings, creditorErr := s.store.LoadCreditorMappings()
	if creditorErr != nil {
		s.logger.WithError(creditorErr).Warn("Failed to load creditor mappings during reload")
	}

	debtorMappings, debtorErr := s.store.LoadDebtorMappings()
	if debtorErr != nil {
		s.logger.WithError(debtorErr).Warn("Failed to load debtor mappings during reload")
	}

	// Build new maps with normalized keys (outside lock)
	newCreditorMappings := make(map[string]string, 100)
	if creditorErr == nil {
		for key, value := range creditorMappings {
			newCreditorMappings[strings.ToLower(key)] = value
		}
	}

	newDebtorMappings := make(map[string]string, 100)
	if debtorErr == nil {
		for key, value := range debtorMappings {
			newDebtorMappings[strings.ToLower(key)] = value
		}
	}

	// Atomic swap under lock (very brief critical section)
	s.mu.Lock()
	s.creditorMappings = newCreditorMappings
	s.debtorMappings = newDebtorMappings
	s.mu.Unlock()

	s.logger.WithFields(
		logging.Field{Key: "creditor_count", Value: len(newCreditorMappings)},
		logging.Field{Key: "debtor_count", Value: len(newDebtorMappings)},
	).Debug("Reloaded mappings for DirectMappingStrategy")
}

// UpdateCreditorMapping adds or updates a creditor mapping.
func (s *DirectMappingStrategy) UpdateCreditorMapping(partyName, categoryName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Performance optimization: Use helper function to minimize allocations during mapping updates
	s.creditorMappings[strings.ToLower(partyName)] = categoryName
}

// UpdateDebtorMapping adds or updates a debtor mapping.
func (s *DirectMappingStrategy) UpdateDebtorMapping(partyName, categoryName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Performance optimization: Use helper function to minimize allocations during mapping updates
	s.debtorMappings[strings.ToLower(partyName)] = categoryName
}
