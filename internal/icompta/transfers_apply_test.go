package icompta

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func linkOf(t *testing.T, path, splitID string) (linked, splitMod, txMod string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var l, sm, tm sql.NullString
	require.NoError(t, db.QueryRow(`
SELECT s.linkedSplit, s.lastModificationDate, t.lastModificationDate
FROM ICTransactionSplit s JOIN ICTransaction t ON t.ID = s."transaction" WHERE s.ID = ?`, splitID).Scan(&l, &sm, &tm))
	return l.String, sm.String, tm.String
}

func balances(t *testing.T, path string) map[string]string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	rows, err := db.Query(`
SELECT t.account, COALESCE(NULLIF(s.amount,''), t.amount, '0')
FROM ICTransactionSplit s JOIN ICTransaction t ON t.ID = s."transaction"`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	sum := map[string]decimal.Decimal{}
	for rows.Next() {
		var acct, amt string
		require.NoError(t, rows.Scan(&acct, &amt))
		sum[acct] = sum[acct].Add(decimal.RequireFromString(amt))
	}
	out := map[string]string{}
	for k, v := range sum {
		out[k] = v.StringFixed(2)
	}
	return out
}

func backups(t *testing.T, path string) []string {
	t.Helper()
	m, err := filepath.Glob(path + ".bak-*")
	require.NoError(t, err)
	return m
}

func row(debit, credit string, apply bool) TransferRow {
	return TransferRow{DebitSplitID: debit, CreditSplitID: credit, Confidence: "sure", Apply: apply}
}

func TestApplyLinks_LinksBothWaysAndKeepsBalances(t *testing.T) {
	path := newTransferFixtureDB(t)
	before := balances(t, path)
	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{
		row("S1", "S2", true),
		row("S3", "S4", false), // not approved: untouched
	}}

	res, err := ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Applied)
	assert.Equal(t, 0, res.Skipped)
	assert.NotEmpty(t, res.BackupPath)
	assert.FileExists(t, res.BackupPath)

	stamp := fixedNow.UTC().Format(iComptaTimeFormat)
	l, sm, tm := linkOf(t, path, "S1")
	assert.Equal(t, []string{"S2", stamp, stamp}, []string{l, sm, tm})
	l, sm, tm = linkOf(t, path, "S2")
	assert.Equal(t, []string{"S1", stamp, stamp}, []string{l, sm, tm})
	l, _, _ = linkOf(t, path, "S3")
	assert.Empty(t, l)
	assert.Equal(t, before, balances(t, path), "linking never changes an amount")
}

// Review focus 2: the first apply moves the state marker.
func TestApplyLinks_SecondApplyIsRefused(t *testing.T) {
	path := newTransferFixtureDB(t)
	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{row("S1", "S2", true)}}
	_, err := ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)

	_, err = ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run preview again")
}

func TestApplyLinks_SkipsWhatNoLongerFits(t *testing.T) {
	path := newTransferFixtureDB(t)
	// S2 already linked elsewhere before the preview was taken.
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE ICTransactionSplit SET linkedSplit='OTHER' WHERE ID='S2'`)
	require.NoError(t, err)
	_ = db.Close()

	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{
		row("S1", "S2", true),   // S2 already linked
		row("S3", "S6", true),   // -42.10 vs 50: amounts do not match
		row("GONE", "S4", true), // debit split missing
	}}

	log := logging.NewMockLogger()
	res, err := ApplyLinks(context.Background(), rep, opts(path), log)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Applied)
	assert.Equal(t, 3, res.Skipped)
	assert.True(t, log.HasEntry("WARN", "Skipped pair"))
	l, _, _ := linkOf(t, path, "S1")
	assert.Empty(t, l)
}

func TestApplyLinks_SplitInTwoApprovedRowsRejectsEverything(t *testing.T) {
	path := newTransferFixtureDB(t)
	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{
		row("S1", "S2", true),
		row("S1", "S4", true),
	}}
	_, err := ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "S1")
	assert.Empty(t, backups(t, path), "rejected before any backup or write")
	l, _, _ := linkOf(t, path, "S2")
	assert.Empty(t, l)
}

func TestApplyLinks_RefusesWhileICComptaRuns(t *testing.T) {
	path := newTransferFixtureDB(t)
	o := opts(path)
	o.IsRunning = func() (bool, error) { return true, nil }
	_, err := ApplyLinks(context.Background(), TransferReport{DBState: previewState(t, path)}, o, logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "iCompta is running")
	assert.Empty(t, backups(t, path))
}
