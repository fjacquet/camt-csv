package debitparser

import (
	"context"
	"io"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the parser.FullParser interface for Visa Debit CSV files.
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
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithCategorizer(ctx, r, a.GetLogger(), a.GetCategorizer())
}

// ValidateFormat checks if a file is a valid Visa Debit CSV file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	return ValidateFormat(file)
}
