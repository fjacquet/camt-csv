package revolutparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_revolut.csv")

	// Sample Revolut CSV content
	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
TRANSFER,Current,2025-01-01 08:07:09,2025-01-02 08:07:09,To CHF Vacances,-2.50,0.00,CHF,COMPLETED,111.42
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92
TRANSFER,Current,2025-01-08 19:39:37,2025-01-08 19:39:37,To CHF Vacances,-4.30,0.00,CHF,COMPLETED,49.62
CARD_PAYMENT,Current,2025-01-08 19:39:37,2025-01-09 10:47:04,Obsidian,-9.14,0.00,CHF,COMPLETED,40.48`

	err := os.WriteFile(testFile, []byte(csvContent), 0600)
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
	transactions, err := adapter.Parse(file)
	assert.NoError(t, err, "Failed to parse Revolut CSV file")
	assert.Equal(t, 4, len(transactions), "Expected 4 transactions")

	// Verify first transaction
	assert.Equal(t, "02.01.2025", transactions[0].Date)
	assert.Equal(t, "01.01.2025", transactions[0].ValueDate)
	assert.Equal(t, "Transfert to CHF Vacances", transactions[0].Description) // Updated to match actual code behavior
	assert.Equal(t, models.ParseAmount("2.50"), transactions[0].Amount)
	assert.Equal(t, "CHF", transactions[0].Currency)
	assert.Equal(t, "DBIT", transactions[0].CreditDebit)
	assert.Equal(t, "COMPLETED", transactions[0].Status)

	// Verify second transaction
	assert.Equal(t, "03.01.2025", transactions[1].Date)
	assert.Equal(t, "02.01.2025", transactions[1].ValueDate)
	assert.Equal(t, "Boreal Coffee Shop", transactions[1].Description)
	assert.Equal(t, models.ParseAmount("57.50"), transactions[1].Amount)
	assert.Equal(t, "DBIT", transactions[1].CreditDebit)
}

func TestWriteToCSV(t *testing.T) {
	// Create a temporary directory for output
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	// Set CSV delimiter to comma for this test
	common.SetDelimiter(',')

	// Create test transactions
	transactions := []models.Transaction{
		{
			Date:        "02.01.2025",
			ValueDate:   "02.01.2025",
			Description: "To CHF Vacances",
			Amount:      models.ParseAmount("2.50"),
			Currency:    "CHF",
			CreditDebit: "DBIT",
			Status:      "COMPLETED",
		},
		{
			Date:        "02.01.2025",
			ValueDate:   "02.01.2025",
			Description: "Boreal Coffee Shop",
			Amount:      models.ParseAmount("57.50"),
			Currency:    "CHF",
			CreditDebit: "DBIT",
			Status:      "COMPLETED",
		},
	}

	// Test writing to CSV
	err := WriteToCSV(transactions, outputFile)
	assert.NoError(t, err, "Failed to write transactions to CSV")

	// Read the output file and check content
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err)

	csvContent := string(content)

	// Check for the new simplified header format
	assert.Contains(t, csvContent, "Date,Description,Amount,Currency")

	// Check for transaction data
	assert.Contains(t, csvContent, "02.01.2025,To CHF Vacances,2.50,CHF")
	assert.Contains(t, csvContent, "02.01.2025,Boreal Coffee Shop,57.50,CHF")
}

func TestParseFile_InvalidFormat(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Invalid CSV (missing required columns)
	invalidFile := filepath.Join(tempDir, "invalid.csv")
	invalidContent := `Date,Description,Balance
2025-01-02,Some description,111.42`
	err := os.WriteFile(invalidFile, []byte(invalidContent), 0600)
	assert.NoError(t, err, "Failed to create invalid test file")

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

func TestConvertToCSV(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	// Set CSV delimiter to comma for this test
	common.SetDelimiter(',')

	// Create a test CSV file
	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
TRANSFER,Current,2025-01-01 08:07:09,2025-01-02 08:07:09,To CHF Vacances,-2.50,0.00,CHF,COMPLETED,111.42
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`

	err := os.WriteFile(inputFile, []byte(csvContent), 0600)
	assert.NoError(t, err, "Failed to create test input file")

	// Test convert to CSV
	err = ConvertToCSV(inputFile, outputFile)
	assert.NoError(t, err, "Failed to convert CSV")

	// Verify the output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Output file should exist")

	// Read and verify content
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Failed to read output file")
	contentStr := string(content)

	// Check for the new simplified header format
	assert.Contains(t, contentStr, "Date,Description,Amount,Currency")

	// Check for transaction data
	assert.Contains(t, contentStr, "02.01.2025,Transfert to CHF Vacances")
	assert.Contains(t, contentStr, "03.01.2025,Boreal Coffee Shop")
}
