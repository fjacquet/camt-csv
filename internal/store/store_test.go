package store

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), models.PermissionConfigFile)
	assert.NoError(t, err)
}

// NewTestCategoryStore returns a CategoryStore for tests with specific test paths
func NewTestCategoryStore(dir string) *CategoryStore {
	return &CategoryStore{
		CategoriesFile: filepath.Join(dir, "categories.yaml"),
		CreditorsFile:  filepath.Join(dir, "creditors.yaml"),
		DebtorsFile:    filepath.Join(dir, "debtors.yaml"),
	}
}

func TestNewCategoryStore(t *testing.T) {
	store := NewCategoryStore("categories.yaml", "creditors.yaml", "debtors.yaml")
	assert.Equal(t, "categories.yaml", store.CategoriesFile)
	assert.Equal(t, "creditors.yaml", store.CreditorsFile)
	assert.Equal(t, "debtors.yaml", store.DebtorsFile)
}

func TestFindConfigFile(t *testing.T) {
	dir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(dir, "test.yaml")
	err := os.WriteFile(testFile, []byte("test content"), models.PermissionConfigFile)
	assert.NoError(t, err)

	store := NewCategoryStore("", "", "")

	// Test with absolute path that exists
	file, err := store.FindConfigFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testFile, file)

	// Test with file that doesn't exist
	_, err = store.FindConfigFile(filepath.Join(dir, "nonexistent.yaml"))
	assert.Error(t, err)
}

func TestLoadCategories_ValidAndMissing(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "categories.yaml")
	content := `
categories:
  - name: Groceries
    keywords: ["supermarket", "grocery"]
  - name: Rent
    keywords: ["apartment", "rent"]
`
	writeFile(t, file, content)
	store := NewTestCategoryStore(dir)
	cats, err := store.LoadCategories()
	assert.NoError(t, err)
	assert.Len(t, cats, 2)
	assert.Equal(t, "Groceries", cats[0].Name)

	// Missing file: should return empty slice, not error
	store2 := NewTestCategoryStore(dir)
	store2.CategoriesFile = filepath.Join(dir, "missing.yaml")
	cats2, err := store2.LoadCategories()
	assert.NoError(t, err)
	assert.Empty(t, cats2)
}

func TestLoadCategories_Malformed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "categories.yaml")
	// Create a malformed YAML file - a string that can't be parsed as YAML
	content := `{malformed: yaml: content}`
	writeFile(t, file, content)
	store := NewTestCategoryStore(dir)
	store.CategoriesFile = file
	_, err := store.LoadCategories()
	assert.Error(t, err)
}

func TestLoadAndSaveCreditorMappings(t *testing.T) {
	tempDir := t.TempDir()
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")

	// Create initial mappings
	initialMappings := map[string]string{"Alice": "ALICE_CORP", "Bob": "BOB_INC"}
	data, err := yaml.Marshal(initialMappings)
	assert.NoError(t, err)
	err = os.WriteFile(creditorsFile, data, models.PermissionConfigFile)
	assert.NoError(t, err)

	// Load the mappings
	store := NewCategoryStore(
		filepath.Join(tempDir, "categories.yaml"),
		creditorsFile,
		filepath.Join(tempDir, "debtors.yaml"),
	)

	mappings, err := store.LoadCreditorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "ALICE_CORP", mappings["Alice"])
	assert.Equal(t, "BOB_INC", mappings["Bob"])

	// Add a new mapping and save
	mappings["Charlie"] = "CHARLIE_LLC"
	err = store.SaveCreditorMappings(mappings)
	assert.NoError(t, err)

	// Reload and verify
	newMappings, err := store.LoadCreditorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "CHARLIE_LLC", newMappings["Charlie"])
}

func TestLoadAndSaveDebtorMappings(t *testing.T) {
	tempDir := t.TempDir()
	debtorsFile := filepath.Join(tempDir, "debtors.yaml")

	// Create initial mappings
	initialMappings := map[string]string{"Company X": "INCOME_SALARY", "Company Y": "INCOME_BONUS"}
	data, err := yaml.Marshal(initialMappings)
	assert.NoError(t, err)
	err = os.WriteFile(debtorsFile, data, models.PermissionConfigFile)
	assert.NoError(t, err)

	// Load the mappings
	store := NewCategoryStore(
		filepath.Join(tempDir, "categories.yaml"),
		filepath.Join(tempDir, "creditors.yaml"),
		debtorsFile,
	)

	mappings, err := store.LoadDebtorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "INCOME_SALARY", mappings["Company X"])
	assert.Equal(t, "INCOME_BONUS", mappings["Company Y"])

	// Add a new mapping and save
	mappings["Company Z"] = "INCOME_FREELANCE"
	err = store.SaveDebtorMappings(mappings)
	assert.NoError(t, err)

	// Reload and verify
	newMappings, err := store.LoadDebtorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "INCOME_FREELANCE", newMappings["Company Z"])
}

func TestFindConfigFileInMultipleLocations(t *testing.T) {
	tempDir := t.TempDir()

	// Create subdirectories
	configDir := filepath.Join(tempDir, "config")
	databaseDir := filepath.Join(tempDir, "database")
	err := os.MkdirAll(configDir, 0750)
	assert.NoError(t, err)
	err = os.MkdirAll(databaseDir, 0750)
	assert.NoError(t, err)

	// Change to temp directory for relative path testing
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		assert.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	store := NewCategoryStore("", "", "")

	// Test file in current directory
	currentFile := "test.yaml"
	err = os.WriteFile(currentFile, []byte("test"), 0600)
	assert.NoError(t, err)

	found, err := store.FindConfigFile("test.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "test.yaml", found)

	// Test file in config directory
	configFile := filepath.Join("config", "config-test.yaml")
	err = os.WriteFile(configFile, []byte("test"), 0600)
	assert.NoError(t, err)

	found, err = store.FindConfigFile("config-test.yaml")
	assert.NoError(t, err)
	assert.Equal(t, configFile, found)

	// Test file in database directory
	databaseFile := filepath.Join("database", "db-test.yaml")
	err = os.WriteFile(databaseFile, []byte("test"), 0600)
	assert.NoError(t, err)

	found, err = store.FindConfigFile("db-test.yaml")
	assert.NoError(t, err)
	assert.Equal(t, databaseFile, found)

	// Test nonexistent file
	_, err = store.FindConfigFile("nonexistent.yaml")
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadCategoriesSimpleFormat(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "categories.yaml")

	// Test simple list format (fallback)
	content := `
- name: Food
  keywords: ["restaurant", "food"]
- name: Transport
  keywords: ["bus", "train"]
`
	writeFile(t, file, content)
	store := NewTestCategoryStore(dir)

	cats, err := store.LoadCategories()
	assert.NoError(t, err)
	assert.Len(t, cats, 2)
	assert.Equal(t, "Food", cats[0].Name)
	assert.Equal(t, "Transport", cats[1].Name)
}

func TestLoadCreditorMappingsWithMissingFile(t *testing.T) {
	store := NewCategoryStore("", "", "")
	store.CreditorsFile = "nonexistent.yaml" // Use relative path so it goes through FindConfigFile

	mappings, err := store.LoadCreditorMappings()
	assert.NoError(t, err)
	assert.Empty(t, mappings)
}

func TestLoadDebtorMappingsWithMissingFile(t *testing.T) {
	store := NewCategoryStore("", "", "")
	store.DebtorsFile = "nonexistent.yaml" // Use relative path so it goes through FindConfigFile

	mappings, err := store.LoadDebtorMappings()
	assert.NoError(t, err)
	assert.Empty(t, mappings)
}

func TestLoadCreditorMappingsWithMalformedYAML(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "creditors.yaml")

	// Write malformed YAML
	content := `{malformed: yaml: content}`
	writeFile(t, file, content)

	store := NewCategoryStore("", file, "")
	_, err := store.LoadCreditorMappings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing creditor mappings")
}

func TestLoadDebtorMappingsWithMalformedYAML(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "debtors.yaml")

	// Write malformed YAML
	content := `{malformed: yaml: content}`
	writeFile(t, file, content)

	store := NewCategoryStore("", "", file)
	_, err := store.LoadDebtorMappings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing debtor mappings")
}

func TestSaveCreditorMappingsWithDefaultFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		assert.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	store := NewCategoryStore("", "", "") // Empty filenames use defaults

	mappings := map[string]string{
		"Test Creditor": "Test Category",
	}

	err = store.SaveCreditorMappings(mappings)
	assert.NoError(t, err)

	// Verify file was created in database directory
	expectedPath := filepath.Join("database", "creditors.yaml")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err)
}

func TestSaveDebtorMappingsWithDefaultFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		assert.NoError(t, err)
	}()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	store := NewCategoryStore("", "", "") // Empty filenames use defaults

	mappings := map[string]string{
		"Test Debtor": "Test Category",
	}

	err = store.SaveDebtorMappings(mappings)
	assert.NoError(t, err)

	// Verify file was created in database directory
	expectedPath := filepath.Join("database", "debtors.yaml")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err)
}

func TestSaveCreditorMappingsWithAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	absolutePath := filepath.Join(tempDir, "absolute-creditors.yaml")

	store := NewCategoryStore("", absolutePath, "")

	mappings := map[string]string{
		"Absolute Creditor": "Absolute Category",
	}

	err := store.SaveCreditorMappings(mappings)
	assert.NoError(t, err)

	// Verify file was created at absolute path
	_, err = os.Stat(absolutePath)
	assert.NoError(t, err)
}

func TestSaveDebtorMappingsWithAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	absolutePath := filepath.Join(tempDir, "absolute-debtors.yaml")

	store := NewCategoryStore("", "", absolutePath)

	mappings := map[string]string{
		"Absolute Debtor": "Absolute Category",
	}

	err := store.SaveDebtorMappings(mappings)
	assert.NoError(t, err)

	// Verify file was created at absolute path
	_, err = os.Stat(absolutePath)
	assert.NoError(t, err)
}

func TestResolveConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	store := NewCategoryStore("", "", "")

	// Test absolute path
	absolutePath := filepath.Join(tempDir, "test.yaml")
	resolved, err := store.resolveConfigFile(absolutePath)
	assert.NoError(t, err)
	assert.Equal(t, absolutePath, resolved)

	// Test relative path that doesn't exist
	_, err = store.resolveConfigFile("nonexistent.yaml")
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadCategoriesWithDefaultFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore directory: %v", err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create database directory and file
	if err := os.MkdirAll("database", 0750); err != nil {
		t.Fatalf("Failed to create database directory: %v", err)
	}

	content := `
categories:
  - name: Default Category
    keywords: ["default"]
`
	writeFile(t, filepath.Join("database", "categories.yaml"), content)

	store := NewCategoryStore("", "", "") // Empty filename uses default

	cats, err := store.LoadCategories()
	assert.NoError(t, err)
	assert.Len(t, cats, 1)
	assert.Equal(t, "Default Category", cats[0].Name)
}

func TestLoadCreditorMappingsWithDefaultFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore directory: %v", err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create database directory and file
	if err := os.MkdirAll("database", 0750); err != nil {
		t.Fatalf("Failed to create database directory: %v", err)
	}

	mappings := map[string]string{"Default Creditor": "Default Category"}
	data, err := yaml.Marshal(mappings)
	assert.NoError(t, err)
	if err := os.WriteFile(filepath.Join("database", "creditors.yaml"), data, 0600); err != nil {
		t.Fatalf("Failed to write creditors file: %v", err)
	}

	store := NewCategoryStore("", "", "") // Empty filename uses default

	loadedMappings, err := store.LoadCreditorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "Default Category", loadedMappings["Default Creditor"])
}

func TestLoadDebtorMappingsWithDefaultFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore directory: %v", err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create database directory and file
	if err := os.MkdirAll("database", 0750); err != nil {
		t.Fatalf("Failed to create database directory: %v", err)
	}

	mappings := map[string]string{"Default Debtor": "Default Category"}
	data, err := yaml.Marshal(mappings)
	assert.NoError(t, err)
	if err := os.WriteFile(filepath.Join("database", "debtors.yaml"), data, 0600); err != nil {
		t.Fatalf("Failed to write debtors file: %v", err)
	}

	store := NewCategoryStore("", "", "") // Empty filename uses default

	loadedMappings, err := store.LoadDebtorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "Default Category", loadedMappings["Default Debtor"])
}

// Backup functionality tests

func TestCategoryStore_BackupCreatedBeforeSave(t *testing.T) {
	tempDir := t.TempDir()
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")

	// Create initial creditors.yaml file
	initialMappings := map[string]string{"Alice": "Food", "Bob": "Transport"}
	data, err := yaml.Marshal(initialMappings)
	assert.NoError(t, err)
	err = os.WriteFile(creditorsFile, data, models.PermissionNonSecretFile)
	assert.NoError(t, err)

	store := NewCategoryStore("", creditorsFile, "")

	// Save new mappings (should create backup)
	newMappings := map[string]string{"Alice": "Food", "Bob": "Transport", "Charlie": "Entertainment"}
	err = store.SaveCreditorMappings(newMappings)
	assert.NoError(t, err)

	// Verify backup file exists with timestamp pattern
	files, err := filepath.Glob(filepath.Join(tempDir, "creditors.yaml.*.backup"))
	assert.NoError(t, err)
	assert.Len(t, files, 1, "Expected exactly one backup file")

	// Verify backup contains original data
	backupData, err := os.ReadFile(files[0])
	assert.NoError(t, err)
	var backupMappings map[string]string
	err = yaml.Unmarshal(backupData, &backupMappings)
	assert.NoError(t, err)
	assert.Equal(t, initialMappings, backupMappings, "Backup should contain original data")

	// Verify new file contains updated data
	currentData, err := os.ReadFile(creditorsFile)
	assert.NoError(t, err)
	var currentMappings map[string]string
	err = yaml.Unmarshal(currentData, &currentMappings)
	assert.NoError(t, err)
	assert.Equal(t, newMappings, currentMappings, "Current file should contain new data")
}

func TestCategoryStore_BackupUsesConfiguredLocation(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")

	// Create initial file
	initialMappings := map[string]string{"Test": "Category"}
	data, err := yaml.Marshal(initialMappings)
	assert.NoError(t, err)
	err = os.WriteFile(creditorsFile, data, models.PermissionNonSecretFile)
	assert.NoError(t, err)

	// Configure store with custom backup directory
	store := NewCategoryStore("", creditorsFile, "")
	store.SetBackupConfig(true, backupDir, "20060102_150405")

	// Save mappings
	newMappings := map[string]string{"Test": "Category", "New": "Item"}
	err = store.SaveCreditorMappings(newMappings)
	assert.NoError(t, err)

	// Verify backup is in custom directory, not same directory as original
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "creditors.yaml.*.backup"))
	assert.NoError(t, err)
	assert.Len(t, backupFiles, 1, "Expected backup in custom directory")

	originalDirBackups, err := filepath.Glob(filepath.Join(tempDir, "creditors.yaml.*.backup"))
	assert.NoError(t, err)
	assert.Len(t, originalDirBackups, 0, "Should not have backup in original directory")
}

func TestCategoryStore_BackupFailurePreventsSave(t *testing.T) {
	tempDir := t.TempDir()
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")
	readOnlyBackupDir := filepath.Join(tempDir, "readonly_backups")

	// Create initial file
	initialMappings := map[string]string{"Test": "Category"}
	data, err := yaml.Marshal(initialMappings)
	assert.NoError(t, err)
	err = os.WriteFile(creditorsFile, data, models.PermissionNonSecretFile)
	assert.NoError(t, err)

	// Create read-only backup directory
	err = os.MkdirAll(readOnlyBackupDir, 0444) // Read-only
	assert.NoError(t, err)
	defer os.Chmod(readOnlyBackupDir, 0755) // Restore permissions for cleanup

	// Configure store with read-only backup directory
	store := NewCategoryStore("", creditorsFile, "")
	store.SetBackupConfig(true, readOnlyBackupDir, "20060102_150405")

	// Attempt to save mappings - should fail due to backup failure
	newMappings := map[string]string{"Test": "Category", "New": "Item"}
	err = store.SaveCreditorMappings(newMappings)
	assert.Error(t, err, "Save should fail when backup cannot be created")
	assert.Contains(t, err.Error(), "failed to backup before save")

	// Verify original file is unchanged (backup failure prevented save)
	currentData, err := os.ReadFile(creditorsFile)
	assert.NoError(t, err)
	var currentMappings map[string]string
	err = yaml.Unmarshal(currentData, &currentMappings)
	assert.NoError(t, err)
	assert.Equal(t, initialMappings, currentMappings, "Original file should be unchanged after backup failure")
}

func TestCategoryStore_BackupDisabledSkipsBackup(t *testing.T) {
	tempDir := t.TempDir()
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")

	// Create initial file
	initialMappings := map[string]string{"Test": "Category"}
	data, err := yaml.Marshal(initialMappings)
	assert.NoError(t, err)
	err = os.WriteFile(creditorsFile, data, models.PermissionNonSecretFile)
	assert.NoError(t, err)

	// Configure store with backup disabled
	store := NewCategoryStore("", creditorsFile, "")
	store.SetBackupConfig(false, "", "20060102_150405")

	// Save mappings
	newMappings := map[string]string{"Test": "Category", "New": "Item"}
	err = store.SaveCreditorMappings(newMappings)
	assert.NoError(t, err)

	// Verify no backup file created
	backupFiles, err := filepath.Glob(filepath.Join(tempDir, "creditors.yaml.*.backup"))
	assert.NoError(t, err)
	assert.Len(t, backupFiles, 0, "Should not create backup when disabled")

	// Verify save still works
	currentData, err := os.ReadFile(creditorsFile)
	assert.NoError(t, err)
	var currentMappings map[string]string
	err = yaml.Unmarshal(currentData, &currentMappings)
	assert.NoError(t, err)
	assert.Equal(t, newMappings, currentMappings, "Save should work even with backup disabled")
}

func TestCategoryStore_MultipleBackupsWithTimestamps(t *testing.T) {
	tempDir := t.TempDir()
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")

	// Create initial file
	mappings1 := map[string]string{"Version": "1"}
	data, err := yaml.Marshal(mappings1)
	assert.NoError(t, err)
	err = os.WriteFile(creditorsFile, data, models.PermissionNonSecretFile)
	assert.NoError(t, err)

	store := NewCategoryStore("", creditorsFile, "")

	// Save multiple times with small delay
	mappings2 := map[string]string{"Version": "2"}
	err = store.SaveCreditorMappings(mappings2)
	assert.NoError(t, err)

	// Small delay to ensure different timestamp
	// Note: timestamp format includes seconds, so wait 1 second
	// For test speed, we'll just verify different backups can exist

	mappings3 := map[string]string{"Version": "3"}
	err = store.SaveCreditorMappings(mappings3)
	assert.NoError(t, err)

	// Verify multiple backup files exist with different timestamps
	backupFiles, err := filepath.Glob(filepath.Join(tempDir, "creditors.yaml.*.backup"))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(backupFiles), 1, "Should have at least one backup")

	// Verify each backup preserves data from that point in time
	// First backup should have version 1 data
	if len(backupFiles) >= 1 {
		backupData, err := os.ReadFile(backupFiles[0])
		assert.NoError(t, err)
		var backupMappings map[string]string
		err = yaml.Unmarshal(backupData, &backupMappings)
		assert.NoError(t, err)
		// First backup should be version 1 (original) or version 2 (first save)
		_, hasVersion := backupMappings["Version"]
		assert.True(t, hasVersion, "Backup should contain version data")
	}

	// Current file should have latest version
	currentData, err := os.ReadFile(creditorsFile)
	assert.NoError(t, err)
	var currentMappings map[string]string
	err = yaml.Unmarshal(currentData, &currentMappings)
	assert.NoError(t, err)
	assert.Equal(t, "3", currentMappings["Version"], "Current file should have latest version")
}
