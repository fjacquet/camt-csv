package camtparser

import (
	"context"
	"runtime"
	"sort"
	"sync"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// ConcurrentProcessor handles parallel processing of transactions
type ConcurrentProcessor struct {
	logger      logging.Logger
	workerCount int
}

// NewConcurrentProcessor creates a new concurrent processor
func NewConcurrentProcessor(logger logging.Logger) *ConcurrentProcessor {
	return &ConcurrentProcessor{
		logger:      logger,
		workerCount: runtime.NumCPU(),
	}
}

// ProcessTransactions processes transactions concurrently when beneficial
func (cp *ConcurrentProcessor) ProcessTransactions(ctx context.Context, entries []models.Entry, processor func(*models.Entry) models.Transaction) []models.Transaction {
	entryCount := len(entries)

	// Use sequential processing for small datasets to avoid overhead
	if entryCount < 100 {
		return cp.processSequential(ctx, entries, processor)
	}

	return cp.processConcurrent(ctx, entries, processor)
}

// processSequential handles small datasets sequentially
func (cp *ConcurrentProcessor) processSequential(ctx context.Context, entries []models.Entry, processor func(*models.Entry) models.Transaction) []models.Transaction {
	transactions := make([]models.Transaction, 0, len(entries))

	for i := range entries {
		// Check for cancellation
		select {
		case <-ctx.Done():
			cp.logger.Warn("Sequential processing cancelled",
				logging.Field{Key: "processed", Value: len(transactions)},
				logging.Field{Key: "total", Value: len(entries)})
			return transactions
		default:
		}

		tx := processor(&entries[i])
		transactions = append(transactions, tx)
	}

	return transactions
}

// indexedEntry pairs an entry with its original index for order preservation
type indexedEntry struct {
	index int
	entry *models.Entry
}

// processConcurrent handles large datasets with worker pools
func (cp *ConcurrentProcessor) processConcurrent(ctx context.Context, entries []models.Entry, processor func(*models.Entry) models.Transaction) []models.Transaction {

	// Create channels for work distribution
	entryChan := make(chan indexedEntry, cp.workerCount)
	resultChan := make(chan indexedTransaction, len(entries))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < cp.workerCount; i++ {
		wg.Add(1)
		go cp.worker(ctx, &wg, entryChan, resultChan, processor)
	}

	// Send work to workers with their original indices
	go func() {
		defer close(entryChan)
		for i := range entries {
			select {
			case entryChan <- indexedEntry{index: i, entry: &entries[i]}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make([]indexedTransaction, 0, len(entries))
	for result := range resultChan {
		results = append(results, result)
	}

	// Sort results by original index to maintain input order
	sort.Slice(results, func(i, j int) bool {
		return results[i].index < results[j].index
	})

	// Create dense slice from sorted results (handles partial results from cancellation)
	transactions := make([]models.Transaction, len(results))
	for i, result := range results {
		transactions[i] = result.transaction
	}

	cp.logger.Debug("Concurrent processing completed",
		logging.Field{Key: "entries", Value: len(entries)},
		logging.Field{Key: "workers", Value: cp.workerCount})

	return transactions
}

// indexedTransaction preserves the original order of transactions
type indexedTransaction struct {
	index       int
	transaction models.Transaction
}

// worker processes entries from the channel
func (cp *ConcurrentProcessor) worker(ctx context.Context, wg *sync.WaitGroup, entryChan <-chan indexedEntry, resultChan chan<- indexedTransaction, processor func(*models.Entry) models.Transaction) {
	defer wg.Done()

	for {
		select {
		case ie, ok := <-entryChan:
			if !ok {
				return
			}

			tx := processor(ie.entry)

			select {
			case resultChan <- indexedTransaction{
				index:       ie.index,
				transaction: tx,
			}:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}
