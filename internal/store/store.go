// Package store provides functionality for storing and retrieving application data.
// It centralizes all persistence operations to keep them separate from business logic.
package store

import (
	"fmt"
	"os"
	"path/filepath"
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

	var categories []models.CategoryConfig
	if err := yaml.Unmarshal(data, &categories); err != nil {
		return nil, fmt.Errorf("error unmarshalling categories: %w", err)
	}

	log.WithField("count", len(categories)).Info("Categories loaded successfully")
	return categories, nil
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

	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("error unmarshalling creditor mappings: %w", err)
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

	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("error unmarshalling debitor mappings: %w", err)
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
	// Try to use ~/.config/camt-csv/ if it exists or can be created
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
