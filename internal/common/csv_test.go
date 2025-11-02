package common

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

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
	logger := logging.NewLogrusAdapter("info", "text")
	rows, err := ReadCSVFile[TestCSVRow](testCSVPath, logger)
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
	_, err = ReadCSVFile[TestCSVRow]("non-existent-file.csv", logger)
	assert.Error(t, err, "ReadCSVFile should return an error for a non-existent file")
}

// setupTestCategorizer removed - categorizer now uses dependency injection
// Tests should create their own categorizer instances as needed

func TestWriteTransactionsToCSV(t *testing.T) {
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
			Date:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "Transaction 1",
			Amount:      models.ParseAmount("100.00"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeDebit,
		},
		{
			Date:        time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			Description: "Transaction 2",
			Amount:      models.ParseAmount("200.00"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
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
			Date:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
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
			Date:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			ValueDate:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "Test Debit",
			Amount:      models.ParseAmount("123.45"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeDebit,
		},
		{
			Date:        time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			ValueDate:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			Description: "Test Credit",
			Amount:      models.ParseAmount("-67.89"),
			Currency:    "EUR",
			CreditDebit: models.TransactionTypeCredit,
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
	// Note: Now using custom MarshalCSV method, so dates are in DD.MM.YYYY format
	assert.Contains(t, csvStr, "01.01.2023")
	assert.Contains(t, csvStr, "Test Debit")
	assert.Contains(t, csvStr, "123.45")
	assert.Contains(t, csvStr, "CHF")
}

// TestSetLogger removed - common package no longer uses global logging
// Logging is now handled through dependency injection
