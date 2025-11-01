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
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"

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

// Parse reads and parses a Selma CSV file from an io.Reader into a slice of Transaction objects.
// This is the standardized parser interface for reading Selma CSV files.
func Parse(r io.Reader) ([]models.Transaction, error) {
	log.Info("Reading Selma CSV from reader")

	// Buffer the reader content so we can validate and parse from the same data
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	// Check if the file format is valid first
	valid, err := validateFormat(strings.NewReader(string(data)))
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
		return models.Transaction{}, &parsererror.DataExtractionError{
			FilePath:       "(from reader)",
			FieldName:      "Amount",
			RawDataSnippet: row.Amount,
			Msg:            fmt.Sprintf("failed to parse amount: %v", err),
		}
	}

	transaction := models.Transaction{
		BookkeepingNumber: "",
		Date:              row.Date, // Keep original YYYY-MM-DD format
		ValueDate:         row.Date, // Use same date for ValueDate for Selma
		Description:       row.Description,
		Amount:            amount,
		Currency:          row.Currency,
		NumberOfShares:    shares,
		Fund:              row.Fund,
		CreditDebit:       determineCreditDebit(row.Description, row.Amount),
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
//
// Returns:
//   - []models.Transaction: The processed transactions with additional metadata
func ProcessTransactions(transactions []models.Transaction) []models.Transaction {
	log.WithField("count", len(transactions)).Info("Processing Selma transactions")

	processedTransactions := processTransactionsInternal(transactions)

	log.WithField("count", len(processedTransactions)).Info("Successfully processed Selma transactions")
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
func ConvertToCSV(inputFile, outputFile string) error {
	log.WithFields(logrus.Fields{
		"input":  inputFile,
		"output": outputFile,
	}).Info("Converting file to CSV")

	// Open the input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warnf("Failed to close file: %v", err)
		}
	}()

	// Parse the file
	transactions, err := Parse(file)
	if err != nil {
		return err
	}

	// Write to CSV
	if err := WriteToCSV(transactions, outputFile); err != nil {
		return err
	}

	log.WithFields(logrus.Fields{
		"count":  len(transactions),
		"input":  inputFile,
		"output": outputFile,
	}).Info("Successfully converted file to CSV")

	return nil
}

// validateFormat checks if a file is in valid Selma CSV format.
// It verifies the structure and required fields of the CSV file.
func validateFormat(r io.Reader) (bool, error) {
	log.Info("Validating Selma CSV format from reader")

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

	log.Debug("Selma CSV format validation successful")
	return true, nil
}
