// Package pdfparser provides functionality to parse PDF files and extract transaction data.
package pdfparser

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/pkg/categorizer"
)

// ConvertPDFToCSV converts a PDF file to CSV format
func ConvertPDFToCSV(pdfFile string, csvFile string) error {
	// Extract text from PDF
	text, err := extractTextFromPDF(pdfFile)
	if err != nil {
		return fmt.Errorf("error extracting text from PDF: %w", err)
	}

	// Write raw PDF text to debug file
	debugFile := "debug_pdf_extract.txt"
	err = os.WriteFile(debugFile, []byte(text), 0644)
	if err != nil {
		fmt.Printf("Warning: Failed to write debug file: %v\n", err)
	} else {
		fmt.Printf("Wrote raw PDF text to %s\n", debugFile)
	}

	// Write preprocessed text to debug file with line numbers
	preprocessedDebugFile := "debug_pdf_preprocessed.txt"
	var preprocessedLines []string
	for i, line := range strings.Split(text, "\n") {
		preprocessedLines = append(preprocessedLines, fmt.Sprintf("Line %d: %s", i, line))
	}
	err = os.WriteFile(preprocessedDebugFile, []byte(strings.Join(preprocessedLines, "\n")), 0644)
	if err != nil {
		fmt.Printf("Warning: Failed to write preprocessed debug file: %v\n", err)
	} else {
		fmt.Printf("Wrote preprocessed PDF text to %s\n", preprocessedDebugFile)
	}

	// Print the first 500 characters of text for debugging
	if len(text) > 500 {
		fmt.Printf("PDF Text (first 500 chars): %s...\n", text[:500])
	} else {
		fmt.Printf("PDF Text: %s\n", text)
	}

	// Print number of lines
	lines := strings.Split(text, "\n")
	fmt.Printf("Number of lines in PDF: %d\n", len(lines))

	// Print a sample of lines for debugging
	for i, line := range lines {
		if i >= 10 {
			break
		}
		fmt.Printf("Preprocessed Line %d: %s\n", i, line)
	}

	// Parse transactions from the extracted text
	transactions, err := parseTransactions(lines)
	if err != nil {
		return fmt.Errorf("error parsing transactions: %w", err)
	}

	fmt.Printf("Number of transactions extracted: %d\n", len(transactions))

	// Write transactions to CSV
	err = writeTransactionsToCSV(transactions, csvFile)
	if err != nil {
		return fmt.Errorf("error writing transactions to CSV: %w", err)
	}

	return nil
}

// extractTextFromPDF extracts text content from a PDF file using the pdftotext CLI tool
func extractTextFromPDF(pdfFile string) (string, error) {
	// Create a temporary file to store the text output
	tempFile := pdfFile + ".txt"

	// Run pdftotext command with layout option to preserve the format
	cmd := exec.Command("pdftotext", "-layout", pdfFile, tempFile)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run pdftotext: %w", err)
	}

	// Read the text from the temporary file
	textBytes, err := os.ReadFile(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to read output file: %w", err)
	}

	// Clean up the temporary file
	os.Remove(tempFile)

	return string(textBytes), nil
}

// parseTransactions parses transaction data from PDF text content
func parseTransactions(lines []string) ([]models.Transaction, error) {
	// Initialize
	var transactions []models.Transaction
	
	var currentTx *models.Transaction
	var transactionStarted bool
	var currentDate string
	var currentValueDate string
	var currentDesc strings.Builder
	var currentMerchant string
	
	// Track processed transactions to avoid duplicates
	seen := make(map[string]bool)
	
	// Improved date pattern for transaction detection
	datePattern := regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{2})\s+(\d{2}\.\d{2}\.\d{2})`)
	
	// Pattern for markers to skip
	skipLinePattern := regexp.MustCompile(`Taux de conversion|Frais de traitement`)
	
	// Pattern for end markers
	endOfSectionPattern := regexp.MustCompile(`Total intermédiaire|Total carte`)
	
	// Start marker - after "Limite de carte" line, the transactions start
	limitDeCarteFound := false
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check for "Limite de carte" marker
		if strings.Contains(line, "Limite de carte") {
			limitDeCarteFound = true
			continue
		}
		
		// Wait until we see "Limite de carte" before starting to parse
		if !limitDeCarteFound {
			continue
		}
		
		// Check for end markers
		if endOfSectionPattern.MatchString(line) {
			// If we were processing a transaction, finalize it
			if currentTx != nil {
				finalizeTransaction(currentTx, &currentDesc, currentMerchant, seen, &transactions)
				currentTx = nil
				transactionStarted = false
			}
			continue
		}
		
		// Skip lines containing currency conversion or processing fees
		if skipLinePattern.MatchString(line) {
			continue
		}
		
		// Check if it's a new transaction line (starts with a date)
		dateMatches := datePattern.FindStringSubmatch(line)
		if len(dateMatches) > 0 {
			// If we were processing a transaction, finalize it before starting a new one
			if currentTx != nil {
				finalizeTransaction(currentTx, &currentDesc, currentMerchant, seen, &transactions)
			}
			
			// Start a new transaction
			currentDate = dateMatches[1]
			currentValueDate = dateMatches[2]
			
			// Reset description builder
			currentDesc = strings.Builder{}
			currentMerchant = ""
			transactionStarted = true
			
			// Extract the rest of the transaction details from the first line
			detailsStart := len(dateMatches[0])
			details := strings.TrimSpace(line[detailsStart:])
			
			// Create the transaction object
			currentTx = &models.Transaction{
				Date:        currentDate,
				ValueDate:   currentValueDate,
				Status:      "BOOK",  // Default status
				Payer:       "Viseca", // Default payer for credit card statements
				Category:    "",     // Will be determined later
				CreditDebit: "DBIT",   // Default to debit for credit card transactions
			}
			
			// Extract payee
			currentMerchant = extractMerchant(details)
			
			// Append details to description
			currentDesc.WriteString(details)
			
			// Extract currency and amount
			extractCurrencyAndAmount(details, currentTx)
		} else if transactionStarted && currentTx != nil {
			// This is likely a description line
			// Check if it's indented (starts with spaces)
			if strings.HasPrefix(lines[i], " ") {
				// Append to description
				if currentDesc.Len() > 0 {
					currentDesc.WriteString(" - ")
				}
				currentDesc.WriteString(line)
			}
		}
	}
	
	// Don't forget to add the last transaction if it exists
	if currentTx != nil {
		finalizeTransaction(currentTx, &currentDesc, currentMerchant, seen, &transactions)
	}
	
	// Sort transactions by date
	sort.Slice(transactions, func(i, j int) bool {
		iDate, _ := time.Parse("02.01.06", transactions[i].Date)
		jDate, _ := time.Parse("02.01.06", transactions[j].Date)
		return iDate.Before(jDate)
	})
	
	fmt.Printf("Number of transactions extracted: %d\n", len(transactions))
	return transactions, nil
}

// Helper function to finalize and add a transaction
func finalizeTransaction(tx *models.Transaction, desc *strings.Builder, merchant string, seen map[string]bool, transactions *[]models.Transaction) {
	// Create a unique key for this transaction
	txKey := fmt.Sprintf("%s_%s_%s", tx.Date, tx.ValueDate, merchant)
	
	// Only add if we haven't seen this transaction before
	if !seen[txKey] {
		// Extract just the category description from the full description
		fullDesc := desc.String()
		
		// Find the category description - typically after the last "-" in the string
		if idx := strings.LastIndex(fullDesc, " - "); idx != -1 {
			// The category is after the last " - "
			tx.Description = strings.TrimSpace(fullDesc[idx+3:])
		} else {
			// If no category found, use the full description
			tx.Description = fullDesc
		}
		
		// Set the payee
		tx.Payee = merchant
		if tx.Payee == "" {
			tx.Payee = extractMerchant(fullDesc)
		}
		
		// Use the categorizer system to assign the correct category
		// Credit card transactions are always debits (money flowing out)
		catTx := categorizer.Transaction{
			PartyName: tx.Payee,
			IsDebtor:  false, // For credit card, the payee is the creditor (we're paying them)
			Amount:    tx.Amount,
			Date:      tx.Date,
			Info:      tx.Description,
		}
		
		// Try to categorize the transaction
		category, err := categorizer.CategorizeTransaction(catTx)
		if err == nil {
			// Successfully categorized
			tx.Category = category.Name
		} else {
			// Fallback to a generic category if categorization fails
			tx.Category = "Uncategorized"
		}
		
		*transactions = append(*transactions, *tx)
		seen[txKey] = true
		fmt.Printf("Adding transaction: %s / %s - %s %s %s - Category: %s\n", 
			tx.Date, tx.ValueDate, tx.Payee, tx.Amount, tx.Currency, tx.Category)
	}
}

// Helper function to extract currency and amount from a transaction line
func extractCurrencyAndAmount(line string, tx *models.Transaction) {
	// Currency pattern
	currencyPattern := regexp.MustCompile(`(CHF|EUR|USD)`)
	currMatches := currencyPattern.FindAllString(line, -1)
	
	// Amount patterns - handle both formats:
	// 1. Direct amount at the end of the line
	// 2. Foreign amount and CHF amount side by side
	amountPattern := regexp.MustCompile(`(\d+'?\d*\.\d{2})`)
	amountMatches := amountPattern.FindAllString(line, -1)
	
	if len(currMatches) > 0 {
		tx.Currency = currMatches[len(currMatches)-1]
	} else {
		// Default to CHF if no currency specified
		tx.Currency = "CHF"
	}
	
	if len(amountMatches) > 0 {
		// If there are multiple amounts, the last one is typically the CHF amount
		amount := amountMatches[len(amountMatches)-1]
		// Clean up the amount (remove thousands separators)
		amount = strings.ReplaceAll(amount, "'", "")
		tx.Amount = amount
	}
}

// cleanDescription removes unwanted elements from the description
func cleanDescription(description string) string {
	// Remove common classification patterns in Viseca statements
	desc := regexp.MustCompile(`(Magasins de détail|Supermarchés|Services|Restaurants|alimentation|spécialisés|film/musique)`).ReplaceAllString(description, "")
	
	// Clean up exchange rate and fee information
	desc = regexp.MustCompile(`Frais de traitement [0-9.]+%`).ReplaceAllString(desc, "")
	desc = regexp.MustCompile(`Taux de conversion [0-9.]+ du`).ReplaceAllString(desc, "")
	
	// Remove extra spaces and trim
	desc = strings.Join(strings.Fields(desc), " ")
	desc = strings.TrimSpace(desc)
	
	return desc
}

// extractPayee extracts the payee/merchant from a description
func extractPayee(description string) string {
	// If description contains a comma, the part before it is usually the merchant
	parts := strings.SplitN(description, ",", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[0])
	}
	
	// Try to extract based on common patterns in Viseca statements
	// First word is often the merchant name
	fields := strings.Fields(description)
	if len(fields) > 0 {
		// Common merchant patterns
		merchantPattern := regexp.MustCompile(`^(AMZN|APPLE|NETFLIX|SBB|COOP|MIGROS|WWW\.|PAYPAL|FNAC|DG\s+Solutions|Muller|Manor|Boucherie|AWS|LAITERIE|MCDONALDS)`)
		if match := merchantPattern.FindString(description); match != "" {
			// Try to include more context for the merchant name
			merchantEndPos := strings.Index(description, ",")
			if merchantEndPos == -1 {
				// Look for location markers like "CH", "DE", etc.
				locationMarkers := []string{" CH", " DE", " FR", " LU", " US", " IE", " NL", " GB"}
				for _, marker := range locationMarkers {
					if pos := strings.Index(description, marker); pos != -1 {
						merchantEndPos = pos
						break
					}
				}
			}
			
			if merchantEndPos != -1 {
				return strings.TrimSpace(description[:merchantEndPos])
			}
			
			// Just return the first 1-3 words as merchant
			wordCount := min(3, len(fields))
			return strings.Join(fields[:wordCount], " ")
		}
	}
	
	// Return the full description if no patterns matched
	if len(description) > 30 {
		return strings.TrimSpace(description[:30])
	}
	return strings.TrimSpace(description)
}

// extractMerchant extracts the merchant from a description
func extractMerchant(description string) string {
	// If description contains a comma, the part before it is usually the merchant
	parts := strings.SplitN(description, ",", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[0])
	}
	
	// Try to extract based on common patterns in Viseca statements
	// First word is often the merchant name
	fields := strings.Fields(description)
	if len(fields) > 0 {
		// Common merchant patterns
		merchantPattern := regexp.MustCompile(`^(AMZN|APPLE|NETFLIX|SBB|COOP|MIGROS|WWW\.|PAYPAL|FNAC|DG\s+Solutions|Muller|Manor|Boucherie|AWS|LAITERIE|MCDONALDS)`)
		if match := merchantPattern.FindString(description); match != "" {
			// Try to include more context for the merchant name
			merchantEndPos := strings.Index(description, ",")
			if merchantEndPos == -1 {
				// Look for location markers like "CH", "DE", etc.
				locationMarkers := []string{" CH", " DE", " FR", " LU", " US", " IE", " NL", " GB"}
				for _, marker := range locationMarkers {
					if pos := strings.Index(description, marker); pos != -1 {
						merchantEndPos = pos
						break
					}
				}
			}
			
			if merchantEndPos != -1 {
				return strings.TrimSpace(description[:merchantEndPos])
			}
			
			// Just return the first 1-3 words as merchant
			wordCount := min(3, len(fields))
			return strings.Join(fields[:wordCount], " ")
		}
	}
	
	// Return the full description if no patterns matched
	if len(description) > 30 {
		return strings.TrimSpace(description[:30])
	}
	return strings.TrimSpace(description)
}

// sortTransactions sorts transactions by date
func sortTransactions(transactions []models.Transaction) {
	// Sort by date, newest first
	sort.Slice(transactions, func(i, j int) bool {
		dateI, _ := time.Parse("Jan 02, 2006", transactions[i].Date)
		dateJ, _ := time.Parse("Jan 02, 2006", transactions[j].Date)
		return dateI.After(dateJ)
	})
}

// preProcessText performs initial cleanup and restructuring of PDF text
func preProcessText(text string) string {
	// Pre-process the text if it seems to be a single line
	lines := strings.Split(text, "\n")

	// If we only have a few lines, try to split it based on date patterns
	if len(lines) < 5 {
		fmt.Println("PDF content appears to be a single line, attempting to split it...")

		// Create artificial line breaks before each date
		dateRegex := regexp.MustCompile(`(\d{2}\.\d{2}\.(?:\d{4}|\d{2}))`)

		// Find all date matches
		dateIndices := dateRegex.FindAllStringIndex(text, -1)

		if len(dateIndices) > 0 {
			// Create a new text with line breaks before each date
			var newText strings.Builder
			lastIndex := 0

			for _, indices := range dateIndices {
				// Add text up to this date
				if indices[0] > lastIndex {
					newText.WriteString(text[lastIndex:indices[0]])
					newText.WriteString("\n") // Add line break before date
				}

				// Add the date itself
				newText.WriteString(text[indices[0]:indices[1]])
				lastIndex = indices[1]
			}

			// Add remaining text
			if lastIndex < len(text) {
				newText.WriteString(text[lastIndex:])
			}

			// Now split the new text into lines
			text = newText.String()
			lines = strings.Split(text, "\n")
			fmt.Printf("After artificial line breaks: %d lines\n", len(lines))
		}
	}

	// Print first 10 lines after preprocessing
	for i, line := range lines[:10] {
		fmt.Printf("Preprocessed Line %d: %s\n", i, line)
	}

	return text
}

// containsMerchantIdentifier checks if a line contains common merchant identifiers
func containsMerchantIdentifier(line string) bool {
	merchantIndicators := []string{
		"CH", "DE", "FR", "LU", "US", "IE", "NL", "GB",
		"Coop", "Migros", "APPLE", "SBB", "NETFLIX", "AMAZON",
		"WWW", "HTTP", ".COM", "FNAC", "Minestrone", "MCDONALDS",
		"PAYPAL", "DROPBOX", "MANOR", "LAITERIE", "AVIRA", "AWS",
		"BACKBLAZE", "AMZN", "APPLELADEN", "Google", "Microsoft",
		"DG Solutions", "TICKETS", "Mobile Ticket", "Muller", "Stu",
		"GRUYERE", "Montreux", "Boucherie", "Charcuterie", "Rennaz",
		"Stations", "www.",
	}

	for _, indicator := range merchantIndicators {
		if strings.Contains(line, indicator) {
			return true
		}
	}

	return false
}

// containsAmount checks if a line contains a monetary amount
func containsAmount(line string) bool {
	amountPattern := regexp.MustCompile(`\d+\.\d{2}`)
	return amountPattern.MatchString(line)
}

// convertToFullYear converts a DD.MM.YY date to DD.MM.YYYY format
func convertToFullYear(date string) string {
	if len(date) == 8 { // DD.MM.YY format
		day := date[0:2]
		month := date[3:5]
		year := "20" + date[6:8] // Assume 20XX for modern credit card statements
		return day + "." + month + "." + year
	}
	return date // Return unchanged if not in expected format
}

// deduplicateTransactions removes duplicate transactions based on date, description, and amount
func deduplicateTransactions(transactions []models.Transaction) []models.Transaction {
	seen := make(map[string]bool)
	var result []models.Transaction

	for _, tx := range transactions {
		// Create a unique key for each transaction
		key := tx.Date + "|" + tx.Description + "|" + tx.Amount + "|" + tx.Currency

		if !seen[key] {
			seen[key] = true
			result = append(result, tx)
		}
	}

	return result
}

// determineCreditDebit determines if a transaction is a credit or debit based on the description and amount
func determineCreditDebit(description string, amount string) string {
	// Most transactions are debits for credit cards
	creditDebit := "DBIT"

	// Only mark as credit if it's explicitly a payment or refund
	if strings.Contains(description, "Votre paiement") ||
		strings.Contains(description, "remboursement") ||
		strings.Contains(description, "payment") ||
		// If the line has a negative sign with the amount
		strings.Contains(description, amount+"-") {
		creditDebit = "CRDT"
	}

	return creditDebit
}

// writeTransactionsToCSV writes transactions to a CSV file
func writeTransactionsToCSV(transactions []models.Transaction, csvFile string) error {
	file, err := os.Create(csvFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header matching the CAMT output format
	header := []string{
		"Date", "ValueDate", "Description", "BookkeepingNo", "Fund",
		"Amount", "Currency", "CreditDebit", "EntryReference", "AccountServicer",
		"BankTxCode", "Status", "Payee", "Payer", "IBAN",
		"NumberOfShares", "StampDuty", "Category",
	}
	err = writer.Write(header)
	if err != nil {
		return err
	}

	// Write transactions
	for _, transaction := range transactions {
		record := []string{
			transaction.Date,
			transaction.ValueDate,
			transaction.Description,
			transaction.BookkeepingNo,
			transaction.Fund,
			transaction.Amount,
			transaction.Currency,
			transaction.CreditDebit,
			transaction.EntryReference,
			transaction.AccountServicer,
			transaction.BankTxCode,
			transaction.Status,
			transaction.Payee,
			transaction.Payer,
			transaction.IBAN,
			transaction.NumberOfShares,
			transaction.StampDuty,
			transaction.Category,
		}
		err = writer.Write(record)
		if err != nil {
			return err
		}
	}

	return nil
}

// formatDate formats a date string to a standard format
func formatDate(date string) string {
	// Clean the date string first
	date = strings.TrimSpace(date)

	// Try different date formats
	formats := []string{
		"02.01.2006", // DD.MM.YYYY
		"02/01/2006", // DD/MM/YYYY
		"02-01-2006", // DD-MM-YYYY
	}

	for _, format := range formats {
		t, err := time.Parse(format, date)
		if err == nil {
			// Return date in format "Jan 02, 2006"
			return t.Format("Jan 02, 2006")
		}
	}

	// Return original if unable to parse
	return date
}

// ValidatePDF checks if a file is a valid PDF
func ValidatePDF(pdfFile string) (bool, error) {
	// Run pdftotext with -f 1 -l 1 to check if it's a valid PDF
	cmd := exec.Command("pdftotext", "-f", "1", "-l", "1", pdfFile, "/dev/null")
	err := cmd.Run()
	if err != nil {
		return false, nil // Not a valid PDF
	}

	return true, nil
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
