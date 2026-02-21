package dateutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanDateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Already clean", "2023-01-15", "2023-01-15"},
		{"With spaces", "  2023-01-15  ", "2023-01-15"},
		{"Multiple spaces", "2023  01  15", "2023 01 15"},
		{"Empty string", "", ""},
		{"Only whitespace", "   ", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CleanDateString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseDateString(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectErr  bool
		expectYear int
	}{
		{"ISO format", "2023-01-15", false, 2023},
		{"European format", "15.01.2023", false, 2023},
		{"Full timestamp", "2023-01-15 10:30:45", false, 2023},
		{"Empty string", "", false, 0},
		{"Invalid format", "not a date", true, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseDateString(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.expectYear > 0 {
					assert.Equal(t, tc.expectYear, result.Year())
				}
			}
		})
	}
}
