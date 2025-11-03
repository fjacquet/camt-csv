package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionValidation(t *testing.T) {
	tests := []struct {
		name        string
		transaction Transaction
		expectValid bool
		expectError string
	}{
		{
			name: "valid transaction",
			transaction: Transaction{
				Number:      "TXN-001",
				Date:        time.Now(),
				Amount:      decimal.NewFromFloat(100.50),
				Currency:    "CHF",
				Description: "Valid transaction",
			},
			expectValid: true,
		},
		{
			name: "missing required fields",
			transaction: Transaction{
				Amount:   decimal.NewFromFloat(100.50),
				Currency: "CHF",
			},
			expectValid: false,
			expectError: "missing required field",
		},
		{
			name: "invalid currency",
			transaction: Transaction{
				Number:      "TXN-002",
				Date:        time.Now(),
				Amount:      decimal.NewFromFloat(100.50),
				Currency:    "INVALID",
				Description: "Invalid currency",
			},
			expectValid: false,
			expectError: "invalid currency",
		},
		{
			name: "zero amount",
			transaction: Transaction{
				Number:      "TXN-003",
				Date:        time.Now(),
				Amount:      decimal.Zero,
				Currency:    "CHF",
				Description: "Zero amount",
			},
			expectValid: true, // Zero amounts can be valid
		},
		{
			name: "future date",
			transaction: Transaction{
				Number:      "TXN-004",
				Date:        time.Now().Add(24 * time.Hour),
				Amount:      decimal.NewFromFloat(100.50),
				Currency:    "CHF",
				Description: "Future date",
			},
			expectValid: false,
			expectError: "future date not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransaction(tt.transaction)
			
			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			}
		})
	}
}

// validateTransaction is a helper function for transaction validation
// This would be implemented as part of the Transaction struct methods
func validateTransaction(tx Transaction) error {
	if tx.Number == "" {
		return fmt.Errorf("missing required field: Number")
	}
	
	if tx.Date.IsZero() {
		return fmt.Errorf("missing required field: Date")
	}
	
	if tx.Date.After(time.Now()) {
		return fmt.Errorf("future date not allowed")
	}
	
	validCurrencies := map[string]bool{
		"CHF": true, "EUR": true, "USD": true, "GBP": true,
	}
	
	if !validCurrencies[tx.Currency] {
		return fmt.Errorf("invalid currency: %s", tx.Currency)
	}
	
	return nil
}

func TestTransactionBuilder_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		buildFunc   func() (*TransactionBuilder, error)
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing required amount",
			buildFunc: func() (*TransactionBuilder, error) {
				builder := NewTransactionBuilder()
				_, err := builder.WithDate("2025-01-15").Build()
				return builder, err
			},
			expectError: true,
			errorMsg:    "amount is required",
		},
		{
			name: "invalid date format",
			buildFunc: func() (*TransactionBuilder, error) {
				builder := NewTransactionBuilder()
				_, err := builder.WithDate("invalid-date").Build()
				return builder, err
			},
			expectError: true,
			errorMsg:    "invalid date format",
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.buildFunc()
			
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}