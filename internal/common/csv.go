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
	"github.com/sirupsen/logrus"
)

var log = logging.GetLogger()

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

// SetLogger allows setting a configured logger
func SetLogger(logger *logrus.Logger) {
	if logger == nil {
		return // Don't change the logger if nil is passed
	}
	log = logger
}

// ReadCSVFile reads CSV data into a slice of structs using gocsv
// This is a generic function that can be used by any parser
// TCSVRow is the struct type that maps to the CSV columns
func ReadCSVFile[TCSVRow any](filePath string) ([]TCSVRow, error) {
	log.WithField("file", filePath).Info("Reading CSV file")

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open CSV file")
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Warn("Failed to close file")
		}
	}()

	// Parse the CSV into structs
	var rows []TCSVRow
	if err := gocsv.UnmarshalFile(file, &rows); err != nil {
		log.WithError(err).Error("Failed to parse CSV file")
		return nil, fmt.Errorf("error parsing CSV file: %w", err)
	}

	log.WithField("count", len(rows)).Info("Successfully read CSV data")
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
	if transactions == nil {
		return fmt.Errorf("cannot write nil transactions to CSV")
	}

	log.WithFields(logrus.Fields{
		"file":  csvFile,
		"count": len(transactions),
	}).Info("Writing transactions to CSV file")

	// Create the directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		log.WithError(err).Error("Failed to create directory")
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create the file
	file, err := os.Create(csvFile)
	if err != nil {
		log.WithError(err).Error("Failed to create CSV file")
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Warn("Failed to close file")
		}
	}()

	// Update date formats and ensure derived fields are correctly set
	for i := range transactions {
		// Normalize dates to DD.MM.YYYY format
		if transactions[i].Date != "" {
			transactions[i].Date = models.FormatDate(transactions[i].Date)
		}
		if transactions[i].ValueDate != "" {
			transactions[i].ValueDate = models.FormatDate(transactions[i].ValueDate)
		}

		// Update various derived fields
		transactions[i].UpdateNameFromParties()
		transactions[i].UpdateRecipientFromPayee()
		transactions[i].UpdateDebitCreditAmounts()

		// Set DebitFlag based on transactions[i].CreditDebit or amount sign
		if transactions[i].CreditDebit == "DBIT" || transactions[i].Amount.IsNegative() {
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

	// Marshal the transactions
	if err := gocsv.MarshalCSV(transactions, gocsv.NewSafeCSVWriter(csvWriter)); err != nil {
		log.WithError(err).Error("Failed to marshal transactions to CSV")
		return fmt.Errorf("error writing CSV data: %w", err)
	}

	log.WithFields(logrus.Fields{
		"file":  csvFile,
		"count": len(transactions),
	}).Info("Successfully wrote transactions to CSV file")

	return nil
}

// ExportTransactionsToCSV exports a slice of transactions to a CSV file
func ExportTransactionsToCSV(transactions []models.Transaction, csvFile string) error {
	if transactions == nil {
		return fmt.Errorf("cannot write nil transactions to CSV")
	}

	log.WithFields(logrus.Fields{
		"count":     len(transactions),
		"file":      csvFile,
		"delimiter": string(Delimiter),
	}).Info("Exporting transactions to CSV file using WriteTransactionsToCSV")

	// Use the primary function for writing transactions to ensure consistency
	return WriteTransactionsToCSV(transactions, csvFile)
}

// GeneralizedConvertToCSV is a utility function that combines parsing and writing to CSV
// This is used by parsers implementing the standard interface
func GeneralizedConvertToCSV(
	inputFile string,
	outputFile string,
	parseFunc func(string) ([]models.Transaction, error),
	validateFunc func(string) (bool, error),
) error {
	log.WithFields(logrus.Fields{
		"input":  inputFile,
		"output": outputFile,
	}).Info("Converting file to CSV")

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
	if err := WriteTransactionsToCSV(transactions, outputFile); err != nil {
		return fmt.Errorf("error writing transactions to CSV: %w", err)
	}

	log.WithFields(logrus.Fields{
		"input":  inputFile,
		"output": outputFile,
		"count":  len(transactions),
	}).Info("Successfully converted file to CSV")

	return nil
}
