package parsererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError_ErrorBranches(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "with underlying error includes error in message",
			err: &ValidationError{
				FilePath: "/data/input.xml",
				Reason:   "malformed XML",
				Err:      errors.New("unexpected EOF"),
			},
			expected: "validation failed for /data/input.xml: malformed XML: unexpected EOF",
		},
		{
			name: "without underlying error omits error suffix",
			err: &ValidationError{
				FilePath: "/data/input.xml",
				Reason:   "missing root element",
				Err:      nil,
			},
			expected: "validation failed for /data/input.xml: missing root element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestInvalidFormatError_ErrorBranches(t *testing.T) {
	underlying := errors.New("read error")

	tests := []struct {
		name     string
		err      *InvalidFormatError
		expected string
	}{
		{
			name: "with snippet and underlying error",
			err: &InvalidFormatError{
				FilePath:             "/data/file.pdf",
				ExpectedFormat:       "PDF",
				ActualContentSnippet: "<?xml",
				Msg:                  "looks like XML",
				Err:                  underlying,
			},
			expected: "invalid format in file '/data/file.pdf': looks like XML. Expected: PDF. Content snippet: '<?xml': read error",
		},
		{
			name: "with snippet and nil error",
			err: &InvalidFormatError{
				FilePath:             "/data/file.pdf",
				ExpectedFormat:       "PDF",
				ActualContentSnippet: "PK\x03\x04",
				Msg:                  "looks like ZIP",
				Err:                  nil,
			},
			expected: "invalid format in file '/data/file.pdf': looks like ZIP. Expected: PDF. Content snippet: 'PK\x03\x04'",
		},
		{
			name: "without snippet and with underlying error",
			err: &InvalidFormatError{
				FilePath:       "/data/file.csv",
				ExpectedFormat: "Revolut CSV",
				Msg:            "header mismatch",
				Err:            underlying,
			},
			expected: "invalid format in file '/data/file.csv': header mismatch. Expected: Revolut CSV: read error",
		},
		{
			name: "without snippet and without underlying error",
			err: &InvalidFormatError{
				FilePath:       "/data/file.csv",
				ExpectedFormat: "Revolut CSV",
				Msg:            "empty file",
				Err:            nil,
			},
			expected: "invalid format in file '/data/file.csv': empty file. Expected: Revolut CSV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestInvalidFormatError_Unwrap(t *testing.T) {
	tests := []struct {
		name        string
		err         *InvalidFormatError
		expectedErr error
	}{
		{
			name: "returns underlying error when present",
			err: &InvalidFormatError{
				FilePath:       "/data/file.pdf",
				ExpectedFormat: "PDF",
				Msg:            "bad format",
				Err:            errors.New("io timeout"),
			},
			expectedErr: errors.New("io timeout"),
		},
		{
			name: "returns nil when no underlying error",
			err: &InvalidFormatError{
				FilePath:       "/data/file.pdf",
				ExpectedFormat: "PDF",
				Msg:            "bad format",
				Err:            nil,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := tt.err.Unwrap()
			if tt.expectedErr == nil {
				assert.Nil(t, unwrapped)
			} else {
				assert.EqualError(t, unwrapped, tt.expectedErr.Error())
			}
		})
	}
}

func TestInvalidFormatError_UnwrapWithErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := &InvalidFormatError{
		FilePath:       "/data/file.pdf",
		ExpectedFormat: "PDF",
		Msg:            "bad",
		Err:            sentinel,
	}

	assert.True(t, errors.Is(err, sentinel))
	assert.False(t, errors.Is(err, errors.New("other")))
}

func TestDataExtractionError_ErrorBranches(t *testing.T) {
	underlying := errors.New("parse failure")

	tests := []struct {
		name     string
		err      *DataExtractionError
		expected string
	}{
		{
			name: "with snippet and underlying error",
			err: &DataExtractionError{
				FilePath:       "/data/stmt.xml",
				FieldName:      "amount",
				RawDataSnippet: "<Amt>bad</Amt>",
				Reason:         "not a number",
				Msg:            "amount extraction failed",
				Err:            underlying,
			},
			expected: "data extraction failed in file '/data/stmt.xml' for field 'amount': amount extraction failed. Reason: not a number. Raw data snippet: '<Amt>bad</Amt>': parse failure",
		},
		{
			name: "with snippet and nil error",
			err: &DataExtractionError{
				FilePath:       "/data/stmt.xml",
				FieldName:      "date",
				RawDataSnippet: "<Dt>??</Dt>",
				Reason:         "unrecognized format",
				Msg:            "date extraction failed",
				Err:            nil,
			},
			expected: "data extraction failed in file '/data/stmt.xml' for field 'date': date extraction failed. Reason: unrecognized format. Raw data snippet: '<Dt>??</Dt>'",
		},
		{
			name: "without snippet and with underlying error",
			err: &DataExtractionError{
				FilePath:  "/data/stmt.csv",
				FieldName: "currency",
				Reason:    "column missing",
				Msg:       "currency not found",
				Err:       underlying,
			},
			expected: "data extraction failed in file '/data/stmt.csv' for field 'currency': currency not found. Reason: column missing: parse failure",
		},
		{
			name: "without snippet and without underlying error",
			err: &DataExtractionError{
				FilePath:  "/data/stmt.csv",
				FieldName: "balance",
				Reason:    "empty value",
				Msg:       "balance missing",
				Err:       nil,
			},
			expected: "data extraction failed in file '/data/stmt.csv' for field 'balance': balance missing. Reason: empty value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestDataExtractionError_Unwrap(t *testing.T) {
	tests := []struct {
		name        string
		err         *DataExtractionError
		expectedErr error
	}{
		{
			name: "returns underlying error when present",
			err: &DataExtractionError{
				FilePath:  "/data/stmt.xml",
				FieldName: "amount",
				Reason:    "bad",
				Msg:       "fail",
				Err:       errors.New("disk error"),
			},
			expectedErr: errors.New("disk error"),
		},
		{
			name: "returns nil when no underlying error",
			err: &DataExtractionError{
				FilePath:  "/data/stmt.xml",
				FieldName: "amount",
				Reason:    "bad",
				Msg:       "fail",
				Err:       nil,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := tt.err.Unwrap()
			if tt.expectedErr == nil {
				assert.Nil(t, unwrapped)
			} else {
				assert.EqualError(t, unwrapped, tt.expectedErr.Error())
			}
		})
	}
}

func TestDataExtractionError_UnwrapWithErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := &DataExtractionError{
		FilePath:  "/data/stmt.xml",
		FieldName: "amount",
		Reason:    "bad",
		Msg:       "fail",
		Err:       sentinel,
	}

	assert.True(t, errors.Is(err, sentinel))
	assert.False(t, errors.Is(err, errors.New("other")))
}
