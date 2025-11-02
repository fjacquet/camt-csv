// Package common provides shared functionality across different parsers.
package common

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/gocarina/gocsv"
)

// Note: Removed global logger in favor of dependency injection

// Global CSV delimiter - can be configured via centralized config or environment variable
var Delimiter rune = ','

func init() {
	// Fallback to environment variable for backward compatibility
	if val := os.Getenv("CSV_DELIMITER"); val != "" {
		// Use first rune only
		SetDelimiter([]rune(val)[0])
	}
}

// SetDelimiter allows setting the delimiter for CSV output
func SetDelimiter(delim rune) {
	Delimiter = delim
	gocsv.TagSeparator = fmt.Sprintf("%c", delim)
}

// ReadCSVFile reads CSV data into a slice of structs using gocsv
// This is a generic function that can be used by any parser
// TCSVRow is the struct type that maps to the CSV columns
func ReadCSVFile[TCSVRow any](filePath string, logger logging.Logger) ([]TCSVRow, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.WithField("file", filePath).Info("Reading CSV file")

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		logger.WithError(err).Error("Failed to open CSV file")
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close file")
		}
	}()

	// Parse the CSV into structs
	var rows []TCSVRow
	if err := gocsv.UnmarshalFile(file, &rows); err != nil {
		logger.WithError(err).Error("Failed to parse CSV file")
		return nil, fmt.Errorf("error parsing CSV file: %w", err)
	}

	logger.WithField("count", len(rows)).Info("Successfully read CSV data")
	return rows, nil
}

// WriteTransactionsToCSV writes transactions to a CSV file in a standardized format.
// All parsers should use this function to ensure consistent CSV output.
//
// Parameters:
// - transactions: slice of Transaction objects to write
// - csvFile: path to the output CSV file
//
// Returns:
// - error: nil on success, or an error describing what went wrong
func WriteTransactionsToCSV(transactions []models.Transaction, csvFile string) error {
	return WriteTransactionsToCSVWithLogger(transactions, csvFile, nil)
}

// WriteTransactionsToCSVWithLogger writes transactions to a CSV file with a logger
func WriteTransactionsToCSVWithLogger(transactions []models.Transaction, csvFile string, logger logging.Logger) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	if transactions == nil {
		return fmt.Errorf("cannot write nil transactions to CSV")
	}

	logger.WithFields(
		logging.Field{Key: "file", Value: csvFile},
		logging.Field{Key: "count", Value: len(transactions)},
	).Info("Writing transactions to CSV file")

	// Create the directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := os.MkdirAll(dir, models.PermissionDirectory); err != nil {
		logger.WithError(err).Error("Failed to create directory")
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create the file
	file, err := os.Create(csvFile)
	if err != nil {
		logger.WithError(err).Error("Failed to create CSV file")
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close file")
		}
	}()

	// Update date formats and ensure derived fields are correctly set
	for i := range transactions {
		// Dates are now time.Time and will be formatted automatically during CSV marshaling

		// Update various derived fields
		transactions[i].UpdateNameFromParties()
		transactions[i].UpdateRecipientFromPayee()
		transactions[i].UpdateDebitCreditAmounts()

		// Set DebitFlag based on transactions[i].CreditDebit or amount sign
		if transactions[i].CreditDebit == models.TransactionTypeDebit || transactions[i].Amount.IsNegative() {
			transactions[i].DebitFlag = true
		} else {
			transactions[i].DebitFlag = false
		}

		// Ensure all decimal values have 2 decimal places
		// This is needed for proper CSV formatting that passes tests
		transactions[i].Amount = models.ParseAmount(transactions[i].Amount.StringFixed(2))
		transactions[i].Debit = models.ParseAmount(transactions[i].Debit.StringFixed(2))
		transactions[i].Credit = models.ParseAmount(transactions[i].Credit.StringFixed(2))
		transactions[i].AmountExclTax = models.ParseAmount(transactions[i].AmountExclTax.StringFixed(2))
		transactions[i].AmountTax = models.ParseAmount(transactions[i].AmountTax.StringFixed(2))
		transactions[i].TaxRate = models.ParseAmount(transactions[i].TaxRate.StringFixed(2))
		transactions[i].Fees = models.ParseAmount(transactions[i].Fees.StringFixed(2))
		transactions[i].OriginalAmount = models.ParseAmount(transactions[i].OriginalAmount.StringFixed(2))
		transactions[i].ExchangeRate = models.ParseAmount(transactions[i].ExchangeRate.StringFixed(2))
	}

	// Configure CSV writer with custom delimiter
	csvWriter := csv.NewWriter(file)
	csvWriter.Comma = Delimiter

	// Write header manually to ensure correct order
	header := []string{
		"BookkeepingNumber", "Status", "Date", "ValueDate", "Name", "PartyName", "PartyIBAN",
		"Description", "RemittanceInfo", "Amount", "CreditDebit", "IsDebit", "Debit", "Credit", "Currency",
		"AmountExclTax", "AmountTax", "TaxRate", "Recipient", "InvestmentType", "Number", "Category",
		"Type", "Fund", "NumberOfShares", "Fees", "IBAN", "EntryReference", "Reference",
		"AccountServicer", "BankTxCode", "OriginalCurrency", "OriginalAmount", "ExchangeRate",
	}
	if err := csvWriter.Write(header); err != nil {
		logger.WithError(err).Error("Failed to write CSV header")
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write each transaction using custom MarshalCSV method
	for _, transaction := range transactions {
		record, err := transaction.MarshalCSV()
		if err != nil {
			logger.WithError(err).Error("Failed to marshal transaction to CSV")
			return fmt.Errorf("error marshaling transaction: %w", err)
		}
		if err := csvWriter.Write(record); err != nil {
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
	).Info("Successfully wrote transactions to CSV file")

	return nil
}

// ExportTransactionsToCSV exports a slice of transactions to a CSV file
func ExportTransactionsToCSV(transactions []models.Transaction, csvFile string) error {
	return ExportTransactionsToCSVWithLogger(transactions, csvFile, nil)
}

// ExportTransactionsToCSVWithLogger exports transactions with a logger
func ExportTransactionsToCSVWithLogger(transactions []models.Transaction, csvFile string, logger logging.Logger) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	if transactions == nil {
		return fmt.Errorf("cannot write nil transactions to CSV")
	}

	logger.WithFields(
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "file", Value: csvFile},
		logging.Field{Key: "delimiter", Value: string(Delimiter)},
	).Info("Exporting transactions to CSV file using WriteTransactionsToCSV")

	// Use the primary function for writing transactions to ensure consistency
	return WriteTransactionsToCSVWithLogger(transactions, csvFile, logger)
}

// GeneralizedConvertToCSV is a utility function that combines parsing and writing to CSV
// This is used by parsers implementing the standard interface
func GeneralizedConvertToCSV(
	inputFile string,
	outputFile string,
	parseFunc func(string) ([]models.Transaction, error),
	validateFunc func(string) (bool, error),
) error {
	return GeneralizedConvertToCSVWithLogger(inputFile, outputFile, parseFunc, validateFunc, nil)
}

// GeneralizedConvertToCSVWithLogger converts with a logger
func GeneralizedConvertToCSVWithLogger(
	inputFile string,
	outputFile string,
	parseFunc func(string) ([]models.Transaction, error),
	validateFunc func(string) (bool, error),
	logger logging.Logger,
) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.WithFields(
		logging.Field{Key: "input_file", Value: inputFile},
		logging.Field{Key: "output_file", Value: outputFile},
	).Info("Converting file to CSV")

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Validate the file format if a validate function is provided
	if validateFunc != nil {
		isValid, err := validateFunc(inputFile)
		if err != nil {
			return fmt.Errorf("error validating file format: %w", err)
		}
		if !isValid {
			return fmt.Errorf("invalid file format: %s", inputFile)
		}
	}

	// Parse the input file
	transactions, err := parseFunc(inputFile)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// Write transactions to CSV
	if err := WriteTransactionsToCSVWithLogger(transactions, outputFile, logger); err != nil {
		return fmt.Errorf("error writing transactions to CSV: %w", err)
	}

	logger.WithFields(
		logging.Field{Key: "input_file", Value: inputFile},
		logging.Field{Key: "output_file", Value: outputFile},
		logging.Field{Key: "count", Value: len(transactions)},
	).Info("Successfully converted file to CSV")

	return nil
}
