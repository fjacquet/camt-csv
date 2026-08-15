package revolutinvestmentparser

import (
	"context"
	"encoding/csv"
	"io"
	"os"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the parser.FullParser interface for Revolut investment CSV files.
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
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithCategorizer(ctx, r, a.GetLogger(), a.GetCategorizer())
}

// ConvertToCSV implements parser.FullParser.ConvertToCSV
func (a *Adapter) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
	return a.ConvertToCSVDefault(ctx, inputFile, outputFile, a.Parse)
}

// ValidateFormat checks if a file is a valid Revolut Investment CSV file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	f, err := os.Open(file) // #nosec G304 -- CLI tool requires user-provided file paths
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
