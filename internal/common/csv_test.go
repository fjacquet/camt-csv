package common

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
)

// TestCSVRow represents a test CSV row for gocsv unmarshaling
type TestCSVRow struct {
	Name     string `csv:"Name"`
	Age      string `csv:"Age"`
	Email    string `csv:"Email"`
	Country  string `csv:"Country"`
}

func init() {
	// Setup a test logger
	log = logrus.New()
	log.SetLevel(logrus.DebugLevel)
}

func TestReadCSVFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "csv-test")
	assert.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a test CSV file
	csvContent := `Name,Age,Email,Country
John Doe,30,john@example.com,USA
Jane Smith,25,jane@example.com,Canada
,,,
Bob Johnson,42,bob@example.com,UK`

	testCSVPath := filepath.Join(tempDir, "test.csv")
	err = os.WriteFile(testCSVPath, []byte(csvContent), 0644)
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

func TestWriteTransactionsToCSV(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "csv-test")
	assert.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Create sample transactions
	transactions := []models.Transaction{
		{
			Date:        "01.01.2023",
			ValueDate:   "01.01.2023",
			Description: "Test Transaction 1",
			Amount:      "100.00",
			Currency:    "CHF",
			CreditDebit: "DBIT",
			Payee:       "Test Vendor",
		},
		{
			Date:        "02.01.2023",
			ValueDate:   "02.01.2023",
			Description: "Test Transaction 2",
			Amount:      "200.00",
			Currency:    "EUR",
			CreditDebit: "CRDT",
			Payer:       "Test Customer",
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
	assert.Contains(t, csvContent, "Test Transaction 1", "Output CSV should contain first transaction description")
	assert.Contains(t, csvContent, "Test Transaction 2", "Output CSV should contain second transaction description")
	
	// Test with an invalid path
	err = WriteTransactionsToCSV(transactions, "/invalid/path/output.csv")
	assert.Error(t, err, "WriteTransactionsToCSV should return an error for an invalid path")
}

func TestGeneralizedConvertToCSV(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "csv-test")
	assert.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a test CSV file
	csvContent := `Name,Age,Email,Country
John Doe,30,john@example.com,USA
Jane Smith,25,jane@example.com,Canada
Bob Johnson,42,bob@example.com,UK`

	inputPath := filepath.Join(tempDir, "input.csv")
	err = os.WriteFile(inputPath, []byte(csvContent), 0644)
	assert.NoError(t, err, "Failed to write test CSV file")

	// Define the parser functions for testing
	parseFunc := func(string) ([]models.Transaction, error) {
		// Return simple test transactions
		return []models.Transaction{
			{
				Date:        "01.01.2023",
				Description: "Test Transaction",
				Amount:      "100.00",
			},
		}, nil
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
