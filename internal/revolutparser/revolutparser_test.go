package revolutparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
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
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)
	transactions, err := adapter.Parse(file)
	assert.NoError(t, err, "Failed to parse Revolut CSV file")
	assert.Equal(t, 4, len(transactions), "Expected 4 transactions")

	// Verify first transaction
	assert.Equal(t, "02.01.2025", transactions[0].Date.Format(dateutils.DateLayoutEuropean))
	assert.Equal(t, "01.01.2025", transactions[0].ValueDate.Format(dateutils.DateLayoutEuropean))
	assert.Equal(t, "Transfert to CHF Vacances", transactions[0].Description) // Updated to match actual code behavior
	assert.Equal(t, models.ParseAmount("2.50"), transactions[0].Amount)
	assert.Equal(t, "CHF", transactions[0].Currency)
	assert.Equal(t, models.TransactionTypeDebit, transactions[0].CreditDebit)
	assert.Equal(t, models.StatusCompleted, transactions[0].Status)

	// Verify second transaction
	assert.Equal(t, "03.01.2025", transactions[1].Date.Format(dateutils.DateLayoutEuropean))
	assert.Equal(t, "02.01.2025", transactions[1].ValueDate.Format(dateutils.DateLayoutEuropean))
	assert.Equal(t, "Boreal Coffee Shop", transactions[1].Description)
	assert.Equal(t, models.ParseAmount("57.50"), transactions[1].Amount)
	assert.Equal(t, models.TransactionTypeDebit, transactions[1].CreditDebit)
}

func TestWriteToCSV(t *testing.T) {
	// Create a temporary directory for output
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	// CSV delimiter is now a constant (models.DefaultCSVDelimiter)

	// Create test transactions
	transactions := []models.Transaction{
		{
			Date:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			ValueDate:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			Description: "To CHF Vacances",
			Amount:      models.ParseAmount("2.50"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeDebit,
			Status:      models.StatusCompleted,
		},
		{
			Date:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			ValueDate:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			Description: "Boreal Coffee Shop",
			Amount:      models.ParseAmount("57.50"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeDebit,
			Status:      models.StatusCompleted,
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

	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)
	_, err = adapter.Parse(file)
	assert.Error(t, err, "Expected an error when parsing an invalid file")
}

func TestConvertToCSV(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	// CSV delimiter is now a constant (models.DefaultCSVDelimiter)

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

func TestParseWithCategorizer(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_revolut.csv")

	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`

	err := os.WriteFile(testFile, []byte(csvContent), 0600)
	require.NoError(t, err)

	file, err := os.Open(testFile)
	require.NoError(t, err)
	defer file.Close()

	// Mock categorizer
	mockCategorizer := &mockCategorizer{
		category: models.Category{Name: "Food & Dining"},
	}

	logger := logging.NewLogrusAdapter("info", "text")
	transactions, err := ParseWithCategorizer(file, logger, mockCategorizer)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, "Food & Dining", transactions[0].Category)
}

func TestParseWithCategorizerError(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_revolut.csv")

	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`

	err := os.WriteFile(testFile, []byte(csvContent), 0600)
	require.NoError(t, err)

	file, err := os.Open(testFile)
	require.NoError(t, err)
	defer file.Close()

	// Mock categorizer that returns error
	mockCategorizer := &mockCategorizerError{}

	logger := logging.NewLogrusAdapter("info", "text")
	transactions, err := ParseWithCategorizer(file, logger, mockCategorizer)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, models.CategoryUncategorized, transactions[0].Category)
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
		hasError bool
	}{
		{
			name: "valid format",
			content: `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`,
			expected: true,
			hasError: false,
		},
		{
			name: "missing required column",
			content: `Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`,
			expected: false,
			hasError: false,
		},
		{
			name:     "empty file",
			content:  `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance`,
			expected: false,
			hasError: false,
		},
		{
			name:     "malformed CSV",
			content:  `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\nunclosed"quote,field`,
			expected: false,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			valid, err := validateFormat(reader)
			
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, valid)
		})
	}
}

func TestBatchConvert(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")

	// Create input directory
	err := os.MkdirAll(inputDir, 0750)
	require.NoError(t, err)

	// Create valid Revolut CSV file
	validCSV := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`
	
	validFile := filepath.Join(inputDir, "valid.csv")
	err = os.WriteFile(validFile, []byte(validCSV), 0600)
	require.NoError(t, err)

	// Create invalid CSV file
	invalidCSV := `Date,Description,Balance
2025-01-02,Some description,111.42`
	
	invalidFile := filepath.Join(inputDir, "invalid.csv")
	err = os.WriteFile(invalidFile, []byte(invalidCSV), 0600)
	require.NoError(t, err)

	// Create non-CSV file
	nonCSVFile := filepath.Join(inputDir, "document.txt")
	err = os.WriteFile(nonCSVFile, []byte("not a csv"), 0600)
	require.NoError(t, err)

	// Create subdirectory (should be ignored)
	subDir := filepath.Join(inputDir, "subdir")
	err = os.MkdirAll(subDir, 0750)
	require.NoError(t, err)

	// Test batch convert
	count, err := BatchConvert(inputDir, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Only valid file should be processed

	// Verify output file exists
	outputFile := filepath.Join(outputDir, "valid-standardized.csv")
	_, err = os.Stat(outputFile)
	assert.NoError(t, err)
}

func TestWriteToCSVWithNilTransactions(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	err := WriteToCSV(nil, outputFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write nil transactions")
}

func TestPostProcessTransactions(t *testing.T) {
	transactions := []models.Transaction{
		{
			Type:        "TRANSFER",
			Description: "To CHF Vacances",
			CreditDebit: models.TransactionTypeDebit,
		},
		{
			Type:        "TRANSFER", 
			Description: "To CHF Vacances",
			CreditDebit: models.TransactionTypeCredit,
		},
		{
			Type:        "CARD_PAYMENT",
			Description: "Regular payment",
			CreditDebit: models.TransactionTypeDebit,
		},
	}

	processed := postProcessTransactions(transactions)
	
	// First transaction should be processed as debit transfer
	assert.Equal(t, "Transfert to CHF Vacances", processed[0].Description)
	assert.Equal(t, "Transfert to CHF Vacances", processed[0].Name)
	assert.Equal(t, "Transfert to CHF Vacances", processed[0].PartyName)
	assert.Equal(t, "Transfert to CHF Vacances", processed[0].Recipient)
	
	// Second transaction should be processed as credit transfer
	assert.Equal(t, "Transferred To CHF Vacances", processed[1].Description)
	assert.Equal(t, "Transferred To CHF Vacances", processed[1].Name)
	assert.Equal(t, "Transferred To CHF Vacances", processed[1].PartyName)
	assert.Equal(t, "Transferred To CHF Vacances", processed[1].Recipient)
	
	// Third transaction should remain unchanged
	assert.Equal(t, "Regular payment", processed[2].Description)
}

func TestParseSkipsIncompleteTransactions(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_revolut.csv")

	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92
CARD_PAYMENT,Current,2025-01-02 08:07:09,,Empty completed date,-25.00,0.00,CHF,PENDING,28.92
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,,-10.00,0.00,CHF,COMPLETED,18.92
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Pending transaction,-15.00,0.00,CHF,PENDING,3.92`

	err := os.WriteFile(testFile, []byte(csvContent), 0600)
	require.NoError(t, err)

	file, err := os.Open(testFile)
	require.NoError(t, err)
	defer file.Close()

	logger := logging.NewLogrusAdapter("info", "text")
	transactions, err := Parse(file, logger)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1) // Only completed transaction with valid data
	assert.Equal(t, "Boreal Coffee Shop", transactions[0].Description)
}

// Mock categorizer for testing
type mockCategorizer struct {
	category models.Category
}

func (m *mockCategorizer) Categorize(description string, isDebtor bool, amount, date, reference string) (models.Category, error) {
	return m.category, nil
}

// Mock categorizer that returns error
type mockCategorizerError struct{}

func (m *mockCategorizerError) Categorize(description string, isDebtor bool, amount, date, reference string) (models.Category, error) {
	return models.Category{}, fmt.Errorf("categorization failed")
}

func TestConvertRevolutRowToTransactionWithFeeError(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	
	row := RevolutCSVRow{
		Type:          "CARD_PAYMENT",
		Product:       "Current",
		StartedDate:   "2025-01-02 08:07:09",
		CompletedDate: "2025-01-03 15:38:51",
		Description:   "Test Payment",
		Amount:        "-57.50",
		Fee:           "invalid_fee", // Invalid fee should be handled gracefully
		Currency:      "CHF",
		State:         "COMPLETED",
		Balance:       "53.92",
	}

	tx, err := convertRevolutRowToTransaction(row, logger)
	assert.NoError(t, err) // Should not error, just warn and use zero fee
	assert.Equal(t, "Test Payment", tx.Description)
}

func TestConvertRevolutRowToTransactionWithInvalidAmount(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	
	row := RevolutCSVRow{
		Type:          "CARD_PAYMENT",
		Product:       "Current", 
		StartedDate:   "2025-01-02 08:07:09",
		CompletedDate: "2025-01-03 15:38:51",
		Description:   "Test Payment",
		Amount:        "invalid_amount",
		Fee:           "0.00",
		Currency:      "CHF",
		State:         "COMPLETED",
		Balance:       "53.92",
	}

	_, err := convertRevolutRowToTransaction(row, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing amount to decimal")
}

func TestBatchConvertWithInvalidDirectory(t *testing.T) {
	count, err := BatchConvert("/nonexistent/directory", "/tmp/output")
	assert.Error(t, err)
	assert.Equal(t, 0, count)
}

func TestParseWithInvalidCSVData(t *testing.T) {
	invalidCSV := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,"unclosed quote,-57.50,0.00,CHF,COMPLETED,53.92`

	reader := strings.NewReader(invalidCSV)
	logger := logging.NewLogrusAdapter("info", "text")
	
	_, err := Parse(reader, logger)
	assert.Error(t, err)
}

func TestParseWithNilLogger(t *testing.T) {
	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Boreal Coffee Shop,-57.50,0.00,CHF,COMPLETED,53.92`

	reader := strings.NewReader(csvContent)
	
	// Should work with nil logger (creates default)
	transactions, err := Parse(reader, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
}

func TestWriteToCSVWithLogger(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")
	logger := logging.NewLogrusAdapter("info", "text")

	transactions := []models.Transaction{
		{
			Date:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			Description: "Test Transaction",
			Amount:      models.ParseAmount("10.00"),
			Currency:    "CHF",
		},
	}

	err := WriteToCSVWithLogger(transactions, outputFile, logger)
	assert.NoError(t, err)

	// Verify file exists and has content
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Test Transaction")
}

func TestConvertToCSVWithLogger(t *testing.T) {
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")
	logger := logging.NewLogrusAdapter("info", "text")

	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Test Payment,-25.00,0.00,CHF,COMPLETED,75.00`

	err := os.WriteFile(inputFile, []byte(csvContent), 0600)
	require.NoError(t, err)

	err = ConvertToCSVWithLogger(inputFile, outputFile, logger)
	assert.NoError(t, err)

	// Verify output
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Test Payment")
}

func TestBatchConvertWithLogger(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	logger := logging.NewLogrusAdapter("info", "text")

	// Create input directory
	err := os.MkdirAll(inputDir, 0750)
	require.NoError(t, err)

	// Create valid CSV file
	validCSV := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Test Payment,-25.00,0.00,CHF,COMPLETED,75.00`
	
	validFile := filepath.Join(inputDir, "test.csv")
	err = os.WriteFile(validFile, []byte(validCSV), 0600)
	require.NoError(t, err)

	count, err := BatchConvertWithLogger(inputDir, outputDir, logger)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}
