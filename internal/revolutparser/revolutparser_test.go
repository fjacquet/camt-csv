package revolutparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Set up test logger
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.DebugLevel)
	SetLogger(testLogger)
}

func TestParseFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_revolut.csv")

	// Sample Revolut CSV content
	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
TRANSFER,Current,2025-01-02 08:07:09,2025-01-02 08:07:09,To CHF Vacances,-2.50,0.00,CHF,COMPLETED,111.42
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92
TRANSFER,Current,2025-01-08 19:39:37,2025-01-08 19:39:37,To CHF Vacances,-4.30,0.00,CHF,COMPLETED,49.62
CARD_PAYMENT,Current,2025-01-08 19:39:37,2025-01-09 10:47:04,Obsidian,-9.14,0.00,CHF,COMPLETED,40.48`

	err := os.WriteFile(testFile, []byte(csvContent), 0644)
	assert.NoError(t, err, "Failed to create test file")

	// Test parsing
	transactions, err := ParseFile(testFile)
	assert.NoError(t, err, "Failed to parse Revolut CSV file")
	assert.Equal(t, 4, len(transactions), "Expected 4 transactions")

	// Verify first transaction
	assert.Equal(t, "2025-01-02", transactions[0].Date)
	assert.Equal(t, "2025-01-02", transactions[0].ValueDate)
	assert.Equal(t, "To CHF Vacances", transactions[0].Description)
	assert.Equal(t, "2.50", transactions[0].Amount)
	assert.Equal(t, "CHF", transactions[0].Currency)
	assert.Equal(t, "DBIT", transactions[0].CreditDebit)
	assert.Equal(t, "COMPLETED", transactions[0].Status)

	// Verify second transaction
	assert.Equal(t, "2025-01-02", transactions[1].Date)
	assert.Equal(t, "Boreal Coffee Shop", transactions[1].Description)
	assert.Equal(t, "57.50", transactions[1].Amount)
	assert.Equal(t, "DBIT", transactions[1].CreditDebit)
}

func TestWriteToCSV(t *testing.T) {
	// Create a temporary directory for output
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	// Create test transactions
	transactions := []struct {
		date        string
		description string
		amount      string
		currency    string
		creditDebit string
		status      string
	}{
		{"2025-01-02", "To CHF Vacances", "2.50", "CHF", "DBIT", "COMPLETED"},
		{"2025-01-02", "Boreal Coffee Shop", "57.50", "CHF", "DBIT", "COMPLETED"},
	}

	// Convert to models.Transaction
	var testTransactions []models.Transaction
	for _, tx := range transactions {
		testTransactions = append(testTransactions, models.Transaction{
			Date:        tx.date,
			ValueDate:   tx.date,
			Description: tx.description,
			Amount:      tx.amount,
			Currency:    tx.currency,
			CreditDebit: tx.creditDebit,
			Status:      tx.status,
		})
	}

	// Test writing to CSV
	err := WriteToCSV(testTransactions, outputFile)
	assert.NoError(t, err, "Failed to write transactions to CSV")

	// Verify file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Output file was not created")

	// Read back the file to verify content
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Failed to read output file")
	assert.Contains(t, string(content), "Date,Value Date,Description")
	assert.Contains(t, string(content), "2025-01-02,2025-01-02,To CHF Vacances")
	assert.Contains(t, string(content), "2025-01-02,2025-01-02,Boreal Coffee Shop")
}

func TestValidateFormat(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Valid Revolut CSV
	validFile := filepath.Join(tempDir, "valid.csv")
	validContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
TRANSFER,Current,2025-01-02 08:07:09,2025-01-02 08:07:09,To CHF Vacances,-2.50,0.00,CHF,COMPLETED,111.42`
	err := os.WriteFile(validFile, []byte(validContent), 0644)
	assert.NoError(t, err, "Failed to create valid test file")

	// Invalid CSV (missing required columns)
	invalidFile := filepath.Join(tempDir, "invalid.csv")
	invalidContent := `Date,Description,Balance
2025-01-02,Some description,111.42`
	err = os.WriteFile(invalidFile, []byte(invalidContent), 0644)
	assert.NoError(t, err, "Failed to create invalid test file")

	// Empty CSV (header only)
	emptyFile := filepath.Join(tempDir, "empty.csv")
	emptyContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance`
	err = os.WriteFile(emptyFile, []byte(emptyContent), 0644)
	assert.NoError(t, err, "Failed to create empty test file")

	// Test validation
	valid, err := ValidateFormat(validFile)
	assert.NoError(t, err, "Error validating valid file")
	assert.True(t, valid, "Valid file should be recognized as valid")

	valid, err = ValidateFormat(invalidFile)
	assert.NoError(t, err, "Error validating invalid file")
	assert.False(t, valid, "Invalid file should be recognized as invalid")

	valid, err = ValidateFormat(emptyFile)
	assert.NoError(t, err, "Error validating empty file")
	assert.False(t, valid, "Empty file should be recognized as invalid")
}

func TestConvertToCSV(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	// Sample Revolut CSV content
	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
TRANSFER,Current,2025-01-02 08:07:09,2025-01-02 08:07:09,To CHF Vacances,-2.50,0.00,CHF,COMPLETED,111.42
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`

	err := os.WriteFile(inputFile, []byte(csvContent), 0644)
	assert.NoError(t, err, "Failed to create test input file")

	// Test conversion
	err = ConvertToCSV(inputFile, outputFile)
	assert.NoError(t, err, "Failed to convert Revolut CSV to standard format")

	// Verify output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Output file was not created")

	// Read back the file to verify content
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Failed to read output file")
	assert.Contains(t, string(content), "Date,Value Date,Description")
	assert.Contains(t, string(content), "2025-01-02,2025-01-02,To CHF Vacances")
	assert.Contains(t, string(content), "2025-01-02,2025-01-02,Boreal Coffee Shop")
}
