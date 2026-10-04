package icompta

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
	_ "modernc.org/sqlite" // pure-Go driver, registers "sqlite"
)

// requiredColumns are the columns this package reads or writes. A database
// missing one fails fast with a clear message instead of a mid-run SQL error.
var requiredColumns = map[string][]string{
	"ICCategory":         {"ID", "name"},
	"ICTransaction":      {"ID", "lastModificationDate", "date", "name", "comment", "amount", "payee", "investmentTransactionInfo"},
	"ICTransactionSplit": {"ID", "lastModificationDate", "transaction", "amount", "category", "linkedSplit"},
}

// Categories maps iCompta category names to IDs and back. Names are compared
// NFC-normalised.
type Categories struct {
	byName map[string]string
	byID   map[string]string
}

// IDByName resolves a category name to its iCompta ID.
func (c Categories) IDByName(name string) (string, bool) {
	id, ok := c.byName[normalize(name)]
	return id, ok
}

// NameByID resolves an iCompta category ID to its name, "" when unknown.
func (c Categories) NameByID(id string) string { return c.byID[id] }

// Len is the number of categories.
func (c Categories) Len() int { return len(c.byID) }

// Snapshot is a consistent read of everything a preview needs.
type Snapshot struct {
	Candidates []Candidate
	Categories Categories
	State      string
}

// Store reads an iCompta database. It never writes.
type Store struct {
	db *sql.DB
}

// dsn builds a SQLite file URI so spaces and other characters in the path are
// escaped.
func dsn(path, query string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	u := url.URL{Scheme: "file", Path: abs}
	s := u.String()
	if query != "" {
		s += "?" + query
	}
	return s, nil
}

// OpenReadOnly opens the database read-only and checks the expected columns.
func OpenReadOnly(ctx context.Context, path string) (*Store, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	uri, err := dsn(path, "mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	if err := requireColumns(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close releases the connection.
func (s *Store) Close() error { return s.db.Close() }

func requireColumns(ctx context.Context, db *sql.DB) error {
	// Sorted so the first reported problem is deterministic.
	tables := make([]string, 0, len(requiredColumns))
	for table := range requiredColumns {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		cols := requiredColumns[table]
		rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
		if err != nil {
			return fmt.Errorf("inspect %s: %w", table, err)
		}
		have := map[string]bool{}
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull, pk int
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				_ = rows.Close()
				return fmt.Errorf("inspect %s: %w", table, err)
			}
			have[name] = true
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("inspect %s: %w", table, err)
		}
		_ = rows.Close()
		for _, c := range cols {
			if !have[c] {
				return fmt.Errorf("not an iCompta database this tool understands: %s.%s is missing", table, c)
			}
		}
	}
	return nil
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// readState summarises how much the data has changed: the split count and the
// latest modification date of splits and transactions. It is deliberately not
// the file size, which can change when iCompta merely opens the file.
func readState(ctx context.Context, q rowQuerier) (string, error) {
	var splits int
	var splitMod, txMod string
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(MAX(lastModificationDate),'') FROM ICTransactionSplit`).
		Scan(&splits, &splitMod); err != nil {
		return "", fmt.Errorf("read split state: %w", err)
	}
	if err := q.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(lastModificationDate),'') FROM ICTransaction`).Scan(&txMod); err != nil {
		return "", fmt.Errorf("read transaction state: %w", err)
	}
	return fmt.Sprintf("splits=%d;split_modified=%s;tx_modified=%s", splits, splitMod, txMod), nil
}

// State returns the current data-state marker.
func (s *Store) State(ctx context.Context) (string, error) {
	return readState(ctx, s.db)
}

// Snapshot reads categories, every split and the state marker in one read
// transaction.
func (s *Store) Snapshot(ctx context.Context) (Snapshot, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, fmt.Errorf("begin read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cats, err := loadCategories(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	state, err := readState(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT s.ID, t.date, t.name, COALESCE(t.payee,''), COALESCE(t.comment,''),
       COALESCE(NULLIF(s.amount,''), t.amount, ''), COALESCE(s.category,''),
       CASE WHEN COALESCE(t.investmentTransactionInfo,'') <> '' THEN 1 ELSE 0 END,
       CASE WHEN COALESCE(s.linkedSplit,'') <> '' THEN 1 ELSE 0 END
FROM ICTransactionSplit s
JOIN ICTransaction t ON t.ID = s."transaction"
ORDER BY t.date, s.ID`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read splits: %w", err)
	}
	defer rows.Close()

	var cands []Candidate
	for rows.Next() {
		var c Candidate
		var amount string
		var inv, linked int
		if err := rows.Scan(&c.SplitID, &c.Date, &c.Name, &c.Payee, &c.Comment,
			&amount, &c.CategoryID, &inv, &linked); err != nil {
			return Snapshot{}, fmt.Errorf("scan split: %w", err)
		}
		if strings.TrimSpace(amount) != "" {
			c.Amount, err = decimal.NewFromString(strings.TrimSpace(amount))
			if err != nil {
				return Snapshot{}, fmt.Errorf("split %s: amount %q: %w", c.SplitID, amount, err)
			}
		}
		c.CategoryName = cats.NameByID(c.CategoryID)
		c.IsInvestment = inv == 1
		c.IsLinked = linked == 1
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("read splits: %w", err)
	}
	return Snapshot{Candidates: cands, Categories: cats, State: state}, nil
}

type rowsQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func loadCategories(ctx context.Context, q rowsQuerier) (Categories, error) {
	rows, err := q.QueryContext(ctx, `SELECT ID, name FROM ICCategory`)
	if err != nil {
		return Categories{}, fmt.Errorf("read categories: %w", err)
	}
	defer rows.Close()
	cats := Categories{byName: map[string]string{}, byID: map[string]string{}}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return Categories{}, fmt.Errorf("scan category: %w", err)
		}
		cats.byID[id] = name
		cats.byName[normalize(name)] = id
	}
	return cats, rows.Err()
}
