// Package camtparser provides functionality to parse and process CAMT.053 XML files.
package camtparser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"github.com/sirupsen/logrus"
)

// Use the centralized logger
var log = logging.GetLogger()
var activeParser parser.Parser
var currentParserType string = "iso20022" // Default parser type

// SetLogger allows setting a custom logger
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
		common.SetLogger(logger)
		categorizer.SetLogger(logger)

		// Initialize or update the parser with the new logger
		if activeParser == nil {
			activeParser = NewISO20022Adapter(logger)
		} else {
			activeParser.SetLogger(logger)
		}
	}
}

// ParseFile parses a CAMT.053 XML file and returns a slice of Transaction objects.
// This is the main entry point for parsing CAMT.053 XML files.
func ParseFile(xmlFile string) ([]models.Transaction, error) {
	// Check if file exists
	if _, err := os.Stat(xmlFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", xmlFile)
	}

	// If activeParser isn't initialized, create it first
	if activeParser == nil {
		activeParser = NewISO20022Adapter(log)
	}

	// Direct implementation for specific parser types to avoid circular dependencies
	_, ok := activeParser.(*Adapter)
	if ok {
		return parseFileISO20022(xmlFile)
	}

	// For other parser implementations use interface method
	return activeParser.ParseFile(xmlFile)
}

// WriteToCSV writes a slice of Transaction objects to a CSV file.
// It formats the transactions and applies categorization before writing.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// ConvertToCSV converts a CAMT.053 XML file to a CSV file.
// This is a convenience function that combines ParseFile and WriteToCSV.
func ConvertToCSV(xmlFile, csvFile string) error {
	// Initialize parser if not already initialized
	if activeParser == nil {
		activeParser = NewISO20022Adapter(log)
	}

	return activeParser.ConvertToCSV(xmlFile, csvFile)
}

// BatchConvert converts all XML files in a directory to CSV files.
// It processes all files with a .xml extension in the specified directory.
func BatchConvert(inputDir, outputDir string) (int, error) {
	log.WithFields(logrus.Fields{
		"inputDir":  inputDir,
		"outputDir": outputDir,
	}).Info("Batch converting CAMT.053 XML files")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Read input directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read input directory: %w", err)
	}

	// Process each XML file
	count := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			continue
		}

		inputFile := filepath.Join(inputDir, file.Name())
		outputFile := filepath.Join(outputDir, strings.TrimSuffix(file.Name(), ".xml")+".csv")

		// Validate that it's a CAMT.053 file
		isValid, err := ValidateFormat(inputFile)
		if err != nil {
			log.WithError(err).WithField("file", inputFile).Error("Error validating file format")
			continue
		}
		if !isValid {
			log.WithField("file", inputFile).Debug("Skipping non-CAMT.053 file")
			continue
		}

		// Convert the file
		if err := ConvertToCSV(inputFile, outputFile); err != nil {
			log.WithError(err).WithField("file", inputFile).Error("Failed to convert file")
			continue
		}

		count++
	}

	log.WithField("count", count).Info("Batch conversion completed")
	return count, nil
}

// ValidateFormat checks if a file is a valid CAMT.053 XML file.
// It uses the active parser to validate the file structure.
func ValidateFormat(xmlFile string) (bool, error) {
	log.WithField("file", xmlFile).Info("Validating CAMT.053 format")

	// Check if file exists
	if _, err := os.Stat(xmlFile); os.IsNotExist(err) {
		return false, fmt.Errorf("file does not exist: %s", xmlFile)
	}

	// If activeParser isn't initialized or to prevent circular dependency
	// do a simple check without invoking the Parser interface
	if activeParser == nil {
		// Open the file
		file, err := os.Open(xmlFile)
		if err != nil {
			return false, err
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Warnf("Failed to close file: %v", err)
			}
		}()

		// Read enough bytes to check for XML header and CAMT053 identifiers
		buffer := make([]byte, 4096)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return false, err
		}
		xmlHeader := string(buffer[:n])

		// Basic checks for XML format
		if !strings.Contains(xmlHeader, "<?xml") {
			return false, nil
		}

		// Check for CAMT.053 specific elements (simplified for quick validation)
		isCamt := strings.Contains(xmlHeader, "Document") &&
			(strings.Contains(xmlHeader, "BkToCstmrStmt") ||
				strings.Contains(xmlHeader, "camt.053"))

		if isCamt {
			log.WithField("file", xmlFile).Info("File is a valid CAMT.053 XML")
		} else {
			log.WithField("file", xmlFile).Info("File is not a valid CAMT.053 XML")
		}

		return isCamt, nil
	}

	// Direct implementation for specific parser types to avoid circular dependencies
	_, ok := activeParser.(*Adapter)
	if ok {
		isValid, err := activeParser.ValidateFormat(xmlFile)
		if err != nil {
			return false, err
		}

		if isValid {
			log.WithField("file", xmlFile).Info("File is a valid CAMT.053 XML")
		} else {
			log.WithField("file", xmlFile).Info("File is not a valid CAMT.053 XML")
		}

		return isValid, nil
	}

	// For other parser implementations
	isValid, err := activeParser.ValidateFormat(xmlFile)
	if err != nil {
		return false, err
	}

	if isValid {
		log.WithField("file", xmlFile).Info("File is a valid CAMT.053 XML")
	} else {
		log.WithField("file", xmlFile).Info("File is not a valid CAMT.053 XML")
	}

	return isValid, nil
}

// SetParserType only supports ISO20022 parser after removing XPath implementation
// Valid types: "iso20022"
func SetParserType(newParserType string) {
	lowerType := strings.ToLower(newParserType)

	// Skip if already using the requested parser type
	if currentParserType == "iso20022" && activeParser != nil {
		return
	}

	// Always use ISO20022 parser now that XPath is removed
	currentParserType = "iso20022"

	// Create the ISO20022 parser
	adapter := NewAdapter()
	activeParser = adapter

	if lowerType != "iso20022" && lowerType != "" {
		log.Warnf("Parser type '%s' is not supported. Only ISO20022 parser is available.", newParserType)
	}
}
