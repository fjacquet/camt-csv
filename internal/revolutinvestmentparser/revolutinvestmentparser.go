package revolutinvestmentparser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/common"
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

// ParseWithCategorizer parses a Revolut investment CSV file and categorizes transactions using the provided categorizer.
func ParseWithCategorizer(ctx context.Context, r io.Reader, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
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

		if err := ctx.Err(); err != nil {
			return nil, err
		}

		transaction, err := convertRowToTransaction(row, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert row to transaction",
				logging.Field{Key: "row", Value: i + 2})
			continue
		}

		transactions = append(transactions, transaction)
	}

	transactions, err = common.ProcessTransactionsWithCategorizationStats(ctx, transactions, logger, categorizer, "RevolutInvestment")
	if err != nil {
		return nil, err
	}

	logger.Info("Successfully parsed transactions from Revolut investment CSV",
		logging.Field{Key: "count", Value: len(transactions)})
	return transactions, nil
}

// convertRowToTransaction converts a RevolutInvestmentCSVRow to a models.Transaction using TransactionBuilder
func convertRowToTransaction(row RevolutInvestmentCSVRow, logger logging.Logger) (models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
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
		WithDatetime(formatDate(row.Date)).
		WithValueDatetime(formatDate(row.Date)).
		WithInvestment(row.Ticker).
		WithFund(row.Ticker).
		WithType(row.Type).
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
			quantity, err := parseDecimalField("Quantity", row.Quantity, row.Quantity, "quantity")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithNumberOfShares(quantity)
		}

		// Parse price per share for tax info
		if row.PricePerShare != "" {
			price, err := parseDecimalField("Price per share", row.PricePerShare, cleanAmountString(row.PricePerShare), "price per share")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithTaxInfo(price, decimal.Zero, decimal.Zero)
		}

		// Parse total amount
		if row.TotalAmount != "" {
			amount, err := parseDecimalField("Total Amount", row.TotalAmount, cleanAmountString(row.TotalAmount), "total amount")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithAmount(amount, row.Currency).AsDebit()
		}

		builder = builder.WithDescription(fmt.Sprintf("Buy %s shares of %s", row.Quantity, row.Ticker)).
			WithPayee(partyName, "")

	case strings.Contains(row.Type, "SELL"):
		logger.Debug("Processing SELL transaction")

		// Parse quantity
		if row.Quantity != "" {
			quantity, err := parseDecimalField("Quantity", row.Quantity, row.Quantity, "quantity")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithNumberOfShares(quantity)
		}

		// Parse total amount
		amount := models.ParseAmount(row.TotalAmount)
		builder = builder.WithAmount(amount, row.Currency).AsCredit() // SELL is incoming money

		// Set description
		description := fmt.Sprintf("Sold %s shares of %s", row.Quantity, row.Ticker)
		builder = builder.WithDescription(description).WithPayer(partyName, "")

	case strings.Contains(row.Type, "CUSTODY_FEE"):
		logger.Debug("Processing CUSTODY_FEE transaction")

		// Parse fee amount
		amount := models.ParseAmount(row.TotalAmount)
		builder = builder.WithAmount(amount, row.Currency).
			AsDebit(). // Fees are outgoing
			WithFees(amount)

		// Set description
		description := fmt.Sprintf("Custody fee for %s", row.Ticker)
		builder = builder.WithDescription(description).WithPayee(partyName, "")

	case strings.Contains(row.Type, "DIVIDEND"):
		logger.Debug("Processing DIVIDEND transaction")

		// Parse dividend amount
		if row.TotalAmount != "" {
			amount, err := parseDecimalField("Total Amount", row.TotalAmount, cleanAmountString(row.TotalAmount), "dividend amount")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithAmount(amount, row.Currency).AsCredit()
		}

		builder = builder.WithDescription(fmt.Sprintf("Dividend from %s", row.Ticker)).
			WithPayer(partyName, "")

	case strings.Contains(row.Type, "CASH TOP-UP"):
		logger.Debug("Processing CASH TOP-UP transaction")

		// Parse cash top-up amount
		if row.TotalAmount != "" {
			amount, err := parseDecimalField("Total Amount", row.TotalAmount, cleanAmountString(row.TotalAmount), "cash top-up amount")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithAmount(amount, row.Currency).AsCredit()
		}

		builder = builder.WithDescription("Cash top-up to investment account").
			WithPayer(partyName, "")

	default:
		logger.Debug("Processing default transaction")

		// Handle other transaction types
		if row.TotalAmount != "" {
			amount, err := parseDecimalField("Total Amount", row.TotalAmount, cleanAmountString(row.TotalAmount), "amount")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithAmount(amount, row.Currency).AsDebit()
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

// parseDecimalField parses one numeric column, or reports which one failed.
// raw is kept in the error for diagnosis; cleaned is what gets parsed.
func parseDecimalField(field, raw, cleaned, what string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(cleaned)
	if err != nil {
		return decimal.Zero, &parsererror.DataExtractionError{
			FilePath:       "(from reader)",
			FieldName:      field,
			RawDataSnippet: raw,
			Msg:            fmt.Sprintf("failed to parse %s: %v", what, err),
		}
	}
	return value, nil
}

// cleanAmountString removes currency symbols and codes from amount strings.
// Handles both symbol prefixes (€, $, £) and ISO 4217 code prefixes (e.g. "USD 2.84").
func cleanAmountString(amountStr string) string {
	s := strings.TrimSpace(amountStr)
	// Strip leading 3-letter currency code (e.g. "USD ", "EUR ", "CHF ")
	if len(s) > 4 && s[3] == ' ' {
		allAlpha := true
		for _, c := range s[:3] {
			if c < 'A' || c > 'Z' {
				allAlpha = false
				break
			}
		}
		if allAlpha {
			s = strings.TrimSpace(s[4:])
		}
	}
	// Strip symbol prefixes
	s = strings.TrimPrefix(s, "€")
	s = strings.TrimPrefix(s, "$")
	s = strings.TrimPrefix(s, "£")
	s = strings.ReplaceAll(s, ",", "")
	return strings.TrimSpace(s)
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
