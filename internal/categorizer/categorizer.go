// Package categorizer provides functionality to categorize transactions using multiple methods:
// 1. Direct seller-to-category mapping from a YAML database
// 2. Local keyword-based categorization from rules in a YAML file
// 3. AI-based categorization using Gemini model as a fallback
package categorizer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
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
	store            *store.CategoryStore
	logger           *logrus.Logger
	aiClient         AIClient // New field for AIClient interface
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

// NewCategorizer creates a new instance of Categorizer with the given AIClient, CategoryStore, and logger.
func NewCategorizer(aiClient AIClient, store *store.CategoryStore, logger *logrus.Logger) *Categorizer {
	if logger == nil {
		logger = logging.GetLogger()
	}

	c := &Categorizer{
		categories:       []models.CategoryConfig{},
		creditorMappings: make(map[string]string),
		debitorMappings:  make(map[string]string),
		configMutex:      sync.RWMutex{},
		isDirtyCreditors: false,
		isDirtyDebitors:  false,
		store:            store,
		logger:           logger,
		aiClient:         aiClient,
	}

	// Load categories from YAML
	categories, err := c.store.LoadCategories()
	if err != nil {
		c.logger.Warnf("Failed to load categories: %v", err)
	} else {
		c.categories = categories
	}

	// Load creditor mappings
	creditorMappings, err := c.store.LoadCreditorMappings()
	if err != nil {
		c.logger.Warnf("Failed to load creditor mappings: %v", err)
	} else {
		// Normalize keys to lowercase for case-insensitive lookup
		for key, value := range creditorMappings {
			c.creditorMappings[strings.ToLower(key)] = value
		}
	}

	// Load debitor mappings
	debitorMappings, err := c.store.LoadDebitorMappings()
	if err != nil {
		c.logger.Warnf("Failed to load debitor mappings: %v", err)
	} else {
		// Normalize keys to lowercase for case-insensitive lookup
		for key, value := range debitorMappings {
			c.debitorMappings[strings.ToLower(key)] = value
		}
	}

	return c
}

// initCategorizer initializes the default categorizer instance
func initCategorizer() {
	// Only initialize once
	if defaultCategorizer == nil {
		// Create a default GeminiClient and CategoryStore
		defaultLogger := logging.GetLogger()
		defaultAIClient := NewGeminiClient(defaultLogger) // Assuming NewGeminiClient exists
		defaultStore := store.NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml")

		defaultCategorizer = NewCategorizer(defaultAIClient, defaultStore, defaultLogger)
	}
}

// SetTestCategoryStore allows tests to inject a test CategoryStore.
// This should only be used in tests!
func SetTestCategoryStore(store *store.CategoryStore) {
	// If store is nil, reset to default categorizer
	if store == nil {
		defaultCategorizer = nil
		return
	}

	// For test environments, create a fresh test categorizer
	defaultLogger := logging.GetLogger()
	defaultAIClient := NewGeminiClient(defaultLogger) // Assuming NewGeminiClient exists
	defaultCategorizer = NewCategorizer(defaultAIClient, store, defaultLogger)

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
	category, err := defaultCategorizer.CategorizeTransaction(transaction)

	// If we successfully found a category via AI, let's immediately save it to the database
	// so we don't need to recategorize similar transactions in the future
	if err == nil && category.Name != "" && category.Name != "Uncategorized" {
		// Auto-learn this categorization by saving it to the appropriate database
		if transaction.IsDebtor {
			defaultCategorizer.logger.Debugf("Auto-learning debitor mapping: '%s' → '%s'",
				transaction.PartyName, category.Name)
			defaultCategorizer.updateDebitorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := defaultCategorizer.SaveDebitorsToYAML(); err != nil {
				defaultCategorizer.logger.Warnf("Failed to save debitor mapping: %v", err)
			} else {
				defaultCategorizer.logger.Debugf("Successfully saved new debitor mapping to disk")
			}
		} else {
			defaultCategorizer.logger.Debugf("Auto-learning creditor mapping: '%s' → '%s'",
				transaction.PartyName, category.Name)
			defaultCategorizer.updateCreditorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := defaultCategorizer.SaveCreditorsToYAML(); err != nil {
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

// private method for the Categorizer struct
func (c *Categorizer) categorizeTransaction(transaction Transaction) (models.Category, error) {
	// If party name is empty, return uncategorized immediately
	if strings.TrimSpace(transaction.PartyName) == "" {
		return models.Category{
			Name:        "Uncategorized",
			Description: "No party name provided",
		}, nil
	}

	// Step 1: Try to categorize by debitor mapping
	if category, found := c.categorizeByDebitorMapping(transaction); found {
		return category, nil
	}

	// Step 2: Try to categorize by creditor mapping
	if category, found := c.categorizeByCreditorMapping(transaction); found {
		return category, nil
	}

	// Step 3: Try to categorize by local keywords
	if category, found := c.categorizeLocallyByKeywords(transaction); found {
		return category, nil
	}

	// Step 4: Fallback to AI categorization
	return c.categorizeWithGemini(transaction)
}

func categoryDescriptionFromName(name string) string {
	// In a real-world scenario, you would look up the description from a database
	return "Description for " + name
}

func (c *Categorizer) updateDebitorCategory(partyName, categoryName string) {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	c.debitorMappings[strings.ToLower(partyName)] = categoryName
	c.isDirtyDebitors = true
}

func (c *Categorizer) SaveDebitorsToYAML() error {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	if !c.isDirtyDebitors {
		return nil
	}
	if err := c.store.SaveDebitorMappings(c.debitorMappings); err != nil {
		return err
	}
	c.isDirtyDebitors = false
	return nil
}

func (c *Categorizer) updateCreditorCategory(partyName, categoryName string) {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	c.creditorMappings[strings.ToLower(partyName)] = categoryName
	c.isDirtyCreditors = true
}

func (c *Categorizer) SaveCreditorsToYAML() error {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	if !c.isDirtyCreditors {
		return nil
	}
	if err := c.store.SaveCreditorMappings(c.creditorMappings); err != nil {
		return err
	}
	c.isDirtyCreditors = false
	return nil
}

func UpdateDebitorCategory(partyName, categoryName string) {
	initCategorizer()
	defaultCategorizer.updateDebitorCategory(partyName, categoryName)
}

func UpdateCreditorCategory(partyName, categoryName string) {
	initCategorizer()
	defaultCategorizer.updateCreditorCategory(partyName, categoryName)
}

// categorizeWithGemini attempts to categorize a transaction using the AI client
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (models.Category, error) {
	// If no AI client is available, return uncategorized
	if c.aiClient == nil {
		return models.Category{
			Name:        "Uncategorized",
			Description: "AI categorization not available",
		}, nil
	}

	// Convert Transaction to models.Transaction
	modelTransaction := models.Transaction{
		PartyName:   transaction.PartyName,
		Description: transaction.Description,
		Amount:      models.ParseAmount(transaction.Amount),
		Date:        transaction.Date,
		Category:    "", // Will be filled by AI
	}

	// Use the AI client to categorize
	ctx := context.Background()
	categorizedTransaction, err := c.aiClient.Categorize(ctx, modelTransaction)
	if err != nil {
		c.logger.Warnf("AI categorization error for transaction %s: %v", transaction.PartyName, err)
		return models.Category{
			Name:        "Uncategorized",
			Description: "AI categorization failed",
		}, nil
	}

	// Return the category from the AI response
	return models.Category{
		Name:        categorizedTransaction.Category,
		Description: categoryDescriptionFromName(categorizedTransaction.Category),
	}, nil
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
