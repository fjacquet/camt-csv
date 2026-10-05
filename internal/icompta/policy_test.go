package icompta

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		proposal   Proposal
		wantAction Action
		wantReason string
	}{
		{"empty filled by ai", "", Proposal{"Alimentation", TierAI}, ActionChange, ""},
		{"unknown replaced by semantic", "Divers", Proposal{"Voiture", TierSemantic}, ActionChange, ""},
		{"real never replaced by keyword", "Alimentation", Proposal{"Courses", TierKeyword}, ActionKeep, "does not override"},
		{"unknown filled by keyword", "Divers", Proposal{"Courses", TierKeyword}, ActionChange, ""},
		{"empty filled by keyword", "", Proposal{"Courses", TierKeyword}, ActionChange, ""},
		{"real replaced by direct mapping", "Alimentation", Proposal{"Courses", TierDirectMapping}, ActionChange, ""},
		{"real never replaced by ai", "Alimentation", Proposal{"Courses", TierAI}, ActionKeep, "does not override"},
		{"real never replaced by semantic", "Alimentation", Proposal{"Courses", TierSemantic}, ActionKeep, "does not override"},
		{"same category is unchanged", "Alimentation", Proposal{"Alimentation", TierDirectMapping}, ActionKeep, ReasonUnchanged},
		{"uncategorized proposal is no suggestion", "", Proposal{"Uncategorized", ""}, ActionKeep, ReasonNoSuggestion},
		{"failed ai proposal is no suggestion", "Divers", Proposal{"Uncategorized (AI)", TierAI}, ActionKeep, ReasonNoSuggestion},
		{"empty proposal is no suggestion", "Non Classé", Proposal{}, ActionKeep, ReasonNoSuggestion},
		{"unknown to unknown is no suggestion", "Divers", Proposal{"Non Classé", TierAI}, ActionKeep, ReasonNoSuggestion},
		{"empty to unknown is no suggestion", "", Proposal{"Divers", TierKeyword}, ActionKeep, ReasonNoSuggestion},
		{"real to unknown is no suggestion", "Alimentation", Proposal{"Autre", TierDirectMapping}, ActionKeep, ReasonNoSuggestion},
		{"decomposed accent still unknown", "Non Classé", Proposal{"Alimentation", TierAI}, ActionChange, ""},
		{"decomposed accent still same", "Séjours", Proposal{"Séjours", TierKeyword}, ActionKeep, ReasonUnchanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.current, tt.proposal)
			assert.Equal(t, tt.wantAction, got.Action)
			assert.Contains(t, got.Reason, tt.wantReason)
		})
	}
}

func TestIsUnknownCategory(t *testing.T) {
	for _, n := range []string{"", "  ", "Divers", "Non Classé", "Autre", "Uncategorized", "Uncategorized (AI)"} {
		assert.True(t, IsUnknownCategory(n), n)
	}
	for _, n := range []string{"Alimentation", "Voiture", "Épargne"} {
		assert.False(t, IsUnknownCategory(n), n)
	}
}

func TestExclusion(t *testing.T) {
	assert.Equal(t, "", Exclusion(Candidate{}))
	assert.Equal(t, "investment", Exclusion(Candidate{IsInvestment: true}))
	assert.Equal(t, "linked transfer", Exclusion(Candidate{IsLinked: true}))
	assert.Equal(t, "investment", Exclusion(Candidate{IsInvestment: true, IsLinked: true}))
}

func TestCandidate_PartyNameAndDirection(t *testing.T) {
	assert.Equal(t, "Migros", Candidate{Name: "Achat 12/04", Payee: " Migros "}.PartyName())
	assert.Equal(t, "Achat 12/04", Candidate{Name: "Achat 12/04"}.PartyName())
	assert.True(t, Candidate{Amount: decimal.RequireFromString("-8.70")}.IsDebtor())
	assert.False(t, Candidate{Amount: decimal.RequireFromString("8.70")}.IsDebtor())
	assert.False(t, Candidate{}.IsDebtor(), "zero amount is not a debit")
}
