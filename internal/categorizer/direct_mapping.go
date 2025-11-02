package categorizer

import (
	"context"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// DirectMappingStrategy implements categorization using exact name matches
// from creditor and debitor mapping databases.
type DirectMappingStrategy struct {
	creditorMappings map[string]string // Maps creditor names to categories
	debitorMappings  map[string]string // Maps debitor names to categories
	store            CategoryStoreInterface
	logger           logging.Logger
	mu               sync.RWMutex // Protects the mappings
}

// NewDirectMappingStrategy creates a new DirectMappingStrategy instance.
func NewDirectMappingStrategy(store CategoryStoreInterface, logger logging.Logger) *DirectMappingStrategy {
	strategy := &DirectMappingStrategy{
		creditorMappings: make(map[string]string),
		debitorMappings:  make(map[string]string),
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

	// Convert to lowercase for case-insensitive lookup
	partyNameLower := strings.ToLower(tx.PartyName)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var categoryName string
	var found bool

	// Check appropriate mapping based on transaction direction
	if tx.IsDebtor {
		// For debtor transactions, check debitor mappings
		categoryName, found = s.debitorMappings[partyNameLower]
		if found {
			s.logger.WithFields(
				logging.Field{Key: "strategy", Value: s.Name()},
				logging.Field{Key: "party", Value: tx.PartyName},
				logging.Field{Key: "category", Value: categoryName},
				logging.Field{Key: "mapping_type", Value: "debitor"},
			).Debug("Transaction categorized using direct debitor mapping")
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
		// Normalize keys to lowercase for case-insensitive lookup
		for key, value := range creditorMappings {
			s.creditorMappings[strings.ToLower(key)] = value
		}
		s.mu.Unlock()
		s.logger.WithField("count", len(creditorMappings)).Debug("Loaded creditor mappings for DirectMappingStrategy")
	}

	// Load debitor mappings
	debitorMappings, err := s.store.LoadDebitorMappings()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to load debitor mappings for DirectMappingStrategy")
	} else {
		s.mu.Lock()
		// Normalize keys to lowercase for case-insensitive lookup
		for key, value := range debitorMappings {
			s.debitorMappings[strings.ToLower(key)] = value
		}
		s.mu.Unlock()
		s.logger.WithField("count", len(debitorMappings)).Debug("Loaded debitor mappings for DirectMappingStrategy")
	}
}

// ReloadMappings reloads the mappings from the store.
// This can be called when the underlying YAML files have been updated.
func (s *DirectMappingStrategy) ReloadMappings() {
	s.mu.Lock()
	// Clear existing mappings
	s.creditorMappings = make(map[string]string)
	s.debitorMappings = make(map[string]string)
	s.mu.Unlock()

	// Reload from store
	s.loadMappings()
}

// UpdateCreditorMapping adds or updates a creditor mapping.
func (s *DirectMappingStrategy) UpdateCreditorMapping(partyName, categoryName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creditorMappings[strings.ToLower(partyName)] = categoryName
}

// UpdateDebitorMapping adds or updates a debitor mapping.
func (s *DirectMappingStrategy) UpdateDebitorMapping(partyName, categoryName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.debitorMappings[strings.ToLower(partyName)] = categoryName
}

