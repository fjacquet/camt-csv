package categorizer

import "fjacquet/camt-csv/internal/models"

// CategoryStoreInterface defines the interface for category data storage.
// This allows for dependency injection and easier testing.
type CategoryStoreInterface interface {
	LoadCategories() ([]models.CategoryConfig, error)
	LoadCreditorMappings() (map[string]string, error)
	LoadDebtorMappings() (map[string]string, error)
	SaveCreditorMappings(mappings map[string]string) error
	SaveDebtorMappings(mappings map[string]string) error
}

// StagingStoreInterface defines the interface for staging AI categorization suggestions.
// When auto-learn is disabled, AI results are written to staging files instead of being
// discarded. Users can later review and manually promote entries to the main YAML files.
type StagingStoreInterface interface {
	AppendCreditorSuggestion(partyName, categoryName string) error
	AppendDebtorSuggestion(partyName, categoryName string) error
}
