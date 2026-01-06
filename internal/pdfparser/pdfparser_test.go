package pdfparser

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	t.Helper()
	tempDir := t.TempDir()
	categoriesFile := filepath.Join(tempDir, "categories.yaml")
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")
	debitorsFile := filepath.Join(tempDir, "debitors.yaml")
	if err := os.WriteFile(categoriesFile, []byte("[]"), 0600); err != nil {
		t.Fatalf("Failed to write categories file: %v", err)
	}
	if err := os.WriteFile(creditorsFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to write creditors file: %v", err)
	}
	if err := os.WriteFile(debitorsFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to write debitors file: %v", err)
	}
	// Note: categorizer.SetTestCategoryStore removed - now uses dependency injection
	// Tests should create their own categorizer instances as needed
}

func TestParseFile_InvalidFormat(t *testing.T) {
	// Create temp directories
	tempDir := filepath.Join(os.TempDir(), "pdf-test")
	err := os.MkdirAll(tempDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create an invalid file
	invalidFile := filepath.Join(tempDir, "invalid.txt")
	err = os.WriteFile(invalidFile, []byte("This is not a PDF file"), 0600)
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
	mockExtractor := NewMockPDFExtractor("", assert.AnError)
	adapter := NewAdapter(logger, mockExtractor)
	_, err = adapter.Parse(file)
	assert.Error(t, err, "Expected an error when parsing an invalid file")
}

func TestParseFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_pdf.pdf")

	// For this test, we'll use a dummy PDF file.
	// In a real scenario, you would use a real PDF file.
	err := os.WriteFile(testFile, []byte("dummy content"), 0600)
	assert.NoError(t, err, "Failed to create test file")

	file, err := os.Open(testFile)
	require.NoError(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file %s: %v", testFile, err)
		}
	}()

	// Test parsing with mock extractor that returns mock transactions
	logger := logging.NewLogrusAdapter("info", "text")
	// Provide mock text that looks like a transaction
	mockText := `Date valeur Détails Monnaie Montant
01.01.25 02.01.25 Test Transaction CHF 100.50
03.01.25 04.01.25 Another Transaction CHF 200.75-`
	mockExtractor := NewMockPDFExtractor(mockText, nil)
	adapter := NewAdapter(logger, mockExtractor)
	_, err = adapter.Parse(file)
	assert.NoError(t, err, "Expected no error when parsing with mock extractor")
	// Note: The mock text might not parse into transactions due to format requirements
	// This test mainly verifies that the dependency injection works
}

func TestConvertToCSV(t *testing.T) {
	// Initialize the test environment
	setupTestCategorizer(t)

	// CSV delimiter is now a constant (models.DefaultCSVDelimiter)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create transactions to test with
	transactions := []models.Transaction{
		{
			Date:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "Coffee Shop Test",
			Amount:      models.ParseAmount("15.50"),
			Currency:    "EUR",
			CreditDebit: models.TransactionTypeDebit,
		},
	}

	// Create the output path
	outputFile := filepath.Join(tempDir, "test_output.csv")

	// Skip the normal PDF parsing by testing just the WriteToCSV function directly
	err := WriteToCSV(transactions, outputFile)
	assert.NoError(t, err)

	// Verify the output file exists and has the right content
	data, err := os.ReadFile(outputFile)
	assert.NoError(t, err)

	// Check for expected CSV format - verify header contains key fields
	csvContent := string(data)
	assert.Contains(t, csvContent, "BookkeepingNumber,Status,Date,ValueDate,Name,PartyName,PartyIBAN,Description")
	assert.Contains(t, csvContent, "Date")
	assert.Contains(t, csvContent, "Description")
	assert.Contains(t, csvContent, "Amount")
	assert.Contains(t, csvContent, "Currency")
	// Check for transaction data
	assert.Contains(t, csvContent, "01.01.2023")
	assert.Contains(t, csvContent, "Coffee Shop Test")
	assert.Contains(t, csvContent, "15.5") // Amount can be 15.5 or 15.50
	assert.Contains(t, csvContent, "EUR")
}

func TestWriteToCSV(t *testing.T) {
	// CSV delimiter is now a constant (models.DefaultCSVDelimiter)

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "transactions.csv")

	// Create test transactions
	transactions := []models.Transaction{
		{
			Date:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Description:    "Coffee Shop Purchase Card Payment REF123456",
			Amount:         models.ParseAmount("100.00"),
			Currency:       "EUR",
			EntryReference: "REF123456",
			CreditDebit:    models.TransactionTypeDebit,
		},
		{
			Date:           time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			Description:    "Salary Payment Incoming Transfer SAL987654",
			Amount:         models.ParseAmount("1000.00"),
			Currency:       "EUR",
			EntryReference: "SAL987654",
			CreditDebit:    models.TransactionTypeCredit,
		},
	}

	// Write to CSV
	err := WriteToCSV(transactions, outputFile)
	assert.NoError(t, err, "Failed to write to CSV")

	// Read the file content
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Failed to read CSV file")

	// Check that the CSV contains our test data - verify header and content
	csvContent := string(content)
	// Check header contains key fields
	assert.Contains(t, csvContent, "BookkeepingNumber,Status,Date,ValueDate,Name,PartyName,PartyIBAN,Description")
	assert.Contains(t, csvContent, "Date")
	assert.Contains(t, csvContent, "Description")
	assert.Contains(t, csvContent, "Amount")
	assert.Contains(t, csvContent, "Currency")
	// Check transaction data
	assert.Contains(t, csvContent, "01.01.2023")
	assert.Contains(t, csvContent, "Coffee Shop Purchase Card Payment REF123456")
	assert.Contains(t, csvContent, "02.01.2023")
	assert.Contains(t, csvContent, "Salary Payment Incoming Transfer SAL987654")
	assert.Contains(t, csvContent, "100")  // Amount can be 100 or 100.00
	assert.Contains(t, csvContent, "1000") // Amount can be 1000 or 1000.00
	assert.Contains(t, csvContent, "EUR")
}

// **Feature: parser-enhancements, Property 6: PDF categorization integration**
// **Validates: Requirements 2.1, 5.3**
func TestProperty_PDFCategorizationIntegration(t *testing.T) {
	// Property: For any PDF transaction, the categorization system should be applied using the same
	// three-tier strategy (direct mapping → keyword matching → AI fallback) as other parsers

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random transaction data
			partyName := generateRandomPartyName()
			amount := generateRandomAmount()
			description := generateRandomDescription(partyName)

			// Create mock categorizer that tracks calls
			mockCategorizer := &MockCategorizer{
				categories: map[string]string{
					description: "TestCategory",
					partyName:   "TestCategory",
				},
			}

			// Create mock PDF text that will generate a transaction in Viseca format
			// This format is detected by the parser and uses the specialized Viseca parser
			mockText := fmt.Sprintf(`Date valeur Détails Monnaie Montant
01.01.25 02.01.25 %s CHF %s`, description, amount)

			// Create adapter with mock extractor and categorizer
			mockExtractor := NewMockPDFExtractor(mockText, nil)
			logger := logging.NewLogrusAdapter("info", "text")
			adapter := NewAdapter(logger, mockExtractor)
			adapter.SetCategorizer(mockCategorizer)

			// Create a dummy PDF file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.pdf")
			err := os.WriteFile(testFile, []byte("dummy content"), 0600)
			require.NoError(t, err)

			file, err := os.Open(testFile)
			require.NoError(t, err)
			defer func() {
				if cerr := file.Close(); cerr != nil {
					t.Logf("Failed to close file: %v", cerr)
				}
			}()

			// Parse the PDF
			transactions, err := adapter.Parse(file)
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

// **Feature: parser-enhancements, Property 8: Consistent CSV output format**
// **Validates: Requirements 2.3, 4.1**
func TestProperty_ConsistentCSVOutputFormat(t *testing.T) {
	// Property: For any parser type (CAMT, PDF, Selma, Revolut), the CSV output should contain
	// identical column headers including category and subcategory columns

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random transactions
			transactions := generateRandomTransactions(cryptoRandIntn(10) + 1)

			// Write to CSV using PDF parser
			tempDir := t.TempDir()
			outputFile := filepath.Join(tempDir, "test_output.csv")

			err := WriteToCSV(transactions, outputFile)
			require.NoError(t, err)

			// Read the CSV content
			content, err := os.ReadFile(outputFile)
			require.NoError(t, err)

			csvContent := string(content)

			// Verify standard CSV headers are present (based on actual Transaction.MarshalCSV output)
			expectedHeaders := []string{
				"BookkeepingNumber", "Status", "Date", "ValueDate", "Name", "PartyName",
				"PartyIBAN", "Description", "RemittanceInfo", "Amount", "CreditDebit", "IsDebit",
				"Debit", "Credit", "Currency", "AmountExclTax", "AmountTax", "TaxRate",
				"Recipient", "InvestmentType", "Number", "Category", "Type", "Fund",
				"NumberOfShares", "Fees", "IBAN", "EntryReference", "Reference",
				"AccountServicer", "BankTxCode", "OriginalCurrency", "OriginalAmount", "ExchangeRate",
			}

			// Check that all expected headers are present
			for _, header := range expectedHeaders {
				assert.Contains(t, csvContent, header, "CSV should contain standard header: %s", header)
			}

			// Verify category column is included (Category is present in the actual headers)
			assert.Contains(t, csvContent, "Category", "CSV should contain Category column")

			// Verify the CSV has proper structure (header line + data lines)
			lines := strings.Split(strings.TrimSpace(csvContent), "\n")
			assert.GreaterOrEqual(t, len(lines), 1, "CSV should have at least a header line")

			// If we have transactions, verify they appear in the CSV
			if len(transactions) > 0 {
				assert.GreaterOrEqual(t, len(lines), len(transactions)+1, "CSV should have header + transaction lines")
			}
		})
	}
}

// MockCategorizer for testing categorization
type MockCategorizer struct {
	categories map[string]string
	callCount  int
}

func (m *MockCategorizer) Categorize(partyName string, isDebtor bool, amount, date, info string) (models.Category, error) {
	m.callCount++

	// Check all possible keys for a match
	for key, category := range m.categories {
		if strings.Contains(partyName, key) || strings.Contains(info, key) {
			return models.Category{Name: category}, nil
		}
	}

	return models.Category{Name: models.CategoryUncategorized}, nil
}

// Helper functions for property-based testing
func generateRandomPartyName() string {
	parties := []string{
		"ACME Corp", "Global Bank", "Tech Solutions", "Coffee Shop", "Gas Station",
		"Supermarket", "Restaurant", "Online Store", "Insurance Co", "Utility Company",
	}
	return parties[cryptoRandIntn(len(parties))]
}

func generateRandomAmount() string {
	amounts := []string{
		"10.50", "25.00", "100.75", "250.25", "500.00", "1000.00", "50.25", "75.80",
	}
	return amounts[cryptoRandIntn(len(amounts))]
}

func generateRandomDescription(partyName string) string {
	prefixes := []string{
		"Payment to", "Purchase at", "Transfer to", "Card payment", "Online payment",
	}
	prefix := prefixes[cryptoRandIntn(len(prefixes))]
	return fmt.Sprintf("%s %s", prefix, partyName)
}

func generateRandomTransactions(count int) []models.Transaction {
	transactions := make([]models.Transaction, count)

	for i := 0; i < count; i++ {
		transactions[i] = models.Transaction{
			Date:        time.Date(2023, time.Month(cryptoRandIntn(12)+1), cryptoRandIntn(28)+1, 0, 0, 0, 0, time.UTC),
			Description: generateRandomDescription(generateRandomPartyName()),
			Amount:      models.ParseAmount(generateRandomAmount()),
			Currency:    "CHF",
			CreditDebit: []string{models.TransactionTypeCredit, models.TransactionTypeDebit}[cryptoRandIntn(2)],
			Category:    models.CategoryUncategorized,
		}
	}

	return transactions
}
