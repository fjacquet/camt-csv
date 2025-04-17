// Package models provides the data structures used throughout the application.
package models

// Category represents a transaction category
type Category struct {
	Name        string
	Description string
}

// CategoryConfig represents a category configuration in the YAML file
type CategoryConfig struct {
	Name     string   `yaml:"name"`
	Keywords []string `yaml:"keywords"`
}

// CategoriesConfig represents the structure of the categories YAML file
type CategoriesConfig struct {
	Categories []CategoryConfig `yaml:"categories"`
}

// CreditorsConfig represents the structure of the creditors YAML file (recipients of payments)
type CreditorsConfig struct {
	Creditors map[string]string `yaml:"creditors"`
}

// DebitorsConfig represents the structure of the debitors YAML file (senders of payments)
type DebitorsConfig struct {
	Debitors map[string]string `yaml:"debitors"`
}

// PayeesConfig represents the old structure of the payees YAML file (for backward compatibility)
// This is being replaced by CreditorsConfig and DebitorsConfig but is kept for migration purposes
type PayeesConfig struct {
	Payees map[string]string `yaml:"payees"`
}

// TransactionParty represents a party in a financial transaction
// Can be either a creditor (recipient) or debitor (sender)
type TransactionParty struct {
	Name     string
	IsDebtor bool // true if this party is the debtor (sender of funds)
}

// CategorizerTransaction represents a financial transaction to be categorized
// This is separate from the main Transaction struct to avoid circular dependencies
type CategorizerTransaction struct {
	Party    TransactionParty
	Amount   string
	Date     string
	Info     string
}
