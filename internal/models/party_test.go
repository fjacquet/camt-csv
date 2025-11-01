package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewParty(t *testing.T) {
	party := NewParty("John Doe", "CH1234567890123456")
	
	assert.Equal(t, "John Doe", party.Name)
	assert.Equal(t, "CH1234567890123456", party.IBAN)
}

func TestPartyIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		party    Party
		expected bool
	}{
		{
			name:     "EmptyParty",
			party:    Party{},
			expected: true,
		},
		{
			name:     "PartyWithName",
			party:    Party{Name: "John Doe"},
			expected: false,
		},
		{
			name:     "PartyWithIBAN",
			party:    Party{IBAN: "CH1234567890123456"},
			expected: false,
		},
		{
			name:     "PartyWithBoth",
			party:    Party{Name: "John Doe", IBAN: "CH1234567890123456"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.party.IsEmpty())
		})
	}
}

func TestPartyValidateIBAN(t *testing.T) {
	tests := []struct {
		name     string
		iban     string
		expected bool
	}{
		{
			name:     "ValidSwissIBAN",
			iban:     "CH1234567890123456",
			expected: true,
		},
		{
			name:     "ValidGermanIBAN",
			iban:     "DE89370400440532013000",
			expected: true,
		},
		{
			name:     "EmptyIBAN",
			iban:     "",
			expected: true, // Empty is considered valid
		},
		{
			name:     "InvalidIBAN",
			iban:     "INVALID",
			expected: false,
		},
		{
			name:     "TooShortIBAN",
			iban:     "CH123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			party := Party{IBAN: tt.iban}
			assert.Equal(t, tt.expected, party.ValidateIBAN())
		})
	}
}

func TestPartyString(t *testing.T) {
	tests := []struct {
		name     string
		party    Party
		expected string
	}{
		{
			name:     "NameAndIBAN",
			party:    Party{Name: "John Doe", IBAN: "CH1234567890123456"},
			expected: "John Doe (CH1234567890123456)",
		},
		{
			name:     "NameOnly",
			party:    Party{Name: "John Doe"},
			expected: "John Doe",
		},
		{
			name:     "IBANOnly",
			party:    Party{IBAN: "CH1234567890123456"},
			expected: "CH1234567890123456",
		},
		{
			name:     "Empty",
			party:    Party{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.party.String())
		})
	}
}