package icompta

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
)

// A candidate pair's dates may be at most maxGapDays apart; only pairs at most
// sureGapDays apart, with no competing candidate, are pre-approved.
const (
	maxGapDays  = 4
	sureGapDays = 2
)

// transferCategory is the category a transfer between own accounts is filed
// under. A pair whose categories differ is pre-approved only when one side is
// filed as a transfer or not filed at all: a card purchase that happens to
// match a Revolut pocket move (Apple 3.00 vs "To CHF Vacances" 3.00) carries
// two real spending categories.
const transferCategory = "Virements"

func transferLike(category string) bool {
	return models.IsUnknownCategory(category) || normalize(category) == normalize(transferCategory)
}

// Leg is one split that could be one side of a transfer.
type Leg struct {
	SplitID, TxID, AccountID, AccountName, CurrencyID string
	Date                                              string // YYYY-MM-DD
	Name                                              string
	Amount                                            decimal.Decimal
	Category                                          string
	Status                                            string
	SplitCount                                        int
	Linked                                            bool
}

// Pair is a candidate transfer: money leaving Debit's account and arriving in
// Credit's.
type Pair struct {
	Debit, Credit Leg
	GapDays       int
	Sure          bool
	Reason        string // why it is not sure; "" when Sure
	Note          string // informational, e.g. "categories differ"
}

// EligibleLegs keeps the legs that may be one side of a transfer: in scope, not
// already linked, the only split of their transaction, non-zero, and not
// planned. Dates are not filtered here; see PairsInRange.
func EligibleLegs(legs []Leg, scope map[string]bool) []Leg {
	var out []Leg
	for _, l := range legs {
		switch {
		case !scope[l.AccountID], l.Linked, l.SplitCount != 1, l.Amount.Round(2).IsZero(),
			strings.Contains(l.Status, "Planned"):
			continue
		}
		out = append(out, l)
	}
	return out
}

// MatchTransfers pairs debits with credits of the same amount, in another
// account of the same currency, at most maxGapDays apart. Every combination is
// returned; a pair is Sure only when it is close and unambiguous on both sides.
func MatchTransfers(legs []Leg) []Pair {
	var debits, credits []Leg
	for _, l := range legs {
		if l.Amount.IsNegative() {
			debits = append(debits, l)
		} else {
			credits = append(credits, l)
		}
	}
	type candidate struct{ d, c, gap int }
	var cands []candidate
	perDebit := map[int]int{}
	perCredit := map[int]int{}
	for i, d := range debits {
		for j, c := range credits {
			if d.AccountID == c.AccountID || d.CurrencyID != c.CurrencyID {
				continue
			}
			if !d.Amount.Round(2).Neg().Equal(c.Amount.Round(2)) {
				continue
			}
			gap, ok := dayGap(d.Date, c.Date)
			if !ok || gap > maxGapDays {
				continue
			}
			cands = append(cands, candidate{i, j, gap})
			perDebit[i]++
			perCredit[j]++
		}
	}

	pairs := make([]Pair, 0, len(cands))
	for _, k := range cands {
		p := Pair{Debit: debits[k.d], Credit: credits[k.c], GapDays: k.gap}
		var reasons []string
		if k.gap > sureGapDays {
			reasons = append(reasons, fmt.Sprintf("gap %d days", k.gap))
		}
		if n := perDebit[k.d]; n > 1 {
			reasons = append(reasons, fmt.Sprintf("%d candidates for the debit", n))
		}
		if n := perCredit[k.c]; n > 1 {
			reasons = append(reasons, fmt.Sprintf("%d candidates for the credit", n))
		}
		if normalize(p.Debit.Category) != normalize(p.Credit.Category) {
			p.Note = "categories differ"
			if !transferLike(p.Debit.Category) && !transferLike(p.Credit.Category) {
				reasons = append(reasons, "categories differ, neither is a transfer")
			}
		}
		p.Sure = len(reasons) == 0
		p.Reason = strings.Join(reasons, "; ")
		pairs = append(pairs, p)
	}

	sort.SliceStable(pairs, func(a, b int) bool {
		x, y := pairs[a], pairs[b]
		switch {
		case x.Debit.Date != y.Debit.Date:
			return x.Debit.Date < y.Debit.Date
		case x.Debit.AccountName != y.Debit.AccountName:
			return x.Debit.AccountName < y.Debit.AccountName
		case !x.Debit.Amount.Equal(y.Debit.Amount):
			return x.Debit.Amount.GreaterThan(y.Debit.Amount) // -10 before -20: smaller amount first
		case x.Debit.SplitID != y.Debit.SplitID:
			return x.Debit.SplitID < y.Debit.SplitID
		case x.Credit.Date != y.Credit.Date:
			return x.Credit.Date < y.Credit.Date
		}
		return x.Credit.SplitID < y.Credit.SplitID
	})
	return pairs
}

// PairsInRange keeps the pairs with at least one leg dated in [from, to]
// (inclusive, YYYY-MM-DD; an empty bound is open). Matching runs on every
// eligible leg first, so a transfer that crosses the range boundary is still
// found and ambiguity counts include legs just outside the range.
func PairsInRange(pairs []Pair, from, to string) []Pair {
	in := func(d string) bool { return (from == "" || d >= from) && (to == "" || d <= to) }
	var out []Pair
	for _, p := range pairs {
		if in(p.Debit.Date) || in(p.Credit.Date) {
			out = append(out, p)
		}
	}
	return out
}

// dayGap is the absolute number of days between two YYYY-MM-DD dates.
func dayGap(a, b string) (int, bool) {
	ta, err := time.Parse("2006-01-02", a)
	if err != nil {
		return 0, false
	}
	tb, err := time.Parse("2006-01-02", b)
	if err != nil {
		return 0, false
	}
	d := int(ta.Sub(tb).Hours() / 24)
	if d < 0 {
		d = -d
	}
	return d, true
}
