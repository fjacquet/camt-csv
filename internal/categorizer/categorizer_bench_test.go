package categorizer

import (
	"context"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// MockCategoryStore for benchmarking
type MockCategoryStore struct {
	categories       []models.CategoryConfig
	creditorMappings map[string]string
	debtorMappings   map[string]string
}

func (m *MockCategoryStore) LoadCategories() ([]models.CategoryConfig, error) {
	return m.categories, nil
}

func (m *MockCategoryStore) LoadCreditorMappings() (map[string]string, error) {
	return m.creditorMappings, nil
}

func (m *MockCategoryStore) LoadDebtorMappings() (map[string]string, error) {
	return m.debtorMappings, nil
}

func (m *MockCategoryStore) SaveCreditorMappings(mappings map[string]string) error {
	return nil
}

func (m *MockCategoryStore) SaveDebtorMappings(mappings map[string]string) error {
	return nil
}

// Deprecated backward compatibility methods
func (m *MockCategoryStore) LoadDebitorMappings() (map[string]string, error) {
	return m.LoadDebtorMappings()
}

func (m *MockCategoryStore) SaveDebitorMappings(mappings map[string]string) error {
	return m.SaveDebtorMappings(mappings)
}

// MockLogger for benchmarking
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields ...logging.Field)              {}
func (m *MockLogger) Info(msg string, fields ...logging.Field)               {}
func (m *MockLogger) Warn(msg string, fields ...logging.Field)               {}
func (m *MockLogger) Error(msg string, fields ...logging.Field)              {}
func (m *MockLogger) Fatal(msg string, fields ...logging.Field)              {}
func (m *MockLogger) Fatalf(msg string, args ...interface{})                 {}
func (m *MockLogger) WithError(err error) logging.Logger                     { return m }
func (m *MockLogger) WithField(key string, value interface{}) logging.Logger { return m }
func (m *MockLogger) WithFields(fields ...logging.Field) logging.Logger      { return m }

// BenchmarkDirectMappingStrategy benchmarks the direct mapping strategy
func BenchmarkDirectMappingStrategy(b *testing.B) {
	// Create test data
	creditorMappings := make(map[string]string, 1000)
	debtorMappings := make(map[string]string, 1000)

	// Populate with test data
	for i := 0; i < 1000; i++ {
		creditorMappings[strings.ToLower("CREDITOR_"+string(rune(i)))] = "TestCategory"
		debtorMappings[strings.ToLower("DEBTOR_"+string(rune(i)))] = "TestCategory"
	}

	store := &MockCategoryStore{
		creditorMappings: creditorMappings,
		debtorMappings:   debtorMappings,
	}

	logger := &MockLogger{}
	strategy := NewDirectMappingStrategy(store, logger)

	tx := Transaction{
		PartyName: "CREDITOR_500",
		IsDebtor:  false,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = strategy.Categorize(ctx, tx)
	}
}

// BenchmarkKeywordStrategy benchmarks the keyword strategy
func BenchmarkKeywordStrategy(b *testing.B) {
	categories := []models.CategoryConfig{
		{
			Name:     "Groceries",
			Keywords: []string{"COOP", "MIGROS", "ALDI", "LIDL"},
		},
		{
			Name:     "Transport",
			Keywords: []string{"SBB", "CFF", "MOBILITY"},
		},
	}

	store := &MockCategoryStore{
		categories: categories,
	}

	logger := &MockLogger{}
	strategy := NewKeywordStrategy(store, logger)

	tx := Transaction{
		PartyName: "COOP SUPERMARKET",
		Info:      "Purchase at grocery store",
		IsDebtor:  true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = strategy.Categorize(ctx, tx)
	}
}

// BenchmarkStringOperations benchmarks string operations before and after optimization
func BenchmarkStringOperations(b *testing.B) {
	testStrings := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		testStrings[i] = "Test String " + string(rune(i)) + " With Some Content"
	}

	b.Run("StringsToLower", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				_ = strings.ToLower(s)
			}
		}
	})

	b.Run("StringsBuilderToLower", func(b *testing.B) {
		var builder strings.Builder
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				builder.Reset()
				builder.Grow(len(s))
				builder.WriteString(s)
				_ = strings.ToLower(builder.String())
			}
		}
	})
}

// BenchmarkMapAllocation benchmarks map allocation patterns
func BenchmarkMapAllocation(b *testing.B) {
	testData := make(map[string]string, 1000)
	for i := 0; i < 1000; i++ {
		testData["key"+string(rune(i))] = "value" + string(rune(i))
	}

	b.Run("WithoutPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[string]string)
			for k, v := range testData {
				m[k] = v
			}
		}
	})

	b.Run("WithPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[string]string, len(testData))
			for k, v := range testData {
				m[k] = v
			}
		}
	})
}

// BenchmarkSliceAllocation benchmarks slice allocation patterns
func BenchmarkSliceAllocation(b *testing.B) {
	testData := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		testData[i] = "item" + string(rune(i))
	}

	b.Run("WithoutPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result []string
			for _, item := range testData {
				result = append(result, item)
			}
		}
	})

	b.Run("WithPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := make([]string, 0, len(testData))
			for _, item := range testData {
				result = append(result, item)
			}
		}
	})
}
