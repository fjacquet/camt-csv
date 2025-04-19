// Package debitparser provides functionality to parse Visa Debit CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Visa Debit CSV export files.
package debitparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

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

	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open Visa Debit CSV file")
		return nil, fmt.Errorf("error opening Visa Debit CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';' // CSV uses semicolon as delimiter
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Failed to read CSV header")
		return nil, fmt.Errorf("error reading CSV header: %w", err)
	}
	
	// Map column indices
	indexMap := make(map[string]int)
	for i, columnName := range header {
		indexMap[columnName] = i
	}
	
	// Verify required columns exist
	requiredColumns := []string{"Bénéficiaire", "Date", "Montant", "Monnaie"}
	for _, col := range requiredColumns {
		if _, exists := indexMap[col]; !exists {
			log.WithField("column", col).Error("Required column not found in CSV")
			return nil, fmt.Errorf("required column not found in CSV: %s", col)
		}
	}

	// Parse transactions
	var transactions []models.Transaction
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.WithError(err).Error("Error reading CSV record")
			return nil, fmt.Errorf("error reading CSV record: %w", err)
		}

		// Get data from CSV
		beneficiary := record[indexMap["Bénéficiaire"]]
		date := record[indexMap["Date"]]
		amount := record[indexMap["Montant"]]
		currency := record[indexMap["Monnaie"]]

		// Process beneficiary to remove "PMT CARTE " prefix if present
		if strings.HasPrefix(beneficiary, "PMT CARTE ") {
			beneficiary = strings.TrimPrefix(beneficiary, "PMT CARTE ")
		}

		// Determine credit/debit based on amount sign
		creditDebit := "CRDT" // Default to credit
		// If amount starts with minus sign, it's a debit
		if strings.HasPrefix(amount, "-") {
			creditDebit = "DBIT"
			// Remove the minus sign for standard format
			amount = strings.TrimPrefix(amount, "-")
		}

		// Replace comma with dot in amount (European format to standard)
		amount = strings.Replace(amount, ",", ".", 1)

		// Create transaction
		tx := models.Transaction{
			Date:        date,
			ValueDate:   date, // Use the same date for ValueDate
			Description: beneficiary,
			Amount:      amount,
			Currency:    currency,
			CreditDebit: creditDebit,
			Payee:       beneficiary, // Use beneficiary as payee for categorization
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file.
// It formats the transactions and applies categorization before writing.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
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
