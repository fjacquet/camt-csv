package debitparser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetLogger(t *testing.T) {
	// This test is no longer relevant since we use dependency injection for loggers
	// The Parse function now accepts a logger parameter
	logger := logging.NewLogrusAdapter("debug", "text")
	assert.NotNil(t, logger)
}

func TestValidateFormat(t *testing.T) {
	// Create a temporary valid debit CSV file
	validContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;-4,21;CHF
PMT CARTE Parking-Relais Lausa;02.04.2025;-4,00;CHF`

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.csv")
	if err := os.WriteFile(validFile, []byte(validContent), 0600); err != nil {
		t.Fatalf("Failed to write valid file: %v", err)
	}

	// Create a temporary invalid CSV file
	invalidContent := `SomeHeader1;SomeHeader2
Value1;Value2`

	invalidFile := filepath.Join(tempDir, "invalid.csv")
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0600); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	// Test validation on valid file
	valid, err := ValidateFormat(validFile)
	if err != nil {
		t.Errorf("ValidateFormat returned an error for a valid file: %v", err)
	}
	if !valid {
		t.Errorf("ValidateFormat returned false for a valid file")
	}

	// Test validation on invalid file
	valid, err = ValidateFormat(invalidFile)
	if err != nil {
		t.Errorf("ValidateFormat returned an error for an invalid file: %v", err)
	}
	if valid {
		t.Errorf("ValidateFormat returned true for an invalid file")
	}
}

func TestParseFile(t *testing.T) {
	// Create a temporary valid debit CSV file
	validContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;-4,21;CHF
PMT CARTE Parking-Relais Lausa;02.04.2025;-4,00;CHF
RETRAIT BCV MONTREUX FORUM;28.03.2025;-260,00;CHF`

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.csv")
	if err := os.WriteFile(validFile, []byte(validContent), 0600); err != nil {
		t.Fatalf("Failed to write valid file: %v", err)
	}

	// Parse the file
	transactions, err := ParseFile(validFile)
	if err != nil {
		t.Fatalf("ParseFile returned an error: %v", err)
	}

	// Check the number of transactions
	if len(transactions) != 3 {
		t.Errorf("Expected 3 transactions, got %d", len(transactions))
	}

	// Check the first transaction
	assert.Equal(t, 2025, transactions[0].Date.Year())
	assert.Equal(t, time.April, transactions[0].Date.Month())
	assert.Equal(t, 15, transactions[0].Date.Day())
	assert.Equal(t, "RATP", transactions[0].Description)
	assert.Equal(t, models.ParseAmount("4.21"), transactions[0].Amount)
	assert.Equal(t, "CHF", transactions[0].Currency)
	assert.Equal(t, models.TransactionTypeDebit, transactions[0].CreditDebit)
}

func setupTestCategorizer(t *testing.T) {
	// The new categorizer system uses dependency injection and doesn't require global setup
	// Tests that need categorization should create their own categorizer instances
}

func TestWriteToCSV(t *testing.T) {
	setupTestCategorizer(t)
	// Create test transactions
	transactions := []struct {
		description string
		date        string
		amount      string
		currency    string
		creditDebit string
	}{
		{"RATP", "15.04.2025", "4.21", "CHF", models.TransactionTypeDebit},
		{"Parking-Relais Lausa", "02.04.2025", "4.00", "CHF", models.TransactionTypeDebit},
	}

	// Convert to models.Transaction
	var modelTransactions []struct {
		Description string
		Date        string
		ValueDate   string
		Amount      string
		Currency    string
		CreditDebit string
		Payee       string
	}
	for _, tx := range transactions {
		modelTransactions = append(modelTransactions, struct {
			Description string
			Date        string
			ValueDate   string
			Amount      string
			Currency    string
			CreditDebit string
			Payee       string
		}{
			Description: tx.description,
			Date:        tx.date,
			ValueDate:   tx.date,
			Amount:      tx.amount,
			Currency:    tx.currency,
			CreditDebit: tx.creditDebit,
			Payee:       tx.description,
		})
	}

	// Create a temporary output file
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	// Write transactions to CSV
	err := WriteToCSV(nil, outputFile)
	if err == nil {
		t.Errorf("WriteToCSV should have returned an error for nil transactions")
	}
}

func TestConvertToCSV(t *testing.T) {
	setupTestCategorizer(t)
	// Create a temporary valid debit CSV file
	validContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;-4,21;CHF
PMT CARTE Parking-Relais Lausa;02.04.2025;-4,00;CHF`

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	if err := os.WriteFile(inputFile, []byte(validContent), 0600); err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.csv")

	// Convert to CSV
	err := ConvertToCSV(inputFile, outputFile)
	if err != nil {
		t.Fatalf("ConvertToCSV returned an error: %v", err)
	}

	// Check if output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}

	// Test with invalid input file
	err = ConvertToCSV("nonexistent.csv", outputFile)
	if err == nil {
		t.Errorf("ConvertToCSV should have returned an error for a nonexistent file")
	}
}

func TestParse(t *testing.T) {
	validContent := `Bénéficiaire;Date;Montant;Monnaie;Buchungs-Nr.;Referenznummer;Status Kontoführung
PMT CARTE RATP;15.04.2025;-4,21;CHF;12345;REF123;COMPLETED
PMT CARTE Parking-Relais Lausa;02.04.2025;-4,00;CHF;12346;REF124;COMPLETED`

	reader := strings.NewReader(validContent)
	logger := logging.NewLogrusAdapter("info", "text")

	transactions, err := Parse(reader, logger)
	assert.NoError(t, err)
	assert.Len(t, transactions, 2)
	assert.Equal(t, "RATP", transactions[0].Description)
	assert.Equal(t, "Parking-Relais Lausa", transactions[1].Description)
}

func TestParseWithCategorizer(t *testing.T) {
	validContent := `Bénéficiaire;Date;Montant;Monnaie;Buchungs-Nr.;Referenznummer;Status Kontoführung
PMT CARTE RATP;15.04.2025;-4,21;CHF;12345;REF123;COMPLETED`

	reader := strings.NewReader(validContent)
	logger := logging.NewLogrusAdapter("info", "text")

	// Mock categorizer
	mockCategorizer := &mockCategorizer{
		category: models.Category{Name: "Transport"},
	}

	transactions, err := ParseWithCategorizer(reader, logger, mockCategorizer)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, "Transport", transactions[0].Category)
}

func TestParseWithCategorizerError(t *testing.T) {
	validContent := `Bénéficiaire;Date;Montant;Monnaie;Buchungs-Nr.;Referenznummer;Status Kontoführung
PMT CARTE RATP;15.04.2025;-4,21;CHF;12345;REF123;COMPLETED`

	reader := strings.NewReader(validContent)
	logger := logging.NewLogrusAdapter("info", "text")

	// Mock categorizer that returns error
	mockCategorizer := &mockCategorizerError{}

	transactions, err := ParseWithCategorizer(reader, logger, mockCategorizer)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, models.CategoryUncategorized, transactions[0].Category)
}

func TestParseWithNilLogger(t *testing.T) {
	validContent := `Bénéficiaire;Date;Montant;Monnaie;Buchungs-Nr.;Referenznummer;Status Kontoführung
PMT CARTE RATP;15.04.2025;-4,21;CHF;12345;REF123;COMPLETED`

	reader := strings.NewReader(validContent)

	// Should work with nil logger (creates default)
	transactions, err := Parse(reader, nil)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
}

func TestParseWithInvalidCSV(t *testing.T) {
	invalidContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;"unclosed quote;CHF`

	reader := strings.NewReader(invalidContent)
	logger := logging.NewLogrusAdapter("info", "text")

	_, err := Parse(reader, logger)
	assert.Error(t, err)
}

func TestParseFileWithLogger(t *testing.T) {
	validContent := `Bénéficiaire;Date;Montant;Monnaie;Buchungs-Nr.;Referenznummer;Status Kontoführung
PMT CARTE RATP;15.04.2025;-4,21;CHF;12345;REF123;COMPLETED`

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.csv")
	err := os.WriteFile(validFile, []byte(validContent), 0600)
	require.NoError(t, err)

	logger := logging.NewLogrusAdapter("info", "text")
	transactions, err := ParseFileWithLogger(validFile, logger)
	assert.NoError(t, err)
	assert.Len(t, transactions, 1)
}

func TestParseFileWithInvalidFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.csv")
	assert.Error(t, err)
}

func TestConvertDebitRowToTransaction(t *testing.T) {
	tests := []struct {
		name        string
		row         DebitCSVRow
		expectError bool
		expected    func(*testing.T, models.Transaction)
	}{
		{
			name: "valid debit transaction",
			row: DebitCSVRow{
				Beneficiaire:       "PMT CARTE RATP",
				Datum:              "15.04.2025",
				Betrag:             "-4,21",
				Waehrung:           "CHF",
				BuchungsNr:         "12345",
				Referenznummer:     "REF123",
				StatusKontofuhrung: "COMPLETED",
			},
			expectError: false,
			expected: func(t *testing.T, tx models.Transaction) {
				assert.Equal(t, "RATP", tx.Description)
				assert.Equal(t, models.ParseAmount("4.21"), tx.Amount)
				assert.Equal(t, "CHF", tx.Currency)
				assert.Equal(t, models.TransactionTypeDebit, tx.CreditDebit)
			},
		},
		{
			name: "valid credit transaction",
			row: DebitCSVRow{
				Beneficiaire:       "Salary Payment",
				Datum:              "15.04.2025",
				Betrag:             "1000,00",
				Waehrung:           "CHF",
				BuchungsNr:         "12346",
				Referenznummer:     "REF124",
				StatusKontofuhrung: "COMPLETED",
			},
			expectError: false,
			expected: func(t *testing.T, tx models.Transaction) {
				assert.Equal(t, "Salary Payment", tx.Description)
				assert.Equal(t, models.ParseAmount("1000.00"), tx.Amount)
				assert.Equal(t, "CHF", tx.Currency)
				assert.Equal(t, models.TransactionTypeCredit, tx.CreditDebit)
			},
		},
		{
			name: "empty date",
			row: DebitCSVRow{
				Beneficiaire: "Test",
				Datum:        "",
				Betrag:       "10,00",
				Waehrung:     "CHF",
			},
			expectError: true,
		},
		{
			name: "empty amount",
			row: DebitCSVRow{
				Beneficiaire: "Test",
				Datum:        "15.04.2025",
				Betrag:       "",
				Waehrung:     "CHF",
			},
			expectError: true, // TransactionBuilder requires non-zero amount
		},
		{
			name: "invalid amount",
			row: DebitCSVRow{
				Beneficiaire: "Test",
				Datum:        "15.04.2025",
				Betrag:       "invalid",
				Waehrung:     "CHF",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := convertDebitRowToTransaction(tt.row)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected != nil {
					tt.expected(t, tx)
				}
			}
		})
	}
}

func TestWriteToCSVEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	// Test with nil transactions
	err := WriteToCSV(nil, outputFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transactions is nil")

	// Test with empty transactions
	err = WriteToCSV([]models.Transaction{}, outputFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transactions to write")
}

func TestValidateFormatWithLogger(t *testing.T) {
	tempDir := t.TempDir()
	logger := logging.NewLogrusAdapter("info", "text")

	// Test with valid file
	validContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;-4,21;CHF`

	validFile := filepath.Join(tempDir, "valid.csv")
	err := os.WriteFile(validFile, []byte(validContent), 0600)
	require.NoError(t, err)

	valid, err := ValidateFormatWithLogger(validFile, logger)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test with nonexistent file
	valid, err = ValidateFormatWithLogger("/nonexistent/file.csv", logger)
	assert.Error(t, err)
	assert.False(t, valid)

	// Test with empty file (header only)
	emptyContent := `Bénéficiaire;Date;Montant;Monnaie`
	emptyFile := filepath.Join(tempDir, "empty.csv")
	err = os.WriteFile(emptyFile, []byte(emptyContent), 0600)
	require.NoError(t, err)

	valid, err = ValidateFormatWithLogger(emptyFile, logger)
	assert.NoError(t, err)
	assert.True(t, valid) // Empty file but valid format
}

func TestBatchConvert(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")

	// Create input directory
	err := os.MkdirAll(inputDir, 0750)
	require.NoError(t, err)

	// Create valid debit CSV file
	validCSV := `Bénéficiaire;Date;Montant;Monnaie;Buchungs-Nr.;Referenznummer;Status Kontoführung
PMT CARTE RATP;15.04.2025;-4,21;CHF;12345;REF123;COMPLETED`

	validFile := filepath.Join(inputDir, "valid.csv")
	err = os.WriteFile(validFile, []byte(validCSV), 0600)
	require.NoError(t, err)

	// Create invalid CSV file
	invalidCSV := `SomeHeader1;SomeHeader2
Value1;Value2`

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
	outputFile := filepath.Join(outputDir, "valid_processed.csv")
	_, err = os.Stat(outputFile)
	assert.NoError(t, err)
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
	validCSV := `Bénéficiaire;Date;Montant;Monnaie;Buchungs-Nr.;Referenznummer;Status Kontoführung
PMT CARTE Test;15.04.2025;-25,00;CHF;12345;REF123;COMPLETED`

	validFile := filepath.Join(inputDir, "test.csv")
	err = os.WriteFile(validFile, []byte(validCSV), 0600)
	require.NoError(t, err)

	count, err := BatchConvertWithLogger(inputDir, outputDir, logger)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestBatchConvertErrors(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")

	// Test with nonexistent input directory
	count, err := BatchConvertWithLogger("/nonexistent/dir", "/tmp/output", logger)
	assert.Error(t, err)
	assert.Equal(t, 0, count)

	// Test with file instead of directory
	tempDir := t.TempDir()
	notADir := filepath.Join(tempDir, "notadir.txt")
	err = os.WriteFile(notADir, []byte("content"), 0600)
	require.NoError(t, err)

	count, err = BatchConvertWithLogger(notADir, "/tmp/output", logger)
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "input path is not a directory")
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
