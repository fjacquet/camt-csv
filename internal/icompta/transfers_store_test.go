package icompta

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const transferFixtureSchema = `
CREATE TABLE ICCategory (ID TEXT UNIQUE NOT NULL, name TEXT NOT NULL);
CREATE TABLE ICAccount (ID TEXT UNIQUE NOT NULL, name TEXT, class TEXT, parent TEXT, currency TEXT);
CREATE TABLE ICTransaction (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, account TEXT, date TEXT NOT NULL, name TEXT NOT NULL,
	comment TEXT, amount TEXT, payee TEXT, investmentTransactionInfo TEXT, status TEXT);
CREATE TABLE ICTransactionSplit (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, "transaction" TEXT NOT NULL,
	amount TEXT, category TEXT, linkedSplit TEXT);
`

// newTransferFixtureDB: Fred owns BCV (A1) and Selma (A2), CHF, plus a EUR
// account (A3); Lydie's account (A4) is outside. S1/S2 are a sure transfer.
func newTransferFixtureDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ic link.cdb")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	for _, s := range []string{
		transferFixtureSchema,
		`INSERT INTO ICCategory VALUES ('C-VIR','Virements'),('C-ALI','Alimentation')`,
		`INSERT INTO ICAccount VALUES ('F','Fred','ICAccountsGroup',NULL,NULL),
			('A1','Compte privé','ICAccount','F','CHF'),('A2','Selma','ICAccount','F','CHF'),
			('A3','Epargne EUR','ICAccount','F','EUR'),
			('L','Lydie','ICAccountsGroup',NULL,NULL),('A4','Livret A','ICAccount','L','CHF'),
			('P','Florence','ICPerson','F',NULL)`,
		// Review focus 3: a date stored with a time must match like a plain date.
		`INSERT INTO ICTransaction VALUES ('T1',NULL,'A1','2026-01-10 00:00:00','VERS SELMA','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S1',NULL,'T1','-750','C-VIR',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T2',NULL,'A2','2026-01-11','DEPOT','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S2',NULL,'T2','750','C-VIR','')`,
		`INSERT INTO ICTransaction VALUES ('T3',NULL,'A1','2026-01-15','MIGROS','','-42.10','',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S3',NULL,'T3','','C-ALI',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T4',NULL,'A4','2026-01-11','LYDIE','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S4',NULL,'T4','750','C-VIR',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T5',NULL,'A1','2026-01-20','SPLIT','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S5a',NULL,'T5','-30','C-ALI',NULL),('S5b',NULL,'T5','-20','C-ALI',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T6',NULL,'A2','2026-01-20','PLANNED','',NULL,'',NULL,'ICTransactionStatus.PlannedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S6',NULL,'T6','50','C-VIR',NULL)`,
	} {
		_, err := db.Exec(s)
		require.NoError(t, err, s)
	}
	return path
}

func TestTransferSnapshot_ReadsAccountsAndLegs(t *testing.T) {
	s, err := OpenReadOnly(context.Background(), newTransferFixtureDB(t))
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	snap, err := s.TransferSnapshot(context.Background())
	require.NoError(t, err)
	assert.Len(t, snap.Accounts, 7)
	assert.Contains(t, snap.State, "splits=7")

	byID := map[string]Leg{}
	for _, l := range snap.Legs {
		byID[l.SplitID] = l
	}
	require.Len(t, byID, 7)
	s1 := byID["S1"]
	assert.Equal(t, "2026-01-10", s1.Date, "time part dropped")
	assert.Equal(t, "T1", s1.TxID)
	assert.Equal(t, "A1", s1.AccountID)
	assert.Equal(t, "Compte privé", s1.AccountName)
	assert.Equal(t, "CHF", s1.CurrencyID)
	assert.Equal(t, "Virements", s1.Category)
	assert.Equal(t, "-750", s1.Amount.String())
	assert.Equal(t, 1, s1.SplitCount)
	assert.False(t, s1.Linked)
	assert.Equal(t, "-42.1", byID["S3"].Amount.String(), "empty split amount falls back to the transaction")
	assert.Equal(t, 2, byID["S5a"].SplitCount)
	assert.Contains(t, byID["S6"].Status, "Planned")
}

func TestTransferSnapshot_EndToEndFindsTheOneSurePair(t *testing.T) {
	s, err := OpenReadOnly(context.Background(), newTransferFixtureDB(t))
	require.NoError(t, err)
	defer func() { _ = s.Close() }()
	snap, err := s.TransferSnapshot(context.Background())
	require.NoError(t, err)

	scope, err := ResolveScope(snap.Accounts, []string{"Fred"})
	require.NoError(t, err)
	pairs := MatchTransfers(EligibleLegs(snap.Legs, scope))
	require.Len(t, pairs, 1, "Lydie's S4 is out of scope, so S1/S2 stay unambiguous")
	assert.Equal(t, "S1", pairs[0].Debit.SplitID)
	assert.Equal(t, "S2", pairs[0].Credit.SplitID)
	assert.True(t, pairs[0].Sure)
}

func TestTransferSnapshot_MissingAccountTableIsClear(t *testing.T) {
	s, _ := openFixture(t) // the recategorize fixture has no ICAccount table
	_, err := s.TransferSnapshot(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ICAccount.ID is missing")
}
