// Package camtparser provides functionality to parse and process CAMT.053 XML files.
package camtparser

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// SetLogger allows setting a custom logger
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
		common.SetLogger(logger)
	}
}

// ParseFile parses a CAMT.053 XML file and returns a slice of Transaction objects.
// This is the main entry point for parsing CAMT.053 XML files.
func ParseFile(xmlFile string) ([]models.Transaction, error) {
	log.WithField("file", xmlFile).Info("Parsing CAMT.053 XML file")

	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		log.WithError(err).Error("Failed to read XML file")
		return nil, fmt.Errorf("error reading XML file: %w", err)
	}

	var camt053 models.CAMT053
	err = xml.Unmarshal(xmlData, &camt053)
	if err != nil {
		log.WithError(err).Error("Failed to unmarshal XML data")
		return nil, fmt.Errorf("error unmarshalling XML: %w", err)
	}

	transactions := extractTransactions(&camt053)
	log.WithField("count", len(transactions)).Info("Successfully extracted transactions from CAMT.053 file")

	return transactions, nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file.
// It formats the transactions and applies categorization before writing.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// ConvertToCSV converts a CAMT.053 XML file to a CSV file.
// This is a convenience function that combines ParseFile and WriteToCSV.
func ConvertToCSV(xmlFile, csvFile string) error {
	return common.GeneralizedConvertToCSV(xmlFile, csvFile, ParseFile, ValidateFormat)
}

// BatchConvert converts all XML files in a directory to CSV files.
// It processes all files with a .xml extension in the specified directory.
func BatchConvert(inputDir, outputDir string) (int, error) {
	log.WithFields(logrus.Fields{
		"inputDir":  inputDir,
		"outputDir": outputDir,
	}).Info("Batch converting CAMT.053 XML files")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.WithError(err).Error("Failed to create output directory")
		return 0, fmt.Errorf("error creating output directory: %w", err)
	}

	// Get all XML files in input directory
	files, err := filepath.Glob(filepath.Join(inputDir, "*.xml"))
	if err != nil {
		log.WithError(err).Error("Failed to read input directory")
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	// Process each XML file
	var processed int
	for _, file := range files {
		// Create output file path
		baseName := filepath.Base(file)
		baseNameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		outputFile := filepath.Join(outputDir, baseNameWithoutExt+".csv")

		// Convert the file
		if err := ConvertToCSV(file, outputFile); err != nil {
			log.WithFields(logrus.Fields{
				"file":  file,
				"error": err,
			}).Warning("Failed to convert file, skipping")
			continue
		}
		processed++
	}

	log.WithField("count", processed).Info("Batch conversion completed")
	return processed, nil
}

// ValidateFormat checks if a file is a valid CAMT.053 XML file.
// It tries to unmarshal the XML data and checks for the expected structure.
func ValidateFormat(xmlFile string) (bool, error) {
	log.WithField("file", xmlFile).Info("Validating CAMT.053 format")

	// Check if file exists
	_, err := os.Stat(xmlFile)
	if err != nil {
		log.WithError(err).Error("XML file does not exist")
		return false, fmt.Errorf("error checking XML file: %w", err)
	}

	// Read the file
	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		log.WithError(err).Error("Failed to read XML file")
		return false, fmt.Errorf("error reading XML file: %w", err)
	}

	var camt053 models.CAMT053
	if err := xml.Unmarshal(xmlData, &camt053); err != nil {
		return false, nil // File is not valid XML, but we don't return an error
	}

	// Check if the file has the required CAMT.053 elements
	if camt053.BkToCstmrStmt.Stmt.Id == "" {
		return false, nil
	}

	log.WithField("file", xmlFile).Info("File is a valid CAMT.053 XML")
	return true, nil
}
