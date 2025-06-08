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
	BookkeepingNumber string          `csv:"BookkeepingNumber"` // Bookkeeping number (replaces BookkeepingNo)
	Status            string          `csv:"Status"`            // Status code
	Date              string          `csv:"Date"`              // Date in DD.MM.YYYY format
	ValueDate         string          `csv:"ValueDate"`         // Value date in DD.MM.YYYY format
	Name              string          `csv:"Name"`              // Name of the other party (combined from Payee/Payer)
	PartyName         string          `csv:"PartyName"`         // Name of the other party (standardized field)
	PartyIBAN         string          `csv:"PartyIBAN"`         // IBAN of the other party
	Description       string          `csv:"Description"`       // Description of the transaction
	RemittanceInfo    string          `csv:"RemittanceInfo"`    // Unstructured remittance information
	Amount            decimal.Decimal `csv:"Amount"`            // Amount as decimal value
	CreditDebit       string          `csv:"CreditDebit"`       // Either "DBIT" (debit) or "CRDT" (credit)
	DebitFlag         bool            `csv:"IsDebit"`           // True if transaction is a debit, false if credit
	Debit             decimal.Decimal `csv:"Debit"`             // Debit amount (negative)
	Credit            decimal.Decimal `csv:"Credit"`            // Credit amount (positive)
	Currency          string          `csv:"Currency"`          // Currency code (CHF, EUR, etc)
	AmountExclTax     decimal.Decimal `csv:"AmountExclTax"`     // Amount excluding tax
	AmountTax         decimal.Decimal `csv:"AmountTax"`         // Tax amount
	TaxRate           decimal.Decimal `csv:"TaxRate"`           // Tax rate percentage
	Recipient         string          `csv:"Recipient"`         // Recipient/beneficiary name
	Investment        string          `csv:"InvestmentType"`    // Type of investment (Buy, Sell, Income, etc.)
	Number            string          `csv:"Number"`            // Transaction number
	Category          string          `csv:"Category"`          // Transaction category
	Type              string          `csv:"Type"`              // Transaction type
	Fund              string          `csv:"Fund"`              // Fund name if applicable
	NumberOfShares    int             `csv:"NumberOfShares"`    // Number of shares for investment transactions
	Fees              decimal.Decimal `csv:"Fees"`              // Transaction fees (includes stamp duty)
	IBAN              string          `csv:"IBAN"`              // IBAN if available
	EntryReference    string          `csv:"EntryReference"`    // Entry reference number
	Reference         string          `csv:"Reference"`         // Reference number
	AccountServicer   string          `csv:"AccountServicer"`   // Account servicer reference
	BankTxCode        string          `csv:"BankTxCode"`        // Bank transaction code
	OriginalCurrency  string          `csv:"OriginalCurrency"`  // Original currency for foreign currency transactions
	OriginalAmount    decimal.Decimal `csv:"OriginalAmount"`    // Original amount in foreign currency
	ExchangeRate      decimal.Decimal `csv:"ExchangeRate"`      // Exchange rate for currency conversion
	
	// Fields not exported to CSV but used internally
	Payee             string          `csv:"-"`                 // Beneficiary/recipient name (kept for backwards compatibility)
	Payer             string          `csv:"-"`                 // Payer name (kept for backwards compatibility)
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

// GetFeesAsDecimal returns the transaction fees as a decimal.Decimal
func (t *Transaction) GetFeesAsDecimal() decimal.Decimal {
	return t.Fees
}

// SetFeesFromDecimal sets the transaction fees from a decimal.Decimal value
func (t *Transaction) SetFeesFromDecimal(fees decimal.Decimal) {
	t.Fees = fees
}

// IsDebit returns true if the transaction is a debit (outgoing money)
func (t *Transaction) IsDebit() bool {
	return t.DebitFlag || t.CreditDebit == "DBIT" || t.Amount.IsNegative()
}

// IsCredit returns true if the transaction is a credit (incoming money)
func (t *Transaction) IsCredit() bool {
	return t.CreditDebit == "CRDT" || (t.CreditDebit != "DBIT" && t.CreditDebit != "UNKNOWN" && !t.DebitFlag && !t.Amount.IsNegative())
}

// UpdateNameFromParties sets the Name field based on the transaction type
// - For debits, Name is set to Payee
// - For credits, Name is set to Payer
func (t *Transaction) UpdateNameFromParties() {
	if t.IsDebit() {
		t.Name = t.Payee
	} else if t.IsCredit() {
		t.Name = t.Payer
	}
}

// UpdateRecipientFromPayee sets the Recipient field from Payee for compatibility
func (t *Transaction) UpdateRecipientFromPayee() {
	t.Recipient = t.Payee
}

// UpdateInvestmentTypeFromInvestment ensures the InvestmentType field is set
// UpdateInvestmentTypeFromLegacyField populates Investment when legacy parsers stored
// this information in the Type field. If Investment is empty and Type is set, copy it.
func (t *Transaction) UpdateInvestmentTypeFromLegacyField() {
        if t.Investment == "" && t.Type != "" {
                t.Investment = t.Type
        }
}

// UpdateDebitCreditAmounts populates the Debit and Credit fields based on the main Amount
func (t *Transaction) UpdateDebitCreditAmounts() {
	if t.IsDebit() {
		t.Debit = t.Amount
		t.Credit = decimal.Zero
	} else if t.IsCredit() {
		t.Credit = t.Amount
		t.Debit = decimal.Zero
	}
}

// GetPartyName returns the relevant party name based on the transaction type
func (t *Transaction) GetPartyName() string {
	if t.IsDebit() {
		return t.Payee
	}
	return t.Payer
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
	// Make sure the derived fields are populated correctly
	t.UpdateNameFromParties()
	t.UpdateRecipientFromPayee()
	t.UpdateDebitCreditAmounts()
	t.UpdateInvestmentTypeFromLegacyField()

	return []string{
		t.BookkeepingNumber,
		t.Status,
		t.Date,
		t.ValueDate,
		t.Name,
		t.Description,
		t.RemittanceInfo,
		t.PartyName,
		t.PartyIBAN,
		t.Amount.StringFixed(2),
		t.CreditDebit,
		fmt.Sprintf("%t", t.DebitFlag),
		t.Debit.StringFixed(2),
		t.Credit.StringFixed(2),
		t.Currency,
		t.AmountExclTax.StringFixed(2),
		t.AmountTax.StringFixed(2),
		t.TaxRate.StringFixed(2),
		t.Recipient,
		t.Investment,
		t.Number,
		t.Category,
		t.Type,
		t.Fund,
		fmt.Sprintf("%d", t.NumberOfShares),
		t.Fees.StringFixed(2),
		t.IBAN,
		t.EntryReference,
		t.Reference,
		t.AccountServicer,
		t.BankTxCode,
		t.OriginalCurrency,
		t.OriginalAmount.StringFixed(2),
		t.ExchangeRate.StringFixed(2),
	}, nil
}

func (t *Transaction) UnmarshalCSV(record []string) error {
	t.BookkeepingNumber = record[0]
	t.Status = record[1]
	t.Date = record[2]
	t.ValueDate = record[3]
	t.Name = record[4]
	t.Description = record[5]
	t.RemittanceInfo = record[6]
	t.PartyName = record[7]
	t.PartyIBAN = record[8]
	var err error
	t.Amount, err = decimal.NewFromString(record[9])
	if err != nil {
		return err
	}
	t.CreditDebit = record[10]
	t.DebitFlag, err = strconv.ParseBool(record[11])
	if err != nil {
		return err
	}
	t.Debit, err = decimal.NewFromString(record[12])
	if err != nil {
		return err
	}
	t.Credit, err = decimal.NewFromString(record[13])
	if err != nil {
		return err
	}
	t.Currency = record[14]
	t.AmountExclTax, err = decimal.NewFromString(record[15])
	if err != nil {
		return err
	}
	t.AmountTax, err = decimal.NewFromString(record[16])
	if err != nil {
		return err
	}
	t.TaxRate, err = decimal.NewFromString(record[17])
	if err != nil {
		return err
	}
	t.Recipient = record[18]
	t.Investment = record[19]
	t.Number = record[20]
	t.Category = record[21]
	t.Type = record[22]
	t.Fund = record[23]
	var numberOfShares int
	numberOfShares, err = strconv.Atoi(record[24])
	if err != nil {
		return err
	}
	t.NumberOfShares = numberOfShares
	// Fees includes stamp duty
	t.Fees, err = decimal.NewFromString(record[25])
	if err != nil {
		return err
	}
	t.IBAN = record[26]
	t.EntryReference = record[27]
	t.Reference = record[28]
	t.AccountServicer = record[29]
	t.BankTxCode = record[30]
	t.OriginalCurrency = record[31]
	t.OriginalAmount, err = decimal.NewFromString(record[32])
	if err != nil {
		return err
	}
	t.ExchangeRate, err = decimal.NewFromString(record[33])
	if err != nil {
		return err
	}
	return nil
}
