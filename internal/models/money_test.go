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