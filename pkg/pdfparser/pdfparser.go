// Package pdfparser provides functionality to parse PDF files and extract transaction data.
package pdfparser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// Transaction represents a financial transaction extracted from a PDF
type Transaction struct {
	Date           string
	Description    string
	BookkeepingNo  string
	Fund           string
	Amount         string
	Currency       string
	CreditDebit    string
	NumberOfShares string
	StampDuty      string
	Investment     string
}

// ConvertPDFToCSV converts a PDF file to CSV format
func ConvertPDFToCSV(pdfFile string, csvFile string) error {
	// Extract text from PDF
	text, err := extractTextFromPDF(pdfFile)
	if err != nil {
		return fmt.Errorf("error extracting text from PDF: %w", err)
	}

	// Parse transactions from the extracted text
	transactions, err := parseTransactions(text)
	if err != nil {
		return fmt.Errorf("error parsing transactions: %w", err)
	}

	// Write transactions to CSV
	err = writeTransactionsToCSV(transactions, csvFile)
	if err != nil {
		return fmt.Errorf("error writing transactions to CSV: %w", err)
	}

	return nil
}

// extractTextFromPDF extracts text content from a PDF file
func extractTextFromPDF(pdfFile string) (string, error) {
	f, r, err := pdf.Open(pdfFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}

	buf.ReadFrom(b)
	return buf.String(), nil
}

// parseTransactions parses transaction data from PDF text content
func parseTransactions(text string) ([]Transaction, error) {
	var transactions []Transaction

	// Split text into lines
	lines := strings.Split(text, "\n")

	// Define patterns to match transaction data
	datePattern := regexp.MustCompile(`\d{2}[./-]\d{2}[./-]\d{4}`)
	amountPattern := regexp.MustCompile(`[-+]?\d+\.\d{2}`)
	currencyPattern := regexp.MustCompile(`(CHF|EUR|USD)`)

	// Process each line
	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Extract date
		dateMatch := datePattern.FindString(line)
		if dateMatch == "" {
			continue // Skip lines without dates (likely not transaction lines)
		}

		// Extract amount
		amountMatch := amountPattern.FindString(line)
		if amountMatch == "" {
			continue // Skip lines without amounts
		}

		// Extract currency
		currencyMatch := currencyPattern.FindString(line)
		if currencyMatch == "" {
			currencyMatch = "CHF" // Default currency if not found
		}

		// Determine credit/debit
		creditDebit := "DBIT"
		if strings.HasPrefix(amountMatch, "+") || !strings.HasPrefix(amountMatch, "-") {
			creditDebit = "CRDT"
		}
		// Remove sign from amount
		amountMatch = strings.TrimPrefix(amountMatch, "+")
		amountMatch = strings.TrimPrefix(amountMatch, "-")

		// Extract description (everything except date, amount, and currency)
		description := line
		description = datePattern.ReplaceAllString(description, "")
		description = amountPattern.ReplaceAllString(description, "")
		description = currencyPattern.ReplaceAllString(description, "")
		description = strings.TrimSpace(description)

		// Format date
		formattedDate := formatDate(dateMatch)

		// Extract bookkeeping number
		bookkeepingNo := extractBookkeepingNo(description)

		// Extract fund information
		fund := extractFund(description)

		// Clean description
		cleanedDescription := cleanDescription(description)

		// Create transaction
		transaction := Transaction{
			Date:           formattedDate,
			Description:    cleanedDescription,
			BookkeepingNo:  bookkeepingNo,
			Fund:           fund,
			Amount:         amountMatch,
			Currency:       currencyMatch,
			CreditDebit:    creditDebit,
			NumberOfShares: "",
			StampDuty:      "",
			Investment:     description,
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// writeTransactionsToCSV writes transactions to a CSV file
func writeTransactionsToCSV(transactions []Transaction, csvFile string) error {
	file, err := os.Create(csvFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Date", "Description", "BookkeepingNo", "Fund", "Amount", "Currency", "CreditDebit", "NumberOfShares", "StampDuty", "Investment"}
	err = writer.Write(header)
	if err != nil {
		return err
	}

	// Write transactions
	for _, transaction := range transactions {
		record := []string{
			transaction.Date,
			transaction.Description,
			transaction.BookkeepingNo,
			transaction.Fund,
			transaction.Amount,
			transaction.Currency,
			transaction.CreditDebit,
			transaction.NumberOfShares,
			transaction.StampDuty,
			transaction.Investment,
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
	// Try different date formats
	formats := []string{
		"02.01.2006",
		"02/01/2006",
		"02-01-2006",
	}

	for _, format := range formats {
		t, err := time.Parse(format, date)
		if err == nil {
			return t.Format("Jan 02, 2006")
		}
	}

	return date // Return original if parsing fails
}

// extractBookkeepingNo attempts to extract a bookkeeping number from text
func extractBookkeepingNo(text string) string {
	// Look for patterns that might be bookkeeping numbers
	
	// Check for numeric sequences that might be bookkeeping numbers
	numericPattern := regexp.MustCompile(`\b\d{5,10}\b`)
	matches := numericPattern.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}
	
	// Check for patterns like "No. 12345" or "Ref: 12345"
	refPattern := regexp.MustCompile(`(?i)(No\.|Ref:?|Reference:?|Booking:?)\s*(\d+)`)
	matches = refPattern.FindStringSubmatch(text)
	if len(matches) > 2 {
		return matches[2]
	}
	
	return ""
}

// extractFund attempts to extract fund information from text
func extractFund(text string) string {
	// Look for patterns that might indicate fund information
	
	// Check for patterns like "Fund: XYZ" or "Investment in XYZ"
	fundPattern := regexp.MustCompile(`(?i)(Fund:?|Investment in)\s*([A-Za-z0-9\s]+)`)
	matches := fundPattern.FindStringSubmatch(text)
	if len(matches) > 2 {
		return strings.TrimSpace(matches[2])
	}
	
	return ""
}

// cleanDescription cleans up the text to create a better description
func cleanDescription(text string) string {
	// Remove any excessively long numeric sequences
	cleaned := regexp.MustCompile(`\b\d{10,}\b`).ReplaceAllString(text, "")
	
	// Remove any reference prefixes
	cleaned = regexp.MustCompile(`(?i)(Ref:?|Reference:?|No\.|Booking:?)\s*\d+`).ReplaceAllString(cleaned, "")
	
	// Trim spaces and remove duplicate spaces
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	
	return cleaned
}

// ValidatePDF checks if a file is a valid PDF
func ValidatePDF(pdfFile string) (bool, error) {
	f, _, err := pdf.Open(pdfFile)
	if err != nil {
		return false, nil // Not a valid PDF
	}
	defer f.Close()
	
	return true, nil
}
