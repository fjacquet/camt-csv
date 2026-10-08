package icompta

import (
	"strings"

	"golang.org/x/text/unicode/norm"

	"fjacquet/camt-csv/internal/models"
)

// normalize makes category names comparable: macOS can return decomposed
// accents, so compare NFC.
func normalize(s string) string {
	return norm.NFC.String(strings.TrimSpace(s))
}

// IsUnknownCategory reports whether a category name carries no decision.
func IsUnknownCategory(name string) bool {
	return models.IsUnknownCategory(name)
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

// usable reports whether the categorizer actually suggested something. A
// suggestion that is itself an "unknown" category (Divers, Non Classé, Autre,
// Uncategorized) says nothing: swapping one for another is noise, and turning a
// chosen category into one would be a downgrade.
func (p Proposal) usable() bool {
	return !IsUnknownCategory(p.Category)
}

// Decide applies the overwrite rule: an unknown or empty category takes any
// suggestion; a real category is replaced only by a direct mapping, an exact
// match on the party name. Keyword matching is a fill-in, not a verdict: on real
// data it filed supermarkets under Pension and swapped near-synonyms
// (Transferts/Virements, Alimentation/Courses), so it never overrides a choice.
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
	if p.Tier == TierDirectMapping {
		return Decision{ActionChange, ""}
	}
	return Decision{ActionKeep, "tier " + p.Tier + " does not override a chosen category"}
}
