package icompta

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const fixtureSchema = `
CREATE TABLE ICCategory (ID TEXT UNIQUE NOT NULL, name TEXT NOT NULL);
CREATE TABLE ICTransaction (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, date TEXT NOT NULL, name TEXT NOT NULL,
	comment TEXT, amount TEXT, payee TEXT, investmentTransactionInfo TEXT);
CREATE TABLE ICTransactionSplit (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, "transaction" TEXT NOT NULL,
	amount TEXT, category TEXT, linkedSplit TEXT);
`

// newFixtureDB builds a database with the columns recategorize uses and a small
// data set covering every case the reader must handle.
func newFixtureDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ic test.cdb") // a space: the DSN must escape it
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	stmts := []string{
		fixtureSchema,
		`INSERT INTO ICCategory VALUES ('C-ALI','Alimentation'),('C-DIV','Divers'),('C-NC','Non Classe` + "́" + `')`,
		// T1: categorised, debit
		`INSERT INTO ICTransaction VALUES ('T1','2026-01-01','2026-01-02','COOP CITY','', '-10.50','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S1',NULL,'T1','-10.50','C-ALI','')`,
		// T2: no category, payee set
		`INSERT INTO ICTransaction VALUES ('T2',NULL,'2026-01-03','VIREMENT','ref 7','200','Employeur SA',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S2',NULL,'T2','200',NULL,NULL)`,
		// T3: investment
		`INSERT INTO ICTransaction VALUES ('T3',NULL,'2026-01-04','BUY','', '-500','','info')`,
		`INSERT INTO ICTransactionSplit VALUES ('S3',NULL,'T3','-500',NULL,NULL)`,
		// T4: linked transfer leg
		`INSERT INTO ICTransaction VALUES ('T4',NULL,'2026-01-05','TRANSFER','', '-50','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S4',NULL,'T4','-50',NULL,'S9')`,
		// T5: split amount empty, falls back to the transaction amount
		`INSERT INTO ICTransaction VALUES ('T5',NULL,'2026-01-06','SPLITLESS','', '-7.25','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S5',NULL,'T5','',NULL,NULL)`,
		// T6: both amounts empty
		`INSERT INTO ICTransaction VALUES ('T6',NULL,'2026-01-07','NOAMOUNT','', NULL,'',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S6',NULL,'T6',NULL,'C-DIV',NULL)`,
		// T7: category ID that no longer exists
		`INSERT INTO ICTransaction VALUES ('T7',NULL,'2026-01-08','DANGLING','', '-1','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S7',NULL,'T7','-1','C-GONE',NULL)`,
	}
	for _, s := range stmts {
		_, err := db.Exec(s)
		require.NoError(t, err, s)
	}
	return path
}

func openFixture(t *testing.T) (*Store, string) {
	t.Helper()
	path := newFixtureDB(t)
	s, err := OpenReadOnly(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func TestStore_Snapshot(t *testing.T) {
	s, _ := openFixture(t)
	snap, err := s.Snapshot(context.Background())
	require.NoError(t, err)

	byID := map[string]Candidate{}
	for _, c := range snap.Candidates {
		byID[c.SplitID] = c
	}
	require.Len(t, byID, 7)

	s1 := byID["S1"]
	assert.Equal(t, "COOP CITY", s1.Name)
	assert.Equal(t, "Alimentation", s1.CategoryName)
	assert.True(t, s1.Amount.Equal(decimal.RequireFromString("-10.50")))
	assert.True(t, s1.IsDebtor())

	s2 := byID["S2"]
	assert.Equal(t, "Employeur SA", s2.PartyName())
	assert.Equal(t, "ref 7", s2.Comment)
	assert.Empty(t, s2.CategoryID)

	assert.True(t, byID["S3"].IsInvestment)
	assert.True(t, byID["S4"].IsLinked)

	// Review focus 1: empty split amount falls back, then falls to zero.
	assert.True(t, byID["S5"].Amount.Equal(decimal.RequireFromString("-7.25")))
	assert.True(t, byID["S6"].Amount.IsZero())
	assert.Equal(t, "Divers", byID["S6"].CategoryName)

	// A dangling category ID resolves to no name rather than failing.
	assert.Equal(t, "C-GONE", byID["S7"].CategoryID)
	assert.Empty(t, byID["S7"].CategoryName)
}

// Review focus 2: the database holds a decomposed accent; the lookup is by
// normalised name.
func TestCategories_ResolveDecomposedNames(t *testing.T) {
	s, _ := openFixture(t)
	snap, err := s.Snapshot(context.Background())
	require.NoError(t, err)

	id, ok := snap.Categories.IDByName("Non Classé") // composed
	assert.True(t, ok)
	assert.Equal(t, "C-NC", id)

	_, ok = snap.Categories.IDByName("Inexistante")
	assert.False(t, ok)
	assert.Equal(t, 3, snap.Categories.Len())
}

func TestStore_StateChangesWhenDataChanges(t *testing.T) {
	s, path := openFixture(t)
	before, err := s.State(context.Background())
	require.NoError(t, err)
	assert.Contains(t, before, "splits=7")

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`UPDATE ICTransactionSplit SET lastModificationDate='2026-10-04 19:00:00' WHERE ID='S1'`)
	require.NoError(t, err)

	after, err := s.State(context.Background())
	require.NoError(t, err)
	assert.NotEqual(t, before, after)
}

func TestOpenReadOnly_Errors(t *testing.T) {
	_, err := OpenReadOnly(context.Background(), filepath.Join(t.TempDir(), "missing.cdb"))
	require.Error(t, err)

	// A database without the expected columns fails with a clear message.
	path := filepath.Join(t.TempDir(), "other.cdb")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE ICCategory (ID TEXT)`)
	require.NoError(t, err)
	_ = db.Close()

	_, err = OpenReadOnly(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ICCategory")
}

func TestStore_IsReadOnly(t *testing.T) {
	s, _ := openFixture(t)
	_, err := s.db.Exec(`UPDATE ICTransactionSplit SET category='C-DIV' WHERE ID='S1'`)
	require.Error(t, err, "the preview connection must not be able to write")
}
