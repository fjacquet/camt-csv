// Package parser provides the base parser functionality and common interfaces.
package parser

import (
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// BaseParser provides common functionality for all parser implementations.
// It implements the LoggerConfigurable interface and provides shared methods
// that eliminate code duplication across different parser types.
//
// Parsers should embed BaseParser to inherit common functionality:
//
//	type MyParser struct {
//		BaseParser
//		// parser-specific fields
//	}
//
// This follows the composition pattern and provides a foundation for
// implementing the segregated parser interfaces.
type BaseParser struct {
	logger      logging.Logger
	categorizer models.TransactionCategorizer
}

// NewBaseParser creates a new BaseParser instance with the provided logger.
// If logger is nil, a default logger will be used.
//
// Parameters:
//   - logger: The logger instance to use for logging operations
//
// Returns:
//   - BaseParser: A new BaseParser instance ready for embedding
func NewBaseParser(logger logging.Logger) BaseParser {
	if logger == nil {
		// Use a default logger if none provided
		logger = logging.NewLogrusAdapter("info", "text")
	}

	return BaseParser{
		logger: logger,
	}
}

// SetLogger implements the LoggerConfigurable interface.
// This method allows parsers to configure their logging instance.
//
// Parameters:
//   - logger: The new logger instance to use
func (b *BaseParser) SetLogger(logger logging.Logger) {
	if logger != nil {
		b.logger = logger
	}
}

// GetLogger returns the current logger instance.
// This is a helper method for parser implementations to access the logger.
//
// Returns:
//   - logging.Logger: The current logger instance
func (b *BaseParser) GetLogger() logging.Logger {
	return b.logger
}

// SetCategorizer implements the CategorizerConfigurable interface.
// This method allows parsers to configure their categorizer instance.
//
// Parameters:
//   - categorizer: The categorizer instance to use for transaction categorization
func (b *BaseParser) SetCategorizer(categorizer models.TransactionCategorizer) {
	b.categorizer = categorizer
}

// GetCategorizer returns the current categorizer instance.
// This is a helper method for parser implementations to access the categorizer.
//
// Returns:
//   - models.TransactionCategorizer: The current categorizer instance (nil if not set)
func (b *BaseParser) GetCategorizer() models.TransactionCategorizer {
	return b.categorizer
}
