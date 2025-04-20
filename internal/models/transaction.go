// Package models provides the data structures used throughout the application.
package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	
	"github.com/shopspring/decimal"
)

// Transaction represents a financial transaction from various sources
type Transaction struct {
	Date             string          `csv:"Date"`           // Date in DD.MM.YYYY format
	ValueDate        string          `csv:"ValueDate"`      // Value date in DD.MM.YYYY format
	Description      string          `csv:"Description"`    // Description of the transaction
	BookkeepingNo    string          `csv:"BookkeepingNo"`  // Bookkeeping number
	Fund             string          `csv:"Fund"`           // Fund name if applicable
	Amount           decimal.Decimal `csv:"Amount"`         // Amount as decimal value
	Currency         string          `csv:"Currency"`       // Currency code (CHF, EUR, etc)
	CreditDebit      string          `csv:"CreditDebit"`    // Either "DBIT" (debit) or "CRDT" (credit)
	EntryReference   string          `csv:"EntryReference"` // Entry reference number
	AccountServicer  string          `csv:"AccountServicer"`// Account servicer reference
	BankTxCode       string          `csv:"BankTxCode"`     // Bank transaction code
	Status           string          `csv:"Status"`         // Status code
	Payee            string          `csv:"Payee"`          // Beneficiary/recipient name
	Payer            string          `csv:"Payer"`          // Payer name
	IBAN             string          `csv:"IBAN"`           // IBAN if available
	NumberOfShares   int             `csv:"NumberOfShares"` // Number of shares for investment transactions
	StampDuty        string          `csv:"StampDutyAmount"`// Stamp duty
	Category         string          `csv:"Category"`       // Transaction category
	Investment       string          `csv:"Investment"`     // Investment type (Buy, Sell, Income, etc.)
	OriginalCurrency string          `csv:"OriginalCurrency"` // Original currency for foreign currency transactions
	OriginalAmount   decimal.Decimal `csv:"OriginalAmount"`   // Original amount in foreign currency
	ExchangeRate     decimal.Decimal `csv:"ExchangeRate"`     // Exchange rate for currency conversion
	Fee              decimal.Decimal `csv:"Fee"`              // Transaction fees
}

// ParseAmount parses a string amount to decimal.Decimal with proper formatting
// This is a utility function for converting string representations to the decimal type
func ParseAmount(amountStr string) decimal.Decimal {
	// Replace comma with dot for decimal separator
	amount := strings.ReplaceAll(amountStr, ",", ".")
	// Remove any currency symbols or spaces
	amount = strings.TrimSpace(amount)
	amount = strings.ReplaceAll(amount, " ", "")
	amount = strings.ReplaceAll(amount, "CHF", "")
	amount = strings.ReplaceAll(amount, "EUR", "")
	amount = strings.ReplaceAll(amount, "USD", "")
	amount = strings.ReplaceAll(amount, "$", "")
	amount = strings.ReplaceAll(amount, "€", "")
	// Remove thousand separators
	amount = strings.ReplaceAll(amount, "'", "")

	// Convert to decimal
	dec, err := decimal.NewFromString(amount)
	if err != nil {
		return decimal.Zero
	}
	return dec
}

// GetAmountAsFloat returns the Amount as a float64
// DEPRECATED: This method is maintained for backward compatibility only.
// Use direct decimal operations instead for financial calculations.
func (t *Transaction) GetAmountAsFloat() float64 {
	f, _ := t.Amount.Float64()
	return f
}

// GetOriginalAmountAsFloat returns the OriginalAmount as a float64
// DEPRECATED: This method is maintained for backward compatibility only.
func (t *Transaction) GetOriginalAmountAsFloat() float64 {
	f, _ := t.OriginalAmount.Float64()
	return f
}

// GetAmountAsDecimal returns the Amount as a decimal.Decimal for precise calculations
// This is the recommended way to access the Amount field for financial calculations
func (t *Transaction) GetAmountAsDecimal() decimal.Decimal {
	return t.Amount
}

// SetAmountFromDecimal sets the Amount field from a decimal.Decimal value
func (t *Transaction) SetAmountFromDecimal(amount decimal.Decimal) {
	t.Amount = amount
}

// GetOriginalAmountAsDecimal returns the OriginalAmount as a decimal.Decimal
func (t *Transaction) GetOriginalAmountAsDecimal() decimal.Decimal {
	return t.OriginalAmount
}

// GetExchangeRateAsDecimal returns the ExchangeRate as a decimal.Decimal
func (t *Transaction) GetExchangeRateAsDecimal() decimal.Decimal {
	return t.ExchangeRate
}

// GetFeeAsDecimal returns the Fee as a decimal.Decimal
func (t *Transaction) GetFeeAsDecimal() decimal.Decimal {
	return t.Fee
}

// IsDebit returns true if the transaction is a debit (outgoing money)
func (t *Transaction) IsDebit() bool {
	return t.CreditDebit == "DBIT"
}

// IsCredit returns true if the transaction is a credit (incoming money)
func (t *Transaction) IsCredit() bool {
	return t.CreditDebit == "CRDT"
}

// GetPartyName returns the relevant party name based on the transaction type
func (t *Transaction) GetPartyName() string {
	if t.IsDebit() {
		return t.Payee // For outgoing money, the payee is the relevant party
	}
	return t.Payer // For incoming money, the payer is the relevant party
}

// StandardizeAmount formats amount consistently with 2 decimal places
func StandardizeAmount(amountStr string) string {
	// Remove any currency symbols, spaces, commas, etc.
	amount := strings.TrimSpace(amountStr)
	amount = strings.ReplaceAll(amount, " ", "")
	amount = strings.ReplaceAll(amount, "'", "")
	amount = strings.ReplaceAll(amount, "CHF", "")
	amount = strings.ReplaceAll(amount, "EUR", "")
	amount = strings.ReplaceAll(amount, "USD", "")
	amount = strings.ReplaceAll(amount, "$", "")
	amount = strings.ReplaceAll(amount, "€", "")
	
	// Replace comma with dot for decimal separator
	amount = strings.ReplaceAll(amount, ",", ".")
	
	// Remove minus sign if present (we'll handle the sign separately)
	isNegative := strings.HasPrefix(amount, "-")
	if isNegative {
		amount = strings.TrimPrefix(amount, "-")
	}
	
	// Convert to decimal and back to string to standardize decimal places
	dec, err := decimal.NewFromString(amount)
	if err != nil {
		return amountStr // Return original if parsing fails
	}
	
	// Format with exactly 2 decimal places
	formatted := dec.StringFixed(2)
	
	// Add back minus sign if it was negative
	if isNegative {
		return "-" + formatted
	}
	
	return formatted
}

// FormatDate standardizes date strings to the DD.MM.YYYY format
// It handles various input formats from different sources
func FormatDate(dateStr string) string {
	// Skip processing if empty
	if dateStr == "" {
		return ""
	}
	
	// Clean the input string
	cleanDate := strings.TrimSpace(dateStr)
	
	// If it's already in the target format (DD.MM.YYYY), return as is
	if matched, _ := regexp.MatchString(`^\d{2}\.\d{2}\.\d{4}$`, cleanDate); matched {
		return cleanDate
	}
	
	// Try various date formats commonly found in financial data
	formats := []string{
		"2006-01-02",                 // YYYY-MM-DD (ISO)
		"2006-01-02 15:04:05",        // YYYY-MM-DD HH:MM:SS
		"2006-01-02T15:04:05Z",       // ISO 8601
		"2006-01-02T15:04:05-07:00",  // ISO 8601 with timezone
		"02/01/2006",                 // DD/MM/YYYY
		"01/02/2006",                 // MM/DD/YYYY (US format)
		"02-01-2006",                 // DD-MM-YYYY
		"01-02-2006",                 // MM-DD-YYYY
		"2.1.2006",                   // D.M.YYYY
		"02.01.2006",                 // DD.MM.YYYY
		"January 2, 2006",            // Month D, YYYY
		"2 January 2006",             // D Month YYYY
		"02 Jan 2006",                // DD MMM YYYY
		"Jan 02, 2006",               // MMM DD, YYYY
		"January 2006",               // Month YYYY (for monthly statements)
		"Jan 2006",                   // MMM YYYY (abbreviated month)
		"01/2006",                    // MM/YYYY
		"2006/01",                    // YYYY/MM
	}
	
	// European day-first preference (important for ambiguous formats)
	// This list matches the same formats but with European day-first parsing
	europeanFormats := []string{
		"02/01/2006",                 // DD/MM/YYYY (European)
		"02-01-2006",                 // DD-MM-YYYY (European)
	}
	
	// First try European formats (as they're more common in Swiss financial data)
	for _, format := range europeanFormats {
		if t, err := time.Parse(format, cleanDate); err == nil {
			return t.Format("02.01.2006") // Return as DD.MM.YYYY
		}
	}
	
	// Then try all other formats
	for _, format := range formats {
		if t, err := time.Parse(format, cleanDate); err == nil {
			return t.Format("02.01.2006") // Return as DD.MM.YYYY
		}
	}
	
	// If we can't parse the date, log a warning and return the original
	// log.WithField("date", dateStr).Warning("Unable to parse date format")
	return dateStr
}

func (t *Transaction) MarshalCSV() ([]string, error) {
	return []string{
		t.Date,
		t.ValueDate,
		t.Description,
		t.BookkeepingNo,
		t.Fund,
		t.Amount.StringFixed(2),
		t.Currency,
		t.CreditDebit,
		t.EntryReference,
		t.AccountServicer,
		t.BankTxCode,
		t.Status,
		t.Payee,
		t.Payer,
		t.IBAN,
		fmt.Sprintf("%d", t.NumberOfShares),
		t.StampDuty,
		t.Category,
		t.Investment,
		t.OriginalCurrency,
		t.OriginalAmount.StringFixed(2),
		t.ExchangeRate.StringFixed(2),
		t.Fee.StringFixed(2),
	}, nil
}

func (t *Transaction) UnmarshalCSV(record []string) error {
	t.Date = record[0]
	t.ValueDate = record[1]
	t.Description = record[2]
	t.BookkeepingNo = record[3]
	t.Fund = record[4]
	var err error
	t.Amount, err = decimal.NewFromString(record[5])
	if err != nil {
		return err
	}
	t.Currency = record[6]
	t.CreditDebit = record[7]
	t.EntryReference = record[8]
	t.AccountServicer = record[9]
	t.BankTxCode = record[10]
	t.Status = record[11]
	t.Payee = record[12]
	t.Payer = record[13]
	t.IBAN = record[14]
	var numberOfShares int
	numberOfShares, err = strconv.Atoi(record[15])
	if err != nil {
		return err
	}
	t.NumberOfShares = numberOfShares
	t.StampDuty = record[16]
	t.Category = record[17]
	t.Investment = record[18]
	t.OriginalCurrency = record[19]
	t.OriginalAmount, err = decimal.NewFromString(record[20])
	if err != nil {
		return err
	}
	t.ExchangeRate, err = decimal.NewFromString(record[21])
	if err != nil {
		return err
	}
	t.Fee, err = decimal.NewFromString(record[22])
	if err != nil {
		return err
	}
	return nil
}
