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

func TestKeywordStrategy_Name(t *testing.T) {
	strategy := &KeywordStrategy{}
	assert.Equal(t, "Keyword", strategy.Name())
}

func TestKeywordStrategy_Categorize(t *testing.T) {
	tests := []struct {
		name             string
		transaction      Transaction
		categories       []models.CategoryConfig
		expectedCategory string
		expectedFound    bool
		expectedError    bool
	}{
		{
			name: "keyword match in party name",
			transaction: Transaction{
				PartyName: "COOP Supermarket",
				IsDebtor:  false,
				Info:      "Purchase",
			},
			categories: []models.CategoryConfig{
				{
					Name:     models.CategoryGroceries,
					Keywords: []string{"COOP", "MIGROS"},
				},
			},
			expectedCategory: models.CategoryGroceries,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "keyword match in description",
			transaction: Transaction{
				PartyName: "Store ABC",
				IsDebtor:  false,
				Info:      "COOP purchase",
			},
			categories: []models.CategoryConfig{
				{
					Name:     models.CategoryGroceries,
					Keywords: []string{"COOP", "MIGROS"},
				},
			},
			expectedCategory: models.CategoryGroceries,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "case insensitive matching",
			transaction: Transaction{
				PartyName: "coop supermarket",
				IsDebtor:  false,
				Info:      "Purchase",
			},
			categories: []models.CategoryConfig{
				{
					Name:     models.CategoryGroceries,
					Keywords: []string{"COOP", "MIGROS"},
				},
			},
			expectedCategory: models.CategoryGroceries,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "multiple categories - first match wins",
			transaction: Transaction{
				PartyName: "COOP Restaurant",
				IsDebtor:  false,
				Info:      "Purchase",
			},
			categories: []models.CategoryConfig{
				{
					Name:     models.CategoryGroceries,
					Keywords: []string{"COOP"},
				},
				{
					Name:     models.CategoryRestaurants,
					Keywords: []string{"RESTAURANT"},
				},
			},
			expectedCategory: models.CategoryGroceries, // First match wins
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "no keyword match",
			transaction: Transaction{
				PartyName: "Unknown Store",
				IsDebtor:  false,
				Info:      "Purchase",
			},
			categories: []models.CategoryConfig{
				{
					Name:     models.CategoryGroceries,
					Keywords: []string{"COOP", "MIGROS"},
				},
			},
			expectedFound: false,
			expectedError: false,
		},
		{
			name: "empty party name",
			transaction: Transaction{
				PartyName: "",
				IsDebtor:  false,
				Info:      "COOP purchase",
			},
			categories: []models.CategoryConfig{
				{
					Name:     models.CategoryGroceries,
					Keywords: []string{"COOP"},
				},
			},
			expectedFound: false,
			expectedError: false,
		},
		{
			name: "hardcoded pattern match - supermarket",
			transaction: Transaction{
				PartyName: "MIGROS Store",
				IsDebtor:  false,
				Info:      "Purchase",
			},
			categories: []models.CategoryConfig{}, // No YAML categories
			expectedCategory: models.CategoryGroceries,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "hardcoded pattern match - bank code",
			transaction: Transaction{
				PartyName: "Card Payment",
				IsDebtor:  true,
				Info:      "PMT " + models.BankCodePOS + " transaction",
			},
			categories:       []models.CategoryConfig{}, // No YAML categories
			expectedCategory: models.CategoryShopping,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "hardcoded pattern match - unknown payee card payment",
			transaction: Transaction{
				PartyName: "UNKNOWN PAYEE",
				IsDebtor:  true,
				Info:      "PMT CARTE 12345",
			},
			categories:       []models.CategoryConfig{}, // No YAML categories
			expectedCategory: models.CategoryShopping,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "no match in hardcoded patterns",
			transaction: Transaction{
				PartyName: "Random Store",
				IsDebtor:  false,
				Info:      "Random transaction",
			},
			categories: []models.CategoryConfig{}, // No YAML categories
			expectedFound: false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock store
			mockStore := &store.MockCategoryStore{
				Categories: tt.categories,
			}

			// Create mock logger
			mockLogger := &logging.MockLogger{}

			// Create strategy
			strategy := NewKeywordStrategy(mockStore, mockLogger)

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
		})
	}
}

func TestKeywordStrategy_ReloadCategories(t *testing.T) {
	// Create mock store with initial categories
	mockStore := &store.MockCategoryStore{
		Categories: []models.CategoryConfig{
			{
				Name:     "Initial Category",
				Keywords: []string{"INITIAL"},
			},
		},
	}

	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create strategy
	strategy := NewKeywordStrategy(mockStore, mockLogger)

	// Verify initial category works
	ctx := context.Background()
	transaction := Transaction{
		PartyName: "INITIAL Store",
		IsDebtor:  false,
		Info:      "Purchase",
	}

	category, found, err := strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Initial Category", category.Name)

	// Update the mock store with new categories
	mockStore.Categories = []models.CategoryConfig{
		{
			Name:     "Updated Category",
			Keywords: []string{"UPDATED"},
		},
	}

	// Reload categories
	strategy.ReloadCategories()

	// Verify old category no longer works
	category, found, err = strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.False(t, found) // Should not find with old keyword

	// Verify new category works
	transaction = Transaction{
		PartyName: "UPDATED Store",
		IsDebtor:  false,
		Info:      "Purchase",
	}

	category, found, err = strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Updated Category", category.Name)
}

func TestKeywordStrategy_HardcodedPatterns(t *testing.T) {
	// Test some specific hardcoded patterns to ensure they work
	tests := []struct {
		name             string
		transaction      Transaction
		expectedCategory string
		expectedFound    bool
	}{
		{
			name: "SBB transport",
			transaction: Transaction{
				PartyName: "SBB CFF FFS",
				IsDebtor:  true,
				Info:      "Train ticket",
			},
			expectedCategory: models.CategoryTransport,
			expectedFound:    true,
		},
		{
			name: "ATM withdrawal",
			transaction: Transaction{
				PartyName: "ATM Machine",
				IsDebtor:  true,
				Info:      "Cash withdrawal",
			},
			expectedCategory: models.CategoryWithdrawals,
			expectedFound:    true,
		},
		{
			name: "Restaurant",
			transaction: Transaction{
				PartyName: "PIZZERIA Mario",
				IsDebtor:  true,
				Info:      "Dinner",
			},
			expectedCategory: models.CategoryRestaurants,
			expectedFound:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock store with no categories (to test hardcoded patterns)
			mockStore := &store.MockCategoryStore{
				Categories: []models.CategoryConfig{},
			}

			// Create mock logger
			mockLogger := &logging.MockLogger{}

			// Create strategy
			strategy := NewKeywordStrategy(mockStore, mockLogger)

			// Execute
			ctx := context.Background()
			category, found, err := strategy.Categorize(ctx, tt.transaction)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFound, found)

			if tt.expectedFound {
				assert.Equal(t, tt.expectedCategory, category.Name)
				assert.NotEmpty(t, category.Description)
			}
		})
	}
}