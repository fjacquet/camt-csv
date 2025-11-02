package categorizer

import "fjacquet/camt-csv/internal/models"

// CategoryStoreInterface defines the interface for category data storage.
// This allows for dependency injection and easier testing.
type CategoryStoreInterface interface {
	LoadCategories() ([]models.CategoryConfig, error)
	LoadCreditorMappings() (map[string]string, error)
	LoadDebitorMappings() (map[string]string, error)
	SaveCreditorMappings(mappings map[string]string) error
	SaveDebitorMappings(mappings map[string]string) error
}