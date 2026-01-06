// Package common provides shared utilities used across the application.
package common

import (
	"errors"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCategorizer is a mock implementation of TransactionCategorizer for testing
type MockCategorizer struct {
	mock.Mock
}

func (m *MockCategorizer) Categorize(partyName string, isDebtor bool, amount, date, info string) (models.Category, error) {
	args := m.Called(partyName, isDebtor, amount, date, info)
	return args.Get(0).(models.Category), args.Error(1)
}

// MockLogger is a mock implementation of Logger for testing
type MockLogger struct {
	mock.Mock
	InfoCalls  []LogCall
	DebugCalls []LogCall
	WarnCalls  []LogCall
}

type LogCall struct {
	Message string
	Fields  []logging.Field
}

func (m *MockLogger) Info(msg string, fields ...logging.Field) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
	m.Called(msg, fields)
}

func (m *MockLogger) Debug(msg string, fields ...logging.Field) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Fields: fields})
	m.Called(msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...logging.Field) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
	m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...logging.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Fatal(msg string, fields ...logging.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Fatalf(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *MockLogger) WithError(err error) logging.Logger {
	return m
}

func (m *MockLogger) WithField(key string, value interface{}) logging.Logger {
	return m
}

func (m *MockLogger) WithFields(fields ...logging.Field) logging.Logger {
	return m
}

// Property 9: Categorization fallback behavior
// For any parser processing transactions, when categorization fails or is unavailable,
// transactions should be assigned "Uncategorized" category and appropriate statistics should be tracked
func TestProperty9_CategorizationFallbackBehavior(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Run("iteration", func(t *testing.T) {
			// Generate test transactions
			transactions := generateTestTransactions(5)
			mockLogger := &MockLogger{}
			mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()
			mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()
			mockLogger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return()

			// Test case 1: No categorizer provided
			result1 := ProcessTransactionsWithCategorizationStats(transactions, mockLogger, nil, "TestParser")

			// All transactions should be uncategorized
			for _, tx := range result1 {
				assert.Equal(t, "Uncategorized", tx.Category, "Transaction should be uncategorized when no categorizer provided")
			}

			// Test case 2: Categorizer that always fails
			mockCategorizer := &MockCategorizer{}
			mockCategorizer.On("Categorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(models.Category{}, errors.New("categorization failed"))

			result2 := ProcessTransactionsWithCategorizationStats(transactions, mockLogger, mockCategorizer, "TestParser")

			// All transactions should be uncategorized due to errors
			for _, tx := range result2 {
				assert.Equal(t, "Uncategorized", tx.Category, "Transaction should be uncategorized when categorization fails")
			}

			// Test case 3: Categorizer that returns empty category
			mockCategorizer2 := &MockCategorizer{}
			mockCategorizer2.On("Categorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(models.Category{Name: ""}, nil)

			result3 := ProcessTransactionsWithCategorizationStats(transactions, mockLogger, mockCategorizer2, "TestParser")

			// All transactions should be uncategorized due to empty category
			for _, tx := range result3 {
				assert.Equal(t, "Uncategorized", tx.Category, "Transaction should be uncategorized when category name is empty")
			}

			// Verify that statistics logging was called
			assert.True(t, len(mockLogger.InfoCalls) > 0, "Statistics should be logged")

			// Find the categorization summary log call
			var summaryFound bool
			for _, call := range mockLogger.InfoCalls {
				if call.Message == "Categorization summary" {
					summaryFound = true
					// Verify that uncategorized count matches transaction count
					var uncategorizedCount int
					for _, field := range call.Fields {
						if field.Key == "uncategorized" {
							uncategorizedCount = field.Value.(int)
						}
					}
					assert.Equal(t, len(transactions), uncategorizedCount, "Uncategorized count should match transaction count")
					break
				}
			}
			assert.True(t, summaryFound, "Categorization summary should be logged")
		})
	}
}

// Property 10: Categorization statistics logging
// For any parser processing completion, categorization statistics (successful/failed/uncategorized counts) should be logged
func TestProperty10_CategorizationStatisticsLogging(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Run("iteration", func(t *testing.T) {
			// Generate test transactions
			transactions := generateTestTransactions(10)
			mockLogger := &MockLogger{}
			mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()
			mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()
			mockLogger.On("Warn", mock.AnythingOfType("string"), mock.Anything).Return()

			// Create a categorizer that succeeds for some transactions and fails for others
			mockCategorizer := &MockCategorizer{}

			// Set up different responses based on party name
			mockCategorizer.On("Categorize", mock.MatchedBy(func(partyName string) bool {
				return strings.Contains(partyName, "success")
			}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(models.Category{Name: "TestCategory"}, nil)

			mockCategorizer.On("Categorize", mock.MatchedBy(func(partyName string) bool {
				return strings.Contains(partyName, "fail")
			}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(models.Category{}, errors.New("categorization failed"))

			mockCategorizer.On("Categorize", mock.MatchedBy(func(partyName string) bool {
				return strings.Contains(partyName, "uncategorized")
			}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(models.Category{Name: "Uncategorized"}, nil)

			// Default case for other party names
			mockCategorizer.On("Categorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(models.Category{Name: "DefaultCategory"}, nil)

			// Modify transactions to have predictable party names
			for i := range transactions {
				switch i % 4 {
				case 0:
					transactions[i].PartyName = "success_party"
				case 1:
					transactions[i].PartyName = "fail_party"
				case 2:
					transactions[i].PartyName = "uncategorized_party"
				default:
					transactions[i].PartyName = "default_party"
				}
			}

			// Process transactions
			result := ProcessTransactionsWithCategorizationStats(transactions, mockLogger, mockCategorizer, "TestParser")

			// Verify that all transactions were processed
			assert.Equal(t, len(transactions), len(result), "All transactions should be processed")

			// Find and verify the categorization summary log
			var summaryFound bool
			var totalCount, successfulCount, failedCount, uncategorizedCount int
			var successRate float64

			for _, call := range mockLogger.InfoCalls {
				if call.Message == "Categorization summary" {
					summaryFound = true

					// Extract statistics from log fields
					for _, field := range call.Fields {
						switch field.Key {
						case "total_transactions":
							totalCount = field.Value.(int)
						case "successful":
							successfulCount = field.Value.(int)
						case "failed":
							failedCount = field.Value.(int)
						case "uncategorized":
							uncategorizedCount = field.Value.(int)
						case "success_rate":
							successRate = field.Value.(float64)
						case "parser_type":
							assert.Equal(t, "TestParser", field.Value.(string), "Parser type should be logged correctly")
						}
					}
					break
				}
			}

			// Verify that statistics were logged
			assert.True(t, summaryFound, "Categorization summary should be logged")
			assert.Equal(t, len(transactions), totalCount, "Total count should match transaction count")
			assert.Equal(t, successfulCount+failedCount+uncategorizedCount, totalCount, "Sum of categories should equal total")

			// Verify success rate calculation
			expectedSuccessRate := float64(successfulCount) / float64(totalCount) * 100.0
			if totalCount == 0 {
				expectedSuccessRate = 0.0
			}
			assert.Equal(t, expectedSuccessRate, successRate, "Success rate should be calculated correctly")

			// Verify that statistics are non-negative
			assert.GreaterOrEqual(t, totalCount, 0, "Total count should be non-negative")
			assert.GreaterOrEqual(t, successfulCount, 0, "Successful count should be non-negative")
			assert.GreaterOrEqual(t, failedCount, 0, "Failed count should be non-negative")
			assert.GreaterOrEqual(t, uncategorizedCount, 0, "Uncategorized count should be non-negative")
			assert.GreaterOrEqual(t, successRate, 0.0, "Success rate should be non-negative")
			assert.LessOrEqual(t, successRate, 100.0, "Success rate should not exceed 100%")
		})
	}
}

// Helper function to generate test transactions
func generateTestTransactions(count int) []models.Transaction {
	transactions := make([]models.Transaction, count)

	for i := 0; i < count; i++ {
		transactions[i] = models.Transaction{
			Date:        time.Now().AddDate(0, 0, -i),
			ValueDate:   time.Now().AddDate(0, 0, -i),
			PartyName:   "TestParty",
			Description: "Test transaction",
			Amount:      decimal.NewFromFloat(100.0 + float64(i)),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
		}
	}

	return transactions
}

// Test CategorizationStats methods
func TestCategorizationStats(t *testing.T) {
	stats := models.NewCategorizationStats()

	// Test initial state
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Successful)
	assert.Equal(t, 0, stats.Failed)
	assert.Equal(t, 0, stats.Uncategorized)
	assert.Equal(t, 0.0, stats.GetSuccessRate())

	// Test increment methods
	stats.IncrementTotal()
	stats.IncrementSuccessful()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Successful)
	assert.Equal(t, 100.0, stats.GetSuccessRate())

	stats.IncrementTotal()
	stats.IncrementFailed()
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 50.0, stats.GetSuccessRate())

	stats.IncrementTotal()
	stats.IncrementUncategorized()
	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 1, stats.Uncategorized)
	assert.InDelta(t, 33.33, stats.GetSuccessRate(), 0.01)
}

// Test LogSummary method
func TestCategorizationStatsLogSummary(t *testing.T) {
	stats := &models.CategorizationStats{
		Total:         10,
		Successful:    7,
		Failed:        2,
		Uncategorized: 1,
	}

	mockLogger := &MockLogger{}
	mockLogger.On("Info", mock.AnythingOfType("string"), mock.Anything).Return()

	stats.LogSummary(mockLogger, "TestParser")

	// Verify that Info was called
	assert.Equal(t, 1, len(mockLogger.InfoCalls))
	call := mockLogger.InfoCalls[0]
	assert.Equal(t, "Categorization summary", call.Message)

	// Verify all expected fields are present
	fieldMap := make(map[string]interface{})
	for _, field := range call.Fields {
		fieldMap[field.Key] = field.Value
	}

	assert.Equal(t, "TestParser", fieldMap["parser_type"])
	assert.Equal(t, 10, fieldMap["total_transactions"])
	assert.Equal(t, 7, fieldMap["successful"])
	assert.Equal(t, 2, fieldMap["failed"])
	assert.Equal(t, 1, fieldMap["uncategorized"])
	assert.Equal(t, 70.0, fieldMap["success_rate"])
}

// Test with nil logger
func TestProcessTransactionsWithCategorizationStats_NilLogger(t *testing.T) {
	transactions := generateTestTransactions(1)

	// Should not panic with nil logger
	result := ProcessTransactionsWithCategorizationStats(transactions, nil, nil, "TestParser")

	assert.Equal(t, len(transactions), len(result))
	assert.Equal(t, "Uncategorized", result[0].Category)
}

// Test CategorizeTransactionWithStats function
func TestCategorizeTransactionWithStats(t *testing.T) {
	tx := generateTestTransactions(1)[0]
	stats := models.NewCategorizationStats()
	mockLogger := &MockLogger{}
	mockLogger.On("Debug", mock.AnythingOfType("string"), mock.Anything).Return()

	// Test with nil categorizer
	CategorizeTransactionWithStats(&tx, nil, stats, mockLogger, "TestParser")

	assert.Equal(t, "Uncategorized", tx.Category)
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 0, stats.Successful)
	assert.Equal(t, 0, stats.Failed)
	assert.Equal(t, 1, stats.Uncategorized)
}
