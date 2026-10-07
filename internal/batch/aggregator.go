// Package batch provides functionality for batch processing and aggregation of financial files
package batch

import (
	"sort"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// BatchAggregator handles the aggregation of multiple files by account
type BatchAggregator struct {
	logger logging.Logger
}

// NewBatchAggregator creates a new BatchAggregator instance
func NewBatchAggregator(logger logging.Logger) *BatchAggregator {
	return &BatchAggregator{
		logger: logger,
	}
}

// Consolidate orders a merged transaction set chronologically and reports
// potential duplicates without removing any.
//
// Merging a directory of mixed formats is exactly where overlaps appear — a
// Viseca PDF statement alongside the Viseca CSV export of the same month. They
// are reported rather than deduplicated because a similarity heuristic would
// eventually erase two genuine identical purchases made on the same day;
// iCompta runs its own duplicate detection at import.
//
// label names the batch in the duplicate warnings so a user reading the log
// knows which run they refer to.
func (ba *BatchAggregator) Consolidate(transactions []models.Transaction, label string) []models.Transaction {
	ba.sortTransactionsChronologically(transactions)
	ba.detectAndLogDuplicates(transactions, label)
	return transactions
}

// sortTransactionsChronologically sorts transactions by date, then by value date as secondary sort
func (ba *BatchAggregator) sortTransactionsChronologically(transactions []models.Transaction) {
	sort.Slice(transactions, func(i, j int) bool {
		// Primary sort: by transaction date
		if !transactions[i].Date.Equal(transactions[j].Date) {
			return transactions[i].Date.Before(transactions[j].Date)
		}

		// Secondary sort: by value date
		if !transactions[i].ValueDate.Equal(transactions[j].ValueDate) {
			return transactions[i].ValueDate.Before(transactions[j].ValueDate)
		}

		// Tertiary sort: by amount (for consistency)
		return transactions[i].Amount.LessThan(transactions[j].Amount)
	})
}

// detectAndLogDuplicates identifies potential duplicate transactions and logs warnings
// This helps users identify overlapping data but doesn't remove duplicates.
//
// Requires transactions to already be sorted chronologically by date — this is
// an unexported method, and its one production caller, Consolidate, sorts
// immediately before calling it. arePotentialDuplicates requires an exact date
// match, so once transactions[j].Date no longer equals transactions[i].Date,
// no later j can match i either: the inner loop breaks there instead of
// scanning the rest of a merged batch that can run into the thousands.
func (ba *BatchAggregator) detectAndLogDuplicates(transactions []models.Transaction, accountID string) {
	duplicateCount := 0

	// Simple duplicate detection: same date, amount, and party
	for i := 0; i < len(transactions)-1; i++ {
		for j := i + 1; j < len(transactions); j++ {
			if !transactions[j].Date.Equal(transactions[i].Date) {
				break
			}

			// Check if transactions are potential duplicates
			if ba.arePotentialDuplicates(transactions[i], transactions[j]) {
				duplicateCount++
				ba.logger.Warn("Potential duplicate transaction",
					logging.Field{Key: "account", Value: accountID},
					logging.Field{Key: "date", Value: transactions[i].Date.Format("2006-01-02")},
					logging.Field{Key: "amount", Value: transactions[i].Amount.String()},
					logging.Field{Key: "party", Value: transactions[i].GetCounterparty()})
				break // Only log once per transaction
			}
		}
	}

	if duplicateCount > 0 {
		ba.logger.Warn("Found potential duplicate transactions",
			logging.Field{Key: "count", Value: duplicateCount},
			logging.Field{Key: "account", Value: accountID})
	}
}

// arePotentialDuplicates checks if two transactions might be duplicates
func (ba *BatchAggregator) arePotentialDuplicates(tx1, tx2 models.Transaction) bool {
	// Same date
	if !tx1.Date.Equal(tx2.Date) {
		return false
	}

	// Same amount
	if !tx1.Amount.Equal(tx2.Amount) {
		return false
	}

	// Same counterparty (case-insensitive)
	party1 := strings.ToLower(strings.TrimSpace(tx1.GetCounterparty()))
	party2 := strings.ToLower(strings.TrimSpace(tx2.GetCounterparty()))
	if party1 != party2 {
		return false
	}

	return true
}
