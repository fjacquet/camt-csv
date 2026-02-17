package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewStagingStore(t *testing.T) {
	t.Run("uses defaults when empty", func(t *testing.T) {
		s := NewStagingStore("", "")
		assert.Equal(t, "staging_creditors.yaml", s.creditorsFile)
		assert.Equal(t, "staging_debtors.yaml", s.debtorsFile)
	})

	t.Run("uses provided paths", func(t *testing.T) {
		s := NewStagingStore("/tmp/cred.yaml", "/tmp/deb.yaml")
		assert.Equal(t, "/tmp/cred.yaml", s.creditorsFile)
		assert.Equal(t, "/tmp/deb.yaml", s.debtorsFile)
	})
}

func TestStagingStore_AppendCreditorSuggestion(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "staging_creditors.yaml")
	debFile := filepath.Join(tmpDir, "staging_debtors.yaml")
	s := NewStagingStore(credFile, debFile)

	t.Run("creates file and adds entry", func(t *testing.T) {
		err := s.AppendCreditorSuggestion("Starbucks", "Restaurants")
		require.NoError(t, err)

		mappings := readYAMLMap(t, credFile)
		assert.Equal(t, "Restaurants", mappings["starbucks"])
	})

	t.Run("appends to existing file", func(t *testing.T) {
		err := s.AppendCreditorSuggestion("Amazon", "Shopping")
		require.NoError(t, err)

		mappings := readYAMLMap(t, credFile)
		assert.Equal(t, "Restaurants", mappings["starbucks"])
		assert.Equal(t, "Shopping", mappings["amazon"])
	})

	t.Run("overwrites existing entry", func(t *testing.T) {
		err := s.AppendCreditorSuggestion("Amazon", "Electronics")
		require.NoError(t, err)

		mappings := readYAMLMap(t, credFile)
		assert.Equal(t, "Electronics", mappings["amazon"])
	})

	t.Run("normalizes key to lowercase", func(t *testing.T) {
		err := s.AppendCreditorSuggestion("MIGROS", "Groceries")
		require.NoError(t, err)

		mappings := readYAMLMap(t, credFile)
		assert.Equal(t, "Groceries", mappings["migros"])
	})
}

func TestStagingStore_AppendDebtorSuggestion(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "staging_creditors.yaml")
	debFile := filepath.Join(tmpDir, "staging_debtors.yaml")
	s := NewStagingStore(credFile, debFile)

	err := s.AppendDebtorSuggestion("Employer SA", "Salary")
	require.NoError(t, err)

	mappings := readYAMLMap(t, debFile)
	assert.Equal(t, "Salary", mappings["employer sa"])
}

func TestStagingStore_CorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "staging_creditors.yaml")
	debFile := filepath.Join(tmpDir, "staging_debtors.yaml")

	// Write corrupt YAML
	err := os.WriteFile(credFile, []byte("not: [valid: yaml: {{{"), 0600) // #nosec G306
	require.NoError(t, err)

	s := NewStagingStore(credFile, debFile)
	err = s.AppendCreditorSuggestion("Test", "TestCat")
	require.NoError(t, err)

	mappings := readYAMLMap(t, credFile)
	assert.Equal(t, "TestCat", mappings["test"])
}

func TestStagingStore_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "staging_creditors.yaml")
	debFile := filepath.Join(tmpDir, "staging_debtors.yaml")
	s := NewStagingStore(credFile, debFile)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			party := "party" + string(rune('a'+idx%26))
			_ = s.AppendCreditorSuggestion(party, "Category")
		}(i)
	}
	wg.Wait()

	// File should be valid YAML
	mappings := readYAMLMap(t, credFile)
	assert.NotEmpty(t, mappings)
}

func TestStagingStore_ResolvePath(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("absolute path used as-is", func(t *testing.T) {
		s := &StagingStore{}
		path := filepath.Join(tmpDir, "test.yaml")
		assert.Equal(t, path, s.resolvePath(path))
	})

	t.Run("relative path defaults to database/", func(t *testing.T) {
		s := &StagingStore{}
		result := s.resolvePath("staging_creditors.yaml")
		assert.Equal(t, filepath.Join("database", "staging_creditors.yaml"), result)
	})
}

func readYAMLMap(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var mappings map[string]string
	require.NoError(t, yaml.Unmarshal(data, &mappings))
	return mappings
}
