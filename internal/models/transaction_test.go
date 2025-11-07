package models

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAmountAsDecimal(t *testing.T) {
	testCases := []struct {
		name     string
		amount   string
		expected string
	}{
		{"SimpleAmount", "123.45", "123.45"},
		{"AmountWithComma", "123,45", "123.45"},
		{"NegativeAmount", "-123.45", "-123.45"},
		{"WithCurrencySymbol", "€123.45", "123.45"},
		{"WithCurrencyCode", "EUR 123.45", "123.45"},
		{"WithSpaces", " 123.45 ", "123.45"},
		{"WithThousandSeparator", "1'234.56", "1234.56"},
		{"InvalidAmount", "not-a-number", "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expected, _ := decimal.NewFromString(tc.expected)
			tx := &Transaction{Amount: ParseAmount(tc.amount)}
			result := tx.GetAmountAsDecimal()
			assert.True(t, expected.Equal(result), "GetAmountAsDecimal() with Amount=%s should return %s, got %s", tc.amount, tc.expected, result.String())
		})
	}
}

func TestCreditDebitMethods(t *testing.T) {
	t.Run("IsDebit", func(t *testing.T) {
		debitTx := &Transaction{CreditDebit: TransactionTypeDebit}
		creditTx := &Transaction{CreditDebit: TransactionTypeCredit}
		unknownTx := &Transaction{CreditDebit: "UNKNOWN"}

		assert.True(t, debitTx.IsDebit(), "Transaction with CreditDebit=DBIT should return true for IsDebit()")
		assert.False(t, creditTx.IsDebit(), "Transaction with CreditDebit=CRDT should return false for IsDebit()")
		assert.False(t, unknownTx.IsDebit(), "Transaction with CreditDebit=UNKNOWN should return false for IsDebit()")
	})

	t.Run("IsCredit", func(t *testing.T) {
		debitTx := &Transaction{CreditDebit: TransactionTypeDebit}
		creditTx := &Transaction{CreditDebit: TransactionTypeCredit}
		unknownTx := &Transaction{CreditDebit: "UNKNOWN"}

		assert.False(t, debitTx.IsCredit(), "Transaction with CreditDebit=DBIT should return false for IsCredit()")
		assert.True(t, creditTx.IsCredit(), "Transaction with CreditDebit=CRDT should return true for IsCredit()")
		assert.False(t, unknownTx.IsCredit(), "Transaction with CreditDebit=UNKNOWN should return false for IsCredit()")
	})
}

func TestGetPartyName(t *testing.T) {
	testCases := []struct {
		name        string
		creditDebit string
		payee       string
		payer       string
		expected    string
	}{
		{"DebitTransaction", TransactionTypeDebit, "Payee Example", "Payer Example", "Payee Example"},
		{"CreditTransaction", TransactionTypeCredit, "Payee Example", "Payer Example", "Payer Example"},
		{"EmptyParties", TransactionTypeCredit, "", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tx := &Transaction{
				CreditDebit: tc.creditDebit,
				Payee:       tc.payee,
				Payer:       tc.payer,
			}

			result := tx.GetPartyName()
			assert.Equal(t, tc.expected, result, "GetPartyName() should return the correct party name")
		})
	}
}

func TestStandardizeAmount(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"SimpleAmount", "123.45", "123.45"},
		{"AmountWithComma", "123,45", "123.45"},
		{"NegativeAmount", "-123.45", "-123.45"},
		{"WithCurrencySymbol", "€123.45", "123.45"},
		{"WithCurrencyCode", "EUR 123.45", "123.45"},
		{"WithSpaces", " 123.45 ", "123.45"},
		{"WithThousandSeparator", "1'234.56", "1234.56"},
		{"NoDecimalPlaces", "42", "42.00"},
		{"SingleDecimalPlace", "42.5", "42.50"},
		{"ManyDecimalPlaces", "42.4242", "42.42"},
		{"InvalidAmount", "not-a-number", "not-a-number"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := StandardizeAmount(tc.input)
			assert.Equal(t, tc.expected, result, "StandardizeAmount(%s) should return %s", tc.input, tc.expected)
		})
	}
}

func TestUpdateInvestmentTypeFromLegacyField(t *testing.T) {
	t.Run("CopyFromType", func(t *testing.T) {
		tx := &Transaction{Type: "Buy"}
		tx.UpdateInvestmentTypeFromLegacyField()
		assert.Equal(t, "Buy", tx.Investment)
	})

	t.Run("DoNotOverride", func(t *testing.T) {
		tx := &Transaction{Investment: "Sell", Type: "Buy"}
		tx.UpdateInvestmentTypeFromLegacyField()
		assert.Equal(t, "Sell", tx.Investment)
	})

	t.Run("EmptyFields", func(t *testing.T) {
		tx := &Transaction{}
		tx.UpdateInvestmentTypeFromLegacyField()
		assert.Equal(t, "", tx.Investment)
	})
}

func TestTransaction_GetPayee_DirectionBased(t *testing.T) {
	tests := []struct {
		name        string
		tx          Transaction
		expected    string
		description string
	}{
		{
			name: "debit transaction returns payee",
			tx: Transaction{
				CreditDebit: TransactionTypeDebit,
				DebitFlag:   true,
				Payer:       "Account Holder",
				Payee:       "Store ABC",
			},
			expected:    "Store ABC",
			description: "For debit transactions, GetPayee() should return the payee (who receives money)",
		},
		{
			name: "credit transaction returns payer",
			tx: Transaction{
				CreditDebit: TransactionTypeCredit,
				DebitFlag:   false,
				Payer:       "Employer Corp",
				Payee:       "Account Holder",
			},
			expected:    "Employer Corp",
			description: "For credit transactions, GetPayee() should return the payer (who sent money to us)",
		},
		{
			name: "unknown direction with zero amount returns payer",
			tx: Transaction{
				CreditDebit: "UNKNOWN",
				DebitFlag:   false,
				Payer:       "Some Payer",
				Payee:       "Some Payee",
				Amount:      decimal.Zero,
			},
			expected:    "Some Payer",
			description: "For unknown direction with zero amount (credit behavior), GetPayee() returns payer",
		},
		{
			name: "negative amount treated as debit",
			tx: Transaction{
				CreditDebit: "",
				DebitFlag:   false,
				Amount:      decimal.NewFromFloat(-50.00),
				Payer:       "Account Holder",
				Payee:       "Utility Company",
			},
			expected:    "Utility Company",
			description: "Negative amounts are treated as debits, so GetPayee() returns the payee",
		},
		{
			name: "empty parties in debit",
			tx: Transaction{
				CreditDebit: TransactionTypeDebit,
				DebitFlag:   true,
				Payer:       "",
				Payee:       "",
			},
			expected:    "",
			description: "Empty payee in debit transaction should return empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tx.GetPayee()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestTransaction_GetPayer_DirectionBased(t *testing.T) {
	tests := []struct {
		name        string
		tx          Transaction
		expected    string
		description string
	}{
		{
			name: "debit transaction returns payer",
			tx: Transaction{
				CreditDebit: TransactionTypeDebit,
				DebitFlag:   true,
				Payer:       "Account Holder",
				Payee:       "Store ABC",
			},
			expected:    "Account Holder",
			description: "For debit transactions, GetPayer() should return the payer (account holder)",
		},
		{
			name: "credit transaction returns payee",
			tx: Transaction{
				CreditDebit: TransactionTypeCredit,
				DebitFlag:   false,
				Payer:       "Employer Corp",
				Payee:       "Account Holder",
			},
			expected:    "Account Holder",
			description: "For credit transactions, GetPayer() should return the payee (account holder)",
		},
		{
			name: "unknown direction with zero amount returns payee",
			tx: Transaction{
				CreditDebit: "UNKNOWN",
				DebitFlag:   false,
				Payer:       "Some Payer",
				Payee:       "Some Payee",
				Amount:      decimal.Zero,
			},
			expected:    "Some Payee",
			description: "For unknown direction with zero amount (credit behavior), GetPayer() returns payee",
		},
		{
			name: "negative amount treated as debit",
			tx: Transaction{
				CreditDebit: "",
				DebitFlag:   false,
				Amount:      decimal.NewFromFloat(-50.00),
				Payer:       "Account Holder",
				Payee:       "Utility Company",
			},
			expected:    "Account Holder",
			description: "Negative amounts are treated as debits, so GetPayer() returns the payer",
		},
		{
			name: "empty parties in credit",
			tx: Transaction{
				CreditDebit: TransactionTypeCredit,
				DebitFlag:   false,
				Payer:       "",
				Payee:       "",
			},
			expected:    "",
			description: "Empty payee in credit transaction should return empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tx.GetPayer()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestTransaction_GetAmountAsFloat_Precision(t *testing.T) {
	tests := []struct {
		name     string
		amount   decimal.Decimal
		expected float64
	}{
		{
			name:     "simple amount",
			amount:   decimal.NewFromFloat(100.50),
			expected: 100.50,
		},
		{
			name:     "zero amount",
			amount:   decimal.Zero,
			expected: 0.0,
		},
		{
			name:     "negative amount",
			amount:   decimal.NewFromFloat(-75.25),
			expected: -75.25,
		},
		{
			name:     "large amount",
			amount:   decimal.NewFromFloat(999999.99),
			expected: 999999.99,
		},
		{
			name:     "small fractional amount",
			amount:   decimal.NewFromFloat(0.01),
			expected: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := Transaction{Amount: tt.amount}
			result := tx.GetAmountAsFloat()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransaction_SetPayerInfo(t *testing.T) {
	tx := Transaction{}
	tx.SetPayerInfo("John Doe", "CH1234567890")

	assert.Equal(t, "John Doe", tx.Payer)
	assert.Equal(t, "CH1234567890", tx.PartyIBAN)
}

func TestTransaction_SetPayeeInfo(t *testing.T) {
	tx := Transaction{}
	tx.SetPayeeInfo("Acme Corp", "CH0987654321")

	assert.Equal(t, "Acme Corp", tx.Payee)
	assert.Equal(t, "CH0987654321", tx.PartyIBAN)
}

func TestTransaction_SetAmountFromFloat(t *testing.T) {
	tx := Transaction{}
	tx.SetAmountFromFloat(123.45, "EUR")

	assert.True(t, decimal.NewFromFloat(123.45).Equal(tx.Amount))
	assert.Equal(t, "EUR", tx.Currency)
}

func TestTransaction_ToBuilder(t *testing.T) {
	original := Transaction{
		Number:      "TXN-001",
		Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:      decimal.NewFromFloat(100.50),
		Currency:    "CHF",
		Description: "Test transaction",
		Payer:       "John Doe",
		Payee:       "Acme Corp",
	}

	builder := original.ToBuilder()

	// Verify the builder has the same data
	assert.Equal(t, original.Number, builder.tx.Number)
	assert.Equal(t, original.Date, builder.tx.Date)
	assert.True(t, original.Amount.Equal(builder.tx.Amount))
	assert.Equal(t, original.Currency, builder.tx.Currency)
	assert.Equal(t, original.Description, builder.tx.Description)
	assert.Equal(t, original.Payer, builder.tx.Payer)
	assert.Equal(t, original.Payee, builder.tx.Payee)

	// Modify through builder and build
	modified, err := builder.WithDescription("Modified description").Build()
	require.NoError(t, err)

	assert.Equal(t, "Modified description", modified.Description)
	assert.Equal(t, original.Number, modified.Number) // Other fields should remain
}

func TestNewTransactionFromBuilder(t *testing.T) {
	builder := NewTransactionFromBuilder()

	assert.NotNil(t, builder)
	assert.IsType(t, &TransactionBuilder{}, builder)

	// Should be able to build a transaction
	tx, err := builder.
		WithDate("2025-01-15").
		WithAmount(decimal.NewFromFloat(100), "CHF").
		Build()

	require.NoError(t, err)
	assert.NotEmpty(t, tx.Number)
}
