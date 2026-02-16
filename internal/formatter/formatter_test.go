package formatter

import (
	"testing"
	"time"

	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestTransaction creates a sample transaction for testing
func createTestTransaction() models.Transaction {
	tx := models.Transaction{
		BookkeepingNumber: "TXN001",
		Status:            "BOOK",
		Date:              time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		Name:              "Coffee Shop",
		PartyName:         "Coffee Shop Inc",
		PartyIBAN:         "CH1234567890",
		Description:       "Coffee purchase",
		RemittanceInfo:    "Payment ref 123",
		Amount:            decimal.NewFromFloat(-15.50),
		CreditDebit:       "DBIT",
		DebitFlag:         true,
		Debit:             decimal.NewFromFloat(-15.50),
		Credit:            decimal.Zero,
		Currency:          "CHF",
		AmountExclTax:     decimal.NewFromFloat(-14.00),
		AmountTax:         decimal.NewFromFloat(-1.50),
		TaxRate:           decimal.NewFromFloat(7.7),
		Recipient:         "Coffee Shop",
		Investment:        "",
		Number:            "001",
		Category:          "Food & Dining",
		Type:              "Card Payment",
		Fund:              "",
		NumberOfShares:    0,
		Fees:              decimal.Zero,
		IBAN:              "CH9876543210",
		EntryReference:    "REF001",
		Reference:         "R001",
		AccountServicer:   "BCHEXXXX",
		BankTxCode:        "PMNT",
		OriginalCurrency:  "",
		OriginalAmount:    decimal.Zero,
		ExchangeRate:      decimal.Zero,
		Payee:             "Coffee Shop", // Set Payee for debit transaction
		Payer:             "",
	}
	return tx
}

func TestFormatterRegistry(t *testing.T) {
	registry := NewFormatterRegistry()

	t.Run("Get standard formatter", func(t *testing.T) {
		formatter, err := registry.Get("standard")
		require.NoError(t, err)
		assert.NotNil(t, formatter)
		assert.IsType(t, &StandardFormatter{}, formatter)
	})

	t.Run("Get icompta formatter", func(t *testing.T) {
		formatter, err := registry.Get("icompta")
		require.NoError(t, err)
		assert.NotNil(t, formatter)
		assert.IsType(t, &iComptaFormatter{}, formatter)
	})

	t.Run("Get non-existent formatter", func(t *testing.T) {
		formatter, err := registry.Get("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, formatter)
		assert.Contains(t, err.Error(), "formatter not found")
	})

	t.Run("Register custom formatter", func(t *testing.T) {
		customFormatter := NewStandardFormatter() // Use standard as custom for testing
		registry.Register("custom", customFormatter)

		formatter, err := registry.Get("custom")
		require.NoError(t, err)
		assert.NotNil(t, formatter)
	})

	t.Run("ListAvailable formatters", func(t *testing.T) {
		names := registry.ListAvailable()
		assert.Contains(t, names, "standard")
		assert.Contains(t, names, "icompta")
		assert.GreaterOrEqual(t, len(names), 2)
	})
}

func TestStandardFormatter(t *testing.T) {
	formatter := NewStandardFormatter()

	t.Run("Header returns 29 columns", func(t *testing.T) {
		header := formatter.Header()
		assert.Len(t, header, 29)
		assert.Equal(t, "Status", header[0])
		assert.Equal(t, "Date", header[1])
		assert.Equal(t, "ValueDate", header[2])
		assert.Equal(t, "ExchangeRate", header[28]) // Last column
	})

	t.Run("Delimiter is comma", func(t *testing.T) {
		assert.Equal(t, ',', formatter.Delimiter())
	})

	t.Run("Format single transaction", func(t *testing.T) {
		tx := createTestTransaction()
		rows, err := formatter.Format([]models.Transaction{tx})
		require.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Len(t, rows[0], 29) // 29 columns (removed 6 redundant fields)

		// Verify key fields
		assert.Equal(t, "BOOK", rows[0][0])         // Status
		assert.Equal(t, "15.02.2026", rows[0][1])   // Date
		assert.Equal(t, "Coffee Shop", rows[0][3])  // Name
		assert.Equal(t, "-15.50", rows[0][8])       // Amount
		assert.Equal(t, "CHF", rows[0][10])         // Currency
		assert.Equal(t, "", rows[0][11])            // Product (empty for this test)
		assert.Equal(t, "Food & Dining", rows[0][16]) // Category
	})

	t.Run("Format multiple transactions", func(t *testing.T) {
		tx1 := createTestTransaction()
		tx2 := createTestTransaction()
		tx2.Payee = "Gas Station"  // Payee is used to set Name for debit transactions
		tx2.Name = "Gas Station"

		rows, err := formatter.Format([]models.Transaction{tx1, tx2})
		require.NoError(t, err)
		assert.Len(t, rows, 2)
		assert.Equal(t, "Coffee Shop", rows[0][3])  // Name is now at index 3
		assert.Equal(t, "Gas Station", rows[1][3])
	})

	t.Run("Format empty transactions", func(t *testing.T) {
		rows, err := formatter.Format([]models.Transaction{})
		require.NoError(t, err)
		assert.Len(t, rows, 0)
	})
}

func TestIComptaFormatter(t *testing.T) {
	formatter := NewIComptaFormatter()

	t.Run("Header returns 10 columns", func(t *testing.T) {
		header := formatter.Header()
		assert.Len(t, header, 10)
		assert.Equal(t, "Date", header[0])
		assert.Equal(t, "Name", header[1])
		assert.Equal(t, "Amount", header[2])
		assert.Equal(t, "Type", header[9]) // Last column
	})

	t.Run("Delimiter is semicolon", func(t *testing.T) {
		assert.Equal(t, ';', formatter.Delimiter())
	})

	t.Run("Format single transaction", func(t *testing.T) {
		tx := createTestTransaction()
		rows, err := formatter.Format([]models.Transaction{tx})
		require.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Len(t, rows[0], 10) // 10 columns

		// Verify key fields
		assert.Equal(t, "15.02.2026", rows[0][0])     // Date in dd.MM.yyyy
		assert.Equal(t, "Coffee Shop", rows[0][1])    // Name
		assert.Equal(t, "-15.50", rows[0][2])         // Amount
		assert.Equal(t, "Coffee purchase", rows[0][3]) // Description
		assert.Equal(t, "cleared", rows[0][4])        // Status (BOOK→cleared)
		assert.Equal(t, "Food & Dining", rows[0][5])  // Category
		assert.Equal(t, "-15.50", rows[0][6])         // SplitAmount
		assert.Equal(t, "-14.00", rows[0][7])         // SplitAmountExclTax
		assert.Equal(t, "7.70", rows[0][8])           // SplitTaxRate
		assert.Equal(t, "Card Payment", rows[0][9])   // Type
	})

	t.Run("Date format is dd.MM.yyyy", func(t *testing.T) {
		tx := createTestTransaction()
		tx.Date = time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

		rows, err := formatter.Format([]models.Transaction{tx})
		require.NoError(t, err)
		assert.Equal(t, "31.12.2026", rows[0][0])
	})

	t.Run("Zero date returns empty string", func(t *testing.T) {
		tx := createTestTransaction()
		tx.Date = time.Time{} // Zero time

		rows, err := formatter.Format([]models.Transaction{tx})
		require.NoError(t, err)
		assert.Equal(t, "", rows[0][0])
	})

	t.Run("Name falls back to PartyName", func(t *testing.T) {
		tx := createTestTransaction()
		tx.Name = "" // Empty name
		tx.PartyName = "Fallback Party"

		rows, err := formatter.Format([]models.Transaction{tx})
		require.NoError(t, err)
		assert.Equal(t, "Fallback Party", rows[0][1])
	})

	t.Run("Empty category uses Uncategorized", func(t *testing.T) {
		tx := createTestTransaction()
		tx.Category = ""

		rows, err := formatter.Format([]models.Transaction{tx})
		require.NoError(t, err)
		assert.Equal(t, "Uncategorized", rows[0][5])
	})

	t.Run("Status mapping", func(t *testing.T) {
		testCases := []struct {
			camtStatus     string
			expectedStatus string
		}{
			{"BOOK", "cleared"},
			{"RCVD", "cleared"},
			{"PDNG", "pending"},
			{"REVD", "reverted"},
			{"CANC", "reverted"},
			{"UNKNOWN", "cleared"}, // Default
			{"", "cleared"},        // Empty default
		}

		for _, tc := range testCases {
			t.Run(tc.camtStatus, func(t *testing.T) {
				tx := createTestTransaction()
				tx.Status = tc.camtStatus

				rows, err := formatter.Format([]models.Transaction{tx})
				require.NoError(t, err)
				assert.Equal(t, tc.expectedStatus, rows[0][4])
			})
		}
	})

	t.Run("Decimal formatting", func(t *testing.T) {
		tx := createTestTransaction()
		tx.Amount = decimal.NewFromFloat(1234.567) // Should round to 2 decimals
		tx.AmountExclTax = decimal.NewFromFloat(1000.001)
		tx.TaxRate = decimal.NewFromFloat(8.12345)

		rows, err := formatter.Format([]models.Transaction{tx})
		require.NoError(t, err)
		assert.Equal(t, "1234.57", rows[0][2])   // Amount
		assert.Equal(t, "1000.00", rows[0][7])   // AmountExclTax
		assert.Equal(t, "8.12", rows[0][8])      // TaxRate
	})

	t.Run("Format multiple transactions", func(t *testing.T) {
		tx1 := createTestTransaction()
		tx2 := createTestTransaction()
		tx2.Name = "Restaurant"
		tx2.Amount = decimal.NewFromFloat(45.80)

		rows, err := formatter.Format([]models.Transaction{tx1, tx2})
		require.NoError(t, err)
		assert.Len(t, rows, 2)
		assert.Equal(t, "Coffee Shop", rows[0][1])
		assert.Equal(t, "Restaurant", rows[1][1])
	})
}

func TestMapStatusToICompta(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"BOOK", "cleared"},
		{"RCVD", "cleared"},
		{"PDNG", "pending"},
		{"REVD", "reverted"},
		{"CANC", "reverted"},
		{"UNKNOWN", "cleared"},
		{"", "cleared"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapStatusToICompta(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
