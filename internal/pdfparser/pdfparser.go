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

var log = logging.GetLogger()

// Parse extracts and parses transaction data from a PDF file provided as an io.Reader.
// This function uses the default RealPDFExtractor for backward compatibility.
func Parse(r io.Reader) ([]models.Transaction, error) {
	return ParseWithExtractor(r, NewRealPDFExtractor())
}

// ParseWithExtractor extracts and parses transaction data from a PDF file using the provided extractor.
func ParseWithExtractor(r io.Reader, extractor PDFExtractor) ([]models.Transaction, error) {

	// Read the content of the reader into a temporary file for PDF processing
	tempFile, err := os.CreateTemp("", "*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary PDF file: %w", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			log.WithError(err).Warn("Failed to remove temporary file",
				logging.Field{Key: "file", Value: tempFile.Name()})
		}
	}()
	defer func() {
		if err := tempFile.Close(); err != nil {
			log.WithError(err).Warn("Failed to close temporary file",
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

	log.Info("Parsing PDF file",
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
		log.WithError(err).Warn("Failed to write debug file")
	} else {
		log.Debug("Wrote raw PDF text to debug file",
			logging.Field{Key: "file", Value: debugFile})
	}

	// Preprocess the text to clean it up and identify transaction blocks
	processedText := preProcessText(text)

	// Split text into lines for processing
	lines := strings.Split(processedText, "\n")

	// Parse the lines to extract transactions
	transactions, err := parseTransactions(lines)
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
	// Open the input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warn("Failed to close file",
				logging.Field{Key: "error", Value: err})
		}
	}()

	// Parse the file using the new Parse method
	transactions, err := Parse(file)
	if err != nil {
		return err
	}

	// Handle empty transactions list
	if len(transactions) == 0 {
		log.Info("No transactions found, created empty CSV file with headers",
			logging.Field{Key: "file", Value: outputFile},
			logging.Field{Key: "delimiter", Value: string(common.Delimiter)})

		emptyTransactions := []models.Transaction{}
		return common.WriteTransactionsToCSV(emptyTransactions, outputFile)
	}

	// Write the transactions to the CSV file
	log.Info("Writing transactions to CSV file",
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

	log.Info("Successfully wrote transactions to CSV file",
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "file", Value: outputFile})

	return nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file in a simplified format
// that is specifically used by the PDF parser tests.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	log.Info("Writing transactions to CSV file using common implementation",
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
func validateFormat(pdfFile string) (bool, error) {
	log.Info("Validating PDF format",
		logging.Field{Key: "file", Value: pdfFile})

	// Try to extract text as a validation check using the real extractor
	extractor := NewRealPDFExtractor()
	_, err := extractor.ExtractText(pdfFile)
	if err != nil {
		log.WithError(err).Error("PDF validation failed")
		return false, nil
	}

	return true, nil
}
