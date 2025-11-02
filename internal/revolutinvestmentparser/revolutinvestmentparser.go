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

// convertRowToTransaction converts a RevolutInvestmentCSVRow to a models.Transaction using TransactionBuilder
func convertRowToTransaction(row RevolutInvestmentCSVRow) (models.Transaction, error) {
	// Parse FX rate
	var fxRate decimal.Decimal
	if row.FXRate != "" {
		var err error
		fxRate, err = decimal.NewFromString(row.FXRate)
		if err != nil {
			logger.WithError(err).Warn("Failed to parse FX rate",
				logging.Field{Key: "fxRate", Value: row.FXRate})
			fxRate = decimal.NewFromInt(1) // Default to 1
		}
	} else {
		fxRate = decimal.NewFromInt(1)
	}

	// Start building the transaction
	builder := models.NewTransactionBuilder().
		WithDateFromTime(formatDate(row.Date)).
		WithValueDateFromTime(formatDate(row.Date)).
		WithInvestment(row.Ticker).
		WithFund(row.Ticker).
		WithType(row.Type).
		WithCurrency(row.Currency).
		WithOriginalAmount(decimal.Zero, row.Currency).
		WithExchangeRate(fxRate)

	// Set party name
	partyName := "Revolut Investment"
	if row.Ticker != "" {
		partyName = fmt.Sprintf("Revolut Investment - %s", row.Ticker)
	}
	builder = builder.WithPartyName(partyName)

	// Handle different transaction types
	logger.Debug("Processing transaction type",
		logging.Field{Key: "type", Value: row.Type})
	
	switch {
	case strings.Contains(row.Type, "BUY"):
		logger.Debug("Processing BUY transaction")
		
		// Parse quantity
		if row.Quantity != "" {
			if quantity, err := decimal.NewFromString(row.Quantity); err == nil {
				builder = builder.WithNumberOfShares(int(quantity.IntPart()))
			} else {
				return models.Transaction{}, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Quantity",
					RawDataSnippet: row.Quantity,
					Msg:            fmt.Sprintf("failed to parse quantity: %v", err),
				}
			}
		}

		// Parse price per share for tax info
		if row.PricePerShare != "" {
			priceStr := cleanAmountString(row.PricePerShare)
			if price, err := decimal.NewFromString(priceStr); err == nil {
				builder = builder.WithTaxInfo(price, decimal.Zero, decimal.Zero)
			} else {
				return models.Transaction{}, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Price per share",
					RawDataSnippet: row.PricePerShare,
					Msg:            fmt.Sprintf("failed to parse price per share: %v", err),
				}
			}
		}

		// Parse total amount
		if row.TotalAmount != "" {
			amountStr := cleanAmountString(row.TotalAmount)
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				builder = builder.WithAmount(amount, row.Currency).AsDebit()
			} else {
				return models.Transaction{}, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse total amount: %v", err),
				}
			}
		}

		builder = builder.WithDescription(fmt.Sprintf("Buy %s shares of %s", row.Quantity, row.Ticker)).
			WithPayee(partyName, "")

	case strings.Contains(row.Type, "DIVIDEND"):
		logger.Debug("Processing DIVIDEND transaction")
		
		// Parse dividend amount
		if row.TotalAmount != "" {
			amountStr := cleanAmountString(row.TotalAmount)
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				builder = builder.WithAmount(amount, row.Currency).AsCredit()
			} else {
				return models.Transaction{}, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse dividend amount: %v", err),
				}
			}
		}

		builder = builder.WithDescription(fmt.Sprintf("Dividend from %s", row.Ticker)).
			WithPayer(partyName, "")

	case strings.Contains(row.Type, "CASH TOP-UP"):
		logger.Debug("Processing CASH TOP-UP transaction")
		
		// Parse cash top-up amount
		if row.TotalAmount != "" {
			amountStr := cleanAmountString(row.TotalAmount)
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				builder = builder.WithAmount(amount, row.Currency).AsCredit()
			} else {
				return models.Transaction{}, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse cash top-up amount: %v", err),
				}
			}
		}

		builder = builder.WithDescription("Cash top-up to investment account").
			WithPayer(partyName, "")

	default:
		logger.Debug("Processing default transaction")
		
		// Handle other transaction types
		if row.TotalAmount != "" {
			amountStr := cleanAmountString(row.TotalAmount)
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				builder = builder.WithAmount(amount, row.Currency).AsDebit()
			} else {
				return models.Transaction{}, &parsererror.DataExtractionError{
					FilePath:       "(from reader)",
					FieldName:      "Total Amount",
					RawDataSnippet: row.TotalAmount,
					Msg:            fmt.Sprintf("failed to parse amount: %v", err),
				}
			}
		}
		
		builder = builder.WithDescription(fmt.Sprintf("%s transaction for %s", row.Type, row.Ticker)).
			WithPayee(partyName, "")
	}

	// Build the transaction
	transaction, err := builder.Build()
	if err != nil {
		return models.Transaction{}, fmt.Errorf("error building transaction: %w", err)
	}

	return transaction, nil
}

// cleanAmountString removes currency symbols and formatting from amount strings
func cleanAmountString(amountStr string) string {
	cleaned := strings.TrimPrefix(amountStr, "€")
	cleaned = strings.TrimPrefix(cleaned, "$")
	cleaned = strings.TrimPrefix(cleaned, "£")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	return cleaned
}

// formatDate parses the date string and returns time.Time
func formatDate(dateStr string) time.Time {
	// Parse ISO format date
	if t, err := time.Parse(time.RFC3339Nano, dateStr); err == nil {
		return t
	}

	// If parsing fails, return zero time
	return time.Time{}
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
