package pdfparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"
	"fjacquet/camt-csv/internal/categorizer"

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
	os.WriteFile(categoriesFile, []byte("[]"), 0644)
	os.WriteFile(creditorsFile, []byte("{}"), 0644)
	os.WriteFile(debitorsFile, []byte("{}"), 0644)
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
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock PDF file (just a text file with .pdf extension for testing)
	validFile := filepath.Join(tempDir, "valid.pdf")
	invalidFile := filepath.Join(tempDir, "invalid.txt")
	
	// Create a valid PDF-like file and an invalid file
	err = os.WriteFile(validFile, []byte("%PDF-1.5\nSome PDF content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write valid test file: %v", err)
	}
	err = os.WriteFile(invalidFile, []byte("This is not a PDF file"), 0644)
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
	os.Setenv("TEST_ENV", "1")
	defer os.Unsetenv("TEST_ENV")
	
	// Create temp directories
	tempDir := filepath.Join(os.TempDir(), "pdf-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock PDF file with transaction-like content
	mockPDFFile := filepath.Join(tempDir, "statement.pdf")
	err = os.WriteFile(mockPDFFile, []byte("%PDF-1.5\nBank Statement"), 0644)
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
	
	// Check for expected CSV format
	csvContent := string(data)
	assert.Contains(t, csvContent, "Date,Description,Amount,Currency")
	assert.Contains(t, csvContent, "2023-01-01,Coffee Shop Test,15.50,EUR")
}

func TestWriteToCSV(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "transactions.csv")
	
	// Create test transactions
	transactions := []models.Transaction{
		{
			Date:          "2023-01-01",
			Description:   "Coffee Shop Purchase Card Payment REF123456",
			Amount:        models.ParseAmount("100.00"),
			Currency:      "EUR",
			EntryReference: "REF123456",
			CreditDebit:   "DBIT",
		},
		{
			Date:          "2023-01-02",
			Description:   "Salary Payment Incoming Transfer SAL987654",
			Amount:        models.ParseAmount("1000.00"),
			Currency:      "EUR",
			EntryReference: "SAL987654",
			CreditDebit:   "CRDT",
		},
	}
	
	// Write to CSV
	err := WriteToCSV(transactions, outputFile)
	assert.NoError(t, err, "Failed to write to CSV")
	
	// Read the file content
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Failed to read CSV file")
	
	// Check that the CSV contains our test data
	csvContent := string(content)
	assert.Contains(t, csvContent, "Date,Description,Amount,Currency")
	assert.Contains(t, csvContent, "2023-01-01,Coffee Shop Purchase")
	assert.Contains(t, csvContent, "2023-01-02,Salary Payment")
	assert.Contains(t, csvContent, "100.00,EUR")
	assert.Contains(t, csvContent, "1000.00,EUR")
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
