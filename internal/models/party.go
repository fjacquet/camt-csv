package models

import (
	"fmt"
	"regexp"
	"strings"
)

// Party represents a transaction party (payer or payee)
type Party struct {
	Name string `json:"name" yaml:"name"`
	IBAN string `json:"iban" yaml:"iban"`
}

// NewParty creates a new Party instance
func NewParty(name, iban string) Party {
	return Party{
		Name: strings.TrimSpace(name),
		IBAN: strings.TrimSpace(iban),
	}
}

// IsEmpty returns true if both name and IBAN are empty
func (p Party) IsEmpty() bool {
	return p.Name == "" && p.IBAN == ""
}

// HasName returns true if the party has a non-empty name
func (p Party) HasName() bool {
	return strings.TrimSpace(p.Name) != ""
}

// HasIBAN returns true if the party has a non-empty IBAN
func (p Party) HasIBAN() bool {
	return strings.TrimSpace(p.IBAN) != ""
}

// String returns a string representation of the party
func (p Party) String() string {
	if p.Name != "" && p.IBAN != "" {
		return fmt.Sprintf("%s (%s)", p.Name, p.IBAN)
	}
	if p.Name != "" {
		return p.Name
	}
	if p.IBAN != "" {
		return p.IBAN
	}
	return ""
}

// ValidateIBAN performs basic IBAN format validation
// Returns true if the IBAN appears to be in a valid format
func (p Party) ValidateIBAN() bool {
	if p.IBAN == "" {
		return true // Empty IBAN is considered valid (optional field)
	}

	// Remove spaces and convert to uppercase
	iban := strings.ToUpper(strings.ReplaceAll(p.IBAN, " ", ""))

	// Basic IBAN format check: 2 letters followed by 2 digits, then alphanumeric characters
	// Length should be between 15 and 34 characters
	ibanRegex := regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$`)
	return ibanRegex.MatchString(iban)
}

// NormalizedIBAN returns the IBAN in normalized format (uppercase, no spaces)
func (p Party) NormalizedIBAN() string {
	if p.IBAN == "" {
		return ""
	}
	return strings.ToUpper(strings.ReplaceAll(p.IBAN, " ", ""))
}

// FormattedIBAN returns the IBAN in a human-readable format with spaces every 4 characters
func (p Party) FormattedIBAN() string {
	normalized := p.NormalizedIBAN()
	if normalized == "" {
		return ""
	}

	// Add spaces every 4 characters
	var formatted strings.Builder
	for i, char := range normalized {
		if i > 0 && i%4 == 0 {
			formatted.WriteRune(' ')
		}
		formatted.WriteRune(char)
	}

	return formatted.String()
}

// Equal returns true if two parties are equal (same name and IBAN)
func (p Party) Equal(other Party) bool {
	return strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(other.Name)) &&
		strings.EqualFold(p.NormalizedIBAN(), other.NormalizedIBAN())
}

// SimilarTo returns true if two parties are similar (same name, ignoring case and whitespace)
// This is useful for matching parties when IBAN might not be available
func (p Party) SimilarTo(other Party) bool {
	return strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(other.Name))
}
