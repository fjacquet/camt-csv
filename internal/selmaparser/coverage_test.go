package selmaparser

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSelmaCSVHeader = "Date,Description,Bookkeeping No.,Fund,Amount,Currency,Number of Shares\n"
const validSelmaCSVRow = "2024-01-15,trade,12345,CSIF Bond,100.50,CHF,10\n"

func writeSelmaCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0600)
	require.NoError(t, err)
	return path
}

func TestAdapter_ValidateFormat(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		path := writeSelmaCSV(t, dir, "valid.csv", validSelmaCSVHeader+validSelmaCSVRow)

		valid, err := adapter.ValidateFormat(path)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("invalid file - missing headers", func(t *testing.T) {
		dir := t.TempDir()
		path := writeSelmaCSV(t, dir, "invalid.csv", "Foo,Bar\n1,2\n")

		valid, err := adapter.ValidateFormat(path)
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		valid, err := adapter.ValidateFormat("/nonexistent/file.csv")
		assert.Error(t, err)
		assert.False(t, valid)
	})
}

func TestCategorizeTransaction_AllBranches(t *testing.T) {
	tests := []struct {
		name       string
		investment string
		desc       string
		amount     decimal.Decimal
		wantCat    string
	}{
		{"Buy investment", "Buy", "trade", decimal.NewFromInt(100), "Investissements"},
		{"Sell investment", "Sell", "trade", decimal.NewFromInt(100), "Investissements"},
		{"Income", "Income", "income", decimal.NewFromInt(50), "Revenus Financiers"},
		{"Dividend investment", "Dividend", "div", decimal.NewFromInt(25), "Revenus Financiers"},
		{"Expense", "Expense", "fee", decimal.NewFromInt(10), "Frais Bancaires"},
		{"selma_fee desc", "", "selma_fee", decimal.NewFromInt(5), "Frais Bancaires"},
		{"cash_transfer desc", "", "cash_transfer", decimal.NewFromInt(200), "Revenus Financiers"},
		{"trade negative", "", "trade", decimal.NewFromInt(-100), "Investissements"},
		{"trade positive", "", "trade", decimal.NewFromInt(100), "Revenus Financiers"},
		{"dividend desc", "", "dividend", decimal.NewFromInt(10), "Revenus Financiers"},
		{"unknown - no category", "", "other", decimal.NewFromInt(10), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := models.Transaction{
				Investment:  tt.investment,
				Description: tt.desc,
				Amount:      tt.amount,
			}
			result := categorizeTransaction(tx)
			assert.Equal(t, tt.wantCat, result.Category)
		})
	}
}

func TestProcessTransactionsWithCategorizer_NilLogger(t *testing.T) {
	transactions := []models.Transaction{
		{Description: "test", Amount: decimal.NewFromInt(100)},
	}

	result, err := ProcessTransactionsWithCategorizer(context.Background(), transactions, nil, nil)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestParse_FrenchHeaders(t *testing.T) {
	csv := "Date,Description,Tenue des comptes n°,Nom du fonds,ISIN,Montant,Devise,Nombre d’actions\n" +
		"2026-09-07,trade,59301984361,iShares ETF,IE00BMDBMP28,-832.4,CHF,184.0\n"
	a := NewAdapter(logging.NewLogrusAdapter("info", "text"))
	ok, err := validateFormat(strings.NewReader(csv), nil)
	require.NoError(t, err)
	require.True(t, ok)
	txs, err := a.Parse(context.Background(), strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, "IE00BMDBMP28", txs[0].PartyName)
	assert.Equal(t, "59301984361", txs[0].BookkeepingNumber)
	assert.Equal(t, "-832.4", txs[0].Amount.String())
}

func TestAdapter_SecurityNames(t *testing.T) {
	csv := "Date,Description,Bookkeeping No.,Fund,Amount,Currency,Number of Shares\n" +
		"2026-09-07,trade,1,IE00BMDBMP28,-8,CHF,1\n2026-09-07,trade,2,XX0000000000,-8,CHF,1\n"
	a := NewAdapter(logging.NewLogrusAdapter("info", "text"))
	a.SetSecurityNames(map[string]string{"IE00BMDBMP28": "IE00BMDBMP28-iShares"})
	txs, err := a.Parse(context.Background(), strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, txs, 2)
	assert.Equal(t, "IE00BMDBMP28-iShares", txs[0].Fund)
	assert.Equal(t, "XX0000000000", txs[1].Fund) // unknown: left bare, warned
}
