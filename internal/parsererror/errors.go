package parsererror

import "fmt"

// ParseError represents an error during parsing
type ParseError struct {
	Parser string
	Field  string
	Value  string
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s: failed to parse %s='%s': %v", 
		e.Parser, e.Field, e.Value, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation failure
type ValidationError struct {
	FilePath string
	Reason   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.FilePath, e.Reason)
}

// CategorizationError represents a categorization failure
type CategorizationError struct {
	Transaction string
	Strategy    string
	Err         error
}

func (e *CategorizationError) Error() string {
	return fmt.Sprintf("categorization failed for %s using %s: %v",
		e.Transaction, e.Strategy, e.Err)
}

func (e *CategorizationError) Unwrap() error {
	return e.Err
}

// InvalidFormatError represents an error where the input file does not conform
// to the expected format for a specific parser.
type InvalidFormatError struct {
	FilePath             string
	ExpectedFormat       string
	ActualContentSnippet string // Optional: a snippet of the actual content for debugging
	Msg                  string
}

func (e *InvalidFormatError) Error() string {
	if e.ActualContentSnippet != "" {
		return fmt.Sprintf("invalid format in file '%s': %s. Expected: %s. Content snippet: '%s'",
			e.FilePath, e.Msg, e.ExpectedFormat, e.ActualContentSnippet)
	}
	return fmt.Sprintf("invalid format in file '%s': %s. Expected: %s",
		e.FilePath, e.Msg, e.ExpectedFormat)
}

// DataExtractionError represents an error where specific required data could not be extracted
// from a file, even if the file format itself might be valid.
type DataExtractionError struct {
	FilePath       string
	FieldName      string
	RawDataSnippet string // Optional: a snippet of the raw data where extraction failed
	Reason         string
	Msg            string
}

func (e *DataExtractionError) Error() string {
	if e.RawDataSnippet != "" {
		return fmt.Sprintf("data extraction failed in file '%s' for field '%s': %s. Reason: %s. Raw data snippet: '%s'",
			e.FilePath, e.FieldName, e.Msg, e.Reason, e.RawDataSnippet)
	}
	return fmt.Sprintf("data extraction failed in file '%s' for field '%s': %s. Reason: %s",
		e.FilePath, e.FieldName, e.Msg, e.Reason)
}
