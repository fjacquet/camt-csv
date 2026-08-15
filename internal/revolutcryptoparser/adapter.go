package revolutcryptoparser

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

// Adapter implements the parser.FullParser interface for Revolut Crypto CSV files.
type Adapter struct {
	parser.BaseParser
}

// NewAdapter creates a new Adapter for the revolutcryptoparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithCategorizer(r, a.GetLogger(), a.GetCategorizer())
}

// ConvertToCSV implements parser.FullParser.ConvertToCSV.
func (a *Adapter) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
	return a.ConvertToCSVDefault(ctx, inputFile, outputFile, a.Parse)
}

// ValidateFormat checks if a file is a valid Revolut Crypto CSV file.
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
	if err != nil || len(header) < 7 {
		return false, nil
	}
	// Check for the three distinctive headers that differ from standard Revolut CSV
	required := map[string]bool{"Symbol": false, "Type": false, "Date": false}
	for _, h := range header {
		if _, ok := required[strings.TrimSpace(h)]; ok {
			required[strings.TrimSpace(h)] = true
		}
	}
	for _, found := range required {
		if !found {
			return false, nil
		}
	}
	return true, nil
}
