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

func TestCategorizer_StrategyOrchestration(t *testing.T) {
	tests := []struct {
		name             string
		transaction      Transaction
		creditorMappings map[string]string
		debtorMappings   map[string]string
		categories       []models.CategoryConfig
		aiResponse       string
		expectedCategory string
		expectedStrategy string // Which strategy should succeed
	}{
		{
			name: "direct mapping strategy wins - creditor",
			transaction: Transaction{
				PartyName: "COOP Store",
				IsDebtor:  false,
				Amount:    "25.50",
				Date:      "2025-01-15",
			},
			creditorMappings: map[string]string{
				"coop store": models.CategoryGroceries,
			},
			debtorMappings:   map[string]string{},
			categories:       []models.CategoryConfig{},
			expectedCategory: models.CategoryGroceries,
			expectedStrategy: "DirectMapping",
		},
		{
			name: "direct mapping strategy wins - debtor",
			transaction: Transaction{
				PartyName: "John Doe",
				IsDebtor:  true,
				Amount:    "2500.00",
				Date:      "2025-01-15",
			},
			creditorMappings: map[string]string{},
			debtorMappings: map[string]string{
				"john doe": models.CategorySalary,
			},
			categories:       []models.CategoryConfig{},
			expectedCategory: models.CategorySalary,
			expectedStrategy: "DirectMapping",
		},
		{
			name: "keyword strategy wins - YAML config",
			transaction: Transaction{
				PartyName: "Local Supermarket",
				IsDebtor:  false,
				Amount:    "45.20",
				Date:      "2025-01-15",
				Info:      "MIGROS purchase",
			},
			creditorMappings: map[string]string{},
			debtorMappings:   map[string]string{},
			categories: []models.CategoryConfig{
				{
					Name:     models.CategoryGroceries,
					Keywords: []string{"MIGROS", "COOP"},
				},
			},
			expectedCategory: models.CategoryGroceries,
			expectedStrategy: "Keyword",
		},
		{
			name: "keyword strategy wins - hardcoded patterns",
			transaction: Transaction{
				PartyName: "SBB CFF FFS",
				IsDebtor:  true,
				Amount:    "12.40",
				Date:      "2025-01-15",
				Info:      "Train ticket",
			},
			creditorMappings: map[string]string{},
			debtorMappings:   map[string]string{},
			categories:       []models.CategoryConfig{},
			expectedCategory: models.CategoryTransport,
			expectedStrategy: "Keyword",
		},
		{
			name: "AI strategy wins",
			transaction: Transaction{
				PartyName: "Unknown Coffee Shop",
				IsDebtor:  true,
				Amount:    "4.50",
				Date:      "2025-01-15",
				Info:      "Coffee purchase",
			},
			creditorMappings: map[string]string{},
			debtorMappings:   map[string]string{},
			categories:       []models.CategoryConfig{},
			aiResponse:       models.CategoryRestaurants,
			expectedCategory: models.CategoryRestaurants,
			expectedStrategy: "AI",
		},
		{
			name: "no strategy succeeds - uncategorized",
			transaction: Transaction{
				PartyName: "Unknown Business",
				IsDebtor:  true,
				Amount:    "100.00",
				Date:      "2025-01-15",
				Info:      "Unknown transaction",
			},
			creditorMappings: map[string]string{},
			debtorMappings:   map[string]string{},
			categories:       []models.CategoryConfig{},
			aiResponse:       models.CategoryUncategorized,
			expectedCategory: models.CategoryUncategorized,
			expectedStrategy: "", // No strategy succeeds
		},
		{
			name: "empty party name - uncategorized",
			transaction: Transaction{
				PartyName: "",
				IsDebtor:  true,
				Amount:    "50.00",
				Date:      "2025-01-15",
			},
			creditorMappings: map[string]string{},
			debtorMappings:   map[string]string{},
			categories:       []models.CategoryConfig{},
			expectedCategory: models.CategoryUncategorized,
			expectedStrategy: "", // No strategy is tried
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock store
			mockStore := &store.MockCategoryStore{
				Categories:       tt.categories,
				CreditorMappings: tt.creditorMappings,
				DebtorMappings:   tt.debtorMappings,
			}

			// Create mock AI client
			var mockAIClient AIClient
			if tt.aiResponse != "" {
				mockAIClient = &MockAIClient{
					CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
						tx.Category = tt.aiResponse
						return tx, nil
					},
				}
			}

			// Create mock logger
			mockLogger := &logging.MockLogger{}

			// Create categorizer
			categorizer := NewCategorizer(mockAIClient, mockStore, mockLogger)

			// Execute
			category, err := categorizer.CategorizeTransaction(context.Background(), tt.transaction)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCategory, category.Name)

			// Verify logging based on expected strategy
			if tt.expectedStrategy != "" {
				logEntries := mockLogger.GetEntriesByLevel("DEBUG")
				var found bool
				for _, entry := range logEntries {
					// Check if the expected strategy succeeded
					if entry.Message == "Transaction categorized successfully" {
						for _, field := range entry.Fields {
							if field.Key == "strategy" && field.Value == tt.expectedStrategy {
								found = true
								break
							}
						}
					}
					if found {
						break
					}
				}
				assert.True(t, found, "Expected strategy %s to log success message", tt.expectedStrategy)
			}
		})
	}
}

func TestCategorizer_StrategyPriority(t *testing.T) {
	// Test that strategies are tried in the correct priority order
	// DirectMapping > Keyword > AI

	// Create a transaction that could match multiple strategies
	transaction := Transaction{
		PartyName: "COOP Restaurant",
		IsDebtor:  false,
		Amount:    "30.00",
		Date:      "2025-01-15",
		Info:      "Dinner",
	}

	// Set up data so multiple strategies could match
	mockStore := &store.MockCategoryStore{
		Categories: []models.CategoryConfig{
			{
				Name:     models.CategoryRestaurants,
				Keywords: []string{"RESTAURANT"},
			},
		},
		CreditorMappings: map[string]string{
			"coop restaurant": models.CategoryGroceries, // Direct mapping should win
		},
		DebtorMappings: map[string]string{},
	}

	// AI would return a different category
	mockAIClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
			tx.Category = models.CategoryShopping
			return tx, nil
		},
	}

	mockLogger := &logging.MockLogger{}

	// Create categorizer
	categorizer := NewCategorizer(mockAIClient, mockStore, mockLogger)

	// Execute
	category, err := categorizer.CategorizeTransaction(context.Background(), transaction)

	// Assert
	require.NoError(t, err)
	// DirectMapping should win over Keyword and AI
	assert.Equal(t, models.CategoryGroceries, category.Name)

	// Verify that DirectMapping strategy was used
	logEntries := mockLogger.GetEntriesByLevel("DEBUG")
	var foundDirectMapping bool
	for _, entry := range logEntries {
		if entry.Message == "Transaction categorized successfully" {
			for _, field := range entry.Fields {
				if field.Key == "strategy" && field.Value == "DirectMapping" {
					foundDirectMapping = true
					break
				}
			}
		}
		if foundDirectMapping {
			break
		}
	}
	assert.True(t, foundDirectMapping, "Expected DirectMapping strategy to log success message")
}

func TestCategorizer_StrategyErrorHandling(t *testing.T) {
	// Test that if one strategy fails, the next one is tried

	// Create a mock store that will cause DirectMapping to fail
	mockStore := &store.MockCategoryStore{
		LoadCreditorMappingsError: assert.AnError,
		LoadDebtorMappingsError:   assert.AnError,
		Categories: []models.CategoryConfig{
			{
				Name:     models.CategoryGroceries,
				Keywords: []string{"COOP"},
			},
		},
	}

	// AI client that works
	mockAIClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
			tx.Category = models.CategoryShopping
			return tx, nil
		},
	}

	mockLogger := &logging.MockLogger{}

	// Create categorizer
	categorizer := NewCategorizer(mockAIClient, mockStore, mockLogger)

	// Test transaction
	transaction := Transaction{
		PartyName: "COOP Store",
		IsDebtor:  false,
		Amount:    "25.00",
		Date:      "2025-01-15",
	}

	// Execute
	category, err := categorizer.CategorizeTransaction(context.Background(), transaction)

	// Assert
	require.NoError(t, err)
	// Should fall back to Keyword strategy (which should work with COOP)
	assert.Equal(t, models.CategoryGroceries, category.Name)

	// Verify that Keyword strategy was used (DirectMapping should not have matched)
	logEntries := mockLogger.GetEntriesByLevel("DEBUG")
	var foundKeywordSuccess bool
	var foundDirectMappingNoMatch bool
	for _, entry := range logEntries {
		// Check for Keyword strategy success
		if entry.Message == "Transaction categorized successfully" {
			for _, field := range entry.Fields {
				if field.Key == "strategy" && field.Value == "Keyword" {
					foundKeywordSuccess = true
					break
				}
			}
		}
		// Check that DirectMapping was tried but didn't find a match
		if entry.Message == "Strategy did not find a match" {
			for _, field := range entry.Fields {
				if field.Key == "strategy" && field.Value == "DirectMapping" {
					foundDirectMappingNoMatch = true
					break
				}
			}
		}
	}
	assert.True(t, foundKeywordSuccess, "Expected Keyword strategy to log success message")
	assert.True(t, foundDirectMappingNoMatch, "Expected DirectMapping strategy to try and not find a match")
}

func TestCategorizer_BackwardCompatibility(t *testing.T) {
	// Test that the new strategy-based categorizer produces the same results
	// as the old implementation for common scenarios

	testCases := []struct {
		name        string
		transaction Transaction
		setupStore  func() *store.MockCategoryStore
		setupAI     func() AIClient
		expected    string
	}{
		{
			name: "creditor mapping",
			transaction: Transaction{
				PartyName: "MIGROS",
				IsDebtor:  false,
			},
			setupStore: func() *store.MockCategoryStore {
				return &store.MockCategoryStore{
					CreditorMappings: map[string]string{
						"migros": models.CategoryGroceries,
					},
				}
			},
			setupAI:  func() AIClient { return nil },
			expected: models.CategoryGroceries,
		},
		{
			name: "debtor mapping",
			transaction: Transaction{
				PartyName: "Employer Corp",
				IsDebtor:  true,
			},
			setupStore: func() *store.MockCategoryStore {
				return &store.MockCategoryStore{
					DebtorMappings: map[string]string{
						"employer corp": models.CategorySalary,
					},
				}
			},
			setupAI:  func() AIClient { return nil },
			expected: models.CategorySalary,
		},
		{
			name: "keyword matching",
			transaction: Transaction{
				PartyName: "SBB Ticket",
				IsDebtor:  true,
				Info:      "Train fare",
			},
			setupStore: func() *store.MockCategoryStore {
				return &store.MockCategoryStore{}
			},
			setupAI:  func() AIClient { return nil },
			expected: models.CategoryTransport,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := tc.setupStore()
			mockAI := tc.setupAI()
			mockLogger := &logging.MockLogger{}

			categorizer := NewCategorizer(mockAI, mockStore, mockLogger)

			category, err := categorizer.CategorizeTransaction(context.Background(), tc.transaction)

			require.NoError(t, err)
			assert.Equal(t, tc.expected, category.Name)
		})
	}
}
