package textutils_test

import (
	"testing"

	"fjacquet/camt-csv/internal/textutils"

	"github.com/stretchr/testify/assert"
)

func TestExtractBookkeepingNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "BookKeeping Number pattern",
			input:    "BookKeeping Number: 12345-67890",
			expected: "12345-67890",
		},
		{
			name:     "Booking no pattern",
			input:    "Booking no: 98765",
			expected: "98765",
		},
		{
			name:     "No booking pattern",
			input:    "No booking: 11111-22222",
			expected: "11111-22222",
		},
		{
			name:     "Reference pattern",
			input:    "Reference: 55555",
			expected: "55555",
		},
		{
			name:     "No match",
			input:    "Some random text without a booking number",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutils.ExtractBookkeepingNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFund(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Fund pattern",
			input:    "Fund: UBS Global Equity",
			expected: "UBS Global Equity",
		},
		{
			name:     "Investment Fund pattern",
			input:    "Investment Fund: Vanguard S&P 500",
			expected: "Vanguard S&P 500",
		},
		{
			name:     "Fund with comma separator",
			input:    "Fund: Swiss Equity, Amount: 1000",
			expected: "Swiss Equity",
		},
		{
			name:     "No match",
			input:    "Some transaction description",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutils.ExtractFund(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPayee(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Payee pattern",
			input:    "Payee: John Doe",
			expected: "John Doe",
		},
		{
			name:     "Beneficiaire pattern (French)",
			input:    "Bénéficiaire: Marie Dupont",
			expected: "Marie Dupont",
		},
		{
			name:     "Recipient pattern",
			input:    "Recipient: ACME Corp",
			expected: "ACME Corp",
		},
		{
			name:     "To pattern",
			input:    "To: Swiss Bank AG",
			expected: "Swiss Bank AG",
		},
		{
			name:     "Payment to pattern",
			input:    "Payment to: Electric Company",
			expected: "Electric Company",
		},
		{
			name:     "No match",
			input:    "Transfer completed successfully",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutils.ExtractPayee(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractMerchant(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Card purchase at merchant",
			input:    "Card purchase at Migros on 2024-01-15",
			expected: "migros",
		},
		{
			name:     "TWINT payment",
			input:    "TWINT at Coop on 2024-01-15",
			expected: "coop",
		},
		{
			name:     "No card or TWINT keyword",
			input:    "Bank transfer to John",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutils.ExtractMerchant(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFundInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UBS Fund",
			input:    "Purchase of UBS Fund Global Equity",
			expected: "UBS FUND GLOBAL EQUITY",
		},
		{
			name:     "Vanguard ETF MSCI",
			input:    "Dividend from VANGUARD ETF MSCI Europe",
			expected: "VANGUARD ETF MSCI EUROPE",
		},
		{
			name:     "iShares Fund",
			input:    "Trade ISHARES MSCI World",
			expected: "ISHARES MSCI WORLD",
		},
		{
			name:     "No fund match",
			input:    "Regular bank transfer",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutils.ExtractFundInfo(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBankTxCode(t *testing.T) {
	tests := []struct {
		name      string
		domain    string
		family    string
		subfamily string
		expected  string
	}{
		{
			name:      "Full code",
			domain:    "PMNT",
			family:    "RCDT",
			subfamily: "ESCT",
			expected:  "PMNT.RCDT.ESCT",
		},
		{
			name:      "Domain and family only",
			domain:    "PMNT",
			family:    "RCDT",
			subfamily: "",
			expected:  "PMNT.RCDT",
		},
		{
			name:      "Domain only",
			domain:    "PMNT",
			family:    "",
			subfamily: "",
			expected:  "PMNT",
		},
		{
			name:      "Empty domain",
			domain:    "",
			family:    "RCDT",
			subfamily: "ESCT",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := textutils.FormatBankTxCode(tt.domain, tt.family, tt.subfamily)
			assert.Equal(t, tt.expected, result)
		})
	}
}
