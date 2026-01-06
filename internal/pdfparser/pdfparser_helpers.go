// Package pdfparser provides functionality to parse PDF files and extract transaction data.
package pdfparser

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
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

// getDefaultLogger returns a default logger for backward compatibility
func getDefaultLogger() logging.Logger {
	return logging.NewLogrusAdapter("info", "text")
}

// extractTextFromPDF is a function variable to allow test mocking
// Note: This is intentionally a package-level variable to support testing
var extractTextFromPDF = extractTextFromPDFImpl

func extractTextFromPDFImpl(pdfFile string) (string, error) {
	// Create a temporary file to store the text output
	tempFile := pdfFile + ".txt"

	// Use pdftotext command-line tool to extract text
	// Add the -raw option to preserve the original text layout
	cmd := exec.Command("pdftotext", "-layout", "-raw", pdfFile, tempFile) // #nosec G204 -- Expected subprocess for PDF text extraction
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running pdftotext: %w", err)
	}

	// Read the extracted text
	output, err := os.ReadFile(tempFile) // #nosec G304 -- reading from app-generated temp file
	if err != nil {
		return "", fmt.Errorf("error reading extracted text: %w", err)
	}

	// Remove the temporary file
	if err := os.Remove(tempFile); err != nil {
		getDefaultLogger().WithError(err).Warn("Failed to remove temporary file")
	}

	return string(output), nil
}

// parseTransactionsWithCategorizer parses transaction data from PDF text content and applies categorization
func parseTransactionsWithCategorizer(lines []string, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
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
			getDefaultLogger().Debug("Detected Viseca PDF format - header pattern matched")
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
		return parseVisecaTransactionsWithCategorizer(lines, logger, categorizer)
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
	processedTransactions := common.ProcessTransactionsWithCategorizationStats(
		transactions, logger, categorizer, "PDF")

	logger.Info("Extracted transactions from PDF",
		logging.Field{Key: "count", Value: len(processedTransactions)})
	return processedTransactions, nil
}

// parseVisecaTransactionsWithCategorizer is a specialized parser for Viseca credit card statements with categorization
func parseVisecaTransactionsWithCategorizer(lines []string, logger logging.Logger, categorizer models.TransactionCategorizer) ([]models.Transaction, error) {
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
	processedTransactions := common.ProcessTransactionsWithCategorizationStats(
		transactions, logger, categorizer, "PDF-Viseca")

	logger.Info("Extracted transactions from Viseca PDF",
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

// extractAmount extracts the amount from a transaction line
func extractAmount(text string) (string, decimal.Decimal, bool) {
	// Match patterns like "123.45 USD" or "USD 123.45"
	amountMatch := amountGenericPattern.FindString(text)

	if amountMatch != "" {
		amountStr := amountMatch
		// Default to debit unless explicitly marked as credit or has a plus sign
		isCredit := strings.Contains(strings.ToLower(text), "credit") ||
			strings.Contains(strings.ToLower(text), "incoming") ||
			strings.Contains(text, "+")
		decimalAmount := models.ParseAmount(amountStr)
		return amountStr, decimalAmount, isCredit
	}

	return "", decimal.Zero, false
}

// cleanDescription removes unwanted elements from the description
func cleanDescription(description string) string {
	// Replace multiple spaces with a single space
	description = multipleSpacePattern.ReplaceAllString(description, " ")

	// Remove leading/trailing whitespace
	description = strings.TrimSpace(description)

	// Remove common noise phrases using pre-compiled patterns
	for _, pattern := range noisePhrasePatterns {
		description = pattern.ReplaceAllString(description, "")
	}

	return description
}

// extractPayee extracts the payee/merchant from a description
func extractPayee(description string) string {
	// Try pre-compiled payee patterns
	for _, re := range payeePatterns {
		matches := re.FindStringSubmatch(description)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// If no pattern matches, try to extract the most likely merchant name
	words := strings.Fields(description)

	// Skip transaction-related terms
	skipWords := map[string]bool{
		"transaction": true, "date": true, "amount": true, "credit": true,
		"debit": true, "card": true, "payment": true, "transfer": true,
		"fee": true, "charge": true, "balance": true, "available": true,
		"withdrawal": true, "deposit": true, "reference": true,
	}

	// Find first sequence of capitalized words
	var merchantWords []string
	inMerchant := false

	for _, word := range words {
		word = strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})

		if word == "" || skipWords[strings.ToLower(word)] {
			continue
		}

		// Check if word is mostly uppercase or a proper noun
		isSignificant := strings.ToUpper(word) == word || (unicode.IsUpper(rune(word[0])) && strings.ToLower(word[1:]) == word[1:])

		if isSignificant {
			merchantWords = append(merchantWords, word)
			inMerchant = true
		} else if inMerchant {
			// Stop collecting words once we hit non-significant words after starting collection
			break
		}
	}

	if len(merchantWords) > 0 {
		return strings.Join(merchantWords, " ")
	}

	// If all else fails, return a shortened version of the description
	if len(description) > 30 {
		return description[:30] + "..."
	}
	return description
}

// extractMerchant extracts the merchant from a description
func extractMerchant(description string) string {
	// Try pre-compiled merchant patterns
	for _, re := range merchantPatterns {
		matches := re.FindStringSubmatch(description)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Check for common transaction patterns and extract merchant name
	words := strings.Fields(description)
	for i, word := range words {
		if strings.HasPrefix(strings.ToLower(word), "card") && i+1 < len(words) {
			// "Card purchase at MERCHANT"
			if strings.ToLower(word) == "card" && i+2 < len(words) &&
				(strings.ToLower(words[i+1]) == "purchase" || strings.ToLower(words[i+1]) == "payment") &&
				strings.ToLower(words[i+2]) == "at" && i+3 < len(words) {
				return strings.Join(words[i+3:min(i+6, len(words))], " ")
			}
		}
	}

	return ""
}

// sortTransactions sorts transactions by date
func sortTransactions(transactions []models.Transaction) {
	sort.Slice(transactions, func(i, j int) bool {
		// Compare time.Time values directly
		return transactions[i].Date.Before(transactions[j].Date)
	})
}

// preProcessText performs initial cleanup and restructuring of PDF text
func preProcessText(text string) string {
	// Replace non-standard line breaks and ensure proper line splitting
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Replace multiple consecutive spaces with a single space
	// But preserve alignment for amount values which are typically right-aligned
	text = tripleSpacePattern.ReplaceAllString(text, "   ")

	// Remove empty lines
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	// Join the lines back together
	return strings.Join(lines, "\n")
}

// containsMerchantIdentifier checks if a line contains common merchant identifiers
func containsMerchantIdentifier(line string) bool {
	identifiers := []string{
		"merchant", "vendor", "shop", "store", "payee name",
		"business", "company", "paid to", "payment to",
	}

	lowerLine := strings.ToLower(line)
	for _, id := range identifiers {
		if strings.Contains(lowerLine, id) {
			return true
		}
	}

	// Look for "Card purchase at" pattern
	if cardPurchasePattern.MatchString(line) {
		return true
	}

	return false
}

// containsAmount checks if a line contains a monetary amount
func containsAmount(line string) bool {
	return amountCurrencyPattern.MatchString(line)
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

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
