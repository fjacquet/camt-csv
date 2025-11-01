package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionDirectionString(t *testing.T) {
	tests := []struct {
		name      string
		direction TransactionDirection
		expected  string
	}{
		{
			name:      "Debit",
			direction: DirectionDebit,
			expected:  "DEBIT",
		},
		{
			name:      "Credit",
			direction: DirectionCredit,
			expected:  "CREDIT",
		},
		{
			name:      "Unknown",
			direction: DirectionUnknown,
			expected:  "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.direction.String())
		})
	}
}

func TestTransactionDirectionFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TransactionDirection
	}{
		{
			name:     "DEBIT",
			input:    "DEBIT",
			expected: DirectionDebit,
		},
		{
			name:     "DBIT",
			input:    "DBIT",
			expected: DirectionDebit,
		},
		{
			name:     "CREDIT",
			input:    "CREDIT",
			expected: DirectionCredit,
		},
		{
			name:     "CRDT",
			input:    "CRDT",
			expected: DirectionCredit,
		},
		{
			name:     "Unknown",
			input:    "UNKNOWN",
			expected: DirectionUnknown,
		},
		{
			name:     "Invalid",
			input:    "INVALID",
			expected: DirectionUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, TransactionDirectionFromString(tt.input))
		})
	}
}

func TestTransactionDirectionMethods(t *testing.T) {
	assert.True(t, DirectionDebit.IsDebit())
	assert.False(t, DirectionDebit.IsCredit())
	
	assert.True(t, DirectionCredit.IsCredit())
	assert.False(t, DirectionCredit.IsDebit())
	
	assert.False(t, DirectionUnknown.IsDebit())
	assert.False(t, DirectionUnknown.IsCredit())
}