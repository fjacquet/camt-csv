// Package categorizer provides functionality to categorize transactions using multiple methods:
// 1. Direct seller-to-category mapping from a YAML database
// 2. Local keyword-based categorization from rules in a YAML file
// 3. AI-based categorization using Gemini model as a fallback
package categorizer

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fjacquet/camt-csv/pkg/config"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

// Transaction represents a financial transaction to be categorized
type Transaction struct {
	Payee   string
	Amount  string
	Date    string
	Info    string
}

// Category represents a transaction category
type Category struct {
	Name        string
	Description string
}

// CategoryConfig represents a category configuration in the YAML file
type CategoryConfig struct {
	Name     string   `yaml:"name"`
	Keywords []string `yaml:"keywords"`
}

// CategoriesConfig represents the structure of the categories YAML file
type CategoriesConfig struct {
	Categories []CategoryConfig `yaml:"categories"`
}

// SellersConfig represents the structure of the sellers YAML file
type SellersConfig struct {
	Sellers map[string]string `yaml:"sellers"`
}

// Categorizer handles the categorization of transactions and manages
// the category and payee mapping databases
type Categorizer struct {
	categories     []CategoryConfig
	payeeMappings  map[string]string
	configMutex    sync.RWMutex
	isDirty        bool // Track if payeeMappings has been modified and needs to be saved
	geminiClient   *genai.Client
	geminiModel    *genai.GenerativeModel
}

// Global singleton instance
var defaultCategorizer *Categorizer
var initOnce sync.Once

// initCategorizer initializes the default categorizer instance
func initCategorizer() {
	defaultCategorizer = &Categorizer{
		payeeMappings: make(map[string]string),
	}
	
	// Load categories from YAML
	err := defaultCategorizer.loadCategoriesFromYAML()
	if err != nil {
		log.Printf("Warning: Could not load categories from YAML: %v", err)
		log.Printf("Falling back to default categories")
		
		// Create default categories if YAML loading fails
		defaultCategorizer.categories = []CategoryConfig{
			{Name: "Food & Dining", Keywords: []string{"restaurant", "food", "dining", "cafe"}},
			{Name: "Groceries", Keywords: []string{"supermarket", "grocery", "coop", "migros"}},
			{Name: "Shopping", Keywords: []string{"shop", "store", "retail", "amazon"}},
			{Name: "Utilities", Keywords: []string{"electric", "water", "utility", "bill"}},
			{Name: "Transportation", Keywords: []string{"transport", "train", "bus", "taxi"}},
			{Name: "Uncategorized", Keywords: []string{"unknown", "other"}},
		}
	}
	
	// Load payee mappings from YAML
	err = defaultCategorizer.loadPayeesFromYAML()
	if err != nil {
		log.Printf("Warning: Could not load payee mappings from YAML: %v", err)
		log.Printf("Starting with empty payee mappings database")
	}
}

// ensureGeminiClient ensures the Gemini client is initialized
func (c *Categorizer) ensureGeminiClient() error {
	if c.geminiClient != nil {
		return nil
	}

	// Get API key from environment variables
	apiKey := config.GetGeminiAPIKey()
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	// Initialize the Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	c.geminiClient = client
	c.geminiModel = client.GenerativeModel("gemini-1.0-pro")
	return nil
}

// loadCategoriesFromYAML loads the categories from the YAML configuration file
func (c *Categorizer) loadCategoriesFromYAML() error {
	yamlPath, err := c.findConfigFile("categories.yaml")
	if err != nil {
		return err
	}
	
	// Read and parse the YAML file
	yamlData, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("could not read categories.yaml file: %w", err)
	}
	
	var config CategoriesConfig
	err = yaml.Unmarshal(yamlData, &config)
	if err != nil {
		return fmt.Errorf("could not parse categories.yaml file: %w", err)
	}
	
	// Store the loaded categories
	c.categories = config.Categories
	log.Printf("Loaded %d categories from YAML file", len(c.categories))
	
	return nil
}

// loadPayeesFromYAML loads the payee-to-category mappings from the YAML configuration file
func (c *Categorizer) loadPayeesFromYAML() error {
	yamlPath, err := c.findConfigFile("payees.yaml")
	if err != nil {
		// If the file doesn't exist yet, this is not an error
		// We'll create an empty map and save it later
		c.configMutex.Lock()
		c.payeeMappings = make(map[string]string)
		c.isDirty = true  // Mark as dirty so it gets saved
		c.configMutex.Unlock()
		log.Printf("Payee mappings file not found, will create it on next save")
		return nil
	}
	
	// Read and parse the YAML file
	yamlData, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("could not read payees.yaml file: %w", err)
	}
	
	// First, try to unmarshal with the expected structure (with 'payees' key)
	var config SellersConfig
	err = yaml.Unmarshal(yamlData, &config)
	if err != nil {
		// Try parsing directly as a map (without the 'payees' key)
		directMap := make(map[string]string)
		err = yaml.Unmarshal(yamlData, &directMap)
		if err != nil {
			return fmt.Errorf("could not parse payees.yaml file: %w", err)
		}
		
		// If successful, use the direct map
		c.configMutex.Lock()
		c.payeeMappings = directMap
		c.isDirty = false
		c.configMutex.Unlock()
		log.Printf("Loaded %d payee mappings from YAML file (direct format)", len(directMap))
		return nil
	}
	
	// Acquire write lock before updating the map
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Store the loaded payee mappings
	if config.Sellers != nil {
		c.payeeMappings = config.Sellers
		log.Printf("Loaded %d payee mappings from YAML file", len(c.payeeMappings))
	} else {
		c.payeeMappings = make(map[string]string)
		log.Printf("Initialized empty payee mappings")
	}
	
	// Reset the dirty flag since we just loaded from disk
	c.isDirty = false
	
	return nil
}

// savePayeesToYAML saves the current payee mappings to the YAML file
// This should be called at the end of processing or when the application exits
func (c *Categorizer) savePayeesToYAML() error {
	// If no changes were made, don't bother saving
	if !c.isDirty {
		log.Printf("No changes to payee mappings, skipping save")
		return nil
	}
	
	yamlPath, err := c.findConfigFile("payees.yaml")
	if err != nil {
		// If we can't find the existing file, create a new one in the database directory
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine executable path: %w", err)
		}
		
		baseDir := filepath.Dir(execPath)
		dbDir := filepath.Join(baseDir, "database")
		
		// Create the database directory if it doesn't exist
		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			err = os.MkdirAll(dbDir, 0755)
			if err != nil {
				return fmt.Errorf("could not create database directory: %w", err)
			}
		}
		
		yamlPath = filepath.Join(dbDir, "payees.yaml")
	}
	
	// Acquire read lock to create a copy of the map
	c.configMutex.RLock()
	config := SellersConfig{
		Sellers: make(map[string]string, len(c.payeeMappings)),
	}
	
	// Copy the map to avoid concurrent access issues
	for payee, category := range c.payeeMappings {
		config.Sellers[payee] = category
	}
	c.configMutex.RUnlock()
	
	// Generate a header comment for the file
	header := `# Payee to category mappings
# This file maps payee names to their respective categories
# It is automatically updated when new payees are categorized
# Format:
#   payees:
#     "Payee Name 1": "Category Name"
#     "Payee Name 2": "Category Name"

`
	
	// Marshal the config to YAML
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("could not marshal payees to YAML: %w", err)
	}
	
	// Combine header and YAML data
	finalData := []byte(header + string(yamlData))
	
	// Write the YAML data to file
	err = ioutil.WriteFile(yamlPath, finalData, 0644)
	if err != nil {
		return fmt.Errorf("could not write payees.yaml file: %w", err)
	}
	
	log.Printf("Saved %d payee mappings to YAML file", len(config.Sellers))
	
	// Reset the dirty flag
	c.configMutex.Lock()
	c.isDirty = false
	c.configMutex.Unlock()
	
	return nil
}

// findConfigFile attempts to locate a configuration file in various paths
func (c *Categorizer) findConfigFile(filename string) (string, error) {
	// Try executable directory first
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}
	
	baseDir := filepath.Dir(execPath)
	yamlPath := filepath.Join(baseDir, "database", filename)
	
	// Check if the file exists
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		// Try a relative path from current directory
		yamlPath = filepath.Join("database", filename)
		if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
			return "", fmt.Errorf("%s file not found: %w", filename, err)
		}
	}
	
	return yamlPath, nil
}

// getCategories returns the list of category names
func (c *Categorizer) getCategories() []string {
	var categories []string
	for _, category := range c.categories {
		categories = append(categories, category.Name)
	}
	return categories
}

// updatePayeeCategory adds or updates a payee-to-category mapping
func (c *Categorizer) updatePayeeCategory(payee, category string) {
	// Normalize the payee name (trim spaces)
	normalizedPayee := strings.TrimSpace(payee)
	if normalizedPayee == "" {
		return
	}
	
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Check if the mapping already exists and is the same
	existingCategory, exists := c.payeeMappings[normalizedPayee]
	if exists && existingCategory == category {
		return // No change needed
	}
	
	// Update the mapping
	c.payeeMappings[normalizedPayee] = category
	c.isDirty = true
	
	log.Printf("Added/updated payee mapping: %s -> %s", normalizedPayee, category)
}

// getPayeeCategory retrieves the category for a given payee if it exists
func (c *Categorizer) getPayeeCategory(payee string) (string, bool) {
	// Normalize the payee name (trim spaces)
	normalizedPayee := strings.TrimSpace(payee)
	if normalizedPayee == "" {
		return "", false
	}
	
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	category, exists := c.payeeMappings[normalizedPayee]
	return category, exists
}

// categorizeByPayeeMapping attempts to categorize a transaction using the payee mapping database
func (c *Categorizer) categorizeByPayeeMapping(transaction Transaction) (Category, bool) {
	if transaction.Payee == "" {
		return Category{}, false
	}
	
	category, exists := c.getPayeeCategory(transaction.Payee)
	if !exists {
		return Category{}, false
	}
	
	return Category{
		Name:        category,
		Description: fmt.Sprintf("Categorized based on known payee: %s", transaction.Payee),
	}, true
}

// categorizeLocallyByKeywords attempts to categorize a transaction using the local keyword database
func (c *Categorizer) categorizeLocallyByKeywords(transaction Transaction) (Category, bool) {
	// Combine all text fields for better matching
	text := strings.ToLower(transaction.Payee + " " + transaction.Info)
	
	// Check each category's keywords for matches
	for _, category := range c.categories {
		for _, keyword := range category.Keywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				return Category{
					Name:        category.Name,
					Description: fmt.Sprintf("Matched keyword: %s", keyword),
				}, true
			}
		}
	}
	
	return Category{}, false
}

// categorizeWithGemini uses the Gemini API to categorize a transaction
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (Category, error) {
	// Ensure the Gemini client is initialized
	err := c.ensureGeminiClient()
	if err != nil {
		return Category{}, err
	}
	
	ctx := context.Background()
	
	// Prepare the prompt for Gemini
	prompt := fmt.Sprintf(`Categorize the following financial transaction:
Payee: %s
Amount: %s
Date: %s
Additional Info: %s

Please assign this transaction to exactly one of the following categories:
%s

Respond in this format:
Category: [Selected Category Name]
Description: [Brief explanation of why you chose this category]`,
		transaction.Payee,
		transaction.Amount,
		transaction.Date,
		transaction.Info,
		strings.Join(c.getCategories(), ", "))
	
	// Create the generation request
	resp, err := c.geminiModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return Category{}, fmt.Errorf("Gemini API error: %w", err)
	}
	
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return Category{}, fmt.Errorf("no response from Gemini API")
	}
	
	// Extract category and description from response
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	categoryName, description := c.extractCategoryFromResponse(responseText)
	
	log.Printf("Gemini classified transaction %s as %s: %s",
		transaction.Payee, categoryName, description)
	
	// Update the payee mapping in our database
	if transaction.Payee != "" {
		c.updatePayeeCategory(transaction.Payee, categoryName)
	}
	
	return Category{
		Name:        categoryName,
		Description: description,
	}, nil
}

// extractCategoryFromResponse parses the Gemini API response to extract the category
func (c *Categorizer) extractCategoryFromResponse(response string) (string, string) {
	lines := strings.Split(response, "\n")
	var categoryName, description string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Category:") {
			categoryName = strings.TrimSpace(strings.TrimPrefix(line, "Category:"))
		} else if strings.HasPrefix(line, "Description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}
	
	// If no structured response was found, use the entire response as the description
	if categoryName == "" {
		// Try to find a matching category in the response
		for _, c := range c.getCategories() {
			if strings.Contains(response, c) {
				categoryName = c
				break
			}
		}
		if categoryName == "" {
			categoryName = "Uncategorized"
		}
		description = strings.TrimSpace(response)
	}
	
	return categoryName, description
}

// categorizeTransaction categorizes a transaction using the following sequence:
// 1. Check if the payee already exists in the payee mapping database
// 2. Try to match using local keyword patterns
// 3. Fall back to Gemini API as a last resort
func (c *Categorizer) categorizeTransaction(transaction Transaction) (Category, error) {
	// 1. Check if the payee already exists in our mapping database
	if category, found := c.categorizeByPayeeMapping(transaction); found {
		log.Printf("Transaction categorized by payee mapping as: %s", category.Name)
		return category, nil
	}
	
	// 2. Try to categorize locally by keywords
	if category, matched := c.categorizeLocallyByKeywords(transaction); matched {
		log.Printf("Transaction categorized by keywords as: %s", category.Name)
		// Update the payee mapping for future use
		if transaction.Payee != "" {
			c.updatePayeeCategory(transaction.Payee, category.Name)
		}
		return category, nil
	}
	
	// 3. If previous methods fail, use the Gemini API
	log.Printf("Local categorization failed for: %s, using Gemini API", transaction.Payee)
	return c.categorizeWithGemini(transaction)
}

// -------- Public API (Facade Methods) --------

// GetCategories returns the list of category names for external use
// This is a package-level function that uses the default categorizer
func GetCategories() []string {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.getCategories()
}

// UpdatePayeeCategory adds or updates a payee-to-category mapping
// This is a package-level function that uses the default categorizer
func UpdatePayeeCategory(payee, category string) {
	initOnce.Do(initCategorizer)
	defaultCategorizer.updatePayeeCategory(payee, category)
}

// GetPayeeCategory retrieves the category for a given payee if it exists
// This is a package-level function that uses the default categorizer
func GetPayeeCategory(payee string) (string, bool) {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.getPayeeCategory(payee)
}

// CategorizeTransaction categorizes a transaction using the default categorizer
// This is a package-level function that uses the default categorizer
func CategorizeTransaction(transaction Transaction) (Category, error) {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.categorizeTransaction(transaction)
}

// SavePayeesToYAML saves the current payee mappings to the YAML file
// This is a package-level function that uses the default categorizer
func SavePayeesToYAML() error {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.savePayeesToYAML()
}
