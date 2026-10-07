// Package pdfparser provides functionality to parse PDF files and extract transaction data.
package pdfparser

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// Pre-compiled regex patterns for performance
var (
	// Transaction date patterns
	datePatternSimple    = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{2,4}`)
	datePatternCapture   = regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{2,4})`)
	dateValuePattern     = regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{2,4})(?:\s+(\d{2}\.\d{2}\.\d{2,4}))?`)
	nonDigitDotPattern   = regexp.MustCompile(`[^\d.]`)
	multipleSpacePattern = regexp.MustCompile(`\s+`)
	tripleSpacePattern   = regexp.MustCompile(`[ ]{3,}`)

	// Amount patterns
	amountEndPattern      = regexp.MustCompile(`(\d+[\'.,]\d+)\s*(-)?$`)
	amountGenericPattern  = regexp.MustCompile(`[+-]?\s*(\d+'?)+[,\.]?\d*\s*(CHF|EUR|USD|\$|€)?`)
	amountCurrencyPattern = regexp.MustCompile(`\d+[\.,]\d+\s*[A-Z]{3}|[A-Z]{3}\s*\d+[\.,]\d+`)

	// Currency and fee patterns
	foreignCurrencyPattern = regexp.MustCompile(`([A-Z]{3})\s+(\d+[\'.,]\d+)`)
	exchangeRatePattern    = regexp.MustCompile(`Taux de conversion\s+(\d+\.\d+)`)
	processingFeePattern   = regexp.MustCompile(`Frais de traitement\s+.+?\s+(\d+\.\d+)`)

	// Merchant identifier pattern
	cardPurchasePattern = regexp.MustCompile(`(?i)card\s+purchase\s+at`)

	// Payee extraction patterns
	payeePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)payee:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)to:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)merchant:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)payment to:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)paid to:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)transfer to:\s*([^,;]+)`),
	}

	// Merchant extraction patterns
	merchantPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)merchant:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)vendor:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)shop:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)store:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)payee name:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)business:\s*([^,;]+)`),
		regexp.MustCompile(`(?i)company:\s*([^,;]+)`),
	}

	// Noise phrase patterns for description cleaning
	noisePhrasePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)transaction date`),
		regexp.MustCompile(`(?i)value date`),
		regexp.MustCompile(`(?i)description`),
		regexp.MustCompile(`(?i)amount`),
	}
)

// parseTransactionsWithCategorizer parses transaction data from PDF text content and applies categorization
func parseTransactionsWithCategorizer(ctx context.Context, lines []string, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
	// Pre-allocate slice with estimated capacity (typically 10-50 transactions per PDF)
	transactions := make([]models.Transaction, 0, 50)
	var currentTx models.Transaction
	var description strings.Builder
	var merchant string
	// Pre-allocate map with size hint for duplicate detection
	seen := make(map[string]bool, 50)

	inTransaction := false
	isVisecaFormat := false

	// Check if this is a Viseca file format (look for typical Viseca PDF headers and patterns)
	for _, line := range lines {
		// Check for the column headers that appear in Viseca statements
		if strings.Contains(line, "Date") && strings.Contains(line, "valeur") &&
			strings.Contains(line, "Détails") && strings.Contains(line, "Monnaie") &&
			strings.Contains(line, "Montant") {
			isVisecaFormat = true
			logger.Debug("Detected Viseca PDF format - header pattern matched")
			break
		}

		// Also look for Viseca card number pattern
		if strings.Contains(line, "Visa Gold") || strings.Contains(line, "Visa Platinum") ||
			strings.Contains(line, "Mastercard") || strings.Contains(line, "XXXX") {
			isVisecaFormat = true
			logger.Debug("Detected Viseca PDF format - card pattern matched")
			break
		}

		// Check for typical Viseca statement features
		if strings.Contains(line, "Montant total dernier relevé") ||
			strings.Contains(line, "Votre paiement - Merci") {
			isVisecaFormat = true
			logger.Debug("Detected Viseca PDF format - statement pattern matched")
			break
		}
	}

	logger.Debug("Format detection result",
		logging.Field{Key: "isVisecaFormat", Value: isVisecaFormat})

	// For Viseca format, use a specialized transaction extraction approach
	if isVisecaFormat {
		return parseVisecaTransactionsWithCategorizer(ctx, lines, logger, categorizer)
	}

	// Standard PDF format parsing continues below
	// Preprocess the lines to identify transaction blocks
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// Skip header lines
		if i < 5 && !containsAmount(trimmedLine) {
			continue
		}

		// Identify transaction start by date pattern (DD.MM.YY or similar)
		if datePatternSimple.MatchString(trimmedLine) {
			logger.Debug("Found potential transaction start",
				logging.Field{Key: "line", Value: trimmedLine})

			// Finalize previous transaction if we're in one
			if inTransaction {
				logger.Debug("Finalizing previous transaction")
				finalizeTransactionWithCategorizer(&currentTx, &description, merchant, seen, &transactions, categorizer, logger)
			}

			// Start a new transaction
			inTransaction = true
			currentTx = models.Transaction{} // Keep minimal struct for temporary storage during parsing
			description.Reset()
			// merchant would be empty string here, but we don't need to assign it

			// Extract date
			fields := strings.Fields(trimmedLine)
			if len(fields) > 0 {
				currentTx.Date = formatDate(fields[0])
				logger.Debug("Extracted transaction date",
					logging.Field{Key: "date", Value: currentTx.Date.Format(dateutils.DateLayoutEuropean)})

				// Try to extract value date if present
				if len(fields) > 1 && datePatternSimple.MatchString(fields[1]) {
					currentTx.ValueDate = formatDate(fields[1])
				}
			}

			// Extract amount and currency
			_, amount, isCredit := extractAmount(trimmedLine)
			currentTx.Amount = amount
			currentTx.CreditDebit = models.TransactionTypeDebit
			if isCredit {
				currentTx.CreditDebit = models.TransactionTypeCredit
			}

			// Add to description
			description.WriteString(trimmedLine)

			// Try to extract merchant name
			merchant = extractMerchant(trimmedLine)
		} else if inTransaction {
			// Continuing a transaction - append to description
			logger.Debug("Continuing transaction with line",
				logging.Field{Key: "line", Value: trimmedLine})
			description.WriteString(" ")
			description.WriteString(trimmedLine)

			// Extract amount and currency if not yet set
			if currentTx.Amount.IsZero() {
				_, amount, _ := extractAmount(trimmedLine)
				currentTx.Amount = amount
			}

			// Try to extract merchant if not yet found
			if merchant == "" && containsMerchantIdentifier(trimmedLine) {
				merchant = extractMerchant(trimmedLine)
			}
		}
	}

	// Finalize the last transaction if needed
	if inTransaction {
		logger.Debug("Finalizing last transaction")
		finalizeTransactionWithCategorizer(&currentTx, &description, merchant, seen, &transactions, categorizer, logger)
	}

	// Sort transactions by date
	sortTransactions(transactions)

	// Remove duplicates
	transactions = deduplicateTransactions(transactions)

	// Process transactions with categorization statistics
	processedTransactions, err := common.ProcessTransactionsWithCategorizationStats(
		ctx, transactions, logger, categorizer, "PDF")
	if err != nil {
		return nil, err
	}

	logger.Info("Extracted transactions from PDF",
		logging.Field{Key: "count", Value: len(processedTransactions)})
	return processedTransactions, nil
}

// finalizeTransactionWithCategorizer finalizes a transaction with categorization and adds it to the list of transactions
func finalizeTransactionWithCategorizer(tx *models.Transaction, desc *strings.Builder, merchant string, seen map[string]bool, transactions *[]models.Transaction, categorizer models.TransactionCategorizer, logger logging.Logger) {
	// Clean the description
	cleanDesc := cleanDescription(desc.String())

	// Extract and set merchant/payee
	payee := merchant
	if payee == "" {
		payee = extractPayee(cleanDesc)
	}

	// Set credit/debit indicator if not already set
	creditDebit := tx.CreditDebit
	if creditDebit == "" {
		creditDebit = determineCreditDebit(cleanDesc)
	}

	// Use TransactionBuilder to construct the final transaction
	builder := models.NewTransactionBuilder().
		WithDatetime(tx.Date).
		WithAmount(tx.Amount, "CHF"). // Default to CHF for PDF statements
		WithDescription(cleanDesc).
		WithPayee(payee, "").
		WithCategory(models.CategoryUncategorized)

	// Set value date if available, otherwise use transaction date
	if !tx.ValueDate.IsZero() {
		builder = builder.WithValueDatetime(tx.ValueDate)
	} else {
		builder = builder.WithValueDatetime(tx.Date)
	}

	// Set transaction direction
	if creditDebit == models.TransactionTypeCredit {
		builder = builder.AsCredit()
	} else {
		builder = builder.AsDebit()
	}

	// Build the final transaction
	finalTx, err := builder.Build()
	if err != nil {
		// Fallback to original transaction if builder fails
		tx.Description = cleanDesc
		tx.Payee = payee
		tx.CreditDebit = creditDebit
		if tx.ValueDate.IsZero() {
			tx.ValueDate = tx.Date
		}
		if tx.Category == "" {
			tx.Category = models.CategoryUncategorized
		}
		finalTx = *tx
	}

	// Generate a unique key for deduplication
	key := fmt.Sprintf("%s-%s-%s", finalTx.Date.Format(dateutils.DateLayoutISO), finalTx.Description, finalTx.Amount.String())

	// Only add if we haven't seen this transaction before
	if !seen[key] {
		seen[key] = true
		*transactions = append(*transactions, finalTx)
	}
}

// sortTransactions sorts transactions by date
func sortTransactions(transactions []models.Transaction) {
	sort.Slice(transactions, func(i, j int) bool {
		// Compare time.Time values directly
		return transactions[i].Date.Before(transactions[j].Date)
	})
}

// deduplicateTransactions removes duplicate transactions based on date, description, and amount
func deduplicateTransactions(transactions []models.Transaction) []models.Transaction {
	if len(transactions) <= 1 {
		return transactions
	}

	seen := make(map[string]bool)
	var result []models.Transaction

	for _, tx := range transactions {
		// Create a key using date, description, and amount
		key := fmt.Sprintf("%s|%s|%s", tx.Date, tx.Description, tx.Amount.String())

		if !seen[key] {
			seen[key] = true
			result = append(result, tx)
		}
	}

	return result
}

// determineCreditDebit determines if a transaction is a credit or debit based on the description
func determineCreditDebit(description string) string {
	lowerDesc := strings.ToLower(description)

	// Check for explicit keywords
	if strings.Contains(lowerDesc, "withdrawal") ||
		strings.Contains(lowerDesc, "payment") ||
		strings.Contains(lowerDesc, "purchase") {
		return models.TransactionTypeDebit
	}

	if strings.Contains(lowerDesc, "deposit") ||
		strings.Contains(lowerDesc, "credit") ||
		strings.Contains(lowerDesc, "refund") {
		return models.TransactionTypeCredit
	}

	// Default to debit if we can't determine
	return models.TransactionTypeDebit
}
