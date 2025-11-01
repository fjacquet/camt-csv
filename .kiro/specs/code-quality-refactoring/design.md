# Design Document

## Overview

This design document outlines the architectural changes needed to refactor the camt-csv codebase, addressing critical code quality issues while maintaining backward compatibility. The refactoring will be implemented in phases to minimize risk and allow for incremental testing.

## Architecture

### High-Level Architecture Changes

The refactoring transforms the current architecture from a global-state-based design to a dependency-injection-based design:

**Current Architecture:**
```
Global Singletons → Business Logic → File I/O
     ↓
  Tight Coupling
```

**Target Architecture:**
```
Main → Dependency Container → Services (injected dependencies) → Domain Logic
                                   ↓
                          Interfaces (abstraction layer)
```

### Core Principles

1. **Dependency Injection**: All dependencies passed through constructors
2. **Interface Segregation**: Small, focused interfaces
3. **Single Responsibility**: Each component has one clear purpose
4. **Composition over Inheritance**: Use embedding and composition
5. **Testability**: All components easily mockable

## Components and Interfaces

### 1. Logging Abstraction Layer

**Package**: `internal/logging`

**Interface Design:**

```go
// Logger provides structured logging capabilities
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithError(err error) Logger
    WithField(key string, value interface{}) Logger
    WithFields(fields ...Field) Logger
}

// Field represents a key-value pair for structured logging
type Field struct {
    Key   string
    Value interface{}
}

// LogrusAdapter adapts logrus.Logger to our Logger interface
type LogrusAdapter struct {
    logger *logrus.Logger
}
```

**Rationale**: Decouples business logic from logging framework, enables testing with mock loggers, allows future framework changes.

### 2. Parser Interface Segregation

**Package**: `internal/parser`

**Interface Design:**

```go
// Parser defines core parsing capability
type Parser interface {
    Parse(r io.Reader) ([]models.Transaction, error)
}

// Validator defines format validation capability
type Validator interface {
    ValidateFormat(filePath string) (bool, error)
}

// CSVConverter defines CSV conversion capability
type CSVConverter interface {
    ConvertToCSV(inputFile, outputFile string) error
}

// LoggerConfigurable allows logger injection
type LoggerConfigurable interface {
    SetLogger(logger logging.Logger)
}

// FullParser combines all parser capabilities
type FullParser interface {
    Parser
    Validator
    CSVConverter
    LoggerConfigurable
}
```


**Base Parser Implementation:**

```go
// BaseParser provides common functionality for all parsers
type BaseParser struct {
    logger logging.Logger
}

func NewBaseParser(logger logging.Logger) BaseParser {
    return BaseParser{logger: logger}
}

func (b *BaseParser) SetLogger(logger logging.Logger) {
    b.logger = logger
}

func (b *BaseParser) GetLogger() logging.Logger {
    return b.logger
}

// Common CSV writing functionality
func (b *BaseParser) WriteToCSV(transactions []models.Transaction, csvFile string) error {
    return common.WriteTransactionsToCSV(transactions, csvFile)
}
```

**Rationale**: Eliminates code duplication, provides consistent behavior, simplifies parser implementation.

### 3. Transaction Model Decomposition

**Package**: `internal/models`

**New Structure Design:**

```go
// Money represents a monetary value with currency
type Money struct {
    Amount   decimal.Decimal
    Currency string
}

// Party represents a transaction party (payer or payee)
type Party struct {
    Name string
    IBAN string
}

// TransactionCore contains essential transaction data
type TransactionCore struct {
    ID            string
    Date          time.Time
    ValueDate     time.Time
    Amount        Money
    Description   string
    Status        string
    Reference     string
}

// TransactionWithParties adds party information
type TransactionWithParties struct {
    TransactionCore
    Payer         Party
    Payee         Party
    Direction     TransactionDirection // DEBIT or CREDIT
}


// CategorizedTransaction adds categorization data
type CategorizedTransaction struct {
    TransactionWithParties
    Category      string
    Type          string
    Fund          string
}

// Transaction maintains backward compatibility
// This will be the main type used throughout the codebase
type Transaction struct {
    CategorizedTransaction
    
    // Additional fields for specific formats
    BookkeepingNumber string
    RemittanceInfo    string
    PartyIBAN         string
    Investment        string
    NumberOfShares    int
    Fees              Money
    EntryReference    string
    AccountServicer   string
    BankTxCode        string
    OriginalAmount    Money
    ExchangeRate      decimal.Decimal
    
    // Tax-related fields
    AmountExclTax Money
    AmountTax     Money
    TaxRate       decimal.Decimal
}

// TransactionDirection represents debit or credit
type TransactionDirection int

const (
    DirectionUnknown TransactionDirection = iota
    DirectionDebit
    DirectionCredit
)
```

**Migration Strategy**: 
1. Introduce new types alongside existing Transaction
2. Add adapter methods to convert between old and new formats
3. Gradually migrate parsers to use new structure
4. Deprecate old direct field access
5. Remove deprecated code after migration complete

**Rationale**: Separates concerns, improves clarity, enables better testing, maintains flexibility.

### 4. Categorization Strategy Pattern

**Package**: `internal/categorizer`

**Strategy Interface Design:**


```go
// CategorizationStrategy defines a method for categorizing transactions
type CategorizationStrategy interface {
    Categorize(ctx context.Context, tx Transaction) (Category, bool, error)
    Name() string // For logging/debugging
}

// DirectMappingStrategy uses exact name matching
type DirectMappingStrategy struct {
    mappings map[string]string
    logger   logging.Logger
}

// KeywordStrategy uses keyword pattern matching
type KeywordStrategy struct {
    patterns map[string]string
    logger   logging.Logger
}

// AIStrategy uses AI service for categorization
type AIStrategy struct {
    client AIClient
    logger logging.Logger
}

// Categorizer orchestrates multiple strategies
type Categorizer struct {
    strategies []CategorizationStrategy
    store      *store.CategoryStore
    logger     logging.Logger
    mu         sync.RWMutex
}

func NewCategorizer(
    store *store.CategoryStore,
    aiClient AIClient,
    logger logging.Logger,
) *Categorizer {
    c := &Categorizer{
        store:  store,
        logger: logger,
    }
    
    // Initialize strategies in priority order
    c.strategies = []CategorizationStrategy{
        NewDirectMappingStrategy(store, logger),
        NewKeywordStrategy(store, logger),
        NewAIStrategy(aiClient, logger),
    }
    
    return c
}

func (c *Categorizer) Categorize(ctx context.Context, tx Transaction) (Category, error) {
    for _, strategy := range c.strategies {
        category, found, err := strategy.Categorize(ctx, tx)
        if err != nil {
            c.logger.Warn("Strategy failed", 
                logging.Field{Key: "strategy", Value: strategy.Name()},
                logging.Field{Key: "error", Value: err})
            continue
        }
        if found {
            c.logger.Debug("Transaction categorized",
                logging.Field{Key: "strategy", Value: strategy.Name()},
                logging.Field{Key: "category", Value: category.Name})
            return category, nil
        }
    }
    
    return UncategorizedCategory, nil
}
```

**Rationale**: Separates categorization logic, enables independent testing, allows easy addition of new strategies.


### 5. Transaction Builder Pattern

**Package**: `internal/models`

**Builder Design:**

```go
// TransactionBuilder provides fluent API for transaction construction
type TransactionBuilder struct {
    tx  Transaction
    err error
}

func NewTransactionBuilder() *TransactionBuilder {
    return &TransactionBuilder{
        tx: Transaction{
            TransactionCore: TransactionCore{
                ID:     uuid.New().String(),
                Amount: Money{Amount: decimal.Zero},
            },
        },
    }
}

func (b *TransactionBuilder) WithDate(dateStr string) *TransactionBuilder {
    if b.err != nil {
        return b
    }
    date, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        b.err = fmt.Errorf("invalid date: %w", err)
        return b
    }
    b.tx.Date = date
    return b
}

func (b *TransactionBuilder) WithAmount(amount decimal.Decimal, currency string) *TransactionBuilder {
    b.tx.Amount = Money{Amount: amount, Currency: currency}
    return b
}

func (b *TransactionBuilder) WithPayer(name, iban string) *TransactionBuilder {
    b.tx.Payer = Party{Name: name, IBAN: iban}
    return b
}

func (b *TransactionBuilder) WithPayee(name, iban string) *TransactionBuilder {
    b.tx.Payee = Party{Name: name, IBAN: iban}
    return b
}

func (b *TransactionBuilder) AsDebit() *TransactionBuilder {
    b.tx.Direction = DirectionDebit
    return b
}

func (b *TransactionBuilder) AsCredit() *TransactionBuilder {
    b.tx.Direction = DirectionCredit
    return b
}

func (b *TransactionBuilder) Build() (Transaction, error) {
    if b.err != nil {
        return Transaction{}, b.err
    }
    
    // Validate required fields
    if b.tx.Date.IsZero() {
        return Transaction{}, errors.New("date is required")
    }
    
    // Populate derived fields
    b.populateDerivedFields()
    
    return b.tx, nil
}

func (b *TransactionBuilder) populateDerivedFields() {
    // Set party name based on direction
    // Calculate debit/credit amounts
    // Set other derived fields
}
```

**Usage Example:**

```go
tx, err := NewTransactionBuilder().
    WithDate("2025-01-15").
    WithAmount(decimal.NewFromFloat(100.50), "CHF").
    WithPayer("John Doe", "CH1234567890").
    WithPayee("Acme Corp", "CH0987654321").
    AsDebit().
    Build()
```

**Rationale**: Improves readability, enforces validation, simplifies complex construction.


### 6. Dependency Container

**Package**: `internal/container`

**Container Design:**

```go
// Container holds all application dependencies
type Container struct {
    Logger      logging.Logger
    Config      *config.Config
    Store       *store.CategoryStore
    AIClient    categorizer.AIClient
    Categorizer *categorizer.Categorizer
    Parsers     map[parser.ParserType]parser.FullParser
}

// NewContainer creates and wires all dependencies
func NewContainer(cfg *config.Config) (*Container, error) {
    // Create logger
    logger := logging.NewLogrusAdapter(cfg.Log.Level, cfg.Log.Format)
    
    // Create store
    categoryStore := store.NewCategoryStore(
        cfg.Categories.File,
        cfg.Categories.CreditorsFile,
        cfg.Categories.DebitorsFile,
    )
    
    // Create AI client (if enabled)
    var aiClient categorizer.AIClient
    if cfg.AI.Enabled {
        aiClient = categorizer.NewGeminiClient(cfg.AI.APIKey, logger)
    }
    
    // Create categorizer
    cat := categorizer.NewCategorizer(categoryStore, aiClient, logger)
    
    // Create parsers
    parsers := make(map[parser.ParserType]parser.FullParser)
    parsers[parser.CAMT] = camtparser.NewParser(logger)
    parsers[parser.PDF] = pdfparser.NewParser(logger)
    parsers[parser.Revolut] = revolutparser.NewParser(logger)
    // ... other parsers
    
    return &Container{
        Logger:      logger,
        Config:      cfg,
        Store:       categoryStore,
        AIClient:    aiClient,
        Categorizer: cat,
        Parsers:     parsers,
    }, nil
}

// GetParser returns a parser for the given type
func (c *Container) GetParser(pt parser.ParserType) (parser.FullParser, error) {
    p, ok := c.Parsers[pt]
    if !ok {
        return nil, fmt.Errorf("unknown parser type: %s", pt)
    }
    return p, nil
}
```

**Rationale**: Centralizes dependency creation, simplifies testing, makes dependencies explicit.

## Data Models

### Constants Definition

**Package**: `internal/models/constants.go`

```go
// Transaction types
const (
    TransactionTypeDebit  = "DBIT"
    TransactionTypeCredit = "CRDT"
)

// Transaction statuses
const (
    StatusCompleted = "COMPLETED"
    StatusPending   = "PENDING"
    StatusFailed    = "FAILED"
)

// Categories
const (
    CategoryUncategorized = "Uncategorized"
    CategorySalary        = "Salaire"
    CategoryGroceries     = "Alimentation"
    CategoryTransport     = "Transports Publics"
    // ... other categories
)

// Bank transaction codes
const (
    BankCodeCashWithdrawal = "CWDL"
    BankCodePOS            = "POSD"
    BankCodeCreditCard     = "CCRD"
    // ... other codes
)

// File permissions
const (
    PermissionConfigFile = 0600
    PermissionDirectory  = 0750
)
```


### Error Handling

**Package**: `internal/parsererror`

**Error Types:**

```go
// ParseError represents an error during parsing
type ParseError struct {
    Parser string
    Field  string
    Value  string
    Err    error
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("%s: failed to parse %s='%s': %v", 
        e.Parser, e.Field, e.Value, e.Err)
}

func (e *ParseError) Unwrap() error {
    return e.Err
}

// ValidationError represents a validation failure
type ValidationError struct {
    FilePath string
    Reason   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.FilePath, e.Reason)
}

// CategorizationError represents a categorization failure
type CategorizationError struct {
    Transaction string
    Strategy    string
    Err         error
}

func (e *CategorizationError) Error() string {
    return fmt.Sprintf("categorization failed for %s using %s: %v",
        e.Transaction, e.Strategy, e.Err)
}

func (e *CategorizationError) Unwrap() error {
    return e.Err
}
```

**Error Handling Guidelines:**

1. **Return errors for recoverable issues**: Let callers decide how to handle
2. **Log warnings for degraded functionality**: Continue processing with reduced capability
3. **Never log and return**: Choose one approach per error
4. **Wrap errors with context**: Use fmt.Errorf with %w
5. **Use custom error types**: For domain-specific errors that need special handling

## Testing Strategy

### Unit Testing Approach

**Mock Implementations:**

```go
// MockLogger for testing
type MockLogger struct {
    Entries []LogEntry
}

type LogEntry struct {
    Level   string
    Message string
    Fields  []logging.Field
}

func (m *MockLogger) Info(msg string, fields ...logging.Field) {
    m.Entries = append(m.Entries, LogEntry{
        Level:   "INFO",
        Message: msg,
        Fields:  fields,
    })
}

// MockAIClient for testing
type MockAIClient struct {
    CategorizeFunc func(context.Context, models.Transaction) (models.Transaction, error)
}

func (m *MockAIClient) Categorize(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
    if m.CategorizeFunc != nil {
        return m.CategorizeFunc(ctx, tx)
    }
    tx.Category = "MockCategory"
    return tx, nil
}

// MockCategoryStore for testing
type MockCategoryStore struct {
    Categories       []models.CategoryConfig
    CreditorMappings map[string]string
    DebitorMappings  map[string]string
}
```

**Test Structure:**

```go
func TestCategorizer_Categorize(t *testing.T) {
    tests := []struct {
        name           string
        transaction    Transaction
        mappings       map[string]string
        expectedCat    string
        expectedErr    bool
    }{
        {
            name: "direct mapping found",
            transaction: Transaction{
                Payer: Party{Name: "COOP"},
                Direction: DirectionDebit,
            },
            mappings: map[string]string{
                "coop": "Alimentation",
            },
            expectedCat: "Alimentation",
            expectedErr: false,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockStore := &MockCategoryStore{
                DebitorMappings: tt.mappings,
            }
            mockLogger := &MockLogger{}
            
            cat := NewCategorizer(mockStore, nil, mockLogger)
            
            // Execute
            result, err := cat.Categorize(context.Background(), tt.transaction)
            
            // Assert
            if tt.expectedErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedCat, result.Name)
            }
        })
    }
}
```


## Error Handling

### Error Handling Patterns

**Pattern 1: Unrecoverable Errors (Return)**

```go
func (p *ISO20022Parser) Parse(r io.Reader) ([]models.Transaction, error) {
    data, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("failed to read input: %w", err)
    }
    
    var document models.ISO20022Document
    if err := xml.Unmarshal(data, &document); err != nil {
        return nil, &parsererror.ParseError{
            Parser: "ISO20022",
            Field:  "document",
            Err:    err,
        }
    }
    
    return p.extractTransactions(document)
}
```

**Pattern 2: Recoverable Errors (Log and Continue)**

```go
func (p *ISO20022Parser) entryToTransaction(entry *models.Entry) models.Transaction {
    amount, err := decimal.NewFromString(entry.Amt.Value)
    if err != nil {
        p.logger.Warn("Failed to parse amount, using zero",
            logging.Field{Key: "value", Value: entry.Amt.Value},
            logging.Field{Key: "error", Value: err})
        amount = decimal.Zero
    }
    
    // Continue processing with default value
    return models.Transaction{Amount: amount}
}
```

**Pattern 3: Validation Errors (Custom Error Type)**

```go
func (p *ISO20022Parser) ValidateFormat(filePath string) (bool, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return false, fmt.Errorf("failed to read file: %w", err)
    }
    
    var document models.ISO20022Document
    if err := xml.Unmarshal(data, &document); err != nil {
        return false, &parsererror.ValidationError{
            FilePath: filePath,
            Reason:   "not a valid CAMT.053 XML document",
        }
    }
    
    if len(document.BkToCstmrStmt.Stmt) == 0 {
        return false, &parsererror.ValidationError{
            FilePath: filePath,
            Reason:   "no statements found in document",
        }
    }
    
    return true, nil
}
```

## Performance Optimizations

### 1. String Operations Optimization

**Before:**
```go
func (c *Categorizer) categorizeByMapping(tx Transaction) (Category, bool) {
    partyNameLower := strings.ToLower(tx.PartyName)
    // Multiple string operations
}
```

**After:**
```go
func (c *Categorizer) categorizeByMapping(tx Transaction) (Category, bool) {
    // Pre-allocate builder with known capacity
    var builder strings.Builder
    builder.Grow(len(tx.PartyName))
    builder.WriteString(tx.PartyName)
    partyNameLower := strings.ToLower(builder.String())
    
    // Use the normalized string
}
```

### 2. Lazy Initialization

**Before:**
```go
type Categorizer struct {
    aiClient AIClient
}

func NewCategorizer(aiClient AIClient) *Categorizer {
    return &Categorizer{aiClient: aiClient}
}
```

**After:**
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

### 3. Pre-allocation

**Before:**
```go
var transactions []models.Transaction
for _, entry := range entries {
    transactions = append(transactions, convertEntry(entry))
}
```

**After:**
```go
transactions := make([]models.Transaction, 0, len(entries))
for _, entry := range entries {
    transactions = append(transactions, convertEntry(entry))
}
```


## Migration Strategy

### Phase 1: Foundation (Requirements 1, 6, 7)

**Goal**: Establish core abstractions without breaking existing code

**Steps**:
1. Create logging abstraction layer (`internal/logging`)
2. Create LogrusAdapter implementation
3. Define constants in `internal/models/constants.go`
4. Update imports to use new logging interface (backward compatible)
5. Replace magic strings with constants throughout codebase

**Success Criteria**:
- All tests pass
- No functional changes
- Logging abstraction in place
- Constants defined and used

### Phase 2: Parser Refactoring (Requirements 3, 5, 10)

**Goal**: Eliminate parser code duplication and improve interfaces

**Steps**:
1. Define segregated parser interfaces
2. Create BaseParser with common functionality
3. Refactor each parser to embed BaseParser
4. Remove duplicate CSV writing code
5. Implement PDFExtractor interface for PDF parser
6. Remove TEST_ENV checks from production code

**Success Criteria**:
- All parser tests pass
- Code duplication eliminated
- Interfaces properly segregated
- Test environment detection removed

### Phase 3: Dependency Injection (Requirements 1, 2)

**Goal**: Eliminate global state and standardize error handling

**Steps**:
1. Create dependency container
2. Refactor Categorizer to accept dependencies via constructor
3. Remove global singleton patterns
4. Update CLI commands to use container
5. Standardize error handling patterns
6. Define custom error types

**Success Criteria**:
- No global mutable state
- All dependencies injected
- Consistent error handling
- All tests pass with dependency injection

### Phase 4: Model Decomposition (Requirements 4, 9, 12)

**Goal**: Simplify Transaction model and improve date handling

**Steps**:
1. Define new decomposed transaction types
2. Create TransactionBuilder
3. Add adapter methods for backward compatibility
4. Migrate parsers to use time.Time for dates
5. Update CSV marshaling/unmarshaling
6. Gradually migrate code to use new types

**Success Criteria**:
- New transaction types defined
- Builder pattern implemented
- Date handling uses time.Time
- Backward compatibility maintained
- All tests pass

### Phase 5: Strategy Pattern (Requirement 11)

**Goal**: Refactor categorization to use Strategy pattern

**Steps**:
1. Define CategorizationStrategy interface
2. Implement DirectMappingStrategy
3. Implement KeywordStrategy
4. Implement AIStrategy
5. Refactor Categorizer to use strategies
6. Update tests to use new structure

**Success Criteria**:
- Strategy pattern implemented
- All categorization tests pass
- Strategies independently testable
- Same categorization results as before

### Phase 6: Optimization and Cleanup (Requirements 8, 13, 14, 15)

**Goal**: Optimize performance and improve test coverage

**Steps**:
1. Standardize naming conventions (debitor → debtor)
2. Implement performance optimizations
3. Add comprehensive unit tests
4. Achieve 80%+ code coverage
5. Remove deprecated code
6. Update documentation

**Success Criteria**:
- Naming conventions standardized
- Performance improvements measurable
- 80%+ code coverage achieved
- All deprecated code removed
- Documentation updated

## Backward Compatibility

### Compatibility Guarantees

1. **CLI Interface**: All existing commands and flags remain unchanged
2. **CSV Output**: Identical output format for same input files
3. **Configuration**: Existing config files continue to work
4. **File Formats**: All supported input formats remain supported

### Deprecation Strategy

**Deprecated Code Marking:**

```go
// Deprecated: Use NewCategorizer with dependency injection instead.
// This function will be removed in v2.0.0.
func GetDefaultCategorizer() *Categorizer {
    // Provide backward compatible implementation
}
```

**Migration Guide:**

```go
// Old way (deprecated)
cat := categorizer.GetDefaultCategorizer()

// New way (recommended)
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}
cat := container.Categorizer
```

### Adapter Pattern for Compatibility

```go
// LegacyTransactionAdapter converts new Transaction to legacy format
type LegacyTransactionAdapter struct {
    tx Transaction
}

func (a *LegacyTransactionAdapter) GetAmountAsFloat() float64 {
    f, _ := a.tx.Amount.Amount.Float64()
    return f
}

func (a *LegacyTransactionAdapter) GetPayee() string {
    return a.tx.Payee.Name
}
```


## Design Decisions and Rationales

### Decision 1: Dependency Injection over Service Locator

**Rationale**: 
- Makes dependencies explicit and visible
- Easier to test with mock implementations
- Prevents hidden coupling
- Aligns with Go best practices

**Trade-offs**:
- More verbose constructor signatures
- Requires dependency container setup
- Benefits: Better testability, clearer dependencies

### Decision 2: Interface Segregation for Parsers

**Rationale**:
- Not all parsers need all capabilities
- Smaller interfaces are easier to implement
- Enables composition of capabilities
- Follows Interface Segregation Principle

**Trade-offs**:
- More interfaces to manage
- Slightly more complex type system
- Benefits: Flexibility, clearer contracts

### Decision 3: Strategy Pattern for Categorization

**Rationale**:
- Separates categorization algorithms
- Each strategy independently testable
- Easy to add new strategies
- Clear priority ordering

**Trade-offs**:
- More types and files
- Slightly more complex initialization
- Benefits: Modularity, testability, extensibility

### Decision 4: Builder Pattern for Transactions

**Rationale**:
- Simplifies complex object construction
- Provides validation at build time
- Improves code readability
- Enables fluent API

**Trade-offs**:
- Additional builder type to maintain
- More code for simple cases
- Benefits: Readability, validation, flexibility

### Decision 5: Logging Abstraction

**Rationale**:
- Decouples from specific logging framework
- Enables testing with mock loggers
- Allows future framework changes
- Standard practice in Go

**Trade-offs**:
- Additional abstraction layer
- Slight performance overhead
- Benefits: Testability, flexibility, decoupling

### Decision 6: Phased Migration Approach

**Rationale**:
- Reduces risk of breaking changes
- Allows incremental testing
- Maintains backward compatibility
- Enables gradual team adoption

**Trade-offs**:
- Longer overall timeline
- Temporary code duplication
- Benefits: Safety, stability, team confidence

## Security Considerations

### 1. File Permissions

All file operations use appropriate permissions:
- Config files: 0600 (owner read/write only)
- Directories: 0750 (owner full, group read/execute)
- Output files: 0644 (owner read/write, others read)

### 2. Input Validation

All external inputs validated:
- File paths checked for directory traversal
- XML/CSV content validated before processing
- Amount values validated for reasonable ranges
- Date formats validated before parsing

### 3. Error Messages

Error messages sanitized:
- No sensitive data in error messages
- File paths relativized in logs
- API keys never logged
- Transaction details redacted in non-debug logs

### 4. Dependency Management

Dependencies kept secure:
- Regular dependency updates
- Security scanning in CI/CD
- Minimal dependency footprint
- Pinned versions in go.mod

## Documentation Updates

### Code Documentation

**Package Documentation:**
```go
// Package categorizer provides transaction categorization using multiple strategies.
//
// The categorizer uses a priority-based approach:
//   1. Direct mapping from YAML configuration
//   2. Keyword-based pattern matching
//   3. AI-based categorization (optional)
//
// Example usage:
//
//   store := store.NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml")
//   aiClient := NewGeminiClient(apiKey, logger)
//   cat := NewCategorizer(store, aiClient, logger)
//
//   category, err := cat.Categorize(ctx, transaction)
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Printf("Category: %s\n", category.Name)
package categorizer
```

**Function Documentation:**
```go
// Categorize attempts to categorize a transaction using configured strategies.
//
// The categorization process tries each strategy in priority order:
//   1. DirectMappingStrategy - exact name matches
//   2. KeywordStrategy - pattern matching
//   3. AIStrategy - AI-based categorization
//
// If all strategies fail, returns an "Uncategorized" category.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - tx: Transaction to categorize
//
// Returns:
//   - Category: The assigned category
//   - error: Any error encountered during categorization
//
// Example:
//
//   ctx := context.Background()
//   category, err := categorizer.Categorize(ctx, transaction)
//   if err != nil {
//       return fmt.Errorf("categorization failed: %w", err)
//   }
func (c *Categorizer) Categorize(ctx context.Context, tx Transaction) (Category, error)
```

### User Documentation Updates

**Migration Guide** (`docs/migration-guide.md`):
- How to update code using deprecated APIs
- Examples of old vs new patterns
- Timeline for deprecation removals
- Breaking changes (if any)

**Architecture Documentation** (`docs/architecture.md`):
- Updated architecture diagrams
- Dependency injection patterns
- Strategy pattern usage
- Testing approaches

**Developer Guide** (`docs/developer-guide.md`):
- How to add new parsers
- How to add new categorization strategies
- Testing best practices
- Code organization principles

## Success Metrics

### Code Quality Metrics

1. **Test Coverage**: Achieve 80%+ overall, 100% for critical paths
2. **Cyclomatic Complexity**: Reduce average complexity by 30%
3. **Code Duplication**: Eliminate 90%+ of duplicated code
4. **Dependency Count**: Reduce coupling between packages

### Performance Metrics

1. **Memory Allocations**: Reduce by 20% in hot paths
2. **Processing Time**: Maintain or improve current performance
3. **Startup Time**: No significant increase (<10%)

### Maintainability Metrics

1. **Lines of Code**: May increase slightly due to better structure
2. **Number of Files**: Increase due to better separation
3. **Average File Size**: Decrease due to focused responsibilities
4. **Documentation Coverage**: 100% of public APIs documented

### Team Metrics

1. **Time to Add New Parser**: Reduce by 40%
2. **Time to Add New Strategy**: Reduce by 50%
3. **Bug Fix Time**: Reduce by 30% due to better testability
4. **Onboarding Time**: Reduce by 25% due to clearer structure
