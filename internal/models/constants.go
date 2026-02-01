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

// Category constants
const (
	CategoryUncategorized = "Uncategorized"
	CategorySalary        = "Salary"
	CategoryFood          = "Food"
	CategoryGroceries     = "Groceries"
	CategoryRestaurants   = "Restaurants"
	CategoryTransport     = "Transport"
	CategoryShopping      = "Shopping"
	CategoryWithdrawals   = "Withdrawals"
	CategoryTransfers     = "Transfers"
)

// File permissions
// SECURITY: These constants enforce appropriate permissions based on content type
const (
	PermissionConfigFile    = 0600 // Secret files (credentials, API keys) - owner read/write only
	PermissionNonSecretFile = 0644 // Non-secret files (YAML categories, CSV, debug) - owner read/write, group/others read
	PermissionDirectory     = 0750 // Directories - owner rwx, group rx, others none
	PermissionExecutable    = 0755 // Executable files - owner rwx, group/others rx
)

// CSV formatting
const (
	DefaultCSVDelimiter = ',' // Aligned with config default
	DateFormatCSV       = "02.01.2006"
	DecimalPlaces       = 2
)

// Performance tuning constants
const (
	DefaultMapCapacity      = 100 // Default capacity for maps
	DefaultSliceCapacity    = 50  // Default capacity for slices
	MaxConcurrentOperations = 10  // Maximum concurrent operations
	DefaultTimeoutSeconds   = 30  // Default timeout for operations
)

// Bank transaction codes
const (
	BankCodeCashWithdrawal = "CASH_WITHDRAWAL"
	BankCodePOS            = "POS"
	BankCodeCreditCard     = "CREDIT_CARD"
	BankCodeInternalCredit = "INTERNAL_CREDIT"
	BankCodeDirectDebit    = "DIRECT_DEBIT"
)

// Environment variable names
const (
	EnvLogLevel     = "LOG_LEVEL"
	EnvLogFormat    = "LOG_FORMAT"
	EnvGeminiAPIKey = "GEMINI_API_KEY" // #nosec G101 -- env var name, not a credential
	EnvCSVDelimiter = "CSV_DELIMITER"
)
