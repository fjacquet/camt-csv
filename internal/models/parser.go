package models

import (
	"io"

	"fjacquet/camt-csv/internal/logging"
)

// Parser defines the interface for all parser implementations.
type Parser interface {
	Parse(r io.Reader) ([]Transaction, error)
	ConvertToCSV(inputFile, outputFile string) error
	WriteToCSV(transactions []Transaction, csvFile string) error
	SetLogger(logger logging.Logger)
	ValidateFormat(file string) (bool, error)
	BatchConvert(inputDir, outputDir string) (int, error)
}
