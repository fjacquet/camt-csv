package parser

import (
	"io"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// Parser defines the core parsing capability.
// This interface follows the Interface Segregation Principle by containing only
// the essential parsing method that all parsers must implement.
type Parser interface {
	// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
	// It is responsible for understanding the specific input format (e.g., CAMT XML, PDF, CSV)
	// and transforming it into the standardized Transaction structure.
	// Implementations should return custom error types (e.g., InvalidFormatError, DataExtractionError)
	// for specific parsing failures.
	Parse(r io.Reader) ([]models.Transaction, error)
}

// Validator defines format validation capability.
// Not all parsers need validation, so this is separated from the core Parser interface.
type Validator interface {
	// ValidateFormat checks if the given file path contains data in the expected format.
	// Returns true if the format is valid, false otherwise, along with any error encountered.
	ValidateFormat(filePath string) (bool, error)
}

// CSVConverter defines CSV conversion capability.
// This interface allows parsers to provide a convenient method for converting
// input files directly to CSV format without requiring separate Parse and Write steps.
type CSVConverter interface {
	// ConvertToCSV converts an input file to CSV format and writes it to the output file.
	// This is a convenience method that typically combines Parse and WriteToCSV operations.
	ConvertToCSV(inputFile, outputFile string) error
}

// LoggerConfigurable defines the ability to configure logging.
// This allows parsers to accept logger instances for structured logging.
type LoggerConfigurable interface {
	// SetLogger configures the logger instance for the parser.
	// Parsers should use this logger for all logging operations.
	SetLogger(logger logging.Logger)
}

// FullParser combines all parser capabilities into a single interface.
// Use this interface when you need a parser with all available features.
// Individual interfaces should be used when only specific capabilities are required.
type FullParser interface {
	Parser
	Validator
	CSVConverter
	LoggerConfigurable
}
