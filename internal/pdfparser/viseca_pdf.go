package pdfparser

import (
	"context"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"strings"
	"time"
)

// parseVisecaTransactionsWithCategorizer is a specialized parser for Viseca credit card statements with categorization
func parseVisecaTransactionsWithCategorizer(ctx context.Context, lines []string, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
	logger.Debug("Processing Viseca PDF with specialized parser",
		logging.Field{Key: "lineCount", Value: len(lines)})

	var transactions []models.Transaction
	var currentCategory string

	// For debugging, dump the first few lines
	for i := 0; i < min(20, len(lines)); i++ {
		logger.Debug("Sample line from PDF",
			logging.Field{Key: "line", Value: lines[i]})
	}

	// Process lines
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Skip Viseca header lines
		if strings.Contains(line, "Date de") || (strings.Contains(line, "Date") && strings.Contains(line, "valeur") && strings.Contains(line, "Détails") && strings.Contains(line, "Montant")) {
			logger.Debug("Skipping header line")
			continue
		}

		// Skip footers and other non-transaction lines
		if strings.Contains(line, "Page") ||
			strings.Contains(line, "Total intermédiaire") ||
			strings.Contains(line, "Recouvrement") ||
			strings.Contains(line, "Limite de carte") ||
			strings.Contains(line, "Report") {
			continue
		}

		// Check if the line starts with a date (DD.MM.YY or DD.MM.YYYY format)
		if !datePatternCapture.MatchString(line) {
			// Not a transaction line, could be a category or additional info
			// Store it to potentially attach to the previous transaction
			if strings.TrimSpace(line) != "" && !strings.Contains(line, "XXXX") {
				currentCategory = strings.TrimSpace(line)
				logger.Debug("Found potential category line",
					logging.Field{Key: "category", Value: currentCategory})
			}
			continue
		}

		// This looks like a transaction line - extract the components

		// Extract transaction date and optional value date (DD.MM.YY(YY))
		dateValueMatch := dateValuePattern.FindStringSubmatch(line)
		if len(dateValueMatch) < 2 || dateValueMatch[1] == "" {
			logger.Debug("Invalid transaction line format - missing date",
				logging.Field{Key: "line", Value: line})
			continue
		}

		txDate := dateValueMatch[1]
		valueDate := txDate
		if len(dateValueMatch) >= 3 && dateValueMatch[2] != "" {
			valueDate = dateValueMatch[2]
		}

		// Now extract the amount which should be at the end of the line
		// But first, get the remaining text after the dates
		remainingLine := strings.TrimSpace(line[len(dateValueMatch[0]):])

		// Check for "Montant total" or "Votre paiement" lines - these are summaries, not transactions
		if strings.Contains(remainingLine, "Montant total") ||
			strings.Contains(remainingLine, "Votre paiement") {
			logger.Debug("Skipping summary line",
				logging.Field{Key: "line", Value: line})
			continue
		}

		// The amount is typically right-aligned at the end
		// Look for a number pattern at the end, possibly followed by a minus sign
		amountMatch := amountEndPattern.FindStringSubmatch(remainingLine)

		if len(amountMatch) < 2 {
			logger.Debug("Could not extract amount from transaction line",
				logging.Field{Key: "line", Value: line})
			continue
		}

		amount := amountMatch[1]
		// Remove Swiss formatting (apostrophes as thousand separators)
		amount = strings.ReplaceAll(amount, "'", "")

		// Check if credit (minus sign after amount)
		isCredit := len(amountMatch) > 2 && amountMatch[2] == "-"

		// Extract description (everything between dates and amount)
		descriptionEndPos := strings.LastIndex(remainingLine, amount)
		if descriptionEndPos <= 0 {
			logger.Debug("Could not determine description boundaries",
				logging.Field{Key: "line", Value: line})
			continue
		}

		description := strings.TrimSpace(remainingLine[:descriptionEndPos])

		// Check for foreign currency indicators
		var originalCurrency, originalAmount string
		currencyMatch := foreignCurrencyPattern.FindStringSubmatch(description)
		if len(currencyMatch) > 2 {
			originalCurrency = currencyMatch[1]
			originalAmount = strings.ReplaceAll(currencyMatch[2], "'", "")

			// Clean up description
			description = strings.Replace(description, currencyMatch[0], "", 1)
			description = strings.TrimSpace(description)

			logger.Debug("Found foreign currency transaction",
				logging.Field{Key: "currency", Value: originalCurrency},
				logging.Field{Key: "amount", Value: originalAmount})
		}

		// Create the transaction using TransactionBuilder
		builder := models.NewTransactionBuilder().
			WithDatetime(formatDate(txDate)).
			WithValueDatetime(formatDate(valueDate)).
			WithDescription(description).
			WithAmountFromString(amount, "CHF").
			WithOriginalAmount(models.ParseAmount(originalAmount), originalCurrency)

		// Set transaction direction and parties
		if isCredit {
			builder = builder.AsCredit().WithPayer(description, "")
		} else {
			builder = builder.AsDebit().WithPayee(description, "")
		}

		// Build the transaction
		tx, err := builder.Build()
		if err != nil {
			logger.WithError(err).Warn("Failed to build transaction, skipping",
				logging.Field{Key: "description", Value: description})
			continue
		}

		// Attach category if we have one
		if currentCategory != "" {
			tx.Description = tx.Description + " - " + currentCategory
			logger.Debug("Added category to transaction",
				logging.Field{Key: "category", Value: currentCategory})
			currentCategory = "" // Reset for next transaction
		}

		// Look for additional information in following lines (exchange rate, processing fees)
		var exchangeRateFound, processingFeeFound bool

		for j := i + 1; j < min(i+3, len(lines)) && !exchangeRateFound && !processingFeeFound; j++ {
			nextLine := strings.TrimSpace(lines[j])

			// Skip empty lines
			if nextLine == "" {
				continue
			}

			// If the next line starts with a date, it's a new transaction - stop looking
			if datePatternCapture.MatchString(nextLine) {
				break
			}

			// Look for exchange rate information
			if strings.Contains(nextLine, "Taux de conversion") {
				exchangeRateMatch := exchangeRatePattern.FindStringSubmatch(nextLine)
				if len(exchangeRateMatch) > 1 {
					tx.ExchangeRate = models.ParseAmount(exchangeRateMatch[1])
					exchangeRateFound = true
					logger.Debug("Found exchange rate",
						logging.Field{Key: "exchangeRate", Value: tx.ExchangeRate.String()})
				}
			}

			// Look for processing fee information
			if strings.Contains(nextLine, "Frais de traitement") {
				feeMatch := processingFeePattern.FindStringSubmatch(nextLine)
				if len(feeMatch) > 1 {
					tx.Fees = models.ParseAmount(feeMatch[1])
					processingFeeFound = true
					logger.Debug("Found processing fees",
						logging.Field{Key: "fees", Value: tx.Fees.String()})
				}
			}
		}

		// Add transaction to list
		transactions = append(transactions, tx)
		logger.Debug("Added transaction",
			logging.Field{Key: "date", Value: tx.Date},
			logging.Field{Key: "description", Value: tx.Description},
			logging.Field{Key: "amount", Value: tx.Amount.String()})
	}

	// Log the number of transactions found
	// Process transactions with categorization statistics
	processedTransactions, err := common.ProcessTransactionsWithCategorizationStats(
		ctx, transactions, logger, categorizer, "PDF-Viseca")
	if err != nil {
		return nil, err
	}

	logger.Info("Extracted transactions from Viseca PDF",
		logging.Field{Key: "count", Value: len(processedTransactions)})
	return processedTransactions, nil
}

// formatDate parses a date string and returns time.Time
func formatDate(date string) time.Time {
	// Remove any non-digit or dot characters
	date = nonDigitDotPattern.ReplaceAllString(date, "")

	// Try to identify the format
	formats := []string{
		dateutils.DateLayoutEuropean, // DD.MM.YYYY
		"02.01.06",                   // DD.MM.YY
		"2/1/2006",                   // M/D/YYYY
		"1/2/2006",                   // D/M/YYYY
		dateutils.DateLayoutISO,      // YYYY-MM-DD
	}

	var t time.Time
	var err error

	for _, format := range formats {
		t, err = time.Parse(format, date)
		if err == nil {
			break
		}
	}

	// If we failed to parse any format, return zero time
	if err != nil {
		return time.Time{}
	}

	return t
}
