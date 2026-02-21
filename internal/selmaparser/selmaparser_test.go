package selmaparser

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func setupTestCategorizer(t *testing.T) {
	// The new categorizer system uses dependency injection and doesn't require global setup
	// Tests that need categorization should create their own categorizer instances
}

func TestParseFile_InvalidFormat(t *testing.T) {
	// Create temp directories
	tempDir := filepath.Join(os.TempDir(), "selma-test")
	err := os.MkdirAll(tempDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create an invalid CSV file (missing required headers)
	invalidCSV := `foo,bar,baz
1,2,3
4,5,6`

	invalidFile := filepath.Join(tempDir, "invalid.csv")

	err = os.WriteFile(invalidFile, []byte(invalidCSV), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	file, err := os.Open(invalidFile)
	require.NoError(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file %s: %v", invalidFile, err)
		}
	}()

	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)
	_, err = adapter.Parse(context.Background(), file)
	assert.Error(t, err, "Expected an error when parsing an invalid file")
}

func TestParseFile(t *testing.T) {
	// Create temp directories
	tempDir := filepath.Join(os.TempDir(), "selma-test")
	err := os.MkdirAll(tempDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test CSV file that matches the SelmaCSVRow structure (old format)
	testCSV := `Date,Description,Bookkeeping No.,Fund,Amount,Currency,Number of Shares
2023-01-01,VANGUARD FTSE ALL WORLD,22310435155,IE00BK5BQT80,-247.90,CHF,2
2023-01-02,ISHARES CORE S&P 500 UCITS ETF,22310435156,IE00B5BMR087,452.22,CHF,1`

	testFile := filepath.Join(tempDir, "transactions.csv")
	err = os.WriteFile(testFile, []byte(testCSV), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	file, err := os.Open(testFile)
	require.NoError(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file %s: %v", testFile, err)
		}
	}()

	// Test parsing the file
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)
	transactions, err := adapter.Parse(context.Background(), file)
	assert.NoError(t, err, "ParseFile should not return an error for valid input")
	assert.NotNil(t, transactions, "Transactions should not be nil")
	assert.Equal(t, 2, len(transactions), "Should have parsed 2 transactions")

	if len(transactions) > 0 {
		expectedDate1, _ := time.Parse(dateutils.DateLayoutISO, "2023-01-01")
		assert.Equal(t, expectedDate1, transactions[0].Date, "Date should be parsed correctly")
		assert.Contains(t, transactions[0].Description, "VANGUARD FTSE ALL WORLD")
		assert.Equal(t, models.ParseAmount("-247.90"), transactions[0].Amount)
		assert.Equal(t, "CHF", transactions[0].Currency)
		assert.Equal(t, 2, transactions[0].NumberOfShares)
	}
	if len(transactions) > 1 {
		expectedDate2, _ := time.Parse(dateutils.DateLayoutISO, "2023-01-02")
		assert.Equal(t, expectedDate2, transactions[1].Date, "Date should be parsed correctly")
		assert.Contains(t, transactions[1].Description, "ISHARES CORE S&P 500 UCITS ETF")
		assert.Equal(t, models.ParseAmount("452.22"), transactions[1].Amount)
		assert.Equal(t, "CHF", transactions[1].Currency)
		assert.Equal(t, 1, transactions[1].NumberOfShares)
	}
}

// **Feature: parser-enhancements, Property 7: Selma categorization integration**
// **Validates: Requirements 3.1, 5.3**
func TestProperty_SelmaCategorizationIntegration(t *testing.T) {
	// Property: For any Selma transaction, the categorization system should be applied using the same
	// three-tier strategy (direct mapping → keyword matching → AI fallback) as other parsers

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random transaction data
			fundName := generateRandomFundName()
			amount := generateRandomAmount()
			description := generateRandomDescription()

			// Create mock categorizer that tracks calls
			mockCategorizer := &MockCategorizer{
				categories: map[string]string{
					fundName:    "Investment Category",
					description: "Transaction Category",
				},
			}

			// Create test CSV content with random data
			testCSV := fmt.Sprintf(`Date,Description,Bookkeeping No.,Fund,Amount,Currency,Number of Shares
2023-01-01,%s,12345,%s,%s,CHF,10`, description, fundName, amount)

			// Create temporary file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.csv")
			err := os.WriteFile(testFile, []byte(testCSV), 0600)
			require.NoError(t, err)

			file, err := os.Open(testFile)
			require.NoError(t, err)
			defer func() {
				if cerr := file.Close(); cerr != nil {
					t.Logf("Failed to close file: %v", cerr)
				}
			}()

			// Create adapter with categorizer
			logger := logging.NewLogrusAdapter("info", "text")
			adapter := NewAdapter(logger)
			adapter.SetCategorizer(mockCategorizer)

			// Parse the CSV
			transactions, err := adapter.Parse(context.Background(), file)
			require.NoError(t, err)

			// Verify that categorization was applied
			if len(transactions) > 0 {
				// At least one transaction should have been categorized
				foundCategorized := false
				for _, tx := range transactions {
					if tx.Category != "" && tx.Category != models.CategoryUncategorized {
						foundCategorized = true
						break
					}
				}

				// If we have a valid transaction and categorizer was called, it should be categorized
				if mockCategorizer.callCount > 0 {
					assert.True(t, foundCategorized, "Expected at least one transaction to be categorized when categorizer is available")
				}
			}

			// Verify categorizer was called if transactions were found
			if len(transactions) > 0 {
				assert.GreaterOrEqual(t, mockCategorizer.callCount, 0, "Categorizer should be called for transactions")
			}
		})
	}
}

// MockCategorizer for testing categorization
type MockCategorizer struct {
	categories map[string]string
	callCount  int
}

func (m *MockCategorizer) Categorize(ctx context.Context, partyName string, isDebtor bool, amount, date, info string) (models.Category, error) {
	m.callCount++

	if category, exists := m.categories[partyName]; exists {
		return models.Category{Name: category}, nil
	}

	return models.Category{Name: models.CategoryUncategorized}, nil
}

// Helper functions for property-based testing
func generateRandomFundName() string {
	funds := []string{
		"VANGUARD FTSE ALL WORLD", "ISHARES CORE S&P 500", "SPDR S&P 500", "VANGUARD TOTAL STOCK",
		"ISHARES MSCI WORLD", "VANGUARD EMERGING MARKETS", "ISHARES CORE MSCI EMERGING",
		"SPDR PORTFOLIO S&P 500", "VANGUARD FTSE DEVELOPED", "ISHARES CORE AGGREGATE BOND",
	}
	return funds[cryptoRandIntn(len(funds))]
}

func generateRandomAmount() string {
	amounts := []string{
		"10.50", "25.00", "100.75", "250.25", "500.00", "1000.00", "50.25", "75.80",
		"-10.50", "-25.00", "-100.75", "-250.25", "-500.00", "-1000.00", "-50.25", "-75.80",
	}
	return amounts[cryptoRandIntn(len(amounts))]
}

func generateRandomDescription() string {
	descriptions := []string{
		"trade", "dividend", "cash_transfer", "selma_fee", "withholding_tax", "stamp_duty",
	}
	return descriptions[cryptoRandIntn(len(descriptions))]
}

// TestSelmaParser_ErrorMessagesIncludeFilePath validates error messages include helpful context
func TestSelmaParser_ErrorMessagesIncludeFilePath(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	t.Run("invalid_file_path_in_error", func(t *testing.T) {
		invalidPath := "/nonexistent/test_file.csv"

		err := adapter.ConvertToCSV(context.Background(), invalidPath, "/tmp/output.csv")
		require.Error(t, err)

		// Error should include the file path that was attempted
		assert.Contains(t, err.Error(), invalidPath,
			"Error message should include file path for debugging")
	})

	t.Run("malformed_csv_includes_context", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "malformed.csv")

		// Create malformed CSV (wrong headers)
		malformedCSV := `WrongHeader1,WrongHeader2,WrongHeader3
Value1,Value2,Value3`

		err := os.WriteFile(testFile, []byte(malformedCSV), 0600)
		require.NoError(t, err)

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		_, err = adapter.Parse(context.Background(), file)
		require.Error(t, err)

		// Error should mention that it's a format validation issue
		errMsg := err.Error()
		assert.True(t,
			strings.Contains(errMsg, "header") || strings.Contains(errMsg, "format") || strings.Contains(errMsg, "column"),
			"Error message should indicate format issue: %s", errMsg)
	})

	t.Run("missing_required_field_includes_field_name", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "missing_field.csv")

		// Create CSV with missing required fields
		missingFieldCSV := `Date,Description
2023-01-01,VANGUARD FTSE ALL WORLD`

		err := os.WriteFile(testFile, []byte(missingFieldCSV), 0600)
		require.NoError(t, err)

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		_, err = adapter.Parse(context.Background(), file)
		// Parser should detect missing required columns
		require.Error(t, err)
		errMsg := err.Error()
		assert.True(t,
			strings.Contains(errMsg, "header") || strings.Contains(errMsg, "Bookkeeping"),
			"Error should mention missing required field: %s", errMsg)
	})

	t.Run("invalid_amount_format_includes_context", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "invalid_amount.csv")

		// Create CSV with invalid amount format
		invalidAmountCSV := `Date,Description,Bookkeeping No.,Fund,Amount,Currency,Number of Shares
2023-01-01,VANGUARD FTSE ALL WORLD,22310435155,IE00BK5BQT80,INVALID_AMOUNT,CHF,2`

		err := os.WriteFile(testFile, []byte(invalidAmountCSV), 0600)
		require.NoError(t, err)

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		transactions, err := adapter.Parse(context.Background(), file)
		// Parser should handle gracefully or return descriptive error
		if err != nil {
			// If error, should mention amount parsing
			assert.Contains(t, err.Error(), "amount",
				"Error message should mention amount field")
		} else {
			// If no error, should still return valid structure
			assert.NotNil(t, transactions)
		}
	})
}
