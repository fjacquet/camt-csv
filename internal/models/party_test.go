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

// Test uncovered Party methods
func TestParty_UncoveredMethods(t *testing.T) {
	t.Run("HasName", func(t *testing.T) {
		partyWithName := Party{Name: "John Doe"}
		partyWithoutName := Party{IBAN: "CH1234567890123456"}
		emptyParty := Party{}

		assert.True(t, partyWithName.HasName())
		assert.False(t, partyWithoutName.HasName())
		assert.False(t, emptyParty.HasName())
	})

	t.Run("HasIBAN", func(t *testing.T) {
		partyWithIBAN := Party{IBAN: "CH1234567890123456"}
		partyWithoutIBAN := Party{Name: "John Doe"}
		emptyParty := Party{}

		assert.True(t, partyWithIBAN.HasIBAN())
		assert.False(t, partyWithoutIBAN.HasIBAN())
		assert.False(t, emptyParty.HasIBAN())
	})

	t.Run("NormalizedIBAN", func(t *testing.T) {
		tests := []struct {
			name     string
			iban     string
			expected string
		}{
			{
				name:     "WithSpaces",
				iban:     "CH12 3456 7890 1234 56",
				expected: "CH1234567890123456",
			},
			{
				name:     "WithoutSpaces",
				iban:     "CH1234567890123456",
				expected: "CH1234567890123456",
			},
			{
				name:     "LowerCase",
				iban:     "ch1234567890123456",
				expected: "CH1234567890123456",
			},
			{
				name:     "MixedCase",
				iban:     "Ch12 3456 7890 1234 56",
				expected: "CH1234567890123456",
			},
			{
				name:     "Empty",
				iban:     "",
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				party := Party{IBAN: tt.iban}
				result := party.NormalizedIBAN()
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("FormattedIBAN", func(t *testing.T) {
		tests := []struct {
			name     string
			iban     string
			expected string
		}{
			{
				name:     "SwissIBAN",
				iban:     "CH1234567890123456",
				expected: "CH12 3456 7890 1234 56",
			},
			{
				name:     "GermanIBAN",
				iban:     "DE89370400440532013000",
				expected: "DE89 3704 0044 0532 0130 00",
			},
			{
				name:     "AlreadyFormatted",
				iban:     "CH12 3456 7890 1234 56",
				expected: "CH12 3456 7890 1234 56",
			},
			{
				name:     "Empty",
				iban:     "",
				expected: "",
			},
			{
				name:     "TooShort",
				iban:     "CH123",
				expected: "CH12 3", // Actually formats even short IBANs
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				party := Party{IBAN: tt.iban}
				result := party.FormattedIBAN()
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Equal", func(t *testing.T) {
		party1 := Party{Name: "John Doe", IBAN: "CH1234567890123456"}
		party2 := Party{Name: "John Doe", IBAN: "CH1234567890123456"}
		party3 := Party{Name: "Jane Doe", IBAN: "CH1234567890123456"}
		party4 := Party{Name: "John Doe", IBAN: "DE89370400440532013000"}

		assert.True(t, party1.Equal(party2))
		assert.False(t, party1.Equal(party3))
		assert.False(t, party1.Equal(party4))

		// Test with normalized IBANs
		partySpaced := Party{Name: "John Doe", IBAN: "CH12 3456 7890 1234 56"}
		assert.True(t, party1.Equal(partySpaced))
	})

	t.Run("SimilarTo", func(t *testing.T) {
		party1 := Party{Name: "John Doe", IBAN: "CH1234567890123456"}
		party2 := Party{Name: "john doe", IBAN: "CH1234567890123456"}     // Different case
		party3 := Party{Name: "John", IBAN: "CH1234567890123456"}         // Different name
		party4 := Party{Name: "Jane Doe", IBAN: "CH1234567890123456"}     // Different name
		party5 := Party{Name: "John Doe", IBAN: "DE89370400440532013000"} // Same name, different IBAN

		assert.True(t, party1.SimilarTo(party2))  // Case insensitive
		assert.False(t, party1.SimilarTo(party3)) // Different name
		assert.False(t, party1.SimilarTo(party4)) // Different name
		assert.True(t, party1.SimilarTo(party5))  // Same name, IBAN doesn't matter for SimilarTo

		// Test with whitespace
		partyWithSpaces := Party{Name: " John Doe ", IBAN: "CH1234567890123456"}
		assert.True(t, party1.SimilarTo(partyWithSpaces)) // Whitespace trimmed
	})
}
