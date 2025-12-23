// Package debitparser provides functionality to parse Visa Debit CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Visa Debit CSV export files.
package debitparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/gocarina/gocsv"
	"github.com/shopspring/decimal"
)

// DebitCSVRow represents a single row in a Visa Debit CSV file
// It uses struct tags for gocsv unmarshaling
type DebitCSVRow struct {
	Beneficiaire       string `csv:"Bénéficiaire"`
	Datum              string `csv:"Date"`
	Betrag             string `csv:"Montant"`
	Waehrung           string `csv:"Monnaie"`
	BuchungsNr         string `csv:"Buchungs-Nr."`
	Referenznummer     string `csv:"Referenznummer"`
	StatusKontofuhrung string `csv:"Status Kontoführung"`
}

// Parse parses a Visa Debit CSV file from an io.Reader and returns a slice of Transaction objects.
func Parse(r io.Reader, logger logging.Logger) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Parsing Visa Debit CSV from reader")

	// Configure gocsv for semicolon delimiter
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = ';' // CSV uses semicolon as delimiter
		return r
	})

	var debitRows []*DebitCSVRow
	if err := gocsv.Unmarshal(r, &debitRows); err != nil {
		logger.WithError(err).Error("Failed to read Visa Debit CSV from reader")
		return nil, fmt.Errorf("error reading Visa Debit CSV: %w", err)
	}

	// Reset the CSV reader to default for other parsers
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		return csv.NewReader(in)
	})

	logger.Info("Successfully read rows from CSV file",
		logging.Field{Key: "count", Value: len(debitRows)})

	// Convert DebitCSVRow objects to Transaction objects
	var transactions []models.Transaction
	for _, row := range debitRows {
		// Skip empty rows
		if row.Datum == "" {
			continue
		}

		// Convert Debit row to Transaction
		tx, err := convertDebitRowToTransaction(*row)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert row to transaction, skipping")
			continue
		}

		transactions = append(transactions, tx)
	}

	logger.Info("Successfully parsed Visa Debit CSV file",
		logging.Field{Key: "count", Value: len(transactions)})
	return transactions, nil
}

// ParseFile parses a Visa Debit CSV file and returns a slice of Transaction objects.
// This is the main entry point for parsing Visa Debit CSV files.
func ParseFile(filePath string) ([]models.Transaction, error) {
	return ParseFileWithLogger(filePath, nil)
}

func ParseFileWithLogger(filePath string, logger logging.Logger) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.WithField("file", filePath).Info("Parsing Visa Debit CSV file")

	// Check if the file format is valid
	valid, err := ValidateFormatWithLogger(filePath, logger)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid Visa Debit CSV format")
	}

	// Configure gocsv for semicolon delimiter
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = ';' // CSV uses semicolon as delimiter
		return r
	})

	// Use common.ReadCSVFile to read the CSV with the semicolon delimiter
	debitRows, err := common.ReadCSVFile[DebitCSVRow](filePath, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to read Visa Debit CSV file")
		return nil, fmt.Errorf("error reading Visa Debit CSV: %w", err)
	}

	// Reset the CSV reader to default for other parsers
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		return csv.NewReader(in)
	})

	logger.Info("Successfully read rows from CSV file",
		logging.Field{Key: "count", Value: len(debitRows)})

	// Convert DebitCSVRow objects to Transaction objects
	var transactions []models.Transaction
	for _, row := range debitRows {
		// Skip empty rows
		if row.Datum == "" {
			continue
		}

		// Convert Debit row to Transaction
		tx, err := convertDebitRowToTransaction(row)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert row to transaction, skipping")
			continue
		}

		transactions = append(transactions, tx)
	}

	logger.Info("Successfully parsed Visa Debit CSV file",
		logging.Field{Key: "count", Value: len(transactions)})
	return transactions, nil
}

// convertDebitRowToTransaction converts a DebitCSVRow to a Transaction
func convertDebitRowToTransaction(row DebitCSVRow) (models.Transaction, error) {
	// Simple validation
	if row.Datum == "" {
		return models.Transaction{}, fmt.Errorf("date is empty")
	}

	// Process amount and determine credit/debit
	var amount decimal.Decimal
	var creditDebit string

	// The Montant field is used for the amount, which can be positive or negative
	// Negative amounts (-) are debits, positive are credits
	if row.Betrag == "" {
		// If amount is empty, default to 0
		amount = decimal.NewFromFloat(0)
		creditDebit = models.TransactionTypeCredit
	} else {
		// Use StandardizeAmount to handle formatting (comma vs. decimal point)
		var err error
		amount, err = decimal.NewFromString(models.StandardizeAmount(row.Betrag))
		if err != nil {
			return models.Transaction{}, fmt.Errorf("error parsing amount: %w", err)
		}

		// Determine credit/debit based on sign
		if strings.HasPrefix(row.Betrag, "-") {
			creditDebit = models.TransactionTypeDebit
			// Remove negative sign for consistency
			amount = amount.Abs()
		} else {
			creditDebit = models.TransactionTypeCredit
		}
	}

	// Parse date to time.Time (will be formatted automatically during CSV marshaling)
	parsedDate, err := dateutils.ParseDateString(row.Datum)
	if err != nil {
		// If parsing fails, use current date as fallback
		parsedDate = time.Now()
	}

	// Extract just the business name from the description (after "PMT CARTE " prefix)
	description := row.Beneficiaire
	if strings.HasPrefix(description, "PMT CARTE ") {
		description = strings.TrimPrefix(description, "PMT CARTE ")
	}

	// Use TransactionBuilder for consistent transaction construction
	builder := models.NewTransactionBuilder().
		WithDatetime(parsedDate).
		WithValueDatetime(parsedDate).
		WithDescription(description).
		WithAmount(amount, row.Waehrung).
		WithEntryReference(row.Referenznummer).
		WithStatus(row.StatusKontofuhrung)

	// Set transaction direction
	if creditDebit == models.TransactionTypeDebit {
		builder = builder.AsDebit()
	} else {
		builder = builder.AsCredit()
	}

	// Build the transaction
	transaction, err := builder.Build()
	if err != nil {
		return models.Transaction{}, fmt.Errorf("error building transaction: %w", err)
	}

	return transaction, nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file.
// It formats the transactions and applies categorization before writing.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	// Check if transactions is nil or empty
	if transactions == nil {
		return fmt.Errorf("transactions is nil")
	}
	if len(transactions) == 0 {
		return fmt.Errorf("no transactions to write")
	}

	// Use the common CSV writer
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// ValidateFormat checks if the file is a valid Visa Debit CSV file.
func ValidateFormat(filePath string) (bool, error) {
	return ValidateFormatWithLogger(filePath, nil)
}

// ValidateFormatWithLogger checks if the file is a valid Visa Debit CSV file with logger.
func ValidateFormatWithLogger(filePath string, logger logging.Logger) (bool, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Validating Visa Debit CSV format",
		logging.Field{Key: "file", Value: filePath})

	file, err := os.Open(filePath) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		logger.WithError(err).Error("Failed to open file for validation")
		return false, fmt.Errorf("error opening file for validation: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close file")
		}
	}()

	reader := csv.NewReader(file)
	reader.Comma = ';' // CSV uses semicolon as delimiter

	// Read header
	header, err := reader.Read()
	if err != nil {
		logger.WithError(err).Error("Failed to read CSV header")
		return false, fmt.Errorf("error reading CSV header: %w", err)
	}

	// Check if required columns exist
	requiredColumns := []string{"Bénéficiaire", "Date", "Montant", "Monnaie"}
	columnMap := make(map[string]bool)

	for _, col := range header {
		columnMap[col] = true
	}

	for _, required := range requiredColumns {
		if !columnMap[required] {
			logger.Info("Required column not found",
				logging.Field{Key: "column", Value: required})
			return false, nil
		}
	}

	// Read at least one record to validate format
	_, err = reader.Read()
	if err == io.EOF {
		// Empty file but valid format
		return true, nil
	}
	if err != nil {
		logger.WithError(err).Error("Error reading CSV record during validation")
		return false, fmt.Errorf("error reading CSV record during validation: %w", err)
	}

	return true, nil
}

// ConvertToCSV converts a Visa Debit CSV file to the standard CSV format.
// This is a convenience function that combines ParseFile and WriteToCSV.
func ConvertToCSV(inputFile, outputFile string) error {
	return common.GeneralizedConvertToCSV(inputFile, outputFile, ParseFile, ValidateFormat)
}

// BatchConvert converts all CSV files in a directory to the standard CSV format.
// It processes all files with a .csv extension in the specified directory.
func BatchConvert(inputDir, outputDir string) (int, error) {
	return BatchConvertWithLogger(inputDir, outputDir, nil)
}

// BatchConvertWithLogger converts all CSV files in a directory with logger.
func BatchConvertWithLogger(inputDir, outputDir string, logger logging.Logger) (int, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Starting batch conversion of Visa Debit CSV files",
		logging.Field{Key: "inputDir", Value: inputDir},
		logging.Field{Key: "outputDir", Value: outputDir})

	// Check if input directory exists
	inputInfo, err := os.Stat(inputDir)
	if err != nil {
		logger.WithError(err).Error("Failed to access input directory")
		return 0, fmt.Errorf("error accessing input directory: %w", err)
	}
	if !inputInfo.IsDir() {
		return 0, fmt.Errorf("input path is not a directory: %s", inputDir)
	}

	// Create output directory if it doesn't exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0750); err != nil {
			logger.WithError(err).Error("Failed to create output directory")
			return 0, fmt.Errorf("error creating output directory: %w", err)
		}
	}

	// Read input directory
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		logger.WithError(err).Error("Failed to read input directory")
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
			continue
		}

		inputFile := fmt.Sprintf("%s/%s", inputDir, entry.Name())

		// Validate if file is in Visa Debit CSV format
		valid, err := ValidateFormatWithLogger(inputFile, logger)
		if err != nil {
			logger.WithError(err).Warn("Error validating file format, skipping",
				logging.Field{Key: "file", Value: inputFile})
			continue
		}
		if !valid {
			logger.Info("File is not a valid Visa Debit CSV, skipping",
				logging.Field{Key: "file", Value: inputFile})
			continue
		}

		// Define output file name (replace extension with _processed.csv)
		baseName := strings.TrimSuffix(entry.Name(), ".csv")
		outputFile := fmt.Sprintf("%s/%s_processed.csv", outputDir, baseName)

		// Convert the file
		err = ConvertToCSV(inputFile, outputFile)
		if err != nil {
			logger.WithError(err).Warn("Error converting file, skipping",
				logging.Field{Key: "file", Value: inputFile})
			continue
		}

		count++
	}

	logger.Info("Batch conversion completed",
		logging.Field{Key: "count", Value: count})
	return count, nil
}
