// Package selmaparser provides functionality to parse and process Selma CSV files.
package selmaparser

import (
	"fjacquet/camt-csv/internal/models"
	"github.com/sirupsen/logrus"
)

// Parser defines the interface for all Selma CSV parsers
type Parser interface {
	// ParseFile parses a Selma CSV file and returns a slice of Transaction objects
	ParseFile(filePath string) ([]models.Transaction, error)
	
	// ValidateFormat checks if a file is a valid Selma CSV file
	ValidateFormat(filePath string) (bool, error)
	
	// ConvertToCSV converts a Selma CSV file to the standard format
	ConvertToCSV(inputFile, outputFile string) error
	
	// WriteToCSV writes transactions to a CSV file
	WriteToCSV(transactions []models.Transaction, csvFile string) error
	
	// SetLogger sets a custom logger for the parser
	SetLogger(logger *logrus.Logger)
}

// Adapter implements the Parser interface by wrapping
// the package-level functions of selmaparser.
type Adapter struct {
	log *logrus.Logger
}

// ParseFile implements Parser.ParseFile
// by delegating to the package-level function.
func (a *Adapter) ParseFile(filePath string) ([]models.Transaction, error) {
	return ParseFile(filePath)
}

// ValidateFormat implements Parser.ValidateFormat
// by delegating to the package-level function.
func (a *Adapter) ValidateFormat(filePath string) (bool, error) {
	return ValidateFormat(filePath)
}

// ConvertToCSV implements Parser.ConvertToCSV
// by delegating to the package-level function.
func (a *Adapter) ConvertToCSV(inputFile, outputFile string) error {
	return ConvertToCSV(inputFile, outputFile)
}

// WriteToCSV implements Parser.WriteToCSV
// by delegating to the package-level function.
func (a *Adapter) WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return WriteToCSV(transactions, csvFile)
}

// SetLogger implements Parser.SetLogger
// by delegating to the package-level function.
func (a *Adapter) SetLogger(logger *logrus.Logger) {
	a.log = logger
	SetLogger(logger)
}

// NewAdapter creates a new adapter for the selmaparser.
func NewAdapter() Parser {
	return &Adapter{
		log: logrus.New(),
	}
}
