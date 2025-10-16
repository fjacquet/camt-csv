# Component Interfaces (Contracts)

**Date**: mardi 14 octobre 2025

This document outlines the key interfaces that define the contracts between different components within the `camt-csv` application, as per the feature specification.

## 1. Parser Interface

**Location**: `internal/parser/parser.go` (existing, to be strictly adhered to by all new/refactored parsers)

**Description**: The `Parser` interface standardizes how different financial data formats are processed and converted into a common `Transaction` model. This ensures interchangeability and simplifies the integration of new data sources.

```go
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
```

**Key Methods**:
- `Parse(r io.Reader) ([]models.Transaction, error)`: Reads from an `io.Reader` and returns a slice of `Transaction` models or an error.

**Functional Requirements Addressed**:
- FR-001: Centralized mechanism for instantiating different parser implementations.
- FR-003: Use of custom error types (`InvalidFormatError`, `DataExtractionError`).

## 2. AIClient Interface

**Location**: `internal/categorizer/ai_client.go` (new file)

**Description**: The `AIClient` interface abstracts the interaction with AI-based categorization services, allowing the core categorization logic to be tested independently of external API calls. This promotes testability and flexibility in choosing AI providers.

```go
package categorizer

import (
	"context"

	"fjacquet/camt-csv/internal/models"
)

type AIClient interface {
	// Categorize takes a context and a Transaction model, and returns the categorized Transaction
	// or an error if categorization fails.
	// Implementations will interact with an external AI service (e.g., Google Gemini).
	Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error)
}
```

**Key Methods**:
- `Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error)`: Takes a `context.Context` and a `Transaction`, and returns a categorized `Transaction` or an error.

**Functional Requirements Addressed**:
- FR-004: Categorization logic refactored to depend on an `AIClient` interface.
- FR-005: An implementation of `AIClient` for the Gemini API (`GeminiClient`) MUST be provided.
