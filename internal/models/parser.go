package models

import (
	"context"
	"io"

	"fjacquet/camt-csv/internal/logging"
)

// Parser defines the combined interface for parser implementations.
// For new code, prefer the segregated interfaces from internal/parser package
// (parser.Parser, parser.Validator, parser.FullParser, etc.).
type Parser interface {
	Parse(ctx context.Context, r io.Reader) ([]Transaction, error)
	ConvertToCSV(ctx context.Context, inputFile, outputFile string) error
	WriteToCSV(transactions []Transaction, csvFile string) error
	SetLogger(logger logging.Logger)
	ValidateFormat(file string) (bool, error)
	BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error)
}
