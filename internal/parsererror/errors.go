// Package parsererror provides standardized error types and handling for all parsers.
package parsererror

import (
	"errors"
	"fmt"
)

var (
	// ErrFileNotFound is returned when a file cannot be found
	ErrFileNotFound = errors.New("file not found")

	// ErrInvalidFormat is returned when a file has an invalid format
	ErrInvalidFormat = errors.New("invalid file format")

	// ErrParsing is returned when there's an error parsing file contents
	ErrParsing = errors.New("parsing error")

	// ErrIO is returned for general input/output errors
	ErrIO = errors.New("input/output error")

	// ErrInternal is returned for internal processing errors
	ErrInternal = errors.New("internal error")
)

// FileNotFoundError creates a new file not found error with the given path
func FileNotFoundError(path string) error {
	return fmt.Errorf("%w: %s", ErrFileNotFound, path)
}

// InvalidFormatError creates a new invalid format error with the given details
func InvalidFormatError(path, details string) error {
	if details == "" {
		return fmt.Errorf("%w: %s", ErrInvalidFormat, path)
	}
	return fmt.Errorf("%w: %s (%s)", ErrInvalidFormat, path, details)
}

// ParsingError creates a new parsing error with the given details
func ParsingError(details string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrParsing, details)
	}
	return fmt.Errorf("%w: %s: %v", ErrParsing, details, err)
}

// IOError creates a new IO error with the given details
func IOError(operation, path string, err error) error {
	return fmt.Errorf("%w: %s %s: %v", ErrIO, operation, path, err)
}

// IsFileNotFound checks if an error is a file not found error
func IsFileNotFound(err error) bool {
	return errors.Is(err, ErrFileNotFound)
}

// IsInvalidFormat checks if an error is an invalid format error
func IsInvalidFormat(err error) bool {
	return errors.Is(err, ErrInvalidFormat)
}

// IsParsing checks if an error is a parsing error
func IsParsing(err error) bool {
	return errors.Is(err, ErrParsing)
}

// IsIO checks if an error is an IO error
func IsIO(err error) bool {
	return errors.Is(err, ErrIO)
}
