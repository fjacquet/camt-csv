// Package common contains shared functionality for command handlers
package common

import (
	"fmt"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/parser"

	"github.com/sirupsen/logrus"
)

// ProcessFile is a helper function that handles the common validation and conversion logic
func ProcessFile(p parser.Parser, input, output string, validate bool, log *logrus.Logger) error {
	if validate {
		log.Debug("Validating file format...")
		valid, err := p.ValidateFormat(input)
		if err != nil {
			return fmt.Errorf("error validating file: %w", err)
		}
		if !valid {
			return fmt.Errorf("the file is not in a valid format")
		}
		log.Debug("Validation successful.")
	}

	if err := p.ConvertToCSV(input, output); err != nil {
		return fmt.Errorf("error converting file: %w", err)
	}
	
	return nil
}

// SaveCategoryMappings saves any updated creditor/debitor mappings to disk
// This ensures that AI-categorized transactions are properly stored for future use
func SaveCategoryMappings(log *logrus.Logger) {
	// Save creditor mappings if needed
	if err := categorizer.SaveCreditorsToYAML(); err != nil {
		log.Warnf("Failed to save creditor mappings: %v", err)
	}
	
	// Save debitor mappings if needed
	if err := categorizer.SaveDebitorsToYAML(); err != nil {
		log.Warnf("Failed to save debitor mappings: %v", err)
	}
	
	log.Debug("Category mapping save complete")
}
