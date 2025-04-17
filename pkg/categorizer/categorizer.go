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

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/pkg/config"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

//------------------------------------------------------------------------------
// TYPE DEFINITIONS
//------------------------------------------------------------------------------

// Transaction represents a financial transaction to be categorized
type Transaction struct {
	PartyName string // Name of the relevant party (creditor or debitor)
	IsDebtor  bool   // true if the party is a debitor (sender), false if creditor (recipient)
	Amount    string
	Date      string
	Info      string
}

// Categorizer handles the categorization of transactions and manages
// the category, creditor, and debitor mapping databases
type Categorizer struct {
	categories      []models.CategoryConfig
	creditorMappings map[string]string // Maps creditor names to categories
	debitorMappings  map[string]string // Maps debitor names to categories
	configMutex    sync.RWMutex
	isDirtyCreditors bool // Track if creditorMappings has been modified and needs to be saved
	isDirtyDebitors  bool // Track if debitorMappings has been modified and needs to be saved
	geminiClient   *genai.Client
	geminiModel    *genai.GenerativeModel
}

// Global singleton instance
var defaultCategorizer *Categorizer
var initOnce sync.Once

//------------------------------------------------------------------------------
// INITIALIZATION
//------------------------------------------------------------------------------

// initCategorizer initializes the default categorizer instance
func initCategorizer() {
	defaultCategorizer = &Categorizer{
		creditorMappings: make(map[string]string),
		debitorMappings: make(map[string]string),
	}
	
	// Load categories from YAML
	err := defaultCategorizer.loadCategoriesFromYAML()
	if err != nil {
		log.Printf("Warning: Could not load categories from YAML: %v", err)
		log.Printf("Falling back to default categories")
		
		// Create default categories if YAML loading fails
		defaultCategorizer.categories = []models.CategoryConfig{
			{Name: "Food & Dining", Keywords: []string{"restaurant", "food", "dining", "cafe"}},
			{Name: "Groceries", Keywords: []string{"supermarket", "grocery", "coop", "migros"}},
			{Name: "Shopping", Keywords: []string{"shop", "store", "retail", "amazon"}},
			{Name: "Utilities", Keywords: []string{"electric", "water", "utility", "bill"}},
			{Name: "Transportation", Keywords: []string{"transport", "train", "bus", "taxi"}},
			{Name: "Uncategorized", Keywords: []string{"unknown", "other"}},
		}
	}
	
	// Load mappings from YAML files
	// First try to load from old payees.yaml for backward compatibility
	errPayees := defaultCategorizer.migrateFromPayeesYAML()
	if errPayees != nil {
		log.Printf("Warning: Could not migrate from payees.yaml: %v", errPayees)
	}
	
	// Load creditor mappings from YAML
	errCreditors := defaultCategorizer.loadCreditorsFromYAML()
	if errCreditors != nil {
		log.Printf("Warning: Could not load creditor mappings from YAML: %v", errCreditors)
	}
	
	// Load debitor mappings from YAML
	errDebitors := defaultCategorizer.loadDebitorsFromYAML()
	if errDebitors != nil {
		log.Printf("Warning: Could not load debitor mappings from YAML: %v", errDebitors)
	}
	
	if errPayees != nil && errCreditors != nil && errDebitors != nil {
		log.Printf("Starting with empty party mappings database")
	}
}

//------------------------------------------------------------------------------
// GEMINI AI INTEGRATION
//------------------------------------------------------------------------------

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

// categorizeWithGemini uses the Gemini API to categorize a transaction
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (models.Category, error) {
	// Initialize Gemini client if not already initialized
	if err := c.ensureGeminiClient(); err != nil {
		return models.Category{}, fmt.Errorf("failed to initialize Gemini client: %w", err)
	}
	
	// Get available categories to provide to Gemini
	categories := c.getCategories()
	
	// Create a prompt for Gemini
	prompt := fmt.Sprintf(`You are a financial transaction categorizer.
Please categorize the following transaction into the most appropriate category from the list below:

Transaction Details:
- Party Name: %s
- Is Debtor: %t
- Amount: %s
- Date: %s
- Description: %s

Available Categories:
%s

Please respond ONLY with the category name that best matches this transaction.
If you're unsure or if none of the categories seem to fit, respond with "Uncategorized".
Please respond in this exact format: "Category: [category name]"`, 
		transaction.PartyName, 
		transaction.IsDebtor, 
		transaction.Amount, 
		transaction.Date, 
		transaction.Info,
		strings.Join(categories, "\n"))
	
	// Send the prompt to Gemini API
	ctx := context.Background()
	resp, err := c.geminiModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return models.Category{}, fmt.Errorf("error generating content with Gemini: %w", err)
	}
	
	// Extract the category from the response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return models.Category{}, fmt.Errorf("no response from Gemini model")
	}
	
	responseText := resp.Candidates[0].Content.Parts[0].(genai.Text)
	categoryName, description := c.extractCategoryFromResponse(string(responseText))
	
	// Default to "Uncategorized" if no category was found
	if categoryName == "" {
		categoryName = "Uncategorized"
	}
	
	// Add the new mapping to our database for future use
	if transaction.IsDebtor {
		c.updateDebitorCategory(transaction.PartyName, categoryName)
	} else {
		c.updateCreditorCategory(transaction.PartyName, categoryName)
	}
	
	return models.Category{Name: categoryName, Description: description}, nil
}

// extractCategoryFromResponse parses the Gemini API response to extract the category
func (c *Categorizer) extractCategoryFromResponse(response string) (string, string) {
	// Look for "Category:" in the response
	responseLower := strings.ToLower(response)
	categoryIndex := strings.Index(responseLower, "category:")
	
	if categoryIndex == -1 {
		// If "Category:" not found, try to extract the category from the entire response
		words := strings.Fields(response)
		if len(words) > 0 {
			// Check each word against our known categories
			for _, word := range words {
				word = strings.Trim(word, ",.;:\"'()")
				for _, category := range c.categories {
					if strings.EqualFold(word, category.Name) {
						return category.Name, "Extracted from unformatted response"
					}
				}
			}
		}
		return "Uncategorized", "Could not parse response"
	}
	
	// Extract everything after "Category:"
	afterCategory := response[categoryIndex+len("Category:"):]
	
	// Trim whitespace and extract the category name
	categoryName := strings.TrimSpace(afterCategory)
	
	// If there's more text, extract only up to the first newline or period
	endIndex := len(categoryName)
	if nl := strings.Index(categoryName, "\n"); nl != -1 {
		endIndex = nl
	}
	if period := strings.Index(categoryName, "."); period != -1 && period < endIndex {
		endIndex = period
	}
	categoryName = strings.TrimSpace(categoryName[:endIndex])
	
	return categoryName, "Extracted from Gemini response"
}

//------------------------------------------------------------------------------
// LOCAL CATEGORIZATION METHODS
//------------------------------------------------------------------------------

// categorizeByCreditorMapping attempts to categorize a transaction using the creditor mapping database
func (c *Categorizer) categorizeByCreditorMapping(transaction Transaction) (models.Category, bool) {
	// Check if we have the creditor in our mapping database
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	// Normalize creditor name for better matching
	normalizedCreditor := strings.ToLower(strings.TrimSpace(transaction.PartyName))
	
	// Try to find an exact match first
	if category, exists := c.creditorMappings[normalizedCreditor]; exists {
		return models.Category{Name: category, Description: ""}, true
	}
	
	// If no exact match, try to find a substring match
	for creditor, category := range c.creditorMappings {
		if strings.Contains(normalizedCreditor, strings.ToLower(creditor)) {
			return models.Category{Name: category, Description: ""}, true
		}
	}
	
	return models.Category{}, false
}

// categorizeByDebitorMapping attempts to categorize a transaction using the debitor mapping database
func (c *Categorizer) categorizeByDebitorMapping(transaction Transaction) (models.Category, bool) {
	// Check if we have the debitor in our mapping database
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	// Normalize debitor name for better matching
	normalizedDebitor := strings.ToLower(strings.TrimSpace(transaction.PartyName))
	
	// Try to find an exact match first
	if category, exists := c.debitorMappings[normalizedDebitor]; exists {
		return models.Category{Name: category, Description: ""}, true
	}
	
	// If no exact match, try to find a substring match
	for debitor, category := range c.debitorMappings {
		if strings.Contains(normalizedDebitor, strings.ToLower(debitor)) {
			return models.Category{Name: category, Description: ""}, true
		}
	}
	
	return models.Category{}, false
}

// categorizeLocallyByKeywords attempts to categorize a transaction using the local keyword database
func (c *Categorizer) categorizeLocallyByKeywords(transaction Transaction) (models.Category, bool) {
	// Normalize description and party name for better matching
	normalizedText := strings.ToLower(strings.TrimSpace(transaction.PartyName + " " + transaction.Info))
	
	// For each category, check if any of its keywords match the transaction
	for _, category := range c.categories {
		for _, keyword := range category.Keywords {
			if strings.Contains(normalizedText, strings.ToLower(keyword)) {
				return models.Category{Name: category.Name, Description: ""}, true
			}
		}
	}
	
	return models.Category{}, false
}

// categorizeTransaction categorizes a transaction using the following sequence:
// 1. Check if the creditor/debitor already exists in the mapping database
// 2. Try to match using local keyword patterns
// 3. Fall back to Gemini AI as a last resort
func (c *Categorizer) categorizeTransaction(transaction Transaction) (models.Category, error) {
	// 1. Check if the creditor/debitor already exists in our mapping database
	if transaction.IsDebtor {
		if category, found := c.categorizeByDebitorMapping(transaction); found {
			log.Printf("Transaction categorized by debitor mapping as: %s", category.Name)
			return category, nil
		}
	} else {
		if category, found := c.categorizeByCreditorMapping(transaction); found {
			log.Printf("Transaction categorized by creditor mapping as: %s", category.Name)
			return category, nil
		}
	}
	
	// 2. Try to categorize using local keyword matching
	if category, found := c.categorizeLocallyByKeywords(transaction); found {
		log.Printf("Transaction categorized by keyword matching as: %s", category.Name)
		// If found by keywords, add to our mapping database for future quick lookups
		if transaction.PartyName != "" {
			if transaction.IsDebtor {
				c.updateDebitorCategory(transaction.PartyName, category.Name)
			} else {
				c.updateCreditorCategory(transaction.PartyName, category.Name)
			}
		}
		return category, nil
	}
	
	// 3. Fall back to AI-based categorization if all else fails
	log.Printf("No local match found, using Gemini AI for categorization")
	category, err := c.categorizeWithGemini(transaction)
	
	// Even if AI categorization fails, add the party to mappings with "Uncategorized"
	if err != nil && transaction.PartyName != "" {
		category = models.Category{Name: "Uncategorized", Description: ""}
		if transaction.IsDebtor {
			c.updateDebitorCategory(transaction.PartyName, category.Name)
		} else {
			c.updateCreditorCategory(transaction.PartyName, category.Name)
		}
	}
	
	return category, err
}

//------------------------------------------------------------------------------
// YAML CONFIG HANDLING
//------------------------------------------------------------------------------

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
	
	var config models.CategoriesConfig
	err = yaml.Unmarshal(yamlData, &config)
	if err != nil {
		return fmt.Errorf("could not parse categories.yaml file: %w", err)
	}
	
	// Store the loaded categories
	c.categories = config.Categories
	log.Printf("Loaded %d categories from YAML file", len(c.categories))
	
	return nil
}

// loadCreditorsFromYAML loads the creditor-to-category mappings from the YAML configuration file
func (c *Categorizer) loadCreditorsFromYAML() error {
	yamlPath, err := c.findConfigFile("creditors.yaml")
	if err != nil {
		// If the file doesn't exist yet, this is not an error
		// We'll create an empty map and save it later
		c.configMutex.Lock()
		c.creditorMappings = make(map[string]string)
		c.isDirtyCreditors = true  // Mark as dirty so it gets saved
		c.configMutex.Unlock()
		log.Printf("Creditor mappings file not found, will create it on next save")
		return nil
	}
	
	// Read and parse the YAML file
	yamlData, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("could not read creditors.yaml file: %w", err)
	}
	
	// First, try to unmarshal with the expected structure (with 'creditors' key)
	var config models.CreditorsConfig
	err = yaml.Unmarshal(yamlData, &config)
	if err != nil {
		// Try parsing directly as a map (without the 'creditors' key)
		directMap := make(map[string]string)
		err = yaml.Unmarshal(yamlData, &directMap)
		if err != nil {
			return fmt.Errorf("could not parse creditors.yaml file: %w", err)
		}
		
		// If successful, use the direct map
		c.configMutex.Lock()
		c.creditorMappings = directMap
		c.isDirtyCreditors = false
		c.configMutex.Unlock()
		log.Printf("Loaded %d creditor mappings from YAML file (direct format)", len(directMap))
		return nil
	}
	
	// Acquire write lock before updating the map
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Store the loaded creditor mappings
	if config.Creditors != nil {
		c.creditorMappings = config.Creditors
		log.Printf("Loaded %d creditor mappings from YAML file", len(c.creditorMappings))
	} else {
		c.creditorMappings = make(map[string]string)
		log.Printf("Initialized empty creditor mappings")
	}
	
	c.isDirtyCreditors = false
	return nil
}

// loadDebitorsFromYAML loads the debitor-to-category mappings from the YAML configuration file
func (c *Categorizer) loadDebitorsFromYAML() error {
	yamlPath, err := c.findConfigFile("debitors.yaml")
	if err != nil {
		// If the file doesn't exist yet, this is not an error
		// We'll create an empty map and save it later
		c.configMutex.Lock()
		c.debitorMappings = make(map[string]string)
		c.isDirtyDebitors = true  // Mark as dirty so it gets saved
		c.configMutex.Unlock()
		log.Printf("Debitor mappings file not found, will create it on next save")
		return nil
	}
	
	// Read and parse the YAML file
	yamlData, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("could not read debitors.yaml file: %w", err)
	}
	
	// First, try to unmarshal with the expected structure (with 'debitors' key)
	var config models.DebitorsConfig
	err = yaml.Unmarshal(yamlData, &config)
	if err != nil {
		// Try parsing directly as a map (without the 'debitors' key)
		directMap := make(map[string]string)
		err = yaml.Unmarshal(yamlData, &directMap)
		if err != nil {
			return fmt.Errorf("could not parse debitors.yaml file: %w", err)
		}
		
		// If successful, use the direct map
		c.configMutex.Lock()
		c.debitorMappings = directMap
		c.isDirtyDebitors = false
		c.configMutex.Unlock()
		log.Printf("Loaded %d debitor mappings from YAML file (direct format)", len(directMap))
		return nil
	}
	
	// Acquire write lock before updating the map
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Store the loaded debitor mappings
	if config.Debitors != nil {
		c.debitorMappings = config.Debitors
		log.Printf("Loaded %d debitor mappings from YAML file", len(c.debitorMappings))
	} else {
		c.debitorMappings = make(map[string]string)
		log.Printf("Initialized empty debitor mappings")
	}
	
	c.isDirtyDebitors = false
	return nil
}

// saveCreditorsToYAML saves the current creditor mappings to the YAML file
// This should be called at the end of processing or when the application exits
func (c *Categorizer) saveCreditorsToYAML() error {
	// If nothing has changed, no need to save
	c.configMutex.RLock()
	if !c.isDirtyCreditors {
		c.configMutex.RUnlock()
		return nil
	}
	c.configMutex.RUnlock()
	
	// Try to find existing creditors.yaml file, or decide where to create it
	yamlPath, err := c.findConfigFile("creditors.yaml")
	if err != nil {
		// File doesn't exist, create it in the default location
		defaultDir := filepath.Join(".", "database")
		
		// Ensure the directory exists
		err = os.MkdirAll(defaultDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
		
		yamlPath = filepath.Join(defaultDir, "creditors.yaml")
	}
	
	// Acquire read lock to create a copy of the map
	c.configMutex.RLock()
	config := models.CreditorsConfig{
		Creditors: make(map[string]string, len(c.creditorMappings)),
	}
	
	// Create a copy of the map to avoid holding the lock during disk I/O
	for k, v := range c.creditorMappings {
		config.Creditors[k] = v
	}
	c.configMutex.RUnlock()
	
	// Marshal the data to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal creditor mappings to YAML: %w", err)
	}
	
	// Write to file
	err = ioutil.WriteFile(yamlPath, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write creditors.yaml file: %w", err)
	}
	
	// Mark as not dirty after successful save
	c.configMutex.Lock()
	c.isDirtyCreditors = false
	c.configMutex.Unlock()
	
	log.Printf("Saved %d creditor mappings to %s", len(config.Creditors), yamlPath)
	return nil
}

// saveDebitorsToYAML saves the current debitor mappings to the YAML file
// This should be called at the end of processing or when the application exits
func (c *Categorizer) saveDebitorsToYAML() error {
	// If nothing has changed, no need to save
	c.configMutex.RLock()
	if !c.isDirtyDebitors {
		c.configMutex.RUnlock()
		return nil
	}
	c.configMutex.RUnlock()
	
	// Try to find existing debitors.yaml file, or decide where to create it
	yamlPath, err := c.findConfigFile("debitors.yaml")
	if err != nil {
		// File doesn't exist, create it in the default location
		defaultDir := filepath.Join(".", "database")
		
		// Ensure the directory exists
		err = os.MkdirAll(defaultDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
		
		yamlPath = filepath.Join(defaultDir, "debitors.yaml")
	}
	
	// Acquire read lock to create a copy of the map
	c.configMutex.RLock()
	config := models.DebitorsConfig{
		Debitors: make(map[string]string, len(c.debitorMappings)),
	}
	
	// Create a copy of the map to avoid holding the lock during disk I/O
	for k, v := range c.debitorMappings {
		config.Debitors[k] = v
	}
	c.configMutex.RUnlock()
	
	// Marshal the data to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal debitor mappings to YAML: %w", err)
	}
	
	// Write to file
	err = ioutil.WriteFile(yamlPath, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write debitors.yaml file: %w", err)
	}
	
	// Mark as not dirty after successful save
	c.configMutex.Lock()
	c.isDirtyDebitors = false
	c.configMutex.Unlock()
	
	log.Printf("Saved %d debitor mappings to %s", len(config.Debitors), yamlPath)
	return nil
}

// migrateFromPayeesYAML attempts to migrate from the old payees.yaml file to the new creditors.yaml and debitors.yaml files
func (c *Categorizer) migrateFromPayeesYAML() error {
	yamlPath, err := c.findConfigFile("payees.yaml")
	if err != nil {
		// File doesn't exist, nothing to migrate
		return nil
	}
	
	// Read and parse the YAML file
	yamlData, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("could not read payees.yaml file: %w", err)
	}
	
	// First, try to unmarshal with the expected structure (with 'payees' key)
	var config models.PayeesConfig
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
		c.creditorMappings = directMap
		c.debitorMappings = directMap
		c.isDirtyCreditors = true
		c.isDirtyDebitors = true
		c.configMutex.Unlock()
		log.Printf("Migrated %d payee mappings from YAML file (direct format)", len(directMap))
		return nil
	}
	
	// Acquire write lock before updating the maps
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Store the loaded payee mappings
	if config.Payees != nil {
		c.creditorMappings = config.Payees
		c.debitorMappings = config.Payees
		log.Printf("Migrated %d payee mappings from YAML file", len(c.creditorMappings))
	} else {
		c.creditorMappings = make(map[string]string)
		c.debitorMappings = make(map[string]string)
		log.Printf("Initialized empty creditor and debitor mappings")
	}
	
	c.isDirtyCreditors = true
	c.isDirtyDebitors = true
	return nil
}

// findConfigFile attempts to locate a configuration file in various paths
func (c *Categorizer) findConfigFile(filename string) (string, error) {
	// First check in the current directory
	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}
	
	// Check in database/ subdirectory
	dbPath := filepath.Join("database", filename)
	if _, err := os.Stat(dbPath); err == nil {
		return dbPath, nil
	}
	
	// Check in parent directory
	parentPath := filepath.Join("..", filename)
	if _, err := os.Stat(parentPath); err == nil {
		return parentPath, nil
	}
	
	return "", fmt.Errorf("could not find %s in any of the expected locations", filename)
}

//------------------------------------------------------------------------------
// HELPER METHODS
//------------------------------------------------------------------------------

// getCategories returns the list of category names
func (c *Categorizer) getCategories() []string {
	categories := make([]string, len(c.categories))
	for i, category := range c.categories {
		categories[i] = category.Name
	}
	return categories
}

// updateCreditorCategory adds or updates a creditor-to-category mapping
func (c *Categorizer) updateCreditorCategory(creditor, category string) {
	if creditor == "" || category == "" {
		return
	}
	
	// Normalize creditor name for consistent storage
	normalizedCreditor := strings.ToLower(strings.TrimSpace(creditor))
	
	// Acquire write lock before updating the map
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Check if this is a new mapping or if we're changing an existing one
	existingCategory, exists := c.creditorMappings[normalizedCreditor]
	if !exists || existingCategory != category {
		c.creditorMappings[normalizedCreditor] = category
		c.isDirtyCreditors = true  // Mark as dirty so it gets saved
	}
}

// updateDebitorCategory adds or updates a debitor-to-category mapping
func (c *Categorizer) updateDebitorCategory(debitor, category string) {
	if debitor == "" || category == "" {
		return
	}
	
	// Normalize debitor name for consistent storage
	normalizedDebitor := strings.ToLower(strings.TrimSpace(debitor))
	
	// Acquire write lock before updating the map
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Check if this is a new mapping or if we're changing an existing one
	existingCategory, exists := c.debitorMappings[normalizedDebitor]
	if !exists || existingCategory != category {
		c.debitorMappings[normalizedDebitor] = category
		c.isDirtyDebitors = true  // Mark as dirty so it gets saved
	}
}

// getCreditorCategory retrieves the category for a given creditor if it exists
func (c *Categorizer) getCreditorCategory(creditor string) (string, bool) {
	if creditor == "" {
		return "", false
	}
	
	// Normalize creditor name for consistent lookup
	normalizedCreditor := strings.ToLower(strings.TrimSpace(creditor))
	
	// Acquire read lock before accessing the map
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	category, exists := c.creditorMappings[normalizedCreditor]
	return category, exists
}

// getDebitorCategory retrieves the category for a given debitor if it exists
func (c *Categorizer) getDebitorCategory(debitor string) (string, bool) {
	if debitor == "" {
		return "", false
	}
	
	// Normalize debitor name for consistent lookup
	normalizedDebitor := strings.ToLower(strings.TrimSpace(debitor))
	
	// Acquire read lock before accessing the map
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	category, exists := c.debitorMappings[normalizedDebitor]
	return category, exists
}

//------------------------------------------------------------------------------
// PUBLIC API
//------------------------------------------------------------------------------

// GetCategories returns the list of category names for external use
// This is a package-level function that uses the default categorizer
func GetCategories() []string {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.getCategories()
}

// UpdateCreditorCategory adds or updates a creditor-to-category mapping
// This is a package-level function that uses the default categorizer
func UpdateCreditorCategory(creditor, category string) {
	initOnce.Do(initCategorizer)
	defaultCategorizer.updateCreditorCategory(creditor, category)
}

// UpdateDebitorCategory adds or updates a debitor-to-category mapping
// This is a package-level function that uses the default categorizer
func UpdateDebitorCategory(debitor, category string) {
	initOnce.Do(initCategorizer)
	defaultCategorizer.updateDebitorCategory(debitor, category)
}

// GetCreditorCategory retrieves the category for a given creditor if it exists
// This is a package-level function that uses the default categorizer
func GetCreditorCategory(creditor string) (string, bool) {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.getCreditorCategory(creditor)
}

// GetDebitorCategory retrieves the category for a given debitor if it exists
// This is a package-level function that uses the default categorizer
func GetDebitorCategory(debitor string) (string, bool) {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.getDebitorCategory(debitor)
}

// CategorizeTransaction categorizes a transaction using the default categorizer
// This is a package-level function that uses the default categorizer
func CategorizeTransaction(transaction Transaction) (models.Category, error) {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.categorizeTransaction(transaction)
}

// SaveCreditorsToYAML saves the current creditor mappings to the YAML file
// This is a package-level function that uses the default categorizer
func SaveCreditorsToYAML() error {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.saveCreditorsToYAML()
}

// SaveDebitorsToYAML saves the current debitor mappings to the YAML file
// This is a package-level function that uses the default categorizer
func SaveDebitorsToYAML() error {
	initOnce.Do(initCategorizer)
	return defaultCategorizer.saveDebitorsToYAML()
}
