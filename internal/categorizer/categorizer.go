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
	"time"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"os"

	"strconv"

	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

//------------------------------------------------------------------------------
// TYPE DEFINITIONS
//------------------------------------------------------------------------------

// Transaction represents a financial transaction to be categorized
type Transaction struct {
	PartyName   string // Name of the relevant party (creditor or debitor)
	IsDebtor    bool   // true if the party is a debitor (sender), false if creditor (recipient)
	Amount      string
	Date        string
	Info        string
	Description string
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

// initGeminiClient initializes the Gemini API client
func (c *Categorizer) initGeminiClient() error {
	apiKey := getGeminiAPIKey()
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	// Create a new GenAI client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create GenAI client: %w", err)
	}
	c.geminiClient = client

	// Get model name from environment
	modelName := getGeminiModelName()
	c.logger.Infof("Initializing Gemini with model: %s", modelName)

	// Create a generative model using configured model
	c.geminiModel = c.geminiClient.GenerativeModel(modelName)
	if c.geminiModel == nil {
		return fmt.Errorf("failed to create Gemini model")
	}

	// NOTE: We're not setting any safety settings as they are not compatible
	// across different Gemini model versions

	return nil
}

// Function to create a prompt for AI categorization
func (c *Categorizer) createCategorizationPrompt(transaction Transaction) string {
	// Create a prompt for the Gemini API
	prompt := "Categorize the following transaction:\n"
	prompt += fmt.Sprintf("Description: %s\n", transaction.Description)
	prompt += fmt.Sprintf("Party: %s\n", transaction.PartyName)
	prompt += fmt.Sprintf("Amount: %s\n", transaction.Amount)
	prompt += fmt.Sprintf("Type: %s\n", func() string {
		if transaction.IsDebtor {
			return "Debit (spending)"
		}
		return "Credit (income)"
	}())

	prompt += "\nYou MUST choose ONE category from the following list:\n"

	// Get the list of categories
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()

	for _, category := range c.categories {
		prompt += fmt.Sprintf("- %s\n", category.Name)
	}

	prompt += `
Respond with ONLY the category name that best matches this transaction. Do not add any explanations or other text.
For example, if it's a restaurant expense, just respond with "Food & Dining".`

	return prompt
}

// categorizeWithGemini attempts to categorize a transaction using the Gemini API
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (models.Category, error) {
	if c.geminiClient == nil || c.geminiModel == nil {
		// Initialize Gemini client if not already initialized
		if err := c.initGeminiClient(); err != nil {
			return models.Category{}, err
		}
	}

	// Create a prompt for AI categorization
	prompt := c.createCategorizationPrompt(transaction)
	c.logger.Infof("Gemini request model: %s, prompt length: %d", getGeminiModelName(), len(prompt))

	// Wait if needed to respect the rate limit before making a Gemini API call
	waitForRateLimit(c.logger)

	// Generate a response from Gemini
	// Create a context with a timeout (5 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.geminiModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			c.logger.Warnf("Gemini API timed out after 5 seconds for transaction: %s", transaction.PartyName)
			return models.Category{}, fmt.Errorf("gemini API timed out after 5 seconds")
		}
		c.logger.Warnf("Gemini API error for transaction %s: %v", transaction.PartyName, err)
		return models.Category{}, fmt.Errorf("gemini API error: %w", err)
	}

	// Extract the category from the response
	var categoryName string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
		categoryName = c.extractCategoryFromResponse(responseText)
		c.logger.Infof("Gemini response for '%s': '%s' extracted as '%s'",
			transaction.PartyName, responseText, categoryName)
	} else {
		c.logger.Warnf("Empty response from Gemini for transaction: %s", transaction.PartyName)
	}

	if categoryName == "" {
		return models.Category{}, fmt.Errorf("failed to get category from Gemini")
	}

	// Find the category in our list
	for _, category := range c.categories {
		if strings.EqualFold(category.Name, categoryName) {
			// Create a new Category with name and description
			return models.Category{
				Name:        category.Name, // Use the exact case from our database
				Description: categoryDescriptionFromName(category.Name),
			}, nil
		}
	}

	// If we get here, the category wasn't found in our list
	return models.Category{}, fmt.Errorf("category '%s' not found in database", categoryName)
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

	// Convert to lowercase for case-insensitive lookup
	partyNameLower := strings.ToLower(transaction.PartyName)

	c.configMutex.RLock()
	categoryName, found := c.creditorMappings[partyNameLower]
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

	// Convert to lowercase for case-insensitive lookup
	partyNameLower := strings.ToLower(transaction.PartyName)

	c.configMutex.RLock()
	categoryName, found := c.debitorMappings[partyNameLower]
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
	partyName := strings.ToUpper(transaction.PartyName)
	description := strings.ToUpper(transaction.Info)

	// Maps of keyword patterns to categories
	// These are ordered from most specific to most general
	merchantCategories := map[string]string{
		// Transfers & Banking - add additional keywords to detect transfers
		"VIRT BANC":   "Virements",
		"VIR TWINT":   "Virements",
		"VIRT":        "Virements",
		"TRANSFERT":   "Virements",
		"ORDRE LSV":   "Virements",
		"TWINT":       "Virements",   // CR TWINT
		"CR TWINT":    "Virements", 
		"BCV-NET":     "Virements",
		"TRANSFER":    "Virements",
		"IMPOTS":      "Virements",
		"FLORENCE":    "Virements",   // Common family transfers
		"JACQUET":     "Virements",   // Common family transfers
		
		// Supermarkets
		"COOP":       "Alimentation",
		"MIGROS":     "Alimentation",
		"ALDI":       "Alimentation",
		"LIDL":       "Alimentation",
		"DENNER":     "Alimentation",
		"MANOR":      "Shopping",
		"MIGROLINO":  "Alimentation",
		"KIOSK":      "Alimentation",
		"MINERALOEL": "Alimentation",
		
		// Restaurants & Food
		"PIZZERIA":   "Restaurants",
		"CAFE":       "Restaurants",
		"RESTAURANT": "Restaurants",
		"SUSHI":      "Restaurants",
		"KEBAB":      "Restaurants",
		"BOUCHERIE": "Alimentation",
		"BOULANGERIE": "Alimentation",
		"RAMEN":      "Restaurants",
		"KAMUY":      "Restaurants",
		"MINESTRONE": "Restaurants",
		
		// Leisure
		"PISCINE":    "Loisirs",
		"SPA":        "Loisirs",
		"CINEMA":     "Loisirs",
		"PILATUS":    "Loisirs",
		"TOTEM":      "Sport",
		"ESCALADE":   "Sport",
		
		// Shops
		"OCHSNER":    "Shopping",
		"SPORT":      "Shopping",
		"MAMMUT":     "Shopping",
		"MULLER":     "Shopping", 
		"BAZAR":      "Shopping",
		"INTERDISCOUNT": "Shopping",
		"IKEA":       "Mobilier",
		"PAYOT":      "Shopping",
		"CALIDA":     "Shopping",
		"DIGITAL":    "Shopping",
		"RHB":        "Shopping",
		"WEBSHOP":    "Shopping",
		"POST":       "Shopping",
		
		// Services
		"PRESSING":   "Services",
		"777-PRESSING": "Services",
		"5ASEC":      "Services",
		
		// Cash withdrawals
		"RETRAIT":    "Retraits",
		"ATM":        "Retraits",
		"WITHDRAWAL": "Retraits",
		
		// Utilities and telecom
		"ROMANDE ENERGIE": "Utilités",
		"WINGO":      "Services",
		
		// Transportation
		"SBB":        "Transports Publics",
		"CFF":        "Transports Publics",
		"MOBILITY":   "Voiture",
		"PAYBYPHONE": "Transports Publics",
		
		// Insurance
		"ASSURANCE":  "Assurances",
		"VAUDOISE":   "Assurances",
		"GENERALI":   "Assurances",
		"AVENIR":     "Assurance Maladie",
		
		// Financial
		"SELMA_FEE":  "Frais Bancaires",
		"CEMBRAPAY":  "Services",
		"VISECA":     "Abonnements",
		
		// Housing
		"PUBLICA":    "Logement",
		"ASLOCA":     "Logement",
		
		// Companies
		"DELL":       "Virements",
	}

	// Match bank transaction codes to categories
	txCodeCategories := map[string]string{
		"CWDL":       "Retraits",        // Cash withdrawals
		"POSD":       "Shopping",        // Point of sale debit
		"CCRD":       "Shopping",        // Credit card payment
		"ICDT":       "Virements",       // Internal credit transfer
		"DMCT":       "Virements",       // Direct debit
		"RDDT":       "Virements",       // Direct debit
		"AUTT":       "Virements",       // Automatic transfer
		"RCDT":       "Virements",       // Received credit transfer
		"PMNT":       "Virements",       // Payment
		"RMDR":       "Services",        // Reminders
	}

	// First try to match merchant name
	for keyword, category := range merchantCategories {
		if strings.Contains(partyName, keyword) || strings.Contains(description, keyword) {
			return models.Category{
				Name:        category,
				Description: categoryDescriptionFromName(category),
			}, true
		}
	}

	// Try to detect transaction types from bank codes
	for bankCode, category := range txCodeCategories {
		// Check if bank code appears in transaction info
		if strings.Contains(transaction.Info, bankCode) {
			return models.Category{
				Name:        category,
				Description: categoryDescriptionFromName(category),
			}, true
		}
	}

	// Look for credit cards and cash withdrawals
	if transaction.IsDebtor {
		// Special case for Unknown Payee for card payments - don't default to Salaire
		if strings.Contains(partyName, "UNKNOWN PAYEE") || partyName == "UNKNOWN PAYEE" {
			// If it looks like a card payment or cash withdrawal
			if strings.Contains(description, "PMT CARTE") || 
			   strings.Contains(description, "PMT TWINT") ||
			   strings.Contains(description, "RETRAIT") ||
			   strings.Contains(description, "WITHDRAWAL") {
				return models.Category{
					Name:        "Shopping",
					Description: categoryDescriptionFromName("Shopping"),
				}, true
			}
		}
	}

	// No match found
	return models.Category{}, false
}

// categorizeTransaction categorizes a transaction using the following sequence:
// 1. Check if the creditor/debitor already exists in the mapping database
// 2. Try to match using local keyword patterns
// 3. Fall back to Gemini AI for classification if enabled
func (c *Categorizer) categorizeTransaction(transaction Transaction) (models.Category, error) {
	// First try to find an exact match in our mappings
	if category, found := c.categorizeByCreditorMapping(transaction); found {
		c.logger.Debugf("Transaction categorized from creditor mapping: %s -> %s",
			transaction.PartyName, category.Name)
		return category, nil
	}

	if category, found := c.categorizeByDebitorMapping(transaction); found {
		c.logger.Debugf("Transaction categorized from debitor mapping: %s -> %s",
			transaction.PartyName, category.Name)
		return category, nil
	}

	// Next try to match by keywords
	if category, found := c.categorizeLocallyByKeywords(transaction); found {
		c.logger.Debugf("Transaction categorized by keywords: %s -> %s",
			transaction.PartyName, category.Name)

		// Save this mapping for future use
		if transaction.IsDebtor {
			c.updateDebitorCategory(transaction.PartyName, category.Name)
		} else {
			c.updateCreditorCategory(transaction.PartyName, category.Name)
		}
		return category, nil
	}

	// Only try AI categorization if explicitly enabled AND no matches were found above
	if isAICategorizeEnabled() {
		category, err := c.categorizeWithGemini(transaction)
		if err != nil {
			c.logger.Warnf("AI categorization failed for transaction %s: %v, using Uncategorized instead",
				transaction.PartyName, err)

			// Just return Miscellaneous without propagating the error
			return models.Category{
				Name:        "Uncategorized",
				Description: "Uncategorized transaction",
			}, nil
		}

		// If Gemini returned a valid category, update the database for future use
		c.logger.Infof("Transaction categorized by Gemini AI: %s -> %s",
			transaction.PartyName, category.Name)

		// Save this mapping for future use
		if transaction.IsDebtor {
			c.updateDebitorCategory(transaction.PartyName, category.Name)
		} else {
			c.updateCreditorCategory(transaction.PartyName, category.Name)
		}

		return category, nil
	}

	// Return a generic "Uncategorized" category
	c.logger.Debugf("No category found for transaction: %s", transaction.PartyName)
	return models.Category{
		Name:        "Uncategorized",
		Description: "Uncategorized transaction",
	}, nil
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
func (c *Categorizer) updateCreditorCategory(creditor, categoryName string) {
	// Convert creditor name to lowercase for consistency
	creditorLower := strings.ToLower(creditor)

	c.configMutex.Lock()
	defer c.configMutex.Unlock()

	c.creditorMappings[creditorLower] = categoryName
	c.isDirtyCreditors = true
}

// updateDebitorCategory adds or updates a debitor-to-category mapping
func (c *Categorizer) updateDebitorCategory(debitor, categoryName string) {
	// Convert debitor name to lowercase for consistency
	debitorLower := strings.ToLower(debitor)

	c.configMutex.Lock()
	defer c.configMutex.Unlock()

	c.debitorMappings[debitorLower] = categoryName
	c.isDirtyDebitors = true
}

// getCreditorCategory retrieves the category for a given creditor if it exists
func (c *Categorizer) getCreditorCategory(creditor string) (string, bool) {
	if creditor == "" {
		return "", false
	}

	c.configMutex.RLock()
	defer c.configMutex.RUnlock()

	category, found := c.creditorMappings[strings.ToLower(creditor)]
	return category, found
}

// getDebitorCategory retrieves the category for a given debitor if it exists
func (c *Categorizer) getDebitorCategory(debitor string) (string, bool) {
	if debitor == "" {
		return "", false
	}

	c.configMutex.RLock()
	defer c.configMutex.RUnlock()

	category, found := c.debitorMappings[strings.ToLower(debitor)]
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
		"Uncategorized":     "Other uncategorized transactions",
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

// Helper function to get the Gemini model name
func getGeminiModelName() string {
	// Default to gemini-2.0-flash if not specified
	return strings.TrimSpace(getEnv("GEMINI_MODEL", "gemini-2.0-flash"))
}

// Helper function to get the Gemini API rate limit (requests per minute)
func getGeminiRateLimit() int {
	// Default to 10 requests per minute if not specified
	limitStr := getEnv("GEMINI_REQUESTS_PER_MINUTE", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return 10 // Default to 10 if parsing fails or value is invalid
	}
	return limit
}

// Helper function to check if AI categorization is enabled
func isAICategorizeEnabled() bool {
	value := strings.ToLower(getEnv("USE_AI_CATEGORIZATION", "false"))
	return value == "true" || value == "1" || value == "yes"
}

// Rate limiter for Gemini API calls
var (
	geminiLastCallTime time.Time
	geminiMutex        sync.Mutex
)

// Wait if needed to respect the rate limit before making a Gemini API call
func waitForRateLimit(logger *logrus.Logger) {
	geminiMutex.Lock()
	defer geminiMutex.Unlock()

	// Calculate minimum time between API calls in milliseconds
	requestsPerMinute := getGeminiRateLimit()
	minTimeBetweenCalls := time.Duration(60000/requestsPerMinute) * time.Millisecond

	// Calculate time since last call
	timeSinceLastCall := time.Since(geminiLastCallTime)

	// If we need to wait to respect rate limit
	if timeSinceLastCall < minTimeBetweenCalls && !geminiLastCallTime.IsZero() {
		waitTime := minTimeBetweenCalls - timeSinceLastCall
		if logger != nil {
			logger.Infof("Rate limiting: waiting %dms before next Gemini API call", waitTime.Milliseconds())
		}
		time.Sleep(waitTime)
	}

	// Update last call time
	geminiLastCallTime = time.Now()
}

// Helper function to get an environment variable with a default value
func getEnv(key, defaultValue string) string {
	// Get the value from environment variables
	value := os.Getenv(key)

	// If empty, use the default value
	if value == "" {
		value = defaultValue
	}

	return strings.TrimSpace(value)
}
