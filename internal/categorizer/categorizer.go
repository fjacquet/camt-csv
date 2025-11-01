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
	logger           logging.Logger
	aiClient         AIClient // New field for AIClient interface
}

// Global singleton instance - simple approach for a CLI tool
// Deprecated: Use dependency injection with NewCategorizer instead.
// This will be removed in v2.0.0.
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
func SetLogger(logger logging.Logger) {
	if logger != nil {
		log = logger

		// Update default categorizer if it exists
		if defaultCategorizer != nil {
			defaultCategorizer.logger = logging.NewLogrusAdapterFromLogger(logrus.New())
		}
	}
}

// NewCategorizer creates a new instance of Categorizer with the given AIClient, CategoryStore, and logger.
func NewCategorizer(aiClient AIClient, store *store.CategoryStore, logger logging.Logger) *Categorizer {
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
		c.logger.WithError(err).Warn("Failed to load categories")
	} else {
		c.categories = categories
	}

	// Load creditor mappings
	creditorMappings, err := c.store.LoadCreditorMappings()
	if err != nil {
		c.logger.WithError(err).Warn("Failed to load creditor mappings")
	} else {
		// Normalize keys to lowercase for case-insensitive lookup
		for key, value := range creditorMappings {
			c.creditorMappings[strings.ToLower(key)] = value
		}
	}

	// Load debitor mappings
	debitorMappings, err := c.store.LoadDebitorMappings()
	if err != nil {
		c.logger.WithError(err).Warn("Failed to load debitor mappings")
	} else {
		// Normalize keys to lowercase for case-insensitive lookup
		for key, value := range debitorMappings {
			c.debitorMappings[strings.ToLower(key)] = value
		}
	}

	return c
}

// initCategorizer initializes the default categorizer instance
// Deprecated: Use dependency injection with NewCategorizer instead.
// This function will be removed in v2.0.0.
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
// Deprecated: Use NewCategorizer with dependency injection instead.
// This function will be removed in v2.0.0.
//
// Migration example:
//   // Old way (deprecated)
//   category, err := categorizer.CategorizeTransaction(transaction)
//
//   // New way (recommended)
//   container, err := container.NewContainer(config)
//   if err != nil {
//       log.Fatal(err)
//   }
//   category, err := container.Categorizer.CategorizeTransaction(transaction)
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
	if err == nil && category.Name != "" && category.Name != models.CategoryUncategorized {
		// Auto-learn this categorization by saving it to the appropriate database
		if transaction.IsDebtor {
			defaultCategorizer.logger.WithFields(
				logging.Field{Key: "party", Value: transaction.PartyName},
				logging.Field{Key: "category", Value: category.Name},
			).Debug("Auto-learning debitor mapping")
			defaultCategorizer.updateDebitorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := defaultCategorizer.SaveDebitorsToYAML(); err != nil {
				defaultCategorizer.logger.WithError(err).Warn("Failed to save debitor mapping")
			} else {
				defaultCategorizer.logger.Debug("Successfully saved new debitor mapping to disk")
			}
		} else {
			defaultCategorizer.logger.WithFields(
				logging.Field{Key: "party", Value: transaction.PartyName},
				logging.Field{Key: "category", Value: category.Name},
			).Debug("Auto-learning creditor mapping")
			defaultCategorizer.updateCreditorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := defaultCategorizer.SaveCreditorsToYAML(); err != nil {
				defaultCategorizer.logger.WithError(err).Warn("Failed to save creditor mapping")
			} else {
				defaultCategorizer.logger.Debug("Successfully saved new creditor mapping to disk")
			}
		}
	}

	return category, err
}

// CategorizeTransactionWithCategorizer categorizes a transaction using the provided categorizer instance.
// This function provides a migration path from the global singleton pattern to dependency injection.
//
// Parameters:
//   - categorizer: The categorizer instance to use
//   - transaction: The transaction to categorize
//
// Returns:
//   - models.Category: The assigned category
//   - error: Any error encountered during categorization
//
// Example usage:
//   container, err := container.NewContainer(config)
//   if err != nil {
//       return err
//   }
//   category, err := categorizer.CategorizeTransactionWithCategorizer(container.Categorizer, transaction)
func CategorizeTransactionWithCategorizer(cat *Categorizer, transaction Transaction) (models.Category, error) {
	if cat == nil {
		return models.Category{}, fmt.Errorf("categorizer cannot be nil")
	}
	
	// Get the actual categorization
	category, err := cat.CategorizeTransaction(transaction)

	// If we successfully found a category via AI, let's immediately save it to the database
	// so we don't need to recategorize similar transactions in the future
	if err == nil && category.Name != "" && category.Name != models.CategoryUncategorized {
		// Auto-learn this categorization by saving it to the appropriate database
		if transaction.IsDebtor {
			cat.logger.WithFields(
				logging.Field{Key: "party", Value: transaction.PartyName},
				logging.Field{Key: "category", Value: category.Name},
			).Debug("Auto-learning debitor mapping")
			cat.updateDebitorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := cat.SaveDebitorsToYAML(); err != nil {
				cat.logger.WithError(err).Warn("Failed to save debitor mapping")
			} else {
				cat.logger.Debug("Successfully saved new debitor mapping to disk")
			}
		} else {
			cat.logger.WithFields(
				logging.Field{Key: "party", Value: transaction.PartyName},
				logging.Field{Key: "category", Value: category.Name},
			).Debug("Auto-learning creditor mapping")
			cat.updateCreditorCategory(transaction.PartyName, category.Name)
			// Force immediate save to disk
			if err := cat.SaveCreditorsToYAML(); err != nil {
				cat.logger.WithError(err).Warn("Failed to save creditor mapping")
			} else {
				cat.logger.Debug("Successfully saved new creditor mapping to disk")
			}
		}
	}

	return category, err
}

// CategorizeTransaction categorizes a transaction using this categorizer instance.
// This is the preferred method for dependency injection.
//
// Parameters:
//   - transaction: The transaction to categorize
//
// Returns:
//   - models.Category: The assigned category
//   - error: Any error encountered during categorization
//
// Example usage:
//   container, err := container.NewContainer(config)
//   if err != nil {
//       return err
//   }
//   category, err := container.Categorizer.CategorizeTransaction(transaction)
func (c *Categorizer) CategorizeTransaction(transaction Transaction) (models.Category, error) {
	return c.categorizeTransaction(transaction)
}

// private method for the Categorizer struct
func (c *Categorizer) categorizeTransaction(transaction Transaction) (models.Category, error) {
	// If party name is empty, return uncategorized immediately
	if strings.TrimSpace(transaction.PartyName) == "" {
		return models.Category{
			Name:        models.CategoryUncategorized,
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

// UpdateDebitorCategory updates a debitor category mapping for this categorizer instance.
func (c *Categorizer) UpdateDebitorCategory(partyName, categoryName string) {
	c.updateDebitorCategory(partyName, categoryName)
}

func (c *Categorizer) updateDebitorCategory(partyName, categoryName string) {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	c.debitorMappings[strings.ToLower(partyName)] = categoryName
	c.isDirtyDebitors = true
}

// SaveDebitorsToYAML saves debitor mappings to YAML file if they have been modified.
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

// UpdateCreditorCategory updates a creditor category mapping for this categorizer instance.
func (c *Categorizer) UpdateCreditorCategory(partyName, categoryName string) {
	c.updateCreditorCategory(partyName, categoryName)
}

func (c *Categorizer) updateCreditorCategory(partyName, categoryName string) {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	c.creditorMappings[strings.ToLower(partyName)] = categoryName
	c.isDirtyCreditors = true
}

// SaveCreditorsToYAML saves creditor mappings to YAML file if they have been modified.
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

// UpdateDebitorCategory updates a debitor category mapping using the default categorizer
// Deprecated: Use dependency injection with NewCategorizer instead.
// This function will be removed in v2.0.0.
func UpdateDebitorCategory(partyName, categoryName string) {
	initCategorizer()
	defaultCategorizer.updateDebitorCategory(partyName, categoryName)
}

// UpdateCreditorCategory updates a creditor category mapping using the default categorizer
// Deprecated: Use dependency injection with NewCategorizer instead.
// This function will be removed in v2.0.0.
func UpdateCreditorCategory(partyName, categoryName string) {
	initCategorizer()
	defaultCategorizer.updateCreditorCategory(partyName, categoryName)
}

// categorizeWithGemini attempts to categorize a transaction using the AI client
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (models.Category, error) {
	// If no AI client is available, return uncategorized
	if c.aiClient == nil {
		return models.Category{
			Name:        models.CategoryUncategorized,
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
		c.logger.WithError(err).WithField("party", transaction.PartyName).Warn("AI categorization error for transaction")
		return models.Category{
			Name:        models.CategoryUncategorized,
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
		"COOP":       models.CategoryGroceries,
		"MIGROS":     models.CategoryGroceries,
		"ALDI":       models.CategoryGroceries,
		"LIDL":       models.CategoryGroceries,
		"DENNER":     models.CategoryGroceries,
		"MANOR":      models.CategoryShopping,
		"MIGROLINO":  models.CategoryGroceries,
		"KIOSK":      models.CategoryGroceries,
		"MINERALOEL": models.CategoryGroceries,

		// Restaurants & Food
		"PIZZERIA":    models.CategoryRestaurants,
		"CAFE":        models.CategoryRestaurants,
		"RESTAURANT":  models.CategoryRestaurants,
		"SUSHI":       models.CategoryRestaurants,
		"KEBAB":       models.CategoryRestaurants,
		"BOUCHERIE":   models.CategoryGroceries,
		"BOULANGERIE": models.CategoryGroceries,
		"RAMEN":       models.CategoryRestaurants,
		"KAMUY":       models.CategoryRestaurants,
		"MINESTRONE":  models.CategoryRestaurants,

		// Leisure
		"PISCINE":  "Loisirs",
		"SPA":      "Loisirs",
		"CINEMA":   "Loisirs",
		"PILATUS":  "Loisirs",
		"TOTEM":    "Sport",
		"ESCALADE": "Sport",

		// Shops
		"OCHSNER":       models.CategoryShopping,
		"SPORT":         models.CategoryShopping,
		"MAMMUT":        models.CategoryShopping,
		"MULLER":        models.CategoryShopping,
		"BAZAR":         models.CategoryShopping,
		"INTERDISCOUNT": models.CategoryShopping,
		"IKEA":          "Mobilier",
		"PAYOT":         models.CategoryShopping,
		"CALIDA":        models.CategoryShopping,
		"DIGITAL":       models.CategoryShopping,
		"RHB":           models.CategoryShopping,
		"WEBSHOP":       models.CategoryShopping,
		"POST":          models.CategoryShopping,

		// Services
		"PRESSING":     "Services",
		"777-PRESSING": "Services",
		"5ASEC":        "Services",

		// Cash withdrawals
		"RETRAIT":    models.CategoryWithdrawals,
		"ATM":        models.CategoryWithdrawals,
		"WITHDRAWAL": models.CategoryWithdrawals,

		// Utilities and telecom
		"ROMANDE ENERGIE": "Utilités",
		"WINGO":           "Services",

		// Transportation
		"SBB":        models.CategoryTransport,
		"CFF":        models.CategoryTransport,
		"MOBILITY":   "Voiture",
		"PAYBYPHONE": models.CategoryTransport,

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
		models.BankCodeCashWithdrawal: models.CategoryWithdrawals, // Cash withdrawals
		models.BankCodePOS:            models.CategoryShopping,    // Point of sale debit
		models.BankCodeCreditCard:     models.CategoryShopping,    // Credit card payment
		models.BankCodeInternalCredit: models.CategoryTransfers,   // Internal credit transfer
		models.BankCodeDirectDebit:    models.CategoryTransfers,   // Direct debit
		"RDDT":                       models.CategoryTransfers,   // Direct debit
		"AUTT":                       models.CategoryTransfers,   // Automatic transfer
		"RCDT":                       models.CategoryTransfers,   // Received credit transfer
		"PMNT":                       models.CategoryTransfers,   // Payment
		"RMDR":                       "Services",                 // Reminders
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
					Name:        models.CategoryShopping,
					Description: categoryDescriptionFromName(models.CategoryShopping),
				}, true
			}
		}
	}

	// No match found
	return models.Category{}, false
}
