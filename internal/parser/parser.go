package parser

import (
	"io"

	"fjacquet/camt-csv/internal/models"
)

type Parser interface {
	// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
	// It is responsible for understanding the specific input format (e.g., CAMT XML, PDF, CSV)
	// and transforming it into the standardized Transaction structure.
	// Implementations should return custom error types (e.g., InvalidFormatError, DataExtractionError)
	// for specific parsing failures.
	Parse(r io.Reader) ([]models.Transaction, error)
}
