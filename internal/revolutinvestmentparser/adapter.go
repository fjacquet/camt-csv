package revolutinvestmentparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the models.Parser interface for Revolut investment CSV files.
type Adapter struct {
	parser.BaseParser
}

// NewAdapter creates a new adapter for the revolutinvestmentparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(r io.Reader) ([]models.Transaction, error) {
	return Parse(r)
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



// ValidateFormat checks if a file is a valid Revolut Investment CSV file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			a.GetLogger().WithError(err).Warn("Failed to close file during format validation",
				logging.Field{Key: "file", Value: file})
		}
	}()

	// For now, we'll just check if it's a valid CSV file
	// A more robust implementation would check for specific headers
	_, err = csv.NewReader(f).Read()
	return err == nil, nil
}

// BatchConvert is not implemented for this parser.
func (a *Adapter) BatchConvert(inputDir, outputDir string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}
