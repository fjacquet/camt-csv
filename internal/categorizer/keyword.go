package categorizer

import (
	"context"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// KeywordStrategy implements categorization using keyword pattern matching
// from category configuration loaded from YAML files.
type KeywordStrategy struct {
	categories []models.CategoryConfig
	store      CategoryStoreInterface
	logger     logging.Logger
}

// NewKeywordStrategy creates a new KeywordStrategy instance.
func NewKeywordStrategy(store CategoryStoreInterface, logger logging.Logger) *KeywordStrategy {
	strategy := &KeywordStrategy{
		categories: []models.CategoryConfig{},
		store:      store,
		logger:     logger,
	}

	// Load categories from store
	strategy.loadCategories()

	return strategy
}

// Name returns the name of this strategy for logging and debugging.
func (s *KeywordStrategy) Name() string {
	return "Keyword"
}

// Categorize attempts to categorize a transaction using keyword pattern matching.
func (s *KeywordStrategy) Categorize(ctx context.Context, tx Transaction) (models.Category, bool, error) {
	// If party name is empty, cannot categorize
	if strings.TrimSpace(tx.PartyName) == "" {
		return models.Category{}, false, nil
	}

	// Convert transaction data to uppercase for case-insensitive matching
	partyName := strings.ToUpper(tx.PartyName)
	description := strings.ToUpper(tx.Info)

	// Try to match against category keywords
	for _, categoryConfig := range s.categories {
		for _, keyword := range categoryConfig.Keywords {
			keywordUpper := strings.ToUpper(keyword)
			
			// Check if keyword appears in party name or description
			if strings.Contains(partyName, keywordUpper) || strings.Contains(description, keywordUpper) {
				s.logger.WithFields(
					logging.Field{Key: "strategy", Value: s.Name()},
					logging.Field{Key: "party", Value: tx.PartyName},
					logging.Field{Key: "keyword", Value: keyword},
					logging.Field{Key: "category", Value: categoryConfig.Name},
				).Debug("Transaction categorized using keyword matching")

				category := models.Category{
					Name:        categoryConfig.Name,
					Description: categoryDescriptionFromName(categoryConfig.Name),
				}

				return category, true, nil
			}
		}
	}

	// Also try hardcoded keyword patterns for backward compatibility
	// This includes the existing logic from categorizeLocallyByKeywords
	if category, found := s.categorizeWithHardcodedPatterns(tx); found {
		return category, true, nil
	}

	return models.Category{}, false, nil
}

// categorizeWithHardcodedPatterns implements the existing hardcoded keyword logic
// for backward compatibility. This should eventually be moved to YAML configuration.
func (s *KeywordStrategy) categorizeWithHardcodedPatterns(tx Transaction) (models.Category, bool) {
	partyName := strings.ToUpper(tx.PartyName)
	description := strings.ToUpper(tx.Info)

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
			s.logger.WithFields(
				logging.Field{Key: "strategy", Value: s.Name()},
				logging.Field{Key: "party", Value: tx.PartyName},
				logging.Field{Key: "keyword", Value: keyword},
				logging.Field{Key: "category", Value: category},
				logging.Field{Key: "pattern_type", Value: "merchant"},
			).Debug("Transaction categorized using hardcoded merchant pattern")

			return models.Category{
				Name:        category,
				Description: categoryDescriptionFromName(category),
			}, true
		}
	}

	// Try to detect transaction types from bank codes
	for bankCode, category := range txCodeCategories {
		// Check if bank code appears in transaction info
		if strings.Contains(tx.Info, bankCode) {
			s.logger.WithFields(
				logging.Field{Key: "strategy", Value: s.Name()},
				logging.Field{Key: "party", Value: tx.PartyName},
				logging.Field{Key: "bank_code", Value: bankCode},
				logging.Field{Key: "category", Value: category},
				logging.Field{Key: "pattern_type", Value: "bank_code"},
			).Debug("Transaction categorized using bank code pattern")

			return models.Category{
				Name:        category,
				Description: categoryDescriptionFromName(category),
			}, true
		}
	}

	// Look for credit cards and cash withdrawals
	if tx.IsDebtor {
		// Special case for Unknown Payee for card payments - don't default to Salaire
		if strings.Contains(partyName, "UNKNOWN PAYEE") || partyName == "UNKNOWN PAYEE" {
			// If it looks like a card payment or cash withdrawal
			if strings.Contains(description, "PMT CARTE") ||
				strings.Contains(description, "PMT TWINT") ||
				strings.Contains(description, "RETRAIT") ||
				strings.Contains(description, "WITHDRAWAL") {
				
				s.logger.WithFields(
					logging.Field{Key: "strategy", Value: s.Name()},
					logging.Field{Key: "party", Value: tx.PartyName},
					logging.Field{Key: "category", Value: models.CategoryShopping},
					logging.Field{Key: "pattern_type", Value: "unknown_payee_card"},
				).Debug("Transaction categorized as shopping for unknown payee card payment")

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

// loadCategories loads category configurations from the store.
func (s *KeywordStrategy) loadCategories() {
	categories, err := s.store.LoadCategories()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to load categories for KeywordStrategy")
	} else {
		s.categories = categories
		s.logger.WithField("count", len(categories)).Debug("Loaded categories for KeywordStrategy")
	}
}

// ReloadCategories reloads the categories from the store.
// This can be called when the underlying YAML files have been updated.
func (s *KeywordStrategy) ReloadCategories() {
	s.loadCategories()
}