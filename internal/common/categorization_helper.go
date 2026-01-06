// Package common provides shared utilities used across the application.
package common

import (
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// ProcessTransactionsWithCategorizationStats processes transactions with categorization
// and tracks statistics, providing fallback behavior for failed categorization
func ProcessTransactionsWithCategorizationStats(
	transactions []models.Transaction,
	logger logging.Logger,
	categorizer models.TransactionCategorizer,
	parserType string,
) []models.Transaction {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	stats := models.NewCategorizationStats()
	processedTransactions := make([]models.Transaction, len(transactions))

	for i, tx := range transactions {
		stats.IncrementTotal()
		processedTransactions[i] = tx

		// Skip categorization if no categorizer is provided
		if categorizer == nil {
			logger.Debug("No categorizer provided, skipping categorization",
				logging.Field{Key: "parser_type", Value: parserType})
			stats.IncrementUncategorized()
			processedTransactions[i].Category = "Uncategorized"
			continue
		}

		// Attempt categorization
		partyName := tx.GetPartyName()
		if partyName == "" {
			// Fallback to other party name fields
			if tx.PartyName != "" {
				partyName = tx.PartyName
			} else if tx.Name != "" {
				partyName = tx.Name
			} else if tx.Recipient != "" {
				partyName = tx.Recipient
			}
		}

		if partyName == "" {
			logger.Debug("No party name available for categorization",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "transaction_description", Value: tx.Description})
			stats.IncrementUncategorized()
			processedTransactions[i].Category = "Uncategorized"
			continue
		}

		category, err := categorizer.Categorize(
			partyName,
			tx.IsDebit(),
			tx.Amount.String(),
			tx.Date.Format("2006-01-02"),
			tx.Description,
		)

		if err != nil {
			logger.WithError(err).Warn("Categorization failed",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "party_name", Value: partyName},
				logging.Field{Key: "amount", Value: tx.Amount.String()})
			stats.IncrementFailed()
			processedTransactions[i].Category = "Uncategorized"
		} else if category.Name == "" || category.Name == "Uncategorized" {
			logger.Debug("Transaction categorized as uncategorized",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "party_name", Value: partyName})
			stats.IncrementUncategorized()
			processedTransactions[i].Category = "Uncategorized"
		} else {
			logger.Debug("Transaction categorized successfully",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "party_name", Value: partyName},
				logging.Field{Key: "category", Value: category.Name})
			stats.IncrementSuccessful()
			processedTransactions[i].Category = category.Name
		}
	}

	// Log summary statistics
	stats.LogSummary(logger, parserType)

	return processedTransactions
}

// CategorizeTransactionWithStats categorizes a single transaction and updates statistics
func CategorizeTransactionWithStats(
	tx *models.Transaction,
	categorizer models.TransactionCategorizer,
	stats *models.CategorizationStats,
	logger logging.Logger,
	parserType string,
) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	stats.IncrementTotal()

	// Skip categorization if no categorizer is provided
	if categorizer == nil {
		logger.Debug("No categorizer provided, skipping categorization",
			logging.Field{Key: "parser_type", Value: parserType})
		stats.IncrementUncategorized()
		tx.Category = "Uncategorized"
		return
	}

	// Attempt categorization
	partyName := tx.GetPartyName()
	if partyName == "" {
		// Fallback to other party name fields
		if tx.PartyName != "" {
			partyName = tx.PartyName
		} else if tx.Name != "" {
			partyName = tx.Name
		} else if tx.Recipient != "" {
			partyName = tx.Recipient
		}
	}

	if partyName == "" {
		logger.Debug("No party name available for categorization",
			logging.Field{Key: "parser_type", Value: parserType},
			logging.Field{Key: "transaction_description", Value: tx.Description})
		stats.IncrementUncategorized()
		tx.Category = "Uncategorized"
		return
	}

	category, err := categorizer.Categorize(
		partyName,
		tx.IsDebit(),
		tx.Amount.String(),
		tx.Date.Format("2006-01-02"),
		tx.Description,
	)

	if err != nil {
		logger.WithError(err).Warn("Categorization failed",
			logging.Field{Key: "parser_type", Value: parserType},
			logging.Field{Key: "party_name", Value: partyName},
			logging.Field{Key: "amount", Value: tx.Amount.String()})
		stats.IncrementFailed()
		tx.Category = "Uncategorized"
	} else if category.Name == "" || category.Name == "Uncategorized" {
		logger.Debug("Transaction categorized as uncategorized",
			logging.Field{Key: "parser_type", Value: parserType},
			logging.Field{Key: "party_name", Value: partyName})
		stats.IncrementUncategorized()
		tx.Category = "Uncategorized"
	} else {
		logger.Debug("Transaction categorized successfully",
			logging.Field{Key: "parser_type", Value: parserType},
			logging.Field{Key: "party_name", Value: partyName},
			logging.Field{Key: "category", Value: category.Name})
		stats.IncrementSuccessful()
		tx.Category = category.Name
	}
}
