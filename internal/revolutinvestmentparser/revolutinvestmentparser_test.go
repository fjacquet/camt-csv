package revolutinvestmentparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set up logging for tests
	logger = logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress log output during tests
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

	adapter := NewAdapter()
	transactions, err := adapter.Parse(file)
	require.NoError(t, err)
	assert.Len(t, transactions, 3)

	// Check first transaction (CASH TOP-UP)
	txn1 := transactions[0]
	assert.Equal(t, "30.05.2025", txn1.Date)
	assert.Equal(t, "30.05.2025", txn1.ValueDate)
	assert.Equal(t, "Revolut Investment", txn1.PartyName)
	assert.Equal(t, "Cash top-up to investment account", txn1.Description)
	assert.Equal(t, "EUR", txn1.Currency)
	assert.Equal(t, "EUR", txn1.OriginalCurrency)
	assert.Equal(t, "CRDT", txn1.CreditDebit)
	assert.False(t, txn1.DebitFlag)

	// Check second transaction (BUY)
	txn2 := transactions[1]
	assert.Equal(t, "30.05.2025", txn2.Date)
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

	adapter := NewAdapter()
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

	txn, err := convertRowToTransaction(row)
	require.NoError(t, err)

	assert.Equal(t, "30.05.2025", txn.Date)
	assert.Equal(t, "2B7K", txn.Investment)
	assert.Equal(t, "BUY - MARKET", txn.Type)
	assert.Equal(t, "EUR", txn.Currency)
	assert.Equal(t, "EUR", txn.OriginalCurrency)
}

func TestFormatDate(t *testing.T) {
	formatted := formatDate("2025-05-30T10:31:05.452Z")
	assert.Equal(t, "30.05.2025", formatted)

	// Test with invalid date
	invalid := formatDate("invalid-date")
	assert.Equal(t, "invalid-date", invalid)
}

func TestWriteToCSV(t *testing.T) {
	transactions := []models.Transaction{
		{
			Date:           "30.05.2025",
			Description:    "Test transaction",
			Amount:         models.ParseAmount("100"),
			Currency:       "EUR",
			CreditDebit:    "CRDT",
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
