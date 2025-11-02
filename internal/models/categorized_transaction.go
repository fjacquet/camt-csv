package models

// CategorizedTransaction adds categorization data to TransactionWithParties
type CategorizedTransaction struct {
	TransactionWithParties
	Category string `json:"category" yaml:"category"`
	Type     string `json:"type" yaml:"type"`
	Fund     string `json:"fund" yaml:"fund"`
}

// NewCategorizedTransaction creates a new CategorizedTransaction instance
func NewCategorizedTransaction() CategorizedTransaction {
	return CategorizedTransaction{
		TransactionWithParties: NewTransactionWithParties(),
		Category:               CategoryUncategorized, // Default category
	}
}

// NewCategorizedTransactionFromParties creates a new CategorizedTransaction from TransactionWithParties
func NewCategorizedTransactionFromParties(twp TransactionWithParties) CategorizedTransaction {
	return CategorizedTransaction{
		TransactionWithParties: twp,
		Category:               CategoryUncategorized, // Default category
	}
}

// IsCategorized returns true if the transaction has been categorized (not "Uncategorized")
func (ct CategorizedTransaction) IsCategorized() bool {
	return ct.Category != "" && ct.Category != CategoryUncategorized
}

// HasType returns true if the transaction has a type assigned
func (ct CategorizedTransaction) HasType() bool {
	return ct.Type != ""
}

// HasFund returns true if the transaction has a fund assigned
func (ct CategorizedTransaction) HasFund() bool {
	return ct.Fund != ""
}

// WithCategory sets the category and returns a new CategorizedTransaction
func (ct CategorizedTransaction) WithCategory(category string) CategorizedTransaction {
	ct.Category = category
	return ct
}

// WithType sets the type and returns a new CategorizedTransaction
func (ct CategorizedTransaction) WithType(transactionType string) CategorizedTransaction {
	ct.Type = transactionType
	return ct
}

// WithFund sets the fund and returns a new CategorizedTransaction
func (ct CategorizedTransaction) WithFund(fund string) CategorizedTransaction {
	ct.Fund = fund
	return ct
}

// Categorize sets the category, type, and fund in one operation
func (ct CategorizedTransaction) Categorize(category, transactionType, fund string) CategorizedTransaction {
	ct.Category = category
	ct.Type = transactionType
	ct.Fund = fund
	return ct
}

// ResetCategorization resets the categorization to uncategorized
func (ct CategorizedTransaction) ResetCategorization() CategorizedTransaction {
	ct.Category = CategoryUncategorized
	ct.Type = ""
	ct.Fund = ""
	return ct
}

// Equal returns true if two CategorizedTransaction instances are equal
func (ct CategorizedTransaction) Equal(other CategorizedTransaction) bool {
	return ct.TransactionWithParties.Equal(other.TransactionWithParties) &&
		ct.Category == other.Category &&
		ct.Type == other.Type &&
		ct.Fund == other.Fund
}

// GetCategoryInfo returns a summary of the categorization information
func (ct CategorizedTransaction) GetCategoryInfo() map[string]string {
	info := make(map[string]string)
	if ct.Category != "" {
		info["category"] = ct.Category
	}
	if ct.Type != "" {
		info["type"] = ct.Type
	}
	if ct.Fund != "" {
		info["fund"] = ct.Fund
	}
	return info
}
