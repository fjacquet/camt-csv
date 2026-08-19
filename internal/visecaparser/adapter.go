package visecaparser

import (
	"context"
	"io"
	"os"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the parser.FullParser interface for Viseca CSV exports.
type Adapter struct {
	parser.BaseParser

	// keepPayments retains the monthly card settlement rows; see Options.
	keepPayments bool
}

// NewAdapter creates a new adapter for the visecaparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// SetKeepPayments controls whether the monthly card settlement rows are
// imported. They are dropped by default because the bank statement already
// carries the same payments.
func (a *Adapter) SetKeepPayments(keep bool) { a.keepPayments = keep }

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithOptions(ctx, r, a.GetLogger(), a.GetCategorizer(),
		Options{KeepPayments: a.keepPayments})
}

// ValidateFormat checks if a file is a valid Viseca CSV export.
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

	return validateFormat(f, a.GetLogger())
}
