# Developer Guide

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Environment](#development-environment)
3. [Architecture Overview](#architecture-overview)
4. [Adding New Parsers](#adding-new-parsers)
5. [Adding Categorization Strategies](#adding-categorization-strategies)
6. [Testing Guidelines](#testing-guidelines)
7. [Code Quality Standards](#code-quality-standards)
8. [Debugging and Troubleshooting](#debugging-and-troubleshooting)
9. [Performance Considerations](#performance-considerations)
10. [Contributing Guidelines](#contributing-guidelines)

## Getting Started

### Prerequisites

- **Go 1.24.2 or higher**: [Download Go](https://golang.org/dl/)
- **Git**: For version control
- **pdftotext**: For PDF processing (`brew install poppler` on macOS)
- **golangci-lint**: For code quality checks
- **IDE/Editor**: VS Code, GoLand, or similar with Go support

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv

# Install dependencies
go mod download
go mod tidy

# Build the application
go build -o camt-csv

# Run tests to verify setup
go test ./...

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Project Structure

```
camt-csv/
├── cmd/                    # CLI command implementations
│   ├── root/              # Root cobra command
│   ├── camt/              # CAMT.053 XML conversion
│   ├── pdf/               # PDF conversion
│   └── ...                # Other format commands
├── internal/              # Private application code
│   ├── models/            # Core data structures
│   ├── parser/            # Parser interfaces and base
│   ├── categorizer/       # Transaction categorization
│   ├── logging/           # Logging abstraction
│   ├── container/         # Dependency injection
│   └── ...                # Format-specific parsers
├── database/              # Configuration YAML files
├── docs/                  # Documentation
├── samples/               # Sample input files
└── main.go               # Application entry point
```

## Development Environment

### IDE Configuration

**VS Code Settings (`.vscode/settings.json`):**
```json
{
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": "package",
    "go.testFlags": ["-v"],
    "go.testTimeout": "30s",
    "editor.formatOnSave": true,
    "go.formatTool": "goimports"
}
```

**Recommended Extensions:**
- Go (Google)
- Go Test Explorer
- YAML (Red Hat)
- Markdown All in One

### Development Commands

```bash
# Format code
go fmt ./...
goimports -w .

# Run linters
golangci-lint run

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Build with debug info
go build -gcflags="all=-N -l" -o camt-csv-debug

# Run specific tests
go test -run TestParserName ./internal/camtparser/
go test -v ./internal/categorizer/ -run TestCategorizer

# Benchmark tests
go test -bench=. ./internal/categorizer/
```

## Architecture Overview

### Dependency Injection Pattern

The application uses dependency injection to eliminate global state and improve testability:

```go
// Container manages all dependencies
type Container struct {
    Logger      logging.Logger
    Config      *config.Config
    Store       *store.CategoryStore
    Categorizer *categorizer.Categorizer
    Parsers     map[parser.ParserType]parser.FullParser
}

// All components receive dependencies through constructors
func NewMyParser(logger logging.Logger) *MyParser {
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}
```

### Interface Segregation

Parsers implement only the interfaces they need:

```go
type Parser interface {
    Parse(r io.Reader) ([]models.Transaction, error)
}

type Validator interface {
    ValidateFormat(filePath string) (bool, error)
}

type CSVConverter interface {
    ConvertToCSV(inputFile, outputFile string) error
}
```

### BaseParser Foundation

All parsers embed `BaseParser` for common functionality:

```go
type MyParser struct {
    parser.BaseParser  // Provides logging and CSV writing
    // parser-specific fields
}
```

## Adding New Parsers

### Step-by-Step Guide

#### 1. Create Parser Package

```bash
mkdir internal/myformatparser
```

#### 2. Define Parser Structure

**File: `internal/myformatparser/myformatparser.go`**
```go
package myformatparser

import (
    "io"
    "github.com/fjacquet/camt-csv/internal/logging"
    "github.com/fjacquet/camt-csv/internal/models"
    "github.com/fjacquet/camt-csv/internal/parser"
    "github.com/fjacquet/camt-csv/internal/parsererror"
)

// MyFormatParser handles parsing of MyFormat files
type MyFormatParser struct {
    parser.BaseParser
    // Add parser-specific fields here
}

// NewMyFormatParser creates a new MyFormat parser with dependency injection
func NewMyFormatParser(logger logging.Logger) *MyFormatParser {
    return &MyFormatParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}

// Parse implements the parser.Parser interface
func (p *MyFormatParser) Parse(r io.Reader) ([]models.Transaction, error) {
    p.GetLogger().Info("Starting MyFormat parsing")
    
    // Read and validate input
    data, err := io.ReadAll(r)
    if err != nil {
        return nil, &parsererror.ParseError{
            Parser: "MyFormat",
            Field:  "input",
            Err:    err,
        }
    }
    
    // Parse the data
    transactions, err := p.parseData(data)
    if err != nil {
        return nil, err
    }
    
    p.GetLogger().Info("MyFormat parsing completed",
        logging.Field{Key: "count", Value: len(transactions)})
    
    return transactions, nil
}

// ValidateFormat implements the parser.Validator interface (optional)
func (p *MyFormatParser) ValidateFormat(filePath string) (bool, error) {
    // Implement format validation logic
    return true, nil
}

// parseData contains the core parsing logic
func (p *MyFormatParser) parseData(data []byte) ([]models.Transaction, error) {
    var transactions []models.Transaction
    
    // Implement parsing logic here
    // Use TransactionBuilder for creating transactions:
    
    tx, err := models.NewTransactionBuilder().
        WithDate("2025-01-15").
        WithAmount(decimal.NewFromFloat(100.50), "CHF").
        WithPayer("John Doe", "CH1234567890").
        WithPayee("Acme Corp", "CH0987654321").
        AsDebit().
        Build()
    
    if err != nil {
        return nil, &parsererror.ParseError{
            Parser: "MyFormat",
            Field:  "transaction",
            Err:    err,
        }
    }
    
    transactions = append(transactions, tx)
    
    return transactions, nil
}
```

#### 3. Create Adapter

**File: `internal/myformatparser/adapter.go`**
```go
package myformatparser

import (
    "github.com/fjacquet/camt-csv/internal/logging"
    "github.com/fjacquet/camt-csv/internal/parser"
)

// Adapter implements the parser interfaces for MyFormat files
type Adapter struct {
    parser.BaseParser
}

// NewAdapter creates a new adapter for MyFormat parser
func NewAdapter(logger logging.Logger) *Adapter {
    return &Adapter{
        BaseParser: parser.NewBaseParser(logger),
    }
}

// Parse delegates to the MyFormatParser
func (a *Adapter) Parse(r io.Reader) ([]models.Transaction, error) {
    parser := NewMyFormatParser(a.GetLogger())
    return parser.Parse(r)
}
```

#### 4. Add Comprehensive Tests

**File: `internal/myformatparser/myformatparser_test.go`**
```go
package myformatparser

import (
    "strings"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/fjacquet/camt-csv/internal/logging"
)

func TestMyFormatParser_Parse(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expected    int
        expectError bool
    }{
        {
            name:        "valid input",
            input:       "sample,data,here",
            expected:    1,
            expectError: false,
        },
        {
            name:        "invalid input",
            input:       "invalid",
            expected:    0,
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create mock logger
            logger := &MockLogger{}
            parser := NewMyFormatParser(logger)
            
            // Execute
            reader := strings.NewReader(tt.input)
            transactions, err := parser.Parse(reader)
            
            // Assert
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Len(t, transactions, tt.expected)
            }
        })
    }
}

// MockLogger for testing
type MockLogger struct {
    messages []string
}

func (m *MockLogger) Info(msg string, fields ...logging.Field) {
    m.messages = append(m.messages, msg)
}

// Implement other Logger interface methods...
```

#### 5. Add CLI Command

**File: `cmd/myformat/convert.go`**
```go
package myformat

import (
    "github.com/spf13/cobra"
    "github.com/fjacquet/camt-csv/internal/container"
    "github.com/fjacquet/camt-csv/internal/factory"
)

// NewConvertCmd creates the myformat conversion command
func NewConvertCmd() *cobra.Command {
    var inputFile, outputFile string
    
    cmd := &cobra.Command{
        Use:   "myformat",
        Short: "Convert MyFormat files to CSV",
        Long:  "Convert MyFormat files to standardized CSV format with transaction categorization",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Create container with dependencies
            container, err := container.NewContainer(config.GetGlobalConfig())
            if err != nil {
                return err
            }
            
            // Get parser from container
            parser, err := container.GetParser(factory.MyFormat)
            if err != nil {
                return err
            }
            
            // Execute conversion
            return parser.ConvertToCSV(inputFile, outputFile)
        },
    }
    
    cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input MyFormat file")
    cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output CSV file")
    cmd.MarkFlagRequired("input")
    cmd.MarkFlagRequired("output")
    
    return cmd
}
```

#### 6. Register Parser in Factory

**File: `internal/factory/factory.go`**
```go
const (
    // ... existing parser types
    MyFormat ParserType = "myformat"
)

func GetParserWithLogger(parserType ParserType, logger logging.Logger) (models.Parser, error) {
    switch parserType {
    // ... existing cases
    case MyFormat:
        return myformatparser.NewAdapter(logger), nil
    default:
        return nil, fmt.Errorf("unknown parser type: %s", parserType)
    }
}
```

#### 7. Add Sample Files

```bash
mkdir samples/myformat
# Add sample input files for testing
```

#### 8. Update Documentation

Update `README.md` and `docs/user-guide.md` to include the new parser.

### Parser Best Practices

#### Error Handling

```go
// Use custom error types with context
if err != nil {
    return nil, &parsererror.ParseError{
        Parser: "MyFormat",
        Field:  "amount",
        Value:  rawValue,
        Err:    err,
    }
}

// Log warnings for recoverable issues
if amount.IsZero() {
    p.GetLogger().Warn("Zero amount detected, continuing",
        logging.Field{Key: "line", Value: lineNumber})
}
```

#### Constants Usage

```go
// Use constants instead of magic strings
transaction.CreditDebit = models.TransactionTypeDebit
transaction.Category = models.CategoryUncategorized
```

#### Structured Logging

```go
p.GetLogger().Info("Processing transaction",
    logging.Field{Key: "file", Value: filename},
    logging.Field{Key: "line", Value: lineNumber},
    logging.Field{Key: "amount", Value: amount.String()})
```

## Adding Categorization Strategies

### Strategy Interface

```go
type CategorizationStrategy interface {
    Categorize(ctx context.Context, tx Transaction) (Category, bool, error)
    Name() string
}
```

### Example Implementation

**File: `internal/categorizer/my_strategy.go`**
```go
package categorizer

import (
    "context"
    "strings"
    
    "github.com/fjacquet/camt-csv/internal/logging"
    "github.com/fjacquet/camt-csv/internal/models"
)

// MyStrategy implements a custom categorization strategy
type MyStrategy struct {
    logger logging.Logger
    rules  map[string]string
}

// NewMyStrategy creates a new instance of MyStrategy
func NewMyStrategy(logger logging.Logger) *MyStrategy {
    return &MyStrategy{
        logger: logger,
        rules:  make(map[string]string),
    }
}

// Name returns the strategy name for logging
func (s *MyStrategy) Name() string {
    return "MyStrategy"
}

// Categorize attempts to categorize a transaction using custom logic
func (s *MyStrategy) Categorize(ctx context.Context, tx models.Transaction) (models.Category, bool, error) {
    // Implement categorization logic
    description := strings.ToLower(tx.Description)
    
    for pattern, category := range s.rules {
        if strings.Contains(description, pattern) {
            s.logger.Debug("Transaction categorized",
                logging.Field{Key: "strategy", Value: s.Name()},
                logging.Field{Key: "pattern", Value: pattern},
                logging.Field{Key: "category", Value: category})
            
            return models.Category{Name: category}, true, nil
        }
    }
    
    return models.Category{}, false, nil
}
```

### Register Strategy

**File: `internal/categorizer/categorizer.go`**
```go
func NewCategorizer(store *store.CategoryStore, aiClient AIClient, logger logging.Logger) *Categorizer {
    c := &Categorizer{
        store:  store,
        logger: logger,
    }
    
    // Initialize strategies in priority order
    c.strategies = []CategorizationStrategy{
        NewDirectMappingStrategy(store, logger),
        NewKeywordStrategy(store, logger),
        NewMyStrategy(logger),  // Add your strategy
        NewAIStrategy(aiClient, logger),
    }
    
    return c
}
```

## Testing Guidelines

### Unit Testing Structure

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name        string
        input       InputType
        expected    ExpectedType
        expectError bool
    }{
        {
            name:        "valid case",
            input:       validInput,
            expected:    expectedOutput,
            expectError: false,
        },
        {
            name:        "error case",
            input:       invalidInput,
            expected:    ExpectedType{},
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockDeps := setupMocks()
            
            // Execute
            result, err := MyFunction(tt.input, mockDeps)
            
            // Assert
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Mock Dependencies

```go
type MockLogger struct {
    entries []LogEntry
}

type LogEntry struct {
    Level   string
    Message string
    Fields  []logging.Field
}

func (m *MockLogger) Info(msg string, fields ...logging.Field) {
    m.entries = append(m.entries, LogEntry{
        Level:   "INFO",
        Message: msg,
        Fields:  fields,
    })
}
```

### Integration Testing

```go
func TestEndToEndConversion(t *testing.T) {
    // Create temporary directories
    tempDir := t.TempDir()
    inputFile := filepath.Join(tempDir, "input.xml")
    outputFile := filepath.Join(tempDir, "output.csv")
    
    // Create test input
    testData := `<xml>test data</xml>`
    err := os.WriteFile(inputFile, []byte(testData), 0644)
    require.NoError(t, err)
    
    // Execute conversion
    container, err := container.NewContainer(config.GetGlobalConfig())
    require.NoError(t, err)
    
    parser, err := container.GetParser(factory.CAMT)
    require.NoError(t, err)
    
    err = parser.ConvertToCSV(inputFile, outputFile)
    require.NoError(t, err)
    
    // Verify output
    assert.FileExists(t, outputFile)
    
    // Verify content
    content, err := os.ReadFile(outputFile)
    require.NoError(t, err)
    assert.Contains(t, string(content), "expected,content")
}
```

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total
```

## Code Quality Standards

### Linting Configuration

**File: `.golangci.yml`**
```yaml
linters-settings:
  govet:
    check-shadowing: true
  golint:
    min-confidence: 0
  gocyclo:
    min-complexity: 15
  maligned:
    suggest-new: true
  dupl:
    threshold: 100
  goconst:
    min-len: 2
    min-occurrences: 2

linters:
  enable:
    - bodyclose
    - deadcode
    - depguard
    - dogsled
    - dupl
    - errcheck
    - gochecknoinits
    - goconst
    - gocyclo
    - gofmt
    - goimports
    - golint
    - gosec
    - gosimple
    - govet
    - ineffassign
    - interfacer
    - maligned
    - misspell
    - nakedret
    - scopelint
    - staticcheck
    - structcheck
    - stylecheck
    - typecheck
    - unconvert
    - unparam
    - unused
    - varcheck
    - whitespace

run:
  timeout: 5m
```

### Code Formatting

```bash
# Format all Go files
go fmt ./...

# Organize imports
goimports -w .

# Run all linters
golangci-lint run
```

### Documentation Standards

```go
// Package mypackage provides functionality for handling MyFormat files.
//
// This package implements the parser interface for MyFormat financial data,
// supporting both parsing and validation of input files.
package mypackage

// MyFunction performs a specific operation on the input data.
//
// It takes an input parameter and returns the processed result along with
// any error that occurred during processing.
//
// Parameters:
//   - input: The data to be processed
//   - config: Configuration options for processing
//
// Returns:
//   - ProcessedData: The result of processing
//   - error: Any error that occurred during processing
//
// Example:
//
//   result, err := MyFunction(inputData, config)
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Printf("Result: %v\n", result)
func MyFunction(input InputData, config Config) (ProcessedData, error) {
    // Implementation
}
```

## Debugging and Troubleshooting

### Debug Logging

```go
// Enable debug logging
logger := logging.NewLogrusAdapter("debug", "text")

// Add debug information
logger.Debug("Processing entry",
    logging.Field{Key: "entry_id", Value: entry.ID},
    logging.Field{Key: "amount", Value: entry.Amount},
    logging.Field{Key: "raw_data", Value: string(rawData)})
```

### Common Issues

#### 1. Parser Not Found

**Error**: `unknown parser type: myformat`

**Solution**: Ensure parser is registered in factory:
```go
case MyFormat:
    return myformatparser.NewAdapter(logger), nil
```

#### 2. Dependency Injection Issues

**Error**: `nil pointer dereference`

**Solution**: Ensure all dependencies are properly injected:
```go
func NewMyParser(logger logging.Logger) *MyParser {
    if logger == nil {
        logger = logging.GetLogger() // Fallback
    }
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}
```

#### 3. Test Failures

**Error**: Tests fail with mock dependencies

**Solution**: Ensure mocks implement all interface methods:
```go
// Verify interface compliance
var _ logging.Logger = (*MockLogger)(nil)
```

### Debugging Tools

```bash
# Run with race detection
go test -race ./...

# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Profile memory usage
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Debug with delve
dlv debug
```

## Performance Considerations

### Memory Optimization

```go
// Pre-allocate slices with known capacity
transactions := make([]models.Transaction, 0, expectedCount)

// Use strings.Builder for string concatenation
var builder strings.Builder
builder.Grow(estimatedSize)
```

### CPU Optimization

```go
// Use sync.Pool for frequently allocated objects
var transactionPool = sync.Pool{
    New: func() interface{} {
        return &models.Transaction{}
    },
}

func getTransaction() *models.Transaction {
    return transactionPool.Get().(*models.Transaction)
}

func putTransaction(tx *models.Transaction) {
    // Reset transaction
    *tx = models.Transaction{}
    transactionPool.Put(tx)
}
```

### Benchmarking

```go
func BenchmarkMyFunction(b *testing.B) {
    input := setupBenchmarkInput()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := MyFunction(input)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Contributing Guidelines

### Pull Request Process

1. **Fork and Clone**: Fork the repository and clone your fork
2. **Create Branch**: Create a feature branch from `main`
3. **Implement Changes**: Follow the coding standards and patterns
4. **Add Tests**: Ensure comprehensive test coverage
5. **Update Documentation**: Update relevant documentation
6. **Run Quality Checks**: Ensure all linters and tests pass
7. **Submit PR**: Create a pull request with clear description

### Commit Message Format

```
type(scope): description

Longer description if needed

Fixes #issue-number
```

**Types**: feat, fix, docs, style, refactor, test, chore

**Examples**:
```
feat(parser): add support for MyFormat files

Implements parser for MyFormat financial data with validation
and error handling following established patterns.

Fixes #123
```

### Code Review Checklist

- [ ] Follows established architecture patterns
- [ ] Uses dependency injection properly
- [ ] Implements proper error handling
- [ ] Includes comprehensive tests
- [ ] Updates documentation
- [ ] Passes all quality checks
- [ ] Maintains backward compatibility

This developer guide provides the foundation for contributing to the CAMT-CSV project while maintaining code quality and architectural consistency.