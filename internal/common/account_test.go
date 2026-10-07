package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSanitizeAccountID tests account ID sanitization
func TestSanitizeAccountID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean alphanumeric ID",
			input:    "ABC123",
			expected: "ABC123",
		},
		{
			name:     "ID with spaces",
			input:    "ABC 123 XYZ",
			expected: "ABC_123_XYZ",
		},
		{
			name:     "ID with special characters",
			input:    "ABC@123#XYZ",
			expected: "ABC_123_XYZ",
		},
		{
			name:     "ID with multiple consecutive spaces",
			input:    "ABC   123",
			expected: "ABC_123",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "UNKNOWN",
		},
		{
			name:     "Only special characters",
			input:    "@#$%",
			expected: "UNKNOWN",
		},
		{
			name:     "Leading and trailing underscores",
			input:    "_ABC123_",
			expected: "ABC123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeAccountID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// cryptoRandIntn returns a random int in [0, n) using crypto/rand
func cryptoRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	max := big.NewInt(int64(n))
	result, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return int(result.Int64())
}

// Property-based test for account ID sanitization
// **Feature: parser-enhancements, Property 13: Account identification from filenames**
// **Validates: Requirements 6.1, 7.3**
func TestProperty_AccountIDSanitization(t *testing.T) {
	// Property: For any input string, SanitizeAccountID should return a filesystem-safe identifier

	const iterations = 100

	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random input with various characters
			input := generateRandomAccountID()

			// Test sanitization
			result := SanitizeAccountID(input)

			// Property assertions: result should be filesystem-safe
			assert.NotEmpty(t, result, "Sanitized ID should not be empty")

			// Check that result contains only safe characters
			for _, r := range result {
				assert.True(t, isFilesystemSafeChar(r),
					"Character '%c' in result '%s' should be filesystem-safe (input: '%s')", r, result, input)
			}

			// Should not contain consecutive underscores
			assert.False(t, strings.Contains(result, "__"),
				"Result should not contain consecutive underscores: %s (input: %s)", result, input)

			// Should not start or end with underscore
			assert.False(t, strings.HasPrefix(result, "_"),
				"Result should not start with underscore: %s (input: %s)", result, input)
			assert.False(t, strings.HasSuffix(result, "_"),
				"Result should not end with underscore: %s (input: %s)", result, input)
		})
	}
}

func generateRandomAccountID() string {
	// Generate random string with mix of safe and unsafe characters
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-. @#$%^&*()[]{}|\\:;\"'<>?/~`"
	length := cryptoRandIntn(20) + 1 // 1-20 characters
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[cryptoRandIntn(len(chars))]
	}
	return string(result)
}

func isFilesystemSafeChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' || r == '-' || r == '.'
}

// AccountKeyFromFilename answers which own-account a statement belongs to, or
// none at all. An empty answer is meaningful — it is what routes a file to the
// "unknown" output — so a whole-basename fallback would be actively wrong here.
// would be actively wrong here.
func TestAccountKeyFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "CAMT export names the account after the format prefix",
			filename: "CAMT.053_54293249_2026-04-01_2026-04-30_1.xml",
			want:     "54293249",
		},
		{
			name:     "lowercase camt prefix is the same export",
			filename: "camt.053_54293249_2026-04-01_2026-04-30_1.xml",
			want:     "54293249",
		},
		{
			name:     "full path resolves on the base name only",
			filename: "/tmp/e-documents/CAMT.053_53153547_2026-05-01_2026-05-31_1.xml",
			want:     "53153547",
		},
		{
			name:     "PDF statement leads with the account number",
			filename: "54293249_2026-05-01_E100_96411.pdf",
			want:     "54293249",
		},
		{
			name:     "the 053 of the CAMT prefix is never the account",
			filename: "CAMT.053_54293250_2026-07-01_2026-07-31_1.xml",
			want:     "54293250",
		},
		{
			name:     "a name carrying no account number yields none",
			filename: "releves.csv",
			want:     "",
		},
		{
			name:     "a short leading number is a sequence, not an account",
			filename: "1_statement.csv",
			want:     "",
		},
		{
			name:     "digits later in the name are not the account",
			filename: "statement_54293249.csv",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AccountKeyFromFilename(tt.filename))
		})
	}
}

// Statement exports are routinely named by date, and a compact date is a run
// of eight digits in exactly the position an account number occupies. Reading
// one as an account splits a single account into one CSV per month — the
// mirror image of the bug per-account output exists to prevent.
func TestAccountKeyFromFilename_CompactDateIsNotAnAccount(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"compact date leading a statement name", "20260401_releve.pdf", ""},
		{"another month of the same account", "20260501_releve.pdf", ""},
		{"a date-like number that is not a valid date is an account", "20261301_releve.pdf", "20261301"},
		{"an eight-digit account number that is not a date", "54293249_2026-05-01_E100.pdf", "54293249"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AccountKeyFromFilename(tt.filename))
		})
	}
}

// A statement names its account by IBAN, but file names carry only the bank's
// short account number. Both must resolve to the same key or one account ends
// up split across two CSVs depending on how each file happened to be named.
func TestAccountKeyFromIBAN(t *testing.T) {
	tests := []struct {
		name string
		iban string
		want string
	}{
		{
			name: "Swiss IBAN ends in the number its file names carry",
			iban: "CH1700767000K54293249",
			want: "54293249",
		},
		{
			name: "another account of the same bank",
			iban: "CH6000767000Z53153547",
			want: "53153547",
		},
		{
			name: "spacing is how humans write IBANs",
			iban: "CH17 0076 7000 K542 9324 9",
			want: "54293249",
		},
		{
			name: "a trailing digit run is taken whole",
			iban: "DE89370400440532013000",
			want: "89370400440532013000",
		},
		{
			name: "an IBAN ending in letters keeps its whole identifier",
			iban: "CH930024024044090701E",
			want: "CH930024024044090701E",
		},
		{
			name: "no identifier, no key",
			iban: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AccountKeyFromIBAN(tt.iban))
		})
	}
}
