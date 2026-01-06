package pdfparser

import (
	"fmt"
	"io"
	"os"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the models.Parser interface for PDF bank statements.
type Adapter struct {
	parser.BaseParser
	extractor PDFExtractor
}

// NewAdapter creates a new adapter for the pdfparser with dependency injection.
func NewAdapter(logger logging.Logger, extractor PDFExtractor) *Adapter {
	if extractor == nil {
		extractor = NewRealPDFExtractor()
	}
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
		extractor:  extractor,
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(r io.Reader) ([]models.Transaction, error) {
	return ParseWithExtractorAndCategorizer(r, a.extractor, a.GetLogger(), a.GetCategorizer())
}

// ConvertToCSV implements models.Parser.ConvertToCSV
func (a *Adapter) ConvertToCSV(inputFile, outputFile string) error {
	file, err := os.Open(inputFile) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			a.GetLogger().WithError(err).Warn("Failed to close input file",
				logging.Field{Key: "file", Value: inputFile})
		}
	}()

	transactions, err := a.Parse(file)
	if err != nil {
		return err
	}

	return a.WriteToCSV(transactions, outputFile)
}

// ValidateFormat checks if a file is a valid PDF file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	a.GetLogger().Info("Validating PDF format",
		logging.Field{Key: "file", Value: file})

	// Try to extract text as a validation check using the injected extractor
	_, err := a.extractor.ExtractText(file)
	if err != nil {
		a.GetLogger().WithError(err).Error("PDF validation failed")
		return false, nil
	}

	return true, nil
}

// BatchConvert is not implemented for this parser.
func (a *Adapter) BatchConvert(inputDir, outputDir string) (int, error) {
	return 0, fmt.Errorf("not implemented")
}
