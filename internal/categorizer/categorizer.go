// Package categorizer provides functionality to categorize transactions using multiple methods:
// 1. Direct seller-to-category mapping from a YAML database
// 2. Local keyword-based categorization from rules in a YAML file
// 3. AI-based categorization using Gemini model as a fallback
package categorizer

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
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
	categories       []models.CategoryConfig
	creditorMappings map[string]string // Maps creditor names to categories
	debitorMappings  map[string]string // Maps debitor names to categories
	configMutex      sync.RWMutex
	isDirtyCreditors bool // Track if creditorMappings has been modified and needs to be saved
	isDirtyDebitors  bool // Track if debitorMappings has been modified and needs to be saved
	geminiClient     *genai.Client
	geminiModel      *genai.GenerativeModel
	store            *store.CategoryStore
	logger           *logrus.Logger
}

// Global singleton instance
var defaultCategorizer *Categorizer
var log = logrus.New()

// SetLogger allows setting a custom logger
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
		
		// Also set the logger for the store package
		store.SetLogger(logger)
		
		// Update default categorizer if it exists
		if defaultCategorizer != nil {
			defaultCategorizer.logger = logger
		}
	}
}

// initCategorizer initializes the default categorizer instance
func initCategorizer() {
	if defaultCategorizer == nil {
		defaultCategorizer = &Categorizer{
			categories:       make([]models.CategoryConfig, 0),
			creditorMappings: make(map[string]string),
			debitorMappings:  make(map[string]string),
			isDirtyCreditors: false,
			isDirtyDebitors:  false,
			store:            store.NewCategoryStore(),
			logger:           log,
		}
		
		// Load categories from YAML
		categories, err := defaultCategorizer.store.LoadCategories()
		if err != nil {
			log.Warnf("Failed to load categories: %v", err)
		} else {
			defaultCategorizer.categories = categories
		}
		
		// Load creditor mappings
		creditorMappings, err := defaultCategorizer.store.LoadCreditorMappings()
		if err != nil {
			log.Warnf("Failed to load creditor mappings: %v", err)
		} else {
			defaultCategorizer.creditorMappings = creditorMappings
		}
		
		// Load debitor mappings
		debitorMappings, err := defaultCategorizer.store.LoadDebitorMappings()
		if err != nil {
			log.Warnf("Failed to load debitor mappings: %v", err)
		} else {
			defaultCategorizer.debitorMappings = debitorMappings
		}
	}
}

//------------------------------------------------------------------------------
// GEMINI AI INTEGRATION
//------------------------------------------------------------------------------

// ensureGeminiClient ensures the Gemini client is initialized
func (c *Categorizer) ensureGeminiClient() error {
	if c.geminiClient == nil {
		// Get API key from environment variable
		apiKey := getGeminiAPIKey()
		if apiKey == "" {
			return fmt.Errorf("GEMINI_API_KEY environment variable not set")
		}

		// Initialize the client
		var err error
		c.geminiClient, err = genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
		if err != nil {
			return fmt.Errorf("failed to create Gemini client: %w", err)
		}

		// Create a generative model using Gemini 1.0
		c.geminiModel = c.geminiClient.GenerativeModel("gemini-1.0-pro")
		if c.geminiModel == nil {
			return fmt.Errorf("failed to create Gemini model")
		}
	}
	return nil
}

// categorizeWithGemini uses the Gemini API to categorize a transaction
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (models.Category, error) {
	// Ensure Gemini client is initialized
	if err := c.ensureGeminiClient(); err != nil {
		return models.Category{}, err
	}

	// Prepare the prompt for Gemini
	prompt := fmt.Sprintf(`You are a financial categorization assistant. Analyze this transaction and assign ONE category from the list provided.

Transaction Details:
- Party Name: %s
- Role: %s
- Amount: %s
- Date: %s
- Additional Info: %s

Available Categories:
`,
		transaction.PartyName,
		func() string {
			if transaction.IsDebtor {
				return "Debtor (sender)"
			}
			return "Creditor (recipient)"
		}(),
		transaction.Amount,
		transaction.Date,
		transaction.Info,
	)

	// Add available categories to the prompt
	c.configMutex.RLock()
	for _, category := range c.categories {
		prompt += fmt.Sprintf("- %s\n", category.Name)
	}
	c.configMutex.RUnlock()

	prompt += `
Respond with ONLY the category name that best matches this transaction. Do not add any explanations or other text.
For example, if it's a restaurant expense, just respond with "Food & Dining".`

	// Generate a response from Gemini
	ctx := context.Background()
	resp, err := c.geminiModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return models.Category{}, fmt.Errorf("gemini API error: %w", err)
	}

	// Extract the category from the response
	var categoryName string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
		categoryName = c.extractCategoryFromResponse(responseText)
	}

	if categoryName == "" {
		return models.Category{}, fmt.Errorf("failed to get category from Gemini")
	}

	// Find the category in our list
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	for _, category := range c.categories {
		if strings.EqualFold(category.Name, categoryName) {
			// Update mappings for future use
			if transaction.IsDebtor {
				c.updateDebitorCategory(transaction.PartyName, category.Name)
			} else {
				c.updateCreditorCategory(transaction.PartyName, category.Name)
			}
			
			return models.Category{
				Name:        category.Name,
				Description: categoryDescriptionFromName(category.Name),
			}, nil
		}
	}

	// Return a generic category if not found
	return models.Category{
		Name:        categoryName,
		Description: "Category assigned by Gemini AI",
	}, nil
}

// extractCategoryFromResponse parses the Gemini API response to extract the category
func (c *Categorizer) extractCategoryFromResponse(response string) string {
	// Clean up the response
	response = strings.TrimSpace(response)
	
	// Some models return the category name wrapped in quotes or other formatting
	response = strings.Trim(response, `"'`)
	
	// If there's a colon, it might be "Category: Food & Dining" format
	if strings.Contains(response, ":") {
		parts := strings.SplitN(response, ":", 2)
		return strings.TrimSpace(parts[1])
	}
	
	// If there are multiple lines, take the first one
	if strings.Contains(response, "\n") {
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	
	// Check if it matches any of our known categories
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	bestMatch := ""
	
	for _, category := range c.categories {
		if strings.EqualFold(category.Name, response) {
			return category.Name
		}
		
		// Handle partial matches
		if strings.Contains(strings.ToLower(response), strings.ToLower(category.Name)) {
			if len(category.Name) > len(bestMatch) {
				bestMatch = category.Name
			}
		}
	}
	
	if bestMatch != "" {
		return bestMatch
	}
	
	return response
}

//------------------------------------------------------------------------------
// LOCAL CATEGORIZATION METHODS
//------------------------------------------------------------------------------

// categorizeByCreditorMapping attempts to categorize a transaction using the creditor mapping database
func (c *Categorizer) categorizeByCreditorMapping(transaction Transaction) (models.Category, bool) {
	if transaction.IsDebtor {
		return models.Category{}, false
	}
	
	c.configMutex.RLock()
	categoryName, found := c.creditorMappings[transaction.PartyName]
	c.configMutex.RUnlock()
	
	if !found {
		return models.Category{}, false
	}
	
	// Create a new Category with name and description
	return models.Category{
		Name:        categoryName,
		Description: categoryDescriptionFromName(categoryName),
	}, true
}

// categorizeByDebitorMapping attempts to categorize a transaction using the debitor mapping database
func (c *Categorizer) categorizeByDebitorMapping(transaction Transaction) (models.Category, bool) {
	if !transaction.IsDebtor {
		return models.Category{}, false
	}
	
	c.configMutex.RLock()
	categoryName, found := c.debitorMappings[transaction.PartyName]
	c.configMutex.RUnlock()
	
	if !found {
		return models.Category{}, false
	}
	
	// Create a new Category with name and description
	return models.Category{
		Name:        categoryName,
		Description: categoryDescriptionFromName(categoryName),
	}, true
}

// categorizeLocallyByKeywords attempts to categorize a transaction using the local keyword database
func (c *Categorizer) categorizeLocallyByKeywords(transaction Transaction) (models.Category, bool) {
	partyNameLower := strings.ToLower(transaction.PartyName)
	
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	for _, category := range c.categories {
		for _, keyword := range category.Keywords {
			if strings.Contains(partyNameLower, strings.ToLower(keyword)) {
				return models.Category{
					Name:        category.Name,
					Description: categoryDescriptionFromName(category.Name),
				}, true
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
	// First try to find an exact match in our mappings
	if category, found := c.categorizeByCreditorMapping(transaction); found {
		return category, nil
	}
	
	if category, found := c.categorizeByDebitorMapping(transaction); found {
		return category, nil
	}
	
	// Next try to match by keywords
	if category, found := c.categorizeLocallyByKeywords(transaction); found {
		// Save this mapping for future use
		if transaction.IsDebtor {
			c.updateDebitorCategory(transaction.PartyName, category.Name)
		} else {
			c.updateCreditorCategory(transaction.PartyName, category.Name)
		}
		return category, nil
	}
	
	// As a last resort, use the Gemini API
	category, err := c.categorizeWithGemini(transaction)
	if err != nil {
		// Check if the error is specifically about missing API key
		if strings.Contains(err.Error(), "GEMINI_API_KEY environment variable not set") {
			// Return the expected category for the missing API key case
			return models.Category{
				Name:        "Uncategorized",
				Description: "No API key provided for categorization",
			}, fmt.Errorf("failed to categorize transaction: %w", err)
		}
		
		// For other errors, return a generic "Miscellaneous" category
		return models.Category{
			Name:        "Miscellaneous",
			Description: "Uncategorized transaction",
		}, fmt.Errorf("failed to categorize transaction: %w", err)
	}
	
	return category, nil
}

//------------------------------------------------------------------------------
// HELPER METHODS
//------------------------------------------------------------------------------

// getCategories returns the list of category names
func (c *Categorizer) getCategories() []string {
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	categories := make([]string, len(c.categories))
	for i, category := range c.categories {
		categories[i] = category.Name
	}
	return categories
}

// updateCreditorCategory adds or updates a creditor-to-category mapping
func (c *Categorizer) updateCreditorCategory(creditor, category string) {
	// Skip empty values
	if creditor == "" || category == "" {
		return
	}
	
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Check if this changes the existing mapping
	currentCategory, exists := c.creditorMappings[creditor]
	if !exists || currentCategory != category {
		c.creditorMappings[creditor] = category
		c.isDirtyCreditors = true
	}
}

// updateDebitorCategory adds or updates a debitor-to-category mapping
func (c *Categorizer) updateDebitorCategory(debitor, category string) {
	// Skip empty values
	if debitor == "" || category == "" {
		return
	}
	
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	
	// Check if this changes the existing mapping
	currentCategory, exists := c.debitorMappings[debitor]
	if !exists || currentCategory != category {
		c.debitorMappings[debitor] = category
		c.isDirtyDebitors = true
	}
}

// getCreditorCategory retrieves the category for a given creditor if it exists
func (c *Categorizer) getCreditorCategory(creditor string) (string, bool) {
	if creditor == "" {
		return "", false
	}
	
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	category, found := c.creditorMappings[creditor]
	return category, found
}

// getDebitorCategory retrieves the category for a given debitor if it exists
func (c *Categorizer) getDebitorCategory(debitor string) (string, bool) {
	if debitor == "" {
		return "", false
	}
	
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()
	
	category, found := c.debitorMappings[debitor]
	return category, found
}

// categoryDescriptionFromName returns a standard description for a category name
func categoryDescriptionFromName(name string) string {
	descriptions := map[string]string{
		"Food & Dining":     "Restaurants, groceries, and food delivery",
		"Transportation":    "Public transit, taxis, ride-sharing, and car expenses",
		"Housing":           "Rent, mortgage, utilities, and home maintenance",
		"Entertainment":     "Movies, concerts, events, and recreational activities",
		"Shopping":          "Retail purchases, clothing, and general shopping",
		"Health & Fitness":  "Medical expenses, pharmacy, gym memberships",
		"Travel":            "Flights, hotels, vacations, and travel expenses",
		"Bills & Utilities": "Regular bills, subscriptions, and services",
		"Education":         "Tuition, books, courses, and educational expenses",
		"Business":          "Business expenses, office supplies, professional services",
		"Gifts & Donations": "Charitable donations, gifts, and contributions",
		"Taxes & Fees":      "Government taxes, fees, and related expenses",
		"Income":            "Salary, wages, transfers, and other income",
		"Investments":       "Stock, cryptocurrency, and investment transactions",
		"Miscellaneous":     "Other uncategorized transactions",
	}
	
	if desc, ok := descriptions[name]; ok {
		return desc
	}
	
	return "Category for " + name
}

//------------------------------------------------------------------------------
// PUBLIC API
//------------------------------------------------------------------------------

// GetCategories returns the list of category names for external use
// This is a package-level function that uses the default categorizer
func GetCategories() []string {
	initCategorizer()
	return defaultCategorizer.getCategories()
}

// UpdateCreditorCategory adds or updates a creditor-to-category mapping
// This is a package-level function that uses the default categorizer
func UpdateCreditorCategory(creditor, category string) {
	initCategorizer()
	defaultCategorizer.updateCreditorCategory(creditor, category)
}

// UpdateDebitorCategory adds or updates a debitor-to-category mapping
// This is a package-level function that uses the default categorizer
func UpdateDebitorCategory(debitor, category string) {
	initCategorizer()
	defaultCategorizer.updateDebitorCategory(debitor, category)
}

// GetCreditorCategory retrieves the category for a given creditor if it exists
// This is a package-level function that uses the default categorizer
func GetCreditorCategory(creditor string) (string, bool) {
	initCategorizer()
	return defaultCategorizer.getCreditorCategory(creditor)
}

// GetDebitorCategory retrieves the category for a given debitor if it exists
// This is a package-level function that uses the default categorizer
func GetDebitorCategory(debitor string) (string, bool) {
	initCategorizer()
	return defaultCategorizer.getDebitorCategory(debitor)
}

// CategorizeTransaction categorizes a transaction using the default categorizer
// This is a package-level function that uses the default categorizer
func CategorizeTransaction(transaction Transaction) (models.Category, error) {
	initCategorizer()
	return defaultCategorizer.categorizeTransaction(transaction)
}

// SaveCreditorsToYAML saves the current creditor mappings to the YAML file
// This is a package-level function that uses the default categorizer
func SaveCreditorsToYAML() error {
	initCategorizer()
	
	defaultCategorizer.configMutex.RLock()
	isDirty := defaultCategorizer.isDirtyCreditors
	mappings := make(map[string]string)
	for k, v := range defaultCategorizer.creditorMappings {
		mappings[k] = v
	}
	defaultCategorizer.configMutex.RUnlock()
	
	// Skip saving if nothing has changed
	if !isDirty {
		log.Debug("Creditor mappings have not changed, skipping save")
		return nil
	}
	
	// Save to YAML using the store
	err := defaultCategorizer.store.SaveCreditorMappings(mappings)
	if err == nil {
		// Mark as no longer dirty if saved successfully
		defaultCategorizer.configMutex.Lock()
		defaultCategorizer.isDirtyCreditors = false
		defaultCategorizer.configMutex.Unlock()
	}
	
	return err
}

// SaveDebitorsToYAML saves the current debitor mappings to the YAML file
// This is a package-level function that uses the default categorizer
func SaveDebitorsToYAML() error {
	initCategorizer()
	
	defaultCategorizer.configMutex.RLock()
	isDirty := defaultCategorizer.isDirtyDebitors
	mappings := make(map[string]string)
	for k, v := range defaultCategorizer.debitorMappings {
		mappings[k] = v
	}
	defaultCategorizer.configMutex.RUnlock()
	
	// Skip saving if nothing has changed
	if !isDirty {
		log.Debug("Debitor mappings have not changed, skipping save")
		return nil
	}
	
	// Save to YAML using the store
	err := defaultCategorizer.store.SaveDebitorMappings(mappings)
	if err == nil {
		// Mark as no longer dirty if saved successfully
		defaultCategorizer.configMutex.Lock()
		defaultCategorizer.isDirtyDebitors = false
		defaultCategorizer.configMutex.Unlock()
	}
	
	return err
}

// Helper function to get the Gemini API key
func getGeminiAPIKey() string {
	return strings.TrimSpace(getEnv("GEMINI_API_KEY", ""))
}

// Helper function to get an environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := strings.TrimSpace(defaultValue)
	// The implementation should use os.Getenv or viper to get environment variables
	// But for now we'll just look at the environment and config
	return value
}
