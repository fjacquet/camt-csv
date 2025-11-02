package categorizer

import (
	"context"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"
)

// BenchmarkKeywordStrategy_StringOperations benchmarks the optimized string operations
// in KeywordStrategy to demonstrate the performance improvements from using strings.Builder
func BenchmarkKeywordStrategy_StringOperations(b *testing.B) {
	// Create mock store with categories
	mockStore := &store.MockCategoryStore{
		Categories: []models.CategoryConfig{
			{
				Name:     models.CategoryGroceries,
				Keywords: []string{"COOP", "MIGROS", "ALDI", "LIDL", "DENNER"},
			},
			{
				Name:     models.CategoryRestaurants,
				Keywords: []string{"PIZZERIA", "CAFE", "RESTAURANT", "SUSHI", "KEBAB"},
			},
			{
				Name:     models.CategoryTransport,
				Keywords: []string{"SBB", "CFF", "MOBILITY", "PAYBYPHONE"},
			},
		},
	}

	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create strategy
	strategy := NewKeywordStrategy(mockStore, mockLogger)

	// Test transaction with long party name and description to stress string operations
	transaction := Transaction{
		PartyName: "COOP SUPERMARKET CHAIN STORE LOCATION DOWNTOWN SHOPPING CENTER",
		IsDebtor:  true,
		Info:      "PURCHASE AT GROCERY STORE WITH MULTIPLE ITEMS AND LONG DESCRIPTION TEXT",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, _, err := strategy.Categorize(ctx, transaction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDirectMappingStrategy_StringOperations benchmarks the optimized string operations
// in DirectMappingStrategy to demonstrate the performance improvements from using strings.Builder
func BenchmarkDirectMappingStrategy_StringOperations(b *testing.B) {
	// Create mock store with mappings
	mockStore := &store.MockCategoryStore{
		CreditorMappings: map[string]string{
			"starbucks coffee":     models.CategoryRestaurants,
			"migros supermarket":   models.CategoryGroceries,
			"shell gas station":    "Transport",
			"amazon online store":  models.CategoryShopping,
			"netflix subscription": "Entertainment",
		},
		DebtorMappings: map[string]string{
			"john doe salary":       models.CategorySalary,
			"rent payment monthly":  "Housing",
			"insurance premium":     "Insurance",
			"loan payment bank":     "Financial",
			"tax refund government": "Government",
		},
	}

	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create strategy
	strategy := NewDirectMappingStrategy(mockStore, mockLogger)

	// Test transaction with long party name to stress string normalization
	transaction := Transaction{
		PartyName: "STARBUCKS COFFEE DOWNTOWN LOCATION WITH VERY LONG NAME",
		IsDebtor:  false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, _, err := strategy.Categorize(ctx, transaction)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStringNormalization_Comparison compares the old vs new string normalization approaches
func BenchmarkStringNormalization_Comparison(b *testing.B) {
	testString := "VERY LONG PARTY NAME WITH MULTIPLE WORDS AND SPECIAL CHARACTERS 123"

	b.Run("Old_ToLower", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Old approach: direct strings.ToLower call
			_ = strings.ToLower(testString)
		}
	})

	b.Run("New_StringsBuilder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// New approach: using strings.Builder with pre-allocated capacity
			builder := strings.Builder{}
			builder.Grow(len(testString))
			builder.WriteString(strings.ToLower(testString))
			_ = builder.String()
		}
	})

	b.Run("Optimized_Helper", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Optimized approach: using our helper function
			_ = normalizeStringToLower(testString)
		}
	})
}

// BenchmarkCategorizer_FullFlow benchmarks the complete categorization flow
// to measure the overall performance impact of string optimizations
func BenchmarkCategorizer_FullFlow(b *testing.B) {
	// Create mock store with comprehensive data
	mockStore := &store.MockCategoryStore{
		Categories: []models.CategoryConfig{
			{
				Name:     models.CategoryGroceries,
				Keywords: []string{"COOP", "MIGROS", "ALDI", "LIDL", "DENNER", "MANOR"},
			},
			{
				Name:     models.CategoryRestaurants,
				Keywords: []string{"PIZZERIA", "CAFE", "RESTAURANT", "SUSHI", "KEBAB", "RAMEN"},
			},
			{
				Name:     models.CategoryTransport,
				Keywords: []string{"SBB", "CFF", "MOBILITY", "PAYBYPHONE", "UBER", "TAXI"},
			},
		},
		CreditorMappings: map[string]string{
			"starbucks coffee": models.CategoryRestaurants,
			"migros":           models.CategoryGroceries,
			"shell":            "Transport",
		},
		DebtorMappings: map[string]string{
			"john doe":    models.CategorySalary,
			"rent office": "Housing",
			"insurance":   "Insurance",
		},
	}

	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create categorizer
	categorizer := NewCategorizer(nil, mockStore, mockLogger)

	// Test transactions with various scenarios
	transactions := []Transaction{
		{
			PartyName: "COOP SUPERMARKET DOWNTOWN LOCATION",
			IsDebtor:  true,
			Info:      "GROCERY SHOPPING WITH MULTIPLE ITEMS",
		},
		{
			PartyName: "STARBUCKS COFFEE CHAIN STORE",
			IsDebtor:  false,
			Info:      "COFFEE AND PASTRY PURCHASE",
		},
		{
			PartyName: "UNKNOWN MERCHANT WITH LONG NAME",
			IsDebtor:  true,
			Info:      "TRANSACTION WITH DETAILED DESCRIPTION",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tx := range transactions {
			_, err := categorizer.CategorizeTransaction(tx)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
