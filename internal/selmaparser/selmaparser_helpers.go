// Package selmaparser provides functionality for processing Selma investment CSV files.
package selmaparser

import (
	"strings"

	"fjacquet/camt-csv/internal/models"
)

// SelmaCSVRow represents a single row in a Selma CSV file
// It uses struct tags for gocsv unmarshaling
type SelmaCSVRow struct {
	Date            string `csv:"Date"`
	Name            string `csv:"Name"`
	ISIN            string `csv:"ISIN"`
	TransactionType string `csv:"Transaction Type"`
	NumberOfShares  string `csv:"Number of Shares"`
	Price           string `csv:"Price"`
	Currency        string `csv:"Currency"`
	TransactionFee  string `csv:"Transaction Fee"`
	TotalAmount     string `csv:"Total Amount"`
	PortfolioId     string `csv:"Portfolio Id"`
}

// processTransactionsInternal processes a slice of Transaction objects from Selma CSV data.
func processTransactionsInternal(transactions []models.Transaction) []models.Transaction {
	// Enrich transactions with additional information
	for i := range transactions {
		// Standardize date format to DD.MM.YYYY
		transactions[i].Date = models.FormatDate(transactions[i].Date)
		if transactions[i].ValueDate != "" {
			transactions[i].ValueDate = models.FormatDate(transactions[i].ValueDate)
		} else {
			// If no value date, use the transaction date
			transactions[i].ValueDate = transactions[i].Date
		}
		
		// Additionally, apply category logic like stamp duty detection
		transactions[i] = handleStampDuty(transactions[i])
	}
	
	return transactions
}

// handleStampDuty categorizes stamp duty transactions
func handleStampDuty(transaction models.Transaction) models.Transaction {
	// If this is a stamp duty transaction, categorize it as Tax
	if strings.Contains(strings.ToLower(transaction.Description), "stamp duty") {
		transaction.Category = "Tax"
	}
	return transaction
}
