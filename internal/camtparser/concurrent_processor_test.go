package camtparser

import (
	"runtime"
	"sync/atomic"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConcurrentProcessor(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	assert.NotNil(t, processor)
	assert.NotNil(t, processor.logger)
	assert.Equal(t, runtime.NumCPU(), processor.workerCount)
}

func TestConcurrentProcessor_ProcessTransactions_UsesSequentialForSmall(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Create fewer than 100 entries (threshold for sequential)
	entries := make([]models.Entry, 50)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:     models.Amount{Value: "100", Ccy: "CHF"},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
			ValDt:   models.EntryDate{Dt: "2023-01-01"},
		}
	}

	// Simple processor that returns a transaction
	simpleProcessor := func(entry *models.Entry) models.Transaction {
		return models.Transaction{
			Amount:   decimal.NewFromFloat(100),
			Currency: "CHF",
		}
	}

	transactions := processor.ProcessTransactions(entries, simpleProcessor)

	assert.Len(t, transactions, 50)
}

func TestConcurrentProcessor_ProcessTransactions_UsesConcurrentForLarge(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	processor := NewConcurrentProcessor(logger)

	// Create more than 100 entries (threshold for concurrent)
	entries := make([]models.Entry, 150)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:     models.Amount{Value: "200", Ccy: "EUR"},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
			ValDt:   models.EntryDate{Dt: "2023-01-01"},
		}
	}

	// Simple processor that returns a transaction
	simpleProcessor := func(entry *models.Entry) models.Transaction {
		return models.Transaction{
			Amount:   decimal.NewFromFloat(200),
			Currency: "EUR",
		}
	}

	transactions := processor.ProcessTransactions(entries, simpleProcessor)

	// Should have processed all entries
	assert.Len(t, transactions, 150)
}

func TestConcurrentProcessor_ProcessSequential(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	tests := []struct {
		name          string
		entryCount    int
		expectedCount int
	}{
		{
			name:          "empty entries",
			entryCount:    0,
			expectedCount: 0,
		},
		{
			name:          "single entry",
			entryCount:    1,
			expectedCount: 1,
		},
		{
			name:          "multiple entries",
			entryCount:    10,
			expectedCount: 10,
		},
		{
			name:          "at threshold boundary",
			entryCount:    99,
			expectedCount: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := make([]models.Entry, tt.entryCount)
			for i := range entries {
				entries[i] = models.Entry{
					Amt:     models.Amount{Value: "50", Ccy: "CHF"},
					BookgDt: models.EntryDate{Dt: "2023-01-01"},
				}
			}

			processedCount := 0
			simpleProcessor := func(entry *models.Entry) models.Transaction {
				processedCount++
				return models.Transaction{Amount: decimal.NewFromFloat(50)}
			}

			transactions := processor.processSequential(entries, simpleProcessor)

			assert.Len(t, transactions, tt.expectedCount)
			assert.Equal(t, tt.expectedCount, processedCount)
		})
	}
}

func TestConcurrentProcessor_ProcessSequential_MaintainsOrder(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Create entries with unique amounts to track order
	entries := make([]models.Entry, 10)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:     models.Amount{Value: string(rune('0' + i)), Ccy: "CHF"},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
		}
	}

	// Processor that preserves the amount as a marker
	indexProcessor := func(entry *models.Entry) models.Transaction {
		amount, _ := decimal.NewFromString(entry.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: "CHF",
		}
	}

	transactions := processor.processSequential(entries, indexProcessor)

	require.Len(t, transactions, 10)
	for i := 0; i < 10; i++ {
		expected, _ := decimal.NewFromString(string(rune('0' + i)))
		assert.True(t, transactions[i].Amount.Equal(expected),
			"Expected amount %v at index %d, got %v", expected, i, transactions[i].Amount)
	}
}

func TestConcurrentProcessor_ProcessConcurrent(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	processor := NewConcurrentProcessor(logger)

	tests := []struct {
		name          string
		entryCount    int
		expectedCount int
	}{
		{
			name:          "just over threshold",
			entryCount:    100,
			expectedCount: 100,
		},
		{
			name:          "large dataset",
			entryCount:    500,
			expectedCount: 500,
		},
		{
			name:          "very large dataset",
			entryCount:    1000,
			expectedCount: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := make([]models.Entry, tt.entryCount)
			for i := range entries {
				entries[i] = models.Entry{
					Amt:     models.Amount{Value: "75", Ccy: "USD"},
					BookgDt: models.EntryDate{Dt: "2023-01-01"},
				}
			}

			simpleProcessor := func(entry *models.Entry) models.Transaction {
				return models.Transaction{
					Amount:   decimal.NewFromFloat(75),
					Currency: "USD",
				}
			}

			transactions := processor.processConcurrent(entries, simpleProcessor)

			assert.Len(t, transactions, tt.expectedCount)
		})
	}
}

func TestConcurrentProcessor_ProcessConcurrent_AllEntriesProcessed(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Create entries with different amounts to verify all are processed
	entryCount := 200
	entries := make([]models.Entry, entryCount)

	for i := range entries {
		amount := decimal.NewFromInt(int64(i + 1))
		entries[i] = models.Entry{
			Amt:     models.Amount{Value: amount.String(), Ccy: "CHF"},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
		}
	}

	// Use atomic counter for thread-safe counting
	var processedCount int64
	sumProcessor := func(entry *models.Entry) models.Transaction {
		atomic.AddInt64(&processedCount, 1)
		amount, _ := decimal.NewFromString(entry.Amt.Value)
		return models.Transaction{Amount: amount}
	}

	transactions := processor.processConcurrent(entries, sumProcessor)

	// Verify correct count is returned
	require.Len(t, transactions, entryCount)

	// Verify the processor function was called for all entries
	assert.Equal(t, int64(entryCount), processedCount,
		"Processor should be called once per entry")
}

func TestConcurrentProcessor_ProcessConcurrent_MaintainsOrder(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Create entries with unique amounts to track order
	entryCount := 150
	entries := make([]models.Entry, entryCount)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:     models.Amount{Value: decimal.NewFromInt(int64(i + 1)).String(), Ccy: "CHF"},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
		}
	}

	// Processor that preserves the amount as a marker
	indexProcessor := func(entry *models.Entry) models.Transaction {
		amount, _ := decimal.NewFromString(entry.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: "CHF",
		}
	}

	transactions := processor.processConcurrent(entries, indexProcessor)

	require.Len(t, transactions, entryCount)

	// Verify that order is maintained - each transaction should have
	// the amount matching its original position (1-indexed)
	for i := 0; i < entryCount; i++ {
		expected := decimal.NewFromInt(int64(i + 1))
		assert.True(t, transactions[i].Amount.Equal(expected),
			"Expected amount %v at index %d, got %v", expected, i, transactions[i].Amount)
	}
}

func TestConcurrentProcessor_WorkerCount(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Worker count should equal number of CPUs
	assert.Equal(t, runtime.NumCPU(), processor.workerCount)
	assert.Greater(t, processor.workerCount, 0)
}

func TestConcurrentProcessor_EmptyEntries(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	entries := []models.Entry{}

	simpleProcessor := func(entry *models.Entry) models.Transaction {
		return models.Transaction{}
	}

	transactions := processor.ProcessTransactions(entries, simpleProcessor)

	assert.Empty(t, transactions)
}

func TestConcurrentProcessor_ProcessorFunctionCalled(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	entries := make([]models.Entry, 5)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:     models.Amount{Value: "100", Ccy: "CHF"},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
		}
	}

	callCount := 0
	countingProcessor := func(entry *models.Entry) models.Transaction {
		callCount++
		return models.Transaction{}
	}

	processor.ProcessTransactions(entries, countingProcessor)

	assert.Equal(t, 5, callCount, "Processor function should be called once per entry")
}

func TestConcurrentProcessor_ProcessorReceivesCorrectEntry(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	entries := []models.Entry{
		{Amt: models.Amount{Value: "111", Ccy: "CHF"}, BookgDt: models.EntryDate{Dt: "2023-01-01"}},
		{Amt: models.Amount{Value: "222", Ccy: "EUR"}, BookgDt: models.EntryDate{Dt: "2023-01-02"}},
		{Amt: models.Amount{Value: "333", Ccy: "USD"}, BookgDt: models.EntryDate{Dt: "2023-01-03"}},
	}

	receivedAmounts := make([]string, 0, len(entries))
	capturingProcessor := func(entry *models.Entry) models.Transaction {
		receivedAmounts = append(receivedAmounts, entry.Amt.Value)
		return models.Transaction{}
	}

	processor.ProcessTransactions(entries, capturingProcessor)

	assert.Len(t, receivedAmounts, 3)
	assert.Contains(t, receivedAmounts, "111")
	assert.Contains(t, receivedAmounts, "222")
	assert.Contains(t, receivedAmounts, "333")
}

func TestConcurrentProcessor_ThresholdBoundary(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	simpleProcessor := func(entry *models.Entry) models.Transaction {
		return models.Transaction{Amount: decimal.NewFromFloat(1)}
	}

	// Test exactly at threshold (99 = sequential)
	entries99 := make([]models.Entry, 99)
	for i := range entries99 {
		entries99[i] = models.Entry{Amt: models.Amount{Value: "1", Ccy: "CHF"}}
	}
	tx99 := processor.ProcessTransactions(entries99, simpleProcessor)
	assert.Len(t, tx99, 99)

	// Test just over threshold (100 = concurrent)
	entries100 := make([]models.Entry, 100)
	for i := range entries100 {
		entries100[i] = models.Entry{Amt: models.Amount{Value: "1", Ccy: "CHF"}}
	}
	tx100 := processor.ProcessTransactions(entries100, simpleProcessor)
	assert.Len(t, tx100, 100)
}
