// Package models provides the data structures used throughout the application.
package models

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Transaction represents a financial transaction from various sources
type Transaction struct {
	Date            string `csv:"Date"`           // Date in DD.MM.YYYY format
	ValueDate       string `csv:"ValueDate"`      // Value date in DD.MM.YYYY format
	Description     string `csv:"Description"`    // Description of the transaction
	BookkeepingNo   string `csv:"BookkeepingNo"`  // Bookkeeping number
	Fund            string `csv:"Fund"`           // Fund name if applicable
	Amount          string `csv:"Amount"`         // Amount as string (without currency symbol)
	Currency        string `csv:"Currency"`       // Currency code (CHF, EUR, etc)
	CreditDebit     string `csv:"CreditDebit"`    // Either "DBIT" (debit) or "CRDT" (credit)
	EntryReference  string `csv:"EntryReference"` // Entry reference number
	AccountServicer string `csv:"AccountServicer"`// Account servicer reference
	BankTxCode      string `csv:"BankTxCode"`     // Bank transaction code
	Status          string `csv:"Status"`         // Status code
	Payee           string `csv:"Payee"`          // Beneficiary/recipient name
	Payer           string `csv:"Payer"`          // Payer name
	IBAN            string `csv:"IBAN"`           // IBAN if available
	NumberOfShares  int    `csv:"NumberOfShares"` // Number of shares for investment transactions
	StampDuty       string `csv:"StampDutyAmount"`// Stamp duty
	Category        string `csv:"Category"`       // Transaction category
	Investment      string `csv:"Investment"`     // Investment type (Buy, Sell, Income, etc.)
}

// GetAmountAsFloat returns the Amount as a float64
func (t *Transaction) GetAmountAsFloat() float64 {
	// Replace comma with dot for decimal separator
	amount := strings.ReplaceAll(t.Amount, ",", ".")
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

	// Convert to float
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0
	}
	return f
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
	
	// Convert to float and back to string to standardize decimal places
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return amountStr // Return original if parsing fails
	}
	
	// Format with exactly 2 decimal places
	formatted := strconv.FormatFloat(f, 'f', 2, 64)
	
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
