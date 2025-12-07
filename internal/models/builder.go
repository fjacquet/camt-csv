package models

import (
	"errors"
	"fmt"
	"time"

	"fjacquet/camt-csv/internal/dateutils"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransactionBuilder provides fluent API for transaction construction
type TransactionBuilder struct {
	tx  Transaction
	err error
}

// NewTransactionBuilder creates a new TransactionBuilder with sensible defaults
func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{
		tx: Transaction{
			Number:         uuid.New().String(),   // Generate unique ID
			Status:         StatusCompleted,       // Default status
			Currency:       "CHF",                 // Default currency
			Category:       CategoryUncategorized, // Default category
			Amount:         decimal.Zero,          // Default amount
			Debit:          decimal.Zero,          // Default debit
			Credit:         decimal.Zero,          // Default credit
			Fees:           decimal.Zero,          // Default fees
			AmountExclTax:  decimal.Zero,          // Default amount excluding tax
			AmountTax:      decimal.Zero,          // Default tax amount
			TaxRate:        decimal.Zero,          // Default tax rate
			OriginalAmount: decimal.Zero,          // Default original amount
			ExchangeRate:   decimal.Zero,          // Default exchange rate
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

// WithStatus sets the transaction status
func (b *TransactionBuilder) WithStatus(status string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Status = status
	return b
}

// WithDate sets the transaction date from a string in YYYY-MM-DD format
func (b *TransactionBuilder) WithDate(dateStr string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	date, err := time.Parse(dateutils.DateLayoutISO, dateStr)
	if err != nil {
		b.err = fmt.Errorf("invalid date format '%s': %w", dateStr, err)
		return b
	}
	b.tx.Date = date
	return b
}

// WithDatetime sets the transaction date from a time.Time
func (b *TransactionBuilder) WithDatetime(date time.Time) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Date = date
	return b
}

// WithValueDate sets the value date from a string in YYYY-MM-DD format
func (b *TransactionBuilder) WithValueDate(dateStr string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	date, err := time.Parse(dateutils.DateLayoutISO, dateStr)
	if err != nil {
		b.err = fmt.Errorf("invalid value date format '%s': %w", dateStr, err)
		return b
	}
	b.tx.ValueDate = date
	return b
}

// WithValueDatetime sets the value date from a time.Time
func (b *TransactionBuilder) WithValueDatetime(date time.Time) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.ValueDate = date
	return b
}

// WithDateFromDatetime sets the transaction date from a datetime string (YYYY-MM-DD HH:MM:SS)
func (b *TransactionBuilder) WithDateFromDatetime(datetimeStr string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	// Try parsing as datetime first
	date, err := time.Parse(dateutils.DateLayoutFull, datetimeStr)
	if err != nil {
		// If that fails, try parsing as date only
		date, err = time.Parse(dateutils.DateLayoutISO, datetimeStr)
		if err != nil {
			b.err = fmt.Errorf("invalid datetime format '%s': %w", datetimeStr, err)
			return b
		}
	}
	b.tx.Date = date
	return b
}

// WithValueDateFromDatetime sets the value date from a datetime string (YYYY-MM-DD HH:MM:SS)
func (b *TransactionBuilder) WithValueDateFromDatetime(datetimeStr string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	// Try parsing as datetime first
	date, err := time.Parse(dateutils.DateLayoutFull, datetimeStr)
	if err != nil {
		// If that fails, try parsing as date only
		date, err = time.Parse(dateutils.DateLayoutISO, datetimeStr)
		if err != nil {
			b.err = fmt.Errorf("invalid datetime format '%s': %w", datetimeStr, err)
			return b
		}
	}
	b.tx.ValueDate = date
	return b
}

// WithAmount sets the transaction amount and currency
func (b *TransactionBuilder) WithAmount(amount decimal.Decimal, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Amount = amount
	b.tx.Currency = currency
	return b
}

// WithAmountFromFloat sets the transaction amount from float64 and currency
func (b *TransactionBuilder) WithAmountFromFloat(amount float64, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Amount = decimal.NewFromFloat(amount)
	b.tx.Currency = currency
	return b
}

// WithAmountFromString sets the transaction amount from string and currency
func (b *TransactionBuilder) WithAmountFromString(amountStr, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		b.err = fmt.Errorf("invalid amount string '%s': %w", amountStr, err)
		return b
	}
	b.tx.Amount = amount
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

// WithPayer sets the payer name and IBAN
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

// WithPayee sets the payee name and IBAN
func (b *TransactionBuilder) WithPayee(name, iban string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Payee = name
	if iban != "" {
		b.tx.PartyIBAN = iban
	}
	return b
}

// WithPartyName sets the party name (counterparty)
func (b *TransactionBuilder) WithPartyName(name string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.PartyName = name
	return b
}

// WithPartyIBAN sets the party IBAN (counterparty)
func (b *TransactionBuilder) WithPartyIBAN(iban string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.PartyIBAN = iban
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

// WithFund sets the fund name
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

// WithFeesFromFloat sets the transaction fees from float64
func (b *TransactionBuilder) WithFeesFromFloat(fees float64) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.Fees = decimal.NewFromFloat(fees)
	return b
}

// WithIBAN sets the IBAN
func (b *TransactionBuilder) WithIBAN(iban string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.IBAN = iban
	return b
}

// WithOriginalAmount sets the original amount and currency for foreign currency transactions
func (b *TransactionBuilder) WithOriginalAmount(amount decimal.Decimal, currency string) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.OriginalAmount = amount
	b.tx.OriginalCurrency = currency
	return b
}

// WithExchangeRate sets the exchange rate for currency conversion
func (b *TransactionBuilder) WithExchangeRate(rate decimal.Decimal) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.ExchangeRate = rate
	return b
}

// WithTax sets the tax-related fields
func (b *TransactionBuilder) WithTax(amountExclTax, amountTax, taxRate decimal.Decimal) *TransactionBuilder {
	if b.err != nil {
		return b
	}
	b.tx.AmountExclTax = amountExclTax
	b.tx.AmountTax = amountTax
	b.tx.TaxRate = taxRate
	return b
}

// WithTaxInfo sets the tax-related fields (alias for WithTax for backward compatibility)
func (b *TransactionBuilder) WithTaxInfo(amountExclTax, amountTax, taxRate decimal.Decimal) *TransactionBuilder {
	return b.WithTax(amountExclTax, amountTax, taxRate)
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

// Build validates and constructs the final Transaction
func (b *TransactionBuilder) Build() (Transaction, error) {
	if b.err != nil {
		return Transaction{}, b.err
	}

	// Validate required fields
	if b.tx.Date.IsZero() {
		return Transaction{}, errors.New("transaction date is required")
	}

	if b.tx.Amount.IsZero() && b.tx.Debit.IsZero() && b.tx.Credit.IsZero() {
		return Transaction{}, errors.New("transaction amount is required")
	}

	if b.tx.Currency == "" {
		return Transaction{}, errors.New("currency is required")
	}

	// Populate derived fields
	b.populateDerivedFields()

	return b.tx, nil
}

// populateDerivedFields sets derived fields based on the transaction data
func (b *TransactionBuilder) populateDerivedFields() {
	// Set value date to transaction date if not specified
	if b.tx.ValueDate.IsZero() {
		b.tx.ValueDate = b.tx.Date
	}

	// Determine transaction direction if not explicitly set
	if b.tx.CreditDebit == "" {
		if b.tx.Amount.IsNegative() {
			b.tx.CreditDebit = TransactionTypeDebit
			b.tx.DebitFlag = true
		} else {
			b.tx.CreditDebit = TransactionTypeCredit
			b.tx.DebitFlag = false
		}
	}

	// Update name from parties
	b.tx.UpdateNameFromParties()

	// Update recipient from payee
	b.tx.UpdateRecipientFromPayee()

	// Update debit/credit amounts
	b.tx.UpdateDebitCreditAmounts()

	// Update investment type from legacy field
	b.tx.UpdateInvestmentTypeFromLegacyField()

	// Set PartyName if not already set
	if b.tx.PartyName == "" {
		b.tx.PartyName = b.tx.GetPartyName()
	}
}

// Reset clears the builder state and returns a new builder with defaults
func (b *TransactionBuilder) Reset() *TransactionBuilder {
	return NewTransactionBuilder()
}

// Clone creates a copy of the current builder state
func (b *TransactionBuilder) Clone() *TransactionBuilder {
	return &TransactionBuilder{
		tx:  b.tx,
		err: b.err,
	}
}
