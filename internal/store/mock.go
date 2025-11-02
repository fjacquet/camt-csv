package store

import (
	"fjacquet/camt-csv/internal/models"
)

// MockCategoryStore is a mock implementation of CategoryStore for testing.
type MockCategoryStore struct {
	Categories       []models.CategoryConfig
	CreditorMappings map[string]string
	DebitorMappings  map[string]string
	
	// Error flags for testing error conditions
	LoadCategoriesError       error
	LoadCreditorMappingsError error
	LoadDebitorMappingsError  error
	SaveCreditorMappingsError error
	SaveDebitorMappingsError  error
}

// LoadCategories returns the mock categories.
func (m *MockCategoryStore) LoadCategories() ([]models.CategoryConfig, error) {
	if m.LoadCategoriesError != nil {
		return nil, m.LoadCategoriesError
	}
	return m.Categories, nil
}

// LoadCreditorMappings returns the mock creditor mappings.
func (m *MockCategoryStore) LoadCreditorMappings() (map[string]string, error) {
	if m.LoadCreditorMappingsError != nil {
		return nil, m.LoadCreditorMappingsError
	}
	if m.CreditorMappings == nil {
		return make(map[string]string), nil
	}
	// Return a copy to avoid external modifications
	result := make(map[string]string)
	for k, v := range m.CreditorMappings {
		result[k] = v
	}
	return result, nil
}

// LoadDebitorMappings returns the mock debitor mappings.
func (m *MockCategoryStore) LoadDebitorMappings() (map[string]string, error) {
	if m.LoadDebitorMappingsError != nil {
		return nil, m.LoadDebitorMappingsError
	}
	if m.DebitorMappings == nil {
		return make(map[string]string), nil
	}
	// Return a copy to avoid external modifications
	result := make(map[string]string)
	for k, v := range m.DebitorMappings {
		result[k] = v
	}
	return result, nil
}

// SaveCreditorMappings updates the mock creditor mappings.
func (m *MockCategoryStore) SaveCreditorMappings(mappings map[string]string) error {
	if m.SaveCreditorMappingsError != nil {
		return m.SaveCreditorMappingsError
	}
	if m.CreditorMappings == nil {
		m.CreditorMappings = make(map[string]string)
	}
	// Update the mock mappings
	for k, v := range mappings {
		m.CreditorMappings[k] = v
	}
	return nil
}

// SaveDebitorMappings updates the mock debitor mappings.
func (m *MockCategoryStore) SaveDebitorMappings(mappings map[string]string) error {
	if m.SaveDebitorMappingsError != nil {
		return m.SaveDebitorMappingsError
	}
	if m.DebitorMappings == nil {
		m.DebitorMappings = make(map[string]string)
	}
	// Update the mock mappings
	for k, v := range mappings {
		m.DebitorMappings[k] = v
	}
	return nil
}

// FindConfigFile is a mock implementation that returns a dummy path.
func (m *MockCategoryStore) FindConfigFile(filename string) (string, error) {
	return "/mock/path/" + filename, nil
}