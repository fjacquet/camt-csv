package parsererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ParseError
		expected string
	}{
		{
			name: "basic parse error",
			err: &ParseError{
				Parser: "CAMT",
				Field:  "amount",
				Value:  "invalid",
				Err:    errors.New("invalid decimal"),
			},
			expected: "CAMT: failed to parse amount='invalid': invalid decimal",
		},
		{
			name: "parse error with empty value",
			err: &ParseError{
				Parser: "PDF",
				Field:  "date",
				Value:  "",
				Err:    errors.New("empty date"),
			},
			expected: "PDF: failed to parse date='': empty date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestParseError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	parseErr := &ParseError{
		Parser: "CAMT",
		Field:  "amount",
		Value:  "invalid",
		Err:    originalErr,
	}

	assert.Equal(t, originalErr, parseErr.Unwrap())
	assert.True(t, errors.Is(parseErr, originalErr))
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "basic validation error",
			err: &ValidationError{
				FilePath: "/path/to/file.xml",
				Reason:   "not a valid CAMT.053 XML document",
			},
			expected: "validation failed for /path/to/file.xml: not a valid CAMT.053 XML document",
		},
		{
			name: "validation error with empty statements",
			err: &ValidationError{
				FilePath: "/path/to/empty.xml",
				Reason:   "no statements found in document",
			},
			expected: "validation failed for /path/to/empty.xml: no statements found in document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestCategorizationError(t *testing.T) {
	tests := []struct {
		name     string
		err      *CategorizationError
		expected string
	}{
		{
			name: "basic categorization error",
			err: &CategorizationError{
				Transaction: "TX123",
				Strategy:    "AIStrategy",
				Err:         errors.New("API timeout"),
			},
			expected: "categorization failed for TX123 using AIStrategy: API timeout",
		},
		{
			name: "categorization error with keyword strategy",
			err: &CategorizationError{
				Transaction: "COOP Payment",
				Strategy:    "KeywordStrategy",
				Err:         errors.New("no matching patterns"),
			},
			expected: "categorization failed for COOP Payment using KeywordStrategy: no matching patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestCategorizationError_Unwrap(t *testing.T) {
	originalErr := errors.New("network error")
	catErr := &CategorizationError{
		Transaction: "TX123",
		Strategy:    "AIStrategy",
		Err:         originalErr,
	}

	assert.Equal(t, originalErr, catErr.Unwrap())
	assert.True(t, errors.Is(catErr, originalErr))
}

func TestInvalidFormatError(t *testing.T) {
	tests := []struct {
		name     string
		err      *InvalidFormatError
		expected string
	}{
		{
			name: "invalid format error with content snippet",
			err: &InvalidFormatError{
				FilePath:             "/path/to/file.pdf",
				ExpectedFormat:       "PDF",
				ActualContentSnippet: "<?xml version=",
				Msg:                  "file appears to be XML",
			},
			expected: "invalid format in file '/path/to/file.pdf': file appears to be XML. Expected: PDF. Content snippet: '<?xml version='",
		},
		{
			name: "invalid format error without content snippet",
			err: &InvalidFormatError{
				FilePath:       "/path/to/file.csv",
				ExpectedFormat: "Revolut CSV",
				Msg:            "missing required headers",
			},
			expected: "invalid format in file '/path/to/file.csv': missing required headers. Expected: Revolut CSV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestDataExtractionError(t *testing.T) {
	tests := []struct {
		name     string
		err      *DataExtractionError
		expected string
	}{
		{
			name: "data extraction error with raw data snippet",
			err: &DataExtractionError{
				FilePath:       "/path/to/file.xml",
				FieldName:      "amount",
				RawDataSnippet: "<Amt Ccy=\"CHF\">invalid</Amt>",
				Reason:         "invalid decimal format",
				Msg:            "could not parse amount",
			},
			expected: "data extraction failed in file '/path/to/file.xml' for field 'amount': could not parse amount. Reason: invalid decimal format. Raw data snippet: '<Amt Ccy=\"CHF\">invalid</Amt>'",
		},
		{
			name: "data extraction error without raw data snippet",
			err: &DataExtractionError{
				FilePath:  "/path/to/file.csv",
				FieldName: "date",
				Reason:    "unsupported date format",
				Msg:       "could not parse date",
			},
			expected: "data extraction failed in file '/path/to/file.csv' for field 'date': could not parse date. Reason: unsupported date format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// Test error wrapping and unwrapping patterns
func TestErrorWrappingPatterns(t *testing.T) {
	t.Run("ParseError can be wrapped and unwrapped", func(t *testing.T) {
		originalErr := errors.New("original error")
		parseErr := &ParseError{
			Parser: "CAMT",
			Field:  "amount",
			Value:  "invalid",
			Err:    originalErr,
		}

		// Test direct unwrapping
		assert.Equal(t, originalErr, parseErr.Unwrap())

		// Test errors.Is
		assert.True(t, errors.Is(parseErr, originalErr))

		// Test errors.As
		var targetParseErr *ParseError
		assert.True(t, errors.As(parseErr, &targetParseErr))
		assert.Equal(t, parseErr, targetParseErr)
	})

	t.Run("CategorizationError can be wrapped and unwrapped", func(t *testing.T) {
		originalErr := errors.New("network timeout")
		catErr := &CategorizationError{
			Transaction: "TX123",
			Strategy:    "AIStrategy",
			Err:         originalErr,
		}

		// Test direct unwrapping
		assert.Equal(t, originalErr, catErr.Unwrap())

		// Test errors.Is
		assert.True(t, errors.Is(catErr, originalErr))

		// Test errors.As
		var targetCatErr *CategorizationError
		assert.True(t, errors.As(catErr, &targetCatErr))
		assert.Equal(t, catErr, targetCatErr)
	})

	t.Run("ValidationError implements Unwrap", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		valErr := &ValidationError{
			FilePath: "/path/to/file.xml",
			Reason:   "invalid format",
			Err:      underlyingErr,
		}

		// ValidationError should have an Unwrap method
		assert.Implements(t, (*interface{ Unwrap() error })(nil), valErr)
		assert.Equal(t, underlyingErr, valErr.Unwrap())

		// Test with nil underlying error
		valErrNoWrap := &ValidationError{
			FilePath: "/path/to/file.xml",
			Reason:   "invalid format",
		}
		assert.Nil(t, valErrNoWrap.Unwrap())
	})
}

// Test error type assertions
func TestErrorTypeAssertions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected interface{}
	}{
		{
			name: "ParseError type assertion",
			err: &ParseError{
				Parser: "CAMT",
				Field:  "amount",
				Value:  "invalid",
				Err:    errors.New("test"),
			},
			expected: &ParseError{},
		},
		{
			name: "ValidationError type assertion",
			err: &ValidationError{
				FilePath: "/path/to/file.xml",
				Reason:   "invalid format",
			},
			expected: &ValidationError{},
		},
		{
			name: "CategorizationError type assertion",
			err: &CategorizationError{
				Transaction: "TX123",
				Strategy:    "AIStrategy",
				Err:         errors.New("test"),
			},
			expected: &CategorizationError{},
		},
		{
			name: "InvalidFormatError type assertion",
			err: &InvalidFormatError{
				FilePath:       "/path/to/file.pdf",
				ExpectedFormat: "PDF",
				Msg:            "test",
			},
			expected: &InvalidFormatError{},
		},
		{
			name: "DataExtractionError type assertion",
			err: &DataExtractionError{
				FilePath:  "/path/to/file.xml",
				FieldName: "amount",
				Reason:    "test",
				Msg:       "test",
			},
			expected: &DataExtractionError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.IsType(t, tt.expected, tt.err)
		})
	}
}
