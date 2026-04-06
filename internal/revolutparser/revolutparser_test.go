package revolutparser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	assert.Equal(t, models.ParseAmount("-2.50"), transactions[0].Amount)
	assert.Equal(t, "CHF", transactions[0].Currency)
	assert.Equal(t, models.TransactionTypeDebit, transactions[0].CreditDebit)
	assert.Equal(t, models.StatusCompleted, transactions[0].Status)

	// Verify second transaction
	assert.Equal(t, "03.01.2025", transactions[1].Date.Format(dateutils.DateLayoutEuropean))
	assert.Equal(t, "02.01.2025", transactions[1].ValueDate.Format(dateutils.DateLayoutEuropean))
	assert.Equal(t, "Boreal Coffee Shop", transactions[1].Description)
	assert.Equal(t, models.ParseAmount("-57.50"), transactions[1].Amount)
	assert.Equal(t, models.TransactionTypeDebit, transactions[1].CreditDebit)
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

// --- normalizeCSVData tests ---

func TestNormalizeCSVData_FrenchHeaders(t *testing.T) {
	frenchCSV := "Type,Produit,Date de début,Date de fin,Description,Montant,Frais,Devise,État,Solde\n" +
		"Paiement par carte,Valeur actuelle,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,TERMINÉ,100.00\n"

	result := normalizeCSVData([]byte(frenchCSV))
	resultStr := string(result)

	assert.Contains(t, resultStr, "Product")
	assert.Contains(t, resultStr, "Started Date")
	assert.Contains(t, resultStr, "Completed Date")
	assert.Contains(t, resultStr, "Amount")
	assert.Contains(t, resultStr, "Fee")
	assert.Contains(t, resultStr, "Currency")
	assert.Contains(t, resultStr, "State")
	assert.Contains(t, resultStr, "Balance")
	assert.NotContains(t, resultStr, "Produit")
	assert.NotContains(t, resultStr, "Montant")
}

func TestNormalizeCSVData_FrenchValues(t *testing.T) {
	frenchCSV := "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n" +
		"Paiement par carte,Valeur actuelle,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,TERMINÉ,100.00\n"

	result := normalizeCSVData([]byte(frenchCSV))
	resultStr := string(result)

	assert.Contains(t, resultStr, "CARD_PAYMENT")
	assert.Contains(t, resultStr, "CURRENT")
	assert.Contains(t, resultStr, "COMPLETED")
	assert.NotContains(t, resultStr, "Paiement par carte")
	assert.NotContains(t, resultStr, "TERMINÉ")
}

func TestNormalizeCSVData_AllFrenchTypes(t *testing.T) {
	tests := []struct {
		french  string
		english string
	}{
		{"Paiement par carte", "CARD_PAYMENT"},
		{"Virement", "TRANSFER"},
		{"Changes", "EXCHANGE"},
		{"Ajout de fonds", "TOPUP"},
		{"Remboursement des frais", "FEE_REFUND"},
		{"Retrait d'espèces", "ATM"},
	}

	for _, tt := range tests {
		t.Run(tt.french, func(t *testing.T) {
			csv := "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n" +
				tt.french + ",Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Test,-10.50,0.00,CHF,COMPLETED,100.00\n"
			result := string(normalizeCSVData([]byte(csv)))
			assert.Contains(t, result, tt.english)
		})
	}
}

func TestNormalizeCSVData_AllFrenchStates(t *testing.T) {
	tests := []struct {
		french  string
		english string
	}{
		{"TERMINÉ", "COMPLETED"},
		{"ANNULÉ", "REVERTED"},
		{"EN ATTENTE", "PENDING"},
		{"DÉCLINÉ", "DECLINED"},
		{"RÉVISÉ", "REVERTED"},
	}

	for _, tt := range tests {
		t.Run(tt.french, func(t *testing.T) {
			csv := "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n" +
				"CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Test,-10.50,0.00,CHF," + tt.french + ",100.00\n"
			result := string(normalizeCSVData([]byte(csv)))
			assert.Contains(t, result, tt.english)
		})
	}
}

func TestNormalizeCSVData_FrenchProducts(t *testing.T) {
	tests := []struct {
		french  string
		english string
	}{
		{"Valeur actuelle", "CURRENT"},
		{"Épargne", "SAVINGS"},
	}

	for _, tt := range tests {
		t.Run(tt.french, func(t *testing.T) {
			csv := "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n" +
				"CARD_PAYMENT," + tt.french + ",2025-01-02 08:07:09,2025-01-03 15:38:51,Test,-10.50,0.00,CHF,COMPLETED,100.00\n"
			result := string(normalizeCSVData([]byte(csv)))
			assert.Contains(t, result, tt.english)
		})
	}
}

func TestNormalizeCSVData_EnglishPassthrough(t *testing.T) {
	englishCSV := "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n" +
		"CARD_PAYMENT,Current,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,COMPLETED,100.00\n"

	result := normalizeCSVData([]byte(englishCSV))
	// Description should not be altered
	assert.Contains(t, string(result), "Coffee")
	assert.Contains(t, string(result), "CARD_PAYMENT")
}

func TestNormalizeCSVData_InvalidCSV(t *testing.T) {
	invalidCSV := []byte("not\"valid,csv\n")
	result := normalizeCSVData(invalidCSV)
	// Should return original data on parse error
	assert.Equal(t, invalidCSV, result)
}

func TestNormalizeCSVData_EmptyInput(t *testing.T) {
	result := normalizeCSVData([]byte{})
	assert.Equal(t, []byte{}, result)
}

func TestNormalizeCSVData_DescriptionNotAltered(t *testing.T) {
	// French descriptions should NOT be normalized
	csv := "Type,Produit,Date de début,Date de fin,Description,Montant,Frais,Devise,État,Solde\n" +
		"Paiement par carte,Valeur actuelle,2025-01-02,2025-01-03,Épicerie du village,-50.00,0.00,CHF,TERMINÉ,200.00\n"
	result := string(normalizeCSVData([]byte(csv)))
	assert.Contains(t, result, "Épicerie du village") // description preserved
	assert.Contains(t, result, "CARD_PAYMENT")        // type normalized
}

func TestValidateFormat_FrenchCSV(t *testing.T) {
	frenchCSV := "Type,Produit,Date de début,Date de fin,Description,Montant,Frais,Devise,État,Solde\n" +
		"Paiement par carte,Valeur actuelle,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee,-10.50,0.00,CHF,TERMINÉ,100.00\n"

	valid, err := validateFormat(strings.NewReader(frenchCSV))
	assert.NoError(t, err)
	assert.True(t, valid, "French-localized CSV should be accepted after normalization")
}

func TestParseWithCategorizer_FrenchCSV(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	frenchCSV := "Type,Produit,Date de début,Date de fin,Description,Montant,Frais,Devise,État,Solde\n" +
		"Paiement par carte,Valeur actuelle,2025-01-02 08:07:09,2025-01-03 15:38:51,Coffee Shop,-10.50,0.00,CHF,TERMINÉ,100.00\n"

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "revolut_fr.csv")
	err := os.WriteFile(tmpFile, []byte(frenchCSV), 0600)
	require.NoError(t, err)

	file, err := os.Open(tmpFile)
	require.NoError(t, err)
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			t.Logf("Failed to close file: %v", closeErr)
		}
	}()

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	data = normalizeCSVData(data)

	transactions, err := ParseWithCategorizer(strings.NewReader(string(data)), logger, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, "Coffee Shop", transactions[0].Description)
}
