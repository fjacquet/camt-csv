package common

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFormatter implements formatter.OutputFormatter for testing.
type mockFormatter struct {
	header    []string
	rows      [][]string
	delimiter rune
	formatErr error
}

func (m *mockFormatter) Header() []string {
	return m.header
}

func (m *mockFormatter) Format(transactions []models.Transaction) ([][]string, error) {
	if m.formatErr != nil {
		return nil, m.formatErr
	}
	return m.rows, nil
}

func (m *mockFormatter) Delimiter() rune {
	return m.delimiter
}

// sampleTransactions returns a small set of transactions for testing.
func sampleTransactions() []models.Transaction {
	return []models.Transaction{
		{
			Date:        time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			ValueDate:   time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			Description: "Grocery Store",
			Amount:      models.ParseAmount("-50.00"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeDebit,
			Payee:       "Migros",
		},
		{
			Date:        time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC),
			ValueDate:   time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC),
			Description: "Salary",
			Amount:      models.ParseAmount("5000.00"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
			Payer:       "Employer AG",
		},
	}
}

// ---------------------------------------------------------------------------
// WriteTransactionsToCSVWithFormatter tests (0% coverage)
// ---------------------------------------------------------------------------

func TestWriteTransactionsToCSVWithFormatter(t *testing.T) {
	tests := []struct {
		name        string
		txs         []models.Transaction
		formatter   *mockFormatter
		delimiter   rune
		setupPath   func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "success with valid transactions",
			txs:  sampleTransactions(),
			formatter: &mockFormatter{
				header:    []string{"Date", "Description", "Amount"},
				rows:      [][]string{{"15.03.2024", "Grocery Store", "-50.00"}, {"16.03.2024", "Salary", "5000.00"}},
				delimiter: ',',
			},
			delimiter: ',',
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "output.csv")
			},
			wantErr: false,
		},
		{
			name:        "nil transactions returns error",
			txs:         nil,
			formatter:   &mockFormatter{},
			delimiter:   ',',
			setupPath:   func(t *testing.T) string { return filepath.Join(t.TempDir(), "out.csv") },
			wantErr:     true,
			errContains: "cannot write nil transactions",
		},
		{
			name: "formatter error propagates",
			txs:  sampleTransactions(),
			formatter: &mockFormatter{
				formatErr: errors.New("format explosion"),
			},
			delimiter:   ',',
			setupPath:   func(t *testing.T) string { return filepath.Join(t.TempDir(), "out.csv") },
			wantErr:     true,
			errContains: "error formatting transactions",
		},
		{
			name: "nil logger creates default logger",
			txs:  sampleTransactions(),
			formatter: &mockFormatter{
				header:    []string{"Col1"},
				rows:      [][]string{{"val1"}, {"val2"}},
				delimiter: ';',
			},
			delimiter: ';',
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "subdir", "output.csv")
			},
			wantErr: false,
		},
		{
			name:      "empty transactions skips output file",
			txs:       []models.Transaction{},
			delimiter: ',',
			formatter: &mockFormatter{
				header:    []string{"A", "B"},
				rows:      [][]string{},
				delimiter: ',',
			},
			setupPath: func(t *testing.T) string { return filepath.Join(t.TempDir(), "empty.csv") },
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csvPath := tt.setupPath(t)

			err := WriteTransactionsToCSVWithFormatter(tt.txs, csvPath, nil, tt.formatter, tt.delimiter)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)

			if len(tt.txs) == 0 {
				_, statErr := os.Stat(csvPath)
				assert.True(t, os.IsNotExist(statErr), "output file should not be created for 0 transactions")
				return
			}

			content, readErr := os.ReadFile(csvPath)
			require.NoError(t, readErr)
			assert.NotEmpty(t, content, "output CSV file should not be empty")
		})
	}
}

func TestWriteTransactionsToCSVWithFormatter_InvalidDirectory(t *testing.T) {
	txs := sampleTransactions()
	f := &mockFormatter{
		header: []string{"A"},
		rows:   [][]string{{"v"}},
	}

	// Use /dev/null/impossible on unix to trigger MkdirAll failure
	err := WriteTransactionsToCSVWithFormatter(txs, "/dev/null/impossible/out.csv", nil, f, ',')
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error creating directory")
}

func TestWriteTransactionsToCSVWithFormatter_WithExplicitLogger(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	txs := sampleTransactions()
	f := &mockFormatter{
		header:    []string{"Date", "Amount"},
		rows:      [][]string{{"15.03.2024", "-50.00"}, {"16.03.2024", "5000.00"}},
		delimiter: ';',
	}
	csvPath := filepath.Join(t.TempDir(), "logged.csv")

	err := WriteTransactionsToCSVWithFormatter(txs, csvPath, logger, f, ';')
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// ReadCSVFile error paths (75% -> higher)
// ---------------------------------------------------------------------------

func TestReadCSVFile_NilLogger(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "data.csv")
	err := os.WriteFile(csvPath, []byte("Name,Age\nAlice,30\n"), 0600)
	require.NoError(t, err)

	rows, err := ReadCSVFile[TestCSVRow](csvPath, nil)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestReadCSVFile_MalformedCSV(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "bad.csv")
	// Write content that gocsv cannot unmarshal into TestCSVRow (wrong number of fields)
	err := os.WriteFile(csvPath, []byte("Name,Age,Email,Country\n\"unclosed quote\n"), 0600)
	require.NoError(t, err)

	logger := logging.NewLogrusAdapter("info", "text")
	_, err = ReadCSVFile[TestCSVRow](csvPath, logger)
	assert.Error(t, err, "malformed CSV should produce a parse error")
	assert.Contains(t, err.Error(), "error parsing CSV file")
}

func TestReadCSVFile_NonExistentFile(t *testing.T) {
	_, err := ReadCSVFile[TestCSVRow]("/nonexistent/path/file.csv", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error opening CSV file")
}

// ---------------------------------------------------------------------------
// WriteTransactionsToCSVWithLogger error paths (76.9% -> higher)
// ---------------------------------------------------------------------------

func TestWriteTransactionsToCSVWithLogger_NilTransactions(t *testing.T) {
	err := WriteTransactionsToCSVWithLogger(nil, "/tmp/test.csv", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write nil transactions")
}

func TestWriteTransactionsToCSVWithLogger_InvalidPath(t *testing.T) {
	txs := sampleTransactions()
	err := WriteTransactionsToCSVWithLogger(txs, "/dev/null/impossible/file.csv", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error creating directory")
}

func TestWriteTransactionsToCSVWithLogger_NilLogger(t *testing.T) {
	csvPath := filepath.Join(t.TempDir(), "nil_logger.csv")
	txs := sampleTransactions()

	err := WriteTransactionsToCSVWithLogger(txs, csvPath, nil)
	require.NoError(t, err)

	content, readErr := os.ReadFile(csvPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "Grocery Store")
}

func TestWriteTransactionsToCSVWithLogger_DebitFlagFromNegativeAmount(t *testing.T) {
	csvPath := filepath.Join(t.TempDir(), "debit_flag.csv")
	txs := []models.Transaction{
		{
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "Negative amount debit",
			Amount:      models.ParseAmount("-100.00"),
			Currency:    "CHF",
			CreditDebit: "", // Leave empty so the Amount sign determines debit flag
		},
		{
			Date:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Description: "Positive amount credit",
			Amount:      models.ParseAmount("200.00"),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
		},
	}

	err := WriteTransactionsToCSVWithLogger(txs, csvPath, nil)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// GeneralizedConvertToCSVWithLogger error paths (77.8% -> higher)
// ---------------------------------------------------------------------------

func TestGeneralizedConvertToCSVWithLogger_NonExistentInput(t *testing.T) {
	parseFunc := func(string) ([]models.Transaction, error) {
		return nil, nil
	}
	validateFunc := func(string) (bool, error) {
		return true, nil
	}

	err := GeneralizedConvertToCSVWithLogger(
		"/nonexistent/input.xml",
		"/tmp/output.csv",
		parseFunc,
		validateFunc,
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input file does not exist")
}

func TestGeneralizedConvertToCSVWithLogger_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.xml")
	require.NoError(t, os.WriteFile(inputPath, []byte("<xml/>"), 0600))

	parseFunc := func(string) ([]models.Transaction, error) {
		return sampleTransactions(), nil
	}
	validateFunc := func(string) (bool, error) {
		return false, errors.New("validation failed")
	}

	err := GeneralizedConvertToCSVWithLogger(
		inputPath,
		filepath.Join(tmpDir, "output.csv"),
		parseFunc,
		validateFunc,
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error validating file format")
}

func TestGeneralizedConvertToCSVWithLogger_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.xml")
	require.NoError(t, os.WriteFile(inputPath, []byte("<xml/>"), 0600))

	parseFunc := func(string) ([]models.Transaction, error) {
		return sampleTransactions(), nil
	}
	validateFunc := func(string) (bool, error) {
		return false, nil // valid call, but format is invalid
	}

	err := GeneralizedConvertToCSVWithLogger(
		inputPath,
		filepath.Join(tmpDir, "output.csv"),
		parseFunc,
		validateFunc,
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file format")
}

func TestGeneralizedConvertToCSVWithLogger_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.xml")
	require.NoError(t, os.WriteFile(inputPath, []byte("<xml/>"), 0600))

	parseFunc := func(string) ([]models.Transaction, error) {
		return nil, errors.New("parse explosion")
	}
	validateFunc := func(string) (bool, error) {
		return true, nil
	}

	err := GeneralizedConvertToCSVWithLogger(
		inputPath,
		filepath.Join(tmpDir, "output.csv"),
		parseFunc,
		validateFunc,
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing file")
}

func TestGeneralizedConvertToCSVWithLogger_NilValidateFunc(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.xml")
	require.NoError(t, os.WriteFile(inputPath, []byte("<xml/>"), 0600))
	outputPath := filepath.Join(tmpDir, "output.csv")

	parseFunc := func(string) ([]models.Transaction, error) {
		return sampleTransactions(), nil
	}

	err := GeneralizedConvertToCSVWithLogger(inputPath, outputPath, parseFunc, nil, nil)
	require.NoError(t, err)

	content, readErr := os.ReadFile(outputPath)
	require.NoError(t, readErr)
	assert.NotEmpty(t, content)
}

func TestGeneralizedConvertToCSVWithLogger_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.xml")
	require.NoError(t, os.WriteFile(inputPath, []byte("<xml/>"), 0600))

	parseFunc := func(string) ([]models.Transaction, error) {
		return sampleTransactions(), nil
	}
	validateFunc := func(string) (bool, error) {
		return true, nil
	}

	// Write to an impossible path to trigger write error
	err := GeneralizedConvertToCSVWithLogger(
		inputPath,
		"/dev/null/impossible/output.csv",
		parseFunc,
		validateFunc,
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error writing transactions to CSV")
}

// ---------------------------------------------------------------------------
// ProcessTransactionsWithCategorizationStats edge cases (79.5% -> higher)
// ---------------------------------------------------------------------------

func TestProcessTransactionsWithCategorizationStats_EmptyPartyName(t *testing.T) {
	txs := []models.Transaction{
		{
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "Mystery transaction",
			Amount:      decimal.NewFromFloat(42.00),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
		},
	}

	// Use nil categorizer — empty party name should result in Uncategorized
	result := ProcessTransactionsWithCategorizationStats(txs, nil, nil, "test")
	assert.Equal(t, "Uncategorized", result[0].Category)
}

func TestProcessTransactionsWithCategorizationStats_FallbackToName(t *testing.T) {
	txs := []models.Transaction{
		{
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "Named transaction",
			Amount:      decimal.NewFromFloat(100.00),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
			Name:        "Fallback Name",
		},
	}

	mockCat := &simpleCategorizer{category: "TestCat"}
	result := ProcessTransactionsWithCategorizationStats(txs, nil, mockCat, "test")
	assert.Equal(t, "TestCat", result[0].Category)
}

func TestProcessTransactionsWithCategorizationStats_FallbackToRecipient(t *testing.T) {
	txs := []models.Transaction{
		{
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "Recipient transaction",
			Amount:      decimal.NewFromFloat(100.00),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
			Recipient:   "Recipient Corp",
		},
	}

	mockCat := &simpleCategorizer{category: "RecipientCat"}
	result := ProcessTransactionsWithCategorizationStats(txs, nil, mockCat, "test")
	assert.Equal(t, "RecipientCat", result[0].Category)
}

func TestProcessTransactionsWithCategorizationStats_FallbackToPartyNameField(t *testing.T) {
	txs := []models.Transaction{
		{
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "PartyName field transaction",
			Amount:      decimal.NewFromFloat(100.00),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
			PartyName:   "Party Corp",
		},
	}

	mockCat := &simpleCategorizer{category: "PartyCat"}
	result := ProcessTransactionsWithCategorizationStats(txs, nil, mockCat, "test")
	assert.Equal(t, "PartyCat", result[0].Category)
}

// simpleCategorizer always returns a fixed category -- tests values, not implementation.
type simpleCategorizer struct {
	category string
	err      error
}

func (c *simpleCategorizer) Categorize(_ context.Context, _ string, _ bool, _, _, _ string) (models.Category, error) {
	if c.err != nil {
		return models.Category{}, c.err
	}
	return models.Category{Name: c.category}, nil
}

func TestProcessTransactionsWithCategorizationStats_EmptyPartyNameWithCategorizer(t *testing.T) {
	// Transaction where GetPartyName returns "" AND all fallback fields are empty,
	// but a categorizer IS provided. This covers the "no party name available" debug path.
	txs := []models.Transaction{
		{
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "No party at all",
			Amount:      decimal.NewFromFloat(42.00),
			Currency:    "CHF",
			CreditDebit: models.TransactionTypeCredit,
			// All party name fields empty: Payee, Payer, PartyName, Name, Recipient
		},
	}

	mockCat := &simpleCategorizer{category: "ShouldNotReach"}
	result := ProcessTransactionsWithCategorizationStats(txs, nil, mockCat, "test")
	assert.Equal(t, "Uncategorized", result[0].Category)
}

func TestWriteTransactionsToCSVWithFormatter_ReadOnlyDir(t *testing.T) {
	// Create a directory, write-protect it, then attempt to create a file inside.
	// This triggers the os.Create failure after MkdirAll succeeds.
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0750))
	require.NoError(t, os.Chmod(readOnlyDir, 0555)) // #nosec G302 -- restrictive for testing
	t.Cleanup(func() {
		_ = os.Chmod(readOnlyDir, 0750) // #nosec G302 -- restore for cleanup
	})

	txs := sampleTransactions()
	f := &mockFormatter{
		header: []string{"A"},
		rows:   [][]string{{"v"}},
	}

	err := WriteTransactionsToCSVWithFormatter(txs, filepath.Join(readOnlyDir, "out.csv"), nil, f, ',')
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error creating CSV file")
}

func TestWriteTransactionsToCSVWithLogger_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0750))
	require.NoError(t, os.Chmod(readOnlyDir, 0555)) // #nosec G302 -- restrictive for testing
	t.Cleanup(func() {
		_ = os.Chmod(readOnlyDir, 0750) // #nosec G302 -- restore for cleanup
	})

	txs := sampleTransactions()
	err := WriteTransactionsToCSVWithLogger(txs, filepath.Join(readOnlyDir, "out.csv"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error creating CSV file")
}

// ---------------------------------------------------------------------------
// SanitizeAccountID edge cases (93.8% -> higher)
// ---------------------------------------------------------------------------

func TestSanitizeAccountID_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string returns UNKNOWN",
			input: "",
			want:  "UNKNOWN",
		},
		{
			name:  "only whitespace returns UNKNOWN",
			input: "   ",
			want:  "UNKNOWN",
		},
		{
			name:  "only special characters returns UNKNOWN",
			input: "@#$%^&*()",
			want:  "UNKNOWN",
		},
		{
			name:  "path traversal dots replaced",
			input: "foo..bar",
			want:  "foo_bar",
		},
		{
			name:  "multiple consecutive path traversals",
			input: "a....b",
			want:  "a_b",
		},
		{
			name:  "multiple consecutive underscores collapsed",
			input: "foo___bar",
			want:  "foo_bar",
		},
		{
			name:  "spaces become underscores",
			input: "hello world",
			want:  "hello_world",
		},
		{
			name:  "leading and trailing underscores trimmed",
			input: "_foo_",
			want:  "foo",
		},
		{
			name:  "leading and trailing dots trimmed",
			input: ".foo.",
			want:  "foo",
		},
		{
			name:  "mixed special chars replaced with underscore",
			input: "acc@123/456",
			want:  "acc_123_456",
		},
		{
			name:  "unicode characters replaced",
			input: "compte-epargne",
			want:  "compte-epargne",
		},
		{
			name:  "only dots returns UNKNOWN",
			input: "....",
			want:  "UNKNOWN",
		},
		{
			name:  "only underscores returns UNKNOWN",
			input: "____",
			want:  "UNKNOWN",
		},
		{
			name:  "dot-underscore combo at edges",
			input: "._test_.",
			want:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeAccountID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
