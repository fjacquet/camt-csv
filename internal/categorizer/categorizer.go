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

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// normalizeStringToLowerCategorizer converts a string to lowercase using strings.Builder
// for optimal performance in hot paths. Pre-allocates capacity to minimize allocations.
//
// Performance rationale: Centralizes string normalization logic and ensures
// consistent memory allocation patterns across the categorizer. The helper
// function approach reduces code duplication and provides a single point
// for optimization improvements.
func normalizeStringToLowerCategorizer(input string) string {
	if input == "" {
		return ""
	}

	// Fast path for ASCII-only strings (common case)
	if isASCII(input) {
		// Performance optimization: Pre-allocate builder capacity to avoid reallocations
		builder := strings.Builder{}
		builder.Grow(len(input))
		for i := 0; i < len(input); i++ {
			c := input[i]
			if c >= 'A' && c <= 'Z' {
				builder.WriteByte(c + 32) // Convert to lowercase
			} else {
				builder.WriteByte(c)
			}
		}
		return builder.String()
	}

	// Fallback for Unicode strings
	return strings.ToLower(input)
}

// isASCII checks if a string contains only ASCII characters
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

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

// Categorizer handles the categorization of transactions using multiple strategies.
// It orchestrates different categorization approaches in priority order.
type Categorizer struct {
	// Strategy-based categorization
	strategies []CategorizationStrategy

	// Legacy fields for backward compatibility and auto-learning
	categories       []models.CategoryConfig
	creditorMappings map[string]string // Maps creditor names to categories
	debitorMappings  map[string]string // Maps debitor names to categories
	configMutex      sync.RWMutex
	isDirtyCreditors bool // Track if creditorMappings has been modified and needs to be saved
	isDirtyDebitors  bool // Track if debitorMappings has been modified and needs to be saved
	store            CategoryStoreInterface
	logger           logging.Logger

	// Lazy initialization for AI client
	aiClient  AIClient // New field for AIClient interface
	aiFactory func() AIClient
}

// Note: log variable removed as part of dependency injection refactoring

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

// NewCategorizer creates a new instance of Categorizer with the given AIClient, CategoryStore, and logger.
func NewCategorizer(aiClient AIClient, store CategoryStoreInterface, logger logging.Logger) *Categorizer {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	c := &Categorizer{
		categories:       make([]models.CategoryConfig, 0, 50), // Pre-allocate with reasonable capacity
		creditorMappings: make(map[string]string, 100),         // Pre-allocate with size hint
		debitorMappings:  make(map[string]string, 100),         // Pre-allocate with size hint
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

	// Initialize strategies in priority order
	// Note: SemanticStrategy needs loaded categories to build its index
	c.strategies = []CategorizationStrategy{
		NewDirectMappingStrategy(store, logger),
		NewKeywordStrategy(store, logger),
		NewSemanticStrategy(aiClient, logger, c.categories),
		NewAIStrategy(aiClient, logger),
	}

	// Load creditor mappings
	creditorMappings, err := c.store.LoadCreditorMappings()
	if err != nil {
		c.logger.WithError(err).Warn("Failed to load creditor mappings")
	} else {
		// Pre-allocate with known size and normalize keys to lowercase for case-insensitive lookup
		if len(creditorMappings) > len(c.creditorMappings) {
			// Expand map if needed
			newMap := make(map[string]string, len(creditorMappings))
			for k, v := range c.creditorMappings {
				newMap[k] = v
			}
			c.creditorMappings = newMap
		}

		// Performance optimization: Use helper function to minimize allocations when loading creditor mappings
		for key, value := range creditorMappings {
			c.creditorMappings[normalizeStringToLowerCategorizer(key)] = value
		}
	}

	// Load debtor mappings
	debitorMappings, err := c.store.LoadDebtorMappings()
	if err != nil {
		c.logger.WithError(err).Warn("Failed to load debtor mappings")
	} else {
		// Pre-allocate with known size and normalize keys to lowercase for case-insensitive lookup
		if len(debitorMappings) > len(c.debitorMappings) {
			// Expand map if needed
			newMap := make(map[string]string, len(debitorMappings))
			for k, v := range c.debitorMappings {
				newMap[k] = v
			}
			c.debitorMappings = newMap
		}

		// Performance optimization: Use helper function to minimize allocations when loading debtor mappings
		for key, value := range debitorMappings {
			c.debitorMappings[normalizeStringToLowerCategorizer(key)] = value
		}
	}

	return c
}

// SetAIClientFactory sets a factory function for lazy AI client initialization.
// This allows expensive AI client creation to be deferred until actually needed.
func (c *Categorizer) SetAIClientFactory(factory func() AIClient) {
	c.aiFactory = factory
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
//
//	container, err := container.NewContainer(config)
//	if err != nil {
//	    return err
//	}
//	category, err := categorizer.CategorizeTransactionWithCategorizer(container.Categorizer, transaction)
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
//
//	container, err := container.NewContainer(config)
//	if err != nil {
//	    return err
//	}
//	category, err := container.Categorizer.CategorizeTransaction(transaction)
func (c *Categorizer) CategorizeTransaction(transaction Transaction) (models.Category, error) {
	return c.categorizeTransaction(transaction)
}

// Categorize implements the models.TransactionCategorizer interface.
// This method provides a simple interface for categorizing transactions without
// requiring the caller to create a Transaction struct.
//
// This method includes auto-learning: when a category is successfully determined,
// it automatically saves the mapping to the appropriate database (creditors or debtors)
// for future use.
//
// Parameters:
//   - partyName: The name of the transaction party
//   - isDebtor: true if the party is a debtor (sender), false if creditor
//   - amount: Transaction amount as string
//   - date: Transaction date as string
//   - info: Additional transaction information
//
// Returns:
//   - models.Category: The determined category
//   - error: Any error that occurred during categorization
func (c *Categorizer) Categorize(partyName string, isDebtor bool, amount, date, info string) (models.Category, error) {
	transaction := Transaction{
		PartyName: partyName,
		IsDebtor:  isDebtor,
		Amount:    amount,
		Date:      date,
		Info:      info,
	}

	category, err := c.categorizeTransaction(transaction)

	// Auto-learn: if we successfully found a category, save it to the database
	// so we don't need to recategorize similar transactions in the future
	if err == nil && category.Name != "" && category.Name != models.CategoryUncategorized {
		if isDebtor {
			c.logger.WithFields(
				logging.Field{Key: "party", Value: partyName},
				logging.Field{Key: "category", Value: category.Name},
			).Debug("Auto-learning debitor mapping")
			c.updateDebitorCategory(partyName, category.Name)
			if saveErr := c.SaveDebitorsToYAML(); saveErr != nil {
				c.logger.WithError(saveErr).Warn("Failed to save debitor mapping")
			}
		} else {
			c.logger.WithFields(
				logging.Field{Key: "party", Value: partyName},
				logging.Field{Key: "category", Value: category.Name},
			).Debug("Auto-learning creditor mapping")
			c.updateCreditorCategory(partyName, category.Name)
			if saveErr := c.SaveCreditorsToYAML(); saveErr != nil {
				c.logger.WithError(saveErr).Warn("Failed to save creditor mapping")
			}
		}
	}

	return category, err
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

	// Try each strategy in priority order
	ctx := context.Background()
	for _, strategy := range c.strategies {
		c.logger.WithFields(
			logging.Field{Key: "strategy", Value: strategy.Name()},
			logging.Field{Key: "party", Value: transaction.PartyName},
		).Debug("Trying strategy")

		category, found, err := strategy.Categorize(ctx, transaction)
		if err != nil {
			c.logger.WithError(err).WithFields(
				logging.Field{Key: "strategy", Value: strategy.Name()},
				logging.Field{Key: "party", Value: transaction.PartyName},
			).Warn("Strategy failed during categorization")
			continue
		}

		if found {
			c.logger.WithFields(
				logging.Field{Key: "strategy", Value: strategy.Name()},
				logging.Field{Key: "party", Value: transaction.PartyName},
				logging.Field{Key: "category", Value: category.Name},
			).Debug("Transaction categorized successfully")
			return category, nil
		}

		c.logger.WithFields(
			logging.Field{Key: "strategy", Value: strategy.Name()},
			logging.Field{Key: "party", Value: transaction.PartyName},
		).Debug("Strategy did not find a match")
	}

	// If no strategy succeeded, return uncategorized
	c.logger.WithFields(
		logging.Field{Key: "party", Value: transaction.PartyName},
	).Debug("No strategy could categorize transaction, returning uncategorized")

	return models.Category{
		Name:        models.CategoryUncategorized,
		Description: "No categorization strategy succeeded",
	}, nil
}

func categoryDescriptionFromName(name string) string {
	// In a real-world scenario, you would look up the description from a database
	return "Description for " + name
}

//------------------------------------------------------------------------------
// PARTY MAPPING UPDATES
//------------------------------------------------------------------------------
// The following methods follow a symmetric pattern for debitors and creditors.
// This intentional duplication (2 cases) keeps the code simple and readable,
// rather than adding abstraction overhead for minimal code reuse benefit.
// Each method is small (~10 lines) and self-contained.

// UpdateDebitorCategory updates a debitor category mapping for this categorizer instance.
func (c *Categorizer) UpdateDebitorCategory(partyName, categoryName string) {
	c.updateDebitorCategory(partyName, categoryName)
}

func (c *Categorizer) updateDebitorCategory(partyName, categoryName string) {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	// Performance optimization: Use helper function to minimize allocations during mapping updates
	c.debitorMappings[normalizeStringToLowerCategorizer(partyName)] = categoryName
	c.isDirtyDebitors = true

	// Update the DirectMappingStrategy as well
	for _, strategy := range c.strategies {
		if directMapping, ok := strategy.(*DirectMappingStrategy); ok {
			directMapping.UpdateDebtorMapping(partyName, categoryName)
			break
		}
	}
}

// SaveDebitorsToYAML saves debitor mappings to YAML file if they have been modified.
func (c *Categorizer) SaveDebitorsToYAML() error {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()
	if !c.isDirtyDebitors {
		return nil
	}
	if err := c.store.SaveDebtorMappings(c.debitorMappings); err != nil {
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
	// Performance optimization: Use helper function to minimize allocations during mapping updates
	c.creditorMappings[normalizeStringToLowerCategorizer(partyName)] = categoryName
	c.isDirtyCreditors = true

	// Update the DirectMappingStrategy as well
	for _, strategy := range c.strategies {
		if directMapping, ok := strategy.(*DirectMappingStrategy); ok {
			directMapping.UpdateCreditorMapping(partyName, categoryName)
			break
		}
	}
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
