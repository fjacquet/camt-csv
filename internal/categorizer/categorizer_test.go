package categorizer_test

import (
	"context"
	"os"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// MockAIClient implements the AIClient interface for testing purposes.
type MockAIClient struct {
	CategorizeFunc func(ctx context.Context, transaction models.Transaction) (models.Transaction, error)
}

func (m *MockAIClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	return m.CategorizeFunc(ctx, transaction)
}

func TestCategorizer_CategorizeTransaction(t *testing.T) {
	// Setup a mock AI client
	mockAIClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
			// Simulate AI categorization logic
			if transaction.Description == "Coffee Shop" {
				transaction.Category = "Food & Drink"
			} else if transaction.Description == "Online Store" {
				transaction.Category = "Shopping"
			} else {
				transaction.Category = "Uncategorized (AI)"
			}
			return transaction, nil
		},
	}

	// Create a mock store for testing
	mockStore := &store.CategoryStore{
		CategoriesFile: "testdata/categories.yaml",
		CreditorsFile:  "testdata/creditors.yaml",
		DebtorsFile:    "testdata/debtors.yaml",
	}

	// Create a new logger for the categorizer
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

	// Instantiate the categorizer with the mock AI client and mock store
	categorizerInstance := categorizer.NewCategorizer(mockAIClient, mockStore, logging.NewLogrusAdapterFromLogger(logger))

	// Test cases
	testCases := []struct {
		name             string
		transaction      categorizer.Transaction
		expectedCategory string
		expectError      bool
	}{
		{
			name: "Direct Mapping - Creditor",
			transaction: categorizer.Transaction{
				PartyName:   "Starbucks Coffee",
				IsDebtor:    false,
				Amount:      "5.75",
				Date:        "2023-01-01",
				Info:        "Coffee purchase",
				Description: "Starbucks Coffee",
			},
			expectedCategory: "Restaurants",
			expectError:      false,
		},
		{
			name: "Keyword Mapping - Creditor",
			transaction: categorizer.Transaction{
				PartyName:   "MIGROS",
				IsDebtor:    false,
				Amount:      "87.32",
				Date:        "2023-01-02",
				Info:        "Weekly groceries",
				Description: "MIGROS",
			},
			expectedCategory: "Alimentation",
			expectError:      false,
		},
		{
			name: "Keyword Match - Coffee Shop",
			transaction: categorizer.Transaction{
				PartyName:   "New Coffee Place",
				IsDebtor:    false,
				Amount:      "12.50",
				Date:        "2023-01-03",
				Info:        "Coffee Shop",
				Description: "Coffee Shop",
			},
			expectedCategory: "Food", // From keyword strategy (coffee keyword)
			expectError:      false,
		},
		{
			name: "AI Fallback - Online Store",
			transaction: categorizer.Transaction{
				PartyName:   "Generic Online Store",
				IsDebtor:    false,
				Amount:      "50.00",
				Date:        "2023-01-04",
				Info:        "Online Store",
				Description: "Online Store",
			},
			expectedCategory: "Shopping", // From mock AI
			expectError:      false,
		},
		{
			name: "AI Fallback - Uncategorized",
			transaction: categorizer.Transaction{
				PartyName:   "Truly Unknown",
				IsDebtor:    false,
				Amount:      "20.00",
				Date:        "2023-01-05",
				Info:        "Random expense",
				Description: "Random expense",
			},
			expectedCategory: "Uncategorized (AI)", // From mock AI
			expectError:      false,
		},
		{
			name: "Empty Party Name",
			transaction: categorizer.Transaction{
				PartyName:   "",
				IsDebtor:    false,
				Amount:      "10.00",
				Date:        "2023-01-06",
				Info:        "Empty party",
				Description: "Empty party",
			},
			expectedCategory: models.CategoryUncategorized,
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			category, err := categorizerInstance.CategorizeTransaction(tc.transaction)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCategory, category.Name)
			}
		})
	}
}
