package revolutcryptoparser

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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock categorizer ---

type mockCategorizer struct {
	mock.Mock
}

func (m *mockCategorizer) Categorize(ctx context.Context, partyName string, isDebtor bool, amount, date, description string) (models.Category, error) {
	args := m.Called(ctx, partyName, isDebtor, amount, date, description)
	return args.Get(0).(models.Category), args.Error(1)
}

func newTestLogger() logging.Logger {
	return logging.NewLogrusAdapter("info", "text")
}

func validCryptoCSV() string {
	return `Symbol,Type,Quantity,Price,Value,Fees,Date
BTC,Achat,"0,00014206","69 924,87 CHF","9,94 CHF","0,25 CHF","25 janv. 2026, 13:15:23"
DOT,Récompense de staking,"0,00523871",,,,"15 mars 2026, 08:00:00"
`
}

// --- parseFrenchDate tests ---

func TestParseFrenchDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
		wantYear int
		wantMon  time.Month
		wantDay  int
	}{
		{"valid January", "25 janv. 2026, 13:15:23", false, 2026, time.January, 25},
		{"valid March", "15 mars 2026, 08:00:00", false, 2026, time.March, 15},
		{"valid December", "31 déc. 2025, 23:59:59", false, 2025, time.December, 31},
		{"invalid month", "25 xyz. 2026, 13:15:23", true, 0, 0, 0},
		{"too few parts", "25 janv. 2026", true, 0, 0, 0},
		{"empty string", "", true, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFrenchDate(tt.input)
			if tt.wantZero {
				assert.True(t, got.IsZero())
			} else {
				assert.Equal(t, tt.wantYear, got.Year())
				assert.Equal(t, tt.wantMon, got.Month())
				assert.Equal(t, tt.wantDay, got.Day())
			}
		})
	}
}

// --- parseFrenchAmount tests ---

func TestParseFrenchAmount(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantAmount   string
		wantCurrency string
	}{
		{"with CHF", "69 924,87 CHF", "69924.87", "CHF"},
		{"with EUR", "454,00 EUR", "454", "EUR"},
		{"simple", "9,94 CHF", "9.94", "CHF"},
		{"no currency", "123,45", "123.45", ""},
		{"empty", "", "0", ""},
		{"zero", "0,00 CHF", "0", "CHF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, currency := parseFrenchAmount(tt.input)
			assert.Equal(t, tt.wantAmount, amount.String())
			assert.Equal(t, tt.wantCurrency, currency)
		})
	}
}

// --- ParseWithCategorizer tests ---

func TestParseWithCategorizer_ValidCSV(t *testing.T) {
	logger := newTestLogger()
	cat := &mockCategorizer{}
	cat.On("Categorize", mock.Anything, "Revolut Crypto - BTC", true, mock.Anything, mock.Anything, mock.Anything).
		Return(models.Category{Name: "Crypto"}, nil)
	cat.On("Categorize", mock.Anything, "Revolut Crypto - DOT", false, mock.Anything, mock.Anything, mock.Anything).
		Return(models.Category{Name: "Staking"}, nil)

	transactions, err := ParseWithCategorizer(strings.NewReader(validCryptoCSV()), logger, cat)
	require.NoError(t, err)
	assert.Len(t, transactions, 2)

	assert.Equal(t, "Crypto", transactions[0].Category)
	assert.Contains(t, transactions[0].Description, "Achat BTC")
	assert.Equal(t, "Revolut Crypto - BTC", transactions[0].PartyName)

	assert.Equal(t, "Staking", transactions[1].Category)
	assert.Contains(t, transactions[1].Description, "Récompense de staking DOT")

	cat.AssertExpectations(t)
}

func TestParseWithCategorizer_NilCategorizer(t *testing.T) {
	logger := newTestLogger()
	transactions, err := ParseWithCategorizer(strings.NewReader(validCryptoCSV()), logger, nil)
	require.NoError(t, err)
	assert.Len(t, transactions, 2)
	assert.Equal(t, models.CategoryUncategorized, transactions[0].Category)
	assert.Equal(t, models.CategoryUncategorized, transactions[1].Category)
}

func TestParseWithCategorizer_CategorizerError(t *testing.T) {
	logger := newTestLogger()
	cat := &mockCategorizer{}
	cat.On("Categorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(models.Category{}, fmt.Errorf("API error"))

	transactions, err := ParseWithCategorizer(strings.NewReader(validCryptoCSV()), logger, cat)
	require.NoError(t, err)
	assert.Len(t, transactions, 2)
	assert.Equal(t, models.CategoryUncategorized, transactions[0].Category)
}

func TestParseWithCategorizer_NilLogger(t *testing.T) {
	transactions, err := ParseWithCategorizer(strings.NewReader(validCryptoCSV()), nil, nil)
	require.NoError(t, err)
	assert.Len(t, transactions, 2)
}

func TestParseWithCategorizer_EmptyCSV(t *testing.T) {
	logger := newTestLogger()
	_, err := ParseWithCategorizer(strings.NewReader("Symbol,Type,Quantity,Price,Value,Fees,Date\n"), logger, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseWithCategorizer_InvalidHeaders(t *testing.T) {
	logger := newTestLogger()
	csv := "A,B,C,D,E,F,G\n1,2,3,4,5,6,7\n"
	_, err := ParseWithCategorizer(strings.NewReader(csv), logger, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected header")
}

func TestParseWithCategorizer_InsufficientColumns(t *testing.T) {
	logger := newTestLogger()
	csv := "A,B\n1,2\n"
	_, err := ParseWithCategorizer(strings.NewReader(csv), logger, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient columns")
}

func TestParseWithCategorizer_ShortRowsCauseCSVError(t *testing.T) {
	logger := newTestLogger()
	// csv.ReadAll rejects rows with mismatched field counts
	csv := "Symbol,Type,Quantity,Price,Value,Fees,Date\nBTC,Achat,0.1\n"
	_, err := ParseWithCategorizer(strings.NewReader(csv), logger, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read CSV")
}

func TestParseWithCategorizer_UnknownType(t *testing.T) {
	logger := newTestLogger()
	csv := `Symbol,Type,Quantity,Price,Value,Fees,Date
ETH,Vente,"1,0","3 000,00 CHF","3 000,00 CHF","0,50 CHF","10 févr. 2026, 14:00:00"
`
	transactions, err := ParseWithCategorizer(strings.NewReader(csv), logger, nil)
	require.NoError(t, err)
	assert.Len(t, transactions, 1)
	assert.Contains(t, transactions[0].Description, "Vente ETH")
}

// --- convertRowToTransaction tests ---

func TestConvertRowToTransaction_Achat(t *testing.T) {
	row := cryptoCSVRow{
		Symbol:   "BTC",
		Type:     "Achat",
		Quantity: "0,00014206",
		Price:    "69 924,87 CHF",
		Value:    "9,94 CHF",
		Fees:     "0,25 CHF",
		Date:     "25 janv. 2026, 13:15:23",
	}
	tx, err := convertRowToTransaction(row)
	require.NoError(t, err)
	assert.Equal(t, "Revolut Crypto - BTC", tx.PartyName)
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, models.TransactionTypeDebit, tx.CreditDebit)
	assert.Contains(t, tx.Description, "Achat BTC")
}

func TestConvertRowToTransaction_StakingReward(t *testing.T) {
	row := cryptoCSVRow{
		Symbol:   "DOT",
		Type:     "Récompense de staking",
		Quantity: "0,00523871",
		Date:     "15 mars 2026, 08:00:00",
	}
	tx, err := convertRowToTransaction(row)
	require.NoError(t, err)
	assert.Equal(t, models.TransactionTypeCredit, tx.CreditDebit)
	assert.Contains(t, tx.Description, "Récompense de staking DOT")
}

func TestConvertRowToTransaction_AchatNoCurrency(t *testing.T) {
	row := cryptoCSVRow{
		Symbol:   "ETH",
		Type:     "Achat",
		Quantity: "1,0",
		Value:    "3000,50",
		Fees:     "1,00",
		Date:     "10 avr. 2026, 10:00:00",
	}
	tx, err := convertRowToTransaction(row)
	require.NoError(t, err)
	assert.Equal(t, "CHF", tx.Currency) // defaults to CHF
}

// --- Adapter tests ---

func TestAdapter_ValidateFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid file
	validPath := filepath.Join(tmpDir, "valid.csv")
	err := os.WriteFile(validPath, []byte("Symbol,Type,Quantity,Price,Value,Fees,Date\nBTC,Achat,1,100,100,1,now\n"), 0600)
	require.NoError(t, err)

	adapter := NewAdapter(newTestLogger())
	ok, err := adapter.ValidateFormat(validPath)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Invalid file (wrong headers)
	invalidPath := filepath.Join(tmpDir, "invalid.csv")
	err = os.WriteFile(invalidPath, []byte("A,B,C\n1,2,3\n"), 0600)
	require.NoError(t, err)
	ok, err = adapter.ValidateFormat(invalidPath)
	assert.NoError(t, err)
	assert.False(t, ok)

	// Non-existent file
	ok, err = adapter.ValidateFormat(filepath.Join(tmpDir, "nope.csv"))
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestAdapter_Parse(t *testing.T) {
	adapter := NewAdapter(newTestLogger())
	transactions, err := adapter.Parse(context.Background(), strings.NewReader(validCryptoCSV()))
	require.NoError(t, err)
	assert.Len(t, transactions, 2)
}

func TestAdapter_ConvertToCSV(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.csv")
	outputPath := filepath.Join(tmpDir, "output.csv")

	err := os.WriteFile(inputPath, []byte(validCryptoCSV()), 0600)
	require.NoError(t, err)

	adapter := NewAdapter(newTestLogger())
	err = adapter.ConvertToCSV(context.Background(), inputPath, outputPath)
	assert.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "BTC")
}

func TestAdapter_BatchConvert(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Write one valid file
	err := os.WriteFile(filepath.Join(inputDir, "crypto.csv"), []byte(validCryptoCSV()), 0600)
	require.NoError(t, err)

	// Write one invalid file
	err = os.WriteFile(filepath.Join(inputDir, "bad.csv"), []byte("not,a,valid,file\n"), 0600)
	require.NoError(t, err)

	adapter := NewAdapter(newTestLogger())
	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAdapter_BatchConvert_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "empty")
	outputDir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	adapter := NewAdapter(newTestLogger())
	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestAdapter_BatchConvert_InvalidInputDir(t *testing.T) {
	adapter := NewAdapter(newTestLogger())
	_, err := adapter.BatchConvert(context.Background(), "/nonexistent/dir", "/tmp/out")
	assert.Error(t, err)
}
