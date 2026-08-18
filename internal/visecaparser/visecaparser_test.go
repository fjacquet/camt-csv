package visecaparser

import (
	"context"
	"os"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const visecaHeader = "TransactionId,CardId,Date,ValutaDate,Amount,Currency,OriginalAmount," +
	"OriginalCurrency,MerchantName,MerchantPlace,MerchantCountry,StateType,Details,Type,Exchange Rate"

// visecaCSV assembles a Viseca export from the given data rows.
func visecaCSV(rows ...string) string {
	return visecaHeader + "\n" + strings.Join(rows, "\n") + "\n"
}

const (
	purchaseRow = "TRX2025030300002685972,462723MDBDQN4748,2025-03-01 15:12:22,2025-03-03 00:00:00," +
		"29.000,CHF,29.000,CHF,Jschalp Gastro AG,Davos Platz,CHE,BOOKED,Jschalp Gastro AG,merchant,1.000000"
	refundRow = "TRX2024121900001111111,462723MDBDQN4748,2024-12-19 09:00:00,2024-12-19 00:00:00," +
		"-105.700,CHF,-105.700,CHF,Levis Online Shop,,,BOOKED,Levis Online Shop,merchant,1.000000"
	foreignRow = "TRX2025050500004590684,462723MDBDQN4748,2025-05-05 10:16:57,2025-05-05 00:00:00," +
		"12.900,CHF,15.000,USD,Windsurf,,,BOOKED,WINDSURF,merchant,0.848252"
	noMerchantRow = "TRX2025052800003333333,462723MDBDQN4748,2025-05-28 12:00:00,2025-05-28 00:00:00," +
		"80.000,CHF,80.000,CHF,,,,BOOKED,APPLE.COM/BILL,merchant,1.000000"
)

func parseCSV(t *testing.T, csv string) []models.Transaction {
	t.Helper()
	txs, err := ParseWithCategorizer(
		context.Background(), strings.NewReader(csv), logging.NewLogrusAdapter("error", "text"), nil)
	require.NoError(t, err)
	return txs
}

// TestParse_PurchaseIsDebitWithNegativeAmount pins the sign convention. Viseca
// writes card-issuer signs, where a purchase is positive; every parser here
// emits debit-negative amounts, so the values must be inverted. Getting this
// backwards flips the sign of every imported transaction.
func TestParse_PurchaseIsDebitWithNegativeAmount(t *testing.T) {
	txs := parseCSV(t, visecaCSV(purchaseRow))
	require.Len(t, txs, 1)

	assert.Equal(t, models.TransactionTypeDebit, txs[0].CreditDebit)
	assert.Equal(t, "-29", txs[0].Amount.String())
	assert.True(t, txs[0].DebitFlag)
}

func TestParse_RefundIsCreditWithPositiveAmount(t *testing.T) {
	txs := parseCSV(t, visecaCSV(refundRow))
	require.Len(t, txs, 1)

	assert.Equal(t, models.TransactionTypeCredit, txs[0].CreditDebit)
	assert.Equal(t, "105.7", txs[0].Amount.String())
	assert.False(t, txs[0].DebitFlag)
}

func TestParse_MapsCoreFields(t *testing.T) {
	txs := parseCSV(t, visecaCSV(purchaseRow))
	require.Len(t, txs, 1)
	tx := txs[0]

	assert.Equal(t, "TRX2025030300002685972", tx.BookkeepingNumber)
	assert.Equal(t, "2025-03-01", tx.Date.Format("2006-01-02"))
	assert.Equal(t, "2025-03-03", tx.ValueDate.Format("2006-01-02"))
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "BOOK", tx.Status)
	assert.Equal(t, "merchant", tx.Product)
	assert.Empty(t, tx.Type, "Type must stay empty or Build copies it into InvestmentType")
	assert.Empty(t, tx.Investment)
	assert.Equal(t, "Jschalp Gastro AG", tx.Name)
	assert.Equal(t, "Jschalp Gastro AG", tx.PartyName)
	assert.Equal(t, "Jschalp Gastro AG", tx.Description)
}

// TestParse_FallsBackToDetailsForName covers the rows where Viseca leaves
// MerchantName empty; without a fallback these would categorize as blank.
func TestParse_FallsBackToDetailsForName(t *testing.T) {
	txs := parseCSV(t, visecaCSV(noMerchantRow))
	require.Len(t, txs, 1)

	assert.Equal(t, "APPLE.COM/BILL", txs[0].Name)
	assert.Equal(t, "APPLE.COM/BILL", txs[0].PartyName)
}

func TestParse_CapturesForeignCurrency(t *testing.T) {
	txs := parseCSV(t, visecaCSV(foreignRow))
	require.Len(t, txs, 1)
	tx := txs[0]

	assert.Equal(t, "-12.9", tx.Amount.String())
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "USD", tx.OriginalCurrency)
	assert.Equal(t, "15", tx.OriginalAmount.String())
	assert.Equal(t, "0.848252", tx.ExchangeRate.String())
}

// TestParse_StripsByteOrderMark guards the UTF-8 BOM the Viseca export starts
// with: left in place it becomes part of the first header name, so every
// column lookup by name fails.
func TestParse_StripsByteOrderMark(t *testing.T) {
	txs := parseCSV(t, "\ufeff"+visecaCSV(purchaseRow))
	require.Len(t, txs, 1)
	assert.Equal(t, "TRX2025030300002685972", txs[0].BookkeepingNumber)
}

func TestParse_SkipsMalformedAndEmptyRows(t *testing.T) {
	txs := parseCSV(t, visecaCSV(purchaseRow, "", refundRow))
	assert.Len(t, txs, 2)
}

func TestValidateFormat_AcceptsVisecaExport(t *testing.T) {
	valid, err := validateFormat(strings.NewReader(visecaCSV(purchaseRow)), nil)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestValidateFormat_RejectsOtherCSV(t *testing.T) {
	for name, csv := range map[string]string{
		"selma":   "Date,Description,Bookkeeping No.,Fund,Amount,Currency,Number of Shares\n",
		"revolut": "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n",
		"empty":   "",
	} {
		t.Run(name, func(t *testing.T) {
			valid, _ := validateFormat(strings.NewReader(csv), nil)
			assert.False(t, valid)
		})
	}
}

// TestParse_CommittedSample exercises the checked-in testdata sample, which carries the
// row shapes the inline fixtures cover individually: CHF and foreign-currency
// purchases, refunds, both Type values and a blank MerchantName.
func TestParse_CommittedSample(t *testing.T) {
	f, err := os.Open("testdata/viseca.csv")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	txs, err := ParseWithCategorizer(
		context.Background(), f, logging.NewLogrusAdapter("error", "text"), nil)
	require.NoError(t, err)
	require.Len(t, txs, 9)

	ids := map[string]bool{}
	for _, tx := range txs {
		assert.NotEmpty(t, tx.BookkeepingNumber, "every row carries a transaction id")
		assert.False(t, ids[tx.BookkeepingNumber], "transaction ids are unique")
		ids[tx.BookkeepingNumber] = true

		assert.NotEmpty(t, tx.Name, "MerchantName or its Details fallback must fill Name")
		assert.False(t, tx.Amount.IsZero())
		assert.Equal(t, tx.Amount.IsNegative(), tx.CreditDebit == models.TransactionTypeDebit)
	}

	valid, err := validateFormat(mustOpen(t, "testdata/viseca.csv"), nil)
	require.NoError(t, err)
	assert.True(t, valid)
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}
