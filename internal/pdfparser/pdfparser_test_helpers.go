package pdfparser

import (
	"os"
	"strings"
	"fmt"
	
	"fjacquet/camt-csv/internal/models"
)

// Define a variable to store the original function so we can restore it after tests
var originalExtractTextFromPDF func(pdfFile string) (string, error)

// mockPDFExtraction is a testing helper that replaces the actual PDF extraction
// during tests to avoid dependency on external commands like pdftotext
func mockPDFExtraction() func() {
	// Save the original function
	originalExtractTextFromPDF = extractTextFromPDF
	
	// Create a mock extraction function
	mockExtractionFunction := func(pdfFile string) (string, error) {
		// If the file doesn't exist, return normal error
		_, err := os.Stat(pdfFile)
		if err != nil {
			return "", err
		}
		
		// For existing files, check if it's a valid PDF by looking for PDF header
		fileContent, err := os.ReadFile(pdfFile)
		if err != nil {
			return "", err
		}
		
		content := string(fileContent)
		
		// Mock a valid PDF if it has the PDF header
		if strings.HasPrefix(content, "%PDF") {
			// Return a text format that matches how the parser expects to see transaction data in PDFs
			return `BANK STATEMENT
Account: CH1234567890
Date: 01.01.2023

01.01.2023 Coffee Shop Purchase
Card Payment 
REF123456
Amount: 100.00 EUR
Balance: 900.00 EUR

02.01.2023 Salary Payment 
Incoming Transfer
SAL987654
Amount: 1000.00 EUR
Balance: 1900.00 EUR`, nil
		}
		
		// For files without PDF header, return error to indicate invalid PDF
		return "", fmt.Errorf("invalid PDF format")
	}
	
	// Replace the original function with our mock
	extractTextFromPDF = mockExtractionFunction
	
	// Return a cleanup function to restore the original function
	return func() {
		extractTextFromPDF = originalExtractTextFromPDF
	}
}

// createMockTransactions creates mock transactions for testing
func createMockTransactions() []models.Transaction {
	return []models.Transaction{
		{
			Date:           "2023-01-01",
			Description:    "Coffee Shop Purchase Card Payment REF123456",
			Amount:         models.ParseAmount("100.00"),
			Currency:       "EUR",
			CreditDebit:    "DBIT",
			EntryReference: "REF123456",
		},
		{
			Date:           "2023-01-02",
			Description:    "Salary Payment Incoming Transfer SAL987654",
			Amount:         models.ParseAmount("1000.00"),
			Currency:       "EUR",
			CreditDebit:    "CRDT",
			EntryReference: "SAL987654",
		},
	}
}
