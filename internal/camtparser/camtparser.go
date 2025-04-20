// Package camtparser provides functionality to parse and process CAMT.053 XML files.
package camtparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"gopkg.in/xmlpath.v2"
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
	log.WithField("file", xmlFile).Info("Parsing CAMT.053 XML file (XPath mode)")
	transactions, err := extractTransactionsFromXMLPath(xmlFile)
	if err != nil {
		log.WithError(err).Error("Failed to extract transactions with XPath")
		return nil, fmt.Errorf("error extracting transactions with XPath: %w", err)
	}
	log.WithField("count", len(transactions)).Info("Successfully extracted transactions from CAMT.053 file (XPath mode)")
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
// It uses xmlpath to check for the required elements in the CAMT.053 structure.
func ValidateFormat(xmlFile string) (bool, error) {
	log.WithField("file", xmlFile).Info("Validating CAMT.053 format")

	// Check if file exists
	_, err := os.Stat(xmlFile)
	if err != nil {
		log.WithError(err).Error("XML file does not exist")
		return false, fmt.Errorf("error checking XML file: %w", err)
	}

	// Open the file
	f, err := os.Open(xmlFile)
	if err != nil {
		log.WithError(err).Error("Failed to open XML file")
		return false, fmt.Errorf("error opening XML file: %w", err)
	}
	defer f.Close()

	// Parse the XML
	root, err := xmlpath.Parse(f)
	if err != nil {
		log.WithError(err).Debug("File is not valid XML")
		return false, nil // File is not valid XML, but we don't return an error
	}

	// Check for essential CAMT.053 elements
	// 1. Check if Document/BkToCstmrStmt exists
	path := xmlpath.MustCompile("//BkToCstmrStmt")
	if _, ok := path.String(root); !ok {
		log.Debug("Missing BkToCstmrStmt element, not a CAMT.053 file")
		return false, nil
	}

	// 2. Check if Document/BkToCstmrStmt/Stmt/Id exists
	path = xmlpath.MustCompile("//BkToCstmrStmt/Stmt/Id")
	if _, ok := path.String(root); !ok {
		log.Debug("Missing required Statement ID, not a valid CAMT.053 file")
		return false, nil
	}

	log.WithField("file", xmlFile).Info("File is a valid CAMT.053 XML")
	return true, nil
}
