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

// AppendCreditorSuggestion adds or updates a creditor suggestion in the staging file.
func (s *StagingStore) AppendCreditorSuggestion(partyName, categoryName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendSuggestion(s.creditorsFile, partyName, categoryName)
}

// AppendDebtorSuggestion adds or updates a debtor suggestion in the staging file.
func (s *StagingStore) AppendDebtorSuggestion(partyName, categoryName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendSuggestion(s.debtorsFile, partyName, categoryName)
}

// appendSuggestion reads the existing staging file, updates the map, and writes it back.
func (s *StagingStore) appendSuggestion(filePath, partyName, categoryName string) error {
	resolvedPath := s.resolvePath(filePath)

	mappings := make(map[string]string)
	if data, err := os.ReadFile(resolvedPath); err == nil { // #nosec G304 -- path constructed internally
		if yamlErr := yaml.Unmarshal(data, &mappings); yamlErr != nil {
			// Corrupt file — start fresh rather than failing
			mappings = make(map[string]string)
		}
	}

	mappings[strings.ToLower(partyName)] = categoryName

	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, models.PermissionDirectory); err != nil {
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
