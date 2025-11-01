package pdfparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	store := store.NewCategoryStore(categoriesFile, creditorsFile, debitorsFile)
	categorizer.SetTestCategoryStore(store)
	t.Cleanup(func() {
		categorizer.SetTestCategoryStore(nil)
	})
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

	adapter := NewAdapter()
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

	// Test parsing
	adapter := NewAdapter()
	_, err = adapter.Parse(file)
	assert.Error(t, err, "Expected an error when parsing a dummy PDF file")
}

func TestConvertToCSV(t *testing.T) {
	// Initialize the test environment
	setupTestCategorizer(t)

	// Set CSV delimiter to comma for this test
	common.SetDelimiter(',')

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create transactions to test with
	transactions := []models.Transaction{
		{
			Date:        "2023-01-01",
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
	// Set CSV delimiter to comma for this test
	common.SetDelimiter(',')

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "transactions.csv")

	// Create test transactions
	transactions := []models.Transaction{
		{
			Date:           "2023-01-01",
			Description:    "Coffee Shop Purchase Card Payment REF123456",
			Amount:         models.ParseAmount("100.00"),
			Currency:       "EUR",
			EntryReference: "REF123456",
			CreditDebit:    models.TransactionTypeDebit,
		},
		{
			Date:           "2023-01-02",
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
