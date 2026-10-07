// Package common provides shared utilities used across the application.
package common

import (
	"context"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// DefaultPartyName picks who a transaction is with: the payee or payer, then
// the other party fields, and finally the description, which is all some
// formats (Visa debit exports) carry.
func DefaultPartyName(tx models.Transaction) string {
	for _, name := range []string{tx.GetPartyName(), tx.PartyName, tx.Name, tx.Recipient, tx.Description} {
		if name != "" {
			return name
		}
	}
	return ""
}

// ProcessTransactionsWithCategorizationStats categorizes with DefaultPartyName.
func ProcessTransactionsWithCategorizationStats(
	ctx context.Context,
	transactions []models.Transaction,
	logger logging.Logger,
	categorizer models.TransactionCategorizer,
	parserType string,
) ([]models.Transaction, error) {
	return ProcessTransactionsWithPartyName(ctx, transactions, logger, categorizer, parserType, DefaultPartyName)
}

// ProcessTransactionsWithPartyName processes transactions with categorization
// and tracks statistics, providing fallback behavior for failed categorization.
//
// Categorization can reach a remote AI provider, so ctx governs the whole run:
// it is passed to every Categorize call and checked between transactions. If ctx
// is cancelled the function stops and returns ctx.Err() rather than silently
// returning a partially categorized slice.
func ProcessTransactionsWithPartyName(
	ctx context.Context,
	transactions []models.Transaction,
	logger logging.Logger,
	categorizer models.TransactionCategorizer,
	parserType string,
	partyName func(models.Transaction) string,
) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	stats := models.NewCategorizationStats()
	processedTransactions := make([]models.Transaction, len(transactions))

	for i, tx := range transactions {
		if err := ctx.Err(); err != nil {
			logger.Warn("Categorization cancelled",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "processed", Value: i},
				logging.Field{Key: "total", Value: len(transactions)})
			return nil, err
		}

		stats.IncrementTotal()
		processedTransactions[i] = tx

		// Skip categorization if no categorizer is provided
		if categorizer == nil {
			logger.Debug("No categorizer provided, skipping categorization",
				logging.Field{Key: "parser_type", Value: parserType})
			if tx.Category == "" {
				processedTransactions[i].Category = models.CategoryUncategorized
				stats.IncrementUncategorized()
			} else {
				stats.IncrementSuccessful()
			}
			continue
		}

		// Skip categorization if category already determined by parser-internal logic
		if tx.Category != "" && tx.Category != models.CategoryUncategorized {
			logger.Debug("Category already set, skipping external categorization",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "category", Value: tx.Category})
			stats.IncrementSuccessful()
			continue
		}

		// Attempt categorization
		party := partyName(tx)

		if party == "" {
			logger.Debug("No party name available for categorization",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "transaction_description", Value: tx.Description})
			stats.IncrementUncategorized()
			processedTransactions[i].Category = models.CategoryUncategorized
			continue
		}

		category, err := categorizer.Categorize(
			ctx,
			party,
			tx.IsDebit(),
			tx.Amount.String(),
			tx.Date.Format("2006-01-02"),
			tx.Description,
		)

		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				// Cancelled mid-run, not a per-transaction classification
				// failure: report it rather than filing the rest as Uncategorized.
				return nil, ctxErr
			}

			logger.WithError(err).Warn("Categorization failed",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "party_name", Value: party},
				logging.Field{Key: "amount", Value: tx.Amount.String()})
			stats.IncrementFailed()
			processedTransactions[i].Category = models.CategoryUncategorized
		} else if category.Name == "" || category.Name == models.CategoryUncategorized {
			logger.Debug("Transaction categorized as uncategorized",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "party_name", Value: party})
			stats.IncrementUncategorized()
			processedTransactions[i].Category = models.CategoryUncategorized
		} else {
			logger.Debug("Transaction categorized successfully",
				logging.Field{Key: "parser_type", Value: parserType},
				logging.Field{Key: "party_name", Value: party},
				logging.Field{Key: "category", Value: category.Name})
			stats.IncrementSuccessful()
			processedTransactions[i].Category = category.Name
		}
	}

	// Log summary statistics
	stats.LogSummary(logger, parserType)

	return processedTransactions, nil
}
