package pdfparser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"
)

// Note: Removed global logger in favor of dependency injection

// Parse extracts and parses transaction data from a PDF file provided as an io.Reader.
// This function uses the default RealPDFExtractor for backward compatibility.
func Parse(r io.Reader, logger logging.Logger) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	return ParseWithExtractor(r, NewRealPDFExtractor(), logger)
}

// ParseWithExtractor extracts and parses transaction data from a PDF file using the provided extractor.
func ParseWithExtractor(r io.Reader, extractor PDFExtractor, logger logging.Logger) ([]models.Transaction, error) {
	return ParseWithExtractorAndCategorizer(r, extractor, logger, nil)
}

// ParseWithExtractorAndCategorizer extracts and parses transaction data from a PDF file using the provided extractor and categorizer.
func ParseWithExtractorAndCategorizer(r io.Reader, extractor PDFExtractor, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	// Read the content of the reader into a temporary file for PDF processing
	tempFile, err := os.CreateTemp("", "*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary PDF file: %w", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			logger.WithError(err).Warn("Failed to remove temporary file",
				logging.Field{Key: "file", Value: tempFile.Name()})
		}
	}()
	defer func() {
		if err := tempFile.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close temporary file",
				logging.Field{Key: "file", Value: tempFile.Name()})
		}
	}()

	_, err = io.Copy(tempFile, r)
	if err != nil {
		return nil, fmt.Errorf("failed to write to temporary PDF file: %w", err)
	}

	// Close the file before attempting to extract text
	// This close is already handled by the defer, but if we need to ensure it's closed
	// before an external command accesses it, we might close it here and then
	// re-open if needed, or ensure the external command doesn't hold the file open.
	// For now, the defer is sufficient for cleanup.
	// if err := tempFile.Close(); err != nil {
	// 	return nil, fmt.Errorf("failed to close temporary PDF file: %w", err)
	// }

	// Validate the file format using the provided extractor
	_, err = extractor.ExtractText(tempFile.Name())
	if err != nil {
		return nil, &parsererror.InvalidFormatError{
			FilePath:       tempFile.Name(),
			ExpectedFormat: "PDF",
			Msg:            "file is not a valid PDF",
		}
	}

	logger.Info("Parsing PDF file",
		logging.Field{Key: "file", Value: tempFile.Name()})

	// Extract text from PDF using the provided extractor
	text, err := extractor.ExtractText(tempFile.Name())
	if err != nil {
		return nil, &parsererror.ParseError{
			Parser: "PDF",
			Field:  "text extraction",
			Value:  tempFile.Name(),
			Err:    err,
		}
	}

	// Write raw PDF text to debug file if in debug mode
	debugFile := "debug_pdf_extract.txt"
	err = os.WriteFile(debugFile, []byte(text), 0600)
	if err != nil {
		logger.WithError(err).Warn("Failed to write debug file")
	} else {
		logger.Debug("Wrote raw PDF text to debug file",
			logging.Field{Key: "file", Value: debugFile})
	}

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

// ConvertToCSV converts a PDF bank statement to the standard CSV format.
// This is a convenience function that combines Parse and WriteToCSV.
func ConvertToCSV(inputFile, outputFile string) error {
	return ConvertToCSVWithLogger(inputFile, outputFile, nil)
}

func ConvertToCSVWithLogger(inputFile, outputFile string, logger logging.Logger) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	// Open the input file
	file, err := os.Open(inputFile) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("Failed to close file",
				logging.Field{Key: "error", Value: err})
		}
	}()

	// Parse the file using the new Parse method
	transactions, err := Parse(file, logger)
	if err != nil {
		return err
	}

	// Handle empty transactions list
	if len(transactions) == 0 {
		logger.Info("No transactions found, created empty CSV file with headers",
			logging.Field{Key: "file", Value: outputFile},
			logging.Field{Key: "delimiter", Value: string(common.Delimiter)})

		emptyTransactions := []models.Transaction{}
		return common.WriteTransactionsToCSV(emptyTransactions, outputFile)
	}

	// Write the transactions to the CSV file
	logger.Info("Writing transactions to CSV file",
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "file", Value: outputFile})

	// Create the directory if it doesn't exist
	dir := filepath.Dir(outputFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := common.ExportTransactionsToCSV(transactions, outputFile); err != nil {
		return err
	}

	logger.Info("Successfully wrote transactions to CSV file",
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "file", Value: outputFile})

	return nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file in a simplified format
// that is specifically used by the PDF parser tests.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return WriteToCSVWithLogger(transactions, csvFile, nil)
}

func WriteToCSVWithLogger(transactions []models.Transaction, csvFile string, logger logging.Logger) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Writing transactions to CSV file using common implementation",
		logging.Field{Key: "file", Value: csvFile},
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "delimiter", Value: string(common.Delimiter)})

	// Use the common implementation to ensure consistent delimiter usage
	return common.WriteTransactionsToCSV(transactions, csvFile)
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
