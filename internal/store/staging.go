package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/models"

	"gopkg.in/yaml.v3"
)

// StagingStore manages staging YAML files for unreviewed AI categorization suggestions.
// When auto-learn is disabled, AI results are written here instead of being discarded.
// The format is identical to creditors.yaml/debtors.yaml (map[string]string) for easy
// manual promotion by the user.
type StagingStore struct {
	creditorsFile string
	debtorsFile   string
	mu            sync.Mutex
}

// NewStagingStore creates a new StagingStore with the given file paths.
// Paths are resolved relative to the database/ directory if not absolute.
func NewStagingStore(creditorsFile, debtorsFile string) *StagingStore {
	if creditorsFile == "" {
		creditorsFile = "staging_creditors.yaml"
	}
	if debtorsFile == "" {
		debtorsFile = "staging_debtors.yaml"
	}
	return &StagingStore{
		creditorsFile: creditorsFile,
		debtorsFile:   debtorsFile,
	}
}

// LoadSuggestions reads both staging files. A missing or corrupt file reads
// as empty: staging is a convenience, never a reason to stop a run.
func (s *StagingStore) LoadSuggestions() (map[string]string, map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(s.creditorsFile), s.read(s.debtorsFile), nil
}

// MergeSuggestions adds suggestions to the staging files, keeping what is
// already there, with one read and one write per file.
func (s *StagingStore) MergeSuggestions(creditors, debtors map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.merge(s.creditorsFile, creditors); err != nil {
		return err
	}
	return s.merge(s.debtorsFile, debtors)
}

func (s *StagingStore) read(filePath string) map[string]string {
	mappings := make(map[string]string)
	data, err := os.ReadFile(s.resolvePath(filePath)) // #nosec G304 -- path constructed internally
	if err != nil {
		return mappings
	}
	if yaml.Unmarshal(data, &mappings) != nil {
		return make(map[string]string)
	}
	return mappings
}

func (s *StagingStore) merge(filePath string, suggestions map[string]string) error {
	if len(suggestions) == 0 {
		return nil
	}
	mappings := s.read(filePath)
	for party, category := range suggestions {
		mappings[strings.ToLower(strings.TrimSpace(party))] = category
	}
	resolvedPath := s.resolvePath(filePath)
	if err := os.MkdirAll(filepath.Dir(resolvedPath), models.PermissionDirectory); err != nil {
		return fmt.Errorf("error creating staging directory: %w", err)
	}
	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshaling staging suggestions: %w", err)
	}
	if err := os.WriteFile(resolvedPath, data, models.PermissionNonSecretFile); err != nil {
		return fmt.Errorf("error writing staging file %s: %w", resolvedPath, err)
	}
	return nil
}

// resolvePath resolves a staging file path, defaulting to the database/ subdirectory.
func (s *StagingStore) resolvePath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	// Check if file already exists at the given path
	if _, err := os.Stat(filename); err == nil {
		return filename
	}
	// Check in database/ subdirectory
	dbPath := filepath.Join("database", filepath.Base(filename))
	if _, err := os.Stat(dbPath); err == nil {
		return dbPath
	}
	// Default to database/ for new files
	return filepath.Join("database", filepath.Base(filename))
}
