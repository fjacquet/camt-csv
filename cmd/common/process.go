// Package common contains shared functionality for command handlers
package common

import (
	"context"
	"errors"
	"fmt"
	"os"

	internalcommon "fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"
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
// Calls ProcessFileWithErrorFormatted and calls log.Fatalf on error.
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
		return fmt.Errorf("invalid format '%s': %w. Valid formats: standard, icompta, jumpsoft", format, err)
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
	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Warn("Failed to close file")
		}
	}()

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
