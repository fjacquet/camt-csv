// Package revolutparser provides functionality to parse Revolut CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Revolut CSV export files.
package revolutparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// RevolutCSVRow represents a single row in a Revolut CSV file
// It uses struct tags for gocsv unmarshaling
type RevolutCSVRow struct {
	Type          string `csv:"Type"`
	Product       string `csv:"Product"`
	StartedDate   string `csv:"Started Date"`
	CompletedDate string `csv:"Completed Date"`
	Description   string `csv:"Description"`
	Amount        string `csv:"Amount"`
	Fee           string `csv:"Fee"`
	Currency      string `csv:"Currency"`
	State         string `csv:"State"`
	Balance       string `csv:"Balance"`
}

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

	// Check if the file format is valid
	valid, err := ValidateFormat(filePath)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid Revolut CSV format")
	}

	// Use common.ReadCSVFile to read the CSV
	revolutRows, err := common.ReadCSVFile[RevolutCSVRow](filePath)
	if err != nil {
		log.WithError(err).Error("Failed to read Revolut CSV file")
		return nil, fmt.Errorf("error reading Revolut CSV: %w", err)
	}

	// Convert RevolutCSVRow objects to Transaction objects
	var transactions []models.Transaction
	for _, row := range revolutRows {
		// Skip empty rows
		if row.CompletedDate == "" || row.Description == "" {
			continue
		}

		// Only include completed transactions
		if row.State != "COMPLETED" {
			continue
		}

		// Convert Revolut row to Transaction
		tx, err := convertRevolutRowToTransaction(row)
		if err != nil {
			log.WithError(err).WithField("row", row).Warn("Failed to convert row to transaction")
			continue
		}

		transactions = append(transactions, tx)
	}

	log.WithField("count", len(transactions)).Info("Successfully parsed Revolut CSV file")
	return transactions, nil
}

// convertRevolutRowToTransaction converts a RevolutCSVRow to a Transaction
func convertRevolutRowToTransaction(row RevolutCSVRow) (models.Transaction, error) {
	// Determine credit/debit based on amount sign
	creditDebit := "CRDT" // Default to credit
	amount := row.Amount

	if strings.HasPrefix(amount, "-") {
		creditDebit = "DBIT"
		// Remove the negative sign for consistency
		amount = strings.TrimPrefix(amount, "-")
	}

	// Format dates to DD.MM.YYYY format for consistency
	completedDate := models.FormatDate(row.CompletedDate)
	startedDate := models.FormatDate(row.StartedDate)

	transaction := models.Transaction{
		Date:           completedDate,
		ValueDate:      startedDate,
		Description:    row.Description,
		Amount:         amount,
		Currency:       row.Currency,
		CreditDebit:    creditDebit,
		Status:         row.State,
		NumberOfShares: 0, // Revolut transactions don't have shares
	}

	return transaction, nil
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
