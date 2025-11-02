# Architecture Documentation

## Overview

CAMT-CSV follows a clean, layered architecture built on dependency injection principles. The system transforms various financial statement formats into standardized CSV files with intelligent categorization.

## High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Layer     │    │  Configuration  │    │   Logging       │
│   (cmd/)        │    │   (config/)     │    │  (logging/)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Dependency Container                         │
│                    (container/)                                 │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│    Parsers      │    │  Categorizer    │    │     Store       │
│   (parsers/)    │    │ (categorizer/)  │    │   (store/)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Core Models                                │
│                     (models/)                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Core Principles

### 1. Dependency Injection

All components receive their dependencies through constructors, eliminating global state and improving testability.

**Container Pattern:**
```go
type Container struct {
    Logger      logging.Logger
    Config      *config.Config
    Store       *store.CategoryStore
    Categorizer *categorizer.Categorizer
    Parsers     map[parser.ParserType]parser.FullParser
}
```

**Benefits:**
- Explicit dependencies
- Easy testing with mocks
- No global mutable state
- Runtime configuration flexibility

### 2. Interface Segregation

Parsers implement segregated interfaces based on their capabilities:

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

type LoggerConfigurable interface {
    SetLogger(logger logging.Logger)
}

type FullParser interface {
    Parser
    Validator
    CSVConverter
    LoggerConfigurable
}
```

**Benefits:**
- Clients depend only on needed interfaces
- Easy to implement new parsers
- Clear separation of concerns
- Flexible composition

### 3. BaseParser Foundation

All parsers embed `BaseParser` to eliminate code duplication:

```go
type BaseParser struct {
    logger logging.Logger
}

func (b *BaseParser) SetLogger(logger logging.Logger) {
    b.logger = logger
}

func (b *BaseParser) WriteToCSV(transactions []models.Transaction, csvFile string) error {
    return common.WriteTransactionsToCSV(transactions, csvFile)
}
```

**Benefits:**
- Consistent behavior across parsers
- Shared functionality (logging, CSV writing)
- Reduced code duplication
- Easier maintenance

## Component Architecture

### Parser Layer

**Structure:**
```
internal/
├── parser/
│   ├── parser.go          # Interface definitions
│   ├── base.go           # BaseParser implementation
│   └── constitution.go   # Constitution loading
├── camtparser/           # CAMT.053 XML parser
├── pdfparser/           # PDF statement parser
├── revolutparser/       # Revolut CSV parser
├── revolutinvestmentparser/ # Revolut investment parser
├── selmaparser/         # Selma investment parser
└── debitparser/         # Generic debit CSV parser
```

**Parser Implementation Pattern:**
```go
type MyParser struct {
    parser.BaseParser
    // parser-specific fields
}

func NewMyParser(logger logging.Logger) *MyParser {
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}

func (p *MyParser) Parse(r io.Reader) ([]models.Transaction, error) {
    p.GetLogger().Info("Starting parse operation")
    // implementation
}
```

### Categorization Layer

**Strategy Pattern Implementation:**
```go
type CategorizationStrategy interface {
    Categorize(ctx context.Context, tx Transaction) (Category, bool, error)
    Name() string
}

type Categorizer struct {
    strategies []CategorizationStrategy
    store      *store.CategoryStore
    logger     logging.Logger
}
```

**Three-Tier Approach:**
1. **DirectMappingStrategy**: Exact name matches from YAML files
2. **KeywordStrategy**: Pattern matching from categories.yaml
3. **AIStrategy**: Gemini API fallback with auto-learning

### Data Layer

**Transaction Model Decomposition:**
```go
// Core transaction data
type TransactionCore struct {
    ID          string
    Date        time.Time
    ValueDate   time.Time
    Amount      Money
    Description string
    Status      string
    Reference   string
}

// Adds party information
type TransactionWithParties struct {
    TransactionCore
    Payer     Party
    Payee     Party
    Direction TransactionDirection
}

// Adds categorization
type CategorizedTransaction struct {
    TransactionWithParties
    Category string
    Type     string
    Fund     string
}
```

**Builder Pattern:**
```go
tx, err := NewTransactionBuilder().
    WithDate("2025-01-15").
    WithAmount(decimal.NewFromFloat(100.50), "CHF").
    WithPayer("John Doe", "CH1234567890").
    WithPayee("Acme Corp", "CH0987654321").
    AsDebit().
    Build()
```

## Error Handling Architecture

### Custom Error Types

**Structured Error Hierarchy:**
```go
// Base parsing error with context
type ParseError struct {
    Parser string
    Field  string
    Value  string
    Err    error
}

// Format validation failures
type ValidationError struct {
    FilePath string
    Reason   string
}

// Data extraction failures
type DataExtractionError struct {
    FilePath string
    Field    string
    RawData  string
    Reason   string
}
```

### Error Handling Patterns

**Pattern 1: Unrecoverable Errors (Return)**
```go
if err := xml.Unmarshal(data, &document); err != nil {
    return nil, &parsererror.ParseError{
        Parser: "CAMT",
        Field:  "document",
        Err:    err,
    }
}
```

**Pattern 2: Recoverable Errors (Log and Continue)**
```go
amount, err := decimal.NewFromString(entry.Amt.Value)
if err != nil {
    p.logger.Warn("Failed to parse amount, using zero",
        logging.Field{Key: "value", Value: entry.Amt.Value},
        logging.Field{Key: "error", Value: err})
    amount = decimal.Zero
}
```

## Logging Architecture

### Framework-Agnostic Design

**Logger Interface:**
```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithError(err error) Logger
    WithField(key string, value interface{}) Logger
    WithFields(fields ...Field) Logger
}
```

**Structured Logging:**
```go
logger.Info("Processing transaction",
    logging.Field{Key: "file", Value: filename},
    logging.Field{Key: "count", Value: len(transactions)})
```

**Dependency Injection:**
- All components receive logger through constructor
- BaseParser provides logger to all parsers
- Mock loggers for testing

## Configuration Architecture

### Hierarchical Configuration

**Priority Order (highest to lowest):**
1. CLI flags (`--log-level debug`)
2. Environment variables (`CAMT_LOG_LEVEL=debug`)
3. Config file (`~/.camt-csv/config.yaml`)
4. Default values

**Configuration Structure:**
```yaml
log:
  level: "info"
  format: "text"
csv:
  delimiter: ","
  include_headers: true
ai:
  enabled: true
  model: "gemini-2.0-flash"
```

## Testing Architecture

### Dependency Injection for Testing

**Mock Implementations:**
```go
type MockLogger struct {
    Entries []LogEntry
}

type MockAIClient struct {
    CategorizeFunc func(context.Context, models.Transaction) (models.Transaction, error)
}

type MockCategoryStore struct {
    Categories       []models.CategoryConfig
    CreditorMappings map[string]string
    DebtorMappings   map[string]string
}
```

**Test Structure:**
```go
func TestCategorizer_Categorize(t *testing.T) {
    // Setup
    mockStore := &MockCategoryStore{...}
    mockLogger := &MockLogger{}
    
    cat := NewCategorizer(mockStore, nil, mockLogger)
    
    // Execute & Assert
    result, err := cat.Categorize(context.Background(), transaction)
    assert.NoError(t, err)
    assert.Equal(t, expectedCategory, result.Name)
}
```

## Performance Architecture

### Optimization Strategies

**String Operations:**
```go
// Pre-allocate builder capacity
var builder strings.Builder
builder.Grow(len(input))
```

**Lazy Initialization:**
```go
type Categorizer struct {
    aiClient     AIClient
    aiClientOnce sync.Once
    aiFactory    func() AIClient
}

func (c *Categorizer) getAIClient() AIClient {
    c.aiClientOnce.Do(func() {
        if c.aiClient == nil && c.aiFactory != nil {
            c.aiClient = c.aiFactory()
        }
    })
    return c.aiClient
}
```

**Pre-allocation:**
```go
// Pre-allocate slices with known capacity
transactions := make([]models.Transaction, 0, len(entries))
```

## Security Architecture

### Input Validation

- All file paths validated for directory traversal
- XML/CSV content validated before processing
- Amount values validated for reasonable ranges
- Date formats validated before parsing

### Error Message Sanitization

- No sensitive data in error messages
- File paths relativized in logs
- API keys never logged
- Transaction details redacted in non-debug logs

### File Permissions

- Config files: 0600 (owner read/write only)
- Directories: 0750 (owner full, group read/execute)
- Output files: 0644 (owner read/write, others read)

## Migration and Compatibility

### Backward Compatibility Strategy

**Deprecated Code Marking:**
```go
// Deprecated: Use NewCategorizer with dependency injection instead.
// This function will be removed in v2.0.0.
func GetDefaultCategorizer() *Categorizer {
    // Provide backward compatible implementation
}
```

**Adapter Pattern:**
```go
type LegacyTransactionAdapter struct {
    tx Transaction
}

func (a *LegacyTransactionAdapter) GetAmountAsFloat() float64 {
    f, _ := a.tx.Amount.Amount.Float64()
    return f
}
```

### Migration Path

1. **Phase 1**: Introduce new interfaces alongside existing code
2. **Phase 2**: Add deprecation warnings to old APIs
3. **Phase 3**: Migrate internal usage to new patterns
4. **Phase 4**: Remove deprecated code in major version bump

## Extension Points

### Adding New Parsers

1. Create package: `internal/<format>parser/`
2. Embed BaseParser: `parser.BaseParser`
3. Implement interfaces: `parser.Parser` (minimum)
4. Use dependency injection: Accept logger in constructor
5. Follow error handling patterns: Use custom error types
6. Add comprehensive tests: Mock dependencies

### Adding New Categorization Strategies

1. Implement `CategorizationStrategy` interface
2. Add to strategy list in `Categorizer` constructor
3. Ensure proper priority ordering
4. Add comprehensive tests with mock dependencies

### Adding New Configuration Options

1. Add to config struct in `internal/config/`
2. Update Viper configuration loading
3. Add environment variable mapping
4. Update CLI flags if needed
5. Document in user guide

This architecture provides a solid foundation for maintainable, testable, and extensible financial data processing while ensuring reliability and performance.