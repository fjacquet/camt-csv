// Package store provides functionality for storing and retrieving application data.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Use the centralized logger from config package
var log = config.Logger

// SetLogger allows setting a custom logger
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
	}
}

// CategoryStore manages loading and saving of category data
type CategoryStore struct {
	CategoriesFile string
	CreditorsFile  string
	DebitorsFile   string
}

// NewCategoryStore creates a new store for category-related data
func NewCategoryStore(categoriesFile, creditorsFile, debitorsFile string) *CategoryStore {
	return &CategoryStore{
		CategoriesFile: categoriesFile,
		CreditorsFile:  creditorsFile,
		DebitorsFile:   debitorsFile,
	}
}

// FindConfigFile looks for a configuration file in standard locations
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
		log.Warnf("Configuration file not found: %s", filename)
		return "", err
	}

	return path, nil
}

// LoadCategories loads categories from the YAML file
func (s *CategoryStore) LoadCategories() ([]models.CategoryConfig, error) {
	filename := s.CategoriesFile
	if filename == "" {
		filename = "categories.yaml"
	}

	filePath, err := s.resolveConfigFile(filename)
	if err != nil {
		// If file doesn't exist, return empty slice (not an error)
		if os.IsNotExist(err) {
			log.Warnf("Categories file not found: %s", filename)
			return []models.CategoryConfig{}, nil
		}
		return nil, fmt.Errorf("error resolving categories file: %w", err)
	}

	// Check if the file exists
	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Categories file not found: %s", filePath)
			return []models.CategoryConfig{}, nil
		}
		return nil, fmt.Errorf("error checking categories file: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading categories file: %w", err)
	}

	// First try to unmarshal using the proper CategoriesConfig struct
	// This handles the proper YAML structure: "categories: [...]"
	var categoriesConfig models.CategoriesConfig
	err = yaml.Unmarshal(data, &categoriesConfig)
	if err == nil && len(categoriesConfig.Categories) > 0 {
		log.Debugf("Loaded %d categories from %s using CategoriesConfig",
			len(categoriesConfig.Categories), filePath)
		return categoriesConfig.Categories, nil
	}

	// Fallback 1: Try to unmarshal directly to an array if the file doesn't have the top-level key
	var categories []models.CategoryConfig
	err = yaml.Unmarshal(data, &categories)
	if err == nil && len(categories) > 0 {
		log.Debugf("Loaded %d categories from %s using direct array", len(categories), filePath)
		return categories, nil
	}

	// Fallback 2: Try parsing manually for backward compatibility
	return s.parseExistingCategoriesFile(data)
}

// parseExistingCategoriesFile attempts to manually parse the categories file for maximum compatibility
// parseExistingCategoriesFile attempts to manually parse the categories file for maximum backward compatibility.
// It assumes a simple YAML file with category names as keys and descriptions or keyword lists as values.
// This function is used as a fallback when the standard CategoriesConfig format is not used.
// It returns an empty slice if the file is empty or can't be parsed.
func (s *CategoryStore) parseExistingCategoriesFile(data []byte) ([]models.CategoryConfig, error) {
	// Attempt to unmarshal as a map
	var categoriesMap map[string]interface{}
	if err := yaml.Unmarshal(data, &categoriesMap); err != nil {
		return nil, fmt.Errorf("error parsing categories file: %w", err)
	}

	var categories []models.CategoryConfig

	// Process each key-value pair in the map
	for name, value := range categoriesMap {
		category := models.CategoryConfig{
			Name: name,
		}

		// Value could be a string (description) or a map (with keywords, etc.)
		switch v := value.(type) {
		case string:
			// Simple format: category: description
			// We ignore the description since CategoryConfig has no Description field
		case map[string]interface{}:
			// Extract keywords if present
			if keywordsVal, ok := v["keywords"]; ok {
				if keywordsList, ok := keywordsVal.([]interface{}); ok {
					for _, k := range keywordsList {
						if keyword, ok := k.(string); ok {
							category.Keywords = append(category.Keywords, strings.ToLower(keyword))
						}
					}
				}
			}
		}

		categories = append(categories, category)
	}

	log.Debugf("Parsed %d categories from custom format", len(categories))
	return categories, nil
}

// LoadCreditorMappings loads creditor mappings from YAML
func (s *CategoryStore) LoadCreditorMappings() (map[string]string, error) {
	filename := s.CreditorsFile
	if filename == "" {
		filename = "creditors.yaml"
	}

	filePath, err := s.resolveConfigFile(filename)
	if err != nil {
		// If file doesn't exist, return empty map (not an error)
		if os.IsNotExist(err) {
			log.Warnf("Creditor mappings file not found: %s", filename)
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("error resolving creditor mappings file: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading creditor mappings file: %w", err)
	}

	var mappings map[string]string
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("error parsing creditor mappings: %w", err)
	}

	log.Debugf("Loaded %d creditor mappings from %s", len(mappings), filePath)
	return mappings, nil
}

// LoadDebitorMappings loads debitor mappings from YAML
func (s *CategoryStore) LoadDebitorMappings() (map[string]string, error) {
	filename := s.DebitorsFile
	if filename == "" {
		filename = "debitors.yaml"
	}

	filePath, err := s.resolveConfigFile(filename)
	if err != nil {
		// If file doesn't exist, return empty map (not an error)
		if os.IsNotExist(err) {
			log.Warnf("Debitor mappings file not found: %s", filename)
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("error resolving debitor mappings file: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading debitor mappings file: %w", err)
	}

	var mappings map[string]string
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("error parsing debitor mappings: %w", err)
	}

	log.Debugf("Loaded %d debitor mappings from %s", len(mappings), filePath)
	return mappings, nil
}

// SaveCreditorMappings saves creditor mappings to YAML
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshaling creditor mappings: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing creditor mappings: %w", err)
	}

	log.Debugf("Saved %d creditor mappings to %s", len(mappings), filePath)
	return nil
}

// SaveDebitorMappings saves debitor mappings to YAML
func (s *CategoryStore) SaveDebitorMappings(mappings map[string]string) error {
	filename := s.DebitorsFile
	if filename == "" {
		filename = "debitors.yaml"
	}

	// Find the existing file or use standard locations
	filePath, err := s.FindConfigFile(filename)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("error resolving debitor mappings file: %w", err)
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshaling debitor mappings: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing debitor mappings: %w", err)
	}

	log.Debugf("Saved %d debitor mappings to %s", len(mappings), filePath)
	return nil
}
