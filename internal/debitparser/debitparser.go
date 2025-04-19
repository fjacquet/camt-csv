// Package debitparser provides functionality to parse Visa Debit CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Visa Debit CSV export files.
package debitparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/gocarina/gocsv"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

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

// SetLogger allows setting a configured logger
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
		common.SetLogger(logger)
	}
}

// ParseFile parses a Visa Debit CSV file and returns a slice of Transaction objects.
// This is the main entry point for parsing Visa Debit CSV files.
func ParseFile(filePath string) ([]models.Transaction, error) {
	log.WithField("file", filePath).Info("Parsing Visa Debit CSV file")

	// Check if the file format is valid
	valid, err := ValidateFormat(filePath)
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
	debitRows, err := common.ReadCSVFile[DebitCSVRow](filePath)
	if err != nil {
		log.WithError(err).Error("Failed to read Visa Debit CSV file")
		return nil, fmt.Errorf("error reading Visa Debit CSV: %w", err)
	}

	// Reset the CSV reader to default for other parsers
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		return csv.NewReader(in)
	})

	log.WithField("count", len(debitRows)).Info("Successfully read rows from CSV file")

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
			log.WithError(err).Warning("Failed to convert row to transaction, skipping")
			continue
		}
		
		transactions = append(transactions, tx)
	}
	
	log.WithField("count", len(transactions)).Info("Successfully parsed Visa Debit CSV file")
	return transactions, nil
}

// convertDebitRowToTransaction converts a DebitCSVRow to a Transaction
func convertDebitRowToTransaction(row DebitCSVRow) (models.Transaction, error) {
	// Simple validation
	if row.Datum == "" {
		return models.Transaction{}, fmt.Errorf("date is empty")
	}
	
	// Process amount and determine credit/debit
	var amount string
	var creditDebit string
	
	// The Montant field is used for the amount, which can be positive or negative
	// Negative amounts (-) are debits, positive are credits
	if row.Betrag == "" {
		// If amount is empty, default to 0
		amount = "0"
		creditDebit = "CRDT"
	} else {
		// Use StandardizeAmount to handle formatting (comma vs. decimal point)
		amount = models.StandardizeAmount(row.Betrag)
		
		// Determine credit/debit based on sign
		if strings.HasPrefix(row.Betrag, "-") {
			creditDebit = "DBIT"
			// Remove negative sign for consistency
			amount = strings.TrimPrefix(amount, "-")
		} else {
			creditDebit = "CRDT"
		}
	}
	
	// Format date to standard DD.MM.YYYY format
	formattedDate := models.FormatDate(row.Datum)
	
	// Extract just the business name from the description (after "PMT CARTE " prefix)
	description := row.Beneficiaire
	if strings.HasPrefix(description, "PMT CARTE ") {
		description = strings.TrimPrefix(description, "PMT CARTE ")
	}
	
	// Create and return the transaction
	transaction := models.Transaction{
		Date:           formattedDate,
		ValueDate:      formattedDate, // Use same date for value date since there's no separate value date
		Description:    description,
		Amount:         amount,
		Currency:       row.Waehrung,
		CreditDebit:    creditDebit,
		EntryReference: row.Referenznummer,
		Status:         row.StatusKontofuhrung,
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
	log.WithField("file", filePath).Info("Validating Visa Debit CSV format")

	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open file for validation")
		return false, fmt.Errorf("error opening file for validation: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';' // CSV uses semicolon as delimiter
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Failed to read CSV header")
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
			log.WithField("column", required).Info("Required column not found")
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
		log.WithError(err).Error("Error reading CSV record during validation")
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
	log.WithFields(logrus.Fields{
		"inputDir":  inputDir,
		"outputDir": outputDir,
	}).Info("Starting batch conversion of Visa Debit CSV files")

	// Check if input directory exists
	inputInfo, err := os.Stat(inputDir)
	if err != nil {
		log.WithError(err).Error("Failed to access input directory")
		return 0, fmt.Errorf("error accessing input directory: %w", err)
	}
	if !inputInfo.IsDir() {
		return 0, fmt.Errorf("input path is not a directory: %s", inputDir)
	}

	// Create output directory if it doesn't exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.WithError(err).Error("Failed to create output directory")
			return 0, fmt.Errorf("error creating output directory: %w", err)
		}
	}

	// Read input directory
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		log.WithError(err).Error("Failed to read input directory")
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
			continue
		}

		inputFile := fmt.Sprintf("%s/%s", inputDir, entry.Name())
		
		// Validate if file is in Visa Debit CSV format
		valid, err := ValidateFormat(inputFile)
		if err != nil {
			log.WithError(err).WithField("file", inputFile).Warning("Error validating file format, skipping")
			continue
		}
		if !valid {
			log.WithField("file", inputFile).Info("File is not a valid Visa Debit CSV, skipping")
			continue
		}

		// Define output file name (replace extension with _processed.csv)
		baseName := strings.TrimSuffix(entry.Name(), ".csv")
		outputFile := fmt.Sprintf("%s/%s_processed.csv", outputDir, baseName)

		// Convert the file
		err = ConvertToCSV(inputFile, outputFile)
		if err != nil {
			log.WithError(err).WithField("file", inputFile).Warning("Error converting file, skipping")
			continue
		}

		count++
	}

	log.WithField("count", count).Info("Batch conversion completed")
	return count, nil
}
