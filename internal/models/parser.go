package models

import (
	"context"
	"io"

	"fjacquet/camt-csv/internal/logging"
)

// Parser defines the interface for all parser implementations.
// Deprecated: Use the segregated interfaces from internal/parser package instead.
// This interface will be removed in v3.0.0 to follow Interface Segregation Principle.
//
// Migration example:
//
//	// Old code:
//	var p models.Parser
//
//	// New code:
//	var p parser.FullParser // or specific interfaces like parser.Parser, parser.Validator
type Parser interface {
	Parse(ctx context.Context, r io.Reader) ([]Transaction, error)
	ConvertToCSV(ctx context.Context, inputFile, outputFile string) error
	WriteToCSV(transactions []Transaction, csvFile string) error
	SetLogger(logger logging.Logger)
	ValidateFormat(file string) (bool, error)
	BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error)
}
