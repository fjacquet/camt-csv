package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedBackups writes stale backup files for baseName with the given timestamps.
func seedBackups(t *testing.T, dir, baseName string, stamps ...string) {
	t.Helper()
	for _, stamp := range stamps {
		path := filepath.Join(dir, baseName+"."+stamp+".backup")
		require.NoError(t, os.WriteFile(path, []byte("stale\n"), 0o600))
	}
}

// TestCreateBackup_PrunesOldest verifies that creating a backup drops the
// oldest ones once the retention limit is exceeded.
func TestCreateBackup_PrunesOldest(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "creditors.yaml")
	require.NoError(t, os.WriteFile(file, []byte("a: b\n"), 0o600))

	stale := []string{"20260101_000001", "20260101_000002", "20260101_000003"}
	seedBackups(t, dir, "creditors.yaml", stale...)

	s := NewCategoryStore("", "", "")
	s.SetBackupConfig(true, "", "", 2)
	require.NoError(t, s.createBackup(file))

	backups, err := filepath.Glob(filepath.Join(dir, "creditors.yaml.*.backup"))
	require.NoError(t, err)
	assert.Len(t, backups, 2, "expected retention limit to cap backups")

	// The two oldest seeded backups go; the newest seeded one and the fresh
	// one (timestamped now, so lexically largest) stay.
	assert.NoFileExists(t, filepath.Join(dir, "creditors.yaml.20260101_000001.backup"))
	assert.NoFileExists(t, filepath.Join(dir, "creditors.yaml.20260101_000002.backup"))
	assert.FileExists(t, filepath.Join(dir, "creditors.yaml.20260101_000003.backup"))
}

// TestCreateBackup_LeavesOtherFilesAlone verifies pruning only touches backups
// of the file being saved.
func TestCreateBackup_LeavesOtherFilesAlone(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "creditors.yaml")
	require.NoError(t, os.WriteFile(file, []byte("a: b\n"), 0o600))

	seedBackups(t, dir, "creditors.yaml", "20260101_000001", "20260101_000002")
	seedBackups(t, dir, "debtors.yaml", "20260101_000001", "20260101_000002")

	s := NewCategoryStore("", "", "")
	s.SetBackupConfig(true, "", "", 1)
	require.NoError(t, s.createBackup(file))

	debtorBackups, err := filepath.Glob(filepath.Join(dir, "debtors.yaml.*.backup"))
	require.NoError(t, err)
	assert.Len(t, debtorBackups, 2, "debtor backups must not be pruned by a creditor save")

	creditorBackups, err := filepath.Glob(filepath.Join(dir, "creditors.yaml.*.backup"))
	require.NoError(t, err)
	assert.Len(t, creditorBackups, 1)
}

// TestCreateBackup_RetentionDisabled verifies that a non-positive retention
// keeps every backup.
func TestCreateBackup_RetentionDisabled(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "debtors.yaml")
	require.NoError(t, os.WriteFile(file, []byte("a: b\n"), 0o600))

	seedBackups(t, dir, "debtors.yaml", "20260101_000001", "20260101_000002", "20260101_000003")

	s := NewCategoryStore("", "", "")
	s.SetBackupConfig(true, "", "", 0)
	require.NoError(t, s.createBackup(file))

	backups, err := filepath.Glob(filepath.Join(dir, "debtors.yaml.*.backup"))
	require.NoError(t, err)
	assert.Len(t, backups, 4)
}

// TestSetBackupConfig_EmptyFormatKeepsDefault guards against every backup
// collapsing onto one filename when the config leaves the format unset.
func TestSetBackupConfig_EmptyFormatKeepsDefault(t *testing.T) {
	s := NewCategoryStore("", "", "")
	s.SetBackupConfig(true, "", "", DefaultBackupRetention)
	assert.Equal(t, "20060102_150405", s.backupTimestampFormat)
}

// TestNewCategoryStore_DefaultRetention pins the shipped default.
func TestNewCategoryStore_DefaultRetention(t *testing.T) {
	assert.Equal(t, DefaultBackupRetention, NewCategoryStore("", "", "").backupRetention)
}
