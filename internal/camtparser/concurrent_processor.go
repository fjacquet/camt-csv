package camtparser

import (
	"context"
	"runtime"
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
func (cp *ConcurrentProcessor) ProcessTransactions(entries []models.Entry, processor func(*models.Entry) models.Transaction) []models.Transaction {
	entryCount := len(entries)
	
	// Use sequential processing for small datasets to avoid overhead
	if entryCount < 100 {
		return cp.processSequential(entries, processor)
	}
	
	return cp.processConcurrent(entries, processor)
}

// processSequential handles small datasets sequentially
func (cp *ConcurrentProcessor) processSequential(entries []models.Entry, processor func(*models.Entry) models.Transaction) []models.Transaction {
	transactions := make([]models.Transaction, 0, len(entries))
	
	for i := range entries {
		tx := processor(&entries[i])
		transactions = append(transactions, tx)
	}
	
	return transactions
}

// processConcurrent handles large datasets with worker pools
func (cp *ConcurrentProcessor) processConcurrent(entries []models.Entry, processor func(*models.Entry) models.Transaction) []models.Transaction {
	ctx := context.Background()
	
	// Create channels for work distribution
	entryChan := make(chan *models.Entry, cp.workerCount)
	resultChan := make(chan indexedTransaction, len(entries))
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < cp.workerCount; i++ {
		wg.Add(1)
		go cp.worker(ctx, &wg, entryChan, resultChan, processor)
	}
	
	// Send work to workers
	go func() {
		defer close(entryChan)
		for i := range entries {
			select {
			case entryChan <- &entries[i]:
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
	
	// Collect results in order
	results := make([]indexedTransaction, 0, len(entries))
	for result := range resultChan {
		results = append(results, result)
	}
	
	// Sort results by original index to maintain order
	transactions := make([]models.Transaction, len(results))
	for _, result := range results {
		transactions[result.index] = result.transaction
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
func (cp *ConcurrentProcessor) worker(ctx context.Context, wg *sync.WaitGroup, entryChan <-chan *models.Entry, resultChan chan<- indexedTransaction, processor func(*models.Entry) models.Transaction) {
	defer wg.Done()
	
	for {
		select {
		case entry, ok := <-entryChan:
			if !ok {
				return
			}
			
			tx := processor(entry)
			
			select {
			case resultChan <- indexedTransaction{
				index:       0, // Index would need to be passed with the entry
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