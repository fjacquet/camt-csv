package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransactionBuilder provides a fluent API for constructing transactions
type TransactionBuilder struct {
	tx  Transaction
	err error
}

// NewTransactionBuilder creates a new TransactionBuilder with default values
func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		tx: Transaction{
			// Set default values
			Status:      StatusCompleted,
			Currency:    "CHF", // Default currency
			Category:    CategoryUncategorized,
			CreditDebit: TransactionTypeDebit, // Default to debit
			DebitFlag:   true,
			Amount:      decimal.Zero,
			Debit:       decimal.Zero,
			Credit:      decimal.Zero,
			Fees:        decimal.Zero,
			AmountExclTax: decimal.Zero,
			AmountTax:   decimal.Zero,
			TaxRate:     decimal.Zero,
			OriginalAmount: decimal.Zero,
			ExchangeRate: decimal.NewFromInt(1),
		},
	}
}

// WithID sets the transaction ID
func (b *TransactionBuilder) WithID(id string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Number = id
	return b
}

// WithBookkeepingNumber sets the bookkeeping number
func (b *TransactionBuilder) WithBookkeepingNumber(number string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.BookkeepingNumber = number
	return b
}

// WithDate sets the transaction date from a string in DD.MM.YYYY format
func (b *TransactionBuilder) WithDate(dateStr string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	if dateStr == "" {
		b.err = errors.New("date cannot be empty")
		return b
	}
	
	// Use the existing FormatDate function to standardize the date
	formattedDate := FormatDate(dateStr)
	b.tx.Date = formattedDate
	return b
}

// WithDateFromTime sets the transaction date from a time.Time
func (b *TransactionBuilder) WithDateFromTime(date time.Time) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	if date.IsZero() {
		b.err = errors.New("date cannot be zero")
		return b
	}
	b.tx.Date = date.Format("02.01.2006")
	return b
}

// WithValueDate sets the value date from a string in DD.MM.YYYY format
func (b *TransactionBuilder) WithValueDate(dateStr string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	if dateStr != "" {
		formattedDate := FormatDate(dateStr)
		b.tx.ValueDate = formattedDate
	}
	return b
}

// WithValueDateFromTime sets the value date from a time.Time
func (b *TransactionBuilder) WithValueDateFromTime(valueDate time.Time) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	if !valueDate.IsZero() {
		b.tx.ValueDate = valueDate.Format("02.01.2006")
	}
	return b
}

// WithAmount sets the transaction amount and currency
func (b *TransactionBuilder) WithAmount(amount decimal.Decimal, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Amount = amount
	if currency != "" {
		b.tx.Currency = currency
	}
	return b
}

// WithAmountFromFloat sets the transaction amount from a float64 value
func (b *TransactionBuilder) WithAmountFromFloat(amount float64, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Amount = decimal.NewFromFloat(amount)
	if currency != "" {
		b.tx.Currency = currency
	}
	return b
}

// WithAmountFromString sets the transaction amount from a string
func (b *TransactionBuilder) WithAmountFromString(amountStr, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	amount := ParseAmount(amountStr)
	b.tx.Amount = amount
	if currency != "" {
		b.tx.Currency = currency
	}
	return b
}

// WithCurrency sets the currency
func (b *TransactionBuilder) WithCurrency(currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Currency = currency
	return b
}

// WithDescription sets the transaction description
func (b *TransactionBuilder) WithDescription(description string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Description = description
	return b
}

// WithRemittanceInfo sets the remittance information
func (b *TransactionBuilder) WithRemittanceInfo(info string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.RemittanceInfo = info
	return b
}

// WithPayer sets the payer information
func (b *TransactionBuilder) WithPayer(name, iban string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Payer = name
	if iban != "" {
		b.tx.PartyIBAN = iban
	}
	return b
}

// WithPayee sets the payee information
func (b *TransactionBuilder) WithPayee(name, iban string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Payee = name
	b.tx.Recipient = name // Set recipient for compatibility
	if iban != "" {
		b.tx.PartyIBAN = iban
	}
	return b
}

// WithPartyName sets the party name (generic counterparty)
func (b *TransactionBuilder) WithPartyName(name string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.PartyName = name
	return b
}

// WithPartyIBAN sets the party IBAN
func (b *TransactionBuilder) WithPartyIBAN(iban string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.PartyIBAN = iban
	return b
}

// WithStatus sets the transaction status
func (b *TransactionBuilder) WithStatus(status string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Status = status
	return b
}

// WithReference sets the transaction reference
func (b *TransactionBuilder) WithReference(reference string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Reference = reference
	return b
}

// WithEntryReference sets the entry reference
func (b *TransactionBuilder) WithEntryReference(reference string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.EntryReference = reference
	return b
}

// WithAccountServicer sets the account servicer
func (b *TransactionBuilder) WithAccountServicer(servicer string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.AccountServicer = servicer
	return b
}

// WithBankTxCode sets the bank transaction code
func (b *TransactionBuilder) WithBankTxCode(code string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.BankTxCode = code
	return b
}

// WithCategory sets the transaction category
func (b *TransactionBuilder) WithCategory(category string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Category = category
	return b
}

// WithType sets the transaction type
func (b *TransactionBuilder) WithType(transactionType string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Type = transactionType
	return b
}

// WithFund sets the fund
func (b *TransactionBuilder) WithFund(fund string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Fund = fund
	return b
}

// WithInvestment sets the investment type
func (b *TransactionBuilder) WithInvestment(investment string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Investment = investment
	return b
}

// WithNumberOfShares sets the number of shares for investment transactions
func (b *TransactionBuilder) WithNumberOfShares(shares int) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.NumberOfShares = shares
	return b
}

// WithFees sets the transaction fees
func (b *TransactionBuilder) WithFees(fees decimal.Decimal) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Fees = fees
	return b
}

// WithFeesFromFloat sets the transaction fees from a float64
func (b *TransactionBuilder) WithFeesFromFloat(fees float64) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Fees = decimal.NewFromFloat(fees)
	return b
}

// WithOriginalAmount sets the original amount and currency for foreign exchange transactions
func (b *TransactionBuilder) WithOriginalAmount(amount decimal.Decimal, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.OriginalAmount = amount
	b.tx.OriginalCurrency = currency
	return b
}

// WithExchangeRate sets the exchange rate
func (b *TransactionBuilder) WithExchangeRate(rate decimal.Decimal) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.ExchangeRate = rate
	return b
}

// WithTaxInfo sets tax-related information
func (b *TransactionBuilder) WithTaxInfo(amountExclTax, amountTax, taxRate decimal.Decimal) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.AmountExclTax = amountExclTax
	b.tx.AmountTax = amountTax
	b.tx.TaxRate = taxRate
	return b
}

// AsDebit sets the transaction as a debit (outgoing money)
func (b *TransactionBuilder) AsDebit() *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.CreditDebit = TransactionTypeDebit
	b.tx.DebitFlag = true
	return b
}

// AsCredit sets the transaction as a credit (incoming money)
func (b *TransactionBuilder) AsCredit() *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.CreditDebit = TransactionTypeCredit
	b.tx.DebitFlag = false
	return b
}

// Build validates the transaction and returns the final Transaction
func (b *TransactionBuilder) Build() (Transaction, error) {
	if b.err != nil {
		return Transaction{}, fmt.Errorf("builder error: %w", b.err)
	}
	
	// Validate required fields
	if b.tx.Date == "" {
		return Transaction{}, errors.New("date is required")
	}
	
	if b.tx.Amount.IsZero() && b.tx.Fees.IsZero() {
		return Transaction{}, errors.New("amount or fees must be non-zero")
	}
	
	// Populate derived fields
	b.populateDerivedFields()
	
	return b.tx, nil
}

// populateDerivedFields sets derived fields based on the transaction data
func (b *TransactionBuilder) populateDerivedFields() {
	// Set Name field based on transaction direction and parties
	b.tx.UpdateNameFromParties()
	
	// Set Recipient from Payee for compatibility
	b.tx.UpdateRecipientFromPayee()
	
	// Update debit/credit amounts based on the main amount and direction
	b.tx.UpdateDebitCreditAmounts()
	
	// Update investment type from legacy field if needed
	b.tx.UpdateInvestmentTypeFromLegacyField()
	
	// Set PartyName if not already set
	if b.tx.PartyName == "" {
		b.tx.PartyName = b.tx.GetPartyName()
	}
	
	// Set value date to transaction date if not set
	if b.tx.ValueDate == "" {
		b.tx.ValueDate = b.tx.Date
	}
	
	// Generate a unique number if not set
	if b.tx.Number == "" {
		b.tx.Number = uuid.New().String()
	}
}

// Reset clears the builder state and returns a new builder
func (b *TransactionBuilder) Reset() *TransactionBuilder {
	return NewTransactionBuilder()
}

// Clone creates a copy of the current builder state
func (b *TransactionBuilder) Clone() *TransactionBuilder {
	newBuilder := &TransactionBuilder{
		tx:  b.tx, // This creates a copy of the struct
		err: b.err,
	}
	return newBuilder
}