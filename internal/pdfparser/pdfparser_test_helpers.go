package pdfparser

import (
	"time"
	
	"fjacquet/camt-csv/internal/models"
)

// createMockTransactions creates mock transactions for testing
func createMockTransactions() []models.Transaction {
	return []models.Transaction{
		{
			Date:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Description:    "Coffee Shop Purchase Card Payment REF123456",
			Amount:         models.ParseAmount("100.00"),
			Currency:       "EUR",
			CreditDebit:    models.TransactionTypeDebit,
			EntryReference: "REF123456",
		},
		{
			Date:           time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			Description:    "Salary Payment Incoming Transfer SAL987654",
			Amount:         models.ParseAmount("1000.00"),
			Currency:       "EUR",
			CreditDebit:    models.TransactionTypeCredit,
			EntryReference: "SAL987654",
		},
	}
}
