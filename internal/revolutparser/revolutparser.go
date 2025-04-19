// Package revolutparser provides functionality to parse Revolut CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Revolut CSV export files.
package revolutparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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

// ParseFile parses a Revolut CSV file and returns a slice of Transaction objects.
// This is the main entry point for parsing Revolut CSV files.
func ParseFile(filePath string) ([]models.Transaction, error) {
	log.WithField("file", filePath).Info("Parsing Revolut CSV file")

	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open Revolut CSV file")
		return nil, fmt.Errorf("error opening Revolut CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
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
	requiredColumns := []string{"Started Date", "Description", "Amount", "Currency", "State"}
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

		// Process Started Date to remove time part
		dateStr := record[indexMap["Started Date"]]
		parsedDate, err := time.Parse("2006-01-02 15:04:05", dateStr)
		if err != nil {
			log.WithError(err).WithField("date", dateStr).Warning("Failed to parse date, using as-is")
			// Use the original string if parsing fails
		} else {
			// Format as YYYY-MM-DD
			dateStr = parsedDate.Format("2006-01-02")
		}

		// Determine credit/debit based on amount sign
		amount := record[indexMap["Amount"]]
		creditDebit := "CRDT" // Default to credit
		// If amount starts with minus sign, it's a debit
		if strings.HasPrefix(amount, "-") {
			creditDebit = "DBIT"
			// Remove the minus sign for standard format
			amount = amount[1:]
		}

		tx := models.Transaction{
			Date:        dateStr,
			ValueDate:   dateStr, // Use the same date for ValueDate
			Description: record[indexMap["Description"]],
			Amount:      amount,
			Currency:    record[indexMap["Currency"]],
			CreditDebit: creditDebit,
			Status:      record[indexMap["State"]],
		}

		transactions = append(transactions, tx)
	}

	log.WithField("count", len(transactions)).Info("Successfully extracted transactions from Revolut CSV file")
	return transactions, nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file.
// It formats the transactions and applies categorization before writing.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// ValidateFormat checks if the file is a valid Revolut CSV file.
func ValidateFormat(filePath string) (bool, error) {
	log.WithField("file", filePath).Info("Validating Revolut CSV format")

	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open file for validation")
		return false, fmt.Errorf("error opening file for validation: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Failed to read CSV header")
		return false, fmt.Errorf("error reading CSV header: %w", err)
	}
	
	// Required columns for a valid Revolut CSV
	requiredColumns := []string{
		"Type", "Product", "Started Date", "Description", 
		"Amount", "Currency", "State",
	}
	
	// Map header columns to check if all required ones exist
	headerMap := make(map[string]bool)
	for _, col := range header {
		headerMap[col] = true
	}
	
	// Check if all required columns exist
	for _, requiredCol := range requiredColumns {
		if !headerMap[requiredCol] {
			log.WithField("column", requiredCol).Info("Required column missing from Revolut CSV")
			return false, nil
		}
	}
	
	// Check at least one data row is present
	_, err = reader.Read()
	if err == io.EOF {
		log.Info("Revolut CSV file is empty (header only)")
		return false, nil
	} else if err != nil {
		log.WithError(err).Error("Error reading CSV record")
		return false, fmt.Errorf("error reading CSV record: %w", err)
	}
	
	log.Info("File is a valid Revolut CSV")
	return true, nil
}

// ConvertToCSV converts a Revolut CSV file to the standard CSV format.
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
	}).Info("Batch converting Revolut CSV files")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.WithError(err).Error("Failed to create output directory")
		return 0, fmt.Errorf("error creating output directory: %w", err)
	}

	// Get all CSV files in input directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		log.WithError(err).Error("Failed to read input directory")
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	// Process each CSV file
	var processed int
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
			continue
		}
		
		inputPath := fmt.Sprintf("%s/%s", inputDir, file.Name())
		
		// Validate if it's a Revolut CSV file
		isValid, err := ValidateFormat(inputPath)
		if err != nil {
			log.WithError(err).WithField("file", inputPath).Warning("Error validating file, skipping")
			continue
		}
		
		if !isValid {
			log.WithField("file", inputPath).Info("Not a valid Revolut CSV file, skipping")
			continue
		}
		
		// Create output file path
		baseName := file.Name()
		baseNameWithoutExt := strings.TrimSuffix(baseName, ".csv")
		outputPath := fmt.Sprintf("%s/%s-standardized.csv", outputDir, baseNameWithoutExt)

		// Convert the file
		if err := ConvertToCSV(inputPath, outputPath); err != nil {
			log.WithFields(logrus.Fields{
				"file":  inputPath,
				"error": err,
			}).Warning("Failed to convert file, skipping")
			continue
		}
		processed++
	}

	log.WithField("count", processed).Info("Batch conversion completed")
	return processed, nil
}
