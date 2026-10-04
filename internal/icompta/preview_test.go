package icompta

import (
	"context"
	"errors"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClassifier answers from maps keyed by party name and records which method
// was used, so tests can assert the networked tiers are spared.
type fakeClassifier struct {
	full      map[string]models.Category
	fullErr   map[string]error
	local     map[string]models.Category
	fullCalls []string
	localCall []string
	onFull    func() // lets a test cancel mid-run
}

func (f *fakeClassifier) Full(_ context.Context, tx categorizer.Transaction) (models.Category, error) {
	f.fullCalls = append(f.fullCalls, tx.PartyName)
	if f.onFull != nil {
		f.onFull()
	}
	if err := f.fullErr[tx.PartyName]; err != nil {
		return models.Category{}, err
	}
	if c, ok := f.full[tx.PartyName]; ok {
		return c, nil
	}
	return models.Category{Name: models.CategoryUncategorized}, nil
}

func (f *fakeClassifier) Local(_ context.Context, tx categorizer.Transaction) (models.Category, bool) {
	f.localCall = append(f.localCall, tx.PartyName)
	c, ok := f.local[tx.PartyName]
	return c, ok
}

func testCategories() Categories {
	return Categories{
		byName: map[string]string{"Alimentation": "C-ALI", "Courses": "C-COU", "Voiture": "C-VOI", "Divers": "C-DIV"},
		byID:   map[string]string{"C-ALI": "Alimentation", "C-COU": "Courses", "C-VOI": "Voiture", "C-DIV": "Divers"},
	}
}

func cand(id, name, cat string) Candidate {
	return Candidate{SplitID: id, Date: "2026-01-02", Name: name, Amount: decimal.RequireFromString("-5"), CategoryName: cat}
}

func TestPreview(t *testing.T) {
	cl := &fakeClassifier{
		full: map[string]models.Category{
			"EMPTY AI":    {Name: "Courses", Source: TierAI},
			"EMPTY NOCAT": {Name: "Brand New", Source: TierAI}, // not in ICCategory
			"UNKNOWN SEM": {Name: "Voiture", Source: TierSemantic},
			"ERR":         {},
		},
		fullErr: map[string]error{"ERR": errors.New("provider down")},
		local: map[string]models.Category{
			"REAL KEYWORD": {Name: "Courses", Source: TierKeyword},
			"REAL DIRECT":  {Name: "Courses", Source: TierDirectMapping},
			"REAL SAME":    {Name: "Alimentation", Source: TierDirectMapping},
		},
	}
	snap := Snapshot{
		State:      "splits=9",
		Categories: testCategories(),
		Candidates: []Candidate{
			cand("S1", "EMPTY AI", ""),
			cand("S2", "EMPTY NOCAT", ""),
			cand("S3", "UNKNOWN SEM", "Divers"),
			cand("S4", "NOTHING", ""),
			cand("S5", "REAL KEYWORD", "Alimentation"),
			cand("S11", "REAL DIRECT", "Alimentation"),
			cand("S6", "REAL SAME", "Alimentation"),
			cand("S7", "REAL NOMATCH", "Alimentation"),
			{SplitID: "S8", Name: "BUY", IsInvestment: true},
			{SplitID: "S9", Name: "TRANSFER", IsLinked: true},
			cand("S10", "ERR", ""),
		},
	}

	rep := Preview(context.Background(), snap, cl, logging.NewMockLogger())

	assert.Equal(t, "splits=9", rep.DBState)
	got := map[string]Row{}
	for _, r := range rep.Rows {
		got[r.SplitID] = r
	}

	assert.Equal(t, Row{SplitID: "S1", Date: "2026-01-02", Name: "EMPTY AI", Amount: "-5",
		NewCategory: "Courses", Tier: TierAI, Decision: ActionChange, Apply: true}, got["S1"])
	assert.Equal(t, ActionSkip, got["S2"].Decision)
	assert.Equal(t, "unknown category", got["S2"].Reason)
	assert.False(t, got["S2"].Apply)
	assert.Equal(t, ActionChange, got["S3"].Decision)
	assert.Equal(t, "Divers", got["S3"].OldCategory)
	assert.Equal(t, ActionKeep, got["S4"].Decision)
	assert.Equal(t, ReasonNoSuggestion, got["S4"].Reason)
	assert.Equal(t, ActionChange, got["S11"].Decision)
	assert.Equal(t, TierDirectMapping, got["S11"].Tier)
	assert.Equal(t, ActionSkip, got["S10"].Decision)
	assert.Contains(t, got["S10"].Reason, "categorizer error")

	for _, id := range []string{"S5", "S6", "S7", "S8", "S9"} {
		_, present := got[id]
		assert.False(t, present, "%s must not appear in the report", id)
	}

	// Networked tiers are asked only for empty/unknown splits.
	assert.ElementsMatch(t, []string{"EMPTY AI", "EMPTY NOCAT", "UNKNOWN SEM", "NOTHING", "ERR"}, cl.fullCalls)
	assert.ElementsMatch(t, []string{"REAL KEYWORD", "REAL DIRECT", "REAL SAME", "REAL NOMATCH"}, cl.localCall)
}

type captureClassifier struct{ seen []categorizer.Transaction }

func (c *captureClassifier) Full(_ context.Context, tx categorizer.Transaction) (models.Category, error) {
	c.seen = append(c.seen, tx)
	return models.Category{Name: models.CategoryUncategorized}, nil
}
func (c *captureClassifier) Local(context.Context, categorizer.Transaction) (models.Category, bool) {
	return models.Category{}, false
}

func TestPreview_PassesContextToTheCategorizer(t *testing.T) {
	cl := &captureClassifier{}
	c := Candidate{SplitID: "S1", Name: "COOP", Comment: "ref 9", Date: "2026-03-04",
		Amount: decimal.RequireFromString("-12.30")}
	Preview(context.Background(), Snapshot{Categories: testCategories(), Candidates: []Candidate{c}}, cl, logging.NewMockLogger())

	require.Len(t, cl.seen, 1)
	assert.Equal(t, "COOP", cl.seen[0].PartyName)
	assert.True(t, cl.seen[0].IsDebtor)
	assert.Equal(t, "-12.3", cl.seen[0].Amount)
	assert.Equal(t, "2026-03-04", cl.seen[0].Date)
	assert.Equal(t, "COOP ref 9", cl.seen[0].Info)
}

// Review focus 5: Ctrl-C must leave a valid partial report, not nothing.
func TestPreview_CancelledKeepsPartialReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cl := &fakeClassifier{
		full:   map[string]models.Category{"A": {Name: "Courses", Source: TierAI}, "B": {Name: "Courses", Source: TierAI}},
		onFull: cancel, // cancel as soon as the first split is classified
	}
	snap := Snapshot{State: "s", Categories: testCategories(),
		Candidates: []Candidate{cand("S1", "A", ""), cand("S2", "B", "")}}

	rep := Preview(ctx, snap, cl, logging.NewMockLogger())

	require.Len(t, rep.Rows, 1, "the split decided before cancellation is kept")
	assert.Equal(t, "S1", rep.Rows[0].SplitID)
	assert.Equal(t, "s", rep.DBState)
	assert.Equal(t, []string{"A"}, cl.fullCalls, "no further splits are classified after cancellation")
}

func TestPreview_LogsSummary(t *testing.T) {
	log := logging.NewMockLogger()
	cl := &fakeClassifier{full: map[string]models.Category{"A": {Name: "Courses", Source: TierAI}}}
	Preview(context.Background(), Snapshot{Categories: testCategories(),
		Candidates: []Candidate{cand("S1", "A", ""), {SplitID: "S2", IsInvestment: true}}}, cl, log)

	assert.True(t, log.HasEntry("INFO", "Recategorization preview complete"))
}

// Hundreds of rows are easier to review when identical moves sit together, so
// the report is grouped by old and new category.
func TestPreview_GroupsRowsByCategoryMove(t *testing.T) {
	cl := &fakeClassifier{
		full: map[string]models.Category{
			"A": {Name: "Voiture", Source: TierAI},
			"B": {Name: "Courses", Source: TierAI},
			"C": {Name: "Voiture", Source: TierAI},
			"D": {Name: "Courses", Source: TierAI},
		},
	}
	snap := Snapshot{Categories: testCategories(), Candidates: []Candidate{
		cand("S1", "A", ""), cand("S2", "B", "Divers"), cand("S3", "C", ""), cand("S4", "D", "Divers"),
	}}

	rep := Preview(context.Background(), snap, cl, logging.NewMockLogger())

	var ids []string
	for _, r := range rep.Rows {
		ids = append(ids, r.SplitID)
	}
	// ("" -> Voiture): S1,S3 then (Divers -> Courses): S2,S4, each keeping date order.
	assert.Equal(t, []string{"S1", "S3", "S2", "S4"}, ids)
}
