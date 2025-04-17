// Package models provides the data structures used throughout the application.
package models

// Transaction represents a financial transaction extracted from CAMT.053
type Transaction struct {
	Date            string
	ValueDate       string
	Description     string
	BookkeepingNo   string
	Fund            string
	Amount          string
	Currency        string
	CreditDebit     string
	EntryReference  string
	AccountServicer string
	BankTxCode      string
	Status          string
	Payee           string
	Payer           string
	IBAN            string
	NumberOfShares  string
	StampDuty       string
	Category        string
}
