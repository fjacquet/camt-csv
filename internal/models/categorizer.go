// Package models provides the data structures used throughout the application.
package models

// Category represents a transaction category
type Category struct {
	Name        string
	Description string
}

// TransactionCategorizer defines the interface for categorizing transactions.
// This interface is used by parsers to categorize transactions without
// depending on the concrete categorizer implementation.
//
// Implementations should:
//   - Return a Category with the determined category name
//   - Handle both creditor and debtor transactions based on isDebtor flag
//   - Auto-learn new mappings when appropriate
type TransactionCategorizer interface {
	// Categorize determines the category for a transaction.
	//
	// Parameters:
	//   - partyName: The name of the transaction party (creditor or debtor)
	//   - isDebtor: true if the party is a debtor (sender), false if creditor (recipient)
	//   - amount: Transaction amount as string
	//   - date: Transaction date as string
	//   - info: Additional transaction information
	//
	// Returns:
	//   - Category: The determined category
	//   - error: Any error that occurred during categorization
	Categorize(partyName string, isDebtor bool, amount, date, info string) (Category, error)
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

// DebtorsConfig represents the structure of the debtors YAML file (senders of payments)
// Note: Field name kept as "debitors" for backward compatibility with existing YAML files
type DebtorsConfig struct {
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
	Party  TransactionParty
	Amount string
	Date   string
	Info   string
}
