// Package revolutcryptoparser parses Revolut Crypto account statement CSV files.
// It handles French locale numbers ("69 924,87 CHF") and French locale dates
// ("25 janv. 2026, 13:15:23"), converting them to standard Transaction models.
package revolutcryptoparser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"

	"github.com/shopspring/decimal"
)

// cryptoCSVRow represents one row in a Revolut Crypto account statement.
type cryptoCSVRow struct {
	Symbol   string
	Type     string
	Quantity string
	Price    string
	Value    string
	Fees     string
	Date     string
}

// frenchMonths maps French abbreviated month names to zero-padded month numbers.
var frenchMonths = map[string]string{
	"janv.": "01",
	"févr.": "02",
	"mars":  "03",
	"avr.":  "04",
	"mai":   "05",
	"juin":  "06",
	"juil.": "07",
	"août":  "08",
	"sept.": "09",
	"oct.":  "10",
	"nov.":  "11",
	"déc.":  "12",
}

// parseFrenchDate parses a French locale date string like "25 janv. 2026, 13:15:23".
func parseFrenchDate(s string) time.Time {
	// Remove the comma after the year: "25 janv. 2026, 13:15:23" → "25 janv. 2026 13:15:23"
	s = strings.ReplaceAll(s, ",", "")
	parts := strings.Fields(s)
	// Expected parts: ["25", "janv.", "2026", "13:15:23"]
	if len(parts) != 4 {
		return time.Time{}
	}

	month, ok := frenchMonths[parts[1]]
	if !ok {
		return time.Time{}
	}

	// Build ISO-like string: "2026-01-25 13:15:23"
	normalized := fmt.Sprintf("%s-%s-%02s %s", parts[2], month, parts[0], parts[3])
	t, err := time.ParseInLocation("2006-01-02 15:04:05", normalized, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// parseFrenchAmount parses a French locale amount string like "69 924,87 CHF".
// Returns (decimal.Zero, "") for empty input.
func parseFrenchAmount(s string) (decimal.Decimal, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, ""
	}

	// Extract trailing currency code (3 uppercase letters after a space)
	currency := ""
	if len(s) >= 4 {
		last := s[len(s)-3:]
		allAlpha := true
		for _, c := range last {
			if c < 'A' || c > 'Z' {
				allAlpha = false
				break
			}
		}
		if allAlpha && s[len(s)-4] == ' ' {
			currency = last
			s = strings.TrimSpace(s[:len(s)-4])
		}
	}

	// Remove thousands separator (space) and convert decimal separator (comma → dot)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")

	amount, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, currency
	}
	return amount, currency
}

// ParseWithCategorizer parses a Revolut Crypto CSV reader and returns transactions.
func ParseWithCategorizer(r io.Reader, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	logger.Info("Parsing Revolut Crypto CSV from reader")

	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, &parsererror.InvalidFormatError{
			FilePath:       "(from reader)",
			ExpectedFormat: "Revolut Crypto CSV",
			Msg:            "CSV file is empty or contains only headers",
		}
	}

	// Validate headers
	expected := []string{"Symbol", "Type", "Quantity", "Price", "Value", "Fees", "Date"}
	if len(records[0]) < len(expected) {
		return nil, &parsererror.InvalidFormatError{
			FilePath:       "(from reader)",
			ExpectedFormat: "Revolut Crypto CSV",
			Msg:            "CSV file has insufficient columns",
		}
	}
	for i, h := range expected {
		if strings.TrimSpace(records[0][i]) != h {
			return nil, &parsererror.InvalidFormatError{
				FilePath:       "(from reader)",
				ExpectedFormat: "Revolut Crypto CSV",
				Msg:            fmt.Sprintf("unexpected header at position %d: expected %q, got %q", i, h, strings.TrimSpace(records[0][i])),
			}
		}
	}

	var transactions []models.Transaction

	for i, record := range records[1:] {
		if len(record) < 7 {
			logger.Warn("Skipping row: insufficient columns",
				logging.Field{Key: "row", Value: i + 2})
			continue
		}

		row := cryptoCSVRow{
			Symbol:   strings.TrimSpace(record[0]),
			Type:     strings.TrimSpace(record[1]),
			Quantity: strings.TrimSpace(record[2]),
			Price:    strings.TrimSpace(record[3]),
			Value:    strings.TrimSpace(record[4]),
			Fees:     strings.TrimSpace(record[5]),
			Date:     strings.TrimSpace(record[6]),
		}

		tx, err := convertRowToTransaction(row)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert row to transaction",
				logging.Field{Key: "row", Value: i + 2})
			continue
		}

		if categorizer != nil {
			isDebtor := tx.CreditDebit == models.TransactionTypeDebit
			catDate := ""
			if !tx.Date.IsZero() {
				catDate = tx.Date.Format("02.01.2006")
			}
			category, catErr := categorizer.Categorize(context.Background(), tx.PartyName, isDebtor, tx.Amount.String(), catDate, "")
			if catErr != nil {
				logger.WithError(catErr).Warn("Failed to categorize transaction",
					logging.Field{Key: "party", Value: tx.PartyName})
				tx.Category = models.CategoryUncategorized
			} else {
				tx.Category = category.Name
			}
		} else {
			tx.Category = models.CategoryUncategorized
		}

		transactions = append(transactions, tx)
	}

	logger.Info("Successfully parsed transactions from Revolut Crypto CSV",
		logging.Field{Key: "count", Value: len(transactions)})
	return transactions, nil
}

// convertRowToTransaction converts a cryptoCSVRow to a models.Transaction.
func convertRowToTransaction(row cryptoCSVRow) (models.Transaction, error) {
	date := parseFrenchDate(row.Date)
	partyName := fmt.Sprintf("Revolut Crypto - %s", row.Symbol)

	builder := models.NewTransactionBuilder().
		WithDatetime(date).
		WithValueDatetime(date).
		WithPartyName(partyName).
		WithType(row.Type).
		WithFund(row.Symbol).
		WithInvestment(row.Symbol)

	switch {
	case row.Type == "Achat":
		amount, currency := parseFrenchAmount(row.Value)
		if currency == "" {
			currency = "CHF"
		}
		fees, _ := parseFrenchAmount(row.Fees)

		description := fmt.Sprintf("Achat %s (%s %s)", row.Symbol, row.Quantity, row.Symbol)
		builder = builder.
			WithAmount(amount, currency).
			WithFees(fees).
			WithDescription(description).
			WithPayee(partyName, "").
			AsDebit()

	case row.Type == "Récompense de staking":
		// No CHF value — record the crypto quantity received as the amount in crypto currency
		qty, _ := parseFrenchAmount(row.Quantity)
		if qty.IsZero() {
			qty = decimal.New(1, -8) // guard: builder rejects zero amount
		}
		description := fmt.Sprintf("Récompense de staking %s (%s %s)", row.Symbol, row.Quantity, row.Symbol)
		builder = builder.
			WithAmount(qty, row.Symbol).
			WithDescription(description).
			WithPayer(partyName, "").
			AsCredit()

	default:
		// Unknown type: use Value if present, otherwise zero
		amount, currency := parseFrenchAmount(row.Value)
		if currency == "" {
			currency = "CHF"
		}
		description := fmt.Sprintf("%s %s (%s %s)", row.Type, row.Symbol, row.Quantity, row.Symbol)
		builder = builder.
			WithAmount(amount, currency).
			WithDescription(description).
			WithPayee(partyName, "").
			AsDebit()
	}

	tx, err := builder.Build()
	if err != nil {
		return models.Transaction{}, fmt.Errorf("error building transaction: %w", err)
	}
	return tx, nil
}
