package categorizer_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAIClient implements the AIClient interface for testing purposes.
type MockAIClient struct {
	CategorizeFunc   func(ctx context.Context, transaction models.Transaction) (models.Transaction, error)
	GetEmbeddingFunc func(ctx context.Context, text string) ([]float32, error)
}

func (m *MockAIClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	return m.CategorizeFunc(ctx, transaction)
}

func (m *MockAIClient) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.GetEmbeddingFunc != nil {
		return m.GetEmbeddingFunc(ctx, text)
	}
	return []float32{0.0, 0.0}, nil
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
			expectedCategory: "Groceries",
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
			category, err := categorizerInstance.CategorizeTransaction(context.Background(), tc.transaction)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCategory, category.Name)
			}
		})
	}
}

// Integration tests for strategy pattern categorization
func TestCategorizer_StrategyPatternIntegration(t *testing.T) {
	// Create test data files
	tempDir := t.TempDir()

	// Create categories.yaml
	categoriesContent := `categories:
  - name: "Food"
    keywords: ["coffee", "restaurant", "food", "grocery"]
  - name: "Transport"
    keywords: ["bus", "train", "taxi", "transport"]
  - name: "Shopping"
    keywords: ["shop", "store", "mall"]`

	categoriesFile := filepath.Join(tempDir, "categories.yaml")
	err := os.WriteFile(categoriesFile, []byte(categoriesContent), 0600)
	require.NoError(t, err)

	// Create creditors.yaml
	creditorsContent := `STARBUCKS: "Food"
MIGROS: "Food"
SBB: "Transport"`

	creditorsFile := filepath.Join(tempDir, "creditors.yaml")
	err = os.WriteFile(creditorsFile, []byte(creditorsContent), 0600)
	require.NoError(t, err)

	// Create debtors.yaml
	debtorsContent := `COOP: "Food"
UBER: "Transport"`

	debtorsFile := filepath.Join(tempDir, "debtors.yaml")
	err = os.WriteFile(debtorsFile, []byte(debtorsContent), 0600)
	require.NoError(t, err)

	// Create store with test files
	categoryStore := &store.CategoryStore{
		CategoriesFile: categoriesFile,
		CreditorsFile:  creditorsFile,
		DebtorsFile:    debtorsFile,
	}

	// Mock AI client that categorizes unknown transactions
	mockAIClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
			if strings.Contains(strings.ToLower(transaction.Description), "unknown") {
				transaction.Category = "AI_Category"
			} else {
				transaction.Category = models.CategoryUncategorized
			}
			return transaction, nil
		},
	}

	// Create logger
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

	// Create categorizer with all strategies
	cat := categorizer.NewCategorizer(mockAIClient, categoryStore, logging.NewLogrusAdapterFromLogger(logger))

	testCases := []struct {
		name             string
		transaction      categorizer.Transaction
		expectedCategory string
		expectedStrategy string // Which strategy should handle it
	}{
		{
			name: "Direct mapping - exact creditor match",
			transaction: categorizer.Transaction{
				PartyName:   "STARBUCKS",
				IsDebtor:    false,
				Amount:      "5.50",
				Description: "Coffee purchase",
			},
			expectedCategory: "Food",
			expectedStrategy: "DirectMapping",
		},
		{
			name: "Direct mapping - exact debtor match",
			transaction: categorizer.Transaction{
				PartyName:   "COOP",
				IsDebtor:    true,
				Amount:      "25.80",
				Description: "Grocery shopping",
			},
			expectedCategory: "Food", // Direct mapping takes priority
			expectedStrategy: "DirectMapping",
		},
		{
			name: "Keyword strategy - coffee keyword",
			transaction: categorizer.Transaction{
				PartyName:   "Local Coffee Shop",
				IsDebtor:    false,
				Amount:      "4.20",
				Description: "Morning coffee",
			},
			expectedCategory: "Food",
			expectedStrategy: "Keyword",
		},
		{
			name: "Keyword strategy - transport keyword",
			transaction: categorizer.Transaction{
				PartyName:   "City Transport",
				IsDebtor:    false,
				Amount:      "3.60",
				Description: "Bus ticket",
			},
			expectedCategory: "Transport",
			expectedStrategy: "Keyword",
		},
		{
			name: "AI strategy fallback",
			transaction: categorizer.Transaction{
				PartyName:   "Unknown Merchant",
				IsDebtor:    false,
				Amount:      "15.00",
				Description: "Unknown purchase",
			},
			expectedCategory: "AI_Category",
			expectedStrategy: "AI",
		},
		{
			name: "Uncategorized fallback",
			transaction: categorizer.Transaction{
				PartyName:   "Completely Unknown",
				IsDebtor:    false,
				Amount:      "10.00",
				Description: "Random transaction",
			},
			expectedCategory: models.CategoryUncategorized,
			expectedStrategy: "None",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			category, err := cat.CategorizeTransaction(context.Background(), tc.transaction)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCategory, category.Name)
		})
	}
}

// Test strategy priority and fallback behavior
func TestCategorizer_StrategyPriority(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files with overlapping data
	categoriesContent := `categories:
  - name: "KeywordFood"
    keywords: ["starbucks"]`

	categoriesFile := filepath.Join(tempDir, "categories.yaml")
	err := os.WriteFile(categoriesFile, []byte(categoriesContent), 0600)
	require.NoError(t, err)

	// Direct mapping should take priority over keyword matching
	creditorsContent := `"STARBUCKS": "DirectMappingFood"`

	creditorsFile := filepath.Join(tempDir, "creditors.yaml")
	err = os.WriteFile(creditorsFile, []byte(creditorsContent), 0600)
	require.NoError(t, err)

	debtorsFile := filepath.Join(tempDir, "debtors.yaml")
	err = os.WriteFile(debtorsFile, []byte("{}"), 0600)
	require.NoError(t, err)

	categoryStore := &store.CategoryStore{
		CategoriesFile: categoriesFile,
		CreditorsFile:  creditorsFile,
		DebtorsFile:    debtorsFile,
	}

	mockAIClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
			transaction.Category = "AIFood"
			return transaction, nil
		},
	}

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

	cat := categorizer.NewCategorizer(mockAIClient, categoryStore, logging.NewLogrusAdapterFromLogger(logger))

	// Transaction that matches both direct mapping and keyword
	transaction := categorizer.Transaction{
		PartyName:   "STARBUCKS",
		IsDebtor:    false,
		Amount:      "5.50",
		Description: "Coffee with starbucks keyword",
	}

	category, err := cat.CategorizeTransaction(context.Background(), transaction)

	assert.NoError(t, err)
	// Direct mapping should win over keyword matching
	assert.Equal(t, "DirectMappingFood", category.Name)
}

// Test error handling in strategy pattern
func TestCategorizer_StrategyErrorHandling(t *testing.T) {
	// Create categorizer with failing AI client
	failingAIClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
			return models.Transaction{}, fmt.Errorf("AI service unavailable")
		},
	}

	// Empty store
	emptyStore := &store.CategoryStore{
		CategoriesFile: "/non/existent/categories.yaml",
		CreditorsFile:  "/non/existent/creditors.yaml",
		DebtorsFile:    "/non/existent/debtors.yaml",
	}

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

	cat := categorizer.NewCategorizer(failingAIClient, emptyStore, logging.NewLogrusAdapterFromLogger(logger))

	transaction := categorizer.Transaction{
		PartyName:   "Unknown Merchant",
		IsDebtor:    false,
		Amount:      "10.00",
		Description: "Unknown transaction",
	}

	// Should not fail even if AI client fails
	category, err := cat.CategorizeTransaction(context.Background(), transaction)

	assert.NoError(t, err)
	assert.Equal(t, models.CategoryUncategorized, category.Name)
}

// Test concurrent categorization
func TestCategorizer_ConcurrentCategorization(t *testing.T) {
	tempDir := t.TempDir()

	categoriesContent := `categories:
  - name: "Food"
    keywords: ["food", "restaurant"]`

	categoriesFile := filepath.Join(tempDir, "categories.yaml")
	err := os.WriteFile(categoriesFile, []byte(categoriesContent), 0600)
	require.NoError(t, err)

	creditorsFile := filepath.Join(tempDir, "creditors.yaml")
	err = os.WriteFile(creditorsFile, []byte("{}"), 0600)
	require.NoError(t, err)

	debtorsFile := filepath.Join(tempDir, "debtors.yaml")
	err = os.WriteFile(debtorsFile, []byte("{}"), 0600)
	require.NoError(t, err)

	categoryStore := &store.CategoryStore{
		CategoriesFile: categoriesFile,
		CreditorsFile:  creditorsFile,
		DebtorsFile:    debtorsFile,
	}

	mockAIClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			transaction.Category = "AI_Category"
			return transaction, nil
		},
	}

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

	cat := categorizer.NewCategorizer(mockAIClient, categoryStore, logging.NewLogrusAdapterFromLogger(logger))

	// Run multiple categorizations concurrently
	const numGoroutines = 10
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			transaction := categorizer.Transaction{
				PartyName:   fmt.Sprintf("Food Store %d", id),
				IsDebtor:    false,
				Amount:      "10.00",
				Description: "food purchase",
			}

			category, err := cat.CategorizeTransaction(context.Background(), transaction)
			if err != nil {
				results <- fmt.Sprintf("ERROR: %v", err)
			} else {
				results <- category.Name
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		assert.Equal(t, "Food", result) // Should all be categorized as Food by keyword strategy
	}
}
