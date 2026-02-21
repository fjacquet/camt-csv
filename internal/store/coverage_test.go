package store

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Tests for createBackup branches not covered by existing tests.

func TestCreateBackup_FileDoesNotExist(t *testing.T) {
	s := NewCategoryStore("", "", "")

	err := s.createBackup("/nonexistent/path/file.yaml")
	assert.NoError(t, err, "should not error when original file does not exist")
}

func TestCreateBackup_BackupDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.yaml")
	require.NoError(t, os.WriteFile(file, []byte("data"), models.PermissionNonSecretFile))

	s := NewCategoryStore("", "", "")
	s.SetBackupConfig(false, "", "20060102_150405")

	err := s.createBackup(file)
	assert.NoError(t, err)

	backups, _ := filepath.Glob(filepath.Join(tmpDir, "*.backup"))
	assert.Empty(t, backups, "no backup should be created when disabled")
}

func TestCreateBackup_CannotCreateBackupDir(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.yaml")
	require.NoError(t, os.WriteFile(file, []byte("data"), models.PermissionNonSecretFile))

	// Use a path under a file (not a directory) to force MkdirAll failure
	blockingFile := filepath.Join(tmpDir, "blocker")
	require.NoError(t, os.WriteFile(blockingFile, []byte("x"), 0600))

	s := NewCategoryStore("", "", "")
	s.SetBackupConfig(true, filepath.Join(blockingFile, "subdir"), "20060102_150405")

	err := s.createBackup(file)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating backup directory")
}

func TestCreateBackup_CannotOpenSourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.yaml")
	require.NoError(t, os.WriteFile(file, []byte("data"), models.PermissionNonSecretFile))

	s := NewCategoryStore("", "", "")

	// Make file unreadable after the Stat check passes
	require.NoError(t, os.Chmod(file, 0000))
	defer func() { _ = os.Chmod(file, 0600) }()

	err := s.createBackup(file)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error opening file for backup")
}

func TestCreateBackup_CannotCreateBackupFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.yaml")
	require.NoError(t, os.WriteFile(file, []byte("data"), models.PermissionNonSecretFile))

	// Create backup directory as read-only so backup file creation fails
	backupDir := filepath.Join(tmpDir, "backups")
	require.NoError(t, os.MkdirAll(backupDir, 0750))
	require.NoError(t, os.Chmod(backupDir, 0444))    // #nosec G302 -- intentionally restrictive for testing
	defer func() { _ = os.Chmod(backupDir, 0750) }() // #nosec G302 -- restore for cleanup

	s := NewCategoryStore("", "", "")
	s.SetBackupConfig(true, backupDir, "20060102_150405")

	err := s.createBackup(file)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating backup file")
}

func TestCreateBackup_SameDirectoryAsOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "creditors.yaml")
	require.NoError(t, os.WriteFile(file, []byte("original"), models.PermissionNonSecretFile))

	s := NewCategoryStore("", "", "")
	// backupDirectory is empty, so backup goes to same dir as original
	s.SetBackupConfig(true, "", "20060102_150405")

	err := s.createBackup(file)
	assert.NoError(t, err)

	backups, _ := filepath.Glob(filepath.Join(tmpDir, "creditors.yaml.*.backup"))
	assert.Len(t, backups, 1)

	data, err := os.ReadFile(backups[0])
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

// Tests for SaveCreditorMappings branches.

func TestSaveCreditorMappings_AbsolutePathWhenFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "newcreditors.yaml")

	s := NewCategoryStore("", absPath, "")
	s.SetBackupConfig(false, "", "")

	mappings := map[string]string{"Test": "Cat"}
	err := s.SaveCreditorMappings(mappings)
	assert.NoError(t, err)

	data, err := os.ReadFile(absPath)
	require.NoError(t, err)

	var loaded map[string]string
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	assert.Equal(t, "Cat", loaded["Test"])
}

func TestSaveCreditorMappings_ReadOnlyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0750))

	file := filepath.Join(readOnlyDir, "creditors.yaml")
	require.NoError(t, os.WriteFile(file, []byte("Test: Cat"), models.PermissionNonSecretFile))

	// Make directory read-only so backup file creation fails
	require.NoError(t, os.Chmod(readOnlyDir, 0555))    // #nosec G302 -- restrictive for testing
	defer func() { _ = os.Chmod(readOnlyDir, 0750) }() // #nosec G302 -- restore for cleanup

	s := NewCategoryStore("", file, "")

	err := s.SaveCreditorMappings(map[string]string{"New": "Entry"})
	assert.Error(t, err, "should fail when backup cannot be written")
}

// Tests for SaveDebtorMappings branches.

func TestSaveDebtorMappings_AbsolutePathWhenFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	absPath := filepath.Join(tmpDir, "newdebtors.yaml")

	s := NewCategoryStore("", "", absPath)
	s.SetBackupConfig(false, "", "")

	mappings := map[string]string{"Employer": "Salary"}
	err := s.SaveDebtorMappings(mappings)
	assert.NoError(t, err)

	data, err := os.ReadFile(absPath)
	require.NoError(t, err)

	var loaded map[string]string
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	assert.Equal(t, "Salary", loaded["Employer"])
}

func TestSaveDebtorMappings_ReadOnlyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0750))

	file := filepath.Join(readOnlyDir, "debtors.yaml")
	require.NoError(t, os.WriteFile(file, []byte("Test: Cat"), models.PermissionNonSecretFile))

	require.NoError(t, os.Chmod(readOnlyDir, 0555))    // #nosec G302 -- restrictive for testing
	defer func() { _ = os.Chmod(readOnlyDir, 0750) }() // #nosec G302 -- restore for cleanup

	s := NewCategoryStore("", "", file)

	err := s.SaveDebtorMappings(map[string]string{"New": "Entry"})
	assert.Error(t, err, "should fail when backup cannot be written")
}

func TestSaveDebtorMappings_BackupCreatedBeforeSave(t *testing.T) {
	tmpDir := t.TempDir()
	debtorsFile := filepath.Join(tmpDir, "debtors.yaml")

	initialMappings := map[string]string{"Original": "Category"}
	data, err := yaml.Marshal(initialMappings)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(debtorsFile, data, models.PermissionNonSecretFile))

	s := NewCategoryStore("", "", debtorsFile)

	newMappings := map[string]string{"Original": "Category", "New": "Item"}
	err = s.SaveDebtorMappings(newMappings)
	assert.NoError(t, err)

	backups, _ := filepath.Glob(filepath.Join(tmpDir, "debtors.yaml.*.backup"))
	assert.Len(t, backups, 1, "should create one backup file")

	backupData, err := os.ReadFile(backups[0])
	require.NoError(t, err)
	var backupMappings map[string]string
	require.NoError(t, yaml.Unmarshal(backupData, &backupMappings))
	assert.Equal(t, initialMappings, backupMappings)
}

// Tests for FindConfigFile branches not covered by existing tests.

func TestFindConfigFile_AbsolutePathNotFound(t *testing.T) {
	s := NewCategoryStore("", "", "")

	_, err := s.FindConfigFile("/nonexistent/absolute/path.yaml")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestFindConfigFile_RelativePathNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	s := NewCategoryStore("", "", "")

	_, err = s.FindConfigFile("definitely_not_here.yaml")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// Tests for SetBackupConfig.

func TestSetBackupConfig(t *testing.T) {
	s := NewCategoryStore("", "", "")

	// Verify defaults
	assert.True(t, s.backupEnabled)
	assert.Empty(t, s.backupDirectory)
	assert.Equal(t, "20060102_150405", s.backupTimestampFormat)

	// Override
	s.SetBackupConfig(false, "/custom/backups", "2006-01-02")
	assert.False(t, s.backupEnabled)
	assert.Equal(t, "/custom/backups", s.backupDirectory)
	assert.Equal(t, "2006-01-02", s.backupTimestampFormat)
}

// Tests for LoadCategories edge cases.

func TestLoadCategories_MalformedBothFormats(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "categories.yaml")

	// Content that fails both structured and list YAML unmarshaling:
	// a YAML mapping with a colon-in-value that breaks parsing
	require.NoError(t, os.WriteFile(file, []byte("{{{{not: valid: yaml: [unclosed"), models.PermissionConfigFile))

	s := NewCategoryStore(file, "", "")

	_, err := s.LoadCategories()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing categories file")
}

func TestLoadCategories_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "categories.yaml")

	require.NoError(t, os.WriteFile(file, []byte(""), models.PermissionConfigFile))

	s := NewCategoryStore(file, "", "")

	cats, err := s.LoadCategories()
	assert.NoError(t, err)
	assert.Empty(t, cats)
}

// Test resolveConfigFile with relative path that exists.

func TestResolveConfigFile_RelativePathExists(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, os.WriteFile("found.yaml", []byte("test"), 0600))

	s := NewCategoryStore("", "", "")

	resolved, err := s.resolveConfigFile("found.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "found.yaml", resolved)
}

// Test that SaveCreditorMappings creates parent directory.

func TestSaveCreditorMappings_CreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "creditors.yaml")

	s := NewCategoryStore("", nestedPath, "")
	s.SetBackupConfig(false, "", "")

	err := s.SaveCreditorMappings(map[string]string{"A": "B"})
	assert.NoError(t, err)

	_, err = os.Stat(nestedPath)
	assert.NoError(t, err)
}

func TestSaveDebtorMappings_CreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "debtors.yaml")

	s := NewCategoryStore("", "", nestedPath)
	s.SetBackupConfig(false, "", "")

	err := s.SaveDebtorMappings(map[string]string{"A": "B"})
	assert.NoError(t, err)

	_, err = os.Stat(nestedPath)
	assert.NoError(t, err)
}
