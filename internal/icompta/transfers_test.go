package icompta

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func leg(id, acct, date, amount string) Leg {
	return Leg{SplitID: id, TxID: "T" + id, AccountID: acct, AccountName: "acct " + acct, CurrencyID: "CHF",
		Date: date, Name: "tx " + id, Amount: decimal.RequireFromString(amount), Category: "Virements", SplitCount: 1}
}

func TestMatchTransfers_SurePair(t *testing.T) {
	pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-750"), leg("C", "B", "2026-01-11", "750")})
	require.Len(t, pairs, 1)
	p := pairs[0]
	assert.Equal(t, "D", p.Debit.SplitID)
	assert.Equal(t, "C", p.Credit.SplitID)
	assert.Equal(t, 1, p.GapDays)
	assert.True(t, p.Sure)
	assert.Empty(t, p.Reason)
	assert.Empty(t, p.Note)
}

func TestMatchTransfers_Gaps(t *testing.T) {
	for _, tc := range []struct {
		credit string
		want   int // number of pairs
		sure   bool
		reason string
	}{
		{"2026-01-12", 1, true, ""},
		{"2026-01-13", 1, false, "gap 3 days"},
		{"2026-01-14", 1, false, "gap 4 days"},
		{"2026-01-15", 0, false, ""},
		{"2026-01-06", 1, false, "gap 4 days"}, // credit before debit counts too
	} {
		pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-100"), leg("C", "B", tc.credit, "100")})
		require.Len(t, pairs, tc.want, tc.credit)
		if tc.want == 1 {
			assert.Equal(t, tc.sure, pairs[0].Sure, tc.credit)
			assert.Equal(t, tc.reason, pairs[0].Reason, tc.credit)
		}
	}
}

func TestMatchTransfers_TwoCreditsForOneDebit(t *testing.T) {
	pairs := MatchTransfers([]Leg{
		leg("D", "A", "2026-01-10", "-100"),
		leg("C1", "B", "2026-01-10", "100"),
		leg("C2", "C", "2026-01-11", "100"),
	})
	require.Len(t, pairs, 2, "every combination is listed")
	for _, p := range pairs {
		assert.False(t, p.Sure)
		assert.Equal(t, "2 candidates for the debit", p.Reason)
	}
}

// Review focus 4: one credit, the same debit imported twice.
func TestMatchTransfers_DuplicateDebit(t *testing.T) {
	pairs := MatchTransfers([]Leg{
		leg("D1", "A", "2026-01-10", "-100"),
		leg("D2", "A", "2026-01-10", "-100"),
		leg("C", "B", "2026-01-10", "100"),
	})
	require.Len(t, pairs, 2)
	for _, p := range pairs {
		assert.False(t, p.Sure)
		assert.Equal(t, "2 candidates for the credit", p.Reason)
	}
}

func TestMatchTransfers_NoPair(t *testing.T) {
	eur := leg("C", "B", "2026-01-10", "100")
	eur.CurrencyID = "EUR"
	for name, legs := range map[string][]Leg{
		"different currency": {leg("D", "A", "2026-01-10", "-100"), eur},
		"same account":       {leg("D", "A", "2026-01-10", "-100"), leg("C", "A", "2026-01-10", "100")},
		"amount differs":     {leg("D", "A", "2026-01-10", "-100"), leg("C", "B", "2026-01-10", "100.05")},
		"same sign":          {leg("D", "A", "2026-01-10", "100"), leg("C", "B", "2026-01-10", "100")},
		"bad date":           {leg("D", "A", "10.01.2026", "-100"), leg("C", "B", "2026-01-10", "100")},
	} {
		assert.Empty(t, MatchTransfers(legs), name)
	}
}

func TestMatchTransfers_CentsRounding(t *testing.T) {
	pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-100.004"), leg("C", "B", "2026-01-10", "100.00")})
	require.Len(t, pairs, 1)
}

func TestMatchTransfers_CategoriesDifferIsANoteOnly(t *testing.T) {
	c := leg("C", "B", "2026-01-10", "100")
	c.Category = "Revenus Financiers"
	pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-100"), c})
	require.Len(t, pairs, 1)
	assert.True(t, pairs[0].Sure)
	assert.Equal(t, "categories differ", pairs[0].Note)
}

func TestMatchTransfers_DeterministicOrder(t *testing.T) {
	legs := []Leg{
		leg("D2", "A", "2026-02-01", "-50"), leg("C2", "B", "2026-02-01", "50"),
		leg("D1", "A", "2026-01-01", "-20"), leg("C1", "B", "2026-01-01", "20"),
		leg("D3", "A", "2026-01-01", "-10"), leg("C3", "B", "2026-01-01", "10"),
	}
	first := MatchTransfers(legs)
	reversed := make([]Leg, len(legs))
	for i, l := range legs {
		reversed[len(legs)-1-i] = l
	}
	second := MatchTransfers(reversed)
	require.Len(t, first, 3)
	assert.Equal(t, first, second)
	assert.Equal(t, []string{"D3", "D1", "D2"}, []string{first[0].Debit.SplitID, first[1].Debit.SplitID, first[2].Debit.SplitID},
		"by date, then smaller amount first")
}

func TestEligibleLegs(t *testing.T) {
	scope := map[string]bool{"A": true, "B": true}
	ok := leg("OK", "A", "2026-01-10", "-1")
	out := leg("OUT", "Z", "2026-01-10", "-1")
	linked := leg("LNK", "A", "2026-01-10", "-1")
	linked.Linked = true
	multi := leg("MUL", "A", "2026-01-10", "-1")
	multi.SplitCount = 2
	zero := leg("ZER", "A", "2026-01-10", "0.001")
	planned := leg("PLN", "A", "2026-01-10", "-1")
	planned.Status = "ICTransactionStatus.PlannedStatus"

	got := EligibleLegs([]Leg{ok, out, linked, multi, zero, planned}, scope)
	require.Len(t, got, 1)
	assert.Equal(t, "OK", got[0].SplitID)
}

func TestPairsInRange(t *testing.T) {
	pairs := MatchTransfers([]Leg{leg("D", "A", "2026-08-30", "-100"), leg("C", "B", "2026-09-01", "100")})
	require.Len(t, pairs, 1, "gap 2 days")
	for _, tc := range []struct {
		from, to string
		kept     bool
	}{
		{"2026-09-01", "2026-09-30", true}, // credit inside
		{"2026-08-01", "2026-08-31", true}, // debit inside
		{"2026-10-01", "2026-10-31", false},
		{"", "", true},
	} {
		assert.Equal(t, tc.kept, len(PairsInRange(pairs, tc.from, tc.to)) == 1, tc.from+".."+tc.to)
	}
}

// Ambiguity is decided on every eligible leg, so a candidate just outside the
// range still makes the in-range pair doubtful.
func TestPairsInRange_AmbiguityCountsOutOfRangeLegs(t *testing.T) {
	pairs := MatchTransfers([]Leg{
		leg("D", "A", "2026-08-31", "-100"),
		leg("C", "B", "2026-09-01", "100"),
		leg("D2", "C", "2026-09-02", "-100"),
	})
	kept := PairsInRange(pairs, "2026-09-01", "2026-09-30")
	require.Len(t, kept, 2)
	for _, p := range kept {
		assert.False(t, p.Sure)
		assert.Equal(t, "2 candidates for the credit", p.Reason)
	}
}

// A card purchase that coincides with a pocket move carries two real spending
// categories: it must not be pre-approved.
func TestMatchTransfers_DifferentSpendingCategoriesAreDoubtful(t *testing.T) {
	c := leg("C", "B", "2026-01-10", "100")
	c.Category = "Voyages"
	d := leg("D", "A", "2026-01-10", "-100")
	d.Category = "Abonnements"
	pairs := MatchTransfers([]Leg{d, c})
	require.Len(t, pairs, 1)
	assert.False(t, pairs[0].Sure)
	assert.Equal(t, "categories differ, neither is a transfer", pairs[0].Reason)
	assert.Equal(t, "categories differ", pairs[0].Note)
}

// An uncategorised side does not block pre-approval; the note still shows.
func TestMatchTransfers_UncategorisedSideStaysSure(t *testing.T) {
	c := leg("C", "B", "2026-01-10", "100")
	c.Category = ""
	d := leg("D", "A", "2026-01-10", "-100")
	d.Category = "Frais Bancaires"
	pairs := MatchTransfers([]Leg{d, c})
	require.Len(t, pairs, 1)
	assert.True(t, pairs[0].Sure)
	assert.Empty(t, pairs[0].Reason)
	assert.Equal(t, "categories differ", pairs[0].Note)
}
