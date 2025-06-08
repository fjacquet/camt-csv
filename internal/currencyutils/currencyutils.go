// Package currencyutils provides common currency and decimal operations used throughout the application.
package currencyutils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// SetLogger sets a custom logger for this package
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
	}
}

// ParseAmount parses a string representation of an amount into a decimal value
// It handles various formats like "1,234.56", "1.234,56", "1234.56", "1234,56"
func ParseAmount(amountStr string) (decimal.Decimal, error) {
	// Return zero for empty strings
	if amountStr == "" {
		return decimal.Zero, nil
	}

	// Standardize the amount string (remove currency symbols, extra spaces, etc.)
	standardized := StandardizeAmount(amountStr)

	// Parse the standardized string
	amount, err := decimal.NewFromString(standardized)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse amount '%s': %w", amountStr, err)
	}

	return amount, nil
}

// StandardizeAmount converts various currency string formats to a standard format that can be parsed by decimal.NewFromString
// Handles patterns like "CHF 1'234.56", "€1.234,56", "$1,234.56", "1 234,56", etc.
func StandardizeAmount(amountStr string) string {
	// Remove all currency symbols and extra whitespace
	re := regexp.MustCompile(`[€$£¥₣₤₧₹₺₽₩฿₫₲₴₸₼₪CHF\s]`)
	amountStr = re.ReplaceAllString(amountStr, "")

	// Handle European format (1.234,56) -> (1234.56)
	if strings.Contains(amountStr, ",") && strings.Contains(amountStr, ".") {
		if strings.LastIndex(amountStr, ".") < strings.LastIndex(amountStr, ",") {
			// European format (1.234,56)
			amountStr = strings.ReplaceAll(amountStr, ".", "")
			amountStr = strings.ReplaceAll(amountStr, ",", ".")
		}
	} else if strings.Contains(amountStr, ",") {
		// If only comma is present as decimal separator (1234,56) or thousand separator (1,234)
		// Determine if the comma is used as a decimal separator or thousand separator
		parts := strings.Split(amountStr, ",")
		if len(parts) > 1 && len(parts[len(parts)-1]) <= 2 {
			// Comma used as decimal separator (1234,56)
			amountStr = strings.ReplaceAll(amountStr, ",", ".")
		} else {
			// Comma used as thousand separator (1,234)
			amountStr = strings.ReplaceAll(amountStr, ",", "")
		}
	}

	// Remove apostrophes used as thousand separators (1'234.56)
	amountStr = strings.ReplaceAll(amountStr, "'", "")

	return amountStr
}

// FormatAmount formats a decimal amount to a consistent display format with the specified currency.
// The amount is formatted with two decimal places without inserting thousands separators.
// Returns strings like "CHF 1234.56" or "€1234.56"
func FormatAmount(amount decimal.Decimal, currency string) string {
	// Format the amount with 2 decimal places without thousands separators
	formattedAmount := amount.StringFixed(2)

	// Add currency symbol or code
	if currency != "" {
		switch strings.ToUpper(currency) {
		case "EUR":
			return "€" + formattedAmount
		case "USD":
			return "$" + formattedAmount
		case "GBP":
			return "£" + formattedAmount
		case "JPY":
			return "¥" + formattedAmount
		case "CHF":
			return "CHF " + formattedAmount
		default:
			return currency + " " + formattedAmount
		}
	}

	return formattedAmount
}

// IsNegative checks if an amount is negative
func IsNegative(amount decimal.Decimal) bool {
	return amount.LessThan(decimal.Zero)
}

// IsPositive checks if an amount is positive
func IsPositive(amount decimal.Decimal) bool {
	return amount.GreaterThan(decimal.Zero)
}

// IsZero checks if an amount is zero
func IsZero(amount decimal.Decimal) bool {
	return amount.Equal(decimal.Zero)
}

// CalculateTaxAmount calculates the tax amount given the total amount and tax rate
// e.g., CalculateTaxAmount(100, 7.7) returns 7.7
func CalculateTaxAmount(amount decimal.Decimal, taxRatePercent decimal.Decimal) decimal.Decimal {
	return amount.Mul(taxRatePercent.Div(decimal.NewFromInt(100)))
}

// AmountExcludingTax calculates the amount excluding tax given the total amount and tax rate
// e.g., AmountExcludingTax(107.7, 7.7) returns 100
func AmountExcludingTax(totalAmount decimal.Decimal, taxRatePercent decimal.Decimal) decimal.Decimal {
	if taxRatePercent.Equal(decimal.NewFromInt(100)) {
		return decimal.Zero
	}

	divisor := decimal.NewFromInt(100).Add(taxRatePercent)
	return totalAmount.Mul(decimal.NewFromInt(100)).Div(divisor).Round(2)
}

// AmountIncludingTax calculates the amount including tax given the net amount and tax rate
// e.g., AmountIncludingTax(100, 7.7) returns 107.7
func AmountIncludingTax(netAmount decimal.Decimal, taxRatePercent decimal.Decimal) decimal.Decimal {
	taxAmount := netAmount.Mul(taxRatePercent.Div(decimal.NewFromInt(100)))
	return netAmount.Add(taxAmount)
}
