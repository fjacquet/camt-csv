// Package selmaparser provides functionality to parse and process Selma CSV files.
package selmaparser

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Delimiter for Selma CSV output (default is ',')
var Delimiter rune = ','

func init() {
	if val := os.Getenv("CSV_DELIMITER"); val != "" {
		SetDelimiter([]rune(val)[0])
	}
}

// SetDelimiter allows setting the delimiter for CSV output
func SetDelimiter(delim rune) {
	Delimiter = delim
}

// SetLogger allows setting a configured logger for this package.
// This function enables integration with the application's logging system.
//
// Parameters:
//   - logger: A configured logrus.Logger instance. If nil, no change will occur.
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
	}
	common.SetLogger(logger)
}

// ParseFile reads and parses a Selma CSV file into a slice of Transaction objects.
// This is the standardized parser interface for reading Selma CSV files.
//
// Parameters:
//   - filePath: Path to the Selma CSV file to parse
//
// Returns:
//   - []models.Transaction: Slice of transaction objects extracted from the CSV
//   - error: Any error encountered during parsing
func ParseFile(filePath string) ([]models.Transaction, error) {
	return ReadSelmaCSV(filePath)
}

// ReadSelmaCSV reads and parses a Selma CSV file into a slice of Transaction objects.
// This is the main entry point for reading Selma CSV files.
//
// Parameters:
//   - filePath: Path to the Selma CSV file to parse
//
// Returns:
//   - []models.Transaction: Slice of transaction objects extracted from the CSV
//   - error: Any error encountered during parsing
func ReadSelmaCSV(filePath string) ([]models.Transaction, error) {
	log.WithField("filePath", filePath).Info("Reading Selma CSV file")

	// Check if the file format is valid first
	valid, err := ValidateFormat(filePath)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, err
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open Selma CSV file")
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // allow variable number of fields
	header, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Failed to read Selma CSV header")
		return nil, err
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
			if err.Error() == "EOF" {
				break
			}
			log.WithError(err).Warn("Skipping malformed CSV row")
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
			log.WithError(err).WithField("row", row).Warn("Failed to convert row to transaction")
			continue
		}
		transactions = append(transactions, tx)
	}

	// Process the transactions (categorize, associate related transactions)
	return ProcessTransactions(transactions), nil
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
		return models.Transaction{}, err
	}

	transaction := models.Transaction{
		Date:           row.Date, // Keep original YYYY-MM-DD format
		ValueDate:      row.Date, // Use same date for ValueDate for Selma
		Description:    row.Description,
		BookkeepingNo:  row.BookkeepingNo,
		Amount:         amount,
		Currency:       row.Currency,
		NumberOfShares: shares,
		Fund:           row.Fund,
		CreditDebit:    determineCreditDebit(row.Description, row.Amount),
	}

	return transaction, nil
}

// determineCreditDebit determines if a transaction is a debit or credit
// based on transaction type and amount
func determineCreditDebit(transactionType, amount string) string {
	// For Selma, "trade" with negative amount typically means outgoing money (DBIT)
	if transactionType == "trade" && strings.HasPrefix(amount, "-") {
		return "DBIT"
	} 

	// If we can't determine from type, try to use the amount sign
	if strings.HasPrefix(amount, "-") {
		return "DBIT"
	}

	// Default to credit for anything else
	return "CRDT"
}

// ProcessTransactions processes a slice of Transaction objects from Selma CSV data.
// It applies categorization and associates related transactions like stamp duties.
//
// Parameters:
//   - transactions: A slice of Transaction objects to process
//
// Returns:
//   - []models.Transaction: The processed transactions with additional metadata
func ProcessTransactions(transactions []models.Transaction) []models.Transaction {
	log.WithField("count", len(transactions)).Info("Processing Selma transactions")

	processedTransactions := processTransactionsInternal(transactions)

	log.WithField("count", len(processedTransactions)).Info("Successfully processed Selma transactions")
	return processedTransactions
}

// WriteToCSV writes a slice of Transaction objects to a CSV file in the standard format.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// ValidateFormat checks if a file is in valid Selma CSV format.
// It verifies the structure and required fields of the CSV file.
//
// Parameters:
//   - filePath: Path to the Selma CSV file to validate
//
// Returns:
//   - bool: True if the file is a valid Selma CSV, False otherwise
//   - error: Any error encountered during validation
func ValidateFormat(filePath string) (bool, error) {
	log.WithField("file", filePath).Info("Validating Selma CSV format")

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Error opening file for validation")
		return false, err
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read the header row
	header, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Error reading CSV header")
		return false, fmt.Errorf("error reading CSV header: %w", err)
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
			log.WithField("missing_header", required).Debug("Missing required header")
			return false, fmt.Errorf("input file is not in a valid format")
		}
	}

	log.Debug("Selma CSV format validation successful")
	return true, nil
}

// ConvertToCSV converts a Selma CSV file to the standard CSV format.
// This is a convenience function that combines ParseFile and WriteToCSV.
func ConvertToCSV(inputFile, outputFile string) error {
	return common.GeneralizedConvertToCSV(inputFile, outputFile, ParseFile, ValidateFormat)
}
