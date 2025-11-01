package models

// Transaction types
const (
	TransactionTypeDebit  = "DBIT"
	TransactionTypeCredit = "CRDT"
)

// Transaction statuses
const (
	StatusCompleted = "COMPLETED"
	StatusPending   = "PENDING"
	StatusFailed    = "FAILED"
)

// Categories
const (
	CategoryUncategorized = "Uncategorized"
	CategorySalary        = "Salaire"
	CategoryGroceries     = "Alimentation"
	CategoryTransport     = "Transports Publics"
	CategoryShopping      = "Shopping"
	CategoryRestaurants   = "Restaurants"
	CategoryWithdrawals   = "Retraits"
	CategoryTransfers     = "Virements"
)

// Bank transaction codes
const (
	BankCodeCashWithdrawal = "CWDL"
	BankCodePOS            = "POSD"
	BankCodeCreditCard     = "CCRD"
	BankCodeInternalCredit = "ICDT"
	BankCodeDirectDebit    = "DMCT"
)

// File permissions
const (
	PermissionConfigFile = 0600
	PermissionDirectory  = 0750
	PermissionReportFile = 0644
)