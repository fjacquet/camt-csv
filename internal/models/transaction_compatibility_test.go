package models

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransaction_GetPayee(t *testing.T) {
	tests := []struct {
		name     string
		tx       Transaction
		expected string
	}{
		{
			name: "debit transaction returns payee",
			tx: Transaction{
				CreditDebit: TransactionTypeDebit,
				DebitFlag:   true,
				Payer:       "Account Holder",
				Payee:       "Acme Corp",
			},
			expected: "Acme Corp",
		},
		{
			name: "credit transaction returns payer",
			tx: Transaction{
				CreditDebit: TransactionTypeCredit,
				DebitFlag:   false,
				Payer:       "John Doe",
				Payee:       "Account Holder",
			},
			expected: "John Doe",
		},
		{
			name: "unknown direction defaults to debit behavior",
			tx: Transaction{
				CreditDebit: "UNKNOWN",
				DebitFlag:   false,
				Amount:      decimal.Zero,
				Payer:       "Some Payer",
				Payee:       "Acme Corp",
			},
			expected: "Some Payer", // Credit behavior for unknown with zero amount
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.tx.GetPayee())
		})
	}
}

func TestTransaction_GetPayer(t *testing.T) {
	tests := []struct {
		name     string
		tx       Transaction
		expected string
	}{
		{
			name: "debit transaction returns payer",
			tx: Transaction{
				CreditDebit: TransactionTypeDebit,
				DebitFlag:   true,
				Payer:       "Account Holder",
				Payee:       "Acme Corp",
			},
			expected: "Account Holder",
		},
		{
			name: "credit transaction returns payee",
			tx: Transaction{
				CreditDebit: TransactionTypeCredit,
				DebitFlag:   false,
				Payer:       "John Doe",
				Payee:       "Account Holder",
			},
			expected: "Account Holder",
		},
		{
			name: "unknown direction defaults to credit behavior",
			tx: Transaction{
				CreditDebit: "UNKNOWN",
				DebitFlag:   false,
				Amount:      decimal.Zero,
				Payer:       "Some Payer",
				Payee:       "Some Payee",
			},
			expected: "Some Payee", // Credit behavior for unknown with zero amount
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.tx.GetPayer())
		})
	}
}

func TestTransaction_GetCounterparty(t *testing.T) {
	tests := []struct {
		name        string
		transaction Transaction
		expected    string
	}{
		{
			name: "debit transaction returns payee",
			transaction: Transaction{
				CreditDebit: TransactionTypeDebit,
				DebitFlag:   true,
				Payer:       "John Doe",
				Payee:       "Acme Corp",
			},
			expected: "Acme Corp",
		},
		{
			name: "credit transaction returns payer",
			transaction: Transaction{
				CreditDebit: TransactionTypeCredit,
				DebitFlag:   false,
				Payer:       "John Doe",
				Payee:       "Acme Corp",
			},
			expected: "John Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.transaction.GetCounterparty()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransaction_ToTransactionCore(t *testing.T) {
	tx := Transaction{
		Number:      "TX123",
		Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		ValueDate:   time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
		Amount:      decimal.NewFromFloat(100.50),
		Currency:    "CHF",
		Description: "Test transaction",
		Status:      StatusCompleted,
		Reference:   "REF123",
	}

	core := tx.ToTransactionCore()

	assert.Equal(t, "TX123", core.ID)
	assert.Equal(t, 2025, core.Date.Year())
	assert.Equal(t, time.January, core.Date.Month())
	assert.Equal(t, 15, core.Date.Day())
	assert.Equal(t, 2025, core.ValueDate.Year())
	assert.Equal(t, time.January, core.ValueDate.Month())
	assert.Equal(t, 16, core.ValueDate.Day())
	assert.True(t, core.Amount.Amount.Equal(decimal.NewFromFloat(100.50)))
	assert.Equal(t, "CHF", core.Amount.Currency)
	assert.Equal(t, "Test transaction", core.Description)
	assert.Equal(t, StatusCompleted, core.Status)
	assert.Equal(t, "REF123", core.Reference)
}

func TestTransaction_ToTransactionCore_InvalidDates(t *testing.T) {
	tx := Transaction{
		Number:      "TX123",
		Date:        time.Time{}, // Zero time for invalid date test
		ValueDate:   time.Time{}, // Zero time for invalid date test
		Amount:      decimal.NewFromFloat(100.50),
		Currency:    "CHF",
		Description: "Test transaction",
	}

	core := tx.ToTransactionCore()

	// Invalid dates should result in zero times
	assert.True(t, core.Date.IsZero())
	assert.True(t, core.ValueDate.IsZero())
	// Other fields should still be populated
	assert.Equal(t, "TX123", core.ID)
	assert.True(t, core.Amount.Amount.Equal(decimal.NewFromFloat(100.50)))
}

func TestTransaction_ToTransactionWithParties(t *testing.T) {
	tx := Transaction{
		Number:      "TX123",
		Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:      decimal.NewFromFloat(100.50),
		Currency:    "CHF",
		CreditDebit: TransactionTypeDebit,
		DebitFlag:   true,
		Payer:       "John Doe",
		Payee:       "Acme Corp",
		PartyIBAN:   "CH1234567890",
	}

	twp := tx.ToTransactionWithParties()

	assert.Equal(t, "TX123", twp.ID)
	assert.Equal(t, DirectionDebit, twp.Direction)
	assert.Equal(t, "John Doe", twp.Payer.Name)
	assert.Equal(t, "Acme Corp", twp.Payee.Name)
	assert.Equal(t, "CH1234567890", twp.Payer.IBAN)
	assert.Equal(t, "CH1234567890", twp.Payee.IBAN)
}

func TestTransaction_ToCategorizedTransaction(t *testing.T) {
	tx := Transaction{
		Number:      "TX123",
		Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:      decimal.NewFromFloat(100.50),
		Currency:    "CHF",
		CreditDebit: TransactionTypeCredit,
		DebitFlag:   false,
		Payer:       "John Doe",
		Payee:       "Acme Corp",
		Category:    "Shopping",
		Type:        "Purchase",
		Fund:        "General",
	}

	ct := tx.ToCategorizedTransaction()

	assert.Equal(t, "TX123", ct.ID)
	assert.Equal(t, DirectionCredit, ct.Direction)
	assert.Equal(t, "Shopping", ct.Category)
	assert.Equal(t, "Purchase", ct.Type)
	assert.Equal(t, "General", ct.Fund)
}

func TestTransaction_FromTransactionCore(t *testing.T) {
	core := TransactionCore{
		ID:          "TX123",
		Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		ValueDate:   time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
		Amount:      NewMoney(decimal.NewFromFloat(100.50), "CHF"),
		Description: "Test transaction",
		Status:      StatusCompleted,
		Reference:   "REF123",
	}

	var tx Transaction
	tx.FromTransactionCore(core)

	assert.Equal(t, "TX123", tx.Number)
	assert.Equal(t, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), tx.Date)
	assert.Equal(t, time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC), tx.ValueDate)
	assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(100.50)))
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "Test transaction", tx.Description)
	assert.Equal(t, StatusCompleted, tx.Status)
	assert.Equal(t, "REF123", tx.Reference)
}

func TestTransaction_FromTransactionCore_ZeroDates(t *testing.T) {
	core := TransactionCore{
		ID:          "TX123",
		Date:        time.Time{}, // Zero time
		ValueDate:   time.Time{}, // Zero time
		Amount:      NewMoney(decimal.NewFromFloat(100.50), "CHF"),
		Description: "Test transaction",
	}

	var tx Transaction
	tx.FromTransactionCore(core)

	assert.Equal(t, "TX123", tx.Number)
	assert.True(t, tx.Date.IsZero())      // Should be zero time
	assert.True(t, tx.ValueDate.IsZero()) // Should be zero time
	assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(100.50)))
}

func TestTransaction_FromCategorizedTransaction(t *testing.T) {
	ct := CategorizedTransaction{
		TransactionWithParties: TransactionWithParties{
			TransactionCore: TransactionCore{
				ID:          "TX123",
				Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
				Amount:      NewMoney(decimal.NewFromFloat(100.50), "CHF"),
				Description: "Test transaction",
				Status:      StatusCompleted,
			},
			Payer:     NewParty("John Doe", "CH1234567890"),
			Payee:     NewParty("Acme Corp", "CH0987654321"),
			Direction: DirectionDebit,
		},
		Category: "Shopping",
		Type:     "Purchase",
		Fund:     "General",
	}

	var tx Transaction
	tx.FromCategorizedTransaction(ct)

	assert.Equal(t, "TX123", tx.Number)
	assert.Equal(t, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), tx.Date)
	assert.True(t, tx.Amount.Equal(decimal.NewFromFloat(100.50)))
	assert.Equal(t, "CHF", tx.Currency)
	assert.Equal(t, "John Doe", tx.Payer)
	assert.Equal(t, "Acme Corp", tx.Payee)
	assert.Equal(t, "CH0987654321", tx.PartyIBAN) // Should use payee IBAN for debit
	assert.Equal(t, "Acme Corp", tx.PartyName)    // Should use payee name for debit
	assert.Equal(t, TransactionTypeDebit, tx.CreditDebit)
	assert.True(t, tx.DebitFlag)
	assert.Equal(t, "Shopping", tx.Category)
	assert.Equal(t, "Purchase", tx.Type)
	assert.Equal(t, "General", tx.Fund)

	// Check derived fields
	assert.Equal(t, "Acme Corp", tx.Name)      // For debit, Name should be Payee
	assert.Equal(t, "Acme Corp", tx.Recipient) // Recipient should be set from Payee
	assert.True(t, tx.Debit.Equal(decimal.NewFromFloat(100.50)))
	assert.True(t, tx.Credit.IsZero())
}

func TestTransaction_ConversionRoundTrip(t *testing.T) {
	// Test that converting from Transaction to new format and back preserves data
	original := Transaction{
		Number:      "TX123",
		Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		ValueDate:   time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
		Amount:      decimal.NewFromFloat(100.50),
		Currency:    "CHF",
		Description: "Test transaction",
		Status:      StatusCompleted,
		Reference:   "REF123",
		CreditDebit: TransactionTypeCredit,
		DebitFlag:   false,
		Payer:       "John Doe",
		Payee:       "Acme Corp",
		PartyIBAN:   "CH1234567890",
		Category:    "Shopping",
		Type:        "Purchase",
		Fund:        "General",
	}

	// Convert to new format and back
	ct := original.ToCategorizedTransaction()
	var converted Transaction
	converted.FromCategorizedTransaction(ct)

	// Check key fields are preserved
	assert.Equal(t, original.Number, converted.Number)
	assert.Equal(t, original.Date, converted.Date)
	assert.Equal(t, original.ValueDate, converted.ValueDate)
	assert.True(t, original.Amount.Equal(converted.Amount))
	assert.Equal(t, original.Currency, converted.Currency)
	assert.Equal(t, original.Description, converted.Description)
	assert.Equal(t, original.Status, converted.Status)
	assert.Equal(t, original.Reference, converted.Reference)
	assert.Equal(t, original.CreditDebit, converted.CreditDebit)
	assert.Equal(t, original.DebitFlag, converted.DebitFlag)
	assert.Equal(t, original.Payer, converted.Payer)
	assert.Equal(t, original.Payee, converted.Payee)
	assert.Equal(t, original.Category, converted.Category)
	assert.Equal(t, original.Type, converted.Type)
	assert.Equal(t, original.Fund, converted.Fund)
}

func TestTransaction_BuilderCompatibility(t *testing.T) {
	// Test that TransactionBuilder can create transactions compatible with legacy methods
	tx, err := NewTransactionBuilder().
		WithDate("2025-01-15").
		WithAmountFromFloat(100.50, "CHF").
		WithDescription("Test transaction").
		WithPayer("John Doe", "CH1234567890").
		WithPayee("Acme Corp", "CH0987654321").
		WithCategory("Shopping").
		AsDebit().
		Build()

	require.NoError(t, err)

	// Test legacy methods work with direction-based behavior
	assert.Equal(t, 100.50, tx.GetAmountAsFloat())
	// For debit transaction: GetPayer() returns the payer (account holder)
	assert.Equal(t, "John Doe", tx.GetPayer())
	// For debit transaction: GetPayee() returns the payee (who receives money)
	assert.Equal(t, "Acme Corp", tx.GetPayee())
	assert.Equal(t, "Acme Corp", tx.GetCounterparty()) // For debit, counterparty is payee

	// Test conversion to new format
	ct := tx.ToCategorizedTransaction()
	assert.Equal(t, DirectionDebit, ct.Direction)
	assert.Equal(t, "John Doe", ct.Payer.Name)
	assert.Equal(t, "Acme Corp", ct.Payee.Name)
	assert.Equal(t, "Shopping", ct.Category)
}
