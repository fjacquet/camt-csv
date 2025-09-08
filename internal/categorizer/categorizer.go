// Package categorizer provides functionality to categorize transactions using multiple methods:
// 1. Direct seller-to-category mapping from a YAML database
// 2. Local keyword-based categorization from rules in a YAML file
// 3. AI-based categorization using Gemini model as a fallback
package categorizer

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
	"net/http"
	"bytes"
	"encoding/json"
	"io"
)

//------------------------------------------------------------------------------
// TYPE DEFINITIONS
//------------------------------------------------------------------------------

// Transaction represents a financial transaction to be categorized
type Transaction struct {
	PartyName   string // Name of the relevant party (creditor or debtor)
	IsDebtor    bool   // true if the party is a debtor (sender), false if creditor (recipient)
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
	// Simplified: using direct HTTP calls to Gemini API
	store            *store.CategoryStore
	logger           *logrus.Logger
}

// Global singleton instance - simple approach for a CLI tool
var defaultCategorizer *Categorizer
var log = logging.GetLogger()

// Configuration interface for dependency injection
type Config interface {
	GetAIEnabled() bool
	GetAIAPIKey() string
	GetAIModel() string
	GetAIRequestsPerMinute() int
	GetAITimeoutSeconds() int
	GetAIFallbackCategory() string
	GetCategorizationConfidenceThreshold() float64
}

// SetLogger allows setting a custom logger
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger

		// Update default categorizer if it exists
		if defaultCategorizer != nil {
			defaultCategorizer.logger = logger
		}
	}
}

// initCategorizer initializes the default categorizer instance
func initCategorizer() {
	// Only initialize once
	if defaultCategorizer == nil {
		// Create the categorizer instance with defaults
		defaultCategorizer = &Categorizer{
			categories:       []models.CategoryConfig{},
			creditorMappings: make(map[string]string),
			debitorMappings:  make(map[string]string),
			configMutex:      sync.RWMutex{},
			isDirtyCreditors: false,
			isDirtyDebitors:  false,
			store:            store.NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml"),
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

// SetTestCategoryStore allows tests to inject a test CategoryStore.
// This should only be used in tests!
func SetTestCategoryStore(store *store.CategoryStore) {
	// For test environments, create a fresh test categorizer
	defaultCategorizer = &Categorizer{
		categories:       []models.CategoryConfig{},
		creditorMappings: make(map[string]string),
		debitorMappings:  make(map[string]string),
		configMutex:      sync.RWMutex{},
		isDirtyCreditors: false,
		isDirtyDebitors:  false,
		store:            store,
		logger:           log,
	}

	// Always set test mode in tests to prevent external API calls
	if err := os.Setenv("TEST_MODE", "true"); err != nil {
		// Log error but continue - this is not critical
		log.WithError(err).Warn("Failed to set TEST_MODE environment variable")
	}
}

// CategorizeTransaction categorizes a transaction using the default categorizer
// This is a package-level function that uses the default categorizer
func CategorizeTransaction(transaction Transaction) (models.Category, error) {
	initCategorizer()

	// Safety check
	if defaultCategorizer == nil {
		return models.Category{}, fmt.Errorf("categorizer not initialized")
	}

	// Get the actual categorization
	category, err := defaultCategorizer.categorizeTransaction(transaction)

	// If we successfully found a category via AI, let's immediately save it to the database
	// so we don't need to recategorize similar transactions in the future
	if err == nil && category.Name != "" && category.Name != "Uncategorized" {
		// Auto-learn this categorization by saving it to the appropriate database
		if transaction.IsDebtor {
			defaultCategorizer.logger.Debugf("Auto-learning debitor mapping: '%s' → '%s'",
				transaction.PartyName, category.Name)
			defaultCategorizer.updateDebitorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := defaultCategorizer.saveDebitorsToYAML(); err != nil {
				defaultCategorizer.logger.Warnf("Failed to save debitor mapping: %v", err)
			} else {
				defaultCategorizer.logger.Debugf("Successfully saved new debitor mapping to disk")
			}
		} else {
			defaultCategorizer.logger.Debugf("Auto-learning creditor mapping: '%s' → '%s'",
				transaction.PartyName, category.Name)
			defaultCategorizer.updateCreditorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := defaultCategorizer.saveCreditorsToYAML(); err != nil {
				defaultCategorizer.logger.Warnf("Failed to save creditor mapping: %v", err)
			} else {
				defaultCategorizer.logger.Debugf("Successfully saved new creditor mapping to disk")
			}
		}
	}

	return category, err
}

// Public method for the Categorizer struct
func (c *Categorizer) CategorizeTransaction(transaction Transaction) (models.Category, error) {
	return c.categorizeTransaction(transaction)
}

//------------------------------------------------------------------------------
// GEMINI AI INTEGRATION
//------------------------------------------------------------------------------

// GeminiRequest represents the API request structure
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiResponse represents the API response structure
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Error      *GeminiAPIError   `json:"error,omitempty"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type GeminiAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// initAI initializes AI categorization (simplified)
func (c *Categorizer) initAI() error {
	apiKey := getGeminiAPIKey()
	if apiKey == "" {
		c.logger.Warn("AI categorization disabled: GEMINI_API_KEY not set")
		return nil
	}

	c.logger.Info("AI categorization enabled with Gemini via HTTP API")
	return nil
}

// Function to create a prompt for AI categorization
func (c *Categorizer) createCategorizationPrompt(transaction Transaction) string {
	c.logger.Info("======== CREATING CATEGORIZATION PROMPT - IMPROVED FUNCTION CALLED ========")

	// Create a prompt for the Gemini API
	prompt := "Categorize the following financial transaction into EXACTLY ONE of the allowed categories.\n\n"
	prompt += fmt.Sprintf("Description: %s\n", transaction.Description)
	prompt += fmt.Sprintf("Party: %s\n", transaction.PartyName)
	prompt += fmt.Sprintf("Amount: %s\n", transaction.Amount)
	prompt += fmt.Sprintf("Type: %s\n", func() string {
		if transaction.IsDebtor {
			return "Debit (spending)"
		}
		return "Credit (income)"
	}())

	prompt += "\nYou MUST choose ONE category from the following list EXACTLY as written:\n"

	// Get the list of categories
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()

	// Extract valid category names for easier validation later
	validCategories := make([]string, 0, len(c.categories))
	c.logger.Infof("Number of categories loaded: %d", len(c.categories))

	for _, category := range c.categories {
		prompt += fmt.Sprintf("- %s\n", category.Name)
		validCategories = append(validCategories, category.Name)
	}

	// Add merchant categories from hard-coded map as additional options
	prompt += "\nThese are the ONLY valid responses. Do not make up new categories or modify these names.\n"
	prompt += "Do NOT return 'categories', 'category', 'unknown', or any other text not in the above list.\n"
	prompt += "If you cannot categorize the transaction, use 'Uncategorized'.\n\n"
	prompt += "Respond with ONLY the category name that best matches this transaction. Do not include any explanations, punctuation, or other text.\n"
	prompt += "For example, if it's a restaurant expense, just respond with 'Restaurants' (assuming it's in the list)."

	c.logger.Infof("Prompt: %s", prompt)
	return prompt
}

// callGeminiAPI makes a direct HTTP call to Gemini API
func (c *Categorizer) callGeminiAPI(prompt string) (*GeminiResponse, error) {
	apiKey := getGeminiAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	// Prepare request
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	modelName := getGeminiModelName()
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Warnf("Failed to close response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("API error %d: %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	return &geminiResp, nil
}

// categorizeWithGemini attempts to categorize a transaction using the Gemini API
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (models.Category, error) {
	apiKey := getGeminiAPIKey()
	if apiKey == "" {
		return models.Category{
			Name:        "Uncategorized",
			Description: "AI categorization not available",
		}, nil
	}

	// Create a prompt for AI categorization
	prompt := c.createCategorizationPrompt(transaction)
	c.logger.Infof("Gemini request model: %s, prompt length: %d", getGeminiModelName(), len(prompt))

	// Wait if needed to respect the rate limit before making a Gemini API call
	waitForRateLimit(c.logger)

	// Make HTTP request to Gemini API
	resp, err := c.callGeminiAPI(prompt)
	if err != nil {
		c.logger.Warnf("Gemini API error for transaction %s: %v", transaction.PartyName, err)
		return models.Category{}, fmt.Errorf("gemini API error: %w", err)
	}

	// Extract the category from the response
	var categoryName string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		responseText := resp.Candidates[0].Content.Parts[0].Text
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
		response = strings.TrimSpace(parts[1])
	}

	// If there are multiple lines, take the first one
	if strings.Contains(response, "\n") {
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				response = trimmed
				break
			}
		}
	}

	// Explicitly reject problematic responses
	loweredResponse := strings.ToLower(response)
	if loweredResponse == "categories" || loweredResponse == "category" ||
		loweredResponse == "unknown" || loweredResponse == "none" {
		c.logger.Warnf("Rejecting invalid category: '%s'", response)
		return ""
	}

	// Check if it matches any of our known categories
	c.configMutex.RLock()
	defer c.configMutex.RUnlock()

	// First try exact case-insensitive match (preferred)
	for _, category := range c.categories {
		if strings.EqualFold(category.Name, response) {
			c.logger.Debugf("Found exact match for '%s': '%s'", response, category.Name)
			return category.Name // Return with correct casing from database
		}
	}

	// If no exact match, try partial match as fallback
	bestMatch := ""
	for _, category := range c.categories {
		if strings.Contains(strings.ToLower(response), strings.ToLower(category.Name)) {
			if len(category.Name) > len(bestMatch) {
				bestMatch = category.Name
			}
		}
	}

	if bestMatch != "" {
		c.logger.Debugf("Found partial match for '%s': '%s'", response, bestMatch)
		return bestMatch
	}

	// Return empty string if no match found
	c.logger.Warnf("No category match found for '%s'", response)
	return ""
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
		"VIRT BANC": "Virements",
		"VIR TWINT": "Virements",
		"VIRT":      "Virements",
		"TRANSFERT": "Virements",
		"ORDRE LSV": "Virements",
		"TWINT":     "Virements", // CR TWINT
		"CR TWINT":  "Virements",
		"BCV-NET":   "Virements",
		"TRANSFER":  "Virements",
		"IMPOTS":    "Virements",
		"FLORENCE":  "Virements", // Common family transfers
		"JACQUET":   "Virements", // Common family transfers

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
		"PIZZERIA":    "Restaurants",
		"CAFE":        "Restaurants",
		"RESTAURANT":  "Restaurants",
		"SUSHI":       "Restaurants",
		"KEBAB":       "Restaurants",
		"BOUCHERIE":   "Alimentation",
		"BOULANGERIE": "Alimentation",
		"RAMEN":       "Restaurants",
		"KAMUY":       "Restaurants",
		"MINESTRONE":  "Restaurants",

		// Leisure
		"PISCINE":  "Loisirs",
		"SPA":      "Loisirs",
		"CINEMA":   "Loisirs",
		"PILATUS":  "Loisirs",
		"TOTEM":    "Sport",
		"ESCALADE": "Sport",

		// Shops
		"OCHSNER":       "Shopping",
		"SPORT":         "Shopping",
		"MAMMUT":        "Shopping",
		"MULLER":        "Shopping",
		"BAZAR":         "Shopping",
		"INTERDISCOUNT": "Shopping",
		"IKEA":          "Mobilier",
		"PAYOT":         "Shopping",
		"CALIDA":        "Shopping",
		"DIGITAL":       "Shopping",
		"RHB":           "Shopping",
		"WEBSHOP":       "Shopping",
		"POST":          "Shopping",

		// Services
		"PRESSING":     "Services",
		"777-PRESSING": "Services",
		"5ASEC":        "Services",

		// Cash withdrawals
		"RETRAIT":    "Retraits",
		"ATM":        "Retraits",
		"WITHDRAWAL": "Retraits",

		// Utilities and telecom
		"ROMANDE ENERGIE": "Utilités",
		"WINGO":           "Services",

		// Transportation
		"SBB":        "Transports Publics",
		"CFF":        "Transports Publics",
		"MOBILITY":   "Voiture",
		"PAYBYPHONE": "Transports Publics",

		// Insurance
		"ASSURANCE": "Assurances",
		"VAUDOISE":  "Assurances",
		"GENERALI":  "Assurances",
		"AVENIR":    "Assurance Maladie",

		// Financial
		"SELMA_FEE": "Frais Bancaires",
		"CEMBRAPAY": "Services",
		"VISECA":    "Abonnements",

		// Housing
		"PUBLICA": "Logement",
		"ASLOCA":  "Logement",

		// Companies
		"DELL": "Virements",
	}

	// Match bank transaction codes to categories
	txCodeCategories := map[string]string{
		"CWDL": "Retraits",  // Cash withdrawals
		"POSD": "Shopping",  // Point of sale debit
		"CCRD": "Shopping",  // Credit card payment
		"ICDT": "Virements", // Internal credit transfer
		"DMCT": "Virements", // Direct debit
		"RDDT": "Virements", // Direct debit
		"AUTT": "Virements", // Automatic transfer
		"RCDT": "Virements", // Received credit transfer
		"PMNT": "Virements", // Payment
		"RMDR": "Services",  // Reminders
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
// 3. Fall back to Gemini AI for classification if enabled and not in test mode
func (c *Categorizer) categorizeTransaction(transaction Transaction) (models.Category, error) {
	if c.logger != nil {
		c.logger.WithFields(logrus.Fields{
			"party":    transaction.PartyName,
			"amount":   transaction.Amount,
			"date":     transaction.Date,
			"isDebtor": transaction.IsDebtor,
		}).Debug("Categorizing transaction")
	}

	// Skip categorization for empty party names
	if transaction.PartyName == "" {
		return models.Category{
			Name:        "Uncategorized",
			Description: "No party name provided",
		}, nil
	}

	// 1. Check creditor/debitor mappings first (exact matches)
	if transaction.IsDebtor {
		if category, found := c.categorizeByDebitorMapping(transaction); found {
			return category, nil
		}
	} else {
		if category, found := c.categorizeByCreditorMapping(transaction); found {
			return category, nil
		}
	}

	// 2. Try local keyword-based categorization
	if category, found := c.categorizeLocallyByKeywords(transaction); found {
		return category, nil
	}

	// 3. Skip API calls in test mode regardless of AI categorization setting
	isTestMode := os.Getenv("TEST_MODE") == "true"
	if isTestMode {
		return models.Category{
			Name:        "Uncategorized",
			Description: "Test mode - AI categorization skipped",
		}, nil
	}

	// 4. Use AI categorization only if enabled
	if isAICategorizeEnabled() {
		// Check if we have an API key configured
		apiKey := getGeminiAPIKey()
		if apiKey == "" {
			if c.logger != nil {
				c.logger.Debug("AI categorization disabled: GEMINI_API_KEY not set")
			}
			return models.Category{
				Name:        "Uncategorized",
				Description: "AI categorization not available",
			}, nil
		}

		// Attempt AI categorization
		category, err := c.categorizeWithGemini(transaction)
		if err != nil {
			if c.logger != nil {
				c.logger.WithError(err).Error("Failed to categorize with Gemini AI")
			}
			return models.Category{
				Name:        "Uncategorized",
				Description: "Transaction could not be categorized by AI",
			}, nil
		}

		// Automatically save successful AI categorizations to database
		// This ensures that future classifications of the same transaction are faster
		if category.Name != "" && category.Name != "Uncategorized" {
			if transaction.IsDebtor {
				c.logger.Debugf("Auto-learning debitor mapping for '%s' → '%s'",
					transaction.PartyName, category.Name)
				c.updateDebitorCategory(transaction.PartyName, category.Name)

				// Force save to disk immediately
				if err := c.saveDebitorsToYAML(); err != nil {
					c.logger.Warnf("Failed to save debitor mapping: %v", err)
				} else {
					c.logger.Debugf("Successfully saved debitor mapping to YAML")
				}
			} else {
				c.logger.Debugf("Auto-learning creditor mapping for '%s' → '%s'",
					transaction.PartyName, category.Name)
				c.updateCreditorCategory(transaction.PartyName, category.Name)

				// Force save to disk immediately
				if err := c.saveCreditorsToYAML(); err != nil {
					c.logger.Warnf("Failed to save creditor mapping: %v", err)
				} else {
					c.logger.Debugf("Successfully saved creditor mapping to YAML")
				}
			}
		}

		return category, nil
	}

	// Default category if all methods fail
	return models.Category{
		Name:        "Uncategorized",
		Description: "Transaction could not be categorized with available methods",
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

	// Only mark as dirty if the mapping actually changes
	if existingCategory, exists := c.creditorMappings[creditorLower]; !exists || existingCategory != categoryName {
		c.creditorMappings[creditorLower] = categoryName
		c.isDirtyCreditors = true
		c.logger.Debugf("Creditor mapping changed, marked as dirty: %s -> %s", creditorLower, categoryName)
	}
}

// updateDebitorCategory adds or updates a debitor-to-category mapping
func (c *Categorizer) updateDebitorCategory(debitor, categoryName string) {
	// Convert debitor name to lowercase for consistency
	debitorLower := strings.ToLower(debitor)

	c.configMutex.Lock()
	defer c.configMutex.Unlock()

	// Only mark as dirty if the mapping actually changes
	if existingCategory, exists := c.debitorMappings[debitorLower]; !exists || existingCategory != categoryName {
		c.debitorMappings[debitorLower] = categoryName
		c.isDirtyDebitors = true
		c.logger.Debugf("Debitor mapping changed, marked as dirty: %s -> %s", debitorLower, categoryName)
	}
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

// SaveCreditorsToYAML saves the current creditor mappings to the YAML file
// This is a package-level function that uses the default categorizer
func SaveCreditorsToYAML() error {
	initCategorizer()
	return defaultCategorizer.saveCreditorsToYAML()
}

// SaveDebitorsToYAML saves the current debitor mappings to the YAML file
// This is a package-level function that uses the default categorizer
func SaveDebitorsToYAML() error {
	initCategorizer()
	return defaultCategorizer.saveDebitorsToYAML()
}

// saveCreditorsToYAML saves the current creditor mappings to the YAML file
func (c *Categorizer) saveCreditorsToYAML() error {
	c.configMutex.RLock()
	mappings := make(map[string]string)
	for k, v := range c.creditorMappings {
		mappings[k] = v
	}
	c.configMutex.RUnlock()

	// Always save the mappings when explicitly requested (added for automated learning)
	c.logger.Debug("Saving creditor mappings to YAML file")

	if c.store == nil {
		return fmt.Errorf("store is not initialized")
	}

	if err := c.store.SaveCreditorMappings(mappings); err != nil {
		return fmt.Errorf("error saving creditor mappings: %w", err)
	}

	c.configMutex.Lock()
	c.isDirtyCreditors = false
	c.configMutex.Unlock()

	return nil
}

// saveDebitorsToYAML saves the current debitor mappings to the YAML file
func (c *Categorizer) saveDebitorsToYAML() error {
	c.configMutex.RLock()
	mappings := make(map[string]string)
	for k, v := range c.debitorMappings {
		mappings[k] = v
	}
	c.configMutex.RUnlock()

	// Always save the mappings when explicitly requested (added for automated learning)
	c.logger.Debug("Saving debitor mappings to YAML file")

	if c.store == nil {
		return fmt.Errorf("store is not initialized")
	}

	if err := c.store.SaveDebitorMappings(mappings); err != nil {
		return fmt.Errorf("error saving debitor mappings: %w", err)
	}

	c.configMutex.Lock()
	c.isDirtyDebitors = false
	c.configMutex.Unlock()

	return nil
}

//------------------------------------------------------------------------------
// HELPER FUNCTIONS
//------------------------------------------------------------------------------

// Helper function to get the Gemini API key
func getGeminiAPIKey() string {
	// Try to get from centralized config first
	if cfg := config.GetGlobalConfig(); cfg != nil {
		return cfg.AI.APIKey
	}
	// Fallback to environment variable for backward compatibility
	return strings.TrimSpace(getEnv("GEMINI_API_KEY", ""))
}

// Helper function to get the Gemini model name
func getGeminiModelName() string {
	// Try to get from centralized config first
	if cfg := config.GetGlobalConfig(); cfg != nil {
		return cfg.AI.Model
	}
	// Fallback to environment variable for backward compatibility
	return strings.TrimSpace(getEnv("GEMINI_MODEL", "gemini-2.5-flash"))
}

// Helper function to get the Gemini API rate limit (requests per minute)
func getGeminiRateLimit() int {
	// Try to get from centralized config first
	if cfg := config.GetGlobalConfig(); cfg != nil {
		return cfg.AI.RequestsPerMinute
	}
	// Fallback to environment variable for backward compatibility
	limitStr := getEnv("GEMINI_REQUESTS_PER_MINUTE", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return 10 // Default to 10 if parsing fails or value is invalid
	}
	return limit
}

// Helper function to check if AI categorization is enabled
func isAICategorizeEnabled() bool {
	// Try to get from centralized config first
	if cfg := config.GetGlobalConfig(); cfg != nil {
		return cfg.AI.Enabled
	}
	// Fallback to environment variable for backward compatibility
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

// WithStore is an option for NewCategorizer that sets a custom store
func WithStore(store *store.CategoryStore) func(*Categorizer) {
	return func(c *Categorizer) {
		c.store = store
	}
}

// WithLogger is an option for NewCategorizer that sets a custom logger
func WithLogger(logger *logrus.Logger) func(*Categorizer) {
	return func(c *Categorizer) {
		c.logger = logger
	}
}
