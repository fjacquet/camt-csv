package icompta

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// progressEvery is how often preview logs progress. The AI tier is rate
// limited, so a first run can take a long time and the user needs a heartbeat.
const progressEvery = 100

// Classifier is the slice of the categorizer a preview needs.
type Classifier interface {
	// Full runs every tier, including the semantic and AI tiers.
	Full(ctx context.Context, tx categorizer.Transaction) (models.Category, error)
	// Local runs only the tiers that need no network.
	Local(ctx context.Context, tx categorizer.Transaction) (models.Category, bool)
}

type categorizerClassifier struct{ c *categorizer.Categorizer }

// NewCategorizerClassifier adapts the real categorizer. Full uses
// CategorizeTransaction, which never auto-learns.
func NewCategorizerClassifier(c *categorizer.Categorizer) Classifier {
	return categorizerClassifier{c: c}
}

func (k categorizerClassifier) Full(ctx context.Context, tx categorizer.Transaction) (models.Category, error) {
	return k.c.CategorizeTransaction(ctx, tx)
}

func (k categorizerClassifier) Local(ctx context.Context, tx categorizer.Transaction) (models.Category, bool) {
	return k.c.CategorizeLocal(ctx, tx)
}

// Preview classifies every in-scope split and returns the report. It never
// touches the database. Empty or unknown splits go through every tier; splits
// with a real category go through the local tiers only, since those are the
// only ones allowed to override a chosen category. Cancelling the context
// stops the run and returns the rows decided so far.
func Preview(ctx context.Context, snap Snapshot, cl Classifier, log logging.Logger) Report {
	rep := Report{DBState: snap.State}
	excluded := map[string]int{}
	decisions := map[string]int{}
	tiers := map[string]int{}
	processed := 0

	for _, c := range snap.Candidates {
		if ctx.Err() != nil {
			log.WithFields(
				logging.Field{Key: "processed", Value: processed},
				logging.Field{Key: "total", Value: len(snap.Candidates)},
			).Warn("Preview cancelled, keeping the rows decided so far")
			break
		}
		if why := Exclusion(c); why != "" {
			excluded[why]++
			continue
		}
		processed++
		if processed%progressEvery == 0 {
			log.WithFields(
				logging.Field{Key: "processed", Value: processed},
				logging.Field{Key: "total", Value: len(snap.Candidates)},
			).Info("Preview progress")
		}

		row, report := classify(ctx, c, snap.Categories, cl, log)
		if !report {
			continue
		}
		rep.Rows = append(rep.Rows, row)
		decisions[string(row.Decision)]++
		if row.Tier != "" {
			tiers[row.Tier]++
		}
	}

	// Identical moves side by side are far quicker to review than date order.
	sort.SliceStable(rep.Rows, func(i, j int) bool {
		a, b := rep.Rows[i], rep.Rows[j]
		if a.OldCategory != b.OldCategory {
			return a.OldCategory < b.OldCategory
		}
		return a.NewCategory < b.NewCategory
	})

	log.WithFields(
		logging.Field{Key: "candidates", Value: len(snap.Candidates)},
		logging.Field{Key: "reported", Value: len(rep.Rows)},
		logging.Field{Key: "excluded", Value: fmt.Sprint(excluded)},
		logging.Field{Key: "decisions", Value: fmt.Sprint(decisions)},
		logging.Field{Key: "tiers", Value: fmt.Sprint(tiers)},
	).Info("Recategorization preview complete")
	return rep
}

// classify returns the report row for one split, and whether it belongs in the
// report at all (a real category left unchanged does not).
func classify(ctx context.Context, c Candidate, cats Categories, cl Classifier, log logging.Logger) (Row, bool) {
	tx := categorizer.Transaction{
		PartyName: c.PartyName(),
		IsDebtor:  c.IsDebtor(),
		Amount:    c.Amount.String(),
		Date:      c.Date,
		Info:      strings.TrimSpace(c.Name + " " + c.Comment),
	}
	row := Row{
		SplitID:     c.SplitID,
		Date:        c.Date,
		Name:        c.Name,
		Amount:      c.Amount.String(),
		OldCategory: c.CategoryName,
	}

	var proposal Proposal
	if IsUnknownCategory(c.CategoryName) {
		cat, err := cl.Full(ctx, tx)
		if err != nil {
			log.WithError(err).WithFields(logging.Field{Key: "split", Value: c.SplitID}).
				Warn("Categorizer failed for a split")
			row.Decision, row.Reason = ActionSkip, "categorizer error: "+err.Error()
			return row, true
		}
		proposal = Proposal{Category: cat.Name, Tier: cat.Source}
	} else {
		cat, found := cl.Local(ctx, tx)
		if !found {
			return Row{}, false
		}
		proposal = Proposal{Category: cat.Name, Tier: cat.Source}
	}

	d := Decide(c.CategoryName, proposal)
	if d.Reason == ReasonUnchanged {
		return Row{}, false
	}
	// A real category that a weaker tier merely disagrees with is not news:
	// reporting it would bury the changes worth reviewing.
	if d.Action == ActionKeep && !IsUnknownCategory(c.CategoryName) {
		return Row{}, false
	}
	row.NewCategory = proposal.Category
	row.Tier = proposal.Tier
	row.Decision = d.Action
	row.Reason = d.Reason
	if d.Action != ActionChange {
		return row, true
	}
	if _, ok := cats.IDByName(proposal.Category); !ok {
		row.Decision, row.Reason = ActionSkip, "unknown category"
		return row, true
	}
	row.Apply = true
	return row, true
}
