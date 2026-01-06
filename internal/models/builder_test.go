package models

import (
	"strings"
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

// Additional edge case tests for TransactionBuilder validation
func TestTransactionBuilder_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() *TransactionBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name: "zero amount with debit direction",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.Zero, "CHF").
					AsDebit()
			},
			expectError: true, // Zero amounts are not allowed per business rules
			errorMsg:    "transaction amount is required",
		},
		{
			name: "negative amount auto-converts to debit",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.NewFromFloat(-100), "CHF")
			},
			expectError: false,
		},
		{
			name: "empty currency",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.NewFromFloat(100), "")
			},
			expectError: true,
			errorMsg:    "currency is required",
		},
		{
			name: "future date",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2030-12-31").
					WithAmount(decimal.NewFromFloat(100), "CHF")
			},
			expectError: false, // Future dates should be allowed
		},
		{
			name: "very large amount",
			setupFunc: func() *TransactionBuilder {
				largeAmount, _ := decimal.NewFromString("999999999999.99")
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(largeAmount, "CHF")
			},
			expectError: false,
		},
		{
			name: "very small amount",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.NewFromFloat(0.01), "CHF")
			},
			expectError: false,
		},
		{
			name: "empty party names",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.NewFromFloat(100), "CHF").
					WithPayer("", "").
					WithPayee("", "")
			},
			expectError: false, // Empty party names should be allowed
		},
		{
			name: "invalid IBAN format",
			setupFunc: func() *TransactionBuilder {
				return NewTransactionBuilder().
					WithDate("2025-01-15").
					WithAmount(decimal.NewFromFloat(100), "CHF").
					WithPayer("John Doe", "invalid-iban")
			},
			expectError: false, // IBAN validation not enforced in builder
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupFunc()
			tx, err := builder.Build()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Equal(t, Transaction{}, tx)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, Transaction{}, tx)
			}
		})
	}
}

// Test financial calculation precision
func TestTransactionBuilder_FinancialPrecision(t *testing.T) {
	// Test that decimal calculations maintain precision
	amount1, _ := decimal.NewFromString("100.123456789")
	amount2, _ := decimal.NewFromString("0.000000001")

	tx1, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(amount1, "CHF").
		Build()
	require.NoError(t, err)

	tx2, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(amount2, "CHF").
		Build()
	require.NoError(t, err)

	// Verify precision is maintained
	assert.True(t, amount1.Equal(tx1.Amount))
	assert.True(t, amount2.Equal(tx2.Amount))

	// Test that very precise calculations work
	preciseAmount, _ := decimal.NewFromString("123.456789012345678901234567890")
	tx3, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(preciseAmount, "CHF").
		Build()
	require.NoError(t, err)
	assert.True(t, preciseAmount.Equal(tx3.Amount))
}

// Test currency handling edge cases
func TestTransactionBuilder_CurrencyHandling(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		valid    bool
	}{
		{"Standard CHF", "CHF", true},
		{"Standard EUR", "EUR", true},
		{"Standard USD", "USD", true},
		{"Lowercase", "chf", true},     // Should be allowed
		{"Numeric", "123", true},       // Should be allowed
		{"Special chars", "CH$", true}, // Should be allowed
		{"Empty", "", false},
		{"Whitespace", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewTransactionBuilder().
				WithDate("2025-01-15").
				WithAmount(decimal.NewFromFloat(100), strings.TrimSpace(tt.currency))

			tx, err := builder.Build()

			if tt.valid {
				assert.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tt.currency), tx.Currency)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Test backward compatibility methods
func TestTransaction_BackwardCompatibilityMethods(t *testing.T) {
	// Create a debit transaction
	debitTx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(100.50), "CHF").
		WithPayer("Account Holder", "CH1111111111111111").
		WithPayee("Store", "CH2222222222222222").
		AsDebit().
		Build()
	require.NoError(t, err)

	// Create a credit transaction
	creditTx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(200.75), "EUR").
		WithPayer("Sender", "CH3333333333333333").
		WithPayee("Account Holder", "CH4444444444444444").
		AsCredit().
		Build()
	require.NoError(t, err)

	t.Run("GetPayee backward compatibility", func(t *testing.T) {
		// For debit: GetPayee should return the payee (who receives money)
		assert.Equal(t, "Store", debitTx.GetPayee())

		// For credit: GetPayee should return the payer (who sent money to us)
		assert.Equal(t, "Sender", creditTx.GetPayee())
	})

	t.Run("GetPayer backward compatibility", func(t *testing.T) {
		// For debit: GetPayer should return the payer (account holder)
		assert.Equal(t, "Account Holder", debitTx.GetPayer())

		// For credit: GetPayer should return the payee (account holder perspective)
		assert.Equal(t, "Account Holder", creditTx.GetPayer())
	})

	t.Run("GetCounterparty", func(t *testing.T) {
		// For debit: counterparty is the payee
		assert.Equal(t, "Store", debitTx.GetCounterparty())

		// For credit: counterparty is the payer
		assert.Equal(t, "Sender", creditTx.GetCounterparty())
	})

	t.Run("GetAmountAsFloat backward compatibility", func(t *testing.T) {
		assert.Equal(t, 100.50, debitTx.GetAmountAsFloat())
		assert.Equal(t, 200.75, creditTx.GetAmountAsFloat())
	})

	t.Run("Float conversion methods", func(t *testing.T) {
		// Test debit amounts
		assert.Equal(t, 100.50, debitTx.GetDebitAsFloat())
		assert.Equal(t, 0.0, debitTx.GetCreditAsFloat())

		// Test credit amounts
		assert.Equal(t, 0.0, creditTx.GetDebitAsFloat())
		assert.Equal(t, 200.75, creditTx.GetCreditAsFloat())
	})

	t.Run("Decimal accessor methods", func(t *testing.T) {
		assert.True(t, decimal.NewFromFloat(100.50).Equal(debitTx.GetAmountAsDecimal()))
		assert.True(t, decimal.NewFromFloat(200.75).Equal(creditTx.GetAmountAsDecimal()))
	})
}

// Test transaction direction detection edge cases
func TestTransaction_DirectionDetection(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() Transaction
		expectedDebit  bool
		expectedCredit bool
	}{
		{
			name: "explicit debit flag",
			setupFunc: func() Transaction {
				tx := Transaction{
					DebitFlag:   true,
					CreditDebit: "",
					Amount:      decimal.NewFromFloat(100),
				}
				return tx
			},
			expectedDebit:  true,
			expectedCredit: false,
		},
		{
			name: "explicit credit type",
			setupFunc: func() Transaction {
				tx := Transaction{
					DebitFlag:   false,
					CreditDebit: TransactionTypeCredit,
					Amount:      decimal.NewFromFloat(100),
				}
				return tx
			},
			expectedDebit:  false,
			expectedCredit: true,
		},
		{
			name: "negative amount implies debit",
			setupFunc: func() Transaction {
				tx := Transaction{
					DebitFlag:   false,
					CreditDebit: "",
					Amount:      decimal.NewFromFloat(-100),
				}
				return tx
			},
			expectedDebit:  true,
			expectedCredit: false,
		},
		{
			name: "positive amount with no flags",
			setupFunc: func() Transaction {
				tx := Transaction{
					DebitFlag:   false,
					CreditDebit: "",
					Amount:      decimal.NewFromFloat(100),
				}
				return tx
			},
			expectedDebit:  false,
			expectedCredit: true,
		},
		{
			name: "conflicting signals - both return true (current behavior)",
			setupFunc: func() Transaction {
				tx := Transaction{
					DebitFlag:   true,
					CreditDebit: TransactionTypeCredit,
					Amount:      decimal.NewFromFloat(100),
				}
				return tx
			},
			expectedDebit:  true, // DebitFlag takes precedence in IsDebit
			expectedCredit: true, // CreditDebit takes precedence in IsCredit - this is inconsistent behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := tt.setupFunc()
			assert.Equal(t, tt.expectedDebit, tx.IsDebit())
			assert.Equal(t, tt.expectedCredit, tx.IsCredit())
		})
	}
}

// Test financial calculation accuracy with edge cases
func TestTransaction_FinancialCalculationAccuracy(t *testing.T) {
	t.Run("precision preservation", func(t *testing.T) {
		// Test that decimal precision is preserved through operations
		preciseAmount, _ := decimal.NewFromString("123.456789012345")

		tx := Transaction{
			Amount:   preciseAmount,
			Currency: "CHF",
		}

		// Verify precision is maintained
		assert.True(t, preciseAmount.Equal(tx.GetAmountAsDecimal()))

		// Test that float conversion may lose precision (expected behavior)
		floatAmount := tx.GetAmountAsFloat()
		backToDecimal := decimal.NewFromFloat(floatAmount)

		// The float conversion should be close but may not be exactly equal
		diff := preciseAmount.Sub(backToDecimal).Abs()
		assert.True(t, diff.LessThan(decimal.NewFromFloat(0.000001)))
	})

	t.Run("large number handling", func(t *testing.T) {
		largeAmount, _ := decimal.NewFromString("999999999999.99")

		tx := Transaction{
			Amount:   largeAmount,
			Currency: "CHF",
		}

		assert.True(t, largeAmount.Equal(tx.GetAmountAsDecimal()))
	})

	t.Run("small number handling", func(t *testing.T) {
		smallAmount, _ := decimal.NewFromString("0.000000001")

		tx := Transaction{
			Amount:   smallAmount,
			Currency: "CHF",
		}

		assert.True(t, smallAmount.Equal(tx.GetAmountAsDecimal()))
	})

	t.Run("zero amount handling", func(t *testing.T) {
		tx := Transaction{
			Amount:   decimal.Zero,
			Currency: "CHF",
		}

		assert.True(t, decimal.Zero.Equal(tx.GetAmountAsDecimal()))
		assert.Equal(t, 0.0, tx.GetAmountAsFloat())
	})
}

// Test CSV marshaling/unmarshaling accuracy
func TestTransaction_CSVAccuracy(t *testing.T) {
	original := Transaction{
		BookkeepingNumber: "BK-001",
		Status:            StatusCompleted,
		Date:              time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
		Amount:            func() decimal.Decimal { d, _ := decimal.NewFromString("123.46"); return d }(), // Use 2 decimal places for CSV compatibility
		Currency:          "CHF",
		Description:       "Test transaction with special chars: äöü",
		Payer:             "John Doe",
		Payee:             "Jane Smith",
		CreditDebit:       TransactionTypeDebit,
		DebitFlag:         true,
		Category:          CategoryShopping,
		Fees:              func() decimal.Decimal { d, _ := decimal.NewFromString("5.50"); return d }(),
		OriginalAmount:    func() decimal.Decimal { d, _ := decimal.NewFromString("130.00"); return d }(),
		OriginalCurrency:  "USD",
		ExchangeRate:      func() decimal.Decimal { d, _ := decimal.NewFromString("0.95"); return d }(),
	}

	// Update derived fields
	original.UpdateNameFromParties()
	original.UpdateRecipientFromPayee()
	original.UpdateDebitCreditAmounts()

	// Marshal to CSV
	csvData, err := original.MarshalCSV()
	require.NoError(t, err)

	// Unmarshal back from CSV
	var restored Transaction
	err = restored.UnmarshalCSV(csvData)
	require.NoError(t, err)

	// Verify critical fields are preserved
	assert.Equal(t, original.BookkeepingNumber, restored.BookkeepingNumber)
	assert.Equal(t, original.Status, restored.Status)
	assert.Equal(t, original.Date, restored.Date)
	assert.Equal(t, original.ValueDate, restored.ValueDate)

	assert.True(t, original.Amount.Equal(restored.Amount))
	assert.Equal(t, original.Currency, restored.Currency)
	assert.Equal(t, original.Description, restored.Description)
	assert.Equal(t, original.CreditDebit, restored.CreditDebit)
	assert.Equal(t, original.DebitFlag, restored.DebitFlag)
	assert.Equal(t, original.Category, restored.Category)
	assert.True(t, original.Fees.Equal(restored.Fees))
	assert.True(t, original.OriginalAmount.Equal(restored.OriginalAmount))
	assert.Equal(t, original.OriginalCurrency, restored.OriginalCurrency)
	assert.True(t, original.ExchangeRate.Equal(restored.ExchangeRate))
}

// Test uncovered builder methods
func TestTransactionBuilder_UncoveredMethods(t *testing.T) {
	t.Run("WithDatetime", func(t *testing.T) {
		datetime := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)
		
		tx, err := NewTransactionBuilder().
			WithDatetime(datetime).
			WithAmount(decimal.NewFromFloat(100), "CHF").
			Build()
		
		require.NoError(t, err)
		assert.Equal(t, datetime, tx.Date)
	})

	t.Run("WithValueDatetime", func(t *testing.T) {
		datetime := time.Date(2025, 1, 16, 14, 30, 0, 0, time.UTC)
		
		tx, err := NewTransactionBuilder().
			WithDate("2025-01-15").
			WithValueDatetime(datetime).
			WithAmount(decimal.NewFromFloat(100), "CHF").
			Build()
		
		require.NoError(t, err)
		assert.Equal(t, datetime, tx.ValueDate)
	})

	t.Run("WithDateFromDatetime", func(t *testing.T) {
		datetimeStr := "2025-01-15 14:30:45"
		
		tx, err := NewTransactionBuilder().
			WithDateFromDatetime(datetimeStr).
			WithAmount(decimal.NewFromFloat(100), "CHF").
			Build()
		
		require.NoError(t, err)
		// The method actually preserves the full datetime, not just the date part
		expectedDate := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)
		assert.Equal(t, expectedDate, tx.Date)
	})

	t.Run("WithValueDateFromDatetime", func(t *testing.T) {
		datetimeStr := "2025-01-16 14:30:45"
		
		tx, err := NewTransactionBuilder().
			WithDate("2025-01-15").
			WithValueDateFromDatetime(datetimeStr).
			WithAmount(decimal.NewFromFloat(100), "CHF").
			Build()
		
		require.NoError(t, err)
		// The method actually preserves the full datetime, not just the date part
		expectedDate := time.Date(2025, 1, 16, 14, 30, 45, 0, time.UTC)
		assert.Equal(t, expectedDate, tx.ValueDate)
	})

	t.Run("WithAmountFromFloat", func(t *testing.T) {
		tx, err := NewTransactionBuilder().
			WithDate("2025-01-15").
			WithAmountFromFloat(123.45, "CHF").
			Build()
		
		require.NoError(t, err)
		assert.Equal(t, "123.45", tx.Amount.String())
		assert.Equal(t, "CHF", tx.Currency)
	})

	t.Run("WithPartyName", func(t *testing.T) {
		tx, err := NewTransactionBuilder().
			WithDate("2025-01-15").
			WithAmount(decimal.NewFromFloat(100), "CHF").
			WithPartyName("Test Party").
			Build()
		
		require.NoError(t, err)
		assert.Equal(t, "Test Party", tx.PartyName)
	})

	t.Run("WithPartyIBAN", func(t *testing.T) {
		tx, err := NewTransactionBuilder().
			WithDate("2025-01-15").
			WithAmount(decimal.NewFromFloat(100), "CHF").
			WithPartyIBAN("CH1234567890123456").
			Build()
		
		require.NoError(t, err)
		assert.Equal(t, "CH1234567890123456", tx.PartyIBAN)
	})

	t.Run("WithFeesFromFloat", func(t *testing.T) {
		tx, err := NewTransactionBuilder().
			WithDate("2025-01-15").
			WithAmount(decimal.NewFromFloat(100), "CHF").
			WithFeesFromFloat(5.50).
			Build()
		
		require.NoError(t, err)
		assert.Equal(t, "5.5", tx.Fees.String())
	})

	t.Run("WithTaxInfo", func(t *testing.T) {
		pricePerShare := decimal.NewFromFloat(10.50)
		amountTax := decimal.NewFromFloat(7.70)
		taxRate := decimal.NewFromFloat(7.7)
		
		tx, err := NewTransactionBuilder().
			WithDate("2025-01-15").
			WithAmount(decimal.NewFromFloat(100), "CHF").
			WithTaxInfo(pricePerShare, amountTax, taxRate).
			Build()
		
		require.NoError(t, err)
		// WithTaxInfo sets AmountExclTax, AmountTax, and TaxRate
		assert.True(t, pricePerShare.Equal(tx.AmountExclTax))
		assert.True(t, amountTax.Equal(tx.AmountTax))
		assert.True(t, taxRate.Equal(tx.TaxRate))
	})
}
