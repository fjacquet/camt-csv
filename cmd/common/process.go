// Package common contains shared functionality for command handlers
package common

import (
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
)

// ProcessFile processes a single file using the given parser.
func ProcessFile(parser models.Parser, inputFile, outputFile string, validate bool, log *logrus.Logger) {
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
func SaveMappings(log *logrus.Logger) {
	categorizerInstance := categorizer.NewCategorizer(nil, store.NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml"), log)
	err := categorizerInstance.SaveCreditorsToYAML()
	if err != nil {
		log.Warnf("Failed to save creditor mappings: %v", err)
	}

	err = categorizerInstance.SaveDebitorsToYAML()
	if err != nil {
		log.Warnf("Failed to save debitor mappings: %v", err)
	}
}
