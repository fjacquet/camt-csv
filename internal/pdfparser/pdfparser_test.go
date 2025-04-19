package pdfparser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Setup a test logger
	log = logrus.New()
	log.SetLevel(logrus.DebugLevel)
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

	// Create a mock PDF file and output CSV path
	mockPDFFile := filepath.Join(tempDir, "statement.pdf")
	mockCSVFile := filepath.Join(tempDir, "output.csv")
	
	// Create a minimal PDF-like file
	err = os.WriteFile(mockPDFFile, []byte("%PDF-1.5\nSome PDF content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock PDF file: %v", err)
	}

	// Test the ConvertToCSV function
	err = ConvertToCSV(mockPDFFile, mockCSVFile)
	assert.NoError(t, err)
	
	// Check that the CSV file was created
	_, err = os.Stat(mockCSVFile)
	assert.NoError(t, err)
}

func TestWriteToCSV(t *testing.T) {
	// Create temp directories
	tempDir := filepath.Join(os.TempDir(), "pdf-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test transactions
	transactions := createMockTransactions()

	// Define an output CSV file
	csvFile := filepath.Join(tempDir, "transactions.csv")

	// Test the WriteToCSV function
	err = WriteToCSV(transactions, csvFile)
	assert.NoError(t, err)

	// Verify that the CSV file was created
	_, err = os.Stat(csvFile)
	assert.NoError(t, err)

	// Read the CSV file and check its content
	content, err := os.ReadFile(csvFile)
	assert.NoError(t, err)
	
	// Check that the CSV contains our test data
	csvContent := string(content)
	assert.Contains(t, csvContent, "Date,ValueDate,Description")        // Header
	assert.Contains(t, csvContent, "2023-01-01,,Coffee Shop Purchase")   // First transaction description
	assert.Contains(t, csvContent, "2023-01-02,,Salary Payment")         // Second transaction description
	assert.Contains(t, csvContent, "REF123456")                          // Entry reference for first transaction
	assert.Contains(t, csvContent, "SAL987654")                          // Entry reference for second transaction
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
