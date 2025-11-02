// Package parsererror provides custom error types for the camt-csv application.
// These error types offer structured error information with context, making it easier
// to handle different types of failures in parsing, validation, and categorization operations.
//
// The package follows Go error handling best practices by implementing the error interface
// and providing Unwrap methods for error inspection using errors.Is and errors.As.
package parsererror

import "fmt"

// ParseError represents an error that occurred during the parsing of financial data.
// It provides structured information about which parser failed, what field was being
// processed, the problematic value, and the underlying error.
//
// This error type is used when a parser encounters data that cannot be processed
// according to the expected format or business rules.
type ParseError struct {
	Parser string // Name of the parser that encountered the error (e.g., "CAMT", "PDF", "Revolut")
	Field  string // Name of the field being parsed when the error occurred
	Value  string // The actual value that caused the parsing to fail
	Err    error  // The underlying error that caused the parsing failure
}

// Error returns a formatted error message that includes the parser name, field, value, and underlying error.
// This implements the error interface.
func (e *ParseError) Error() string {
	return fmt.Sprintf("%s: failed to parse %s='%s': %v",
		e.Parser, e.Field, e.Value, e.Err)
}

// Unwrap returns the underlying error, enabling error inspection with errors.Is and errors.As.
// This follows Go 1.13+ error wrapping conventions.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// ValidationError represents a failure during file format validation.
// This error occurs when a file does not meet the basic requirements for processing
// by a specific parser, such as missing required headers, incorrect file structure,
// or unsupported file format.
type ValidationError struct {
	FilePath string // Path to the file that failed validation
	Reason   string // Human-readable explanation of why validation failed
}

// Error returns a formatted error message indicating which file failed validation and why.
// This implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.FilePath, e.Reason)
}

// CategorizationError represents a failure during transaction categorization.
// This error occurs when a categorization strategy encounters an unexpected condition
// that prevents it from completing the categorization process.
type CategorizationError struct {
	Transaction string // Identifier or description of the transaction being categorized
	Strategy    string // Name of the categorization strategy that failed
	Err         error  // The underlying error that caused the categorization failure
}

// Error returns a formatted error message indicating which transaction and strategy failed.
// This implements the error interface.
func (e *CategorizationError) Error() string {
	return fmt.Sprintf("categorization failed for %s using %s: %v",
		e.Transaction, e.Strategy, e.Err)
}

// Unwrap returns the underlying error, enabling error inspection with errors.Is and errors.As.
// This follows Go 1.13+ error wrapping conventions.
func (e *CategorizationError) Unwrap() error {
	return e.Err
}

// InvalidFormatError represents an error where the input file does not conform
// to the expected format for a specific parser. This is more specific than ValidationError
// and includes details about what format was expected versus what was found.
type InvalidFormatError struct {
	FilePath             string // Path to the file with invalid format
	ExpectedFormat       string // Description of the expected file format
	ActualContentSnippet string // Optional: a snippet of the actual content for debugging
	Msg                  string // Additional context about the format mismatch
}

// Error returns a detailed error message about the format mismatch.
// If ActualContentSnippet is provided, it includes a sample of the problematic content.
// This implements the error interface.
func (e *InvalidFormatError) Error() string {
	if e.ActualContentSnippet != "" {
		return fmt.Sprintf("invalid format in file '%s': %s. Expected: %s. Content snippet: '%s'",
			e.FilePath, e.Msg, e.ExpectedFormat, e.ActualContentSnippet)
	}
	return fmt.Sprintf("invalid format in file '%s': %s. Expected: %s",
		e.FilePath, e.Msg, e.ExpectedFormat)
}

// DataExtractionError represents an error where specific required data could not be extracted
// from a file, even if the file format itself might be valid. This occurs when the parser
// can read the file structure but cannot extract meaningful transaction data from it.
type DataExtractionError struct {
	FilePath       string // Path to the file where extraction failed
	FieldName      string // Name of the specific field that could not be extracted
	RawDataSnippet string // Optional: a snippet of the raw data where extraction failed
	Reason         string // Technical reason why extraction failed
	Msg            string // Human-readable description of the extraction failure
}

// Error returns a detailed error message about the data extraction failure.
// If RawDataSnippet is provided, it includes a sample of the problematic raw data.
// This implements the error interface.
func (e *DataExtractionError) Error() string {
	if e.RawDataSnippet != "" {
		return fmt.Sprintf("data extraction failed in file '%s' for field '%s': %s. Reason: %s. Raw data snippet: '%s'",
			e.FilePath, e.FieldName, e.Msg, e.Reason, e.RawDataSnippet)
	}
	return fmt.Sprintf("data extraction failed in file '%s' for field '%s': %s. Reason: %s",
		e.FilePath, e.FieldName, e.Msg, e.Reason)
}
