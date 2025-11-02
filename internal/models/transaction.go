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

// GetPayee returns the appropriate party name based on transaction direction for backward compatibility.
// For debit transactions (money going out), returns the payee (who receives the money).
// For credit transactions (money coming in), returns the payer (who sent the money to us).
// This provides a consistent "other party" perspective from the account holder's viewpoint.
//
// Deprecated: This method is provided for backward compatibility during migration.
// For new code, use TransactionBuilder pattern or access Payer/Payee fields directly.
// Consider using GetCounterparty() for similar functionality.
//
// Migration example:
//
//	// Old code:
//	otherParty := tx.GetPayee()
//
//	// New code (direct access):
//	payee := tx.Payee  // if you specifically need the payee
//
//	// New code (counterparty - recommended):
//	otherParty := tx.GetCounterparty()
func (t *Transaction) GetPayee() string {
	if t.IsDebit() {
		// For debit (outgoing), the payee is who receives our money
		return t.Payee
	}
	// For credit (incoming), the "payee" from our perspective is who sent us money
	return t.Payer
}

// GetPayer returns the appropriate party name based on transaction direction for backward compatibility.
// For debit transactions (money going out), returns the payer (account holder who sent the money).
// For credit transactions (money coming in), returns the payee (who received the money from the sender).
// This provides a consistent "account holder" perspective.
//
// Deprecated: This method is provided for backward compatibility during migration.
// For new code, use TransactionBuilder pattern or access Payer/Payee fields directly.
// Consider using GetCounterparty() for the other party in the transaction.
//
// Migration example:
//
//	// Old code:
//	accountHolder := tx.GetPayer()
//
//	// New code (direct access):
//	payer := tx.Payer  // if you specifically need the payer
//
//	// New code (account holder perspective):
//	// For debits: account holder is the payer
//	// For credits: account holder is the payee
func (t *Transaction) GetPayer() string {
	if t.IsDebit() {
		// For debit (outgoing), the payer is the account holder
		return t.Payer
	}
	// For credit (incoming), the "payer" from account holder's perspective is the payee
	return t.Payee
}

// GetAmountAsFloat returns the amount as float64 for backward compatibility.
// Note: This method may lose precision for large amounts or amounts with many decimal places.
//
// Deprecated: Use GetAmountAsDecimal() instead for precise financial calculations.
// For new code, use TransactionBuilder.WithAmount() to set amounts as decimal.Decimal.
// Migration example:
//
//	// Old code:
//	amount := tx.GetAmountAsFloat()
//
//	// New code:
//	amount := tx.GetAmountAsDecimal()
func (t *Transaction) GetAmountAsFloat() float64 {
	f, _ := t.Amount.Float64()
	return f
}

// GetDebitAsFloat returns the debit amount as float64 for backward compatibility
// Deprecated: Use GetDebitAsDecimal() instead for precise calculations
func (t *Transaction) GetDebitAsFloat() float64 {
	f, _ := t.Debit.Float64()
	return f
}

// GetCreditAsFloat returns the credit amount as float64 for backward compatibility
// Deprecated: Use GetCreditAsDecimal() instead for precise calculations
func (t *Transaction) GetCreditAsFloat() float64 {
	f, _ := t.Credit.Float64()
	return f
}

// GetFeesAsFloat returns the fees as float64 for backward compatibility
// Deprecated: Use GetFeesAsDecimal() instead for precise calculations
func (t *Transaction) GetFeesAsFloat() float64 {
	f, _ := t.Fees.Float64()
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
		t.BookkeepingNumber,
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
	var err error
	t.Date, err = t.parseDateFromCSV(record[2])
	if err != nil {
		return fmt.Errorf("failed to parse date: %w", err)
	}
	t.ValueDate, err = t.parseDateFromCSV(record[3])
	if err != nil {
		return fmt.Errorf("failed to parse value date: %w", err)
	}
	t.Name = record[4]
	t.PartyName = record[5]
	t.PartyIBAN = record[6]
	t.Description = record[7]
	t.RemittanceInfo = record[8]
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

// formatDateForCSV formats a time.Time as DD.MM.YYYY for CSV output
// Returns empty string for zero time
func (t *Transaction) formatDateForCSV(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	return date.Format("02.01.2006")
}

// parseDateFromCSV parses a date string from CSV format (DD.MM.YYYY) to time.Time
// Returns zero time for empty strings
func (t *Transaction) parseDateFromCSV(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse("02.01.2006", dateStr)
}

// ToTransactionCore converts the Transaction to a TransactionCore
func (t *Transaction) ToTransactionCore() TransactionCore {
	return TransactionCore{
		ID:          t.Number,
		Date:        t.Date,
		ValueDate:   t.ValueDate,
		Amount:      NewMoney(t.Amount, t.Currency),
		Description: t.Description,
		Status:      t.Status,
		Reference:   t.Reference,
	}
}

// ToTransactionWithParties converts the Transaction to a TransactionWithParties
func (t *Transaction) ToTransactionWithParties() TransactionWithParties {
	direction := DirectionUnknown
	if t.IsDebit() {
		direction = DirectionDebit
	} else if t.IsCredit() {
		direction = DirectionCredit
	}

	// For backward compatibility, both parties get the same IBAN from PartyIBAN
	// This matches the test expectation where PartyIBAN represents the counterparty's IBAN
	return TransactionWithParties{
		TransactionCore: t.ToTransactionCore(),
		Payer:           NewParty(t.Payer, t.PartyIBAN),
		Payee:           NewParty(t.Payee, t.PartyIBAN),
		Direction:       direction,
	}
}

// ToCategorizedTransaction converts the Transaction to a CategorizedTransaction
func (t *Transaction) ToCategorizedTransaction() CategorizedTransaction {
	return CategorizedTransaction{
		TransactionWithParties: t.ToTransactionWithParties(),
		Category:               t.Category,
		Type:                   t.Type,
		Fund:                   t.Fund,
	}
}

// FromTransactionCore populates the Transaction from a TransactionCore
func (t *Transaction) FromTransactionCore(core TransactionCore) {
	t.Number = core.ID
	t.Date = core.Date
	t.ValueDate = core.ValueDate
	t.Amount = core.Amount.Amount
	t.Currency = core.Amount.Currency
	t.Description = core.Description
	t.Status = core.Status
	t.Reference = core.Reference
}

// FromCategorizedTransaction populates the Transaction from a CategorizedTransaction
func (t *Transaction) FromCategorizedTransaction(ct CategorizedTransaction) {
	// Populate from core
	t.FromTransactionCore(ct.TransactionCore)

	// Set party information
	t.Payer = ct.Payer.Name
	t.Payee = ct.Payee.Name

	// Set direction and related fields
	if ct.Direction == DirectionDebit {
		t.CreditDebit = TransactionTypeDebit
		t.DebitFlag = true
		t.PartyIBAN = ct.Payee.IBAN // For debit, PartyIBAN is payee's IBAN
		t.PartyName = ct.Payee.Name // For debit, PartyName is payee's name
	} else if ct.Direction == DirectionCredit {
		t.CreditDebit = TransactionTypeCredit
		t.DebitFlag = false
		t.PartyIBAN = ct.Payer.IBAN // For credit, PartyIBAN is payer's IBAN
		t.PartyName = ct.Payer.Name // For credit, PartyName is payer's name
	}

	// Set categorization
	t.Category = ct.Category
	t.Type = ct.Type
	t.Fund = ct.Fund

	// Populate derived fields
	t.UpdateNameFromParties()
	t.UpdateRecipientFromPayee()
	t.UpdateDebitCreditAmounts()
}

// NewTransactionFromBuilder creates a Transaction using the builder pattern
// This provides a more readable way to construct transactions
func NewTransactionFromBuilder() *TransactionBuilder {
	return NewTransactionBuilder()
}

// ToBuilder converts an existing Transaction to a TransactionBuilder for modification
// Deprecated: This method is provided for backward compatibility during migration
func (t *Transaction) ToBuilder() *TransactionBuilder {
	builder := NewTransactionBuilder()

	// Copy all fields from the transaction to the builder
	builder.tx = *t

	return builder
}

// SetPayerInfo sets both payer name and IBAN for backward compatibility
// Deprecated: Use TransactionBuilder.WithPayer() for new code
func (t *Transaction) SetPayerInfo(name, iban string) {
	t.Payer = name
	if iban != "" {
		t.PartyIBAN = iban
	}
}

// SetPayeeInfo sets both payee name and IBAN for backward compatibility
// Deprecated: Use TransactionBuilder.WithPayee() for new code
func (t *Transaction) SetPayeeInfo(name, iban string) {
	t.Payee = name
	if iban != "" {
		t.PartyIBAN = iban
	}
}

// SetAmountFromFloat sets the amount from a float64 value for backward compatibility
// Deprecated: Use TransactionBuilder.WithAmountFromFloat() or SetAmountFromDecimal() for new code
func (t *Transaction) SetAmountFromFloat(amount float64, currency string) {
	t.Amount = decimal.NewFromFloat(amount)
	t.Currency = currency
}

// GetTransactionDirection returns the transaction direction as TransactionDirection enum
func (t *Transaction) GetTransactionDirection() TransactionDirection {
	if t.IsDebit() {
		return DirectionDebit
	} else if t.IsCredit() {
		return DirectionCredit
	}
	return DirectionUnknown
}
