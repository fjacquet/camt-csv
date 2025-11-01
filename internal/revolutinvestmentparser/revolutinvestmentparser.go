package revolutinvestmentparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"

	"github.com/shopspring/decimal"
)

// RevolutInvestmentCSVRow represents a single row in a Revolut investment CSV file
type RevolutInvestmentCSVRow struct {
	Date          string `csv:"Date"`
	Ticker        string `csv:"Ticker"`
	Type          string `csv:"Type"`
	Quantity      string `csv:"Quantity"`
	PricePerShare string `csv:"Price per share"`
	TotalAmount   string `csv:"Total Amount"`
	Currency      string `csv:"Currency"`
	FXRate        string `csv:"FX Rate"`
}

var logger = logging.GetLogger()

// Parse parses a Revolut investment CSV file from an io.Reader and returns a slice of transactions
func Parse(r io.Reader) ([]models.Transaction, error) {
	logger.Info("Parsing Revolut investment CSV from reader")

	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, &parsererror.InvalidFormatError{
			FilePath:       "(from reader)",
			ExpectedFormat: "Revolut Investment CSV",
			Msg:            "CSV file is empty or contains only headers",
		}
	}

	// Validate headers
	expectedHeaders := []string{"Date", "Ticker", "Type", "Quantity", "Price per share", "Total Amount", "Currency", "FX Rate"}
	if len(records[0]) < len(expectedHeaders) {
		return nil, &parsererror.InvalidFormatError{
			FilePath:       "(from reader)",
			ExpectedFormat: "Revolut Investment CSV",
			Msg:            "CSV file has insufficient columns",
		}
	}

	for i, header := range expectedHeaders {
		if strings.TrimSpace(records[0][i]) != header {
			return nil, &parsererror.InvalidFormatError{
				FilePath:       "(from reader)",
				ExpectedFormat: "Revolut Investment CSV",
				Msg:            fmt.Sprintf("unexpected header at position %d: expected '%s', got '%s'", i, header, strings.TrimSpace(records[0][i])),
			}
		}
	}

	var transactions []models.Transaction

	// Process each row (skip header)
	for i, record := range records[1:] {
		if len(record) < 8 {
			logger.Warn("Skipping row: insufficient columns",
				logging.Field{Key: "row", Value: i + 2})
			continue
		}

		row := RevolutInvestmentCSVRow{
			Date:          record[0],
			Ticker:        record[1],
			Type:          record[2],
			Quantity:      record[3],
			PricePerShare: record[4],
			TotalAmount:   record[5],
			Currency:      record[6],
			FXRate:        record[7],
		}

		transaction, err := convertRowToTransaction(row)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert row to transaction",
				logging.Field{Key: "row", Value: i + 2})
			continue
		}

		transactions = append(transactions, transaction)
	}

	logger.Info("Successfully parsed transactions from Revolut investment CSV",
		logging.Field{Key: "count", Value: len(transactions)})
	return transactions, nil
}

// convertRowToTransaction converts a RevolutInvestmentCSVRow to a models.Transaction
func convertRowToTransaction(row RevolutInvestmentCSVRow) (models.Transaction, error) {
	var transaction models.Transaction

	// Parse date
	transaction.Date = formatDate(row.Date)
	transaction.ValueDate = transaction.Date

	// Set investment details
	transaction.Investment = row.Ticker
	transaction.Fund = row.Ticker
	transaction.Type = row.Type

	// Parse currency
	transaction.Currency = row.Currency
	transaction.OriginalCurrency = row.Currency

	// Parse FX rate
	if row.FXRate != "" {
		if fxRate, err := decimal.NewFromString(row.FXRate); err == nil {
			transaction.ExchangeRate = fxRate
		} else {
			logger.WithError(err).Warn("Failed to parse FX rate",
				logging.Field{Key: "fxRate", Value: row.FXRate})
		}
	}

	// Handle different transaction types
	logger.Debug("Processing transaction type",
		logging.Field{Key: "type", Value: row.Type})
	switch {
	case strings.Contains(row.Type, "BUY"):
		logger.Debug("Processing BUY transaction")
		// Parse quantity
		if row.Quantity != "" {
			if quantity, err := decimal.NewFromString(row.Quantity); err == nil {
				transaction.NumberOfShares = int(quantity.IntPart())
			} else {
				return transaction, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Quantity",
					RawDataSnippet: row.Quantity,
					Msg:            fmt.Sprintf("failed to parse quantity: %v", err),
				}
			}
		}

		// Parse price per share
		if row.PricePerShare != "" {
			priceStr := strings.TrimPrefix(row.PricePerShare, "€")
			priceStr = strings.TrimPrefix(priceStr, "$")
			priceStr = strings.TrimPrefix(priceStr, "£")
			priceStr = strings.ReplaceAll(priceStr, ",", "")
			if price, err := decimal.NewFromString(priceStr); err == nil {
				transaction.AmountExclTax = price
			} else {
				return transaction, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Price per share",
					RawDataSnippet: row.PricePerShare,
					Msg:            fmt.Sprintf("failed to parse price per share: %v", err),
				}
			}
		}

		// Parse total amount
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// For buy transactions, this is typically a debit
				transaction.DebitFlag = true
				transaction.CreditDebit = models.TransactionTypeDebit
				logger.Debug("Set CreditDebit to DBIT for BUY transaction")
			} else {
				return transaction, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse total amount: %v", err),
				}
			}
		}

		transaction.Description = fmt.Sprintf("Buy %s shares of %s", row.Quantity, row.Ticker)

	case strings.Contains(row.Type, "DIVIDEND"):
		logger.Debug("Processing DIVIDEND transaction")
		// Parse dividend amount
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// Dividends are typically credits
				transaction.DebitFlag = false
				transaction.CreditDebit = models.TransactionTypeCredit
				transaction.Credit = amount
				logger.Debug("Set CreditDebit to CRDT for DIVIDEND transaction")
			} else {
				return transaction, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse dividend amount: %v", err),
				}
			}
		}

		transaction.Description = fmt.Sprintf("Dividend from %s", row.Ticker)

	case strings.Contains(row.Type, "CASH TOP-UP"):
		logger.Debug("Processing CASH TOP-UP transaction")
		// Parse cash top-up amount
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// Cash top-ups are typically credits
				transaction.DebitFlag = false
				transaction.CreditDebit = models.TransactionTypeCredit
				transaction.Credit = amount
				logger.Debug("Set CreditDebit to CRDT for CASH TOP-UP transaction")
			} else {
				return transaction, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse cash top-up amount: %v", err),
				}
			}
		}

		transaction.Description = "Cash top-up to investment account"

	default:
		logger.Debug("Processing default transaction")
		// Handle other transaction types
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// Default to debit for unknown transaction types
				transaction.DebitFlag = true
				transaction.CreditDebit = models.TransactionTypeDebit
				logger.Debug("Set CreditDebit to DBIT for default transaction")
			} else {
				return transaction, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse amount: %v", err),
				}
			}
		}
		transaction.Description = fmt.Sprintf("%s transaction for %s", row.Type, row.Ticker)
	}

	// Set party name
	if row.Ticker != "" {
		transaction.PartyName = fmt.Sprintf("Revolut Investment - %s", row.Ticker)
	} else {
		transaction.PartyName = "Revolut Investment"
	}

	// Set name
	transaction.Name = transaction.PartyName

	// Update derived fields
	transaction.UpdateNameFromParties()
	transaction.UpdateDebitCreditAmounts()

	return transaction, nil
}

// formatDate standardizes the date format
func formatDate(dateStr string) string {
	// Parse ISO format date
	if t, err := time.Parse(time.RFC3339Nano, dateStr); err == nil {
		return t.Format("02.01.2006") // Return as DD.MM.YYYY
	}

	// If parsing fails, return original string
	return dateStr
}

// WriteToCSV writes transactions to a CSV file
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	logger.Info("Writing transactions to CSV file",
		logging.Field{Key: "count", Value: len(transactions)},
		logging.Field{Key: "file", Value: csvFile})

	file, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.WithError(closeErr).Warn("Failed to close file")
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"BookkeepingNumber", "Status", "Date", "ValueDate", "Name", "Description", "RemittanceInfo", "PartyName", "PartyIBAN",
		"Amount", "CreditDebit", "IsDebit", "Debit", "Credit", "Currency", "AmountExclTax", "AmountTax", "TaxRate",
		"Recipient", "InvestmentType", "Number", "Category", "Type", "Fund", "NumberOfShares", "Fees", "IBAN",
		"EntryReference", "Reference", "AccountServicer", "BankTxCode", "OriginalCurrency", "OriginalAmount", "ExchangeRate",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write transactions
	for _, transaction := range transactions {
		record, err := transaction.MarshalCSV()
		if err != nil {
			logger.WithError(err).Warn("Failed to marshal transaction")
			continue
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write transaction: %w", err)
		}
	}

	logger.Info("Successfully wrote transactions to CSV file")
	return nil
}

// ConvertToCSV converts a Revolut investment CSV file to the standardized format
func ConvertToCSV(inputFile, outputFile string) error {
	logger.Info("Converting Revolut investment CSV file",
		logging.Field{Key: "input", Value: inputFile},
		logging.Field{Key: "output", Value: outputFile})

	// Open the input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.WithError(err).Warn("Failed to close file")
		}
	}()

	transactions, err := Parse(file)
	if err != nil {
		return fmt.Errorf("failed to parse input file: %w", err)
	}

	if err := WriteToCSV(transactions, outputFile); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	logger.Info("Successfully converted Revolut investment CSV file")
	return nil
}
