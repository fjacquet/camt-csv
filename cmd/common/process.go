// Package common contains shared functionality for command handlers
package common

import (
	"context"
	"errors"
	"fmt"
	"os"

	"fjacquet/camt-csv/internal/categorizer"
	internalcommon "fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/container"
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
func ProcessFileWithError(ctx context.Context, p parser.FullParser, inputFile, outputFile string, validate bool, log logging.Logger) error {
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

	if err := p.ConvertToCSV(ctx, inputFile, outputFile); err != nil {
		return fmt.Errorf("error converting to CSV: %w", err)
	}
	log.Info("Conversion completed successfully!")
	return nil
}

// ProcessFile processes a single file using the given parser with formatter support.
// Deprecated: Use ProcessFileWithError instead for better error handling and testability.
func ProcessFile(ctx context.Context, p parser.FullParser, inputFile, outputFile string, validate bool, log logging.Logger, c *container.Container, format string, dateFormat string) {
	if err := ProcessFileWithErrorFormatted(ctx, p, inputFile, outputFile, validate, log, c, format, dateFormat); err != nil {
		log.Fatalf("%v", err)
	}
}

// ProcessFileWithErrorFormatted processes a single file using the given parser with formatter support and returns an error on failure.
func ProcessFileWithErrorFormatted(ctx context.Context, p parser.FullParser, inputFile, outputFile string, validate bool, log logging.Logger, c *container.Container, format string, dateFormat string) error {
	// Set the logger on the parser using the new interface
	p.SetLogger(log)

	// Get formatter registry from container
	registry := c.GetFormatterRegistry()
	formatter, err := registry.Get(format)
	if err != nil {
		return fmt.Errorf("invalid format '%s': %w. Valid formats: standard, icompta", format, err)
	}

	// Get delimiter from formatter
	delimiter := formatter.Delimiter()
	log.WithField("format", format).WithField("delimiter", string(delimiter)).Info("Using output format")

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

	// Open and parse the input file
	file, err := os.Open(inputFile) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer file.Close()

	// Parse transactions
	transactions, err := p.Parse(ctx, file)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// Write transactions using the selected formatter
	if err := internalcommon.WriteTransactionsToCSVWithFormatter(transactions, outputFile, log, formatter, delimiter); err != nil {
		return fmt.Errorf("error writing CSV: %w", err)
	}

	log.Info("Conversion completed successfully!")
	return nil
}

// ProcessFileLegacyWithError processes a single file using the legacy parser interface and returns an error on failure.
// This is the preferred function for testable code.
//
// Deprecated: Use ProcessFileWithError with parser.FullParser instead.
func ProcessFileLegacyWithError(ctx context.Context, parser models.Parser, inputFile, outputFile string, validate bool, log logging.Logger) error {
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

	if err := parser.ConvertToCSV(ctx, inputFile, outputFile); err != nil {
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
//	err = fullParser.ConvertToCSV(ctx, inputFile, outputFile)
func ProcessFileLegacy(ctx context.Context, parser models.Parser, inputFile, outputFile string, validate bool, log logging.Logger) {
	if err := ProcessFileLegacyWithError(ctx, parser, inputFile, outputFile, validate, log); err != nil {
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
	categorizerInstance := categorizer.NewCategorizer(nil, store.NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml"), logging.NewLogrusAdapterFromLogger(log), false)
	err := categorizerInstance.SaveCreditorsToYAML()
	if err != nil {
		log.Warnf("Failed to save creditor mappings: %v", err)
	}

	err = categorizerInstance.SaveDebitorsToYAML()
	if err != nil {
		log.Warnf("Failed to save debitor mappings: %v", err)
	}
}
