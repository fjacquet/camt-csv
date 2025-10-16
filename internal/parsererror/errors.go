package parsererror

import "fmt"

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
