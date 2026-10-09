package icompta

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"fjacquet/camt-csv/internal/logging"

	"github.com/shopspring/decimal"
)

// ApplyLinks links the approved pairs of a report, both ways, as iCompta does:
// each split's linkedSplit names the other. Only linkedSplit and modification
// dates are written. A split used by two approved rows rejects the whole report
// before any backup or write.
func ApplyLinks(ctx context.Context, rep TransferReport, opts ApplyOptions, log logging.Logger) (ApplyResult, error) {
	var approved []TransferRow
	used := map[string]int{}
	for _, r := range rep.Rows {
		if !r.Apply {
			continue
		}
		approved = append(approved, r)
		used[r.DebitSplitID]++
		used[r.CreditSplitID]++
	}
	var twice []string
	for id, n := range used {
		if n > 1 {
			twice = append(twice, id)
		}
	}
	if len(twice) > 0 {
		sort.Strings(twice)
		return ApplyResult{}, fmt.Errorf("a split can be linked only once, but these appear in several approved rows: %s", strings.Join(twice, ", "))
	}

	db, backup, err := openForWrite(ctx, opts, rep.DBState, log)
	res := ApplyResult{BackupPath: backup}
	if err != nil {
		return res, err
	}
	defer func() { _ = db.Close() }()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("begin write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stamp := opts.Now().UTC().Format(iComptaTimeFormat)

	for _, r := range approved {
		reason, err := linkPair(ctx, tx, r, stamp)
		if err != nil {
			return res, err
		}
		if reason != "" {
			res.Skipped++
			log.WithFields(
				logging.Field{Key: "debit", Value: r.DebitSplitID},
				logging.Field{Key: "credit", Value: r.CreditSplitID},
				logging.Field{Key: "reason", Value: reason},
			).Warn("Skipped pair")
			continue
		}
		res.Applied++
		log.WithFields(
			logging.Field{Key: "debit_date", Value: r.DebitDate},
			logging.Field{Key: "credit_date", Value: r.CreditDate},
			logging.Field{Key: "from", Value: r.DebitAccount},
			logging.Field{Key: "to", Value: r.CreditAccount},
			logging.Field{Key: "amount", Value: r.Amount},
		).Info("Linked transfer")
	}

	if err := integrityCheck(ctx, tx); err != nil {
		return res, fmt.Errorf("after apply, rolled back: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return res, fmt.Errorf("commit: %w", err)
	}
	log.WithFields(
		logging.Field{Key: "linked", Value: res.Applied},
		logging.Field{Key: "skipped", Value: res.Skipped},
		logging.Field{Key: "backup", Value: backup},
	).Info("Transfers linked")
	return res, nil
}

type linkSide struct {
	txID   string
	amount decimal.Decimal
	linked bool
}

func readLinkSide(ctx context.Context, tx *sql.Tx, splitID string) (linkSide, error) {
	var side linkSide
	var amount, linked string
	err := tx.QueryRowContext(ctx, `
SELECT s."transaction", COALESCE(NULLIF(s.amount,''), t.amount, ''), COALESCE(s.linkedSplit,'')
FROM ICTransactionSplit s JOIN ICTransaction t ON t.ID = s."transaction"
WHERE s.ID = ?`, splitID).Scan(&side.txID, &amount, &linked)
	if err != nil {
		return side, err
	}
	side.linked = linked != ""
	if strings.TrimSpace(amount) != "" {
		side.amount, err = decimal.NewFromString(strings.TrimSpace(amount))
		if err != nil {
			return side, fmt.Errorf("split %s: amount %q: %w", splitID, amount, err)
		}
	}
	return side, nil
}

// linkPair links one approved row. It returns a skip reason when the pair no
// longer fits, or an error when the database misbehaves.
func linkPair(ctx context.Context, tx *sql.Tx, r TransferRow, stamp string) (string, error) {
	d, err := readLinkSide(ctx, tx, r.DebitSplitID)
	if errors.Is(err, sql.ErrNoRows) {
		return "debit split not found", nil
	}
	if err != nil {
		return "", fmt.Errorf("read split %s: %w", r.DebitSplitID, err)
	}
	c, err := readLinkSide(ctx, tx, r.CreditSplitID)
	if errors.Is(err, sql.ErrNoRows) {
		return "credit split not found", nil
	}
	if err != nil {
		return "", fmt.Errorf("read split %s: %w", r.CreditSplitID, err)
	}
	if d.linked || c.linked {
		return "already linked", nil
	}
	if !d.amount.IsNegative() || !d.amount.Round(2).Neg().Equal(c.amount.Round(2)) {
		return "amounts no longer match", nil
	}

	for _, u := range []struct{ split, other, txID string }{
		{r.DebitSplitID, r.CreditSplitID, d.txID},
		{r.CreditSplitID, r.DebitSplitID, c.txID},
	} {
		res, err := tx.ExecContext(ctx,
			`UPDATE ICTransactionSplit SET linkedSplit = ?, lastModificationDate = ? WHERE ID = ? AND COALESCE(linkedSplit,'') = ''`,
			u.other, stamp, u.split)
		if err != nil {
			return "", fmt.Errorf("link split %s: %w", u.split, err)
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return "", fmt.Errorf("link split %s: %d rows affected, expected 1", u.split, n)
		}
		res, err = tx.ExecContext(ctx,
			`UPDATE ICTransaction SET lastModificationDate = ? WHERE ID = ?`, stamp, u.txID)
		if err != nil {
			return "", fmt.Errorf("touch transaction %s: %w", u.txID, err)
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return "", fmt.Errorf("touch transaction %s: %d rows affected, expected 1", u.txID, n)
		}
	}
	return "", nil
}
