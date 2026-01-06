// Package models provides the data structures used throughout the application.
package models

import (
	"fjacquet/camt-csv/internal/logging"
)

// CategorizationStats tracks statistics for transaction categorization
type CategorizationStats struct {
	Total         int // Total number of transactions processed
	Successful    int // Number of transactions successfully categorized
	Failed        int // Number of transactions that failed categorization
	Uncategorized int // Number of transactions left uncategorized
}

// LogSummary logs a summary of categorization statistics
func (cs CategorizationStats) LogSummary(logger logging.Logger, parserType string) {
	if logger == nil {
		return
	}

	logger.Info("Categorization summary",
		logging.Field{Key: "parser_type", Value: parserType},
		logging.Field{Key: "total_transactions", Value: cs.Total},
		logging.Field{Key: "successful", Value: cs.Successful},
		logging.Field{Key: "failed", Value: cs.Failed},
		logging.Field{Key: "uncategorized", Value: cs.Uncategorized},
		logging.Field{Key: "success_rate", Value: cs.GetSuccessRate()},
	)
}

// GetSuccessRate calculates the success rate as a percentage
func (cs CategorizationStats) GetSuccessRate() float64 {
	if cs.Total == 0 {
		return 0.0
	}
	return float64(cs.Successful) / float64(cs.Total) * 100.0
}

// IncrementTotal increments the total transaction count
func (cs *CategorizationStats) IncrementTotal() {
	cs.Total++
}

// IncrementSuccessful increments the successful categorization count
func (cs *CategorizationStats) IncrementSuccessful() {
	cs.Successful++
}

// IncrementFailed increments the failed categorization count
func (cs *CategorizationStats) IncrementFailed() {
	cs.Failed++
}

// IncrementUncategorized increments the uncategorized count
func (cs *CategorizationStats) IncrementUncategorized() {
	cs.Uncategorized++
}

// NewCategorizationStats creates a new CategorizationStats instance
func NewCategorizationStats() *CategorizationStats {
	return &CategorizationStats{
		Total:         0,
		Successful:    0,
		Failed:        0,
		Uncategorized: 0,
	}
}
