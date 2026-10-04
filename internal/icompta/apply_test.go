package icompta

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2026, 10, 4, 21, 30, 5, 0, time.FixedZone("CEST", 2*3600))

func notRunning() (bool, error) { return false, nil }

func categoryOf(t *testing.T, path, splitID string) (cat, modified string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var c, m sql.NullString
	require.NoError(t, db.QueryRow(
		`SELECT category, lastModificationDate FROM ICTransactionSplit WHERE ID=?`, splitID).Scan(&c, &m))
	return c.String, m.String
}

// previewState returns the state marker the fixture currently has.
func previewState(t *testing.T, path string) string {
	t.Helper()
	s, err := OpenReadOnly(context.Background(), path)
	require.NoError(t, err)
	defer func() { _ = s.Close() }()
	st, err := s.State(context.Background())
	require.NoError(t, err)
	return st
}

func opts(path string) ApplyOptions {
	return ApplyOptions{DBPath: path, Now: func() time.Time { return fixedNow }, IsRunning: notRunning}
}

func TestApply_WritesApprovedRowsOnly(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{
		DBState: previewState(t, path),
		Rows: []Row{
			{SplitID: "S2", OldCategory: "", NewCategory: "Alimentation", Decision: ActionChange, Apply: true},
			{SplitID: "S6", OldCategory: "Divers", NewCategory: "Alimentation", Decision: ActionChange, Apply: false},
			{SplitID: "S1", OldCategory: "Alimentation", NewCategory: "Divers", Decision: ActionKeep, Apply: true},
		},
	}
	log := logging.NewMockLogger()

	res, err := Apply(context.Background(), rep, opts(path), log)
	require.NoError(t, err)

	assert.Equal(t, 1, res.Applied)
	assert.Equal(t, 2, res.Skipped)

	cat, mod := categoryOf(t, path, "S2")
	assert.Equal(t, "C-ALI", cat)
	assert.Equal(t, "2026-10-04 19:30:05", mod, "lastModificationDate is written in UTC")

	cat, _ = categoryOf(t, path, "S6")
	assert.Equal(t, "C-DIV", cat, "apply=no must not change the split")
	cat, _ = categoryOf(t, path, "S1")
	assert.Equal(t, "C-ALI", cat, "a keep row must never be written, even with apply=yes")

	assert.True(t, log.HasEntry("INFO", "Recategorized split"))
	assert.True(t, log.HasEntry("WARN", "Skipped split"))
}

func TestApply_BacksUpFirst(t *testing.T) {
	_, path := openFixture(t)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	res, err := Apply(context.Background(), Report{DBState: previewState(t, path)}, opts(path), logging.NewMockLogger())
	require.NoError(t, err)

	assert.Equal(t, path+".bak-20261004T193005Z", res.BackupPath)
	backup, err := os.ReadFile(res.BackupPath)
	require.NoError(t, err)
	assert.Equal(t, before, backup)
}

func TestApply_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, path string, rep *Report, o *ApplyOptions)
		wantErr string
	}{
		{"iCompta running", func(_ *testing.T, _ string, _ *Report, o *ApplyOptions) {
			o.IsRunning = func() (bool, error) { return true, nil }
		}, "iCompta is running"},
		{"running check fails", func(_ *testing.T, _ string, _ *Report, o *ApplyOptions) {
			o.IsRunning = func() (bool, error) { return false, errors.New("pgrep exploded") }
		}, "pgrep exploded"},
		{"database changed since preview", func(t *testing.T, path string, _ *Report, _ *ApplyOptions) {
			db, err := sql.Open("sqlite", path)
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			_, err = db.Exec(`UPDATE ICTransactionSplit SET lastModificationDate='2026-10-04 20:00:00' WHERE ID='S1'`)
			require.NoError(t, err)
		}, "changed since the preview"},
		{"wal file present", func(t *testing.T, path string, _ *Report, _ *ApplyOptions) {
			require.NoError(t, os.WriteFile(path+"-wal", []byte("x"), 0o600))
		}, "-wal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, path := openFixture(t)
			rep := Report{DBState: previewState(t, path), Rows: []Row{
				{SplitID: "S2", NewCategory: "Alimentation", Decision: ActionChange, Apply: true}}}
			o := opts(path)
			tt.mutate(t, path, &rep, &o)

			_, err := Apply(context.Background(), rep, o, logging.NewMockLogger())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)

			cat, _ := categoryOf(t, path, "S2")
			assert.Empty(t, cat, "a refused apply must write nothing")
			matches, _ := filepath.Glob(path + ".bak-*")
			assert.Empty(t, matches, "a refused apply must not leave a backup behind")
		})
	}
}

func TestApply_SkipsStaleAndUnknownRows(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{DBState: previewState(t, path), Rows: []Row{
		// S1 is Alimentation now, the report believes it was Divers.
		{SplitID: "S1", OldCategory: "Divers", NewCategory: "Alimentation", Decision: ActionChange, Apply: true},
		// Category that does not exist.
		{SplitID: "S2", OldCategory: "", NewCategory: "Brand New", Decision: ActionChange, Apply: true},
		// Split that does not exist.
		{SplitID: "NOPE", OldCategory: "", NewCategory: "Alimentation", Decision: ActionChange, Apply: true},
	}}
	log := logging.NewMockLogger()

	res, err := Apply(context.Background(), rep, opts(path), log)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Applied)
	assert.Equal(t, 3, res.Skipped)
	assert.Len(t, log.GetEntriesByLevel("WARN"), 3)
}

// Review focus 4: running the same report twice is harmless.
func TestApply_SecondRunChangesNothing(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{DBState: previewState(t, path), Rows: []Row{
		{SplitID: "S2", OldCategory: "", NewCategory: "Alimentation", Decision: ActionChange, Apply: true}}}

	first, err := Apply(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, first.Applied)

	// The first run changed the data, so the state marker moved: refuse.
	_, err = Apply(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "changed since the preview")

	// Even with a refreshed state, the stale old_category keeps it a no-op.
	rep.DBState = previewState(t, path)
	log := logging.NewMockLogger()
	later := opts(path)
	later.Now = func() time.Time { return fixedNow.Add(time.Minute) } // a new backup name
	second, err := Apply(context.Background(), rep, later, log)
	require.NoError(t, err)
	assert.Equal(t, 0, second.Applied)
	assert.True(t, log.HasEntry("WARN", "Skipped split"))
	cat, _ := categoryOf(t, path, "S2")
	assert.Equal(t, "C-ALI", cat)
}

func TestApply_ResolvesDecomposedCategoryName(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{DBState: previewState(t, path), Rows: []Row{
		// The database stores "Non Classe" + combining accent; the report carries
		// the composed form. Both must resolve to the same category.
		{SplitID: "S6", OldCategory: "Divers", NewCategory: "Non Classé", Decision: ActionChange, Apply: true}}}

	res, err := Apply(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Applied)
	cat, _ := categoryOf(t, path, "S6")
	assert.Equal(t, "C-NC", cat)
}
