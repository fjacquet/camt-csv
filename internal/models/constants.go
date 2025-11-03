package models

import "os"

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
	CategoryTransport     = "Transport"
	CategoryShopping      = "Shopping"
)

// File permissions
const (
	PermissionConfigFile  = 0600
	PermissionDirectory   = 0750
	PermissionExecutable  = 0755
)

// CSV formatting
const (
	DefaultCSVDelimiter = ';'
	DateFormatCSV       = "02.01.2006"
	DecimalPlaces       = 2
)

// Performance tuning constants
const (
	DefaultMapCapacity      = 100  // Default capacity for maps
	DefaultSliceCapacity    = 50   // Default capacity for slices
	MaxConcurrentOperations = 10   // Maximum concurrent operations
	DefaultTimeoutSeconds   = 30   // Default timeout for operations
)

// Environment variable names
const (
	EnvLogLevel     = "LOG_LEVEL"
	EnvLogFormat    = "LOG_FORMAT"
	EnvGeminiAPIKey = "GEMINI_API_KEY"
	EnvCSVDelimiter = "CSV_DELIMITER"
)