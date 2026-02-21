package selmaparser

import (
	"context"
	"os"
	"path/filepath"
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

func TestAdapter_BatchConvert(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	n, err := adapter.BatchConvert(context.Background(), "/tmp", "/tmp")
	assert.Error(t, err)
	assert.Equal(t, 0, n)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestAdapter_ConvertToCSV(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	t.Run("nonexistent input file", func(t *testing.T) {
		err := adapter.ConvertToCSV(context.Background(), "/nonexistent/file.csv", "/tmp/out.csv")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error opening input file")
	})

	t.Run("valid input produces output", func(t *testing.T) {
		dir := t.TempDir()
		inputPath := writeSelmaCSV(t, dir, "input.csv", validSelmaCSVHeader+validSelmaCSVRow)
		outputPath := filepath.Join(dir, "output.csv")

		err := adapter.ConvertToCSV(context.Background(), inputPath, outputPath)
		assert.NoError(t, err)

		// Verify output file exists
		_, statErr := os.Stat(outputPath)
		assert.NoError(t, statErr)
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

	result := ProcessTransactionsWithCategorizer(transactions, nil, nil)
	assert.Len(t, result, 1)
}
