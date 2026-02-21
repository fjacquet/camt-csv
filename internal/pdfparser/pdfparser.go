package pdfparser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"
)

// ParseWithExtractorAndCategorizer extracts and parses transaction data from a PDF file using the provided extractor and categorizer.
func ParseWithExtractorAndCategorizer(ctx context.Context, r io.Reader, extractor PDFExtractor, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	// Create single temp directory for all PDF processing files
	tempDir, err := os.MkdirTemp("", "pdfparse-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			logger.WithError(err).Warn("Failed to remove temporary directory",
				logging.Field{Key: "dir", Value: tempDir})
		}
	}()

	// Create PDF file within temp directory
	pdfPath := filepath.Join(tempDir, "input.pdf")
	pdfFile, err := os.OpenFile(pdfPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary PDF file: %w", err)
	}

	// Read the content of the reader into the PDF file
	_, err = io.Copy(pdfFile, r)
	if err != nil {
		_ = pdfFile.Close()
		return nil, fmt.Errorf("failed to write to temporary PDF file: %w", err)
	}

	// Close the file before external extraction command accesses it
	if err := pdfFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temporary PDF file: %w", err)
	}

	logger.Info("Parsing PDF file",
		logging.Field{Key: "file", Value: pdfPath})

	// Extract text from PDF (validates format and extracts in one call)
	text, err := extractor.ExtractText(pdfPath)
	if err != nil {
		return nil, &parsererror.ParseError{
			Parser: "PDF",
			Field:  "text extraction",
			Value:  pdfPath,
			Err:    err,
		}
	}

	logger.Debug("Extracted PDF text for processing",
		logging.Field{Key: "text_length", Value: len(text)})

	// Preprocess the text to clean it up and identify transaction blocks
	processedText := preProcessText(text)

	// Split text into lines for processing
	lines := strings.Split(processedText, "\n")

	// Parse the lines to extract transactions
	transactions, err := parseTransactionsWithCategorizer(lines, logger, categorizer)
	if err != nil {
		return nil, &parsererror.ParseError{
			Parser: "PDF",
			Field:  "transaction parsing",
			Value:  "extracted text",
			Err:    err,
		}
	}

	return transactions, nil
}

// validateFormat checks if a file is a valid PDF.
// It verifies that the file exists and has the correct format headers.
//
// Parameters:
//   - pdfFile: Path to the PDF file to validate
//
// Returns:
//   - bool: True if the file is a valid PDF, False otherwise
//   - error: Any error encountered during validation
