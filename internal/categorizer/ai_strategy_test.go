package categorizer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockAIClient is a mock implementation of AIClient for testing with additional tracking.
type TestMockAIClient struct {
	CategorizeFunc  func(ctx context.Context, tx models.Transaction) (models.Transaction, error)
	CallCount       int
	LastTransaction models.Transaction
}

func (m *TestMockAIClient) Categorize(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
	m.CallCount++
	m.LastTransaction = tx
	
	if m.CategorizeFunc != nil {
		return m.CategorizeFunc(ctx, tx)
	}
	
	// Default behavior: return transaction with a test category
	tx.Category = "AI Test Category"
	return tx, nil
}

func TestAIStrategy_Name(t *testing.T) {
	strategy := &AIStrategy{}
	assert.Equal(t, "AI", strategy.Name())
}

func TestAIStrategy_Categorize(t *testing.T) {
	tests := []struct {
		name             string
		transaction      Transaction
		aiClient         *TestMockAIClient
		expectedCategory string
		expectedFound    bool
		expectedError    bool
	}{
		{
			name: "successful AI categorization",
			transaction: Transaction{
				PartyName:   "Coffee Shop",
				IsDebtor:    true,
				Amount:      "5.50",
				Date:        "2025-01-15",
				Info:        "Morning coffee",
				Description: "Coffee purchase",
			},
			aiClient: &TestMockAIClient{
				CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
					tx.Category = models.CategoryRestaurants
					return tx, nil
				},
			},
			expectedCategory: models.CategoryRestaurants,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "AI returns uncategorized",
			transaction: Transaction{
				PartyName: "Unknown Store",
				IsDebtor:  true,
			},
			aiClient: &TestMockAIClient{
				CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
					tx.Category = models.CategoryUncategorized
					return tx, nil
				},
			},
			expectedFound: false,
			expectedError: false,
		},
		{
			name: "AI returns empty category",
			transaction: Transaction{
				PartyName: "Unknown Store",
				IsDebtor:  true,
			},
			aiClient: &TestMockAIClient{
				CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
					tx.Category = ""
					return tx, nil
				},
			},
			expectedFound: false,
			expectedError: false,
		},
		{
			name: "AI client error",
			transaction: Transaction{
				PartyName: "Test Store",
				IsDebtor:  true,
			},
			aiClient: &TestMockAIClient{
				CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
					return models.Transaction{}, errors.New("AI service unavailable")
				},
			},
			expectedFound: false,
			expectedError: false, // Strategy handles errors gracefully
		},
		{
			name: "empty party name",
			transaction: Transaction{
				PartyName: "",
				IsDebtor:  true,
			},
			aiClient: &TestMockAIClient{
				CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
					tx.Category = models.CategoryShopping
					return tx, nil
				},
			},
			expectedFound: false,
			expectedError: false,
		},
		{
			name: "whitespace only party name",
			transaction: Transaction{
				PartyName: "   ",
				IsDebtor:  true,
			},
			aiClient: &TestMockAIClient{
				CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
					tx.Category = models.CategoryShopping
					return tx, nil
				},
			},
			expectedFound: false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock logger
			mockLogger := &logging.MockLogger{}

			// Create strategy
			strategy := NewAIStrategy(tt.aiClient, mockLogger)

			// Execute
			ctx := context.Background()
			category, found, err := strategy.Categorize(ctx, tt.transaction)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedFound, found)

			if tt.expectedFound {
				assert.Equal(t, tt.expectedCategory, category.Name)
				assert.NotEmpty(t, category.Description)
			}

			// Verify AI client was called appropriately
			if tt.transaction.PartyName != "" && strings.TrimSpace(tt.transaction.PartyName) != "" {
				assert.Equal(t, 1, tt.aiClient.CallCount, "AI client should be called once for valid transactions")
			}
		})
	}
}

func TestAIStrategy_NoAIClient(t *testing.T) {
	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create strategy with no AI client
	strategy := NewAIStrategy(nil, mockLogger)

	// Test transaction
	transaction := Transaction{
		PartyName: "Test Store",
		IsDebtor:  true,
	}

	// Execute
	ctx := context.Background()
	category, found, err := strategy.Categorize(ctx, transaction)

	// Assert
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, category.Name)
}

func TestAIStrategy_ConvertToModelTransaction(t *testing.T) {
	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create mock AI client
	mockAIClient := &TestMockAIClient{}

	// Create strategy
	strategy := NewAIStrategy(mockAIClient, mockLogger)

	tests := []struct {
		name        string
		transaction Transaction
		expectError bool
	}{
		{
			name: "complete transaction",
			transaction: Transaction{
				PartyName:   "Test Store",
				IsDebtor:    true,
				Amount:      "10.50",
				Date:        "2025-01-15",
				Info:        "Purchase info",
				Description: "Test purchase",
			},
			expectError: false,
		},
		{
			name: "transaction with invalid date",
			transaction: Transaction{
				PartyName:   "Test Store",
				IsDebtor:    true,
				Amount:      "10.50",
				Date:        "invalid-date",
				Info:        "Purchase info",
				Description: "Test purchase",
			},
			expectError: false, // Should handle gracefully
		},
		{
			name: "transaction with empty date",
			transaction: Transaction{
				PartyName:   "Test Store",
				IsDebtor:    true,
				Amount:      "10.50",
				Date:        "",
				Info:        "Purchase info",
				Description: "Test purchase",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute conversion
			modelTx, err := strategy.convertToModelTransaction(tt.transaction)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.transaction.PartyName, modelTx.PartyName)
				
				// Check description combination
				if tt.transaction.Info != "" && tt.transaction.Description != "" {
					assert.Contains(t, modelTx.Description, tt.transaction.Description)
					assert.Contains(t, modelTx.Description, tt.transaction.Info)
				} else if tt.transaction.Info != "" {
					assert.Equal(t, tt.transaction.Info, modelTx.Description)
				} else {
					assert.Equal(t, tt.transaction.Description, modelTx.Description)
				}
			}
		})
	}
}

func TestAIStrategy_Integration(t *testing.T) {
	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create mock AI client that simulates real behavior
	mockAIClient := &TestMockAIClient{
		CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
			// Simulate AI logic based on party name
			switch {
			case strings.Contains(strings.ToUpper(tx.PartyName), "COFFEE"):
				tx.Category = models.CategoryRestaurants
			case strings.Contains(strings.ToUpper(tx.PartyName), "SUPERMARKET"):
				tx.Category = models.CategoryGroceries
			case strings.Contains(strings.ToUpper(tx.PartyName), "GAS"):
				tx.Category = models.CategoryTransport
			default:
				tx.Category = models.CategoryUncategorized
			}
			return tx, nil
		},
	}

	// Create strategy
	strategy := NewAIStrategy(mockAIClient, mockLogger)

	// Test various transactions
	testCases := []struct {
		partyName        string
		expectedCategory string
		expectedFound    bool
	}{
		{"Coffee Shop Downtown", models.CategoryRestaurants, true},
		{"Big Supermarket Chain", models.CategoryGroceries, true},
		{"Gas Station 24/7", models.CategoryTransport, true},
		{"Unknown Business", models.CategoryUncategorized, false},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.partyName, func(t *testing.T) {
			transaction := Transaction{
				PartyName: tc.partyName,
				IsDebtor:  true,
				Amount:    "25.00",
				Date:      "2025-01-15",
			}

			category, found, err := strategy.Categorize(ctx, transaction)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedFound, found)
			
			if tc.expectedFound {
				assert.Equal(t, tc.expectedCategory, category.Name)
			}
		})
	}

	// Verify AI client was called for each test
	assert.Equal(t, len(testCases), mockAIClient.CallCount)
}