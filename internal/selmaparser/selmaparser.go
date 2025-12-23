// Package selmaparser provides functionality to parse and process Selma CSV files.
package selmaparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"

	"github.com/shopspring/decimal"
)

// Parse reads and parses a Selma CSV file from an io.Reader into a slice of Transaction objects.
// This is the standardized parser interface for reading Selma CSV files.
func Parse(r io.Reader, logger logging.Logger) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Reading Selma CSV from reader")

	// Buffer the reader content so we can validate and parse from the same data
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	// Check if the file format is valid first
	valid, err := validateFormat(strings.NewReader(string(data)), logger)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, &parsererror.InvalidFormatError{
			FilePath:       "(from reader)",
			ExpectedFormat: "Selma CSV",
			Msg:            "invalid Selma CSV format",
		}
	}

	// Parse the CSV data
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1 // allow variable number of fields
	header, err := reader.Read()
	if err != nil {
		return nil, &parsererror.ParseError{
			Parser: "Selma",
			Field:  "CSV header",
			Value:  "header row",
			Err:    err,
		}
	}

	// Map header fields to struct fields
	headerMap := make(map[int]string)
	for i, h := range header {
		headerMap[i] = h
	}

	var transactions []models.Transaction
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.WithError(err).Warn("Skipping malformed CSV row")
			continue
		}
		// Pad or truncate record to match header length
		if len(record) < len(header) {
			// Pad with empty strings
			padded := make([]string, len(header))
			copy(padded, record)
			record = padded
		} else if len(record) > len(header) {
			record = record[:len(header)]
		}

		// Map record to SelmaCSVRow
		row := SelmaCSVRow{}
		for i, val := range record {
			switch headerMap[i] {
			case "Date":
				row.Date = val
			case "Description":
				row.Description = val
			case "Bookkeeping No.":
				row.BookkeepingNo = val
			case "Fund":
				row.Fund = val
			case "Amount":
				row.Amount = val
			case "Currency":
				row.Currency = val
			case "Number of Shares":
				row.NumberOfShares = val
			}
		}
		if row.Date == "" || row.Description == "" {
			continue
		}
		tx, err := convertSelmaRowToTransaction(row)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert row to transaction",
				logging.Field{Key: "row", Value: row})
			continue
		}
		transactions = append(transactions, tx)
	}

	// Process the transactions (categorize, associate related transactions)
	return ProcessTransactions(transactions, logger), nil
}

// convertSelmaRowToTransaction converts a SelmaCSVRow to a Transaction
func convertSelmaRowToTransaction(row SelmaCSVRow) (models.Transaction, error) {
	// Convert NumberOfShares from string to int if not empty
	var shares int
	if row.NumberOfShares != "" {
		// Try to parse as float first since some values might have decimal points
		sharesFloat, err := strconv.ParseFloat(row.NumberOfShares, 64)
		if err == nil {
			shares = int(sharesFloat)
		} else {
			// If that fails, try parsing as int
			shares, err = strconv.Atoi(row.NumberOfShares)
			if err != nil {
				shares = 0 // Default to 0 if conversion fails
			}
		}
	}

	// For Selma CSV, we keep the date as is since it's already in YYYY-MM-DD format
	// We don't need to call models.FormatDate as that would convert to DD.MM.YYYY

	amount, err := decimal.NewFromString(row.Amount)
	if err != nil {
		return models.Transaction{}, &parsererror.DataExtractionError{
			FilePath:       "(from reader)",
			FieldName:      "Amount",
			RawDataSnippet: row.Amount,
			Msg:            fmt.Sprintf("failed to parse amount: %v", err),
		}
	}

	// Determine transaction direction
	creditDebit := determineCreditDebit(row.Description, row.Amount)
	isDebit := creditDebit == models.TransactionTypeDebit

	// Use TransactionBuilder for consistent transaction construction
	builder := models.NewTransactionBuilder().
		WithDate(row.Date).
		WithValueDate(row.Date).
		WithDescription(row.Description).
		WithAmount(amount, row.Currency).
		WithNumberOfShares(shares).
		WithFund(row.Fund)

	// Set transaction direction
	if isDebit {
		builder = builder.AsDebit()
	} else {
		builder = builder.AsCredit()
	}

	// Build the transaction
	transaction, err := builder.Build()
	if err != nil {
		return models.Transaction{}, &parsererror.DataExtractionError{
			FilePath:       "(from reader)",
			FieldName:      "Transaction",
			RawDataSnippet: fmt.Sprintf("Date: %s, Amount: %s", row.Date, row.Amount),
			Msg:            fmt.Sprintf("failed to build transaction: %v", err),
		}
	}

	return transaction, nil
}

// determineCreditDebit determines if a transaction is a debit or credit
// based on transaction type and amount
func determineCreditDebit(transactionType, amount string) string {
	// For Selma, "trade" with negative amount typically means outgoing money (DBIT)
	if transactionType == "trade" && strings.HasPrefix(amount, "-") {
		return models.TransactionTypeDebit
	}

	// If we can't determine from type, try to use the amount sign
	if strings.HasPrefix(amount, "-") {
		return models.TransactionTypeDebit
	}

	// Default to credit for anything else
	return models.TransactionTypeCredit
}

// ProcessTransactions processes a slice of Transaction objects from Selma CSV data.
// It applies categorization and associates related transactions like stamp duties.
//
// Parameters:
//   - transactions: A slice of Transaction objects to process
//   - logger: Logger instance for logging operations
//
// Returns:
//   - []models.Transaction: The processed transactions with additional metadata
func ProcessTransactions(transactions []models.Transaction, logger logging.Logger) []models.Transaction {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Processing Selma transactions",
		logging.Field{Key: "count", Value: len(transactions)})

	processedTransactions := processTransactionsInternal(transactions)

	logger.Info("Successfully processed Selma transactions",
		logging.Field{Key: "count", Value: len(processedTransactions)})
	return processedTransactions
}

// WriteToCSV writes a slice of Transaction objects to a CSV file in a simplified format
// that is specifically used by the Selma parser tests.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	// Delegate to standardized CSV writer in common
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// ConvertToCSV converts a Selma CSV file to the standard CSV format.
// This is a convenience function that combines Parse and WriteToCSV.
func ConvertToCSV(inputFile, outputFile string, logger logging.Logger) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Converting file to CSV",
		logging.Field{Key: "input", Value: inputFile},
		logging.Field{Key: "output", Value: outputFile})

	// Open the input file
	file, err := os.Open(inputFile) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("Failed to close file",
				logging.Field{Key: "error", Value: err})
		}
	}()

	// Parse the file
	transactions, err := Parse(file, logger)
	if err != nil {
		return err
	}

	// Write to CSV
	if err := WriteToCSV(transactions, outputFile); err != nil {
		return err
	}

	logger.Info("Successfully converted file to CSV",
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "input", Value: inputFile},
		logging.Field{Key: "output", Value: outputFile})

	return nil
}

// validateFormat checks if a file is in valid Selma CSV format.
// It verifies the structure and required fields of the CSV file.
func validateFormat(r io.Reader, logger logging.Logger) (bool, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Validating Selma CSV format from reader")

	// Create a CSV reader
	reader := csv.NewReader(r)

	// Read the header row
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return false, fmt.Errorf("CSV file is empty")
		}
		return false, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Define the required headers for a valid Selma CSV
	requiredHeaders := []string{
		"Date",
		"Description",
		"Bookkeeping No.",
		"Fund",
		"Amount",
		"Currency",
		"Number of Shares",
	}

	// Check if all required headers are present
	headerMap := make(map[string]bool)
	for _, h := range header {
		headerMap[h] = true
	}

	for _, required := range requiredHeaders {
		if !headerMap[required] {
			return false, &parsererror.ValidationError{
				FilePath: "(from reader)",
				Reason:   fmt.Sprintf("missing required header: %s", required),
			}
		}
	}

	logger.Debug("Selma CSV format validation successful")
	return true, nil
}
