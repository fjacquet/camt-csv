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
	assert.NoError(t, builder.err)
	assert.NotEmpty(t, builder.tx.Number) // Should have generated UUID
	assert.Equal(t, StatusCompleted, builder.tx.Status)
	assert.Equal(t, "CHF", builder.tx.Currency)
	assert.Equal(t, CategoryUncategorized, builder.tx.Category)
	assert.True(t, builder.tx.Amount.IsZero())
}

func TestTransactionBuilder_WithDate(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		expectError bool
	}{
		{
			name:        "valid date",
			dateStr:     "2025-01-15",
			expectError: false,
		},
		{
			name:        "invalid date format",
			dateStr:     "15.01.2025",
			expectError: true,
		},
		{
			name:        "invalid date",
			dateStr:     "2025-13-45",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewTransactionBuilder().WithDate(tt.dateStr)
			
			if tt.expectError {
				assert.Error(t, builder.err)
			} else {
				assert.NoError(t, builder.err)
				expectedDate, _ := time.Parse("2006-01-02", tt.dateStr)
				assert.Equal(t, expectedDate, builder.tx.Date)
			}
		})
	}
}

func TestTransactionBuilder_WithAmount(t *testing.T) {
	amount := decimal.NewFromFloat(100.50)
	currency := "EUR"
	
	builder := NewTransactionBuilder().WithAmount(amount, currency)
	
	assert.NoError(t, builder.err)
	assert.Equal(t, amount, builder.tx.Amount)
	assert.Equal(t, currency, builder.tx.Currency)
}

func TestTransactionBuilder_WithAmountFromString(t *testing.T) {
	tests := []struct {
		name        string
		amountStr   string
		currency    string
		expectError bool
		expected    decimal.Decimal
	}{
		{
			name:        "valid amount",
			amountStr:   "100.50",
			currency:    "CHF",
			expectError: false,
			expected:    decimal.NewFromFloat(100.50),
		},
		{
			name:        "invalid amount",
			amountStr:   "invalid",
			currency:    "CHF",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewTransactionBuilder().WithAmountFromString(tt.amountStr, tt.currency)
			
			if tt.expectError {
				assert.Error(t, builder.err)
			} else {
				assert.NoError(t, builder.err)
				assert.True(t, tt.expected.Equal(builder.tx.Amount))
				assert.Equal(t, tt.currency, builder.tx.Currency)
			}
		})
	}
}

func TestTransactionBuilder_WithPayer(t *testing.T) {
	name := "John Doe"
	iban := "CH1234567890"
	
	builder := NewTransactionBuilder().WithPayer(name, iban)
	
	assert.NoError(t, builder.err)
	assert.Equal(t, name, builder.tx.Payer)
	assert.Equal(t, iban, builder.tx.PartyIBAN)
}

func TestTransactionBuilder_WithPayee(t *testing.T) {
	name := "Acme Corp"
	iban := "CH0987654321"
	
	builder := NewTransactionBuilder().WithPayee(name, iban)
	
	assert.NoError(t, builder.err)
	assert.Equal(t, name, builder.tx.Payee)
	assert.Equal(t, iban, builder.tx.PartyIBAN)
}

func TestTransactionBuilder_AsDebit(t *testing.T) {
	builder := NewTransactionBuilder().AsDebit()
	
	assert.NoError(t, builder.err)
	assert.Equal(t, TransactionTypeDebit, builder.tx.CreditDebit)
	assert.True(t, builder.tx.DebitFlag)
}

func TestTransactionBuilder_AsCredit(t *testing.T) {
	builder := NewTransactionBuilder().AsCredit()
	
	assert.NoError(t, builder.err)
	assert.Equal(t, TransactionTypeCredit, builder.tx.CreditDebit)
	assert.False(t, builder.tx.DebitFlag)
}

func TestTransactionBuilder_FluentAPI(t *testing.T) {
	// Test that methods can be chained
	tx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(100.50), "CHF").
		WithPayer("John Doe", "CH1234567890").
		WithPayee("Acme Corp", "CH0987654321").
		WithDescription("Test transaction").
		WithCategory(CategoryShopping).
		AsDebit().
		Build()
	
	require.NoError(t, err)
	
	expectedDate, _ := time.Parse("2006-01-02", "2025-01-15")
	assert.Equal(t, expectedDate, tx.Date)
	assert.True(t, decimal.NewFromFloat(100.50).Equal(tx.Amount))
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "John Doe", tx.Payer)
	assert.Equal(t, "Acme Corp", tx.Payee)
	assert.Equal(t, "Test transaction", tx.Description)
	assert.Equal(t, CategoryShopping, tx.Category)
	assert.Equal(t, TransactionTypeDebit, tx.CreditDebit)
	assert.True(t, tx.DebitFlag)
}

func TestTransactionBuilder_Build_Validation(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() *TransactionBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing date",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithAmount(decimal.NewFromFloat(100), "CHF")
			},
			expectError: true,
			errorMsg:    "transaction date is required",
		},
		{
			name: "missing amount",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15")
			},
			expectError: true,
			errorMsg:    "transaction amount is required",
		},
		{
			name: "missing currency",
			setupFunc: func() *TransactionBuilder {
				builder := NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.NewFromFloat(100), "CHF")
				builder.tx.Currency = "" // Clear the currency
				return builder
			},
			expectError: true,
			errorMsg:    "currency is required",
		},
		{
			name: "valid transaction",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.NewFromFloat(100), "CHF")
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupFunc()
			tx, err := builder.Build()
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, Transaction{}, tx)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, Transaction{}, tx)
			}
		})
	}
}

func TestTransactionBuilder_PopulateDerivedFields(t *testing.T) {
	tx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(-100.50), "CHF").
		WithPayer("John Doe", "CH1234567890").
		WithPayee("Acme Corp", "CH0987654321").
		Build()
	
	require.NoError(t, err)
	
	// Should set value date to transaction date if not specified
	assert.Equal(t, tx.Date, tx.ValueDate)
	
	// Should determine direction from negative amount
	assert.Equal(t, TransactionTypeDebit, tx.CreditDebit)
	assert.True(t, tx.DebitFlag)
	
	// Should populate debit/credit amounts
	assert.True(t, decimal.NewFromFloat(-100.50).Equal(tx.Debit))
	assert.True(t, decimal.Zero.Equal(tx.Credit))
	
	// Should set Name from Payee for debit transaction
	assert.Equal(t, "Acme Corp", tx.Name)
	
	// Should set Recipient from Payee
	assert.Equal(t, "Acme Corp", tx.Recipient)
	
	// Should set PartyName
	assert.Equal(t, "Acme Corp", tx.PartyName)
}

func TestTransactionBuilder_ErrorPropagation(t *testing.T) {
	// Test that errors are propagated through the chain
	tx, err := NewTransactionBuilder().
		WithDate("invalid-date").
		WithAmount(decimal.NewFromFloat(100), "CHF").
		Build()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
	assert.Equal(t, Transaction{}, tx)
}

func TestTransactionBuilder_WithInvestmentFields(t *testing.T) {
	tx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(1000), "CHF").
		WithInvestment("Buy").
		WithNumberOfShares(10).
		WithFees(decimal.NewFromFloat(5.50)).
		Build()
	
	require.NoError(t, err)
	assert.Equal(t, "Buy", tx.Investment)
	assert.Equal(t, 10, tx.NumberOfShares)
	assert.True(t, decimal.NewFromFloat(5.50).Equal(tx.Fees))
}

func TestTransactionBuilder_WithTax(t *testing.T) {
	amountExclTax := decimal.NewFromFloat(100)
	amountTax := decimal.NewFromFloat(7.7)
	taxRate := decimal.NewFromFloat(7.7)
	
	tx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(107.7), "CHF").
		WithTax(amountExclTax, amountTax, taxRate).
		Build()
	
	require.NoError(t, err)
	assert.True(t, amountExclTax.Equal(tx.AmountExclTax))
	assert.True(t, amountTax.Equal(tx.AmountTax))
	assert.True(t, taxRate.Equal(tx.TaxRate))
}

func TestTransactionBuilder_WithOriginalAmount(t *testing.T) {
	originalAmount := decimal.NewFromFloat(100)
	originalCurrency := "USD"
	exchangeRate := decimal.NewFromFloat(0.92)
	
	tx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(92), "CHF").
		WithOriginalAmount(originalAmount, originalCurrency).
		WithExchangeRate(exchangeRate).
		Build()
	
	require.NoError(t, err)
	assert.True(t, originalAmount.Equal(tx.OriginalAmount))
	assert.Equal(t, originalCurrency, tx.OriginalCurrency)
	assert.True(t, exchangeRate.Equal(tx.ExchangeRate))
}

func TestTransactionBuilder_Clone(t *testing.T) {
	original := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(100), "CHF")
	
	cloned := original.Clone()
	
	// Modify the clone
	cloned.WithDescription("Modified description")
	
	// Original should be unchanged
	assert.Equal(t, "", original.tx.Description)
	assert.Equal(t, "Modified description", cloned.tx.Description)
}

func TestTransactionBuilder_Reset(t *testing.T) {
	builder := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(100), "CHF")
	
	reset := builder.Reset()
	
	// Should be a new builder with defaults
	assert.NotEqual(t, builder, reset)
	assert.True(t, reset.tx.Date.IsZero())
	assert.True(t, reset.tx.Amount.IsZero())
	assert.Equal(t, "CHF", reset.tx.Currency) // Default currency
}

func TestTransactionBuilder_ComplexTransaction(t *testing.T) {
	// Test building a complex transaction with all fields
	tx, err := NewTransactionBuilder().
		WithID("TXN-001").
		WithBookkeepingNumber("BK-001").
		WithStatus(StatusCompleted).
		WithDate("2025-01-15").
		WithValueDate("2025-01-16").
		WithAmount(decimal.NewFromFloat(1500.75), "CHF").
		WithDescription("Investment purchase").
		WithRemittanceInfo("REF: INV-2025-001").
		WithPayer("John Doe", "CH1234567890123456").
		WithPayee("Investment Bank", "CH9876543210987654").
		WithReference("REF-001").
		WithEntryReference("ENTRY-001").
		WithAccountServicer("BANK-CH").
		WithBankTxCode(BankCodePOS).
		WithCategory(CategoryShopping).
		WithType("Investment").
		WithFund("Growth Fund").
		WithInvestment("Buy").
		WithNumberOfShares(15).
		WithFees(decimal.NewFromFloat(12.50)).
		WithIBAN("CH1111222233334444").
		WithOriginalAmount(decimal.NewFromFloat(1600), "USD").
		WithExchangeRate(decimal.NewFromFloat(0.94)).
		WithTax(decimal.NewFromFloat(1400), decimal.NewFromFloat(100.75), decimal.NewFromFloat(7.2)).
		AsDebit().
		Build()
	
	require.NoError(t, err)
	
	// Verify all fields are set correctly
	assert.Equal(t, "TXN-001", tx.Number)
	assert.Equal(t, "BK-001", tx.BookkeepingNumber)
	assert.Equal(t, StatusCompleted, tx.Status)
	
	expectedDate, _ := time.Parse("2006-01-02", "2025-01-15")
	expectedValueDate, _ := time.Parse("2006-01-02", "2025-01-16")
	assert.Equal(t, expectedDate, tx.Date)
	assert.Equal(t, expectedValueDate, tx.ValueDate)
	
	assert.True(t, decimal.NewFromFloat(1500.75).Equal(tx.Amount))
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "Investment purchase", tx.Description)
	assert.Equal(t, "REF: INV-2025-001", tx.RemittanceInfo)
	
	assert.Equal(t, "John Doe", tx.Payer)
	assert.Equal(t, "Investment Bank", tx.Payee)
	assert.Equal(t, "CH9876543210987654", tx.PartyIBAN) // Should be payee's IBAN for debit
	
	assert.Equal(t, "REF-001", tx.Reference)
	assert.Equal(t, "ENTRY-001", tx.EntryReference)
	assert.Equal(t, "BANK-CH", tx.AccountServicer)
	assert.Equal(t, BankCodePOS, tx.BankTxCode)
	
	assert.Equal(t, CategoryShopping, tx.Category)
	assert.Equal(t, "Investment", tx.Type)
	assert.Equal(t, "Growth Fund", tx.Fund)
	assert.Equal(t, "Buy", tx.Investment)
	assert.Equal(t, 15, tx.NumberOfShares)
	
	assert.True(t, decimal.NewFromFloat(12.50).Equal(tx.Fees))
	assert.Equal(t, "CH1111222233334444", tx.IBAN)
	
	assert.True(t, decimal.NewFromFloat(1600).Equal(tx.OriginalAmount))
	assert.Equal(t, "USD", tx.OriginalCurrency)
	assert.True(t, decimal.NewFromFloat(0.94).Equal(tx.ExchangeRate))
	
	assert.True(t, decimal.NewFromFloat(1400).Equal(tx.AmountExclTax))
	assert.True(t, decimal.NewFromFloat(100.75).Equal(tx.AmountTax))
	assert.True(t, decimal.NewFromFloat(7.2).Equal(tx.TaxRate))
	
	assert.Equal(t, TransactionTypeDebit, tx.CreditDebit)
	assert.True(t, tx.DebitFlag)
	
	// Verify derived fields
	assert.Equal(t, "Investment Bank", tx.Name) // Should be payee for debit
	assert.Equal(t, "Investment Bank", tx.Recipient)
	assert.Equal(t, "Investment Bank", tx.PartyName)
	
	// Verify debit/credit amounts
	assert.True(t, decimal.NewFromFloat(1500.75).Equal(tx.Debit))
	assert.True(t, decimal.Zero.Equal(tx.Credit))
}