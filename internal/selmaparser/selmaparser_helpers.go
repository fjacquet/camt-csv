// Package selmaparser provides functionality for processing Selma investment CSV files.
package selmaparser

import (
	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
)

// SelmaCSVRow represents a single row in a Selma CSV file
// It uses struct tags for gocsv unmarshaling
type SelmaCSVRow struct {
	Date           string `csv:"Date"`
	Description    string `csv:"Description"`
	BookkeepingNo  string `csv:"Bookkeeping No."`
	Fund           string `csv:"Fund"`
	Amount         string `csv:"Amount"`
	Currency       string `csv:"Currency"`
	NumberOfShares string `csv:"Number of Shares"`
}

// StampDutyInfo holds information about a stamp duty transaction
type StampDutyInfo struct {
	Date              string
	Fund              string
	Amount            decimal.Decimal
	BookkeepingNumber string
}

// processTransactionsInternalWithCategorizer processes a slice of Transaction objects from Selma CSV data
// with optional categorization support.
func processTransactionsInternalWithCategorizer(transactions []models.Transaction, categorizer models.TransactionCategorizer, logger logging.Logger) []models.Transaction {
	// Map to track stamp duties by date and fund
	stampDuties := make(map[string]map[string]StampDutyInfo)
	var processedTransactions []models.Transaction

	// First pass: identify stamp duties and build the lookup map
	for _, tx := range transactions {
		if tx.Description == "stamp_duty" {
			// Format date as key to match processed transactions
			var date string
			if !tx.Date.IsZero() {
				date = tx.Date.Format(dateutils.DateLayoutEuropean)
			}
			fund := tx.Fund

			// Get the amount as a decimal
			amount, _ := decimal.NewFromString(tx.Amount.String())

			// Initialize the date map if it doesn't exist
			if _, exists := stampDuties[date]; !exists {
				stampDuties[date] = make(map[string]StampDutyInfo)
			}

			// Store the stamp duty info
			stampDuties[date][fund] = StampDutyInfo{
				Date:              date,
				Fund:              fund,
				Amount:            amount,
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

		// Ensure ValueDate is set
		if tx.ValueDate.IsZero() && !tx.Date.IsZero() {
			// If no value date, use the transaction date
			tx.ValueDate = tx.Date
		}

		// Add stamp duty amount if applicable
		if tx.Description == "trade" && tx.Fund != "" {
			dateKey := tx.Date.Format(dateutils.DateLayoutEuropean)
			if dayDuties, exists := stampDuties[dateKey]; exists {
				if dutyInfo, found := dayDuties[tx.Fund]; found {
					// Associate stamp duty as a fee (decimal)
					tx.Fees = dutyInfo.Amount
				}
			}
		}

		// Add investment type and category
		tx = setInvestmentType(tx)

		// Apply external categorization if categorizer is provided
		// Note: Categorization is now handled by the statistics helper after internal processing

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
