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

// TransactionDirection represents debit or credit direction
type TransactionDirection int

const (
	DirectionUnknown TransactionDirection = iota
	DirectionDebit
	DirectionCredit
)

// String returns the string representation of TransactionDirection
func (td TransactionDirection) String() string {
	switch td {
	case DirectionDebit:
		return "DEBIT"
	case DirectionCredit:
		return "CREDIT"
	default:
		return "UNKNOWN"
	}
}

// IsDebit returns true if the direction is debit
func (td TransactionDirection) IsDebit() bool {
	return td == DirectionDebit
}

// IsCredit returns true if the direction is credit
func (td TransactionDirection) IsCredit() bool {
	return td == DirectionCredit
}

// FromString creates a TransactionDirection from a string
func TransactionDirectionFromString(s string) TransactionDirection {
	switch s {
	case "DEBIT", TransactionTypeDebit:
		return DirectionDebit
	case "CREDIT", TransactionTypeCredit:
		return DirectionCredit
	default:
		return DirectionUnknown
	}
}