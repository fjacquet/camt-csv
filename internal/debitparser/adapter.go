package debitparser

import (
	"fmt"
	"io"
	"os"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the models.Parser interface for Visa Debit CSV files.
type Adapter struct {
	parser.BaseParser
}

// NewAdapter creates a new adapter for the debitparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(r io.Reader) ([]models.Transaction, error) {
	return Parse(r, a.GetLogger())
}

// ConvertToCSV implements models.Parser.ConvertToCSV
func (a *Adapter) ConvertToCSV(inputFile, outputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			a.GetLogger().WithError(err).Warn("Failed to close input file",
				logging.Field{Key: "file", Value: inputFile})
		}
	}()

	transactions, err := a.Parse(file)
	if err != nil {
		return err
	}

	return a.WriteToCSV(transactions, outputFile)
}



// ValidateFormat checks if a file is a valid Visa Debit CSV file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	return ValidateFormat(file)
}

// BatchConvert converts all CSV files in a directory to the standard CSV format.
func (a *Adapter) BatchConvert(inputDir, outputDir string) (int, error) {
	return BatchConvert(inputDir, outputDir)
}
