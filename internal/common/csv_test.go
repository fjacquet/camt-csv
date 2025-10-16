package common

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

// TestCSVRow represents a test CSV row for gocsv unmarshaling
type TestCSVRow struct {
	Name    string `csv:"Name"`
	Age     string `csv:"Age"`
	Email   string `csv:"Email"`
	Country string `csv:"Country"`
}

func TestReadCSVFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "csv-test")
	assert.NoError(t, err, "Failed to create temp directory")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test CSV file
	csvContent := `Name,Age,Email,Country
John Doe,30,john@example.com,USA
Jane Smith,25,jane@example.com,Canada
,,,
Bob Johnson,42,bob@example.com,UK`

	testCSVPath := filepath.Join(tempDir, "test.csv")
	err = os.WriteFile(testCSVPath, []byte(csvContent), 0600)
	assert.NoError(t, err, "Failed to write test CSV file")

	// Test reading the CSV file
	rows, err := ReadCSVFile[TestCSVRow](testCSVPath)
	assert.NoError(t, err, "ReadCSVFile should not return an error")
	assert.Len(t, rows, 4, "ReadCSVFile should read all 4 rows including empty row")

	// Verify the contents of the rows
	assert.Equal(t, "John Doe", rows[0].Name)
	assert.Equal(t, "30", rows[0].Age)
	assert.Equal(t, "john@example.com", rows[0].Email)
	assert.Equal(t, "USA", rows[0].Country)

	assert.Equal(t, "Jane Smith", rows[1].Name)
	assert.Equal(t, "25", rows[1].Age)
	assert.Equal(t, "jane@example.com", rows[1].Email)
	assert.Equal(t, "Canada", rows[1].Country)

	// Testing empty row
	assert.Equal(t, "", rows[2].Name)
	assert.Equal(t, "", rows[2].Age)
	assert.Equal(t, "", rows[2].Email)
	assert.Equal(t, "", rows[2].Country)

	// Test with a non-existent file
	_, err = ReadCSVFile[TestCSVRow]("non-existent-file.csv")
	assert.Error(t, err, "ReadCSVFile should return an error for a non-existent file")
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

func TestWriteTransactionsToCSV(t *testing.T) {
	setupTestCategorizer(t)
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "csv-test")
	assert.NoError(t, err, "Failed to create temp directory")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create test transactions
	transactions := []models.Transaction{
		{
			Date:        "2023-01-01",
			Description: "Transaction 1",
			Amount:      models.ParseAmount("100.00"),
			Currency:    "CHF",
			CreditDebit: "DBIT",
		},
		{
			Date:        "2023-01-02",
			Description: "Transaction 2",
			Amount:      models.ParseAmount("200.00"),
			Currency:    "CHF",
			CreditDebit: "CRDT",
		},
	}

	// Write transactions to CSV
	outputPath := filepath.Join(tempDir, "output.csv")
	err = WriteTransactionsToCSV(transactions, outputPath)
	assert.NoError(t, err, "WriteTransactionsToCSV should not return an error")

	// Read the generated file to verify
	content, err := os.ReadFile(outputPath)
	assert.NoError(t, err, "Failed to read output CSV file")

	// Check if the file contains the essential headers and transaction data
	csvContent := string(content)
	assert.Contains(t, csvContent, "Date", "Output CSV should contain Date header")
	assert.Contains(t, csvContent, "Amount", "Output CSV should contain Amount header")
	assert.Contains(t, csvContent, "Currency", "Output CSV should contain Currency header")
	assert.Contains(t, csvContent, "Transaction 1", "Output CSV should contain first transaction description")
	assert.Contains(t, csvContent, "Transaction 2", "Output CSV should contain second transaction description")

	// Test with an invalid path
	err = WriteTransactionsToCSV(transactions, "/invalid/path/output.csv")
	assert.Error(t, err, "WriteTransactionsToCSV should return an error for an invalid path")
}

func TestGeneralizedConvertToCSV(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "csv-test")
	assert.NoError(t, err, "Failed to create temp directory")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test CSV file
	csvContent := `Name,Age,Email,Country
John Doe,30,john@example.com,USA
Jane Smith,25,jane@example.com,Canada
Bob Johnson,42,bob@example.com,UK`

	inputPath := filepath.Join(tempDir, "input.csv")
	err = os.WriteFile(inputPath, []byte(csvContent), 0600)
	assert.NoError(t, err, "Failed to write test CSV file")

	// Define the parser functions for testing
	parseFunc := func(string) ([]models.Transaction, error) {
		// Return simple test transactions
		transaction := models.Transaction{
			Date:        "2023-01-01",
			Description: "Test Transaction",
			Amount:      models.ParseAmount("100.00"),
			Currency:    "CHF",
		}
		return []models.Transaction{transaction}, nil
	}

	validateFunc := func(string) (bool, error) {
		return true, nil
	}

	// Test the generalized conversion
	outputPath := filepath.Join(tempDir, "output.csv")
	err = GeneralizedConvertToCSV(inputPath, outputPath, parseFunc, validateFunc)
	assert.NoError(t, err, "GeneralizedConvertToCSV should not return an error")

	// Test with an invalid validate function
	invalidValidateFunc := func(string) (bool, error) {
		return false, nil
	}

	err = GeneralizedConvertToCSV(inputPath, outputPath, parseFunc, invalidValidateFunc)
	assert.Error(t, err, "GeneralizedConvertToCSV should return an error when validation fails")
}

func TestExportTransactionsToCSV(t *testing.T) {
	tempDir := t.TempDir()
	csvFile := filepath.Join(tempDir, "export.csv")

	// Create sample transactions
	transactions := []models.Transaction{
		{
			Date:        "2023-01-01",
			Description: "Test Debit",
			Amount:      models.ParseAmount("123.45"),
			Currency:    "CHF",
			CreditDebit: "DBIT",
		},
		{
			Date:        "2023-01-02",
			Description: "Test Credit",
			Amount:      models.ParseAmount("-67.89"),
			Currency:    "EUR",
			CreditDebit: "CRDT",
		},
	}

	err := ExportTransactionsToCSV(transactions, csvFile)
	assert.NoError(t, err, "ExportTransactionsToCSV should not return an error")

	// Read the output file and verify content
	content, err := os.ReadFile(csvFile)
	assert.NoError(t, err, "Should be able to read exported CSV file")
	csvStr := string(content)

	// Check for expected fields in the CSV
	assert.Contains(t, csvStr, "Date")
	assert.Contains(t, csvStr, "Description")
	assert.Contains(t, csvStr, "Amount")
	assert.Contains(t, csvStr, "Currency")

	// Check transaction data is present
	assert.Contains(t, csvStr, "01.01.2023")
	assert.Contains(t, csvStr, "Test Debit")
	assert.Contains(t, csvStr, "123.45")
	assert.Contains(t, csvStr, "CHF")
}

func TestSetLogger(t *testing.T) {
	// Create a custom logger
	customLogger := logrus.New()
	customLogger.SetLevel(logrus.WarnLevel)

	// Set the custom logger
	SetLogger(customLogger)

	// Verify that the logger was set
	assert.Equal(t, logrus.WarnLevel, log.GetLevel(), "Logger should be set to the custom logger")

	// Test with nil logger (should not change the current logger)
	originalLogger := log
	SetLogger(nil)
	assert.Equal(t, originalLogger, log, "SetLogger with nil should not change the logger")
}
