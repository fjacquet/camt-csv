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

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// Define extractTextFromPDF as a variable holding a function
var extractTextFromPDF = extractTextFromPDFImpl

func extractTextFromPDFImpl(pdfFile string) (string, error) {
	// Create a temporary file to store the text output
	tempFile := pdfFile + ".txt"

	// Use pdftotext command-line tool to extract text
	// Add the -raw option to preserve the original text layout
	cmd := exec.Command("pdftotext", "-layout", "-raw", pdfFile, tempFile) // #nosec G204 -- Expected subprocess for PDF text extraction
	err := cmd.Run()
	if err != nil {
		logrus.WithError(err).Error("Failed to run pdftotext command")
		return "", fmt.Errorf("error running pdftotext: %w", err)
	}

	// Read the extracted text
	output, err := os.ReadFile(tempFile)
	if err != nil {
		logrus.WithError(err).Error("Failed to read text output")
		return "", fmt.Errorf("error reading extracted text: %w", err)
	}

	// Remove the temporary file
	if err := os.Remove(tempFile); err != nil {
		log.WithError(err).Warn("Failed to remove temporary file")
	}

	return string(output), nil
}

// parseTransactions parses transaction data from PDF text content
func parseTransactions(lines []string) ([]models.Transaction, error) {
	var transactions []models.Transaction
	var currentTx models.Transaction
	var description strings.Builder
	var merchant string
	seen := make(map[string]bool)

	inTransaction := false
	isVisecaFormat := false

	// Check if this is a Viseca file format (look for typical Viseca PDF headers and patterns)
	for _, line := range lines {
		// Check for the column headers that appear in Viseca statements
		if strings.Contains(line, "Date") && strings.Contains(line, "valeur") &&
			strings.Contains(line, "Détails") && strings.Contains(line, "Monnaie") &&
			strings.Contains(line, "Montant") {
			isVisecaFormat = true
			logrus.Debug("Detected Viseca PDF format - header pattern matched")
			break
		}

		// Also look for Viseca card number pattern
		if strings.Contains(line, "Visa Gold") || strings.Contains(line, "Visa Platinum") ||
			strings.Contains(line, "Mastercard") || strings.Contains(line, "XXXX") {
			isVisecaFormat = true
			logrus.Debug("Detected Viseca PDF format - card pattern matched")
			break
		}

		// Check for typical Viseca statement features
		if strings.Contains(line, "Montant total dernier relevé") ||
			strings.Contains(line, "Votre paiement - Merci") {
			isVisecaFormat = true
			logrus.Debug("Detected Viseca PDF format - statement pattern matched")
			break
		}
	}

	logrus.WithField("isVisecaFormat", isVisecaFormat).Debug("Format detection result")

	// For Viseca format, use a specialized transaction extraction approach
	if isVisecaFormat {
		return parseVisecaTransactions(lines)
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
		datePattern := regexp.MustCompile(`^\d{2}\.\d{2}\.\d{2,4}`)
		if datePattern.MatchString(trimmedLine) {
			logrus.WithField("line", trimmedLine).Debug("Found potential transaction start")

			// Finalize previous transaction if we're in one
			if inTransaction {
				logrus.Debug("Finalizing previous transaction")
				finalizeTransaction(&currentTx, &description, merchant, seen, &transactions)
			}

			// Start a new transaction
			inTransaction = true
			currentTx = models.Transaction{}
			description.Reset()
			// merchant would be empty string here, but we don't need to assign it

			// Extract date
			fields := strings.Fields(trimmedLine)
			if len(fields) > 0 {
				currentTx.Date = formatDate(fields[0])
				logrus.WithField("date", currentTx.Date).Debug("Extracted transaction date")

				// Try to extract value date if present
				if len(fields) > 1 && datePattern.MatchString(fields[1]) {
					currentTx.ValueDate = formatDate(fields[1])
				}
			}

			// Extract amount and currency
			_, amount, isCredit := extractAmount(trimmedLine)
			currentTx.Amount = amount
			currentTx.CreditDebit = "DBIT"
			if isCredit {
				currentTx.CreditDebit = "CRDT"
			}

			// Add to description
			description.WriteString(trimmedLine)

			// Try to extract merchant name
			merchant = extractMerchant(trimmedLine)
		} else if inTransaction {
			// Continuing a transaction - append to description
			logrus.WithField("line", trimmedLine).Debug("Continuing transaction with line")
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
		logrus.Debug("Finalizing last transaction")
		finalizeTransaction(&currentTx, &description, merchant, seen, &transactions)
	}

	// Sort transactions by date
	sortTransactions(transactions)

	// Remove duplicates
	transactions = deduplicateTransactions(transactions)

	logrus.WithField("count", len(transactions)).Info("Extracted transactions from PDF")
	return transactions, nil
}

// parseVisecaTransactions is a specialized parser for Viseca credit card statements
func parseVisecaTransactions(lines []string) ([]models.Transaction, error) {
	logrus.WithField("lineCount", len(lines)).Debug("Processing Viseca PDF with specialized parser")

	var transactions []models.Transaction
	var currentCategory string

	// For debugging, dump the first few lines
	for i := 0; i < min(20, len(lines)); i++ {
		logrus.WithField("line", lines[i]).Debug("Sample line from PDF")
	}

	// Process lines
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Skip Viseca header lines
		if strings.Contains(line, "Date de") || (strings.Contains(line, "Date") && strings.Contains(line, "valeur") && strings.Contains(line, "Détails") && strings.Contains(line, "Montant")) {
			logrus.Debug("Skipping header line")
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
		datePattern := regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{2,4})`)
		if !datePattern.MatchString(line) {
			// Not a transaction line, could be a category or additional info
			// Store it to potentially attach to the previous transaction
			if strings.TrimSpace(line) != "" && !strings.Contains(line, "XXXX") {
				currentCategory = strings.TrimSpace(line)
				logrus.WithField("category", currentCategory).Debug("Found potential category line")
			}
			continue
		}

		// This looks like a transaction line - extract the components

		// Extract transaction date and optional value date (DD.MM.YY(YY))
		dateValuePattern := regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{2,4})(?:\s+(\d{2}\.\d{2}\.\d{2,4}))?`)
		dateValueMatch := dateValuePattern.FindStringSubmatch(line)
		if len(dateValueMatch) < 2 || dateValueMatch[1] == "" {
			logrus.WithField("line", line).Debug("Invalid transaction line format - missing date")
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
			logrus.WithField("line", line).Debug("Skipping summary line")
			continue
		}

		// The amount is typically right-aligned at the end
		// Look for a number pattern at the end, possibly followed by a minus sign
		amountPattern := regexp.MustCompile(`(\d+[\'.,]\d+)\s*(-)?$`)
		amountMatch := amountPattern.FindStringSubmatch(remainingLine)

		if len(amountMatch) < 2 {
			logrus.WithField("line", line).Debug("Could not extract amount from transaction line")
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
			logrus.WithField("line", line).Debug("Could not determine description boundaries")
			continue
		}

		description := strings.TrimSpace(remainingLine[:descriptionEndPos])

		// Check for foreign currency indicators
		var originalCurrency, originalAmount string
		currencyMatch := regexp.MustCompile(`([A-Z]{3})\s+(\d+[\'.,]\d+)`).FindStringSubmatch(description)
		if len(currencyMatch) > 2 {
			originalCurrency = currencyMatch[1]
			originalAmount = strings.ReplaceAll(currencyMatch[2], "'", "")

			// Clean up description
			description = strings.Replace(description, currencyMatch[0], "", 1)
			description = strings.TrimSpace(description)

			logrus.WithFields(logrus.Fields{
				"currency": originalCurrency,
				"amount":   originalAmount,
			}).Debug("Found foreign currency transaction")
		}

		// Create the transaction
		tx := models.Transaction{
			Date:             formatDate(txDate),
			ValueDate:        formatDate(valueDate),
			Description:      description,
			Payee:            description,
			Amount:           models.ParseAmount(amount),
			Currency:         "CHF",
			OriginalCurrency: originalCurrency,
			OriginalAmount:   models.ParseAmount(originalAmount),
		}

		// Set credit/debit indicator
		if isCredit {
			tx.CreditDebit = "CRDT"
		} else {
			tx.CreditDebit = "DBIT"
		}

		// Attach category if we have one
		if currentCategory != "" {
			tx.Description = tx.Description + " - " + currentCategory
			logrus.WithField("category", currentCategory).Debug("Added category to transaction")
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
			if datePattern.MatchString(nextLine) {
				break
			}

			// Look for exchange rate information
			if strings.Contains(nextLine, "Taux de conversion") {
				exchangeRateMatch := regexp.MustCompile(`Taux de conversion\s+(\d+\.\d+)`).FindStringSubmatch(nextLine)
				if len(exchangeRateMatch) > 1 {
					tx.ExchangeRate = models.ParseAmount(exchangeRateMatch[1])
					exchangeRateFound = true
					logrus.WithField("exchangeRate", tx.ExchangeRate.String()).Debug("Found exchange rate")
				}
			}

			// Look for processing fee information
			if strings.Contains(nextLine, "Frais de traitement") {
				feeMatch := regexp.MustCompile(`Frais de traitement\s+.+?\s+(\d+\.\d+)`).FindStringSubmatch(nextLine)
				if len(feeMatch) > 1 {
					tx.Fees = models.ParseAmount(feeMatch[1])
					processingFeeFound = true
					logrus.WithField("fees", tx.Fees.String()).Debug("Found processing fees")
				}
			}
		}

		// Try to categorize the transaction
		isDebtor := tx.CreditDebit == "DBIT"

		// Create a categorizer.Transaction
		catTx := categorizer.Transaction{
			PartyName: tx.Payee,
			IsDebtor:  isDebtor,
			Amount:    fmt.Sprintf("%s %s", tx.Amount.String(), tx.Currency),
			Date:      tx.Date,
			Info:      tx.Description,
		}

		// Try to categorize
		if category, err := categorizer.CategorizeTransaction(catTx); err == nil {
			tx.Category = category.Name
			logrus.WithField("category", tx.Category).Debug("Categorized transaction")
		}

		// Add transaction to list
		transactions = append(transactions, tx)
		logrus.WithFields(logrus.Fields{
			"date":        tx.Date,
			"description": tx.Description,
			"amount":      tx.Amount.String(),
		}).Debug("Added transaction")
	}

	// Log the number of transactions found
	logrus.WithField("count", len(transactions)).Info("Extracted transactions from Viseca PDF")
	return transactions, nil
}

// finalizeTransaction finalizes a transaction and adds it to the list of transactions
func finalizeTransaction(tx *models.Transaction, desc *strings.Builder, merchant string, seen map[string]bool, transactions *[]models.Transaction) {
	// Set description and clean it
	tx.Description = cleanDescription(desc.String())

	// Extract and set merchant/payee
	if merchant != "" {
		tx.Payee = merchant
	} else {
		tx.Payee = extractPayee(tx.Description)
	}

	// Set credit/debit indicator if not already set
	if tx.CreditDebit == "" {
		tx.CreditDebit = determineCreditDebit(tx.Description)
	}

	// Fill in empty fields with placeholders to avoid null values
	if tx.ValueDate == "" {
		tx.ValueDate = tx.Date
	}

	// Generate a unique key for deduplication
	key := fmt.Sprintf("%s-%s-%s", tx.Date, tx.Description, tx.Amount.String())

	// Only add if we haven't seen this transaction before
	if !seen[key] {
		seen[key] = true

		// Try to categorize the transaction
		if tx.Category == "" {
			// Determine if the transaction is a debit or credit
			isDebtor := tx.CreditDebit == "DBIT"

			// Create a categorizer.Transaction from our transaction data
			catTx := categorizer.Transaction{
				PartyName: func() string {
					if isDebtor {
						return tx.Payee
					}
					return tx.Payer
				}(),
				IsDebtor: isDebtor,
				Amount:   fmt.Sprintf("%s %s", tx.Amount.String(), tx.Currency),
				Date:     tx.Date,
				Info:     tx.Description,
			}

			// Try to categorize using the categorizer
			if category, err := categorizer.CategorizeTransaction(catTx); err == nil {
				tx.Category = category.Name
			}
		}

		*transactions = append(*transactions, *tx)
	}
}

// extractAmount extracts the amount from a transaction line
func extractAmount(text string) (string, decimal.Decimal, bool) {
	// Match patterns like "123.45 USD" or "USD 123.45"
	amountPattern := regexp.MustCompile(`[+-]?\s*(\d+'?)+[,\.]?\d*\s*(CHF|EUR|USD|\$|€)?`)
	amountMatch := amountPattern.FindString(text)

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
	description = regexp.MustCompile(`\s+`).ReplaceAllString(description, " ")

	// Remove leading/trailing whitespace
	description = strings.TrimSpace(description)

	// Remove common noise phrases
	noisePhrases := []string{"transaction date", "value date", "description", "amount"}
	for _, phrase := range noisePhrases {
		description = regexp.MustCompile(`(?i)`+phrase).ReplaceAllString(description, "")
	}

	return description
}

// extractPayee extracts the payee/merchant from a description
func extractPayee(description string) string {
	// Common patterns for merchant names in transaction descriptions
	patterns := []string{
		`(?i)payee:\s*([^,;]+)`,
		`(?i)to:\s*([^,;]+)`,
		`(?i)merchant:\s*([^,;]+)`,
		`(?i)payment to:\s*([^,;]+)`,
		`(?i)paid to:\s*([^,;]+)`,
		`(?i)transfer to:\s*([^,;]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
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
	// Common patterns for merchant names in transaction descriptions
	patterns := []string{
		`(?i)merchant:\s*([^,;]+)`,
		`(?i)vendor:\s*([^,;]+)`,
		`(?i)shop:\s*([^,;]+)`,
		`(?i)store:\s*([^,;]+)`,
		`(?i)payee name:\s*([^,;]+)`,
		`(?i)business:\s*([^,;]+)`,
		`(?i)company:\s*([^,;]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
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
		// Parse dates for comparison
		dateI, _ := time.Parse("02.01.2006", transactions[i].Date)
		dateJ, _ := time.Parse("02.01.2006", transactions[j].Date)
		return dateI.Before(dateJ)
	})
}

// preProcessText performs initial cleanup and restructuring of PDF text
func preProcessText(text string) string {
	// Replace non-standard line breaks and ensure proper line splitting
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Replace multiple consecutive spaces with a single space
	// But preserve alignment for amount values which are typically right-aligned
	text = regexp.MustCompile(`[ ]{3,}`).ReplaceAllString(text, "   ")

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
	cardPurchasePattern := regexp.MustCompile(`(?i)card\s+purchase\s+at`)
	if cardPurchasePattern.MatchString(line) {
		return true
	}

	return false
}

// containsAmount checks if a line contains a monetary amount
func containsAmount(line string) bool {
	amountPattern := regexp.MustCompile(`\d+[\.,]\d+\s*[A-Z]{3}|[A-Z]{3}\s*\d+[\.,]\d+`)
	return amountPattern.MatchString(line)
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
		return "DBIT"
	}

	if strings.Contains(lowerDesc, "deposit") ||
		strings.Contains(lowerDesc, "credit") ||
		strings.Contains(lowerDesc, "refund") {
		return "CRDT"
	}

	// Default to debit if we can't determine
	return "DBIT"
}

// formatDate formats a date string to a standard format
func formatDate(date string) string {
	// Remove any non-digit or dot characters
	re := regexp.MustCompile(`[^\d.]`)
	date = re.ReplaceAllString(date, "")

	// Try to identify the format
	formats := []string{
		"02.01.2006", // DD.MM.YYYY
		"02.01.06",   // DD.MM.YY
		"2/1/2006",   // M/D/YYYY
		"1/2/2006",   // D/M/YYYY
		"2006-01-02", // YYYY-MM-DD
	}

	var t time.Time
	var err error

	for _, format := range formats {
		t, err = time.Parse(format, date)
		if err == nil {
			break
		}
	}

	// If we failed to parse any format, just return the original
	if err != nil {
		return date
	}

	// Return in the standard DD.MM.YYYY format
	return t.Format("02.01.2006")
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
