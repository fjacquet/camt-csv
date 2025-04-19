// Package selmaparser provides functionality to parse and process Selma CSV files.
package selmaparser

import (
	"encoding/csv"
	"fjacquet/camt-csv/internal/models"
	"fmt"
	"os"
	"strconv"
	"strings"

	"fjacquet/camt-csv/internal/common"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

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

	// Use the common.ReadCSVFile function to read and parse the CSV
	selmaRows, err := common.ReadCSVFile[SelmaCSVRow](filePath)
	if err != nil {
		log.WithError(err).Error("Failed to read Selma CSV file")
		return nil, err
	}

	// Convert SelmaCSVRow objects to Transaction objects
	var transactions []models.Transaction
	for _, row := range selmaRows {
		// Skip empty rows
		if row.Date == "" || row.Name == "" {
			continue
		}

		// Convert Selma row to Transaction
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
	// Convert NumberOfShares from string to int
	shares, err := strconv.Atoi(row.NumberOfShares)
	if err != nil {
		shares = 0 // Default to 0 if conversion fails
	}

	// For Selma, create a description that includes ISIN and transaction type
	description := fmt.Sprintf("%s - %s (%s)", row.Name, row.TransactionType, row.ISIN)

	// Format date to standard DD.MM.YYYY format
	formattedDate := models.FormatDate(row.Date)

	transaction := models.Transaction{
		Date:           formattedDate,
		ValueDate:      formattedDate, // Use same date for ValueDate for Selma
		Description:    description,
		Amount:         row.TotalAmount,
		Currency:       row.Currency,
		NumberOfShares: shares,
		CreditDebit:    determineCreditDebit(row.TransactionType, row.TotalAmount),
		// Store transaction fee in the description if present
		Fund: row.Name, // Use Name as Fund for investment transactions
	}

	return transaction, nil
}

// determineCreditDebit determines if a transaction is a debit or credit
// based on transaction type and amount
func determineCreditDebit(transactionType, amount string) string {
	// For Selma, "BUY" typically means outgoing money (DBIT)
	// "SELL" typically means incoming money (CRDT)
	if transactionType == "BUY" {
		return "DBIT"
	} else if transactionType == "SELL" {
		return "CRDT"
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

// WriteToCSV writes a slice of Transaction structs to a CSV file.
//
// Parameters:
//   - transactions: Slice of Transaction structs to write
//   - csvFile: Path to the output CSV file
//
// Returns:
//   - error: Any error encountered during the writing process
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	log.WithFields(logrus.Fields{
		"file":  csvFile,
		"count": len(transactions),
	}).Info("Writing Selma transactions to CSV file")

	err := common.WriteTransactionsToCSV(transactions, csvFile)
	if err != nil {
		log.WithError(err).Error("Failed to write Selma CSV file")
		return err
	}

	log.Info("Successfully wrote Selma transactions to CSV file")
	return nil
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
	headers, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Error reading headers from CSV file")
		return false, err
	}

	// Check for required headers in Selma CSV format
	// These must match the field names in the SelmaCSVRow struct
	requiredHeaders := []string{"Date", "Name", "Transaction Type", "Total Amount", "Currency"}
	headerMap := make(map[string]bool)

	for _, header := range headers {
		headerMap[header] = true
	}

	for _, required := range requiredHeaders {
		if !headerMap[required] {
			log.WithField("missing_header", required).Debug("Missing required header")
			return false, nil
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
