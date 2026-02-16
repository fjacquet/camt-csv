package pdfparser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fjacquet/camt-csv/internal/batch"
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
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithExtractorAndCategorizer(ctx, r, a.extractor, a.GetLogger(), a.GetCategorizer())
}

// ConvertToCSV implements models.Parser.ConvertToCSV
func (a *Adapter) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
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

	transactions, err := a.Parse(ctx, file)
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

// BatchConvert processes all PDF files in inputDir and writes converted CSV files to outputDir.
// Returns the number of successfully converted files.
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
	processor := batch.NewBatchProcessor(a, a.GetLogger(), nil)

	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
	if err != nil {
		// Config/permission error (not file-level errors)
		return 0, err
	}

	// Log summary
	a.GetLogger().Info("Batch processing completed",
		logging.Field{Key: "total", Value: manifest.TotalFiles},
		logging.Field{Key: "succeeded", Value: manifest.SuccessCount},
		logging.Field{Key: "failed", Value: manifest.FailureCount})

	// Write manifest
	manifestPath := filepath.Join(outputDir, ".manifest.json")
	if err := manifest.WriteManifest(manifestPath); err != nil {
		a.GetLogger().WithError(err).Warn("Failed to write manifest")
	}

	return manifest.SuccessCount, nil
}
