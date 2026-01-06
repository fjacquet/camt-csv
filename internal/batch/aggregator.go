// Package batch provides functionality for batch processing and aggregation of financial files
package batch

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// DateRange represents a date range with start and end dates
type DateRange struct {
	Start time.Time
	End   time.Time
}

// String returns the date range in the format "YYYY-MM-DD_YYYY-MM-DD"
func (dr DateRange) String() string {
	if dr.Start.IsZero() || dr.End.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s_%s",
		dr.Start.Format("2006-01-02"),
		dr.End.Format("2006-01-02"))
}

// Merge combines this date range with another, returning the overall range
func (dr DateRange) Merge(other DateRange) DateRange {
	start := dr.Start
	end := dr.End

	// Handle zero times
	if dr.Start.IsZero() {
		start = other.Start
	} else if !other.Start.IsZero() && other.Start.Before(start) {
		start = other.Start
	}

	if dr.End.IsZero() {
		end = other.End
	} else if !other.End.IsZero() && other.End.After(end) {
		end = other.End
	}

	return DateRange{Start: start, End: end}
}

// FileGroup represents a group of files that belong to the same account
type FileGroup struct {
	AccountID string    // The account identifier
	Files     []string  // List of file paths
	DateRange DateRange // Overall date range for all files
}

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

// GroupFilesByAccount groups files by their account identifier
// It analyzes filenames to extract account information and groups files accordingly
func (ba *BatchAggregator) GroupFilesByAccount(files []string) ([]FileGroup, error) {
	accountGroups := make(map[string]*FileGroup)

	for _, file := range files {
		// Extract account identifier from filename
		accountID := common.ExtractAccountFromFilename(file)

		ba.logger.Debug("File mapped to account",
			logging.Field{Key: "file", Value: filepath.Base(file)},
			logging.Field{Key: "account", Value: accountID.ID},
			logging.Field{Key: "source", Value: accountID.Source})

		// Get or create file group for this account
		group, exists := accountGroups[accountID.ID]
		if !exists {
			group = &FileGroup{
				AccountID: accountID.ID,
				Files:     []string{},
				DateRange: DateRange{},
			}
			accountGroups[accountID.ID] = group
		}

		// Add file to group
		group.Files = append(group.Files, file)

		// Extract date range from filename if possible
		dateRange := ba.extractDateRangeFromFilename(file)
		group.DateRange = group.DateRange.Merge(dateRange)
	}

	// Convert map to slice and sort by account ID for consistent output
	var groups []FileGroup
	for _, group := range accountGroups {
		groups = append(groups, *group)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].AccountID < groups[j].AccountID
	})

	ba.logger.Info("Grouped files into account groups",
		logging.Field{Key: "total_files", Value: len(files)},
		logging.Field{Key: "account_groups", Value: len(groups)})

	return groups, nil
}

// extractDateRangeFromFilename attempts to extract date range from CAMT filename patterns
// Returns zero DateRange if no dates can be extracted
func (ba *BatchAggregator) extractDateRangeFromFilename(filename string) DateRange {
	baseName := filepath.Base(filename)

	// CAMT pattern: CAMT.053_{account}_{start_date}_{end_date}_{sequence}.{ext}
	if strings.HasPrefix(strings.ToUpper(baseName), "CAMT.053_") {
		parts := strings.Split(baseName, "_")
		if len(parts) >= 4 {
			// Try to parse start and end dates
			startDateStr := parts[2]
			endDateStr := parts[3]

			startDate, err1 := time.Parse("2006-01-02", startDateStr)
			endDate, err2 := time.Parse("2006-01-02", endDateStr)

			if err1 == nil && err2 == nil {
				return DateRange{Start: startDate, End: endDate}
			}
		}
	}

	// For other file types, return empty range
	return DateRange{}
}

// AggregateTransactions aggregates transactions from multiple files in a file group
// It sorts transactions chronologically and handles potential duplicates
func (ba *BatchAggregator) AggregateTransactions(group FileGroup, parseFunc func(string) ([]models.Transaction, error)) ([]models.Transaction, error) {
	var allTransactions []models.Transaction
	var sourceFiles []string

	ba.logger.Info("Aggregating transactions for account",
		logging.Field{Key: "account", Value: group.AccountID},
		logging.Field{Key: "file_count", Value: len(group.Files)})

	for _, file := range group.Files {
		ba.logger.Debug("Processing file", logging.Field{Key: "file", Value: filepath.Base(file)})

		transactions, err := parseFunc(file)
		if err != nil {
			ba.logger.Error("Failed to parse file",
				logging.Field{Key: "file", Value: file},
				logging.Field{Key: "error", Value: err})
			continue // Skip this file but continue with others
		}

		ba.logger.Debug("Loaded transactions from file",
			logging.Field{Key: "count", Value: len(transactions)},
			logging.Field{Key: "file", Value: filepath.Base(file)})

		allTransactions = append(allTransactions, transactions...)
		sourceFiles = append(sourceFiles, filepath.Base(file))
	}

	// Sort transactions chronologically by date
	ba.sortTransactionsChronologically(allTransactions)

	// Log potential duplicates (but keep all transactions as per requirements)
	ba.detectAndLogDuplicates(allTransactions, group.AccountID)

	ba.logger.Info("Aggregated transactions for account",
		logging.Field{Key: "total_transactions", Value: len(allTransactions)},
		logging.Field{Key: "account", Value: group.AccountID},
		logging.Field{Key: "source_files", Value: strings.Join(sourceFiles, ", ")})

	return allTransactions, nil
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
// This helps users identify overlapping data but doesn't remove duplicates
func (ba *BatchAggregator) detectAndLogDuplicates(transactions []models.Transaction, accountID string) {
	duplicateCount := 0

	// Simple duplicate detection: same date, amount, and party
	for i := 0; i < len(transactions)-1; i++ {
		for j := i + 1; j < len(transactions); j++ {
			tx1 := transactions[i]
			tx2 := transactions[j]

			// Check if transactions are potential duplicates
			if ba.arePotentialDuplicates(tx1, tx2) {
				duplicateCount++
				ba.logger.Warn("Potential duplicate transaction",
					logging.Field{Key: "account", Value: accountID},
					logging.Field{Key: "date", Value: tx1.Date.Format("2006-01-02")},
					logging.Field{Key: "amount", Value: tx1.Amount.String()},
					logging.Field{Key: "party", Value: tx1.GetCounterparty()})
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

// GenerateOutputFilename creates a filename for the consolidated output
// Format: {account_id}_{start_date}_{end_date}.csv
func (ba *BatchAggregator) GenerateOutputFilename(accountID string, dateRange DateRange) string {
	// Sanitize account ID for filesystem safety
	sanitizedAccountID := common.SanitizeAccountID(accountID)

	// If we have a valid date range, use it
	if !dateRange.Start.IsZero() && !dateRange.End.IsZero() {
		return fmt.Sprintf("%s_%s.csv", sanitizedAccountID, dateRange.String())
	}

	// Fallback: use just the account ID
	return fmt.Sprintf("%s.csv", sanitizedAccountID)
}

// GenerateSourceFileHeader creates a header comment listing source files
func (ba *BatchAggregator) GenerateSourceFileHeader(sourceFiles []string) string {
	if len(sourceFiles) == 0 {
		return ""
	}

	var header strings.Builder
	header.WriteString("# Consolidated from source files:\n")
	for _, file := range sourceFiles {
		header.WriteString(fmt.Sprintf("# - %s\n", file))
	}
	header.WriteString("# Generated on: ")
	header.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	header.WriteString("\n#\n")

	return header.String()
}

// CalculateDateRangeFromTransactions calculates the overall date range from a set of transactions
func (ba *BatchAggregator) CalculateDateRangeFromTransactions(transactions []models.Transaction) DateRange {
	if len(transactions) == 0 {
		return DateRange{}
	}

	start := transactions[0].Date
	end := transactions[0].Date

	for _, tx := range transactions {
		if tx.Date.Before(start) {
			start = tx.Date
		}
		if tx.Date.After(end) {
			end = tx.Date
		}
	}

	return DateRange{Start: start, End: end}
}
