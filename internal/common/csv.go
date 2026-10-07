// Package common provides shared functionality across different parsers.
package common

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"fjacquet/camt-csv/internal/formatter"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// Note: Removed global logger in favor of dependency injection

// Delimiter is the CSV delimiter used for output (immutable, use config for customization)
const Delimiter = models.DefaultCSVDelimiter

// WriteTransactionsToCSVWithFormatter writes transactions to a CSV file using a custom formatter.
// This function enables format-specific output (standard CSV, iCompta, etc.) through the
// OutputFormatter interface.
//
// Parameters:
// - transactions: slice of Transaction objects to write
// - csvFile: path to the output CSV file
// - logger: logging instance (will create default if nil)
// - formatter: OutputFormatter implementation for controlling format
// - delimiter: CSV delimiter to use (overrides formatter.Delimiter() if different)
//
// The formatter parameter controls column layout and value formatting.
// The delimiter parameter allows runtime override of the formatter's preferred delimiter.
//
// Returns:
// - error: nil on success, or an error describing what went wrong
func WriteTransactionsToCSVWithFormatter(
	transactions []models.Transaction,
	csvFile string,
	logger logging.Logger,
	formatter formatter.OutputFormatter,
	delimiter rune,
) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	if transactions == nil {
		return fmt.Errorf("cannot write nil transactions to CSV")
	}
	if len(transactions) == 0 {
		logger.WithField("file", csvFile).Info("No transactions found, skipping output file")
		return nil
	}

	logger.WithFields(
		logging.Field{Key: "file", Value: csvFile},
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "delimiter", Value: string(delimiter)},
	).Info("Writing transactions to CSV file with custom formatter")

	// Create the directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := os.MkdirAll(dir, models.PermissionDirectory); err != nil {
		logger.WithError(err).Error("Failed to create directory")
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Copy transactions to avoid mutating the caller's slice
	prepared := make([]models.Transaction, len(transactions))
	copy(prepared, transactions)
	for i := range prepared {
		prepared[i].UpdateNameFromParties()
		prepared[i].UpdateRecipientFromPayee()
		prepared[i].UpdateDebitCreditAmounts()
	}

	// Format transactions using the provided formatter
	rows, err := formatter.Format(prepared)
	if err != nil {
		logger.WithError(err).Error("Failed to format transactions")
		return fmt.Errorf("error formatting transactions: %w", err)
	}

	// Create the file
	file, err := os.Create(csvFile) // #nosec G304 -- CLI tool requires user-provided output paths
	if err != nil {
		logger.WithError(err).Error("Failed to create CSV file")
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close file")
		}
	}()

	// Configure CSV writer with the specified delimiter
	csvWriter := csv.NewWriter(file)
	csvWriter.Comma = delimiter

	// Write header from formatter
	if err := csvWriter.Write(formatter.Header()); err != nil {
		logger.WithError(err).Error("Failed to write CSV header")
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write formatted rows
	for _, row := range rows {
		if err := csvWriter.Write(row); err != nil {
			logger.WithError(err).Error("Failed to write CSV record")
			return fmt.Errorf("error writing CSV record: %w", err)
		}
	}

	// Flush the writer
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		logger.WithError(err).Error("Failed to flush CSV writer")
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	logger.WithFields(
		logging.Field{Key: "file", Value: csvFile},
		logging.Field{Key: "count", Value: len(transactions)},
	).Info("Successfully wrote transactions to CSV file with custom formatter")

	return nil
}
