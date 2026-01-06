package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
)

// TestExtractAccountFromCAMTFilename tests basic CAMT filename parsing
func TestExtractAccountFromCAMTFilename(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		expectedID     string
		expectedSource string
	}{
		{
			name:           "Valid CAMT XML filename",
			filename:       "CAMT.053_54293249_2025-04-01_2025-04-30_1.xml",
			expectedID:     "54293249",
			expectedSource: "filename",
		},
		{
			name:           "Valid CAMT CSV filename",
			filename:       "CAMT.053_12345678_2024-01-01_2024-01-31_2.csv",
			expectedID:     "12345678",
			expectedSource: "filename",
		},
		{
			name:           "CAMT filename with path",
			filename:       "/path/to/CAMT.053_87654321_2023-12-01_2023-12-31_1.xml",
			expectedID:     "87654321",
			expectedSource: "filename",
		},
		{
			name:           "Invalid CAMT filename pattern",
			filename:       "CAMT.053_invalid_pattern.xml",
			expectedID:     "CAMT.053_invalid_pattern",
			expectedSource: "default",
		},
		{
			name:           "Non-CAMT filename",
			filename:       "some_other_file.csv",
			expectedID:     "some_other_file",
			expectedSource: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAccountFromCAMTFilename(tt.filename)
			assert.Equal(t, tt.expectedID, result.ID)
			assert.Equal(t, tt.expectedSource, result.Source)
		})
	}
}

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

// TestExtractAccountFromPDFContent tests PDF content account extraction
func TestExtractAccountFromPDFContent(t *testing.T) {
	tests := []struct {
		name           string
		transactions   []models.Transaction
		expectedID     string
		expectedSource string
	}{
		{
			name: "Transaction with IBAN",
			transactions: []models.Transaction{
				{IBAN: "CH93 0076 2011 6238 5295 7"},
			},
			expectedID:     "23852957",
			expectedSource: "content",
		},
		{
			name: "Transaction with account servicer",
			transactions: []models.Transaction{
				{AccountServicer: "BANK123"},
			},
			expectedID:     "BANK123",
			expectedSource: "content",
		},
		{
			name:           "No account information",
			transactions:   []models.Transaction{{}},
			expectedID:     "PDF",
			expectedSource: "default",
		},
		{
			name:           "Empty transactions",
			transactions:   []models.Transaction{},
			expectedID:     "PDF",
			expectedSource: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAccountFromPDFContent(tt.transactions)
			assert.Equal(t, tt.expectedID, result.ID)
			assert.Equal(t, tt.expectedSource, result.Source)
		})
	}
}

// TestExtractAccountFromSelmaContent tests Selma content account extraction
func TestExtractAccountFromSelmaContent(t *testing.T) {
	tests := []struct {
		name           string
		transactions   []models.Transaction
		expectedID     string
		expectedSource string
	}{
		{
			name: "Transaction with IBAN",
			transactions: []models.Transaction{
				{IBAN: "CH93 0076 2011 6238 5295 7"},
			},
			expectedID:     "23852957",
			expectedSource: "content",
		},
		{
			name: "Transaction with fund",
			transactions: []models.Transaction{
				{Fund: "SELMA_FUND_123"},
			},
			expectedID:     "SELMA_FUND_123",
			expectedSource: "content",
		},
		{
			name: "Transaction with account servicer",
			transactions: []models.Transaction{
				{AccountServicer: "SELMA_BANK"},
			},
			expectedID:     "SELMA_BANK",
			expectedSource: "content",
		},
		{
			name:           "No account information",
			transactions:   []models.Transaction{{}},
			expectedID:     "SELMA",
			expectedSource: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAccountFromSelmaContent(tt.transactions)
			assert.Equal(t, tt.expectedID, result.ID)
			assert.Equal(t, tt.expectedSource, result.Source)
		})
	}
}

// Property-based test for CAMT filename account identification
// **Feature: parser-enhancements, Property 13: Account identification from filenames**
// **Validates: Requirements 6.1**
func TestProperty_AccountIdentificationFromFilenames(t *testing.T) {
	// Property: For any CAMT filename following the pattern "CAMT.053_{account}_{dates}_{sequence}.csv",
	// the account number should be correctly extracted

	const iterations = 100

	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random valid account number (6-10 digits)
			accountLength := cryptoRandIntn(5) + 6 // 6-10 digits
			account := generateRandomDigits(accountLength)

			// Generate random valid dates
			startDate := generateRandomDate()
			endDate := startDate.AddDate(0, 1, 0) // Add one month

			// Generate random sequence number (1-9)
			sequence := cryptoRandIntn(9) + 1

			// Generate random file extension
			extensions := []string{"xml", "csv"}
			ext := extensions[cryptoRandIntn(len(extensions))]

			// Construct CAMT filename
			filename := fmt.Sprintf("CAMT.053_%s_%s_%s_%d.%s",
				account,
				startDate.Format("2006-01-02"),
				endDate.Format("2006-01-02"),
				sequence,
				ext)

			// Test the extraction
			result := ExtractAccountFromCAMTFilename(filename)

			// Property assertions
			assert.Equal(t, account, result.ID,
				"Account ID should match the generated account number for filename: %s", filename)
			assert.Equal(t, "filename", result.Source,
				"Source should be 'filename' for valid CAMT pattern: %s", filename)
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

// Property-based test for filename-based account extraction
// **Feature: parser-enhancements, Property 13: Account identification from filenames**
// **Validates: Requirements 6.1**
func TestProperty_FilenameAccountExtraction(t *testing.T) {
	// Property: For any filename, ExtractAccountFromFilename should return a valid AccountIdentifier

	const iterations = 100

	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random filename (mix of CAMT and non-CAMT)
			var filename string
			if cryptoRandFloat32() < 0.5 {
				// Generate valid CAMT filename
				account := generateRandomDigits(8)
				startDate := generateRandomDate()
				endDate := startDate.AddDate(0, 1, 0)
				sequence := cryptoRandIntn(9) + 1
				ext := []string{"xml", "csv"}[cryptoRandIntn(2)]
				filename = fmt.Sprintf("CAMT.053_%s_%s_%s_%d.%s",
					account, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), sequence, ext)
			} else {
				// Generate random non-CAMT filename
				filename = generateRandomFilename()
			}

			// Test extraction
			result := ExtractAccountFromFilename(filename)

			// Property assertions
			assert.NotEmpty(t, result.ID, "Account ID should not be empty for filename: %s", filename)
			assert.Contains(t, []string{"filename", "default"}, result.Source,
				"Source should be either 'filename' or 'default' for filename: %s", filename)

			// ID should be filesystem-safe
			for _, r := range result.ID {
				assert.True(t, isFilesystemSafeChar(r),
					"Character '%c' in ID '%s' should be filesystem-safe (filename: %s)", r, result.ID, filename)
			}
		})
	}
}

// Helper functions for property-based testing

// cryptoRandFloat32 returns a random float32 in [0, 1) using crypto/rand
func cryptoRandFloat32() float32 {
	max := big.NewInt(1000000)
	result, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return float32(result.Int64()) / 1000000.0
}

func generateRandomDigits(length int) string {
	digits := "0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = digits[cryptoRandIntn(len(digits))]
	}
	return string(result)
}

func generateRandomDate() time.Time {
	// Generate random date between 2020 and 2030
	year := cryptoRandIntn(11) + 2020
	month := cryptoRandIntn(12) + 1
	day := cryptoRandIntn(28) + 1 // Use 28 to avoid month-specific day issues
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
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

func generateRandomFilename() string {
	// Generate random filename with various patterns
	patterns := []string{
		"document_%s.pdf",
		"statement_%s.csv",
		"export_%s.xml",
		"data_%s_%s.txt",
		"%s_report.xlsx",
		"file_%s.json",
	}

	pattern := patterns[cryptoRandIntn(len(patterns))]

	// Generate random parts
	part1 := generateRandomString(5, 10)
	part2 := generateRandomString(3, 8)

	if strings.Count(pattern, "%s") == 2 {
		return fmt.Sprintf(pattern, part1, part2)
	}
	return fmt.Sprintf(pattern, part1)
}

func generateRandomString(minLen, maxLen int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := cryptoRandIntn(maxLen-minLen+1) + minLen
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

// TestExtractAccountFromFilename tests the generic filename extraction function
func TestExtractAccountFromFilename(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		expectedID     string
		expectedSource string
	}{
		{
			name:           "CAMT filename",
			filename:       "CAMT.053_12345678_2025-01-01_2025-01-31_1.xml",
			expectedID:     "12345678",
			expectedSource: "filename",
		},
		{
			name:           "Non-CAMT filename",
			filename:       "statement_export.csv",
			expectedID:     "statement_export",
			expectedSource: "default",
		},
		{
			name:           "Filename with path",
			filename:       "/path/to/document.pdf",
			expectedID:     "document",
			expectedSource: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAccountFromFilename(tt.filename)
			assert.Equal(t, tt.expectedID, result.ID)
			assert.Equal(t, tt.expectedSource, result.Source)
		})
	}
}
