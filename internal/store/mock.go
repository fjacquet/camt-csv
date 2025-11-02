package store

import (
	"fjacquet/camt-csv/internal/models"
)

// MockCategoryStore is a mock implementation of CategoryStore for testing.
type MockCategoryStore struct {
	Categories       []models.CategoryConfig
	CreditorMappings map[string]string
	DebtorMappings   map[string]string

	// Error flags for testing error conditions
	LoadCategoriesError       error
	LoadCreditorMappingsError error
	LoadDebtorMappingsError   error
	SaveCreditorMappingsError error
	SaveDebtorMappingsError   error
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

// LoadDebtorMappings returns the mock debtor mappings.
func (m *MockCategoryStore) LoadDebtorMappings() (map[string]string, error) {
	if m.LoadDebtorMappingsError != nil {
		return nil, m.LoadDebtorMappingsError
	}
	if m.DebtorMappings == nil {
		return make(map[string]string), nil
	}
	// Return a copy to avoid external modifications
	result := make(map[string]string)
	for k, v := range m.DebtorMappings {
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

// SaveDebtorMappings updates the mock debtor mappings.
func (m *MockCategoryStore) SaveDebtorMappings(mappings map[string]string) error {
	if m.SaveDebtorMappingsError != nil {
		return m.SaveDebtorMappingsError
	}
	if m.DebtorMappings == nil {
		m.DebtorMappings = make(map[string]string)
	}
	// Update the mock mappings
	for k, v := range mappings {
		m.DebtorMappings[k] = v
	}
	return nil
}

// FindConfigFile is a mock implementation that returns a dummy path.
func (m *MockCategoryStore) FindConfigFile(filename string) (string, error) {
	return "/mock/path/" + filename, nil
}
