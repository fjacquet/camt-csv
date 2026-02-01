// Package store provides functionality for storing and retrieving application configuration data.
// It manages YAML-based configuration files for categories, creditor mappings, and debtor mappings
// used by the transaction categorization system.
//
// The store supports flexible file location resolution, checking multiple standard locations
// and providing fallback mechanisms for configuration files. It handles both loading existing
// configurations and saving updated mappings back to disk.
//
// Configuration files supported:
//   - categories.yaml: Category definitions with keywords for pattern matching
//   - creditors.yaml: Direct mappings from creditor names to categories
//   - debtors.yaml: Direct mappings from debtor names to categories
package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"fjacquet/camt-csv/internal/models"

	"gopkg.in/yaml.v3"
)

// Note: Global logger removed in favor of dependency injection.
// All logging is now done through the logger passed to CategoryStore methods.

// CategoryStore manages loading and saving of category-related configuration data.
// It provides a centralized interface for accessing category definitions, creditor mappings,
// and debtor mappings from YAML files with intelligent file location resolution.
//
// The store supports both absolute and relative file paths, with automatic resolution
// of configuration files in standard locations including the current directory,
// config subdirectories, and user home directory.
type CategoryStore struct {
	CategoriesFile string // Path to the categories configuration file
	CreditorsFile  string // Path to the creditor mappings file
	DebtorsFile    string // Path to the debtor mappings file

	// Backup configuration (optional, defaults provided if not set)
	backupEnabled         bool
	backupDirectory       string
	backupTimestampFormat string
}

// NewCategoryStore creates a new CategoryStore instance with the specified file paths.
// If empty strings are provided for any file path, default filenames will be used
// during file operations (categories.yaml, creditors.yaml, debtors.yaml).
//
// Backup is enabled by default with timestamps in format "20060102_150405".
// Use SetBackupConfig to customize backup behavior.
//
// Parameters:
//   - categoriesFile: Path to the categories configuration file
//   - creditorsFile: Path to the creditor mappings file
//   - debtorsFile: Path to the debtor mappings file
//
// Returns:
//   - *CategoryStore: A new store instance ready for use
func NewCategoryStore(categoriesFile, creditorsFile, debtorsFile string) *CategoryStore {
	return &CategoryStore{
		CategoriesFile:        categoriesFile,
		CreditorsFile:         creditorsFile,
		DebtorsFile:           debtorsFile,
		backupEnabled:         true,              // Default: backup enabled
		backupDirectory:       "",                // Default: same directory as original
		backupTimestampFormat: "20060102_150405", // Default: YYYYMMDD_HHMMSS
	}
}

// SetBackupConfig configures the backup behavior for this store.
// This method allows customization of backup settings, typically called from
// the container after reading application configuration.
//
// Parameters:
//   - enabled: Whether to create backups before saving files
//   - directory: Directory for backup files (empty string = same as original)
//   - timestampFormat: Go time format string for backup filename timestamps
func (s *CategoryStore) SetBackupConfig(enabled bool, directory, timestampFormat string) {
	s.backupEnabled = enabled
	s.backupDirectory = directory
	s.backupTimestampFormat = timestampFormat
}

// FindConfigFile looks for a configuration file in standard locations.
// It searches in the following order:
//  1. Current directory
//  2. ./config/ subdirectory
//  3. ./database/ subdirectory
//  4. User home directory under .config/camt-csv/
//
// If the filename is an absolute path, it checks that path directly.
//
// Parameters:
//   - filename: Name or path of the configuration file to find
//
// Returns:
//   - string: Full path to the found configuration file
//   - error: os.ErrNotExist if file is not found, or other error if path resolution fails
func (s *CategoryStore) FindConfigFile(filename string) (string, error) {
	// Check if it's an absolute path
	if filepath.IsAbs(filename) {
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}
		return "", os.ErrNotExist
	}

	// Common locations to check for config files
	locations := []string{
		filename,                            // Current directory
		filepath.Join("config", filename),   // ./config/ directory
		filepath.Join("database", filename), // ./database/ directory
	}

	// Try each location
	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return location, nil
		}
	}

	// If still not found, check in user's home directory under .config/camt-csv/
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(homeDir, ".config", "camt-csv")
		configPath := filepath.Join(configDir, filename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	return "", os.ErrNotExist
}

// resolveConfigFile gets the full path to a config file
func (s *CategoryStore) resolveConfigFile(filename string) (string, error) {
	if filepath.IsAbs(filename) {
		return filename, nil
	}

	path, err := s.FindConfigFile(filename)
	if err != nil {
		return "", err
	}

	return path, nil
}

// LoadCategories loads category definitions from the configured YAML file.
// It supports both structured format (with a "categories" key) and simple list format
// for backward compatibility. If the file is not found, returns an empty slice without error.
//
// Returns:
//   - []models.CategoryConfig: Slice of category configurations loaded from the file
//   - error: Any error encountered during file reading or YAML parsing
func (s *CategoryStore) LoadCategories() ([]models.CategoryConfig, error) {
	filename := s.CategoriesFile
	if filename == "" {
		filename = "categories.yaml"
	}

	filePath, err := s.resolveConfigFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.CategoryConfig{}, nil
		}
		return nil, fmt.Errorf("error resolving categories file: %w", err)
	}

	data, err := os.ReadFile(filePath) // #nosec G304 -- config file path resolved internally
	if err != nil {
		if os.IsNotExist(err) {
			return []models.CategoryConfig{}, nil
		}
		return nil, fmt.Errorf("error reading categories file: %w", err)
	}

	var config struct {
		Categories []models.CategoryConfig `yaml:"categories"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		// Fallback for a simple list of categories
		var categories []models.CategoryConfig
		if err2 := yaml.Unmarshal(data, &categories); err2 != nil {
			return nil, fmt.Errorf("error parsing categories file: %w", err)
		}

		return categories, nil
	}

	return config.Categories, nil
}

// createBackup creates a timestamped backup of the specified file.
// This is called before saving to provide a safety net for category mapping changes.
// If the original file doesn't exist, no backup is created (no error).
// If backup is disabled, no backup is created (no error).
//
// The backup filename follows the pattern: {original}.{timestamp}.backup
// Example: creditors.yaml.20260201_143022.backup
//
// Parameters:
//   - filePath: Path to the file to backup
//
// Returns:
//   - error: Any error encountered during backup creation (critical - prevents save)
func (s *CategoryStore) createBackup(filePath string) error {
	// Skip if backup is disabled
	if !s.backupEnabled {
		return nil
	}

	// Check if original file exists - nothing to backup if it doesn't
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking original file: %w", err)
	}

	// Generate timestamped backup filename
	timestamp := time.Now().Format(s.backupTimestampFormat)
	backupFilename := filepath.Base(filePath) + "." + timestamp + ".backup"

	// Determine backup location
	var backupPath string
	if s.backupDirectory != "" {
		// Use configured backup directory
		if err := os.MkdirAll(s.backupDirectory, models.PermissionDirectory); err != nil {
			return fmt.Errorf("error creating backup directory: %w", err)
		}
		backupPath = filepath.Join(s.backupDirectory, backupFilename)
	} else {
		// Use same directory as original file
		backupPath = filepath.Join(filepath.Dir(filePath), backupFilename)
	}

	// Copy original file to backup location
	source, err := os.Open(filePath) // #nosec G304 -- filePath is resolved internally
	if err != nil {
		return fmt.Errorf("error opening file for backup: %w", err)
	}
	defer func() { _ = source.Close() }()

	destination, err := os.Create(backupPath) // #nosec G304 -- backupPath is constructed internally
	if err != nil {
		return fmt.Errorf("error creating backup file: %w", err)
	}
	defer func() { _ = destination.Close() }()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("error copying file to backup: %w", err)
	}

	// Set backup file permissions to match original (0644 for category mappings per SEC-03)
	if err := os.Chmod(backupPath, models.PermissionNonSecretFile); err != nil {
		return fmt.Errorf("error setting backup file permissions: %w", err)
	}

	return nil
}

// LoadCreditorMappings loads creditor-to-category mappings from the configured YAML file.
// These mappings provide direct associations between creditor names and their assigned categories,
// enabling fast categorization without pattern matching or AI inference.
//
// Returns:
//   - map[string]string: Map of creditor names to category names
//   - error: Any error encountered during file reading or YAML parsing
func (s *CategoryStore) LoadCreditorMappings() (map[string]string, error) {
	filename := s.CreditorsFile
	if filename == "" {
		filename = "creditors.yaml"
	}

	filePath, err := s.resolveConfigFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("error resolving creditor mappings file: %w", err)
	}

	data, err := os.ReadFile(filePath) // #nosec G304 -- config file path resolved internally
	if err != nil {
		return nil, fmt.Errorf("error reading creditor mappings file: %w", err)
	}

	var mappings map[string]string
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("error parsing creditor mappings: %w", err)
	}

	return mappings, nil
}

// LoadDebtorMappings loads debtor-to-category mappings from the configured YAML file.
// These mappings provide direct associations between debtor names and their assigned categories,
// enabling fast categorization without pattern matching or AI inference.
//
// Returns:
//   - map[string]string: Map of debtor names to category names
//   - error: Any error encountered during file reading or YAML parsing
func (s *CategoryStore) LoadDebtorMappings() (map[string]string, error) {
	filename := s.DebtorsFile
	if filename == "" {
		filename = "debtors.yaml"
	}

	filePath, err := s.resolveConfigFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("error resolving debtor mappings file: %w", err)
	}

	data, err := os.ReadFile(filePath) // #nosec G304 -- config file path resolved internally
	if err != nil {
		return nil, fmt.Errorf("error reading debtor mappings file: %w", err)
	}

	var mappings map[string]string
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("error parsing debtor mappings: %w", err)
	}

	return mappings, nil
}

// SaveCreditorMappings saves creditor-to-category mappings to the configured YAML file.
// If the file doesn't exist, it creates it in the database directory. The method ensures
// the parent directory exists before writing and uses appropriate file permissions.
//
// This method is typically called by the auto-learning feature when AI categorization
// successfully categorizes a transaction, allowing future transactions from the same
// creditor to be categorized instantly.
//
// Parameters:
//   - mappings: Map of creditor names to category names to save
//
// Returns:
//   - error: Any error encountered during file writing or directory creation
func (s *CategoryStore) SaveCreditorMappings(mappings map[string]string) error {
	filename := s.CreditorsFile
	if filename == "" {
		filename = "creditors.yaml"
	}

	// Find the existing file or use standard locations
	filePath, err := s.FindConfigFile(filename)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("error resolving creditor mappings file: %w", err)
	}

	// If file not found, use the database directory by default
	if err == os.ErrNotExist {
		if !filepath.IsAbs(filename) {
			// Default to database directory
			filePath = filepath.Join("database", filename)
		} else {
			filePath = filename
		}
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, models.PermissionDirectory); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create backup before modifying the file (critical - prevents data loss)
	if err := s.createBackup(filePath); err != nil {
		return fmt.Errorf("failed to backup before save: %w", err)
	}

	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshaling creditor mappings: %w", err)
	}

	// SECURITY: Creditor mappings are non-secret (just category mappings), use 0644 permissions
	if err := os.WriteFile(filePath, data, models.PermissionNonSecretFile); err != nil {
		return fmt.Errorf("error writing creditor mappings: %w", err)
	}

	return nil
}

// SaveDebtorMappings saves debtor-to-category mappings to the configured YAML file.
// If the file doesn't exist, it creates it in the database directory. The method ensures
// the parent directory exists before writing and uses appropriate file permissions.
//
// This method is typically called by the auto-learning feature when AI categorization
// successfully categorizes a transaction, allowing future transactions from the same
// debtor to be categorized instantly.
//
// Parameters:
//   - mappings: Map of debtor names to category names to save
//
// Returns:
//   - error: Any error encountered during file writing or directory creation
func (s *CategoryStore) SaveDebtorMappings(mappings map[string]string) error {
	filename := s.DebtorsFile
	if filename == "" {
		filename = "debtors.yaml"
	}

	// Find the existing file or use standard locations
	filePath, err := s.FindConfigFile(filename)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("error resolving debtor mappings file: %w", err)
	}

	// If file not found, use the database directory by default
	if err == os.ErrNotExist {
		if !filepath.IsAbs(filename) {
			// Default to database directory
			filePath = filepath.Join("database", filename)
		} else {
			filePath = filename
		}
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, models.PermissionDirectory); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create backup before modifying the file (critical - prevents data loss)
	if err := s.createBackup(filePath); err != nil {
		return fmt.Errorf("failed to backup before save: %w", err)
	}

	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshaling debtor mappings: %w", err)
	}

	// SECURITY: Debtor mappings are non-secret (just category mappings), use 0644 permissions
	if err := os.WriteFile(filePath, data, models.PermissionNonSecretFile); err != nil {
		return fmt.Errorf("error writing debtor mappings: %w", err)
	}

	return nil
}
