package models

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransactionBuilder(t *testing.T) {
	builder := NewTransactionBuilder()
	
	assert.NotNil(t, builder)
	assert.Nil(t, builder.err)
	assert.Equal(t, StatusCompleted, builder.tx.Status)
	assert.Equal(t, "CHF", builder.tx.Currency)
	assert.Equal(t, CategoryUncategorized, builder.tx.Category)
	assert.Equal(t, TransactionTypeDebit, builder.tx.CreditDebit)
	assert.True(t, builder.tx.DebitFlag)
	assert.True(t, builder.tx.Amount.IsZero())
}

func TestTransactionBuilder_WithDate(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		expectError bool
		expectedDate string
	}{
		{
			name:         "valid DD.MM.YYYY format",
			dateStr:      "15.01.2025",
			expectError:  false,
			expectedDate: "15.01.2025",
		},
		{
			name:         "valid YYYY-MM-DD format",
			dateStr:      "2025-01-15",
			expectError:  false,
			expectedDate: "15.01.2025",
		},
		{
			name:        "empty date",
			dateStr:     "",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewTransactionBuilder().WithDate(tt.dateStr)
			
			if tt.expectError {
				assert.NotNil(t, builder.err)
			} else {
				assert.Nil(t, builder.err)
				assert.Equal(t, tt.expectedDate, builder.tx.Date)
			}
		})
	}
}

func TestTransactionBuilder_WithDateFromTime(t *testing.T) {
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	
	builder := NewTransactionBuilder().WithDateFromTime(date)
	
	assert.Nil(t, builder.err)
	assert.Equal(t, "15.01.2025", builder.tx.Date)
}

func TestTransactionBuilder_WithDateFromTime_ZeroTime(t *testing.T) {
	builder := NewTransactionBuilder().WithDateFromTime(time.Time{})
	
	assert.NotNil(t, builder.err)
	assert.Contains(t, builder.err.Error(), "date cannot be zero")
}

func TestTransactionBuilder_WithAmount(t *testing.T) {
	amount := decimal.NewFromFloat(100.50)
	currency := "EUR"
	
	builder := NewTransactionBuilder().WithAmount(amount, currency)
	
	assert.Nil(t, builder.err)
	assert.True(t, builder.tx.Amount.Equal(amount))
	assert.Equal(t, currency, builder.tx.Currency)
}

func TestTransactionBuilder_WithAmountFromString(t *testing.T) {
	tests := []struct {
		name           string
		amountStr      string
		currency       string
		expectedAmount decimal.Decimal
	}{
		{
			name:           "simple amount",
			amountStr:      "100.50",
			currency:       "CHF",
			expectedAmount: decimal.NewFromFloat(100.50),
		},
		{
			name:           "amount with currency symbol",
			amountStr:      "CHF 1234.56",
			currency:       "CHF",
			expectedAmount: decimal.NewFromFloat(1234.56),
		},
		{
			name:           "negative amount",
			amountStr:      "-50.25",
			currency:       "EUR",
			expectedAmount: decimal.NewFromFloat(-50.25),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewTransactionBuilder().WithAmountFromString(tt.amountStr, tt.currency)
			
			assert.Nil(t, builder.err)
			assert.True(t, builder.tx.Amount.Equal(tt.expectedAmount))
			assert.Equal(t, tt.currency, builder.tx.Currency)
		})
	}
}

func TestTransactionBuilder_WithPayer(t *testing.T) {
	name := "John Doe"
	iban := "CH1234567890"
	
	builder := NewTransactionBuilder().WithPayer(name, iban)
	
	assert.Nil(t, builder.err)
	assert.Equal(t, name, builder.tx.Payer)
	assert.Equal(t, iban, builder.tx.PartyIBAN)
}

func TestTransactionBuilder_WithPayee(t *testing.T) {
	name := "Acme Corp"
	iban := "CH0987654321"
	
	builder := NewTransactionBuilder().WithPayee(name, iban)
	
	assert.Nil(t, builder.err)
	assert.Equal(t, name, builder.tx.Payee)
	assert.Equal(t, name, builder.tx.Recipient) // Should also set recipient
	assert.Equal(t, iban, builder.tx.PartyIBAN)
}

func TestTransactionBuilder_AsDebit(t *testing.T) {
	builder := NewTransactionBuilder().AsDebit()
	
	assert.Nil(t, builder.err)
	assert.Equal(t, TransactionTypeDebit, builder.tx.CreditDebit)
	assert.True(t, builder.tx.DebitFlag)
}

func TestTransactionBuilder_AsCredit(t *testing.T) {
	builder := NewTransactionBuilder().AsCredit()
	
	assert.Nil(t, builder.err)
	assert.Equal(t, TransactionTypeCredit, builder.tx.CreditDebit)
	assert.False(t, builder.tx.DebitFlag)
}

func TestTransactionBuilder_FluentChaining(t *testing.T) {
	amount := decimal.NewFromFloat(250.75)
	
	tx, err := NewTransactionBuilder().
		WithDate("15.01.2025").
		WithAmount(amount, "CHF").
		WithDescription("Test transaction").
		WithPayer("John Doe", "CH1234567890").
		WithPayee("Acme Corp", "CH0987654321").
		WithCategory("Shopping").
		AsDebit().
		Build()
	
	require.NoError(t, err)
	assert.Equal(t, "15.01.2025", tx.Date)
	assert.True(t, tx.Amount.Equal(amount))
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "Test transaction", tx.Description)
	assert.Equal(t, "John Doe", tx.Payer)
	assert.Equal(t, "Acme Corp", tx.Payee)
	assert.Equal(t, "Shopping", tx.Category)
	assert.Equal(t, TransactionTypeDebit, tx.CreditDebit)
	assert.True(t, tx.DebitFlag)
}

func TestTransactionBuilder_Build_Validation(t *testing.T) {
	tests := []struct {
		name        string
		setupBuilder func() *TransactionBuilder
		expectError bool
		errorContains string
	}{
		{
			name: "missing date",
			setupBuilder: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithAmountFromFloat(100.0, "CHF")
			},
			expectError:   true,
			errorContains: "date is required",
		},
		{
			name: "zero amount and fees",
			setupBuilder: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("15.01.2025")
			},
			expectError:   true,
			errorContains: "amount or fees must be non-zero",
		},
		{
			name: "valid with amount",
			setupBuilder: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("15.01.2025").
					WithAmountFromFloat(100.0, "CHF")
			},
			expectError: false,
		},
		{
			name: "valid with fees only",
			setupBuilder: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("15.01.2025").
					WithFeesFromFloat(5.0)
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupBuilder()
			tx, err := builder.Build()
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tx.Date)
			}
		})
	}
}

func TestTransactionBuilder_PopulateDerivedFields(t *testing.T) {
	tx, err := NewTransactionBuilder().
		WithDate("15.01.2025").
		WithAmountFromFloat(100.0, "CHF").
		WithPayer("John Doe", "CH1234567890").
		WithPayee("Acme Corp", "CH0987654321").
		AsDebit().
		Build()
	
	require.NoError(t, err)
	
	// Check derived fields
	assert.Equal(t, "Acme Corp", tx.Name) // For debit, Name should be Payee
	assert.Equal(t, "Acme Corp", tx.Recipient)
	assert.Equal(t, "15.01.2025", tx.ValueDate) // Should default to Date
	assert.NotEmpty(t, tx.Number) // Should generate a UUID
	assert.True(t, tx.Debit.Equal(decimal.NewFromFloat(100.0)))
	assert.True(t, tx.Credit.IsZero())
}

func TestTransactionBuilder_ErrorPropagation(t *testing.T) {
	// Create a builder with an error
	builder := NewTransactionBuilder().WithDate("") // This should cause an error
	
	// Subsequent calls should not execute
	tx, err := builder.
		WithAmountFromFloat(100.0, "CHF").
		WithDescription("Test").
		Build()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "date cannot be empty")
	assert.Equal(t, Transaction{}, tx)
}

func TestTransactionBuilder_WithInvestmentFields(t *testing.T) {
	tx, err := NewTransactionBuilder().
		WithDate("15.01.2025").
		WithAmountFromFloat(1000.0, "CHF").
		WithInvestment("Buy").
		WithNumberOfShares(10).
		WithFeesFromFloat(5.0).
		Build()
	
	require.NoError(t, err)
	assert.Equal(t, "Buy", tx.Investment)
	assert.Equal(t, 10, tx.NumberOfShares)
	assert.True(t, tx.Fees.Equal(decimal.NewFromFloat(5.0)))
}

func TestTransactionBuilder_WithTaxInfo(t *testing.T) {
	amountExclTax := decimal.NewFromFloat(100.0)
	amountTax := decimal.NewFromFloat(7.7)
	taxRate := decimal.NewFromFloat(7.7)
	
	tx, err := NewTransactionBuilder().
		WithDate("15.01.2025").
		WithAmountFromFloat(107.7, "CHF").
		WithTaxInfo(amountExclTax, amountTax, taxRate).
		Build()
	
	require.NoError(t, err)
	assert.True(t, tx.AmountExclTax.Equal(amountExclTax))
	assert.True(t, tx.AmountTax.Equal(amountTax))
	assert.True(t, tx.TaxRate.Equal(taxRate))
}

func TestTransactionBuilder_WithOriginalAmount(t *testing.T) {
	originalAmount := decimal.NewFromFloat(100.0)
	exchangeRate := decimal.NewFromFloat(1.1)
	
	tx, err := NewTransactionBuilder().
		WithDate("15.01.2025").
		WithAmountFromFloat(110.0, "CHF").
		WithOriginalAmount(originalAmount, "EUR").
		WithExchangeRate(exchangeRate).
		Build()
	
	require.NoError(t, err)
	assert.True(t, tx.OriginalAmount.Equal(originalAmount))
	assert.Equal(t, "EUR", tx.OriginalCurrency)
	assert.True(t, tx.ExchangeRate.Equal(exchangeRate))
}

func TestTransactionBuilder_Clone(t *testing.T) {
	original := NewTransactionBuilder().
		WithDate("15.01.2025").
		WithAmountFromFloat(100.0, "CHF").
		WithDescription("Original")
	
	cloned := original.Clone().
		WithDescription("Cloned").
		WithAmountFromFloat(200.0, "EUR")
	
	// Original should be unchanged
	assert.Equal(t, "Original", original.tx.Description)
	assert.True(t, original.tx.Amount.Equal(decimal.NewFromFloat(100.0)))
	assert.Equal(t, "CHF", original.tx.Currency)
	
	// Clone should have new values
	assert.Equal(t, "Cloned", cloned.tx.Description)
	assert.True(t, cloned.tx.Amount.Equal(decimal.NewFromFloat(200.0)))
	assert.Equal(t, "EUR", cloned.tx.Currency)
}

func TestTransactionBuilder_Reset(t *testing.T) {
	builder := NewTransactionBuilder().
		WithDate("15.01.2025").
		WithAmountFromFloat(100.0, "CHF")
	
	reset := builder.Reset()
	
	// Should be a new builder with default values
	assert.NotEqual(t, builder, reset)
	assert.Equal(t, "", reset.tx.Date)
	assert.True(t, reset.tx.Amount.IsZero())
	assert.Equal(t, "CHF", reset.tx.Currency) // Default currency
}

func TestTransactionBuilder_ComplexTransaction(t *testing.T) {
	// Test building a complex transaction with all fields
	tx, err := NewTransactionBuilder().
		WithBookkeepingNumber("BK123").
		WithDate("15.01.2025").
		WithValueDate("16.01.2025").
		WithAmountFromFloat(1234.56, "CHF").
		WithDescription("Complex transaction").
		WithRemittanceInfo("Payment for services").
		WithPayer("John Doe", "CH1234567890").
		WithPayee("Acme Corp", "CH0987654321").
		WithStatus(StatusCompleted).
		WithReference("REF123").
		WithEntryReference("ENTRY123").
		WithAccountServicer("BANK123").
		WithBankTxCode(BankCodePOS).
		WithCategory("Services").
		WithType("Payment").
		WithFund("General").
		WithFeesFromFloat(2.50).
		WithOriginalAmount(decimal.NewFromFloat(1000.0), "EUR").
		WithExchangeRate(decimal.NewFromFloat(1.23456)).
		WithTaxInfo(
			decimal.NewFromFloat(1146.86),
			decimal.NewFromFloat(87.70),
			decimal.NewFromFloat(7.7),
		).
		AsCredit().
		Build()
	
	require.NoError(t, err)
	
	// Verify all fields are set correctly
	assert.Equal(t, "BK123", tx.BookkeepingNumber)
	assert.Equal(t, "15.01.2025", tx.Date)
	assert.Equal(t, "16.01.2025", tx.ValueDate)
	assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(1234.56)))
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "Complex transaction", tx.Description)
	assert.Equal(t, "Payment for services", tx.RemittanceInfo)
	assert.Equal(t, "John Doe", tx.Payer)
	assert.Equal(t, "Acme Corp", tx.Payee)
	assert.Equal(t, "Acme Corp", tx.Recipient)
	assert.Equal(t, "CH0987654321", tx.PartyIBAN)
	assert.Equal(t, StatusCompleted, tx.Status)
	assert.Equal(t, "REF123", tx.Reference)
	assert.Equal(t, "ENTRY123", tx.EntryReference)
	assert.Equal(t, "BANK123", tx.AccountServicer)
	assert.Equal(t, BankCodePOS, tx.BankTxCode)
	assert.Equal(t, "Services", tx.Category)
	assert.Equal(t, "Payment", tx.Type)
	assert.Equal(t, "General", tx.Fund)
	assert.True(t, tx.Fees.Equal(decimal.NewFromFloat(2.50)))
	assert.True(t, tx.OriginalAmount.Equal(decimal.NewFromFloat(1000.0)))
	assert.Equal(t, "EUR", tx.OriginalCurrency)
	assert.True(t, tx.ExchangeRate.Equal(decimal.NewFromFloat(1.23456)))
	assert.True(t, tx.AmountExclTax.Equal(decimal.NewFromFloat(1146.86)))
	assert.True(t, tx.AmountTax.Equal(decimal.NewFromFloat(87.70)))
	assert.True(t, tx.TaxRate.Equal(decimal.NewFromFloat(7.7)))
	assert.Equal(t, TransactionTypeCredit, tx.CreditDebit)
	assert.False(t, tx.DebitFlag)
	
	// Verify derived fields
	assert.Equal(t, "John Doe", tx.Name) // For credit, Name should be Payer
	assert.True(t, tx.Credit.Equal(decimal.NewFromFloat(1234.56)))
	assert.True(t, tx.Debit.IsZero())
}