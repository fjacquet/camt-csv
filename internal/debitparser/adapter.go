package debitparser

import (
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
	"github.com/sirupsen/logrus"
)

// Adapter implements the parser.Parser interface for Visa Debit CSV files.
type Adapter struct {
	defaultParser parser.DefaultParser
}

// NewAdapter creates a new adapter for the debitparser.
func NewAdapter() parser.Parser {
	adapter := &Adapter{}

	// Set up the default parser with this adapter as implementation
	adapter.defaultParser = parser.DefaultParser{
		Logger: logrus.New(),
		Impl:   adapter,
	}

	return adapter
}

// ParseFile implements parser.Parser.ParseFile
// by delegating to the package-level function.
func (a *Adapter) ParseFile(filePath string) ([]models.Transaction, error) {
	return ParseFile(filePath)
}

// ValidateFormat implements parser.Parser.ValidateFormat
// by delegating to the package-level function.
func (a *Adapter) ValidateFormat(filePath string) (bool, error) {
	return ValidateFormat(filePath)
}

// ConvertToCSV implements parser.Parser.ConvertToCSV
// Uses the standardized implementation from DefaultParser.
func (a *Adapter) ConvertToCSV(inputFile, outputFile string) error {
	return a.defaultParser.ConvertToCSV(inputFile, outputFile)
}

// WriteToCSV implements parser.Parser.WriteToCSV
// Uses the standardized implementation from DefaultParser.
func (a *Adapter) WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return a.defaultParser.WriteToCSV(transactions, csvFile)
}

// SetLogger implements parser.Parser.SetLogger
// Uses the standardized implementation from DefaultParser.
func (a *Adapter) SetLogger(logger *logrus.Logger) {
	a.defaultParser.Logger = logger
	a.defaultParser.SetLogger(logger)
	SetLogger(logger)
}
