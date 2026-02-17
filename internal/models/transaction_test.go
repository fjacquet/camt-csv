package models

import (
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"

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

// Test uncovered transaction methods
func TestTransaction_UncoveredMethods(t *testing.T) {
	t.Run("SetAmountFromDecimal", func(t *testing.T) {
		tx := Transaction{}
		amount := decimal.NewFromFloat(123.45)

		tx.SetAmountFromDecimal(amount)

		assert.True(t, amount.Equal(tx.Amount))
	})

	t.Run("GetOriginalAmountAsDecimal", func(t *testing.T) {
		originalAmount := decimal.NewFromFloat(100.50)
		tx := Transaction{OriginalAmount: originalAmount}

		result := tx.GetOriginalAmountAsDecimal()
		assert.True(t, originalAmount.Equal(result))

		// Test zero original amount
		txZero := Transaction{OriginalAmount: decimal.Zero}
		assert.True(t, decimal.Zero.Equal(txZero.GetOriginalAmountAsDecimal()))
	})

	t.Run("GetExchangeRateAsDecimal", func(t *testing.T) {
		exchangeRate := decimal.NewFromFloat(0.92)
		tx := Transaction{ExchangeRate: exchangeRate}

		result := tx.GetExchangeRateAsDecimal()
		assert.True(t, exchangeRate.Equal(result))

		// Test zero exchange rate
		txZero := Transaction{ExchangeRate: decimal.Zero}
		assert.True(t, decimal.Zero.Equal(txZero.GetExchangeRateAsDecimal()))
	})

	t.Run("GetFeesAsDecimal", func(t *testing.T) {
		fees := decimal.NewFromFloat(12.50)
		tx := Transaction{Fees: fees}

		result := tx.GetFeesAsDecimal()
		assert.True(t, fees.Equal(result))

		// Test zero fees
		txZero := Transaction{Fees: decimal.Zero}
		assert.True(t, decimal.Zero.Equal(txZero.GetFeesAsDecimal()))
	})

	t.Run("SetFeesFromDecimal", func(t *testing.T) {
		tx := Transaction{}
		fees := decimal.NewFromFloat(7.75)

		tx.SetFeesFromDecimal(fees)

		assert.True(t, fees.Equal(tx.Fees))
	})
}

// Test date formatting functions (these are unexported, so we test them indirectly)
func TestTransaction_DateFormattingIndirect(t *testing.T) {
	t.Run("CSV marshaling with dates", func(t *testing.T) {
		// Test that date formatting works through CSV marshaling
		tx := Transaction{
			Date:      time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC),
			ValueDate: time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC),
			Amount:    decimal.NewFromFloat(100),
			Currency:  "CHF",
		}

		csvData, err := tx.MarshalCSV()
		require.NoError(t, err)

		// Verify dates are formatted correctly in CSV (DD.MM.YYYY format)
		// In new 29-column format: Date is index 1, ValueDate is index 2
		assert.Contains(t, csvData[1], "15.01.2025") // Date field
		assert.Contains(t, csvData[2], "16.01.2025") // ValueDate field
	})

	t.Run("CSV unmarshaling with dates", func(t *testing.T) {
		// Test that date parsing works through CSV unmarshaling
		// Create a complete CSV record with all required fields (29 columns)
		csvData := []string{
			"COMPLETED",   // Status
			"15.01.2025",  // Date
			"16.01.2025",  // ValueDate
			"Test Name",   // Name
			"Test Party",  // PartyName
			"CH123456789", // PartyIBAN
			"Test Desc",   // Description
			"Test Info",   // RemittanceInfo
			"100",         // Amount
			"CRDT",        // CreditDebit
			"CHF",         // Currency
			"",            // Product
			"0",           // AmountExclTax
			"0",           // TaxRate
			"",            // InvestmentType
			"1",           // Number
			"Shopping",    // Category
			"",            // Type
			"",            // Fund
			"0",           // NumberOfShares
			"0",           // Fees
			"CH123456789", // IBAN
			"REF-001",     // EntryReference
			"REF-001",     // Reference
			"BANK-CH",     // AccountServicer
			"PMNT",        // BankTxCode
			"CHF",         // OriginalCurrency
			"100",         // OriginalAmount
			"1",           // ExchangeRate
		}

		var tx Transaction
		err := tx.UnmarshalCSV(csvData)
		require.NoError(t, err)

		expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		expectedValueDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)

		assert.Equal(t, expectedDate, tx.Date)
		assert.Equal(t, expectedValueDate, tx.ValueDate)
	})
}

// Test categorization stats methods
func TestCategorizationStats_UncoveredMethods(t *testing.T) {
	t.Run("NewCategorizationStats", func(t *testing.T) {
		stats := NewCategorizationStats()

		assert.Equal(t, 0, stats.Total)
		assert.Equal(t, 0, stats.Successful)
		assert.Equal(t, 0, stats.Failed)
		assert.Equal(t, 0, stats.Uncategorized)
	})

	t.Run("IncrementTotal", func(t *testing.T) {
		stats := NewCategorizationStats()

		stats.IncrementTotal()
		assert.Equal(t, 1, stats.Total)

		stats.IncrementTotal()
		assert.Equal(t, 2, stats.Total)
	})

	t.Run("IncrementSuccessful", func(t *testing.T) {
		stats := NewCategorizationStats()

		stats.IncrementSuccessful()
		assert.Equal(t, 1, stats.Successful)

		stats.IncrementSuccessful()
		assert.Equal(t, 2, stats.Successful)
	})

	t.Run("IncrementFailed", func(t *testing.T) {
		stats := NewCategorizationStats()

		stats.IncrementFailed()
		assert.Equal(t, 1, stats.Failed)

		stats.IncrementFailed()
		assert.Equal(t, 2, stats.Failed)
	})

	t.Run("IncrementUncategorized", func(t *testing.T) {
		stats := NewCategorizationStats()

		stats.IncrementUncategorized()
		assert.Equal(t, 1, stats.Uncategorized)

		stats.IncrementUncategorized()
		assert.Equal(t, 2, stats.Uncategorized)
	})

	t.Run("GetSuccessRate", func(t *testing.T) {
		stats := NewCategorizationStats()

		// Test with zero total
		assert.Equal(t, 0.0, stats.GetSuccessRate())

		// Test with some successful
		stats.Total = 10
		stats.Successful = 7
		assert.Equal(t, 70.0, stats.GetSuccessRate())

		// Test with 100% success
		stats.Total = 5
		stats.Successful = 5
		assert.Equal(t, 100.0, stats.GetSuccessRate())
	})

	t.Run("LogSummary", func(t *testing.T) {
		stats := NewCategorizationStats()
		stats.Total = 10
		stats.Successful = 7
		stats.Failed = 2
		stats.Uncategorized = 1

		// This method logs to the provided logger, we just test it doesn't panic
		// Note: LogSummary takes (logger, prefix) parameters
		logger := &MockLogger{}
		assert.NotPanics(t, func() {
			stats.LogSummary(logger, "Test")
		})
	})
}

// Mock logger for testing
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields ...logging.Field)         {}
func (m *MockLogger) Info(msg string, fields ...logging.Field)          {}
func (m *MockLogger) Warn(msg string, fields ...logging.Field)          {}
func (m *MockLogger) Error(msg string, fields ...logging.Field)         {}
func (m *MockLogger) Fatal(msg string, fields ...logging.Field)         {}
func (m *MockLogger) Debugf(format string, args ...any)                 {}
func (m *MockLogger) Infof(format string, args ...any)                  {}
func (m *MockLogger) Warnf(format string, args ...any)                  {}
func (m *MockLogger) Errorf(format string, args ...any)                 {}
func (m *MockLogger) Fatalf(format string, args ...any)                 {}
func (m *MockLogger) WithError(err error) logging.Logger                { return m }
func (m *MockLogger) WithField(key string, value any) logging.Logger    { return m }
func (m *MockLogger) WithFields(fields ...logging.Field) logging.Logger { return m }
