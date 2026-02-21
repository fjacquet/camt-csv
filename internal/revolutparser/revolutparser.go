// Package revolutparser provides functionality to parse Revolut CSV files and convert them to the standard format.
// It handles the extraction of transaction data from Revolut CSV export files.
package revolutparser

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

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

// ParseWithCategorizer parses a Revolut CSV file and categorizes transactions using the provided categorizer.
// This is the preferred entry point when categorization is needed.
func ParseWithCategorizer(r io.Reader, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
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

		// Only include completed transactions (log others for visibility)
		if revolutRows[i].State != models.StatusCompleted {
			logger.Info("Skipping non-completed transaction",
				logging.Field{Key: "state", Value: revolutRows[i].State},
				logging.Field{Key: "date", Value: revolutRows[i].CompletedDate},
				logging.Field{Key: "description", Value: revolutRows[i].Description},
				logging.Field{Key: "amount", Value: revolutRows[i].Amount},
				logging.Field{Key: "currency", Value: revolutRows[i].Currency})
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

		// Categorize the transaction using the injected categorizer
		if categorizer != nil {
			// Determine if this is a debit (payment out) or credit (payment in)
			isDebtor := tx.CreditDebit == models.TransactionTypeDebit
			catAmount := tx.Amount.String()
			catDate := ""
			if !tx.Date.IsZero() {
				catDate = tx.Date.Format("02.01.2006")
			}

			category, catErr := categorizer.Categorize(context.Background(), tx.Description, isDebtor, catAmount, catDate, "")
			if catErr != nil {
				logger.WithError(catErr).WithFields(
					logging.Field{Key: "party", Value: tx.Description},
				).Warn("Failed to categorize transaction")
				tx.Category = models.CategoryUncategorized
			} else {
				tx.Category = category.Name
			}
		} else {
			tx.Category = models.CategoryUncategorized
		}

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
		WithFees(feeDecimal).
		WithProduct(row.Product)

	// Handle exchange transactions - preserve both currencies
	if row.Type == "EXCHANGE" {
		// For EXCHANGE type, the Amount is in the account's currency (Currency field)
		// The original currency being exchanged is implied by the sign and description
		// Store exchange metadata for reference
		if !amountDecimal.IsZero() {
			// Exchange rate calculation: if we have FX data in future, use it
			// For now, preserve the currencies and amounts we have
			builder = builder.WithOriginalAmount(amountDecimal, row.Currency)

			logger.Debug("Processing EXCHANGE transaction",
				logging.Field{Key: "amount", Value: amountDecimal.String()},
				logging.Field{Key: "currency", Value: row.Currency},
				logging.Field{Key: "description", Value: row.Description})
		}
	}

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

