package icompta

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// transferColumns are what link-transfers reads beyond requiredColumns. They are
// checked only by TransferSnapshot, so recategorize keeps working on databases
// (and fixtures) without them.
var transferColumns = map[string][]string{
	"ICAccount":     {"ID", "name", "class", "parent", "currency"},
	"ICTransaction": {"account", "status"},
}

// TransferSnapshot is a consistent read of what link-transfers needs.
type TransferSnapshot struct {
	Accounts []Account
	Legs     []Leg
	State    string
}

// TransferSnapshot reads accounts, every split as a Leg, and the state marker in
// one read transaction.
func (s *Store) TransferSnapshot(ctx context.Context) (TransferSnapshot, error) {
	if err := requireColumns(ctx, s.db, transferColumns); err != nil {
		return TransferSnapshot{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TransferSnapshot{}, fmt.Errorf("begin read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cats, err := loadCategories(ctx, tx)
	if err != nil {
		return TransferSnapshot{}, err
	}
	state, err := readState(ctx, tx)
	if err != nil {
		return TransferSnapshot{}, err
	}

	arows, err := tx.QueryContext(ctx, `
SELECT ID, COALESCE(name,''), COALESCE(class,''), COALESCE(parent,''), COALESCE(currency,'')
FROM ICAccount`)
	if err != nil {
		return TransferSnapshot{}, fmt.Errorf("read accounts: %w", err)
	}
	var accounts []Account
	byID := map[string]Account{}
	for arows.Next() {
		var a Account
		if err := arows.Scan(&a.ID, &a.Name, &a.Class, &a.ParentID, &a.CurrencyID); err != nil {
			_ = arows.Close()
			return TransferSnapshot{}, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
		byID[a.ID] = a
	}
	if err := arows.Err(); err != nil {
		_ = arows.Close()
		return TransferSnapshot{}, fmt.Errorf("read accounts: %w", err)
	}
	_ = arows.Close()

	rows, err := tx.QueryContext(ctx, `
SELECT s.ID, t.ID, COALESCE(t.account,''), substr(t.date,1,10), t.name,
       COALESCE(NULLIF(s.amount,''), t.amount, ''), COALESCE(s.category,''), COALESCE(t.status,''),
       (SELECT COUNT(*) FROM ICTransactionSplit s2 WHERE s2."transaction" = t.ID),
       CASE WHEN COALESCE(s.linkedSplit,'') <> '' THEN 1 ELSE 0 END
FROM ICTransactionSplit s
JOIN ICTransaction t ON t.ID = s."transaction"
ORDER BY t.date, s.ID`)
	if err != nil {
		return TransferSnapshot{}, fmt.Errorf("read splits: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var legs []Leg
	for rows.Next() {
		var l Leg
		var amount, categoryID string
		var linked int
		if err := rows.Scan(&l.SplitID, &l.TxID, &l.AccountID, &l.Date, &l.Name,
			&amount, &categoryID, &l.Status, &l.SplitCount, &linked); err != nil {
			return TransferSnapshot{}, fmt.Errorf("scan split: %w", err)
		}
		if strings.TrimSpace(amount) != "" {
			l.Amount, err = decimal.NewFromString(strings.TrimSpace(amount))
			if err != nil {
				return TransferSnapshot{}, fmt.Errorf("split %s: amount %q: %w", l.SplitID, amount, err)
			}
		}
		l.Category = cats.NameByID(categoryID)
		l.AccountName = byID[l.AccountID].Name
		l.CurrencyID = byID[l.AccountID].CurrencyID
		l.Linked = linked == 1
		legs = append(legs, l)
	}
	if err := rows.Err(); err != nil {
		return TransferSnapshot{}, fmt.Errorf("read splits: %w", err)
	}
	return TransferSnapshot{Accounts: accounts, Legs: legs, State: state}, nil
}
