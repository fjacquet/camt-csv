// Package common contains shared functionality for command handlers
package common

import (
	"errors"
	"fmt"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
)

// ErrInvalidFormat is returned when a file fails format validation.
var ErrInvalidFormat = errors.New("file is not in a valid format")

// ProcessFileWithError processes a single file using the given parser and returns an error on failure.
// This is the preferred function for testable code.
func ProcessFileWithError(p parser.FullParser, inputFile, outputFile string, validate bool, log logging.Logger) error {
	// Set the logger on the parser using the new interface
	p.SetLogger(log)

	if validate {
		log.Info("Validating format...")
		valid, err := p.ValidateFormat(inputFile)
		if err != nil {
			return fmt.Errorf("error validating file: %w", err)
		}
		if !valid {
			return ErrInvalidFormat
		}
		log.Info("Validation successful.")
	}

	if err := p.ConvertToCSV(inputFile, outputFile); err != nil {
		return fmt.Errorf("error converting to CSV: %w", err)
	}
	log.Info("Conversion completed successfully!")
	return nil
}

// ProcessFile processes a single file using the given parser.
// Deprecated: Use ProcessFileWithError instead for better error handling and testability.
func ProcessFile(p parser.FullParser, inputFile, outputFile string, validate bool, log logging.Logger) {
	if err := ProcessFileWithError(p, inputFile, outputFile, validate, log); err != nil {
		log.Fatalf("%v", err)
	}
}

// ProcessFileLegacyWithError processes a single file using the legacy parser interface and returns an error on failure.
// This is the preferred function for testable code.
//
// Deprecated: Use ProcessFileWithError with parser.FullParser instead.
func ProcessFileLegacyWithError(parser models.Parser, inputFile, outputFile string, validate bool, log logging.Logger) error {
	// Set the logger on the parser using the new interface
	parser.SetLogger(log)

	if validate {
		log.Info("Validating format...")
		valid, err := parser.ValidateFormat(inputFile)
		if err != nil {
			return fmt.Errorf("error validating file: %w", err)
		}
		if !valid {
			return ErrInvalidFormat
		}
		log.Info("Validation successful.")
	}

	if err := parser.ConvertToCSV(inputFile, outputFile); err != nil {
		return fmt.Errorf("error converting to CSV: %w", err)
	}
	log.Info("Conversion completed successfully!")
	return nil
}

// ProcessFileLegacy processes a single file using the legacy parser interface.
//
// Deprecated: Use ProcessFileWithError with parser.FullParser instead.
// This function will be removed in v3.0.0.
//
// Migration example:
//
//	// Old code:
//	ProcessFileLegacy(parser, inputFile, outputFile, validate, log)
//
//	// New code:
//	container, err := container.NewContainer(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fullParser, err := container.GetParser(parserType)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = fullParser.ConvertToCSV(inputFile, outputFile)
func ProcessFileLegacy(parser models.Parser, inputFile, outputFile string, validate bool, log logging.Logger) {
	if err := ProcessFileLegacyWithError(parser, inputFile, outputFile, validate, log); err != nil {
		log.Fatalf("%v", err)
	}
}

// SaveMappings saves the creditor and debitor mappings.
//
// Deprecated: Use container-based categorizer instead.
// This function will be removed in v3.0.0.
//
// Migration example:
//
//	// Old code:
//	SaveMappings(log)
//
//	// New code:
//	container, err := container.NewContainer(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	categorizer := container.GetCategorizer()
//	err = categorizer.SaveCreditorsToYAML()
//	if err != nil {
//	    log.Error("Failed to save creditor mappings", err)
//	}
//	err = categorizer.SaveDebitorsToYAML()
//	if err != nil {
//	    log.Error("Failed to save debtor mappings", err)
//	}
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
