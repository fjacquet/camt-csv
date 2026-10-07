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

// A keyword is a word, not a fragment: "ai" must not fire inside "SAINT", nor
// "rc" inside "ARRCO", which filed supermarkets under Pension and Assurances.
func TestKeywordStrategy_MatchesWholeWordsOnly(t *testing.T) {
	cats := []models.CategoryConfig{
		{Name: "Pension", Keywords: []string{"ai", "caisse de pension"}},
		{Name: "Assurances", Keywords: []string{"rc", "assurance"}},
		{Name: "Loisirs", Keywords: []string{"b&b", "café"}},
	}
	tests := []struct {
		name  string
		party string
		info  string
		want  string // "" means no match
	}{
		{"short keyword inside a word", "CARREFOUR MARKET SAINT DOULCHARD", "", ""},
		{"short keyword inside another word", "KLESIA AGIRC ARRCO", "", ""},
		{"whole word at the start", "AI Fund", "", "Pension"},
		{"whole word at the end", "Fund AI", "", "Pension"},
		{"whole word in the middle", "Fund AI Inc", "", "Pension"},
		{"punctuation is a boundary", "(RC) cover", "", "Assurances"},
		{"hyphen is a boundary", "Cafe-Bar mon-café-du-lac", "", "Loisirs"},
		{"multi word phrase", "Paiement CAISSE DE PENSION SA", "", "Pension"},
		{"keyword with punctuation", "Hotel B&B Lausanne", "", "Loisirs"},
		{"digit glued to the keyword is not a boundary", "AI2024", "", ""},
		{"match in the description", "Store", "facture RC", "Assurances"},
		{"description has the same rule", "Store", "SAINT", ""},
		{"later occurrence is found after a false one", "SAINT AI", "", "Pension"},
		{"accented keyword, case-insensitive", "CAFÉ DU LAC", "", "Loisirs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewKeywordStrategy(cats, &store.MockCategoryStore{}, testLogger())
			got, found, err := s.Categorize(context.Background(), Transaction{PartyName: tt.party, Info: tt.info})
			require.NoError(t, err)
			assert.Equal(t, tt.want != "", found)
			assert.Equal(t, tt.want, got.Name)
		})
	}
}
