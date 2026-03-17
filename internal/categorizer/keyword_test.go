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
					Name:     "Courses",
					Keywords: []string{"COOP", "MIGROS"},
				},
			},
			expectedCategory: "Courses",
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
					Name:     "Courses",
					Keywords: []string{"COOP", "MIGROS"},
				},
			},
			expectedCategory: "Courses",
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
					Name:     "Courses",
					Keywords: []string{"COOP", "MIGROS"},
				},
			},
			expectedCategory: "Courses",
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
					Name:     "Courses",
					Keywords: []string{"COOP"},
				},
				{
					Name:     "Restaurants",
					Keywords: []string{"RESTAURANT"},
				},
			},
			expectedCategory: "Courses", // First match wins
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
					Name:     "Courses",
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
					Name:     "Courses",
					Keywords: []string{"COOP"},
				},
			},
			expectedFound: false,
			expectedError: false,
		},
		{
			name: "YAML keyword match - SBB transport",
			transaction: Transaction{
				PartyName: "SBB CFF FFS",
				IsDebtor:  true,
				Info:      "Train ticket",
			},
			categories: []models.CategoryConfig{
				{
					Name:     "Transports Publics",
					Keywords: []string{"sbb", "cff"},
				},
			},
			expectedCategory: "Transports Publics",
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "YAML keyword match - ATM withdrawal",
			transaction: Transaction{
				PartyName: "ATM Machine",
				IsDebtor:  true,
				Info:      "Cash withdrawal",
			},
			categories: []models.CategoryConfig{
				{
					Name:     "Divers",
					Keywords: []string{"atm", "retrait", "withdrawal"},
				},
			},
			expectedCategory: "Divers",
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "YAML keyword match - restaurant",
			transaction: Transaction{
				PartyName: "PIZZERIA Mario",
				IsDebtor:  true,
				Info:      "Dinner",
			},
			categories: []models.CategoryConfig{
				{
					Name:     "Restaurants",
					Keywords: []string{"restaurant", "pizzeria", "café"},
				},
			},
			expectedCategory: "Restaurants",
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "no match with empty categories",
			transaction: Transaction{
				PartyName: "Random Store",
				IsDebtor:  false,
				Info:      "Random transaction",
			},
			categories:    []models.CategoryConfig{},
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
			strategy := NewKeywordStrategy(mockStore.Categories, mockStore, mockLogger)

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
	strategy := NewKeywordStrategy(mockStore.Categories, mockStore, mockLogger)

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
