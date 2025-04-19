package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDate(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		// Already correct format
		{"AlreadyCorrectFormat", "01.01.2023", "01.01.2023"},
		{"EmptyString", "", ""},
		
		// ISO format
		{"ISO_Date", "2023-01-01", "01.01.2023"},
		{"ISO_DateTime", "2023-01-01 12:30:45", "01.01.2023"},
		{"ISO_DateTimeZ", "2023-01-01T12:30:45Z", "01.01.2023"},
		{"ISO_DateTimeZone", "2023-01-01T12:30:45+01:00", "01.01.2023"},
		
		// European formats (day first)
		{"European_SlashFormat", "01/01/2023", "01.01.2023"},
		{"European_DashFormat", "01-01-2023", "01.01.2023"},
		
		// US formats (month first)
		{"US_SlashFormat", "12/31/2023", "31.12.2023"},
		{"US_DashFormat", "12-31-2023", "31.12.2023"},
		
		// Text formats
		{"LongFormat", "January 1, 2023", "01.01.2023"},
		{"DayFirstLongFormat", "1 January 2023", "01.01.2023"},
		{"ShortMonthFormat", "01 Jan 2023", "01.01.2023"},
		{"MonthYearOnly", "January 2023", "01.01.2023"},
		{"ShortMonthYearOnly", "Jan 2023", "01.01.2023"},
		
		// Short formats
		{"MonthYear_Slash", "01/2023", "01.01.2023"},
		{"YearMonth_Slash", "2023/01", "01.01.2023"},
		
		// Unusual but valid formats
		{"ShortDayMonthFormat", "1.1.2023", "01.01.2023"},
		
		// Invalid format - should return original
		{"Invalid_Format", "not-a-date", "not-a-date"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDate(tc.input)
			assert.Equal(t, tc.expected, result, "FormatDate(%s) should return %s", tc.input, tc.expected)
		})
	}
}

func TestGetAmountAsFloat(t *testing.T) {
	testCases := []struct {
		name     string
		amount   string
		expected float64
	}{
		{"SimpleAmount", "123.45", 123.45},
		{"AmountWithComma", "123,45", 123.45},
		{"NegativeAmount", "-123.45", -123.45},
		{"WithCurrencySymbol", "€123.45", 123.45},
		{"WithCurrencyCode", "EUR 123.45", 123.45},
		{"WithSpaces", " 123.45 ", 123.45},
		{"WithThousandSeparator", "1'234.56", 1234.56},
		{"InvalidAmount", "not-a-number", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tx := &Transaction{Amount: tc.amount}
			result := tx.GetAmountAsFloat()
			assert.Equal(t, tc.expected, result, "GetAmountAsFloat() with Amount=%s should return %f", tc.amount, tc.expected)
		})
	}
}

func TestCreditDebitMethods(t *testing.T) {
	t.Run("IsDebit", func(t *testing.T) {
		debitTx := &Transaction{CreditDebit: "DBIT"}
		creditTx := &Transaction{CreditDebit: "CRDT"}
		unknownTx := &Transaction{CreditDebit: "UNKNOWN"}
		
		assert.True(t, debitTx.IsDebit(), "Transaction with CreditDebit=DBIT should return true for IsDebit()")
		assert.False(t, creditTx.IsDebit(), "Transaction with CreditDebit=CRDT should return false for IsDebit()")
		assert.False(t, unknownTx.IsDebit(), "Transaction with CreditDebit=UNKNOWN should return false for IsDebit()")
	})
	
	t.Run("IsCredit", func(t *testing.T) {
		debitTx := &Transaction{CreditDebit: "DBIT"}
		creditTx := &Transaction{CreditDebit: "CRDT"}
		unknownTx := &Transaction{CreditDebit: "UNKNOWN"}
		
		assert.False(t, debitTx.IsCredit(), "Transaction with CreditDebit=DBIT should return false for IsCredit()")
		assert.True(t, creditTx.IsCredit(), "Transaction with CreditDebit=CRDT should return true for IsCredit()")
		assert.False(t, unknownTx.IsCredit(), "Transaction with CreditDebit=UNKNOWN should return false for IsCredit()")
	})
}

func TestGetPartyName(t *testing.T) {
	testCases := []struct {
		name       string
		creditDebit string
		payee      string
		payer      string
		expected   string
	}{
		{"DebitTransaction", "DBIT", "Payee Example", "Payer Example", "Payee Example"},
		{"CreditTransaction", "CRDT", "Payee Example", "Payer Example", "Payer Example"},
		{"EmptyParties", "CRDT", "", "", ""},
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
