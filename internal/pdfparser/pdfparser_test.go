package pdfparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Setup a test logger
	log = logrus.New()
	log.SetLevel(logrus.DebugLevel)
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
	store := store.NewCategoryStore(categoriesFile, creditorsFile, debitorsFile)
	categorizer.SetTestCategoryStore(store)
	t.Cleanup(func() {
		categorizer.SetTestCategoryStore(nil)
	})
}

func TestValidateFormat(t *testing.T) {
	// Setup mock PDF extraction
	cleanup := mockPDFExtraction()
	defer cleanup()

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

	// Create a mock PDF file (just a text file with .pdf extension for testing)
	validFile := filepath.Join(tempDir, "valid.pdf")
	invalidFile := filepath.Join(tempDir, "invalid.txt")

	// Create a valid PDF-like file and an invalid file
	err = os.WriteFile(validFile, []byte("%PDF-1.5\nSome PDF content"), 0600)
	if err != nil {
		t.Fatalf("Failed to write valid test file: %v", err)
	}
	err = os.WriteFile(invalidFile, []byte("This is not a PDF file"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	// Test valid file
	valid, err := ValidateFormat(validFile)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test invalid file
	valid, err = ValidateFormat(invalidFile)
	assert.NoError(t, err)
	assert.False(t, valid)

	// Test file that doesn't exist
	valid, err = ValidateFormat(filepath.Join(tempDir, "nonexistent.pdf"))
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestParseFile(t *testing.T) {
	// Setup mock PDF extraction
	cleanup := mockPDFExtraction()
	defer cleanup()

	// Set test environment variable to enable mock transactions
	if err := os.Setenv("TEST_ENV", "1"); err != nil {
		t.Fatalf("Failed to set TEST_ENV: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_ENV"); err != nil {
			t.Logf("Failed to unset TEST_ENV: %v", err)
		}
	}()

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

	// Create a mock PDF file with transaction-like content
	mockPDFFile := filepath.Join(tempDir, "statement.pdf")
	err = os.WriteFile(mockPDFFile, []byte("%PDF-1.5\nBank Statement"), 0600)
	if err != nil {
		t.Fatalf("Failed to write mock PDF file: %v", err)
	}

	// With our mock in place, this should now succeed
	transactions, err := ParseFile(mockPDFFile)
	assert.NoError(t, err)
	assert.NotNil(t, transactions)
	assert.Len(t, transactions, 2) // Our mock returns 2 transactions
}

func TestConvertToCSV(t *testing.T) {
	// Initialize the test environment
	setupTestCategorizer(t)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create transactions to test with
	transactions := []models.Transaction{
		{
			Date:        "2023-01-01",
			Description: "Coffee Shop Test",
			Amount:      models.ParseAmount("15.50"),
			Currency:    "EUR",
			CreditDebit: "DBIT",
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
	assert.Contains(t, csvContent, "15.5")
	assert.Contains(t, csvContent, "EUR")
}

func TestWriteToCSV(t *testing.T) {
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
			CreditDebit:    "DBIT",
		},
		{
			Date:           "2023-01-02",
			Description:    "Salary Payment Incoming Transfer SAL987654",
			Amount:         models.ParseAmount("1000.00"),
			Currency:       "EUR",
			EntryReference: "SAL987654",
			CreditDebit:    "CRDT",
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
	assert.Contains(t, csvContent, "100")
	assert.Contains(t, csvContent, "1000")
	assert.Contains(t, csvContent, "EUR")
}

func TestSetLogger(t *testing.T) {
	// Create a new logger
	newLogger := logrus.New()
	newLogger.SetLevel(logrus.WarnLevel)

	// Set the logger
	SetLogger(newLogger)

	// Verify that the package logger is updated
	assert.Equal(t, newLogger, log)
}
