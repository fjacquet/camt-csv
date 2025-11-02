package models

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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
