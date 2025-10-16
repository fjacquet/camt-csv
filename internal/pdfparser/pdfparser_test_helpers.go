package pdfparser

import (
	"fjacquet/camt-csv/internal/models"
)

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
