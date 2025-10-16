package models

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Parser defines the interface for all parser implementations.
type Parser interface {
	Parse(r io.Reader) ([]Transaction, error)
	ConvertToCSV(inputFile, outputFile string) error
	WriteToCSV(transactions []Transaction, csvFile string) error
	SetLogger(logger *logrus.Logger)
	ValidateFormat(file string) (bool, error)
	BatchConvert(inputDir, outputDir string) (int, error)
}
