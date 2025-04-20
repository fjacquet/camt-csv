// Package common provides shared functionality across different parsers.
package common

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"fjacquet/camt-csv/internal/models"

	"github.com/gocarina/gocsv"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Delimiter for CSV output (default is ',')
var Delimiter rune = ','

func init() {
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
	if logger != nil {
		log = logger
	}
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
	defer file.Close()

	// Parse the CSV into a slice of structs
	var rows []TCSVRow
	if err := gocsv.Unmarshal(file, &rows); err != nil {
		log.WithError(err).Error("Failed to parse CSV file")
		return nil, fmt.Errorf("error parsing CSV file: %w", err)
	}

	log.WithField("count", len(rows)).Info("Successfully read rows from CSV file")
	return rows, nil
}

// WriteTransactionsToCSV is a generalized function to write transactions to CSV 
// All parsers can use this function.
func WriteTransactionsToCSV(transactions []models.Transaction, csvFile string) error {
	// Check for CSV_DELIMITER again to ensure it's applied
	if val := os.Getenv("CSV_DELIMITER"); val != "" {
		SetDelimiter([]rune(val)[0])
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}
	
	// Open or create the file
	file, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer file.Close()
	
	// Create CSV writer
	writer := csv.NewWriter(file)
	writer.Comma = Delimiter  // Set the configured delimiter
	defer writer.Flush()
	
	// Write header
	header := []string{
		"Date", "Description", "Amount", "Currency", "Category", 
		"Payer", "Payee", "Status", "ValueDate", "CreditDebit", 
		"IBAN", "EntryReference", "Reference", "BookkeepingNo", "Fund",
	}
	
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}
	
	// Process transactions
	for _, t := range transactions {
		// Format credit/debit indicator
		creditDebit := "Credit"
		if t.CreditDebit == "DBIT" {
			creditDebit = "Debit"
		}
		
		// Convert decimal.Decimal amount to string with 2 decimal places
		amountStr := t.Amount.StringFixed(2)
		
		// Write the row
		if err := writer.Write([]string{
			t.Date,
			t.Description,
			amountStr,
			t.Currency,
			t.Category,
			t.Payer,
			t.Payee,
			t.Status,
			t.ValueDate,
			creditDebit,
			t.IBAN,
			t.EntryReference,
			t.EntryReference, // Use EntryReference as Reference
			t.BookkeepingNo,
			t.Fund,
		}); err != nil {
			return fmt.Errorf("error writing transaction record: %w", err)
		}
	}
	
	return nil
}

// GeneralizedConvertToCSV is a utility function that combines parsing and writing to CSV
// This is used by parsers implementing the standard interface
func GeneralizedConvertToCSV(
	inputFile string, 
	outputFile string, 
	parseFunc func(string) ([]models.Transaction, error),
	validateFunc func(string) (bool, error),
) error {
	// Validate file format if a validation function is provided
	if validateFunc != nil {
		valid, err := validateFunc(inputFile)
		if err != nil {
			return fmt.Errorf("error validating input file: %w", err)
		}
		if !valid {
			return fmt.Errorf("input file is not in a valid format")
		}
	}

	// Parse the input file
	transactions, err := parseFunc(inputFile)
	if err != nil {
		return fmt.Errorf("error parsing input file: %w", err)
	}
	
	// Write the CSV file with the transactions
	if err := WriteTransactionsToCSV(transactions, outputFile); err != nil {
		return fmt.Errorf("error writing CSV file: %w", err)
	}
	
	return nil
}
