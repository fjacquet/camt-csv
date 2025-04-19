// Package store provides functionality for storing and retrieving application data.
// It centralizes all persistence operations to keep them separate from business logic.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	log = logrus.New()
	mu  sync.RWMutex
)

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
func NewCategoryStore() *CategoryStore {
	return &CategoryStore{
		CategoriesFile: "categories.yaml",
		CreditorsFile:  "creditors.yaml",
		DebitorsFile:   "debitors.yaml",
	}
}

// LoadCategories loads categories from the YAML file
func (s *CategoryStore) LoadCategories() ([]models.CategoryConfig, error) {
	mu.RLock()
	defer mu.RUnlock()

	filePath, err := s.findConfigFile(s.CategoriesFile)
	if err != nil {
		// Create default categories if the file doesn't exist
		if os.IsNotExist(err) {
			if err := s.createDefaultCategoriesYAML(); err != nil {
				return nil, fmt.Errorf("error creating default categories: %w", err)
			}
			filePath, err = s.findConfigFile(s.CategoriesFile)
			if err != nil {
				return nil, fmt.Errorf("error finding categories file after creation: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error finding categories file: %w", err)
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading categories file: %w", err)
	}

	// Try to manually extract and parse categories, handling potential YAML errors
	categoriesList, err := s.parseExistingCategoriesFile(data)
	if err == nil && len(categoriesList) > 0 {
		log.WithField("count", len(categoriesList)).Info("Categories loaded successfully (custom parser)")
		return categoriesList, nil
	}

	// If that fails, try our other methods
	log.Warnf("Failed to parse categories using custom parser: %v, trying standard methods", err)

	// First parse the categories with the exact format in the database file
	var dbFormat struct {
		Categories []struct {
			Name     string   `yaml:"name"`
			Keywords []string `yaml:"keywords"`
		} `yaml:"categories"`
	}

	if err := yaml.Unmarshal(data, &dbFormat); err == nil && len(dbFormat.Categories) > 0 {
		// Convert to our internal model format
		categoriesList = make([]models.CategoryConfig, len(dbFormat.Categories))
		for i, cat := range dbFormat.Categories {
			categoriesList[i] = models.CategoryConfig{
				Name:     cat.Name,
				Keywords: cat.Keywords,
			}
		}
		log.WithField("count", len(categoriesList)).Info("Categories loaded successfully (database format)")
		return categoriesList, nil
	}

	// Try with the nested structure as found in the existing database files
	var categoriesConfig struct {
		Categories []models.CategoryConfig `yaml:"categories"`
	}

	if err := yaml.Unmarshal(data, &categoriesConfig); err == nil && len(categoriesConfig.Categories) > 0 {
		log.WithField("count", len(categoriesConfig.Categories)).Info("Categories loaded successfully (nested format)")
		return categoriesConfig.Categories, nil
	}

	// Fall back to direct array unmarshaling if nested structure isn't found
	var categories []models.CategoryConfig
	if err := yaml.Unmarshal(data, &categories); err != nil {
		return nil, fmt.Errorf("error unmarshalling categories: %w", err)
	}

	log.WithField("count", len(categories)).Info("Categories loaded successfully")
	return categories, nil
}

// parseExistingCategoriesFile attempts to manually parse the categories file for maximum compatibility
func (s *CategoryStore) parseExistingCategoriesFile(data []byte) ([]models.CategoryConfig, error) {
	// Convert data to string for easier processing
	content := string(data)
	
	// Split by lines and process
	lines := strings.Split(content, "\n")
	
	var categoriesList []models.CategoryConfig
	var currentCategory *models.CategoryConfig
	var inKeywords bool
	
	for _, line := range lines {
		// Skip comments and empty lines
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Check for category name
		if strings.HasPrefix(line, "- name:") {
			if currentCategory != nil {
				categoriesList = append(categoriesList, *currentCategory)
			}
			
			// Extract name, handling quoted strings
			name := strings.TrimSpace(strings.TrimPrefix(line, "- name:"))
			name = strings.Trim(name, "\"'") // Remove quotes if present
			
			currentCategory = &models.CategoryConfig{
				Name:     name,
				Keywords: []string{},
			}
			inKeywords = false
		} else if strings.HasPrefix(line, "keywords:") {
			// Start of keywords section
			inKeywords = true
		} else if inKeywords && strings.HasPrefix(line, "- ") && currentCategory != nil {
			// A keyword entry
			keyword := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			keyword = strings.Trim(keyword, "\"'") // Remove quotes if present
			
			// Skip empty keywords
			if keyword != "" {
				currentCategory.Keywords = append(currentCategory.Keywords, keyword)
			}
		} else if strings.HasPrefix(line, "categories:") {
			// Start of categories section - reset any current category
			currentCategory = nil
			inKeywords = false
		}
	}
	
	// Add the last category if any
	if currentCategory != nil {
		categoriesList = append(categoriesList, *currentCategory)
	}
	
	if len(categoriesList) == 0 {
		return nil, fmt.Errorf("no categories found in file")
	}
	
	return categoriesList, nil
}

// LoadCreditorMappings loads creditor mappings from YAML
func (s *CategoryStore) LoadCreditorMappings() (map[string]string, error) {
	mu.RLock()
	defer mu.RUnlock()

	mappings := make(map[string]string)
	filePath, err := s.findConfigFile(s.CreditorsFile)
	if err != nil {
		// It's okay if the file doesn't exist yet
		if os.IsNotExist(err) {
			log.Info("Creditor mappings file doesn't exist yet, starting with empty mappings")
			return mappings, nil
		}
		return nil, fmt.Errorf("error finding creditor mappings file: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading creditor mappings file: %w", err)
	}

	// First try to unmarshal using the nested format as found in existing database files
	var creditorConfig struct {
		Creditors map[string]string `yaml:"creditors"`
	}
	if err := yaml.Unmarshal(data, &creditorConfig); err == nil && len(creditorConfig.Creditors) > 0 {
		// Convert all keys to lowercase
		for key, value := range creditorConfig.Creditors {
			mappings[strings.ToLower(key)] = value
		}
		log.WithField("count", len(mappings)).Info("Creditor mappings loaded successfully (nested format)")
		return mappings, nil
	}

	// If that fails, try as a direct map (for backward compatibility)
	var tempMap map[string]string
	if err := yaml.Unmarshal(data, &tempMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling creditor mappings: %w", err)
	}
	
	// Convert all keys to lowercase
	for key, value := range tempMap {
		mappings[strings.ToLower(key)] = value
	}

	log.WithField("count", len(mappings)).Info("Creditor mappings loaded successfully")
	return mappings, nil
}

// LoadDebitorMappings loads debitor mappings from YAML
func (s *CategoryStore) LoadDebitorMappings() (map[string]string, error) {
	mu.RLock()
	defer mu.RUnlock()

	mappings := make(map[string]string)
	filePath, err := s.findConfigFile(s.DebitorsFile)
	if err != nil {
		// It's okay if the file doesn't exist yet
		if os.IsNotExist(err) {
			log.Info("Debitor mappings file doesn't exist yet, starting with empty mappings")
			return mappings, nil
		}
		return nil, fmt.Errorf("error finding debitor mappings file: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading debitor mappings file: %w", err)
	}

	// First try to unmarshal using the nested format as found in existing database files
	var debitorConfig struct {
		Debitors map[string]string `yaml:"debitors"`
	}
	if err := yaml.Unmarshal(data, &debitorConfig); err == nil && len(debitorConfig.Debitors) > 0 {
		// Convert all keys to lowercase
		for key, value := range debitorConfig.Debitors {
			mappings[strings.ToLower(key)] = value
		}
		log.WithField("count", len(mappings)).Info("Debitor mappings loaded successfully (nested format)")
		return mappings, nil
	}

	// If that fails, try as a direct map (for backward compatibility)
	var tempMap map[string]string
	if err := yaml.Unmarshal(data, &tempMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling debitor mappings: %w", err)
	}
	
	// Convert all keys to lowercase
	for key, value := range tempMap {
		mappings[strings.ToLower(key)] = value
	}

	log.WithField("count", len(mappings)).Info("Debitor mappings loaded successfully")
	return mappings, nil
}

// SaveCreditorMappings saves creditor mappings to YAML
func (s *CategoryStore) SaveCreditorMappings(mappings map[string]string) error {
	mu.Lock()
	defer mu.Unlock()

	if len(mappings) == 0 {
		log.Info("No creditor mappings to save")
		return nil
	}

	// Create a temporary file in the same directory
	configDir, err := s.getConfigDir()
	if err != nil {
		return fmt.Errorf("error getting config directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	filePath := filepath.Join(configDir, s.CreditorsFile)
	tempPath := filePath + ".tmp"

	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshalling creditor mappings: %w", err)
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("error writing creditor mappings to temp file: %w", err)
	}

	// Atomically replace the old file
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("error replacing creditor mappings file: %w", err)
	}

	log.WithField("count", len(mappings)).Info("Creditor mappings saved successfully")
	return nil
}

// SaveDebitorMappings saves debitor mappings to YAML
func (s *CategoryStore) SaveDebitorMappings(mappings map[string]string) error {
	mu.Lock()
	defer mu.Unlock()

	if len(mappings) == 0 {
		log.Info("No debitor mappings to save")
		return nil
	}

	// Create a temporary file in the same directory
	configDir, err := s.getConfigDir()
	if err != nil {
		return fmt.Errorf("error getting config directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	filePath := filepath.Join(configDir, s.DebitorsFile)
	tempPath := filePath + ".tmp"

	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshalling debitor mappings: %w", err)
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("error writing debitor mappings to temp file: %w", err)
	}

	// Atomically replace the old file
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("error replacing debitor mappings file: %w", err)
	}

	log.WithField("count", len(mappings)).Info("Debitor mappings saved successfully")
	return nil
}

// findConfigFile attempts to locate a configuration file in various paths
func (s *CategoryStore) findConfigFile(filename string) (string, error) {
	// Look in current directory first
	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}

	// Look in config subdirectory
	configPath := filepath.Join("config", filename)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}
	
	// Look in database subdirectory
	dbPath := filepath.Join("database", filename)
	if _, err := os.Stat(dbPath); err == nil {
		return dbPath, nil
	}

	// Look in home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}

	// Check ~/.config/camt-csv/
	configDir := filepath.Join(homeDir, ".config", "camt-csv")
	configPath = filepath.Join(configDir, filename)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	return "", os.ErrNotExist
}

// getConfigDir returns the configuration directory to use
func (s *CategoryStore) getConfigDir() (string, error) {
	// First try to use the database directory if it exists
	if _, err := os.Stat("database"); err == nil {
		return "database", nil
	}
	
	// Next try to use config directory if it exists
	if _, err := os.Stat("config"); err == nil {
		return "config", nil
	}
	
	// Fall back to ~/.config/camt-csv/ if the above directories don't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "camt-csv")
	return configDir, nil
}

// createDefaultCategoriesYAML creates a default categories.yaml file
func (s *CategoryStore) createDefaultCategoriesYAML() error {
	configDir, err := s.getConfigDir()
	if err != nil {
		return fmt.Errorf("error getting config directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	filePath := filepath.Join(configDir, s.CategoriesFile)

	// Define default categories - only using Name and Keywords fields which are in the model
	defaultCategories := []models.CategoryConfig{
		{
			Name: "Food & Dining",
			Keywords: []string{
				"restaurant", "cafe", "coffee", "grocery", "food", "dining", "pizza", "bakery",
				"supermarket", "meal", "delivery", "takeout", "dinner", "lunch", "breakfast",
			},
		},
		{
			Name: "Transportation",
			Keywords: []string{
				"uber", "lyft", "taxi", "bus", "train", "metro", "subway", "transit", "transportation",
				"gas", "fuel", "parking", "toll", "car", "auto", "vehicle",
			},
		},
		{
			Name: "Housing",
			Keywords: []string{
				"rent", "mortgage", "apartment", "housing", "home", "property", "real estate",
				"utility", "electric", "water", "gas", "internet", "cable", "maintenance", "repair",
			},
		},
		{
			Name: "Entertainment",
			Keywords: []string{
				"movie", "cinema", "theater", "concert", "event", "ticket", "show", "performance",
				"entertainment", "streaming", "subscription", "music", "game", "festival",
			},
		},
		{
			Name: "Shopping",
			Keywords: []string{
				"amazon", "walmart", "target", "store", "shop", "retail", "mall", "purchase",
				"clothing", "apparel", "fashion", "electronics", "merchandise", "goods",
			},
		},
		{
			Name: "Health & Fitness",
			Keywords: []string{
				"doctor", "medical", "health", "hospital", "clinic", "pharmacy", "prescription",
				"drug", "medicine", "dental", "vision", "gym", "fitness", "workout", "exercise",
			},
		},
		{
			Name: "Travel",
			Keywords: []string{
				"flight", "airline", "airport", "hotel", "motel", "lodging", "accommodation",
				"travel", "vacation", "trip", "booking", "airbnb", "rental", "reservation",
			},
		},
		{
			Name: "Bills & Utilities",
			Keywords: []string{
				"bill", "utility", "subscription", "service", "payment", "fee", "charge",
				"electricity", "water", "gas", "internet", "phone", "mobile", "wireless",
			},
		},
		{
			Name: "Education",
			Keywords: []string{
				"tuition", "school", "college", "university", "education", "course", "class",
				"book", "textbook", "student", "learning", "academic", "training", "workshop",
			},
		},
		{
			Name: "Business",
			Keywords: []string{
				"business", "office", "professional", "service", "consulting", "contractor",
				"client", "supplies", "equipment", "software", "workspace", "coworking",
			},
		},
		{
			Name: "Gifts & Donations",
			Keywords: []string{
				"gift", "donation", "charity", "nonprofit", "contribution", "support",
				"fundraiser", "cause", "organization", "foundation", "giving",
			},
		},
		{
			Name: "Taxes & Fees",
			Keywords: []string{
				"tax", "fee", "penalty", "government", "irs", "audit", "assessment",
				"duty", "customs", "revenue", "fiscal", "legal", "compliance",
			},
		},
		{
			Name: "Income",
			Keywords: []string{
				"salary", "wage", "income", "deposit", "transfer", "payment", "credit",
				"refund", "reimbursement", "remittance", "compensation", "bonus", "dividend",
			},
		},
		{
			Name: "Investments",
			Keywords: []string{
				"stock", "investment", "dividend", "security", "bond", "fund", "portfolio",
				"broker", "trading", "crypto", "bitcoin", "ethereum", "blockchain", "asset",
			},
		},
		{
			Name: "Miscellaneous",
			Keywords: []string{
				"other", "miscellaneous", "misc", "general", "various", "unknown", "uncategorized",
			},
		},
	}

	data, err := yaml.Marshal(defaultCategories)
	if err != nil {
		return fmt.Errorf("error marshalling default categories: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing default categories: %w", err)
	}

	log.Info("Created default categories.yaml file")
	return nil
}
