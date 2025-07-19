package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0600)
	assert.NoError(t, err)
}

// NewTestCategoryStore returns a CategoryStore for tests with specific test paths
func NewTestCategoryStore(dir string) *CategoryStore {
	return &CategoryStore{
		CategoriesFile: filepath.Join(dir, "categories.yaml"),
		CreditorsFile:  filepath.Join(dir, "creditors.yaml"),
		DebitorsFile:   filepath.Join(dir, "debitors.yaml"),
	}
}

func TestNewCategoryStore(t *testing.T) {
	store := NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml")
	assert.Equal(t, "categories.yaml", store.CategoriesFile)
	assert.Equal(t, "creditors.yaml", store.CreditorsFile)
	assert.Equal(t, "debitors.yaml", store.DebitorsFile)
}

func TestFindConfigFile(t *testing.T) {
	dir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(dir, "test.yaml")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
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
	content := `- name: Groceries
  keywords: ["supermarket", "grocery"]
  color: "green"
- name: Rent
  keywords: ["apartment", "rent"]
  color: "blue"
`
	writeFile(t, file, content)
	store := NewTestCategoryStore(dir)
	cats, err := store.LoadCategories()
	assert.NoError(t, err)
	assert.Len(t, cats, 2)
	assert.Equal(t, "Groceries", cats[0].Name)

	// Missing file: should return empty slice, not error
	missingFile := filepath.Join(dir, "missing.yaml")
	store2 := NewTestCategoryStore(dir)
	store2.CategoriesFile = missingFile
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
	err = os.WriteFile(creditorsFile, data, 0600)
	assert.NoError(t, err)

	// Load the mappings
	store := NewCategoryStore(
		filepath.Join(tempDir, "categories.yaml"),
		creditorsFile,
		filepath.Join(tempDir, "debitors.yaml"),
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

func TestLoadAndSaveDebitorMappings(t *testing.T) {
	tempDir := t.TempDir()
	debitorsFile := filepath.Join(tempDir, "debitors.yaml")

	// Create initial mappings
	initialMappings := map[string]string{"Company X": "INCOME_SALARY", "Company Y": "INCOME_BONUS"}
	data, err := yaml.Marshal(initialMappings)
	assert.NoError(t, err)
	err = os.WriteFile(debitorsFile, data, 0600)
	assert.NoError(t, err)

	// Load the mappings
	store := NewCategoryStore(
		filepath.Join(tempDir, "categories.yaml"),
		filepath.Join(tempDir, "creditors.yaml"),
		debitorsFile,
	)

	mappings, err := store.LoadDebitorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "INCOME_SALARY", mappings["Company X"])
	assert.Equal(t, "INCOME_BONUS", mappings["Company Y"])

	// Add a new mapping and save
	mappings["Company Z"] = "INCOME_FREELANCE"
	err = store.SaveDebitorMappings(mappings)
	assert.NoError(t, err)

	// Reload and verify
	newMappings, err := store.LoadDebitorMappings()
	assert.NoError(t, err)
	assert.Equal(t, "INCOME_FREELANCE", newMappings["Company Z"])
}
