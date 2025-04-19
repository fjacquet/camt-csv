// Package models provides the data structures used throughout the application.
package models

// Transaction represents a financial transaction extracted from CAMT.053
type Transaction struct {
	Date            string `csv:"Date"`
	ValueDate       string `csv:"ValueDate"`
	Description     string `csv:"Description"`
	BookkeepingNo   string `csv:"BookkeepingNo"`
	Fund            string `csv:"Fund"`
	Amount          string `csv:"Amount"`
	Currency        string `csv:"Currency"`
	CreditDebit     string `csv:"CreditDebit"`
	EntryReference  string `csv:"EntryReference"`
	AccountServicer string `csv:"AccountServicer"`
	BankTxCode      string `csv:"BankTxCode"`
	Status          string `csv:"Status"`
	Payee           string `csv:"Payee"`
	Payer           string `csv:"Payer"`
	IBAN            string `csv:"IBAN"`
	NumberOfShares  string `csv:"NumberOfShares"`
	StampDuty       string `csv:"StampDuty"`
	Category        string `csv:"Category"`
}
