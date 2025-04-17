// Package selmaparser provides functionality for processing Selma investment CSV files.
// It allows parsing, converting and processing Selma CSV exports into standardized formats.
package selmaparser

import (
	"fjacquet/camt-csv/internal/models"

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
	log.WithField("file", filePath).Info("Reading Selma CSV file")
	
	transactions, err := readSelmaCSVFile(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to read Selma CSV file")
		return nil, err
	}
	
	log.WithField("count", len(transactions)).Info("Successfully read Selma CSV file")
	return transactions, nil
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

// WriteToCSV writes a slice of Transaction objects to a CSV file.
// This is the standardized parser interface for writing transaction data to CSV.
// 
// Parameters:
//   - transactions: A slice of Transaction objects to write to the CSV file
//   - filePath: Path where the CSV file should be written
//
// Returns:
//   - error: Any error encountered during the writing process
func WriteToCSV(transactions []models.Transaction, filePath string) error {
	return WriteSelmaCSV(filePath, transactions)
}

// WriteSelmaCSV writes a slice of Transaction structs to a CSV file in Selma format.
// This function handles the creation of the file and writing the transaction data.
// 
// Parameters:
//   - filePath: Path where the CSV file should be written
//   - transactions: A slice of Transaction objects to write to the CSV file
//
// Returns:
//   - error: Any error encountered during the writing process
func WriteSelmaCSV(filePath string, transactions []models.Transaction) error {
	log.WithFields(logrus.Fields{
		"file": filePath,
		"count": len(transactions),
	}).Info("Writing Selma transactions to CSV file")
	
	err := writeSelmaCSVFile(filePath, transactions)
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
	
	isValid, err := validateSelmaCSVFormat(filePath)
	if err != nil {
		log.WithError(err).Error("Error during Selma CSV validation")
		return false, err
	}
	
	if isValid {
		log.Info("File is in valid Selma CSV format")
	} else {
		log.Info("File is not in valid Selma CSV format")
	}
	
	return isValid, nil
}

// ConvertToCSV is the main function to convert a Selma CSV file to a standardized CSV file.
// It reads the source file, processes the transactions, and writes them to the destination.
// 
// Parameters:
//   - inputFile: Path to the Selma CSV file to be converted
//   - outputFile: Path where the resulting CSV file should be written
//
// Returns:
//   - error: Any error encountered during conversion
func ConvertToCSV(inputFile, outputFile string) error {
	log.WithFields(logrus.Fields{
		"input": inputFile,
		"output": outputFile,
	}).Info("Converting Selma CSV file")
	
	// Read the Selma CSV file
	transactions, err := ReadSelmaCSV(inputFile)
	if err != nil {
		return err
	}
	
	// Process the transactions 
	processedTransactions := ProcessTransactions(transactions)
	
	// Write the processed transactions to CSV
	return WriteToCSV(processedTransactions, outputFile)
}
