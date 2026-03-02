// Package formatter provides output formatting capabilities for transactions.
// It follows the strategy pattern to allow different output formats (standard CSV, iCompta, etc.)
// without modifying parser logic.
package formatter

import (
	"fmt"

	"fjacquet/camt-csv/internal/models"
)

// OutputFormatter defines the interface for formatting transactions into CSV output.
// Implementations can provide different column layouts, date formats, and delimiters
// to support various import targets (standard format, iCompta, etc.).
//
// This interface follows the Interface Segregation Principle by containing only
// the essential formatting methods that all formatters must implement.
type OutputFormatter interface {
	// Header returns the CSV column names for this format.
	// The order of columns returned here must match the order of values
	// returned by Format() for each transaction row.
	Header() []string

	// Format converts a slice of transactions into CSV rows.
	// Each inner slice represents one row of CSV data, with values in the
	// same order as the Header() columns.
	//
	// Returns:
	// - [][]string: 2D slice where each inner slice is a CSV row
	// - error: formatting error if any field cannot be converted
	Format(transactions []models.Transaction) ([][]string, error)

	// Delimiter returns the preferred CSV delimiter for this format.
	// Common values: ',' (comma) for standard CSV, ';' (semicolon) for
	// European formats like iCompta.
	Delimiter() rune
}

// FormatterRegistry manages available output formatters.
// It provides a centralized registry for looking up formatters by name
// and supports extensibility through the Register method.
type FormatterRegistry struct {
	formatters map[string]OutputFormatter
}

// NewFormatterRegistry creates a new registry with built-in formatters pre-registered.
// The following formatters are registered by default:
// - "standard": StandardFormatter (35-column backward-compatible format)
// - "icompta": iComptaFormatter (10-column semicolon-delimited format)
func NewFormatterRegistry() *FormatterRegistry {
	registry := &FormatterRegistry{
		formatters: make(map[string]OutputFormatter),
	}

	// Register built-in formatters
	registry.Register("standard", NewStandardFormatter())
	registry.Register("icompta", NewIComptaFormatter())
	registry.Register("jumpsoft", NewJumpsoftFormatter())

	return registry
}

// Get retrieves a formatter by name.
// Returns an error if the formatter is not found.
//
// Parameters:
// - name: formatter name (e.g., "standard", "icompta")
//
// Returns:
// - OutputFormatter: the requested formatter
// - error: ErrFormatterNotFound if the name is not registered
func (r *FormatterRegistry) Get(name string) (OutputFormatter, error) {
	formatter, ok := r.formatters[name]
	if !ok {
		return nil, fmt.Errorf("formatter not found: %s", name)
	}
	return formatter, nil
}

// Register adds a new formatter to the registry.
// If a formatter with the same name already exists, it will be replaced.
//
// This method enables extensibility, allowing users to add custom formatters
// without modifying the registry code.
//
// Parameters:
// - name: unique identifier for the formatter
// - formatter: OutputFormatter implementation to register
func (r *FormatterRegistry) Register(name string, formatter OutputFormatter) {
	r.formatters[name] = formatter
}
