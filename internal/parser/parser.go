// Package parser provides a common interface for all parsers in the application.
package parser

import (
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"
	"os"

	"github.com/sirupsen/logrus"
)

// Parser defines the common interface for all file parsers in the application.
type Parser interface {
	// ParseFile parses a file and returns a slice of Transaction objects.
	ParseFile(filePath string) ([]models.Transaction, error)

	// ValidateFormat checks if a file is in the correct format for this parser.
	ValidateFormat(filePath string) (bool, error)

	// ConvertToCSV converts a file to CSV format.
	ConvertToCSV(inputFile, outputFile string) error

	// WriteToCSV writes transactions to a CSV file.
	WriteToCSV(transactions []models.Transaction, csvFile string) error

	// SetLogger configures the logger for this parser.
	SetLogger(logger *logrus.Logger)
}

// DefaultParser provides standard implementations of common Parser methods.
// This can be used as a base for composing parsers.
type DefaultParser struct {
	Logger *logrus.Logger
	Impl   Parser // The implementation to delegate to for required methods
}

// WriteToCSV provides a standard implementation using common.WriteTransactionsToCSV
func (p *DefaultParser) WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// ConvertToCSV provides a standard implementation for file conversion
func (p *DefaultParser) ConvertToCSV(inputFile, outputFile string) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return parsererror.FileNotFoundError(inputFile)
	}

	// The implementation needs to provide these methods
	// Use the implementation's ValidateFormat method
	valid, err := p.Impl.ValidateFormat(inputFile)
	if err != nil {
		return err
	}

	if !valid {
		return parsererror.InvalidFormatError(inputFile, "")
	}

	// Use the implementation's ParseFile method
	transactions, err := p.Impl.ParseFile(inputFile)
	if err != nil {
		return err
	}

	// Use our standard WriteToCSV implementation
	return p.WriteToCSV(transactions, outputFile)
}

// SetLogger provides a standard implementation for logger configuration
func (p *DefaultParser) SetLogger(logger *logrus.Logger) {
	if logger != nil {
		p.Logger = logger
		common.SetLogger(logger)
	}
}

// RunConvert is a helper function that encapsulates the common validate-and-convert flow.
func RunConvert(p Parser, inputFile, outputFile string, validate bool) error {
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return parsererror.FileNotFoundError(inputFile)
	}

	if validate {
		valid, err := p.ValidateFormat(inputFile)
		if err != nil {
			return err
		}
		if !valid {
			return parsererror.InvalidFormatError(inputFile, "")
		}
	}

	return p.ConvertToCSV(inputFile, outputFile)
}

// ErrInvalidFormat is returned when a file is not in the expected format.
// Deprecated: Use parsererror.ErrInvalidFormat instead.
var ErrInvalidFormat = parsererror.ErrInvalidFormat

// NewError creates a new error with the given text.
// Deprecated: Use the appropriate function from parsererror package instead.
func NewError(text string) error {
	return parsererror.ParsingError(text, nil)
}
