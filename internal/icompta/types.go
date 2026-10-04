// Package icompta recategorizes transactions stored in an iCompta database.
package icompta

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Categorization tiers, as reported in models.Category.Source.
const (
	TierDirectMapping = "direct_mapping"
	TierKeyword       = "keyword"
	TierSemantic      = "semantic"
	TierAI            = "ai"
)

// Action is what the policy decided for one split.
type Action string

const (
	ActionChange Action = "change"
	ActionKeep   Action = "keep"
	ActionSkip   Action = "skipped"
)

// Reasons attached to a keep decision that callers branch on.
const (
	ReasonUnchanged    = "unchanged"
	ReasonNoSuggestion = "no suggestion"
)

// Decision is the policy outcome: an action and, when it is not a plain
// change, the reason.
type Decision struct {
	Action Action
	Reason string
}

// Proposal is what the categorizer suggested for a split.
type Proposal struct {
	Category string
	Tier     string
}

// Candidate is one iCompta split with the context needed to categorize it.
type Candidate struct {
	SplitID      string
	Date         string // ICTransaction.date, YYYY-MM-DD
	Name         string // ICTransaction.name, usually the bank label
	Payee        string
	Comment      string
	Amount       decimal.Decimal
	CategoryID   string // "" when the split has no category
	CategoryName string // "" when CategoryID is empty or dangling
	IsInvestment bool
	IsLinked     bool
}

// PartyName is the payee when set, else the transaction name. In real data the
// payee is mostly empty and the name carries the bank label.
func (c Candidate) PartyName() string {
	if p := strings.TrimSpace(c.Payee); p != "" {
		return p
	}
	return strings.TrimSpace(c.Name)
}

// IsDebtor follows the repo convention (see revolutinvestmentparser): money
// out is a debtor transaction.
func (c Candidate) IsDebtor() bool {
	return c.Amount.IsNegative()
}
