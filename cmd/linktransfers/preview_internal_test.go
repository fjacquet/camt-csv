package linktransfers

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// fixtureDB is a minimal iCompta database: Fred owns A1 and A2; S1/S2 are a
// sure transfer.
func fixtureDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ic.cdb")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	for _, s := range []string{
		`CREATE TABLE ICCategory (ID TEXT UNIQUE NOT NULL, name TEXT NOT NULL)`,
		`CREATE TABLE ICAccount (ID TEXT UNIQUE NOT NULL, name TEXT, class TEXT, parent TEXT, currency TEXT)`,
		`CREATE TABLE ICTransaction (ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, account TEXT, date TEXT NOT NULL,
			name TEXT NOT NULL, comment TEXT, amount TEXT, payee TEXT, investmentTransactionInfo TEXT, status TEXT)`,
		`CREATE TABLE ICTransactionSplit (ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, "transaction" TEXT NOT NULL,
			amount TEXT, category TEXT, linkedSplit TEXT)`,
		`INSERT INTO ICCategory VALUES ('C-VIR','Virements')`,
		`INSERT INTO ICAccount VALUES ('F','Fred','ICAccountsGroup',NULL,NULL),
			('A1','Compte privé','ICAccount','F','CHF'),('A2','Selma','ICAccount','F','CHF')`,
		`INSERT INTO ICTransaction VALUES ('T1',NULL,'A1','2026-01-10','VERS SELMA','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus'),
			('T2',NULL,'A2','2026-01-11','DEPOT','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S1',NULL,'T1','-750','C-VIR',NULL),('S2',NULL,'T2','750','C-VIR',NULL)`,
	} {
		_, err := db.Exec(s)
		require.NoError(t, err, s)
	}
	return path
}

func TestRunPreview_WritesOneSurePair(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	require.NoError(t, runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "", "", logging.NewMockLogger()))

	b, err := os.ReadFile(out)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	require.Len(t, lines, 4, "magic, state, header, one pair")
	assert.Equal(t, "# camt-csv link-transfers report v1", lines[0])
	assert.True(t, strings.HasPrefix(lines[3], "S1,S2,2026-01-10,2026-01-11,1,"))
	assert.True(t, strings.HasSuffix(lines[3], ",sure,,,yes"))
	fi, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
}

func TestRunPreview_RefusesToOverwriteWithoutForce(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	require.NoError(t, os.WriteFile(out, []byte("reviewed"), 0o600))
	err := runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "", "", logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
	b, _ := os.ReadFile(out)
	assert.Equal(t, "reviewed", string(b))

	require.NoError(t, runPreviewWith(context.Background(), db, out, true, []string{"Fred"}, "", "", logging.NewMockLogger()))
}

func TestRunPreview_UnknownFolderLeavesNoFile(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	err := runPreviewWith(context.Background(), db, out, false, []string{"Nobody"}, "", "", logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `folder "Nobody" not found`)
	assert.NoFileExists(t, out)
}

func TestRunPreview_DateBounds(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	err := runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "10.01.2026", "", logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from")
	assert.NoFileExists(t, out)

	require.NoError(t, runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "2026-01-11", "", logging.NewMockLogger()))
	b, _ := os.ReadFile(out)
	assert.Len(t, strings.Split(strings.TrimSpace(string(b)), "\n"), 3, "S1 is before --from: no pair left")
}
