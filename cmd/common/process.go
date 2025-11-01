// Package common contains shared functionality for command handlers
package common

import (
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
)

// ProcessFile processes a single file using the given parser.
func ProcessFile(p parser.FullParser, inputFile, outputFile string, validate bool, log logging.Logger) {
	// Set the logger on the parser using the new interface
	p.SetLogger(log)

	if validate {
		log.Info("Validating format...")
		valid, err := p.ValidateFormat(inputFile)
		if err != nil {
			log.Fatalf("Error validating file: %v", err)
		}
		if !valid {
			log.Fatal("The file is not in a valid format")
		}
		log.Info("Validation successful.")
	}

	err := p.ConvertToCSV(inputFile, outputFile)
	if err != nil {
		log.Fatalf("Error converting to CSV: %v", err)
	}
	log.Info("Conversion completed successfully!")
}

// ProcessFileLegacy processes a single file using the legacy parser interface.
// Deprecated: Use ProcessFile with parser.FullParser instead.
// This function will be removed in v2.0.0.
func ProcessFileLegacy(parser models.Parser, inputFile, outputFile string, validate bool, log logging.Logger) {
	// Set the logger on the parser using the new interface
	parser.SetLogger(log)

	if validate {
		log.Info("Validating format...")
		valid, err := parser.ValidateFormat(inputFile)
		if err != nil {
			log.Fatalf("Error validating file: %v", err)
		}
		if !valid {
			log.Fatal("The file is not in a valid format")
		}
		log.Info("Validation successful.")
	}

	err := parser.ConvertToCSV(inputFile, outputFile)
	if err != nil {
		log.Fatalf("Error converting to CSV: %v", err)
	}
	log.Info("Conversion completed successfully!")
}

// SaveMappings saves the creditor and debitor mappings.
// Deprecated: Use container-based categorizer instead.
// This function will be removed in v2.0.0.
func SaveMappings(log *logrus.Logger) {
	categorizerInstance := categorizer.NewCategorizer(nil, store.NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml"), logging.NewLogrusAdapterFromLogger(log))
	err := categorizerInstance.SaveCreditorsToYAML()
	if err != nil {
		log.Warnf("Failed to save creditor mappings: %v", err)
	}

	err = categorizerInstance.SaveDebitorsToYAML()
	if err != nil {
		log.Warnf("Failed to save debitor mappings: %v", err)
	}
}
