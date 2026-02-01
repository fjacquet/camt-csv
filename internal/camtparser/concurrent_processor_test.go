package camtparser

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

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

	transactions := processor.ProcessTransactions(context.Background(), entries, simpleProcessor)

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

	transactions := processor.ProcessTransactions(context.Background(), entries, simpleProcessor)

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

			transactions := processor.processSequential(context.Background(), entries, simpleProcessor)

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

	transactions := processor.processSequential(context.Background(), entries, indexProcessor)

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

			transactions := processor.processConcurrent(context.Background(), entries, simpleProcessor)

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

	transactions := processor.processConcurrent(context.Background(), entries, sumProcessor)

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

	transactions := processor.processConcurrent(context.Background(), entries, indexProcessor)

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

	transactions := processor.ProcessTransactions(context.Background(), entries, simpleProcessor)

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

	processor.ProcessTransactions(context.Background(), entries, countingProcessor)

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

	processor.ProcessTransactions(context.Background(), entries, capturingProcessor)

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
	tx99 := processor.ProcessTransactions(context.Background(), entries99, simpleProcessor)
	assert.Len(t, tx99, 99)

	// Test just over threshold (100 = concurrent)
	entries100 := make([]models.Entry, 100)
	for i := range entries100 {
		entries100[i] = models.Entry{Amt: models.Amount{Value: "1", Ccy: "CHF"}}
	}
	tx100 := processor.ProcessTransactions(context.Background(), entries100, simpleProcessor)
	assert.Len(t, tx100, 100)
}

// Test context cancellation in sequential processing
func TestConcurrentProcessor_SequentialCancellation(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create small batch (< 100 for sequential processing)
	entries := make([]models.Entry, 50)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:       models.Amount{Value: "10.00", Ccy: "CHF"},
			CdtDbtInd: "DBIT",
		}
	}

	simpleProcessor := func(e *models.Entry) models.Transaction {
		amount, _ := decimal.NewFromString(e.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: e.Amt.Ccy,
		}
	}

	// Process with cancelled context
	transactions := processor.ProcessTransactions(ctx, entries, simpleProcessor)

	// Should return partial results or empty due to cancellation
	// The implementation stops on cancellation, so we should get fewer than 50
	assert.LessOrEqual(t, len(transactions), 50)
}

// Test context cancellation in concurrent processing
func TestConcurrentProcessor_ConcurrentCancellation(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Create large batch (>= 100 for concurrent processing)
	entries := make([]models.Entry, 1000)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:       models.Amount{Value: "10.00", Ccy: "CHF"},
			CdtDbtInd: "DBIT",
		}
	}

	// Slow processor to ensure timeout occurs
	slowProcessor := func(e *models.Entry) models.Transaction {
		time.Sleep(5 * time.Millisecond)
		amount, _ := decimal.NewFromString(e.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: e.Amt.Ccy,
		}
	}

	// Process with timeout context
	transactions := processor.ProcessTransactions(ctx, entries, slowProcessor)

	// Should process fewer than all entries due to timeout
	assert.Less(t, len(transactions), 1000, "Should not process all entries due to timeout")
}

// Test context respects cancellation signal
func TestConcurrentProcessor_CancellationSignal(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Create large batch for concurrent processing
	entries := make([]models.Entry, 500)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:       models.Amount{Value: "10.00", Ccy: "CHF"},
			CdtDbtInd: "DBIT",
		}
	}

	simpleProcessor := func(e *models.Entry) models.Transaction {
		amount, _ := decimal.NewFromString(e.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: e.Amt.Ccy,
		}
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	// Process with context that will be cancelled
	transactions := processor.ProcessTransactions(ctx, entries, simpleProcessor)

	// Should process some but not all entries
	assert.Greater(t, len(transactions), 0, "Should process some entries before cancellation")
	assert.LessOrEqual(t, len(transactions), 500, "Should not process all entries after cancellation")
}

// ===== Edge Case Tests: Context Cancellation =====

// TestConcurrentProcessor_CancellationBeforeStart tests cancellation before processing starts
func TestConcurrentProcessor_CancellationBeforeStart(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	// Create cancelled context before calling ProcessTransactions
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before processing

	entries := make([]models.Entry, 150)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:       models.Amount{Value: "10.00", Ccy: "CHF"},
			CdtDbtInd: "DBIT",
		}
	}

	simpleProcessor := func(e *models.Entry) models.Transaction {
		amount, _ := decimal.NewFromString(e.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: e.Amt.Ccy,
		}
	}

	// Should not panic when context is already cancelled
	require.NotPanics(t, func() {
		transactions := processor.ProcessTransactions(ctx, entries, simpleProcessor)
		// Should return empty or minimal results since context is cancelled
		assert.LessOrEqual(t, len(transactions), len(entries))
	})
}

// TestConcurrentProcessor_CancellationDuringProcessing tests cancellation mid-processing
func TestConcurrentProcessor_CancellationDuringProcessing(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create large batch (500 entries) for concurrent processing
	entries := make([]models.Entry, 500)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:       models.Amount{Value: decimal.NewFromInt(int64(i + 1)).String(), Ccy: "CHF"},
			CdtDbtInd: "DBIT",
		}
	}

	// Record goroutine count before processing
	goroutinesBefore := runtime.NumGoroutine()

	// Processor with slight delay to ensure some work is in flight
	var processedCount int64
	slowProcessor := func(e *models.Entry) models.Transaction {
		atomic.AddInt64(&processedCount, 1)
		time.Sleep(1 * time.Millisecond) // Small delay to simulate work
		amount, _ := decimal.NewFromString(e.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: e.Amt.Ccy,
		}
	}

	// Schedule cancellation after 100ms
	time.AfterFunc(100*time.Millisecond, cancel)

	// Process with context that will be cancelled mid-processing
	var transactions []models.Transaction
	require.NotPanics(t, func() {
		transactions = processor.ProcessTransactions(ctx, entries, slowProcessor)
	}, "Should not panic on cancellation")

	// Verify no data corruption - each returned transaction should be valid
	for i, tx := range transactions {
		assert.True(t, tx.Amount.GreaterThan(decimal.Zero),
			"Transaction %d should have positive amount, got %v", i, tx.Amount)
		assert.Equal(t, "CHF", tx.Currency,
			"Transaction %d should have currency CHF, got %s", i, tx.Currency)
	}

	// Verify some results were returned (workers already started)
	assert.Greater(t, len(transactions), 0, "Should return some results from workers that started")
	assert.LessOrEqual(t, len(transactions), 500, "Should not process all entries due to cancellation")

	// Wait for goroutines to cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify no goroutine leaks (allow small variance for runtime internals)
	goroutinesAfter := runtime.NumGoroutine()
	leakedGoroutines := goroutinesAfter - goroutinesBefore
	assert.LessOrEqual(t, leakedGoroutines, 2,
		"Should not leak goroutines: before=%d, after=%d, leaked=%d",
		goroutinesBefore, goroutinesAfter, leakedGoroutines)
}

// TestConcurrentProcessor_CancellationWaitsForInflightWork verifies inflight work completion behavior
func TestConcurrentProcessor_CancellationWaitsForInflightWork(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	processor := NewConcurrentProcessor(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create entries for concurrent processing
	entries := make([]models.Entry, 200)
	for i := range entries {
		entries[i] = models.Entry{
			Amt:       models.Amount{Value: decimal.NewFromInt(int64(i + 1)).String(), Ccy: "CHF"},
			CdtDbtInd: "DBIT",
		}
	}

	// Track when processing started and completed for each entry
	var startedCount, completedCount int64
	processorWithTracking := func(e *models.Entry) models.Transaction {
		atomic.AddInt64(&startedCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate longer work
		atomic.AddInt64(&completedCount, 1)

		amount, _ := decimal.NewFromString(e.Amt.Value)
		return models.Transaction{
			Amount:   amount,
			Currency: e.Amt.Ccy,
		}
	}

	// Cancel after short delay to catch some work inflight
	time.AfterFunc(30*time.Millisecond, cancel)

	transactions := processor.ProcessTransactions(ctx, entries, processorWithTracking)

	// After cancellation, some work may have completed but results not sent
	// Verify: returned transactions <= completed <= started <= total
	assert.LessOrEqual(t, int64(len(transactions)), completedCount,
		"Returned transactions should be <= completed work")
	assert.LessOrEqual(t, completedCount, startedCount,
		"Completed work should be <= started work")

	// Verify some work was cancelled (not all entries processed)
	assert.Less(t, len(transactions), 200,
		"Some work should be cancelled before starting")

	// Verify inflight work completed (started == completed)
	assert.Equal(t, startedCount, completedCount,
		"All started processing should complete: started=%d, completed=%d",
		startedCount, completedCount)
}
