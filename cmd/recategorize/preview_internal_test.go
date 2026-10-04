package recategorize

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/icompta"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// newDB builds a minimal iCompta database with n uncategorised splits.
func newDB(t *testing.T, n int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ic.cdb")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	stmts := []string{
		`CREATE TABLE ICCategory (ID TEXT UNIQUE NOT NULL, name TEXT NOT NULL)`,
		`CREATE TABLE ICTransaction (ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, date TEXT NOT NULL,
			name TEXT NOT NULL, comment TEXT, amount TEXT, payee TEXT, investmentTransactionInfo TEXT)`,
		`CREATE TABLE ICTransactionSplit (ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT,
			"transaction" TEXT NOT NULL, amount TEXT, category TEXT, linkedSplit TEXT)`,
		`INSERT INTO ICCategory VALUES ('C1','Courses')`,
	}
	for i := 0; i < n; i++ {
		id := string(rune('A' + i))
		stmts = append(stmts,
			`INSERT INTO ICTransaction VALUES ('T`+id+`',NULL,'2026-01-0`+string(rune('1'+i))+`','SHOP `+id+`','','-5','',NULL)`,
			`INSERT INTO ICTransactionSplit VALUES ('S`+id+`',NULL,'T`+id+`','-5',NULL,NULL)`)
	}
	for _, s := range stmts {
		_, err := db.Exec(s)
		require.NoError(t, err, s)
	}
	return path
}

// cancellingClassifier answers "Courses" and cancels the run after the first call.
type cancellingClassifier struct {
	cancel context.CancelFunc
	calls  int
}

func (c *cancellingClassifier) Full(context.Context, categorizer.Transaction) (models.Category, error) {
	c.calls++
	if c.cancel != nil {
		c.cancel()
	}
	return models.Category{Name: "Courses", Source: icompta.TierAI}, nil
}

func (c *cancellingClassifier) Local(context.Context, categorizer.Transaction) (models.Category, bool) {
	return models.Category{}, false
}

// Review finding: Ctrl-C after tens of minutes of rate-limited AI calls must
// leave the rows decided so far, not nothing.
func TestRunPreview_CancelledWritesPartialReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cl := &cancellingClassifier{cancel: cancel}
	out := filepath.Join(t.TempDir(), "report.csv")

	err := runPreviewWith(ctx, newDB(t, 3), out, false, cl, logging.NewMockLogger())
	require.NoError(t, err)

	f, err := os.Open(out)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rep, err := icompta.ReadReport(f)
	require.NoError(t, err)
	assert.Len(t, rep.Rows, 1, "only the split decided before the cancel is in the report")
	assert.Equal(t, 1, cl.calls)
}

// Review finding: a reviewed report must not be silently overwritten.
func TestRunPreview_RefusesToOverwriteWithoutForce(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.csv")
	require.NoError(t, os.WriteFile(out, []byte("hand edits"), 0o600))

	err := runPreviewWith(context.Background(), newDB(t, 1), out, false, &cancellingClassifier{}, logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
	got, _ := os.ReadFile(out)
	assert.Equal(t, "hand edits", string(got))

	require.NoError(t, runPreviewWith(context.Background(), newDB(t, 1), out, true, &cancellingClassifier{}, logging.NewMockLogger()))
	got, _ = os.ReadFile(out)
	assert.Contains(t, string(got), "camt-csv recategorize report")
}

// Review finding: a bad output path must fail before the slow classification.
func TestRunPreview_BadOutputPathFailsBeforeAnyWork(t *testing.T) {
	cl := &cancellingClassifier{}
	out := filepath.Join(t.TempDir(), "missing-dir", "report.csv")

	err := runPreviewWith(context.Background(), newDB(t, 2), out, false, cl, logging.NewMockLogger())
	require.Error(t, err)
	assert.Zero(t, cl.calls, "no split may be classified when the report cannot be written")
}

// A failure before the report is filled must not leave an empty file that the
// next run would then refuse to overwrite.
func TestRunPreview_FailedRunLeavesNoEmptyReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.csv")
	err := runPreviewWith(context.Background(), filepath.Join(t.TempDir(), "nope.cdb"), out, false,
		&cancellingClassifier{}, logging.NewMockLogger())
	require.Error(t, err)
	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr))
}
