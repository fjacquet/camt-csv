package categorizer

import (
	"context"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAIClient for testing dependency injection
type MockAIClient struct {
	CategorizeFunc func(ctx context.Context, transaction models.Transaction) (models.Transaction, error)
}

func (m *MockAIClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	if m.CategorizeFunc != nil {
		return m.CategorizeFunc(ctx, transaction)
	}
	// Default behavior: return transaction with a mock category
	transaction.Category = "MockCategory"
	return transaction, nil
}

func TestCategorizeTransactionWithCategorizer(t *testing.T) {
	// Create mock store and logger
	testStore := &store.CategoryStore{
		CategoriesFile: "testdata/categories.yaml",
		CreditorsFile:  "testdata/creditors.yaml",
		DebtorsFile:    "testdata/debtors.yaml",
	}
	testLogger := logging.NewLogrusAdapter("debug", "text")

	// Create mock AI client
	mockAI := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
			transaction.Category = "AI_Category"
			return transaction, nil
		},
	}

	// Create categorizer with dependency injection
	cat := NewCategorizer(mockAI, testStore, testLogger)

	tests := []struct {
		name        string
		categorizer *Categorizer
		transaction Transaction
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil categorizer",
			categorizer: nil,
			transaction: Transaction{PartyName: "Test", IsDebtor: true},
			expectError: true,
			errorMsg:    "categorizer cannot be nil",
		},
		{
			name:        "valid categorizer with direct mapping",
			categorizer: cat,
			transaction: Transaction{PartyName: "COOP", IsDebtor: true},
			expectError: false,
		},
		{
			name:        "valid categorizer with AI fallback",
			categorizer: cat,
			transaction: Transaction{PartyName: "Unknown Store", IsDebtor: true},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, err := CategorizeTransactionWithCategorizer(tt.categorizer, tt.transaction)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, category.Name)
			}
		})
	}
}

func TestCategorizer_UpdateMethods(t *testing.T) {
	// Create mock store and logger
	testStore := &store.CategoryStore{
		CategoriesFile: "testdata/categories.yaml",
		CreditorsFile:  "testdata/creditors.yaml",
		DebtorsFile:    "testdata/debtors.yaml",
	}
	testLogger := logging.NewLogrusAdapter("debug", "text")

	// Create categorizer
	cat := NewCategorizer(nil, testStore, testLogger)

	// Test UpdateDebitorCategory
	cat.UpdateDebitorCategory("TestDebitor", "TestCategory")

	// Verify the mapping was added
	transaction := Transaction{PartyName: "TestDebitor", IsDebtor: true}
	category, err := cat.CategorizeTransaction(transaction)
	require.NoError(t, err)
	assert.Equal(t, "TestCategory", category.Name)

	// Test UpdateCreditorCategory
	cat.UpdateCreditorCategory("TestCreditor", "TestCategory2")

	// Verify the mapping was added
	transaction = Transaction{PartyName: "TestCreditor", IsDebtor: false}
	category, err = cat.CategorizeTransaction(transaction)
	require.NoError(t, err)
	assert.Equal(t, "TestCategory2", category.Name)
}

func TestCategorizer_DependencyInjection(t *testing.T) {
	// Create mock store and logger
	testStore := &store.CategoryStore{
		CategoriesFile: "testdata/categories.yaml",
		CreditorsFile:  "testdata/creditors.yaml",
		DebtorsFile:    "testdata/debtors.yaml",
	}
	testLogger := logging.NewLogrusAdapter("debug", "text")

	// Test with nil AI client
	cat1 := NewCategorizer(nil, testStore, testLogger)
	assert.NotNil(t, cat1)
	assert.Nil(t, cat1.aiClient)

	// Test with mock AI client
	mockAI := &MockAIClient{}
	cat2 := NewCategorizer(mockAI, testStore, testLogger)
	assert.NotNil(t, cat2)
	assert.NotNil(t, cat2.aiClient)

	// Test that both categorizers work independently
	transaction := Transaction{PartyName: "Unknown", IsDebtor: true}

	// First categorizer should return uncategorized (no AI)
	category1, err := cat1.CategorizeTransaction(transaction)
	require.NoError(t, err)
	assert.Equal(t, models.CategoryUncategorized, category1.Name)

	// Second categorizer should use AI
	category2, err := cat2.CategorizeTransaction(transaction)
	require.NoError(t, err)
	assert.Equal(t, "MockCategory", category2.Name)
}
