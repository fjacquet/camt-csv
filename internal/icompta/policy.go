package icompta

import (
	"strings"

	"golang.org/x/text/unicode/norm"

	"fjacquet/camt-csv/internal/models"
)

// unknownCategories are the categories that mean "nobody decided". A split in
// one of these may be replaced by any tier.
var unknownCategories = map[string]bool{
	"Divers":             true,
	"Non Classé":         true,
	"Autre":              true,
	"Uncategorized":      true,
	"Uncategorized (AI)": true,
}

// normalize makes category names comparable: macOS can return decomposed
// accents, so compare NFC.
func normalize(s string) string {
	return norm.NFC.String(strings.TrimSpace(s))
}

// IsUnknownCategory reports whether a category name carries no decision.
func IsUnknownCategory(name string) bool {
	n := normalize(name)
	return n == "" || unknownCategories[n]
}

// SameCategory compares two category names after normalisation.
func SameCategory(a, b string) bool {
	return normalize(a) == normalize(b)
}

// Exclusion returns why a split is out of scope, or "" when it is in scope.
// Investment splits and linked transfers are legitimately uncategorised.
func Exclusion(c Candidate) string {
	switch {
	case c.IsInvestment:
		return "investment"
	case c.IsLinked:
		return "linked transfer"
	}
	return ""
}

// usable reports whether the categorizer actually suggested something.
func (p Proposal) usable() bool {
	switch normalize(p.Category) {
	case "", models.CategoryUncategorized, "Uncategorized (AI)":
		return false
	}
	return true
}

// Decide applies the overwrite rule: an unknown or empty category takes any
// suggestion; a real category is replaced only by a deterministic tier.
func Decide(current string, p Proposal) Decision {
	if !p.usable() {
		return Decision{ActionKeep, ReasonNoSuggestion}
	}
	if SameCategory(current, p.Category) {
		return Decision{ActionKeep, ReasonUnchanged}
	}
	if IsUnknownCategory(current) {
		return Decision{ActionChange, ""}
	}
	if p.Tier == TierDirectMapping || p.Tier == TierKeyword {
		return Decision{ActionChange, ""}
	}
	return Decision{ActionKeep, "tier " + p.Tier + " does not override a chosen category"}
}
