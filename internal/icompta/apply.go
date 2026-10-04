package icompta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"fjacquet/camt-csv/internal/logging"
)

// iComptaTimeFormat is how iCompta writes lastModificationDate (UTC).
const iComptaTimeFormat = "2006-01-02 15:04:05"

// ApplyOptions are the collaborators of Apply, injectable for tests.
type ApplyOptions struct {
	DBPath    string
	Now       func() time.Time
	IsRunning func() (bool, error)
}

// ApplyResult summarises a run.
type ApplyResult struct {
	Applied    int
	Skipped    int
	BackupPath string
}

// ICComptaRunning reports whether the iCompta app is running, by process name.
func ICComptaRunning() (bool, error) {
	err := exec.Command("pgrep", "-ix", "iCompta").Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check whether iCompta is running: %w", err)
}

// Apply writes the approved rows of a report into the database. It aborts
// before writing anything if iCompta is running, if the data changed since the
// preview, or if the database cannot be backed up consistently.
func Apply(ctx context.Context, rep Report, opts ApplyOptions, log logging.Logger) (ApplyResult, error) {
	running, err := opts.IsRunning()
	if err != nil {
		return ApplyResult{}, err
	}
	if running {
		return ApplyResult{}, errors.New("iCompta is running: quit it before applying, it would overwrite or lock the database")
	}
	if fi, err := os.Stat(opts.DBPath + "-wal"); err == nil && fi.Size() > 0 {
		return ApplyResult{}, fmt.Errorf("%s-wal is not empty: the database has uncommitted pages, a file copy would not be a consistent backup", opts.DBPath)
	}

	ro, err := OpenReadOnly(ctx, opts.DBPath)
	if err != nil {
		return ApplyResult{}, err
	}
	state, err := ro.State(ctx)
	_ = ro.Close()
	if err != nil {
		return ApplyResult{}, err
	}
	if state != rep.DBState {
		return ApplyResult{}, fmt.Errorf("the database changed since the preview (preview saw %q, now %q): run preview again", rep.DBState, state)
	}

	backup := fmt.Sprintf("%s.bak-%s", opts.DBPath, opts.Now().UTC().Format("20060102T150405Z"))
	if err := copyVerified(opts.DBPath, backup); err != nil {
		return ApplyResult{}, err
	}
	log.WithFields(logging.Field{Key: "backup", Value: backup}).Info("Database backed up")

	res := ApplyResult{BackupPath: backup}
	// _timeout is the validated shorthand for PRAGMA busy_timeout; _txlock=immediate
	// takes the write lock at BEGIN instead of failing late on the first UPDATE.
	uri, err := dsn(opts.DBPath, "_timeout=5000&_txlock=immediate")
	if err != nil {
		return res, err
	}
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return res, fmt.Errorf("open database for writing: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if err := integrityCheck(ctx, db); err != nil {
		return res, fmt.Errorf("before apply: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("begin write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cats, err := loadCategories(ctx, tx)
	if err != nil {
		return res, err
	}
	stamp := opts.Now().UTC().Format(iComptaTimeFormat)

	for _, row := range rep.Rows {
		skip := func(reason string) {
			res.Skipped++
			log.WithFields(
				logging.Field{Key: "split", Value: row.SplitID},
				logging.Field{Key: "name", Value: row.Name},
				logging.Field{Key: "reason", Value: reason},
			).Warn("Skipped split")
		}
		if row.Decision != ActionChange || !row.Apply {
			skip("not approved: decision " + string(row.Decision) + ", apply " + fmt.Sprint(row.Apply))
			continue
		}
		newID, ok := cats.IDByName(row.NewCategory)
		if !ok {
			skip("unknown category " + row.NewCategory)
			continue
		}
		var currentID sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT category FROM ICTransactionSplit WHERE ID = ?`, row.SplitID).Scan(&currentID)
		if errors.Is(err, sql.ErrNoRows) {
			skip("split not found")
			continue
		}
		if err != nil {
			return res, fmt.Errorf("read split %s: %w", row.SplitID, err)
		}
		if !SameCategory(cats.NameByID(currentID.String), row.OldCategory) {
			skip("category changed since preview")
			continue
		}
		r, err := tx.ExecContext(ctx,
			`UPDATE ICTransactionSplit SET category = ?, lastModificationDate = ? WHERE ID = ?`,
			newID, stamp, row.SplitID)
		if err != nil {
			return res, fmt.Errorf("update split %s: %w", row.SplitID, err)
		}
		if n, _ := r.RowsAffected(); n != 1 {
			return res, fmt.Errorf("update split %s: %d rows affected, expected 1", row.SplitID, n)
		}
		res.Applied++
		log.WithFields(
			logging.Field{Key: "split", Value: row.SplitID},
			logging.Field{Key: "name", Value: row.Name},
			logging.Field{Key: "old", Value: row.OldCategory},
			logging.Field{Key: "new", Value: row.NewCategory},
			logging.Field{Key: "tier", Value: row.Tier},
		).Info("Recategorized split")
	}

	if err := integrityCheck(ctx, tx); err != nil {
		return res, fmt.Errorf("after apply, rolled back: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return res, fmt.Errorf("commit: %w", err)
	}
	log.WithFields(
		logging.Field{Key: "applied", Value: res.Applied},
		logging.Field{Key: "skipped", Value: res.Skipped},
		logging.Field{Key: "backup", Value: backup},
	).Info("Recategorization applied")
	return res, nil
}

type integrityQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func integrityCheck(ctx context.Context, q integrityQuerier) error {
	var result string
	if err := q.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result); err != nil {
		return fmt.Errorf("integrity_check: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("integrity_check reported %q", result)
	}
	return nil
}

// copyVerified copies src to dst and checks the copy byte for byte by hash. On
// any failure the partial copy is removed, so a refused run leaves no backup.
func copyVerified(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	defer func() {
		_ = out.Close()
		if err != nil {
			_ = os.Remove(dst)
		}
	}()
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(out, h), in); err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	if err = out.Sync(); err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	copied, err := os.ReadFile(dst)
	if err != nil {
		return fmt.Errorf("verify backup: %w", err)
	}
	sum := sha256.Sum256(copied)
	if !bytes.Equal(sum[:], h.Sum(nil)) {
		err = errors.New("verify backup: copy does not match the source")
		return err
	}
	return nil
}
