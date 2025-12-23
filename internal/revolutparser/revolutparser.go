// Package revolutparser provides functionality to parse Revolut CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Revolut CSV export files.
package revolutparser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"

	"github.com/gocarina/gocsv"
	"github.com/shopspring/decimal"
)

// RevolutCSVRow represents a single row in a Revolut CSV file
// It uses struct tags for gocsv unmarshaling
type RevolutCSVRow struct {
	Type          string `csv:"Type"`
	Product       string `csv:"Product"`
	StartedDate   string `csv:"Started Date"`
	CompletedDate string `csv:"Completed Date"`
	Description   string `csv:"Description"`
	Amount        string `csv:"Amount"`
	Fee           string `csv:"Fee"`
	Currency      string `csv:"Currency"`
	State         string `csv:"State"`
	Balance       string `csv:"Balance"`
}

// Parse parses a Revolut CSV file from an io.Reader and returns a slice of Transaction objects.
// This is the main entry point for parsing Revolut CSV files.
func Parse(r io.Reader, logger logging.Logger) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Parsing Revolut CSV from reader")

	// Buffer the reader content so we can validate and parse from the same data
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	// Check if the file format is valid
	valid, err := validateFormat(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	if !valid {
		return nil, &parsererror.InvalidFormatError{
			FilePath:       "(from reader)",
			ExpectedFormat: "Revolut CSV",
			Msg:            "invalid Revolut CSV format",
		}
	}

	// Parse the CSV data
	var revolutRows []*RevolutCSVRow
	if err := gocsv.Unmarshal(bytes.NewReader(data), &revolutRows); err != nil {
		return nil, &parsererror.ParseError{
			Parser: "Revolut",
			Field:  "CSV",
			Value:  "CSV data",
			Err:    err,
		}
	}

	// Convert RevolutCSVRow objects to Transaction objects
	// Pre-allocate slice with capacity based on input size (most rows will be valid)
	transactions := make([]models.Transaction, 0, len(revolutRows))
	for i := range revolutRows {
		// Skip empty rows
		if revolutRows[i].CompletedDate == "" || revolutRows[i].Description == "" {
			continue
		}

		// Only include completed transactions
		if revolutRows[i].State != models.StatusCompleted {
			continue
		}

		// Process description for special transfers
		if revolutRows[i].Type == "TRANSFER" {
			if strings.Contains(revolutRows[i].Description, "To CHF Vacances") {
				if revolutRows[i].Product == "CURRENT" {
					revolutRows[i].Description = "Transfert to CHF Vacances"
				} else if revolutRows[i].Product == "SAVINGS" {
					revolutRows[i].Description = "Transferred To CHF Vacances"
				}
			}
			// Add other transfer type handling here if needed
		}

		// Convert Revolut row to Transaction
		tx, err := convertRevolutRowToTransaction(*revolutRows[i], logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert row to transaction",
				logging.Field{Key: "row", Value: revolutRows[i]})
			continue
		}

		// Note: Categorization is now handled by the categorizer component
		// through dependency injection, not directly in the parser
		tx.Category = models.CategoryUncategorized

		transactions = append(transactions, tx)
	}

	// Post-process transactions to apply specific description transformations
	processedTransactions := postProcessTransactions(transactions)

	logger.Info("Successfully parsed Revolut CSV from reader",
		logging.Field{Key: "count", Value: len(processedTransactions)})
	return processedTransactions, nil
}

// postProcessTransactions applies additional processing to transactions after they've been created
// specifically for handling special cases like transfer descriptions
func postProcessTransactions(transactions []models.Transaction) []models.Transaction {
	for i := range transactions {
		// Handle descriptions for transfers to CHF Vacances
		if transactions[i].Type == "TRANSFER" && transactions[i].Description == "To CHF Vacances" {
			if transactions[i].CreditDebit == models.TransactionTypeDebit {
				transactions[i].Description = "Transfert to CHF Vacances"
				transactions[i].Name = "Transfert to CHF Vacances"
				transactions[i].PartyName = "Transfert to CHF Vacances"
				transactions[i].Recipient = "Transfert to CHF Vacances"
			} else {
				transactions[i].Description = "Transferred To CHF Vacances"
				transactions[i].Name = "Transferred To CHF Vacances"
				transactions[i].PartyName = "Transferred To CHF Vacances"
				transactions[i].Recipient = "Transferred To CHF Vacances"
			}
		}
	}
	return transactions
}

// convertRevolutRowToTransaction converts a RevolutCSVRow to a Transaction using TransactionBuilder
func convertRevolutRowToTransaction(row RevolutCSVRow, logger logging.Logger) (models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	// Determine if this is a debit or credit transaction
	isDebit := strings.HasPrefix(row.Amount, "-")

	// Parse amount to decimal (remove negative sign for internal calculations)
	amount := strings.TrimPrefix(row.Amount, "-")
	amountDecimal, err := decimal.NewFromString(amount)
	if err != nil {
		return models.Transaction{}, fmt.Errorf("error parsing amount to decimal: %w", err)
	}

	// Parse fee if present
	feeDecimal := decimal.Zero
	if row.Fee != "" {
		feeDecimal, err = decimal.NewFromString(row.Fee)
		if err != nil {
			logger.WithError(err).Warn("Failed to parse fee value, defaulting to zero")
		}
	}

	// Use TransactionBuilder for consistent transaction construction
	builder := models.NewTransactionBuilder().
		WithStatus(row.State).
		WithDateFromDatetime(row.CompletedDate).
		WithValueDateFromDatetime(row.StartedDate).
		WithDescription(row.Description).
		WithAmount(amountDecimal, row.Currency).
		WithPartyName(row.Description).
		WithType(row.Type).
		WithInvestment(row.Type).
		WithFees(feeDecimal)

	// Set transaction direction
	if isDebit {
		builder = builder.AsDebit().WithPayee(row.Description, "")
	} else {
		builder = builder.AsCredit().WithPayer(row.Description, "")
	}

	// Build the transaction
	transaction, err := builder.Build()
	if err != nil {
		return models.Transaction{}, fmt.Errorf("error building transaction: %w", err)
	}

	return transaction, nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file in a simplified format
// that is specifically used by the Revolut parser tests.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return WriteToCSVWithLogger(transactions, csvFile, nil)
}

func WriteToCSVWithLogger(transactions []models.Transaction, csvFile string, logger logging.Logger) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	if transactions == nil {
		return fmt.Errorf("cannot write nil transactions to CSV")
	}

	logger.Info("Writing transactions to CSV file",
		logging.Field{Key: "file", Value: csvFile},
		logging.Field{Key: "count", Value: len(transactions)})

	// Create the directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create the file
	file, err := os.Create(csvFile) // #nosec G304 -- CLI tool requires user-provided output paths
	if err != nil {
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close file")
		}
	}()

	// Configure CSV writer with custom delimiter
	csvWriter := csv.NewWriter(file)
	csvWriter.Comma = common.Delimiter

	// Write the header
	header := []string{"Date", "Description", "Amount", "Currency"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write each transaction
	for _, tx := range transactions {
		// Format date as DD.MM.YYYY
		var date string
		if !tx.Date.IsZero() {
			date = tx.Date.Format(dateutils.DateLayoutEuropean)
		}

		// Format the amount with 2 decimal places
		amount := tx.Amount.StringFixed(2)

		row := []string{
			date,
			tx.Description,
			amount,
			tx.Currency,
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("error writing CSV row: %w", err)
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	logger.Info("Successfully wrote transactions to CSV file",
		logging.Field{Key: "file", Value: csvFile},
		logging.Field{Key: "count", Value: len(transactions)})

	return nil
}

// ConvertToCSV converts a Revolut CSV file to the standard CSV format.
// This is a convenience function that combines Parse and WriteToCSV.
func ConvertToCSV(inputFile, outputFile string) error {
	return ConvertToCSVWithLogger(inputFile, outputFile, nil)
}

func ConvertToCSVWithLogger(inputFile, outputFile string, logger logging.Logger) error {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Converting file to CSV",
		logging.Field{Key: "input", Value: inputFile},
		logging.Field{Key: "output", Value: outputFile})

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

	// Parse the file
	transactions, err := Parse(file, logger)
	if err != nil {
		return err
	}

	// Write to CSV
	if err := WriteToCSV(transactions, outputFile); err != nil {
		return err
	}

	logger.Info("Successfully converted file to CSV",
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "input", Value: inputFile},
		logging.Field{Key: "output", Value: outputFile})

	return nil
}

// validateFormat checks if the file is a valid Revolut CSV file.
func validateFormat(r io.Reader) (bool, error) {
	return validateFormatWithLogger(r, nil)
}

func validateFormatWithLogger(r io.Reader, logger logging.Logger) (bool, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Validating Revolut CSV format from reader")

	reader := csv.NewReader(r)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return false, &parsererror.ValidationError{
			FilePath: "(from reader)",
			Reason:   fmt.Sprintf("failed to read CSV header: %v", err),
		}
	}

	// Required columns for a valid Revolut CSV
	requiredColumns := []string{
		"Type", "Product", "Started Date", "Description",
		"Amount", "Currency", "State",
	}

	// Map header columns to check if all required ones exist
	headerMap := make(map[string]bool)
	for _, col := range header {
		headerMap[col] = true
	}

	// Check if all required columns exist
	for _, requiredCol := range requiredColumns {
		if !headerMap[requiredCol] {
			logger.Info("Required column missing from Revolut CSV",
				logging.Field{Key: "column", Value: requiredCol})
			return false, nil
		}
	}

	// Check at least one data row is present
	_, err = reader.Read()
	if err == io.EOF {
		logger.Info("Revolut CSV file is empty (header only)")
		return false, nil
	} else if err != nil {
		return false, &parsererror.ValidationError{
			FilePath: "(from reader)",
			Reason:   fmt.Sprintf("error reading CSV record: %v", err),
		}
	}

	logger.Info("Reader contains valid Revolut CSV")
	return true, nil
}

// BatchConvert converts all CSV files in a directory to the standard CSV format.
// It processes all files with a .csv extension in the specified directory.
func BatchConvert(inputDir, outputDir string) (int, error) {
	return BatchConvertWithLogger(inputDir, outputDir, nil)
}

func BatchConvertWithLogger(inputDir, outputDir string, logger logging.Logger) (int, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Batch converting Revolut CSV files",
		logging.Field{Key: "inputDir", Value: inputDir},
		logging.Field{Key: "outputDir", Value: outputDir})

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return 0, fmt.Errorf("error creating output directory: %w", err)
	}

	// Get all CSV files in input directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	// Process each CSV file
	var processed int
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
			continue
		}

		inputPath := fmt.Sprintf("%s/%s", inputDir, file.Name())

		// Open the input file for validation and parsing
		inputFile, err := os.Open(inputPath) // #nosec G304 -- CLI tool requires user-provided file paths
		if err != nil {
			logger.WithError(err).Warn("Error opening file, skipping",
				logging.Field{Key: "file", Value: inputPath})
			continue
		}

		// Validate if it's a Revolut CSV file
		isValid, err := validateFormat(inputFile)
		if err != nil {
			logger.WithError(err).Warn("Error validating file, skipping",
				logging.Field{Key: "file", Value: inputPath})
			if err := inputFile.Close(); err != nil {
				logger.WithError(err).Warn("Failed to close file after validation attempt",
					logging.Field{Key: "file", Value: inputPath})
			}
			continue
		}

		if !isValid {
			logger.Info("Not a valid Revolut CSV file, skipping",
				logging.Field{Key: "file", Value: inputPath},
				logging.Field{Key: "reason", Value: "validation_failed"})
			if err := inputFile.Close(); err != nil {
				logger.WithError(err).Warn("Failed to close file after validation attempt",
					logging.Field{Key: "file", Value: inputPath})
			}
			continue
		}

		// Rewind the file to the beginning for parsing after validation
		_, err = inputFile.Seek(0, io.SeekStart)
		if err != nil {
			logger.WithError(err).Warn("Error rewinding file, skipping",
				logging.Field{Key: "file", Value: inputPath})
			if err := inputFile.Close(); err != nil {
				logger.WithError(err).Warn("Failed to close file after rewinding",
					logging.Field{Key: "file", Value: inputPath})
			}
			continue
		}

		// Parse the file
		transactions, err := Parse(inputFile, logger)
		if err := inputFile.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close file after parsing",
				logging.Field{Key: "file", Value: inputPath})
		}
		if err != nil {
			logger.WithError(err).Warn("Failed to parse file, skipping",
				logging.Field{Key: "file", Value: inputPath})
			continue
		}

		// Create output file path
		baseName := file.Name()
		baseNameWithoutExt := strings.TrimSuffix(baseName, ".csv")
		outputPath := fmt.Sprintf("%s/%s-standardized.csv", outputDir, baseNameWithoutExt)

		// Write to CSV
		if err := WriteToCSVWithLogger(transactions, outputPath, logger); err != nil {
			logger.WithError(err).Warn("Failed to write to CSV, skipping",
				logging.Field{Key: "file", Value: inputPath})
			continue
		}
		processed++
	}

	logger.Info("Batch conversion completed",
		logging.Field{Key: "count", Value: processed})
	return processed, nil
}
