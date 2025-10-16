package pdfparser

import (
	"fmt"
	"io"
	"os"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

// Adapter implements the models.Parser interface for PDF bank statements.
type Adapter struct {
	logger *logrus.Logger
}

// NewAdapter creates a new adapter for the pdfparser.
func NewAdapter() models.Parser {
	return &Adapter{
		logger: logrus.New(),
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
			a.logger.WithError(err).Warnf("Failed to close input file %s", inputFile)
		}
	}()

	transactions, err := a.Parse(file)
	if err != nil {
		return err
	}

	return a.WriteToCSV(transactions, outputFile)
}

// WriteToCSV implements models.Parser.WriteToCSV
func (a *Adapter) WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return common.WriteTransactionsToCSV(transactions, csvFile)
}

// SetLogger implements models.Parser.SetLogger
func (a *Adapter) SetLogger(logger *logrus.Logger) {
	a.logger = logger
}

// ValidateFormat checks if a file is a valid PDF file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	return validateFormat(file)
}

// BatchConvert is not implemented for this parser.
func (a *Adapter) BatchConvert(inputDir, outputDir string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}
