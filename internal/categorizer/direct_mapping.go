package categorizer

import (
	"context"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// normalizeStringToLower converts a string to lowercase using strings.Builder
// for optimal performance in hot paths. Pre-allocates capacity to minimize allocations.
//
// Performance rationale: This approach provides consistent memory allocation
// patterns and prevents multiple reallocations during string processing.
// The pre-allocation ensures optimal performance in categorization hot paths
// where string normalization is performed frequently.
func normalizeStringToLower(input string) string {
	if input == "" {
		return ""
	}
	// Performance optimization: Pre-allocate builder capacity to avoid reallocations
	builder := strings.Builder{}
	builder.Grow(len(input))
	builder.WriteString(strings.ToLower(input))
	return builder.String()
}

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
func NewDirectMappingStrategy(store CategoryStoreInterface, logger logging.Logger) *DirectMappingStrategy {
	strategy := &DirectMappingStrategy{
		creditorMappings: make(map[string]string, 100), // Pre-allocate with size hint
		debtorMappings:   make(map[string]string, 100), // Pre-allocate with size hint
		store:            store,
		logger:           logger,
	}

	// Load mappings from store
	strategy.loadMappings()

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
	partyNameLower := normalizeStringToLower(tx.PartyName)

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

	if !found {
		return models.Category{}, false, nil
	}

	// Create category with name and description
	category := models.Category{
		Name:        categoryName,
		Description: categoryDescriptionFromName(categoryName),
	}

	return category, true, nil
}

// loadMappings loads creditor and debitor mappings from the store.
func (s *DirectMappingStrategy) loadMappings() {
	// Load creditor mappings
	creditorMappings, err := s.store.LoadCreditorMappings()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to load creditor mappings for DirectMappingStrategy")
	} else {
		s.mu.Lock()
		// Pre-allocate map if needed and normalize keys to lowercase for case-insensitive lookup
		if len(creditorMappings) > len(s.creditorMappings) {
			newMap := make(map[string]string, len(creditorMappings))
			for k, v := range s.creditorMappings {
				newMap[k] = v
			}
			s.creditorMappings = newMap
		}

		// Performance optimization: Use helper function to minimize allocations when processing mapping keys
		for key, value := range creditorMappings {
			s.creditorMappings[normalizeStringToLower(key)] = value
		}
		s.mu.Unlock()
		s.logger.WithField("count", len(creditorMappings)).Debug("Loaded creditor mappings for DirectMappingStrategy")
	}

	// Load debtor mappings
	debtorMappings, err := s.store.LoadDebtorMappings()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to load debtor mappings for DirectMappingStrategy")
	} else {
		s.mu.Lock()
		// Pre-allocate map if needed and normalize keys to lowercase for case-insensitive lookup
		if len(debtorMappings) > len(s.debtorMappings) {
			newMap := make(map[string]string, len(debtorMappings))
			for k, v := range s.debtorMappings {
				newMap[k] = v
			}
			s.debtorMappings = newMap
		}

		// Performance optimization: Use helper function to minimize allocations when processing mapping keys
		for key, value := range debtorMappings {
			s.debtorMappings[normalizeStringToLower(key)] = value
		}
		s.mu.Unlock()
		s.logger.WithField("count", len(debtorMappings)).Debug("Loaded debtor mappings for DirectMappingStrategy")
	}
}

// ReloadMappings reloads the mappings from the store.
// This can be called when the underlying YAML files have been updated.
func (s *DirectMappingStrategy) ReloadMappings() {
	s.mu.Lock()
	// Clear existing mappings
	s.creditorMappings = make(map[string]string)
	s.debtorMappings = make(map[string]string)
	s.mu.Unlock()

	// Reload from store
	s.loadMappings()
}

// UpdateCreditorMapping adds or updates a creditor mapping.
func (s *DirectMappingStrategy) UpdateCreditorMapping(partyName, categoryName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Performance optimization: Use helper function to minimize allocations during mapping updates
	s.creditorMappings[normalizeStringToLower(partyName)] = categoryName
}

// UpdateDebtorMapping adds or updates a debtor mapping.
func (s *DirectMappingStrategy) UpdateDebtorMapping(partyName, categoryName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Performance optimization: Use helper function to minimize allocations during mapping updates
	s.debtorMappings[normalizeStringToLower(partyName)] = categoryName
}
