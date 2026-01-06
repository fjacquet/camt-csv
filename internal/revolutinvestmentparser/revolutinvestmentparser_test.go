package revolutinvestmentparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCategorizer implements models.TransactionCategorizer for testing
type MockCategorizer struct {
	mock.Mock
}

func (m *MockCategorizer) Categorize(partyName string, isDebtor bool, amount, date, description string) (models.Category, error) {
	args := m.Called(partyName, isDebtor, amount, date, description)
	return args.Get(0).(models.Category), args.Error(1)
}

func TestMain(m *testing.M) {
	// Tests will use the default logger
	os.Exit(m.Run())
}

func TestParseFile(t *testing.T) {
	// Create a temporary CSV file for testing
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722
2025-05-30T10:31:05.452Z,2B7K,BUY - MARKET,39.81059277,€11.40,€454,EUR,1.0722
2025-06-01T18:28:51.951827Z,,CASH TOP-UP,,,€198.68,EUR,1.0729`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	require.NoError(t, err)

	file, err := os.Open(tmpFile)
	require.NoError(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file %s: %v", tmpFile, err)
		}
	}()

	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)
	transactions, err := adapter.Parse(file)
	require.NoError(t, err)
	assert.Len(t, transactions, 3)

	// Check first transaction (CASH TOP-UP)
	txn1 := transactions[0]
	assert.Equal(t, 2025, txn1.Date.Year())
	assert.Equal(t, time.May, txn1.Date.Month())
	assert.Equal(t, 30, txn1.Date.Day())
	assert.Equal(t, 2025, txn1.ValueDate.Year())
	assert.Equal(t, time.May, txn1.ValueDate.Month())
	assert.Equal(t, 30, txn1.ValueDate.Day())
	assert.Equal(t, "Revolut Investment", txn1.PartyName)
	assert.Equal(t, "Cash top-up to investment account", txn1.Description)
	assert.Equal(t, "EUR", txn1.Currency)
	assert.Equal(t, "EUR", txn1.OriginalCurrency)
	assert.Equal(t, models.TransactionTypeCredit, txn1.CreditDebit)
	assert.False(t, txn1.DebitFlag)

	// Check second transaction (BUY)
	txn2 := transactions[1]
	assert.Equal(t, 2025, txn2.Date.Year())
	assert.Equal(t, time.May, txn2.Date.Month())
	assert.Equal(t, 30, txn2.Date.Day())
	assert.Equal(t, "2B7K", txn2.Investment)
	assert.Equal(t, "2B7K", txn2.Fund)
	assert.Equal(t, "BUY - MARKET", txn2.Type)
	assert.Equal(t, "Buy 39.81059277 shares of 2B7K", txn2.Description)
	assert.Equal(t, "Revolut Investment - 2B7K", txn2.PartyName)
	assert.Equal(t, 39, txn2.NumberOfShares)
}

func TestParseFile_InvalidFormat(t *testing.T) {
	// Test with incorrect format
	invalidContent := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
		topup,Current,29.05.2025,30.05.2025,Top-up,,0,EUR,COMPLETED,€1000`

	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.csv")
	err := os.WriteFile(invalidFile, []byte(invalidContent), 0600)
	require.NoError(t, err)

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
	require.Error(t, err)
}

func TestConvertRowToTransaction(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:          "2025-05-30T10:31:05.452Z",
		Ticker:        "2B7K",
		Type:          "BUY - MARKET",
		Quantity:      "39.81059277",
		PricePerShare: "€11.40",
		TotalAmount:   "€454",
		Currency:      "EUR",
		FXRate:        "1.0722",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	txn, err := convertRowToTransaction(row, logger)
	require.NoError(t, err)

	assert.Equal(t, 2025, txn.Date.Year())
	assert.Equal(t, time.May, txn.Date.Month())
	assert.Equal(t, 30, txn.Date.Day())
	assert.Equal(t, "2B7K", txn.Investment)
	assert.Equal(t, "BUY - MARKET", txn.Type)
	assert.Equal(t, "EUR", txn.Currency)
	assert.Equal(t, "EUR", txn.OriginalCurrency)
}

func TestFormatDate(t *testing.T) {
	formatted := formatDate("2025-05-30T10:31:05.452Z")
	assert.Equal(t, 2025, formatted.Year())
	assert.Equal(t, time.May, formatted.Month())
	assert.Equal(t, 30, formatted.Day())

	// Test with invalid date
	invalid := formatDate("invalid-date")
	assert.True(t, invalid.IsZero()) // Should return zero time for invalid date
}

func TestWriteToCSV(t *testing.T) {
	transactions := []models.Transaction{
		{
			Date:           time.Date(2025, 5, 30, 0, 0, 0, 0, time.UTC),
			Description:    "Test transaction",
			Amount:         models.ParseAmount("100"),
			Currency:       "EUR",
			CreditDebit:    models.TransactionTypeCredit,
			DebitFlag:      false,
			PartyName:      "Test Party",
			Name:           "Test Party",
			Investment:     "TEST",
			Type:           "BUY",
			NumberOfShares: 10,
		},
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.csv")

	err := WriteToCSV(transactions, tmpFile)
	require.NoError(t, err)

	// Check that file was created
	_, err = os.Stat(tmpFile)
	require.NoError(t, err)
}

func TestConvertToCSV(t *testing.T) {
	// Create a temporary input CSV file
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722
2025-05-30T10:31:05.452Z,2B7K,BUY - MARKET,39.81059277,€11.40,€454,EUR,1.0722`

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.csv")
	outputFile := filepath.Join(tmpDir, "output.csv")

	err := os.WriteFile(inputFile, []byte(content), 0600)
	require.NoError(t, err)

	err = ConvertToCSV(inputFile, outputFile)
	require.NoError(t, err)

	// Check that output file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

// Additional tests for better coverage

func TestParse_WithNilLogger(t *testing.T) {
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722`

	reader := strings.NewReader(content)
	transactions, err := Parse(reader, nil) // nil logger
	require.NoError(t, err)
	assert.Len(t, transactions, 1)
}

func TestParseWithCategorizer_Success(t *testing.T) {
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722`

	reader := strings.NewReader(content)
	logger := logging.NewLogrusAdapter("info", "text")
	
	mockCategorizer := &MockCategorizer{}
	mockCategorizer.On("Categorize", "Revolut Investment", false, "454", "30.05.2025", "").Return(models.Category{Name: "Investment"}, nil)

	transactions, err := ParseWithCategorizer(reader, logger, mockCategorizer)
	require.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, "Investment", transactions[0].Category)
	
	mockCategorizer.AssertExpectations(t)
}

func TestParseWithCategorizer_CategorizerError(t *testing.T) {
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722`

	reader := strings.NewReader(content)
	logger := logging.NewLogrusAdapter("info", "text")
	
	mockCategorizer := &MockCategorizer{}
	mockCategorizer.On("Categorize", "Revolut Investment", false, "454", "30.05.2025", "").Return(models.Category{}, assert.AnError)

	transactions, err := ParseWithCategorizer(reader, logger, mockCategorizer)
	require.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Equal(t, models.CategoryUncategorized, transactions[0].Category)
	
	mockCategorizer.AssertExpectations(t)
}

func TestParseWithCategorizer_EmptyFile(t *testing.T) {
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate`

	reader := strings.NewReader(content)
	logger := logging.NewLogrusAdapter("info", "text")

	_, err := ParseWithCategorizer(reader, logger, nil)
	require.Error(t, err)
	
	var invalidFormatErr *parsererror.InvalidFormatError
	assert.ErrorAs(t, err, &invalidFormatErr)
	assert.Contains(t, err.Error(), "empty or contains only headers")
}

func TestParseWithCategorizer_InsufficientColumns(t *testing.T) {
	content := `Date,Ticker
2025-05-30T10:31:02.786456Z,TEST`

	reader := strings.NewReader(content)
	logger := logging.NewLogrusAdapter("info", "text")

	_, err := ParseWithCategorizer(reader, logger, nil)
	require.Error(t, err)
	
	var invalidFormatErr *parsererror.InvalidFormatError
	assert.ErrorAs(t, err, &invalidFormatErr)
	assert.Contains(t, err.Error(), "insufficient columns")
}

func TestParseWithCategorizer_WrongHeaders(t *testing.T) {
	content := `Wrong,Headers,Here,Test,Test2,Test3,Test4,Test5
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722`

	reader := strings.NewReader(content)
	logger := logging.NewLogrusAdapter("info", "text")

	_, err := ParseWithCategorizer(reader, logger, nil)
	require.Error(t, err)
	
	var invalidFormatErr *parsererror.InvalidFormatError
	assert.ErrorAs(t, err, &invalidFormatErr)
	assert.Contains(t, err.Error(), "unexpected header")
}

func TestParseWithCategorizer_SkipInsufficientRowColumns(t *testing.T) {
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722
2025-05-30T10:31:05.452Z,2B7K,BUY - MARKET,39.81059277,€11.40,€454,EUR,1.0722`

	reader := strings.NewReader(content)
	logger := logging.NewLogrusAdapter("info", "text")

	transactions, err := ParseWithCategorizer(reader, logger, nil)
	require.NoError(t, err)
	assert.Len(t, transactions, 2) // Should process both valid rows
}

func TestConvertRowToTransaction_DividendType(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Ticker:      "AAPL",
		Type:        "DIVIDEND",
		TotalAmount: "$5.50",
		Currency:    "USD",
		FXRate:      "1.0",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	txn, err := convertRowToTransaction(row, logger)
	require.NoError(t, err)

	assert.Equal(t, "Dividend from AAPL", txn.Description)
	assert.Equal(t, models.TransactionTypeCredit, txn.CreditDebit)
	assert.Equal(t, "5.5", txn.Amount.String()) // Decimal removes trailing zeros
}

func TestConvertRowToTransaction_CashTopUpType(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Type:        "CASH TOP-UP",
		TotalAmount: "€1000",
		Currency:    "EUR",
		FXRate:      "1.0722",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	txn, err := convertRowToTransaction(row, logger)
	require.NoError(t, err)

	assert.Equal(t, "Cash top-up to investment account", txn.Description)
	assert.Equal(t, models.TransactionTypeCredit, txn.CreditDebit)
	assert.Equal(t, "1000", txn.Amount.String())
}

func TestConvertRowToTransaction_DefaultType(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Ticker:      "TSLA",
		Type:        "SELL",
		TotalAmount: "$500",
		Currency:    "USD",
		FXRate:      "1.0",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	txn, err := convertRowToTransaction(row, logger)
	require.NoError(t, err)

	assert.Equal(t, "SELL transaction for TSLA", txn.Description)
	assert.Equal(t, models.TransactionTypeDebit, txn.CreditDebit)
}

func TestConvertRowToTransaction_InvalidQuantity(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Ticker:      "TEST",
		Type:        "BUY - MARKET",
		Quantity:    "invalid-quantity",
		TotalAmount: "€100",
		Currency:    "EUR",
		FXRate:      "1.0",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	_, err := convertRowToTransaction(row, logger)
	require.Error(t, err)
	
	var dataExtractionErr *parsererror.DataExtractionError
	assert.ErrorAs(t, err, &dataExtractionErr)
	assert.Contains(t, err.Error(), "failed to parse quantity")
}

func TestConvertRowToTransaction_InvalidPricePerShare(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:          "2025-05-30T10:31:05.452Z",
		Ticker:        "TEST",
		Type:          "BUY - MARKET",
		Quantity:      "10",
		PricePerShare: "invalid-price",
		TotalAmount:   "€100",
		Currency:      "EUR",
		FXRate:        "1.0",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	_, err := convertRowToTransaction(row, logger)
	require.Error(t, err)
	
	var dataExtractionErr *parsererror.DataExtractionError
	assert.ErrorAs(t, err, &dataExtractionErr)
	assert.Contains(t, err.Error(), "failed to parse price per share")
}

func TestConvertRowToTransaction_InvalidTotalAmount(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Ticker:      "TEST",
		Type:        "BUY - MARKET",
		Quantity:    "10",
		TotalAmount: "invalid-amount",
		Currency:    "EUR",
		FXRate:      "1.0",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	_, err := convertRowToTransaction(row, logger)
	require.Error(t, err)
	
	var dataExtractionErr *parsererror.DataExtractionError
	assert.ErrorAs(t, err, &dataExtractionErr)
	assert.Contains(t, err.Error(), "failed to parse total amount")
}

func TestConvertRowToTransaction_InvalidFXRate(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Type:        "CASH TOP-UP",
		TotalAmount: "€100",
		Currency:    "EUR",
		FXRate:      "invalid-rate",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	txn, err := convertRowToTransaction(row, logger)
	require.NoError(t, err)
	
	// Should default to 1.0 for invalid FX rate
	assert.Equal(t, "1", txn.ExchangeRate.String())
}

func TestConvertRowToTransaction_EmptyFXRate(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Type:        "CASH TOP-UP",
		TotalAmount: "€100",
		Currency:    "EUR",
		FXRate:      "",
	}

	logger := logging.NewLogrusAdapter("info", "text")
	txn, err := convertRowToTransaction(row, logger)
	require.NoError(t, err)
	
	// Should default to 1.0 for empty FX rate
	assert.Equal(t, "1", txn.ExchangeRate.String())
}

func TestConvertRowToTransaction_NilLogger(t *testing.T) {
	row := RevolutInvestmentCSVRow{
		Date:        "2025-05-30T10:31:05.452Z",
		Type:        "CASH TOP-UP",
		TotalAmount: "€100",
		Currency:    "EUR",
		FXRate:      "1.0",
	}

	txn, err := convertRowToTransaction(row, nil) // nil logger
	require.NoError(t, err)
	assert.Equal(t, "Cash top-up to investment account", txn.Description)
}

func TestCleanAmountString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"€100.50", "100.50"},
		{"$1,234.56", "1234.56"},
		{"£999.99", "999.99"},
		{"1,000", "1000"},
		{"500", "500"},
		{"€1,234,567.89", "1234567.89"},
	}

	for _, test := range tests {
		result := cleanAmountString(test.input)
		assert.Equal(t, test.expected, result, "Input: %s", test.input)
	}
}

func TestWriteToCSVWithLogger_NilLogger(t *testing.T) {
	transactions := []models.Transaction{
		{
			Date:        time.Date(2025, 5, 30, 0, 0, 0, 0, time.UTC),
			Description: "Test transaction",
			Amount:      models.ParseAmount("100"),
			Currency:    "EUR",
		},
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.csv")

	err := WriteToCSVWithLogger(transactions, tmpFile, nil) // nil logger
	require.NoError(t, err)

	// Check that file was created
	_, err = os.Stat(tmpFile)
	require.NoError(t, err)
}

func TestConvertToCSVWithLogger_NilLogger(t *testing.T) {
	// Create a temporary input CSV file
	content := `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722`

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.csv")
	outputFile := filepath.Join(tmpDir, "output.csv")

	err := os.WriteFile(inputFile, []byte(content), 0600)
	require.NoError(t, err)

	err = ConvertToCSVWithLogger(inputFile, outputFile, nil) // nil logger
	require.NoError(t, err)

	// Check that output file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}

func TestConvertToCSVWithLogger_FileOpenError(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	
	err := ConvertToCSVWithLogger("/nonexistent/input.csv", "/tmp/output.csv", logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open input file")
}

func TestWriteToCSVWithLogger_CreateFileError(t *testing.T) {
	transactions := []models.Transaction{
		{
			Date:        time.Date(2025, 5, 30, 0, 0, 0, 0, time.UTC),
			Description: "Test transaction",
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")
	
	err := WriteToCSVWithLogger(transactions, "/nonexistent/directory/output.csv", logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create file")
}
