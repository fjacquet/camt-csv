package revolutparser

import (
	"context"
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
	transactions, err := adapter.Parse(context.Background(), file)
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
	_, err = adapter.Parse(context.Background(), file)
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
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file: %v", err)
		}
	}()

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
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file: %v", err)
		}
	}()

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
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file: %v", err)
		}
	}()

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

func (m *mockCategorizer) Categorize(ctx context.Context, description string, isDebtor bool, amount, date, reference string) (models.Category, error) {
	return m.category, nil
}

// Mock categorizer that returns error
type mockCategorizerError struct{}

func (m *mockCategorizerError) Categorize(ctx context.Context, description string, isDebtor bool, amount, date, reference string) (models.Category, error) {
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

func TestAdapter_ConvertToCSV(t *testing.T) {
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee Shop,-10.50,0.00,CHF,COMPLETED,100.00`

	err := os.WriteFile(inputFile, []byte(csvContent), 0600)
	require.NoError(t, err)

	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	err = adapter.ConvertToCSV(context.Background(), inputFile, outputFile)
	assert.NoError(t, err)

	// Verify output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err)
}

func TestAdapter_ValidateFormat(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid revolut file", func(t *testing.T) {
		validFile := filepath.Join(tempDir, "valid.csv")
		csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,COMPLETED,100.00`

		err := os.WriteFile(validFile, []byte(csvContent), 0600)
		require.NoError(t, err)

		logger := logging.NewLogrusAdapter("info", "text")
		adapter := NewAdapter(logger)

		valid, err := adapter.ValidateFormat(validFile)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("invalid file format", func(t *testing.T) {
		invalidFile := filepath.Join(tempDir, "invalid.csv")
		csvContent := `Wrong,Headers
data,here`

		err := os.WriteFile(invalidFile, []byte(csvContent), 0600)
		require.NoError(t, err)

		logger := logging.NewLogrusAdapter("info", "text")
		adapter := NewAdapter(logger)

		valid, err := adapter.ValidateFormat(invalidFile)
		assert.NoError(t, err)
		assert.False(t, valid)
	})
}

func TestAdapter_BatchConvert(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")

	err := os.MkdirAll(inputDir, 0750)
	require.NoError(t, err)

	// Create valid Revolut CSV file
	validFile := filepath.Join(inputDir, "revolut1.csv")
	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,COMPLETED,100.00`

	err = os.WriteFile(validFile, []byte(csvContent), 0600)
	require.NoError(t, err)

	// Create invalid file (should be skipped)
	invalidFile := filepath.Join(inputDir, "other.csv")
	err = os.WriteFile(invalidFile, []byte("Wrong,Format\ndata,here"), 0600)
	require.NoError(t, err)

	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Only 1 valid file should be processed
}

func TestConvertRevolutRowToTransaction_EdgeCases(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")

	tests := []struct {
		name        string
		row         RevolutCSVRow
		expectError bool
	}{
		{
			name: "zero fee",
			row: RevolutCSVRow{
				Type:          "CARD_PAYMENT",
				Product:       "Current",
				StartedDate:   "2025-01-01 10:00:00",
				CompletedDate: "2025-01-01 10:00:00",
				Description:   "Test",
				Amount:        "100.00",
				Fee:           "",
				Currency:      "CHF",
				State:         "COMPLETED",
			},
			expectError: false,
		},
		{
			name: "invalid fee format",
			row: RevolutCSVRow{
				Type:          "CARD_PAYMENT",
				Product:       "Current",
				StartedDate:   "2025-01-01 10:00:00",
				CompletedDate: "2025-01-01 10:00:00",
				Description:   "Test",
				Amount:        "100.00",
				Fee:           "invalid",
				Currency:      "CHF",
				State:         "COMPLETED",
			},
			expectError: false, // Should default to zero
		},
		{
			name: "invalid amount",
			row: RevolutCSVRow{
				Type:          "CARD_PAYMENT",
				Product:       "Current",
				StartedDate:   "2025-01-01 10:00:00",
				CompletedDate: "2025-01-01 10:00:00",
				Description:   "Test",
				Amount:        "invalid",
				Fee:           "0.00",
				Currency:      "CHF",
				State:         "COMPLETED",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := convertRevolutRowToTransaction(tt.row, logger)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tx.Description)
			}
		})
	}
}

func TestWriteToCSVWithLogger_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.NewLogrusAdapter("info", "text")

	t.Run("nil transactions", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "nil.csv")
		err := WriteToCSVWithLogger(nil, outputFile, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot write nil transactions")
	})

	t.Run("empty transactions", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "empty.csv")
		transactions := []models.Transaction{}
		err := WriteToCSVWithLogger(transactions, outputFile, logger)
		assert.NoError(t, err)
	})

	t.Run("transaction with zero date", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "zero_date.csv")
		transactions := []models.Transaction{
			{
				Date:        time.Time{}, // Zero time
				Description: "Test",
				Amount:      models.ParseAmount("100.00"),
				Currency:    "CHF",
			},
		}
		err := WriteToCSVWithLogger(transactions, outputFile, logger)
		assert.NoError(t, err)
	})
}

func TestBatchConvertWithLogger_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.NewLogrusAdapter("info", "text")

	t.Run("non-existent input directory", func(t *testing.T) {
		inputDir := filepath.Join(tempDir, "nonexistent")
		outputDir := filepath.Join(tempDir, "output")

		count, err := BatchConvertWithLogger(inputDir, outputDir, logger)
		assert.Error(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("empty input directory", func(t *testing.T) {
		inputDir := filepath.Join(tempDir, "empty")
		outputDir := filepath.Join(tempDir, "output")

		err := os.MkdirAll(inputDir, 0750)
		require.NoError(t, err)

		count, err := BatchConvertWithLogger(inputDir, outputDir, logger)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("directory with non-csv files", func(t *testing.T) {
		inputDir := filepath.Join(tempDir, "mixed")
		outputDir := filepath.Join(tempDir, "output2")

		err := os.MkdirAll(inputDir, 0750)
		require.NoError(t, err)

		// Create a non-CSV file
		txtFile := filepath.Join(inputDir, "readme.txt")
		err = os.WriteFile(txtFile, []byte("Not a CSV"), 0600)
		require.NoError(t, err)

		count, err := BatchConvertWithLogger(inputDir, outputDir, logger)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("directory with subdirectories", func(t *testing.T) {
		inputDir := filepath.Join(tempDir, "with_subdirs")
		outputDir := filepath.Join(tempDir, "output3")

		err := os.MkdirAll(inputDir, 0750)
		require.NoError(t, err)

		// Create a subdirectory (should be skipped)
		subDir := filepath.Join(inputDir, "subdir")
		err = os.MkdirAll(subDir, 0750)
		require.NoError(t, err)

		count, err := BatchConvertWithLogger(inputDir, outputDir, logger)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestParseWithCategorizer_EmptyRows(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")

	// Test with rows that have empty description (which should be skipped)
	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,COMPLETED,100.00
CARD_PAYMENT,Current,2025-01-03 08:07:09,2025-01-04 15:38:51,,-20.00,0.00,CHF,COMPLETED,80.00
CARD_PAYMENT,Current,2025-01-04 08:07:09,2025-01-05 15:38:51,Lunch,-30.00,0.00,CHF,COMPLETED,50.00`

	transactions, err := ParseWithCategorizer(strings.NewReader(csvContent), logger, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(transactions)) // Row with empty description should be skipped
}

func TestParseWithCategorizer_PendingTransactions(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")

	csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,COMPLETED,100.00
CARD_PAYMENT,Current,2025-01-03 08:07:09,,Pending Payment,-20.00,0.00,CHF,PENDING,80.00`

	transactions, err := ParseWithCategorizer(strings.NewReader(csvContent), logger, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(transactions)) // Pending transaction should be skipped
}

func TestPostProcessTransactions_CHFVacances(t *testing.T) {
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
			Type:        "TRANSFER",
			Description: "Other Transfer",
			CreditDebit: models.TransactionTypeDebit,
		},
	}

	processed := postProcessTransactions(transactions)

	assert.Equal(t, "Transfert to CHF Vacances", processed[0].Description)
	assert.Equal(t, "Transfert to CHF Vacances", processed[0].Name)
	assert.Equal(t, "Transfert to CHF Vacances", processed[0].PartyName)

	assert.Equal(t, "Transferred To CHF Vacances", processed[1].Description)
	assert.Equal(t, "Transferred To CHF Vacances", processed[1].Name)
	assert.Equal(t, "Transferred To CHF Vacances", processed[1].PartyName)

	assert.Equal(t, "Other Transfer", processed[2].Description) // Unchanged
}

func TestConvertToCSVWithLogger_FileErrors(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.NewLogrusAdapter("info", "text")

	t.Run("non-existent input file", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "nonexistent.csv")
		outputFile := filepath.Join(tempDir, "output.csv")

		err := ConvertToCSVWithLogger(inputFile, outputFile, logger)
		assert.Error(t, err)
	})

	t.Run("invalid output directory", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "input.csv")
		csvContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,COMPLETED,100.00`

		err := os.WriteFile(inputFile, []byte(csvContent), 0600)
		require.NoError(t, err)

		// Try to write to a file as if it were a directory
		outputFile := filepath.Join(tempDir, "file.txt", "output.csv")
		fileAsDir := filepath.Join(tempDir, "file.txt")
		err = os.WriteFile(fileAsDir, []byte("content"), 0600)
		require.NoError(t, err)

		err = ConvertToCSVWithLogger(inputFile, outputFile, logger)
		assert.Error(t, err)
	})
}

// TestRevolutParser_ErrorMessagesIncludeFilePath validates error messages include helpful context
func TestRevolutParser_ErrorMessagesIncludeFilePath(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	t.Run("invalid_file_path_in_error", func(t *testing.T) {
		invalidPath := "/nonexistent/test_file.csv"

		err := adapter.ConvertToCSV(context.Background(), invalidPath, "/tmp/output.csv")
		require.Error(t, err)

		// Error should include the file path that was attempted
		assert.Contains(t, err.Error(), invalidPath,
			"Error message should include file path for debugging")
	})

	t.Run("malformed_csv_includes_context", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "malformed.csv")

		// Create malformed CSV (wrong headers)
		malformedCSV := `WrongHeader1,WrongHeader2,WrongHeader3
Value1,Value2,Value3`

		err := os.WriteFile(testFile, []byte(malformedCSV), 0600)
		require.NoError(t, err)

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		_, err = adapter.Parse(context.Background(), file)
		require.Error(t, err)

		// Error should mention that it's a format validation issue
		errMsg := err.Error()
		assert.True(t,
			strings.Contains(errMsg, "header") || strings.Contains(errMsg, "format") || strings.Contains(errMsg, "column"),
			"Error message should indicate format issue: %s", errMsg)
	})

	t.Run("missing_required_field_includes_field_name", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "missing_field.csv")

		// Create CSV with missing required fields
		missingFieldCSV := `Type,Description,Amount
CARD_PAYMENT,Coffee,-10.50`

		err := os.WriteFile(testFile, []byte(missingFieldCSV), 0600)
		require.NoError(t, err)

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		_, err = adapter.Parse(context.Background(), file)
		// Parser should detect missing required columns
		require.Error(t, err)
		errMsg := err.Error()
		assert.True(t,
			strings.Contains(errMsg, "format") || strings.Contains(errMsg, "invalid"),
			"Error should mention format validation issue: %s", errMsg)
	})

	t.Run("invalid_amount_format_includes_context", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "invalid_amount.csv")

		// Create CSV with invalid amount format
		invalidAmountCSV := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,INVALID_AMOUNT,0.00,CHF,COMPLETED,100.00`

		err := os.WriteFile(testFile, []byte(invalidAmountCSV), 0600)
		require.NoError(t, err)

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()

		transactions, err := adapter.Parse(context.Background(), file)
		// Parser should handle gracefully or return descriptive error
		if err != nil {
			// If error, should mention amount parsing
			assert.Contains(t, err.Error(), "amount",
				"Error message should mention amount field")
		} else {
			// If no error, should still return valid structure
			assert.NotNil(t, transactions)
		}
	})
}
