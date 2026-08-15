package revolutinvestmentparser

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"strings"

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

// expectedHeaders are the columns a Revolut Investment CSV export starts with,
// in order. They are what distinguishes this format from every other CSV the
// tool accepts.
var expectedHeaders = []string{"Date", "Ticker", "Type", "Quantity", "Price per share", "Total Amount", "Currency", "FX Rate"}

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

	header, err := csv.NewReader(f).Read()
	if err != nil {
		return false, nil
	}

	if len(header) < len(expectedHeaders) {
		return false, nil
	}
	for i, expected := range expectedHeaders {
		if strings.TrimSpace(header[i]) != expected {
			return false, nil
		}
	}

	return true, nil
}
