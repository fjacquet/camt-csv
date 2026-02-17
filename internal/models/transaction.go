// Package models provides the data structures used throughout the application.
package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents a financial transaction from various sources
type Transaction struct {
	BookkeepingNumber string          `csv:"BookkeepingNumber"` // Bookkeeping number (replaces BookkeepingNo)
	Status            string          `csv:"Status"`            // Status code
	Date              time.Time       `csv:"Date"`              // Transaction date
	ValueDate         time.Time       `csv:"ValueDate"`         // Value date
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
	Product           string          `csv:"Product"`           // Product type (Current, Savings)
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
	Payee string `csv:"-"` // Beneficiary/recipient name (kept for backwards compatibility)
	Payer string `csv:"-"` // Payer name (kept for backwards compatibility)
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

// GetCounterparty returns the relevant party name based on transaction direction
// For debits, returns the payee (who receives the money)
// For credits, returns the payer (who sent the money)
func (t *Transaction) GetCounterparty() string {
	if t.IsDebit() {
		return t.Payee
	}
	return t.Payer
}

// GetAmountAsDecimal returns the Amount as a decimal.Decimal for precise calculations
// This is the recommended way to access the Amount field for financial calculations

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
	return t.DebitFlag || t.CreditDebit == TransactionTypeDebit || t.Amount.IsNegative()
}

// IsCredit returns true if the transaction is a credit (incoming money)
func (t *Transaction) IsCredit() bool {
	return t.CreditDebit == TransactionTypeCredit || (t.CreditDebit != TransactionTypeDebit && t.CreditDebit != "UNKNOWN" && !t.DebitFlag && !t.Amount.IsNegative())
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

func (t *Transaction) MarshalCSV() ([]string, error) {
	// Make sure the derived fields are populated correctly
	t.UpdateNameFromParties()
	t.UpdateRecipientFromPayee()
	t.UpdateDebitCreditAmounts()
	t.UpdateInvestmentTypeFromLegacyField()

	return []string{
		t.Status,
		t.formatDateForCSV(t.Date),
		t.formatDateForCSV(t.ValueDate),
		t.Name,
		t.PartyName,
		t.PartyIBAN,
		t.Description,
		t.RemittanceInfo,
		t.Amount.StringFixed(2),
		t.CreditDebit,
		t.Currency,
		t.Product,
		t.AmountExclTax.StringFixed(2),
		t.TaxRate.StringFixed(2),
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
	t.Status = record[0]
	var err error
	t.Date, err = t.parseDateFromCSV(record[1])
	if err != nil {
		return fmt.Errorf("failed to parse date: %w", err)
	}
	t.ValueDate, err = t.parseDateFromCSV(record[2])
	if err != nil {
		return fmt.Errorf("failed to parse value date: %w", err)
	}
	t.Name = record[3]
	t.PartyName = record[4]
	t.PartyIBAN = record[5]
	t.Description = record[6]
	t.RemittanceInfo = record[7]
	t.Amount, err = decimal.NewFromString(record[8])
	if err != nil {
		return err
	}
	t.CreditDebit = record[9]
	t.Currency = record[10]
	t.Product = record[11]
	t.AmountExclTax, err = decimal.NewFromString(record[12])
	if err != nil {
		return err
	}
	t.TaxRate, err = decimal.NewFromString(record[13])
	if err != nil {
		return err
	}
	t.Investment = record[14]
	t.Number = record[15]
	t.Category = record[16]
	t.Type = record[17]
	t.Fund = record[18]
	var numberOfShares int
	numberOfShares, err = strconv.Atoi(record[19])
	if err != nil {
		return err
	}
	t.NumberOfShares = numberOfShares
	// Fees includes stamp duty
	t.Fees, err = decimal.NewFromString(record[20])
	if err != nil {
		return err
	}
	t.IBAN = record[21]
	t.EntryReference = record[22]
	t.Reference = record[23]
	t.AccountServicer = record[24]
	t.BankTxCode = record[25]
	t.OriginalCurrency = record[26]
	t.OriginalAmount, err = decimal.NewFromString(record[27])
	if err != nil {
		return err
	}
	t.ExchangeRate, err = decimal.NewFromString(record[28])
	if err != nil {
		return err
	}
	return nil
}

// formatDateForCSV formats a time.Time as DD.MM.YYYY for CSV output
// Returns empty string for zero time
func (t *Transaction) formatDateForCSV(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	return date.Format(DateFormatCSV)
}

// parseDateFromCSV parses a date string from CSV format (DD.MM.YYYY) to time.Time
// Returns zero time for empty strings
func (t *Transaction) parseDateFromCSV(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse(DateFormatCSV, dateStr)
}

// NewTransactionFromBuilder creates a Transaction using the builder pattern
// This provides a more readable way to construct transactions
func NewTransactionFromBuilder() *TransactionBuilder {
	return NewTransactionBuilder()
}
