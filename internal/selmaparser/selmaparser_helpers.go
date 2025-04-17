// Package selmaparser provides functionality for processing Selma investment CSV files.
package selmaparser

import (
	"encoding/csv"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/models"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// readSelmaCSVFile reads and parses a Selma CSV file into a slice of Transaction objects.
func readSelmaCSVFile(filePath string) ([]models.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.WithFields(logrus.Fields{"filePath": filePath}).Errorf("Failed to open Selma CSV file: %v", err)
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		log.WithFields(logrus.Fields{"filePath": filePath}).Errorf("Failed to read CSV headers: %v", err)
		return nil, err
	}

	// Create a map of column indices by header name
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(header)] = i
	}

	// Check for minimum required headers
	requiredHeaders := []string{"date", "description", "amount"}
	for _, required := range requiredHeaders {
		if _, ok := headerMap[required]; !ok {
			log.WithFields(logrus.Fields{
				"filePath": filePath,
				"headers":  headers,
				"missing":  required,
			}).Error("Missing required header in CSV")
			return nil, fmt.Errorf("invalid CSV format: missing %s column", required)
		}
	}

	var transactions []models.Transaction
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			log.WithError(err).Error("Error reading CSV row")
			return nil, err
		}

		// Skip empty rows
		if len(row) == 0 {
			continue
		}

		transaction, err := parseSelmaCSVRow(row, headerMap)
		if err != nil {
			log.WithError(err).Error("Error parsing CSV row")
			return nil, err
		}

		transactions = append(transactions, transaction)
	}

	log.WithField("count", len(transactions)).Info("Successfully read Selma transactions")
	return transactions, nil
}

// parseSelmaCSVRow parses a single Selma CSV row into a Transaction object.
func parseSelmaCSVRow(row []string, headerMap map[string]int) (models.Transaction, error) {
	transaction := models.Transaction{}

	// Helper function to safely get value from row by header name
	getValue := func(header string) string {
		if idx, ok := headerMap[header]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	// Map CSV columns to Transaction fields
	transaction.Date = getValue("date")
	transaction.Description = getValue("description")
	transaction.Amount = getValue("amount")
	
	// Optional fields - map if they exist
	transaction.ValueDate = getValue("valuedate")
	if transaction.ValueDate == "" {
		transaction.ValueDate = getValue("value date") // Alternative header format
	}
	
	transaction.Currency = getValue("currency")
	transaction.BookkeepingNo = getValue("bookkeepingno")
	transaction.Fund = getValue("fund")
	transaction.EntryReference = getValue("entryreference")
	transaction.IBAN = getValue("iban")
	transaction.NumberOfShares = getValue("numberofshares")
	transaction.StampDuty = getValue("stampduty")
	
	// Set credit/debit based on amount
	transaction.CreditDebit = getCreditDebitFromAmount(transaction.Amount)

	return transaction, nil
}

// getCreditDebitFromAmount determines if a transaction is a credit or debit based on the amount.
func getCreditDebitFromAmount(amountStr string) string {
	// In financial systems, negative values typically represent debits (outgoing money)
	if strings.HasPrefix(amountStr, "-") {
		return "DBIT"
	}
	// Positive values typically represent credits (incoming money)
	return "CRDT"
}

// processTransactionsInternal processes a slice of Transaction objects from Selma CSV data.
func processTransactionsInternal(transactions []models.Transaction) []models.Transaction {
	var newTransactions []models.Transaction

	for _, transaction := range transactions {
		// Process the transaction
		processedTransaction := transaction

		// Apply categorization
		processedTransaction = categorizeTransaction(processedTransaction)

		// Handle stamp duty association
		processedTransaction = handleStampDuty(processedTransaction)

		newTransactions = append(newTransactions, processedTransaction)
	}

	return newTransactions
}

// categorizeTransaction applies basic categorization to a transaction based on its description.
func categorizeTransaction(transaction models.Transaction) models.Transaction {
	if transaction.Category != "" {
		return transaction
	}

	// Create a categorizer.Transaction to use the categorization system
	isDebtor := transaction.CreditDebit == "DBIT"
	
	catTx := categorizer.Transaction{
		PartyName: transaction.Description,
		IsDebtor:  isDebtor,
		Amount:    transaction.Amount + " " + transaction.Currency,
		Date:      transaction.Date,
		Info:      transaction.Description,
	}

	// Try to categorize
	category, err := categorizer.CategorizeTransaction(catTx)
	if err == nil && category.Name != "" {
		transaction.Category = category.Name
		log.WithFields(logrus.Fields{
			"description": transaction.Description,
			"category":    category.Name,
		}).Debug("Categorized Selma transaction")
	} else {
		// Default categorization based on description keywords
		if strings.Contains(strings.ToLower(transaction.Description), "dividend") {
			transaction.Category = "Dividend"
		} else if strings.Contains(strings.ToLower(transaction.Description), "interest") {
			transaction.Category = "Interest"
		} else if strings.Contains(strings.ToLower(transaction.Description), "purchase") || 
			   strings.Contains(strings.ToLower(transaction.Description), "buy") {
			transaction.Category = "Investment Purchase"
		} else if strings.Contains(strings.ToLower(transaction.Description), "sale") || 
			   strings.Contains(strings.ToLower(transaction.Description), "sell") {
			transaction.Category = "Investment Sale"
		} else if strings.Contains(strings.ToLower(transaction.Description), "stamp duty") {
			transaction.Category = "Tax"
		} else if strings.Contains(strings.ToLower(transaction.Description), "fee") {
			transaction.Category = "Fee"
		} else {
			transaction.Category = "Uncategorized"
		}
	}

	return transaction
}

// handleStampDuty categorizes stamp duty transactions
func handleStampDuty(transaction models.Transaction) models.Transaction {
	// If this is a stamp duty transaction, categorize it as Tax
	if strings.Contains(strings.ToLower(transaction.Description), "stamp duty") {
		transaction.Category = "Tax"
	}

	return transaction
}

// writeSelmaCSVFile writes a slice of Transaction structs to a CSV file in Selma format.
func writeSelmaCSVFile(filePath string, transactions []models.Transaction) error {
	file, err := os.Create(filePath)
	if err != nil {
		log.WithFields(logrus.Fields{"filePath": filePath}).Errorf("Failed to create CSV file: %v", err)
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"Date", "Description", "Bookkeeping No.", "Fund", "Amount", "Currency", "Number of Shares", "Stamp Duty", "Category"}
	if err := writer.Write(headers); err != nil {
		log.WithFields(logrus.Fields{"filePath": filePath}).Errorf("Failed to write CSV headers: %v", err)
		return err
	}

	for _, transaction := range transactions {
		row := []string{
			transaction.Date,
			transaction.Description,
			transaction.BookkeepingNo,
			transaction.Fund,
			transaction.Amount,
			transaction.Currency,
			"", // Number of Shares (not stored in our Transaction model)
			"", // Stamp Duty (not stored in our Transaction model)
			transaction.Category,
		}

		if err := writer.Write(row); err != nil {
			log.WithFields(logrus.Fields{"transaction": transaction}).Errorf("Failed to write transaction row: %v", err)
			return err
		}
	}

	return nil
}

// validateSelmaCSVFormat checks if a file is in valid Selma CSV format.
func validateSelmaCSVFormat(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return false, err
	}

	// Check for required headers in Selma CSV format
	requiredHeaders := []string{"Date", "Description", "Amount", "Currency"}
	headerMap := make(map[string]bool)
	
	for _, header := range headers {
		headerMap[header] = true
	}
	
	for _, required := range requiredHeaders {
		if !headerMap[required] {
			return false, nil
		}
	}
	
	// Read at least one row to verify format
	row, err := reader.Read()
	if err != nil {
		// If we can't read a row, the file might be empty or malformed
		return false, nil
	}
	
	// Check that we have enough columns
	if len(row) < len(requiredHeaders) {
		return false, nil
	}
	
	// Try to parse amount to verify it's a number
	_, err = strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
	if err != nil {
		return false, nil
	}
	
	return true, nil
}
