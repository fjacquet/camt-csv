// Package selmaparser provides functionality for processing Selma investment CSV files.
package selmaparser

import (
	"strings"
	
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
)

// SelmaCSVRow represents a single row in a Selma CSV file
// It uses struct tags for gocsv unmarshaling
type SelmaCSVRow struct {
	Date            string `csv:"Date"`
	Description     string `csv:"Description"`
	BookkeepingNo   string `csv:"Bookkeeping No."`
	Fund            string `csv:"Fund"`
	Amount          string `csv:"Amount"`
	Currency        string `csv:"Currency"`
	NumberOfShares  string `csv:"Number of Shares"`
}

// StampDutyInfo holds information about a stamp duty transaction
type StampDutyInfo struct {
	Date          string
	Fund          string
	Amount        decimal.Decimal
	BookkeepingNumber string
}

// FormatDate converts various date formats to the standard yyyy-mm-dd format
func FormatDate(date string) string {
	// Try common Swiss date format (dd.mm.yyyy)
	if strings.Contains(date, ".") {
		parts := strings.Split(date, ".")
		if len(parts) == 3 {
			// Reorder to yyyy-mm-dd
			return parts[2] + "-" + parts[1] + "-" + parts[0]
		}
	}

	// Already in yyyy-mm-dd format
	if strings.Contains(date, "-") && len(date) == 10 {
		return date
	}

	// If we can't parse it, return as is
	return date
}

// Helper to clean amount strings
func CleanAmount(amount string) string {
	// Remove CHF prefix if present
	if strings.HasPrefix(amount, "CHF") {
		amount = strings.TrimSpace(strings.TrimPrefix(amount, "CHF"))
	}

	// Remove spaces
	amount = strings.ReplaceAll(amount, " ", "")

	// Replace comma with dot
	amount = strings.ReplaceAll(amount, ",", ".")

	return amount
}

// FormatTransaction ensures the transaction is in the standard format
func FormatTransaction(tx *models.Transaction) {
	// Format the date if needed
	if tx.Date != "" {
		tx.Date = FormatDate(tx.Date)
	}

	// Ensure ValueDate is set
	if tx.ValueDate == "" {
		tx.ValueDate = tx.Date
	}

	// Clean and parse amount if it's zero
	if tx.Amount.IsZero() {
		// This can happen if we're migrating from a string-based Amount
		amountStr := tx.Amount.String()
		if amountStr != "0" {
			cleanAmount := CleanAmount(amountStr)
			tx.Amount = models.ParseAmount(cleanAmount)
		}
	}

	// Set direction based on signs or contents
	if tx.CreditDebit == "" {
		// Determine if it's a credit or debit based on amount sign or description
		if tx.Amount.IsNegative() ||
		   strings.Contains(strings.ToLower(tx.Description), "payment") ||
		   strings.Contains(strings.ToLower(tx.Description), "purchase") {
			tx.CreditDebit = "DBIT"
		} else {
			tx.CreditDebit = "CRDT"
		}
	}

	// If currency is not set, default to CHF
	if tx.Currency == "" {
		tx.Currency = "CHF"
	}
}

// processTransactionsInternal processes a slice of Transaction objects from Selma CSV data.
func processTransactionsInternal(transactions []models.Transaction) []models.Transaction {
	// Map to track stamp duties by date and fund
	stampDuties := make(map[string]map[string]StampDutyInfo)
	var processedTransactions []models.Transaction

	// First pass: identify stamp duties and build the lookup map
	for _, tx := range transactions {
		if tx.Description == "stamp_duty" {
			date := tx.Date
			fund := tx.Fund

			// Get the amount as a decimal
			amount, _ := decimal.NewFromString(tx.Amount.String())

			// Initialize the date map if it doesn't exist
			if _, exists := stampDuties[date]; !exists {
				stampDuties[date] = make(map[string]StampDutyInfo)
			}

			// Store the stamp duty info
			stampDuties[date][fund] = StampDutyInfo{
				Date:          date,
				Fund:          fund,
				Amount:        amount,
				BookkeepingNumber: tx.BookkeepingNumber,
			}
		}
	}

	// Second pass: process the transactions, enriching with stamp duty info
	for _, tx := range transactions {
		// Skip stamp duty entries (we're handling them by attaching to their trades)
		if tx.Description == "stamp_duty" {
			continue
		}

		// Standardize date format to DD.MM.YYYY
		tx.Date = models.FormatDate(tx.Date)
		if tx.ValueDate != "" {
			tx.ValueDate = models.FormatDate(tx.ValueDate)
		} else {
			// If no value date, use the transaction date
			tx.ValueDate = tx.Date
		}

		// Add stamp duty amount if applicable
		if tx.Description == "trade" && tx.Fund != "" {
			if dayDuties, exists := stampDuties[tx.Date]; exists {
				if dutyInfo, found := dayDuties[tx.Fund]; found {
					tx.StampDuty = models.StandardizeAmount(dutyInfo.Amount.Neg().String())
				}
			}
		}

		// Add investment type and category
		tx = setInvestmentType(tx)

		processedTransactions = append(processedTransactions, tx)
	}

	return processedTransactions
}

// setInvestmentType sets the investment type based on the transaction description
func setInvestmentType(tx models.Transaction) models.Transaction {
	switch tx.Description {
	case "cash_transfer":
		tx.Investment = "Income"
	case "trade":
		// If amount is negative (starts with -), it's a buy
		if tx.Amount.IsNegative() {
			tx.Investment = "Buy"
		} else {
			tx.Investment = "Sell"
		}
	case "selma_fee":
		tx.Investment = "Expense"
	case "dividend":
		tx.Investment = "Dividend"
	case "withholding_tax":
		tx.Investment = ""
	}

	// Set appropriate category based on investment type
	tx = categorizeTransaction(tx)

	return tx
}

// categorizeTransaction sets the appropriate category based on transaction details and investment type
func categorizeTransaction(tx models.Transaction) models.Transaction {
	switch tx.Investment {
	case "Buy":
		tx.Category = "Investissements"
	case "Sell":
		tx.Category = "Investissements"
	case "Income":
		tx.Category = "Revenus Financiers"
	case "Dividend":
		tx.Category = "Revenus Financiers"
	case "Expense":
		tx.Category = "Frais Bancaires"
	default:
		// Categorize based on description for other cases
		switch tx.Description {
		case "selma_fee":
			tx.Category = "Frais Bancaires"
		case "cash_transfer":
			tx.Category = "Revenus Financiers"
		case "trade":
			// This is a fallback, should be caught by Investment type above
			if tx.Amount.IsNegative() {
				tx.Category = "Investissements"
			} else {
				tx.Category = "Revenus Financiers"
			}
		case "dividend":
			tx.Category = "Revenus Financiers"
		}
	}

	return tx
}
