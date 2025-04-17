// Package pdfparser provides functionality to parse PDF files and extract transaction data.
package pdfparser

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/categorizer"
)

// Define extractTextFromPDF as a variable holding a function
var extractTextFromPDF = func(pdfFile string) (string, error) {
	// Create a temporary file to store the text output
	tempFile := pdfFile + ".txt"
	
	// Use pdftotext command-line tool to extract text
	cmd := exec.Command("pdftotext", "-layout", pdfFile, tempFile)
	err := cmd.Run()
	if err != nil {
		log.WithError(err).Error("Failed to run pdftotext command")
		return "", fmt.Errorf("error running pdftotext: %w", err)
	}
	
	// Read the extracted text
	output, err := os.ReadFile(tempFile)
	if err != nil {
		log.WithError(err).Error("Failed to read text output")
		return "", fmt.Errorf("error reading extracted text: %w", err)
	}
	
	// Remove the temporary file
	os.Remove(tempFile)
	
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
	
	// Preprocess the lines to identify transaction blocks
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Skip header lines
		if i < 5 && !containsAmount(line) {
			continue
		}
		
		// Identify transaction start by date pattern
		datePattern := regexp.MustCompile(`^\d{2}\.\d{2}\.\d{2,4}`)
		if datePattern.MatchString(line) {
			// Finalize previous transaction if we're in one
			if inTransaction {
				finalizeTransaction(&currentTx, &description, merchant, seen, &transactions)
			}
			
			// Start a new transaction
			inTransaction = true
			currentTx = models.Transaction{}
			description.Reset()
			merchant = ""
			
			// Extract date
			dateParts := strings.Fields(line)
			if len(dateParts) > 0 {
				currentTx.Date = formatDate(dateParts[0])
				// Try to extract value date if present
				if len(dateParts) > 1 && datePattern.MatchString(dateParts[1]) {
					currentTx.ValueDate = formatDate(dateParts[1])
				}
			}
			
			// Extract amount and currency if on the same line
			extractCurrencyAndAmount(line, &currentTx)
			
			// Add to description
			description.WriteString(line)
			description.WriteString(" ")
			
			// Try to extract merchant name
			merchant = extractMerchant(line)
		} else if inTransaction {
			// Continuing a transaction - append to description
			description.WriteString(line)
			description.WriteString(" ")
			
			// Extract amount and currency if not yet set
			if currentTx.Amount == "" {
				extractCurrencyAndAmount(line, &currentTx)
			}
			
			// Try to extract merchant if not yet found
			if merchant == "" && containsMerchantIdentifier(line) {
				merchant = extractMerchant(line)
			}
		}
	}
	
	// Finalize the last transaction if needed
	if inTransaction {
		finalizeTransaction(&currentTx, &description, merchant, seen, &transactions)
	}
	
	// Sort transactions by date
	sortTransactions(transactions)
	
	// Remove duplicates
	transactions = deduplicateTransactions(transactions)
	
	log.WithField("count", len(transactions)).Info("Extracted transactions from PDF")
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
	key := fmt.Sprintf("%s-%s-%s", tx.Date, tx.Description, tx.Amount)
	
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
				IsDebtor:  isDebtor,
				Amount:    fmt.Sprintf("%s %s", tx.Amount, tx.Currency),
				Date:      tx.Date,
				Info:      tx.Description,
			}
			
			// Try to categorize using the categorizer
			if category, err := categorizer.CategorizeTransaction(catTx); err == nil {
				tx.Category = category.Name
			}
		}
		
		*transactions = append(*transactions, *tx)
	}
}

// extractCurrencyAndAmount extracts the currency and amount from a transaction line
func extractCurrencyAndAmount(line string, tx *models.Transaction) {
	// Match patterns like "123.45 USD" or "USD 123.45"
	amountPattern := regexp.MustCompile(`(\d+[\.,]\d+)\s*([A-Z]{3})`)
	currencyFirstPattern := regexp.MustCompile(`([A-Z]{3})\s*(\d+[\.,]\d+)`)
	
	// Try first pattern
	matches := amountPattern.FindStringSubmatch(line)
	if len(matches) > 2 {
		tx.Amount = matches[1]
		tx.Currency = matches[2]
		// If the description contains negative indicators, mark as debit
		if strings.Contains(strings.ToLower(line), "withdrawal") ||
			strings.Contains(strings.ToLower(line), "debit") {
			tx.CreditDebit = "DBIT"
		} else if strings.Contains(strings.ToLower(line), "deposit") ||
			strings.Contains(strings.ToLower(line), "credit") {
			tx.CreditDebit = "CRDT"
		}
		return
	}
	
	// Try second pattern
	matches = currencyFirstPattern.FindStringSubmatch(line)
	if len(matches) > 2 {
		tx.Currency = matches[1]
		tx.Amount = matches[2]
	}
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
	// Replace all tab characters with spaces
	text = strings.ReplaceAll(text, "\t", " ")
	
	// Replace multiple spaces with a single space
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	
	// Convert all dates from DD.MM.YY to DD.MM.YYYY format for consistency
	datePattern := regexp.MustCompile(`(\d{2}\.\d{2}\.)(\d{2})(\s|$)`)
	text = datePattern.ReplaceAllStringFunc(text, func(match string) string {
		datePattern := regexp.MustCompile(`(\d{2}\.\d{2}\.)(\d{2})`)
		parts := datePattern.FindStringSubmatch(match)
		if len(parts) > 2 {
			year := parts[2]
			fullYear := convertToFullYear(year)
			return parts[1] + fullYear
		}
		return match
	})
	
	// Restructure common PDF formatting issues
	
	// 1. Merge lines that are part of the same transaction
	lines := strings.Split(text, "\n")
	var mergedLines []string
	var currentLine strings.Builder
	
	datePattern = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}`)
	
	for _, line := range lines {
		if datePattern.MatchString(line) {
			// This line starts a new transaction
			if currentLine.Len() > 0 {
				mergedLines = append(mergedLines, currentLine.String())
				currentLine.Reset()
			}
			currentLine.WriteString(line)
		} else if currentLine.Len() > 0 && len(line) > 0 {
			// Continuation of current transaction
			currentLine.WriteString(" ")
			currentLine.WriteString(line)
		} else if len(line) > 0 {
			// Standalone line not part of a transaction
			mergedLines = append(mergedLines, line)
		}
	}
	
	// Add the last line if not empty
	if currentLine.Len() > 0 {
		mergedLines = append(mergedLines, currentLine.String())
	}
	
	return strings.Join(mergedLines, "\n")
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

// convertToFullYear converts a DD.MM.YY date to DD.MM.YYYY format
func convertToFullYear(year string) string {
	// Convert YY to YYYY
	twoDigitYear, err := strconv.Atoi(year)
	if err != nil {
		return year
	}
	
	// Assume 20YY for years below 50, 19YY for years 50 and above
	if twoDigitYear < 50 {
		return fmt.Sprintf("20%02d", twoDigitYear)
	} else {
		return fmt.Sprintf("19%02d", twoDigitYear)
	}
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
		key := fmt.Sprintf("%s|%s|%s", tx.Date, tx.Description, tx.Amount)
		
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

// max returns the larger of x or y
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
