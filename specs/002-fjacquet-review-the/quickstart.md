# Quickstart Guide: Codebase Improvements

**Date**: mardi 14 octobre 2025

This quickstart guide provides an overview for developers on how to interact with and leverage the new codebase improvements introduced by this feature.

## 1. Centralized Parser Management (Parser Factory)

**Goal**: Easily instantiate different parsers and add new ones.

**How to use**:

Instead of directly instantiating parser adapters (e.g., `camtparser.NewAdapter()`), developers will use a centralized `parser.GetParser` function. This function will return the appropriate `parser.Parser` interface implementation based on a `parser.ParserType`.

**Example (Conceptual)**:

```go
package main

import (
	"fmt"
	"os"

	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/models"
)

func main() {
	// Example: Get a CAMT parser
	camtParser, err := parser.GetParser(parser.CAMT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting CAMT parser: %v\n", err)
		os.Exit(1)
	}

	// Assuming 'someReader' is an io.Reader for your CAMT file
	// transactions, err := camtParser.Parse(someReader)
	// if err != nil {
	//	fmt.Fprintf(os.Stderr, "Error parsing CAMT file: %v\n", err)
	//	os.Exit(1)
	// }

	fmt.Println("Successfully obtained CAMT parser.")

	// To add a new parser:
	// 1. Create a new adapter that implements the `parser.Parser` interface.
	// 2. Define a new `parser.ParserType` constant.
	// 3. Add a new case to the `switch` statement in `parser.GetParser`.
}
```

## 2. Improved Error Handling with Custom Errors

**Goal**: Receive specific and programmatic error types for parsing and data extraction failures.

**How to use**:

When interacting with `parser.Parser` implementations, developers can now expect and handle specific error types defined in `internal/parsererror` (e.g., `InvalidFormatError`, `DataExtractionError`). This allows for more granular error handling logic.

**Example (Conceptual)**:

```go
package main

import (
	"errors"
	"fmt"
	"os"

	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/parsererror"
)

func processFile(p parser.Parser, reader io.Reader) error {
	_, err := p.Parse(reader)
	if err != nil {
		var invalidFormatErr *parsererror.InvalidFormatError
		var dataExtractionErr *parsererror.DataExtractionError

		if errors.As(err, &invalidFormatErr) {
			return fmt.Errorf("file format error in %s: %s (expected %s)", invalidFormatErr.FilePath, invalidFormatErr.Error(), invalidFormatErr.ExpectedFormat)
		} else if errors.As(err, &dataExtractionErr) {
			return fmt.Errorf("data extraction error in %s for field %s: %s", dataExtractionErr.FilePath, dataExtractionErr.FieldName, dataExtractionErr.Error())
		} else {
			return fmt.Errorf("general parsing error: %w", err)
		}
	}
	return nil
}
```

## 3. Testable AI Categorization Logic

**Goal**: Easily test the core categorization logic without making actual calls to the Gemini API.

**How to use**:

The `Categorizer` will now depend on an `internal/categorizer/AIClient` interface. For unit testing, developers can provide a mock implementation of this interface.

**Example (Conceptual - Test File)**:

```go
package categorizer_test

import (
	"context"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/models"
	"github.com/stretchr/testify/assert"
)

// MockAIClient implements the AIClient interface for testing purposes.
type MockAIClient struct {
	CategorizeFunc func(ctx context.Context, transaction models.Transaction) (models.Transaction, error)
}

func (m *MockAIClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	return m.CategorizeFunc(ctx, transaction)
}

func TestCategorizerWithMockAI(t *testing.T) {
	mockClient := &MockAIClient{
		CategorizeFunc: func(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
			// Simulate AI categorization without actual API call
			ransaction.Category = "Mocked Category"
			return transaction, nil
		},
	}

	categorizer := categorizer.NewCategorizer(mockClient) // Assuming NewCategorizer takes an AIClient

	inputTransaction := models.Transaction{Description: "Coffee Shop"}
	ctx := context.Background()

	categorizedTransaction, err := categorizer.CategorizeTransaction(ctx, inputTransaction)

	assert.NoError(t, err)
	assert.Equal(t, "Mocked Category", categorizedTransaction.Category)
}
```

## 4. Standardized Logging

**Goal**: Use consistent field names for structured logging across the application.

**How to use**:

Developers should use predefined constants from `internal/logging/constants.go` (or similar) when adding structured fields to log messages using `logrus`.

**Example (Conceptual)**:

```go
package somepackage

import (
	"fjacquet/camt-csv/internal/logging"
	"github.com/sirupsen/logrus"
)

func processData(filePath string, parserType string) {
	logrus.WithFields(logrus.Fields{
		logging.FieldFile:   filePath,
		logging.FieldParser: parserType,
		"operation":         "start_parsing",
	}).Info("Starting data processing")

	// ... processing logic ...

	logrus.WithFields(logrus.Fields{
		logging.FieldFile:   filePath,
		logging.FieldParser: parserType,
		"operation":         "parsing_complete",
		"status":            "success",
	}).Info("Data processing finished")
}
```
