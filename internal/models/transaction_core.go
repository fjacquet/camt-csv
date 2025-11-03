package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionCore represents the essential data of a financial transaction.
// This is the minimal, immutable core that all transactions must have.
type TransactionCore struct {
	ID          string          `json:"id" yaml:"id"`
	Date        time.Time       `json:"date" yaml:"date"`
	ValueDate   time.Time       `json:"value_date" yaml:"value_date"`
	Amount      Money           `json:"amount" yaml:"amount"`
	Description string          `json:"description" yaml:"description"`
	Status      string          `json:"status" yaml:"status"`
	Reference   string          `json:"reference" yaml:"reference"`
}

// TransactionDirection represents the direction of a transaction
type TransactionDirection string

const (
	DirectionDebit   TransactionDirection = "debit"
	DirectionCredit  TransactionDirection = "credit"
	DirectionUnknown TransactionDirection = "unknown"
)

// TransactionWithParties extends TransactionCore with party information
type TransactionWithParties struct {
	TransactionCore
	Payer     Party                `json:"payer" yaml:"payer"`
	Payee     Party                `json:"payee" yaml:"payee"`
	Direction TransactionDirection `json:"direction" yaml:"direction"`
}

// CategorizedTransaction extends TransactionWithParties with categorization
type CategorizedTransaction struct {
	TransactionWithParties
	Category string `json:"category" yaml:"category"`
	Type     string `json:"type" yaml:"type"`
	Fund     string `json:"fund" yaml:"fund"`
}

// GetCounterparty returns the relevant party based on transaction direction
func (t TransactionWithParties) GetCounterparty() Party {
	if t.Direction == DirectionDebit {
		return t.Payee
	}
	return t.Payer
}

// IsDebit returns true if the transaction is a debit
func (t TransactionWithParties) IsDebit() bool {
	return t.Direction == DirectionDebit
}

// IsCredit returns true if the transaction is a credit
func (t TransactionWithParties) IsCredit() bool {
	return t.Direction == DirectionCredit
}