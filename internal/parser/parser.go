// Package parser provides a common interface for all parsers in the application.
package parser

import "github.com/sirupsen/logrus"

// Parser defines the common interface for all file parsers in the application.
type Parser interface {
	// ValidateFormat checks if a file is in the correct format for this parser.
	ValidateFormat(filePath string) (bool, error)

	// ConvertToCSV converts a file to CSV format.
	ConvertToCSV(inputFile, outputFile string) error

	// SetLogger configures the logger for this parser.
	SetLogger(logger *logrus.Logger)
}

// RunConvert is a helper function that encapsulates the common validate-and-convert flow.
func RunConvert(p Parser, inputFile, outputFile string, validate bool) error {
	if validate {
		valid, err := p.ValidateFormat(inputFile)
		if err != nil {
			return err
		}
		if !valid {
			return ErrInvalidFormat
		}
	}

	return p.ConvertToCSV(inputFile, outputFile)
}

// ErrInvalidFormat is returned when a file is not in the expected format.
var ErrInvalidFormat = NewError("file is not in valid format")

// Error represents a parser error.
type Error struct {
	message string
}

// NewError creates a new parser error with the given message.
func NewError(msg string) Error {
	return Error{message: msg}
}

// Error returns the error message.
func (e Error) Error() string {
	return e.message
}
