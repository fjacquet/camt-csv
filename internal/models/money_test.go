package models

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewMoney(t *testing.T) {
	amount := decimal.NewFromFloat(100.50)
	money := NewMoney(amount, "CHF")

	assert.Equal(t, amount, money.Amount)
	assert.Equal(t, "CHF", money.Currency)
}

func TestNewMoneyFromString(t *testing.T) {
	tests := []struct {
		name           string
		amount         string
		currency       string
		expectedAmount string
		expectError    bool
	}{
		{
			name:           "ValidAmount",
			amount:         "100.50",
			currency:       "CHF",
			expectedAmount: "100.50",
			expectError:    false,
		},
		{
			name:           "InvalidAmount",
			amount:         "invalid",
			currency:       "CHF",
			expectedAmount: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := NewMoneyFromString(tt.amount, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAmount, money.Amount.StringFixed(2))
				assert.Equal(t, tt.currency, money.Currency)
			}
		})
	}
}

func TestMoneyOperations(t *testing.T) {
	money1 := NewMoney(decimal.NewFromFloat(100.50), "CHF")
	money2 := NewMoney(decimal.NewFromFloat(50.25), "CHF")

	// Test Add
	result, err := money1.Add(money2)
	assert.NoError(t, err)
	assert.Equal(t, "150.75", result.Amount.StringFixed(2))

	// Test Sub
	result, err = money1.Sub(money2)
	assert.NoError(t, err)
	assert.Equal(t, "50.25", result.Amount.StringFixed(2))

	// Test different currencies
	money3 := NewMoney(decimal.NewFromFloat(100), "EUR")
	_, err = money1.Add(money3)
	assert.Error(t, err)
}

func TestMoneyComparison(t *testing.T) {
	money1 := NewMoney(decimal.NewFromFloat(100.50), "CHF")
	money2 := NewMoney(decimal.NewFromFloat(100.50), "CHF")
	money3 := NewMoney(decimal.NewFromFloat(50.25), "CHF")

	assert.True(t, money1.Equal(money2))
	assert.False(t, money1.Equal(money3))

	cmp, err := money1.Compare(money3)
	assert.NoError(t, err)
	assert.Equal(t, 1, cmp) // money1 > money3
}

func TestMoneyString(t *testing.T) {
	money := NewMoney(decimal.NewFromFloat(100.50), "CHF")
	assert.Equal(t, "100.50 CHF", money.String())
}

// Test financial calculation accuracy and edge cases
func TestMoney_FinancialAccuracy(t *testing.T) {
	t.Run("precision preservation in calculations", func(t *testing.T) {
		// Test that decimal precision is maintained through complex calculations
		amount1, _ := decimal.NewFromString("123.456789")
		amount2, _ := decimal.NewFromString("0.000001")
		money1 := NewMoney(amount1, "CHF")
		money2 := NewMoney(amount2, "CHF")

		result, err := money1.Add(money2)
		assert.NoError(t, err)

		expected, _ := decimal.NewFromString("123.456790")
		assert.True(t, expected.Equal(result.Amount))
	})

	t.Run("large number calculations", func(t *testing.T) {
		amount1, _ := decimal.NewFromString("999999999999.99")
		amount2, _ := decimal.NewFromString("0.01")
		large1 := NewMoney(amount1, "CHF")
		large2 := NewMoney(amount2, "CHF")

		result, err := large1.Add(large2)
		assert.NoError(t, err)

		expected, _ := decimal.NewFromString("1000000000000.00")
		assert.True(t, expected.Equal(result.Amount))
	})

	t.Run("small number calculations", func(t *testing.T) {
		amount1, _ := decimal.NewFromString("0.000000001")
		amount2, _ := decimal.NewFromString("0.000000002")
		small1 := NewMoney(amount1, "CHF")
		small2 := NewMoney(amount2, "CHF")

		result, err := small1.Add(small2)
		assert.NoError(t, err)

		expected, _ := decimal.NewFromString("0.000000003")
		assert.True(t, expected.Equal(result.Amount))
	})

	t.Run("negative number handling", func(t *testing.T) {
		amount1, _ := decimal.NewFromString("100.50")
		amount2, _ := decimal.NewFromString("-50.25")
		positive := NewMoney(amount1, "CHF")
		negative := NewMoney(amount2, "CHF")

		result, err := positive.Add(negative)
		assert.NoError(t, err)

		expected, _ := decimal.NewFromString("50.25")
		assert.True(t, expected.Equal(result.Amount))
	})

	t.Run("zero handling", func(t *testing.T) {
		amount, _ := decimal.NewFromString("100.50")
		money := NewMoney(amount, "CHF")
		zero := NewMoney(decimal.Zero, "CHF")

		result, err := money.Add(zero)
		assert.NoError(t, err)
		assert.True(t, money.Amount.Equal(result.Amount))

		result, err = money.Sub(zero)
		assert.NoError(t, err)
		assert.True(t, money.Amount.Equal(result.Amount))
	})
}

// Test currency validation and error cases
func TestMoney_CurrencyValidation(t *testing.T) {
	t.Run("different currency operations", func(t *testing.T) {
		chf := NewMoney(decimal.NewFromFloat(100), "CHF")
		eur := NewMoney(decimal.NewFromFloat(100), "EUR")

		_, err := chf.Add(eur)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "different currencies")

		_, err = chf.Sub(eur)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "different currencies")

		_, err = chf.Compare(eur)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "different currencies")
	})

	t.Run("case sensitive currency codes", func(t *testing.T) {
		chf1 := NewMoney(decimal.NewFromFloat(100), "CHF")
		chf2 := NewMoney(decimal.NewFromFloat(100), "chf")

		// Should treat as different currencies (case sensitive)
		_, err := chf1.Add(chf2)
		assert.Error(t, err)
	})

	t.Run("empty currency", func(t *testing.T) {
		money1 := NewMoney(decimal.NewFromFloat(100), "")
		money2 := NewMoney(decimal.NewFromFloat(50), "")

		// Empty currencies should be treated as same
		result, err := money1.Add(money2)
		assert.NoError(t, err)
		assert.Equal(t, "150.00", result.Amount.StringFixed(2))
	})
}

// Test Money comparison operations
func TestMoney_Comparison(t *testing.T) {
	tests := []struct {
		name     string
		money1   Money
		money2   Money
		expected int
	}{
		{
			name:     "equal amounts",
			money1:   NewMoney(decimal.NewFromFloat(100.50), "CHF"),
			money2:   NewMoney(decimal.NewFromFloat(100.50), "CHF"),
			expected: 0,
		},
		{
			name:     "first greater",
			money1:   NewMoney(decimal.NewFromFloat(200.00), "CHF"),
			money2:   NewMoney(decimal.NewFromFloat(100.50), "CHF"),
			expected: 1,
		},
		{
			name:     "first smaller",
			money1:   NewMoney(decimal.NewFromFloat(50.25), "CHF"),
			money2:   NewMoney(decimal.NewFromFloat(100.50), "CHF"),
			expected: -1,
		},
		{
			name:     "negative vs positive",
			money1:   NewMoney(decimal.NewFromFloat(-50.00), "CHF"),
			money2:   NewMoney(decimal.NewFromFloat(50.00), "CHF"),
			expected: -1,
		},
		{
			name:     "zero vs positive",
			money1:   NewMoney(decimal.Zero, "CHF"),
			money2:   NewMoney(decimal.NewFromFloat(0.01), "CHF"),
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.money1.Compare(tt.money2)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			// Test Equal method consistency
			if tt.expected == 0 {
				assert.True(t, tt.money1.Equal(tt.money2))
			} else {
				assert.False(t, tt.money1.Equal(tt.money2))
			}
		})
	}
}

// Test Money string formatting
func TestMoney_StringFormatting(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		expected string
	}{
		{
			name:     "standard amount",
			money:    NewMoney(decimal.NewFromFloat(100.50), "CHF"),
			expected: "100.50 CHF",
		},
		{
			name:     "zero amount",
			money:    NewMoney(decimal.Zero, "EUR"),
			expected: "0.00 EUR",
		},
		{
			name:     "negative amount",
			money:    NewMoney(decimal.NewFromFloat(-25.75), "USD"),
			expected: "-25.75 USD",
		},
		{
			name:     "high precision amount",
			money:    NewMoney(func() decimal.Decimal { d, _ := decimal.NewFromString("123.456789"); return d }(), "CHF"),
			expected: "123.46 CHF", // String() method uses StringFixed(2) for display
		},
		{
			name:     "empty currency",
			money:    NewMoney(decimal.NewFromFloat(100), ""),
			expected: "100.00 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.money.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test Money creation from various string formats
func TestMoney_StringParsing(t *testing.T) {
	tests := []struct {
		name        string
		amountStr   string
		currency    string
		expectError bool
		expected    string
	}{
		{
			name:        "standard decimal",
			amountStr:   "100.50",
			currency:    "CHF",
			expectError: false,
			expected:    "100.5", // Decimal.String() doesn't pad trailing zeros
		},
		{
			name:        "integer",
			amountStr:   "100",
			currency:    "CHF",
			expectError: false,
			expected:    "100",
		},
		{
			name:        "negative",
			amountStr:   "-50.25",
			currency:    "CHF",
			expectError: false,
			expected:    "-50.25",
		},
		{
			name:        "zero",
			amountStr:   "0",
			currency:    "CHF",
			expectError: false,
			expected:    "0",
		},
		{
			name:        "high precision",
			amountStr:   "123.456789012345",
			currency:    "CHF",
			expectError: false,
			expected:    "123.456789012345",
		},
		{
			name:        "invalid format",
			amountStr:   "not-a-number",
			currency:    "CHF",
			expectError: true,
		},
		{
			name:        "empty string",
			amountStr:   "",
			currency:    "CHF",
			expectError: true,
		},
		{
			name:        "multiple decimals",
			amountStr:   "100.50.25",
			currency:    "CHF",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := NewMoneyFromString(tt.amountStr, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, money.Amount.String())
				assert.Equal(t, tt.currency, money.Currency)
			}
		})
	}
}

// Benchmark Money operations for performance
func BenchmarkMoney_Add(b *testing.B) {
	money1 := NewMoney(decimal.NewFromFloat(100.50), "CHF")
	money2 := NewMoney(decimal.NewFromFloat(50.25), "CHF")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := money1.Add(money2)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMoney_Compare(b *testing.B) {
	money1 := NewMoney(decimal.NewFromFloat(100.50), "CHF")
	money2 := NewMoney(decimal.NewFromFloat(50.25), "CHF")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := money1.Compare(money2)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMoney_String(b *testing.B) {
	money := NewMoney(decimal.NewFromFloat(100.50), "CHF")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = money.String()
	}
}

// Test uncovered Money methods
func TestMoney_UncoveredMethods(t *testing.T) {
	t.Run("NewMoneyFromFloat", func(t *testing.T) {
		money := NewMoneyFromFloat(100.50, "CHF")
		assert.Equal(t, "100.5", money.Amount.String())
		assert.Equal(t, "CHF", money.Currency)
	})

	t.Run("ZeroMoney", func(t *testing.T) {
		zero := ZeroMoney("EUR")
		assert.True(t, zero.Amount.IsZero())
		assert.Equal(t, "EUR", zero.Currency)
	})

	t.Run("IsZero", func(t *testing.T) {
		zero := NewMoney(decimal.Zero, "CHF")
		nonZero := NewMoney(decimal.NewFromFloat(1.0), "CHF")

		assert.True(t, zero.IsZero())
		assert.False(t, nonZero.IsZero())
	})

	t.Run("IsPositive", func(t *testing.T) {
		positive := NewMoney(decimal.NewFromFloat(100.0), "CHF")
		negative := NewMoney(decimal.NewFromFloat(-100.0), "CHF")
		zero := NewMoney(decimal.Zero, "CHF")

		assert.True(t, positive.IsPositive())
		assert.False(t, negative.IsPositive())
		assert.False(t, zero.IsPositive())
	})

	t.Run("IsNegative", func(t *testing.T) {
		positive := NewMoney(decimal.NewFromFloat(100.0), "CHF")
		negative := NewMoney(decimal.NewFromFloat(-100.0), "CHF")
		zero := NewMoney(decimal.Zero, "CHF")

		assert.False(t, positive.IsNegative())
		assert.True(t, negative.IsNegative())
		assert.False(t, zero.IsNegative())
	})

	t.Run("Abs", func(t *testing.T) {
		positive := NewMoney(decimal.NewFromFloat(100.0), "CHF")
		negative := NewMoney(decimal.NewFromFloat(-100.0), "CHF")

		absPositive := positive.Abs()
		absNegative := negative.Abs()

		assert.Equal(t, "100", absPositive.Amount.String())
		assert.Equal(t, "100", absNegative.Amount.String())
		assert.Equal(t, "CHF", absPositive.Currency)
		assert.Equal(t, "CHF", absNegative.Currency)
	})

	t.Run("Neg", func(t *testing.T) {
		positive := NewMoney(decimal.NewFromFloat(100.0), "CHF")
		negative := NewMoney(decimal.NewFromFloat(-100.0), "CHF")

		negPositive := positive.Neg()
		negNegative := negative.Neg()

		assert.Equal(t, "-100", negPositive.Amount.String())
		assert.Equal(t, "100", negNegative.Amount.String())
		assert.Equal(t, "CHF", negPositive.Currency)
		assert.Equal(t, "CHF", negNegative.Currency)
	})

	t.Run("Mul", func(t *testing.T) {
		money := NewMoney(decimal.NewFromFloat(100.0), "CHF")
		multiplier := decimal.NewFromFloat(2.5)

		result := money.Mul(multiplier)

		assert.Equal(t, "250", result.Amount.String())
		assert.Equal(t, "CHF", result.Currency)
	})

	t.Run("Div", func(t *testing.T) {
		money := NewMoney(decimal.NewFromFloat(100.0), "CHF")
		divisor := decimal.NewFromFloat(4.0)

		result := money.Div(divisor)

		assert.Equal(t, "25", result.Amount.String())
		assert.Equal(t, "CHF", result.Currency)
	})

	t.Run("StringFixed", func(t *testing.T) {
		money := NewMoney(decimal.NewFromFloat(100.123456), "CHF")

		// StringFixed returns the amount with currency, not just the amount
		assert.Equal(t, "100.12 CHF", money.StringFixed(2))
		assert.Equal(t, "100.1235 CHF", money.StringFixed(4))
		assert.Equal(t, "100 CHF", money.StringFixed(0))
	})

	t.Run("Float64", func(t *testing.T) {
		money := NewMoney(decimal.NewFromFloat(100.50), "CHF")

		result := money.Float64()

		assert.InDelta(t, 100.50, result, 0.001)
	})
}
