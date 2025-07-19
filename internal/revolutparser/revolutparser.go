// Package revolutparser provides functionality to parse Revolut CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Revolut CSV export files.
package revolutparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// SetLogger allows setting a configured logger for this package.
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
	}
}

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

// ParseFile parses a Revolut CSV file and returns a slice of Transaction objects.
// This is the main entry point for parsing Revolut CSV files.
func ParseFile(filePath string) ([]models.Transaction, error) {
	log.WithField("file", filePath).Info("Parsing Revolut CSV file")

	// Check if the file format is valid
	valid, err := ValidateFormat(filePath)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid Revolut CSV format")
	}

	// Use common.ReadCSVFile to read the CSV
	revolutRows, err := common.ReadCSVFile[RevolutCSVRow](filePath)
	if err != nil {
		log.WithError(err).Error("Failed to read Revolut CSV file")
		return nil, fmt.Errorf("error reading Revolut CSV: %w", err)
	}

	// Convert RevolutCSVRow objects to Transaction objects
	var transactions []models.Transaction
	for i := range revolutRows {
		// Skip empty rows
		if revolutRows[i].CompletedDate == "" || revolutRows[i].Description == "" {
			continue
		}

		// Only include completed transactions
		if revolutRows[i].State != "COMPLETED" {
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
		tx, err := convertRevolutRowToTransaction(revolutRows[i])
		if err != nil {
			log.WithError(err).WithField("row", revolutRows[i]).Warn("Failed to convert row to transaction")
			continue
		}

		// Categorize the transaction
		category, err := categorizer.CategorizeTransaction(categorizer.Transaction{
			PartyName:   tx.Description,
			Description: tx.Description,
			Amount:      tx.Amount.StringFixed(2),
			IsDebtor:    tx.CreditDebit == "DBIT",
			Date:        tx.Date,
		})
		if err != nil {
			log.WithError(err).Warn("Failed to categorize transaction")
		} else {
			tx.Category = category.Name
		}

		transactions = append(transactions, tx)
	}

	// Post-process transactions to apply specific description transformations
	processedTransactions := postProcessTransactions(transactions)

	log.WithField("count", len(processedTransactions)).Info("Successfully parsed Revolut CSV file")
	return processedTransactions, nil
}

// postProcessTransactions applies additional processing to transactions after they've been created
// specifically for handling special cases like transfer descriptions
func postProcessTransactions(transactions []models.Transaction) []models.Transaction {
	for i := range transactions {
		// Handle descriptions for transfers to CHF Vacances
		if transactions[i].Type == "TRANSFER" && transactions[i].Description == "To CHF Vacances" {
			if transactions[i].CreditDebit == "DBIT" {
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

// convertRevolutRowToTransaction converts a RevolutCSVRow to a Transaction
func convertRevolutRowToTransaction(row RevolutCSVRow) (models.Transaction, error) {
	var creditDebit string
	var isDebit bool

	// Now that descriptions are modified at the row level in ParseFile,
	// we only need to determine credit/debit status here
	if strings.HasPrefix(row.Amount, "-") {
		isDebit = true
		creditDebit = "DBIT"
	} else {
		isDebit = false
		creditDebit = "CRDT"
	}

	// Format dates to DD.MM.YYYY format for consistency
	completedDate := models.FormatDate(row.CompletedDate)
	startedDate := models.FormatDate(row.StartedDate)

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
			log.WithError(err).Warn("Failed to parse fee value, defaulting to zero")
		}
	}

	// Determine debit and credit amounts
	debitAmount := decimal.Zero
	creditAmount := decimal.Zero
	if creditDebit == "DBIT" {
		debitAmount = amountDecimal
	} else {
		creditAmount = amountDecimal
	}

	// Set payee/payer based on description
	payee := row.Description
	payer := row.Description

	transaction := models.Transaction{
		BookkeepingNumber: "", // Revolut doesn't provide this
		Status:            row.State,
		Date:              completedDate,
		ValueDate:         startedDate,
		Name:              row.Description,
		PartyName:         row.Description,
		Description:       row.Description,
		Amount:            amountDecimal,
		CreditDebit:       creditDebit,
		DebitFlag:         isDebit, // Set the DebitFlag (maps to IsDebit in CSV)
		Debit:             debitAmount,
		Credit:            creditAmount,
		Currency:          row.Currency,
		AmountExclTax:     amountDecimal, // Default to full amount
		AmountTax:         decimal.Zero,  // Revolut doesn't provide tax details
		TaxRate:           decimal.Zero,
		Recipient:         payee,
		Investment:        row.Type, // Use transaction type as investment type
		Number:            "",       // Not provided by Revolut
		Category:          "",       // Will be categorized later
		Type:              row.Type,
		Fund:              "", // Not provided by Revolut
		NumberOfShares:    0,  // Revolut transactions don't have shares
		Fees:              feeDecimal,
		IBAN:              "", // Not provided by Revolut
		EntryReference:    "",
		Reference:         "",
		AccountServicer:   "",
		BankTxCode:        "",
		OriginalCurrency:  "", // Not handling foreign currencies for now
		OriginalAmount:    decimal.Zero,
		ExchangeRate:      decimal.Zero,

		// Keep these for backward compatibility
		Payee: payee,
		Payer: payer,
	}

	return transaction, nil
}

// WriteToCSV writes a slice of Transaction objects to a CSV file in a simplified format
// that is specifically used by the Revolut parser tests.
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	if transactions == nil {
		return fmt.Errorf("cannot write nil transactions to CSV")
	}

	log.WithFields(logrus.Fields{
		"file":  csvFile,
		"count": len(transactions),
	}).Info("Writing transactions to CSV file")

	// Create the directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.WithError(err).Error("Failed to create directory")
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create the file
	file, err := os.Create(csvFile)
	if err != nil {
		log.WithError(err).Error("Failed to create CSV file")
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer file.Close()

	// Configure CSV writer with custom delimiter
	csvWriter := csv.NewWriter(file)
	csvWriter.Comma = common.Delimiter

	// Write the header
	header := []string{"Date", "Description", "Amount", "Currency"}
	if err := csvWriter.Write(header); err != nil {
		log.WithError(err).Error("Failed to write CSV header")
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write each transaction
	for _, tx := range transactions {
		// Ensure date is in DD.MM.YYYY format
		date := models.FormatDate(tx.Date)

		// Format the amount with 2 decimal places
		amount := tx.Amount.StringFixed(2)

		row := []string{
			date,
			tx.Description,
			amount,
			tx.Currency,
		}

		if err := csvWriter.Write(row); err != nil {
			log.WithError(err).Error("Failed to write CSV row")
			return fmt.Errorf("error writing CSV row: %w", err)
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		log.WithError(err).Error("Error flushing CSV writer")
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	log.WithFields(logrus.Fields{
		"file":  csvFile,
		"count": len(transactions),
	}).Info("Successfully wrote transactions to CSV file")

	return nil
}

// ConvertToCSV converts a Revolut CSV file to the standard CSV format.
// This is a convenience function that combines ParseFile and WriteToCSV.
func ConvertToCSV(inputFile, outputFile string) error {
	log.WithFields(logrus.Fields{
		"input":  inputFile,
		"output": outputFile,
	}).Info("Converting file to CSV")

	// Validate the file format
	valid, err := ValidateFormat(inputFile)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("input file is not a valid Revolut CSV")
	}

	// Parse the file
	transactions, err := ParseFile(inputFile)
	if err != nil {
		return err
	}

	// Write to CSV
	if err := WriteToCSV(transactions, outputFile); err != nil {
		return err
	}

	log.WithFields(logrus.Fields{
		"count":  len(transactions),
		"input":  inputFile,
		"output": outputFile,
	}).Info("Successfully converted file to CSV")

	return nil
}

// ValidateFormat checks if the file is a valid Revolut CSV file.
func ValidateFormat(filePath string) (bool, error) {
	log.WithField("file", filePath).Info("Validating Revolut CSV format")

	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open file for validation")
		return false, fmt.Errorf("error opening file for validation: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Failed to read CSV header")
		return false, fmt.Errorf("error reading CSV header: %w", err)
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
			log.WithField("column", requiredCol).Info("Required column missing from Revolut CSV")
			return false, nil
		}
	}

	// Check at least one data row is present
	_, err = reader.Read()
	if err == io.EOF {
		log.Info("Revolut CSV file is empty (header only)")
		return false, nil
	} else if err != nil {
		log.WithError(err).Error("Error reading CSV record")
		return false, fmt.Errorf("error reading CSV record: %w", err)
	}

	log.Info("File is a valid Revolut CSV")
	return true, nil
}

// BatchConvert converts all CSV files in a directory to the standard CSV format.
// It processes all files with a .csv extension in the specified directory.
func BatchConvert(inputDir, outputDir string) (int, error) {
	log.WithFields(logrus.Fields{
		"inputDir":  inputDir,
		"outputDir": outputDir,
	}).Info("Batch converting Revolut CSV files")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.WithError(err).Error("Failed to create output directory")
		return 0, fmt.Errorf("error creating output directory: %w", err)
	}

	// Get all CSV files in input directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		log.WithError(err).Error("Failed to read input directory")
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	// Process each CSV file
	var processed int
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
			continue
		}

		inputPath := fmt.Sprintf("%s/%s", inputDir, file.Name())

		// Validate if it's a Revolut CSV file
		isValid, err := ValidateFormat(inputPath)
		if err != nil {
			log.WithError(err).WithField("file", inputPath).Warning("Error validating file, skipping")
			continue
		}

		if !isValid {
			log.WithField("file", inputPath).Info("Not a valid Revolut CSV file, skipping")
			continue
		}

		// Create output file path
		baseName := file.Name()
		baseNameWithoutExt := strings.TrimSuffix(baseName, ".csv")
		outputPath := fmt.Sprintf("%s/%s-standardized.csv", outputDir, baseNameWithoutExt)

		// Convert the file
		if err := ConvertToCSV(inputPath, outputPath); err != nil {
			log.WithFields(logrus.Fields{
				"file":  inputPath,
				"error": err,
			}).Warning("Failed to convert file, skipping")
			continue
		}
		processed++
	}

	log.WithField("count", processed).Info("Batch conversion completed")
	return processed, nil
}
