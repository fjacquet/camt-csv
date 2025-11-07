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
// This value object encapsulates amount and currency to prevent
// mixing different currencies and provides type safety for financial calculations
type Money struct {
    Amount   decimal.Decimal
    Currency string
}

// NewMoney creates a new Money instance with validation
func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
    if currency == "" {
        return Money{}, errors.New("currency cannot be empty")
    }
    return Money{Amount: amount, Currency: currency}, nil
}

// Add safely adds two Money values of the same currency
func (m Money) Add(other Money) (Money, error) {
    if m.Currency != other.Currency {
        return Money{}, fmt.Errorf("cannot add different currencies: %s and %s", m.Currency, other.Currency)
    }
    return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}

// Party represents a transaction party (payer or payee)
// This value object encapsulates party information and provides validation
type Party struct {
    Name string
    IBAN string
}

// NewParty creates a new Party instance with validation
func NewParty(name, iban string) (Party, error) {
    if name == "" {
        return Party{}, errors.New("party name cannot be empty")
    }
    // IBAN validation could be added here
    return Party{Name: name, IBAN: iban}, nil
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

**Builder Validation Strategy:**

The TransactionBuilder implements comprehensive validation to ensure data integrity:

```go
func (b *TransactionBuilder) Build() (Transaction, error) {
    if b.err != nil {
        return Transaction{}, b.err
    }
    
    // Required field validation
    if b.tx.Date.IsZero() {
        return Transaction{}, &parsererror.ValidationError{
            Field:  "Date",
            Reason: "transaction date is required",
        }
    }
    
    if b.tx.Amount.Amount.IsZero() && b.tx.Direction != DirectionUnknown {
        return Transaction{}, &parsererror.ValidationError{
            Field:  "Amount",
            Reason: "non-zero amount required for debit/credit transactions",
        }
    }
    
    // Business rule validation
    if b.tx.Direction == DirectionDebit && b.tx.Amount.Amount.IsPositive() {
        // Automatically correct sign for debit transactions
        b.tx.Amount.Amount = b.tx.Amount.Amount.Neg()
    }
    
    // Populate derived fields
    b.populateDerivedFields()
    
    return b.tx, nil
}

func (b *TransactionBuilder) populateDerivedFields() {
    // Set party name based on direction for backward compatibility
    if b.tx.Direction == DirectionDebit {
        b.tx.PartyName = b.tx.Payee.Name
        b.tx.PartyIBAN = b.tx.Payee.IBAN
    } else if b.tx.Direction == DirectionCredit {
        b.tx.PartyName = b.tx.Payer.Name
        b.tx.PartyIBAN = b.tx.Payer.IBAN
    }
    
    // Set legacy amount fields for backward compatibility
    if amount, exact := b.tx.Amount.Amount.Float64(); exact {
        b.tx.AmountFloat = amount
    }
    
    // Generate ID if not set
    if b.tx.ID == "" {
        b.tx.ID = uuid.New().String()
    }
}
```

**Migration Strategy for Parsers:**

Each parser will be gradually migrated to use the TransactionBuilder:

```go
// Before (direct struct construction)
func (p *ISO20022Parser) entryToTransaction(entry *models.Entry) models.Transaction {
    return models.Transaction{
        Date:        parseDate(entry.BookgDt.Dt),
        Amount:      parseAmount(entry.Amt.Value),
        Description: entry.AddtlNtryInf,
        // ... many more fields
    }
}

// After (using builder pattern)
func (p *ISO20022Parser) entryToTransaction(entry *models.Entry) (models.Transaction, error) {
    return models.NewTransactionBuilder().
        WithDate(entry.BookgDt.Dt).
        WithAmount(parseAmount(entry.Amt.Value), entry.Amt.Ccy).
        WithDescription(entry.AddtlNtryInf).
        WithPayer(entry.NtryDtls.TxDtls.RltdPties.Dbtr.Nm, entry.NtryDtls.TxDtls.RltdPties.DbtrAcct.Id.IBAN).
        WithPayee(entry.NtryDtls.TxDtls.RltdPties.Cdtr.Nm, entry.NtryDtls.TxDtls.RltdPties.CdtrAcct.Id.IBAN).
        AsDebit().
        Build()
}
```

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
        cfg.Categories.DebtorsFile,
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

**Dependency Lifecycle Management:**

The container manages the complete lifecycle of dependencies:

```go
// Container with proper cleanup
type Container struct {
    Logger      logging.Logger
    Config      *config.Config
    Store       *store.CategoryStore
    AIClient    categorizer.AIClient
    Categorizer *categorizer.Categorizer
    Parsers     map[parser.ParserType]parser.FullParser
    
    // Internal cleanup tracking
    cleanupFuncs []func() error
    mu           sync.RWMutex
}

// NewContainer with error handling and validation
func NewContainer(cfg *config.Config) (*Container, error) {
    if cfg == nil {
        return nil, errors.New("configuration cannot be nil")
    }
    
    container := &Container{
        Config:       cfg,
        cleanupFuncs: make([]func() error, 0),
    }
    
    // Initialize logger first (needed by other components)
    if err := container.initLogger(); err != nil {
        return nil, fmt.Errorf("failed to initialize logger: %w", err)
    }
    
    // Initialize store with validation
    if err := container.initStore(); err != nil {
        return nil, fmt.Errorf("failed to initialize store: %w", err)
    }
    
    // Initialize AI client (optional)
    if err := container.initAIClient(); err != nil {
        return nil, fmt.Errorf("failed to initialize AI client: %w", err)
    }
    
    // Initialize categorizer with all dependencies
    if err := container.initCategorizer(); err != nil {
        return nil, fmt.Errorf("failed to initialize categorizer: %w", err)
    }
    
    // Initialize parsers with dependency injection
    if err := container.initParsers(); err != nil {
        return nil, fmt.Errorf("failed to initialize parsers: %w", err)
    }
    
    return container, nil
}

func (c *Container) initLogger() error {
    logger, err := logging.NewLogrusAdapter(c.Config.Log.Level, c.Config.Log.Format)
    if err != nil {
        return fmt.Errorf("invalid logger configuration: %w", err)
    }
    c.Logger = logger
    return nil
}

func (c *Container) initStore() error {
    store, err := store.NewCategoryStore(
        c.Config.Categories.File,
        c.Config.Categories.CreditorsFile,
        c.Config.Categories.DebtorsFile,
        c.Logger, // Inject logger dependency
    )
    if err != nil {
        return fmt.Errorf("failed to load category store: %w", err)
    }
    c.Store = store
    return nil
}

// Cleanup releases all resources
func (c *Container) Cleanup() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    var errs []error
    for i := len(c.cleanupFuncs) - 1; i >= 0; i-- {
        if err := c.cleanupFuncs[i](); err != nil {
            errs = append(errs, err)
        }
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("cleanup errors: %v", errs)
    }
    return nil
}

// AddCleanup registers a cleanup function
func (c *Container) addCleanup(cleanup func() error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cleanupFuncs = append(c.cleanupFuncs, cleanup)
}
```

**Eliminating Global State:**

```go
// Before (global singleton - avoid this)
var (
    defaultCategorizer *categorizer.Categorizer
    once              sync.Once
)

func GetDefaultCategorizer() *categorizer.Categorizer {
    once.Do(func() {
        // Global initialization
        defaultCategorizer = categorizer.NewCategorizer(...)
    })
    return defaultCategorizer
}

// After (dependency injection)
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    container, err := container.NewContainer(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer container.Cleanup()
    
    // Pass dependencies explicitly to commands
    rootCmd := cmd.NewRootCommand(container)
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

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
6. **Use errors.Is and errors.As**: For error inspection instead of string comparison

**Error Inspection Patterns:**

```go
// Before (string comparison - avoid this)
if err != nil && strings.Contains(err.Error(), "validation failed") {
    // Handle validation error
}

// After (proper error inspection)
var validationErr *parsererror.ValidationError
if errors.As(err, &validationErr) {
    // Handle validation error with access to structured data
    p.logger.Warn("Validation failed", 
        logging.Field{Key: "field", Value: validationErr.Field},
        logging.Field{Key: "reason", Value: validationErr.Reason})
}

// Check for specific error types
if errors.Is(err, parsererror.ErrInvalidFormat) {
    // Handle invalid format specifically
    return fmt.Errorf("unsupported file format: %w", err)
}
```

**Consistent Error Context:**

```go
// Always provide context when wrapping errors
func (p *ISO20022Parser) Parse(r io.Reader) ([]models.Transaction, error) {
    data, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("failed to read CAMT.053 input: %w", err)
    }
    
    var document models.ISO20022Document
    if err := xml.Unmarshal(data, &document); err != nil {
        return nil, &parsererror.ParseError{
            Parser: "ISO20022",
            Field:  "document",
            Value:  string(data[:min(100, len(data))]), // Truncate for logging
            Err:    fmt.Errorf("XML unmarshaling failed: %w", err),
        }
    }
    
    return p.extractTransactions(document)
}
```

## Testing Strategy

### Unit Testing Approach

**Risk-Based Testing Strategy:**

The testing strategy prioritizes critical functionality based on risk assessment:

**Critical Paths (100% Coverage Required):**

- Transaction parsing logic (all parsers)
- Financial calculations and Money operations
- Data validation and error handling
- Categorization strategy implementations
- CSV marshaling/unmarshaling

**High-Risk Areas (90%+ Coverage):**

- Configuration loading and validation
- File I/O operations
- External API integrations (AI client)
- Dependency injection container

**Standard Coverage Areas (70%+ Coverage):**

- CLI command implementations
- Utility functions
- Logging and monitoring code

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
    DebtorMappings   map[string]string
}
```

**Test Structure with Risk-Based Focus:**

```go
func TestCategorizer_Categorize(t *testing.T) {
    // Critical path tests - comprehensive coverage
    tests := []struct {
        name           string
        transaction    Transaction
        mappings       map[string]string
        expectedCat    string
        expectedErr    bool
        riskLevel      string // "critical", "high", "standard"
    }{
        {
            name: "direct mapping found - critical path",
            transaction: Transaction{
                Payer: Party{Name: "COOP"},
                Direction: DirectionDebit,
            },
            mappings: map[string]string{
                "coop": "Alimentation",
            },
            expectedCat: "Alimentation",
            expectedErr: false,
            riskLevel:   "critical",
        },
        {
            name: "financial calculation accuracy - critical path",
            transaction: Transaction{
                Amount: Money{Amount: decimal.NewFromFloat(100.50), Currency: "CHF"},
                Direction: DirectionDebit,
            },
            expectedErr: false,
            riskLevel:   "critical",
        },
        {
            name: "edge case - empty party name",
            transaction: Transaction{
                Payer: Party{Name: ""},
                Direction: DirectionDebit,
            },
            expectedCat: "Uncategorized",
            expectedErr: false,
            riskLevel:   "high",
        },
        // ... more test cases covering edge cases and error scenarios
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup with dependency injection for testability
            mockStore := &MockCategoryStore{
                DebtorMappings: tt.mappings,
            }
            mockLogger := &MockLogger{}
            
            cat := NewCategorizer(mockStore, nil, mockLogger)
            
            // Execute
            result, err := cat.Categorize(context.Background(), tt.transaction)
            
            // Assert with detailed validation for critical paths
            if tt.riskLevel == "critical" {
                // More thorough validation for critical functionality
                require.NoError(t, err, "Critical path must not fail")
                assert.NotEmpty(t, result.Name, "Category must be assigned")
                if tt.expectedCat != "" {
                    assert.Equal(t, tt.expectedCat, result.Name)
                }
            } else {
                // Standard validation for other paths
                if tt.expectedErr {
                    assert.Error(t, err)
                } else {
                    assert.NoError(t, err)
                    if tt.expectedCat != "" {
                        assert.Equal(t, tt.expectedCat, result.Name)
                    }
                }
            }
        })
    }
}

// Benchmark tests for performance-critical paths
func BenchmarkCategorizer_Categorize(b *testing.B) {
    // Setup
    mockStore := &MockCategoryStore{
        DebtorMappings: map[string]string{
            "coop": "Alimentation",
            "migros": "Alimentation",
            // ... more mappings
        },
    }
    cat := NewCategorizer(mockStore, nil, &MockLogger{})
    
    tx := Transaction{
        Payer: Party{Name: "COOP SUPERMARKT"},
        Direction: DirectionDebit,
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := cat.Categorize(context.Background(), tx)
        if err != nil {
            b.Fatal(err)
        }
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
    // Multiple string operations creating temporary strings
    normalized := strings.ReplaceAll(partyNameLower, " ", "")
    normalized = strings.ReplaceAll(normalized, "-", "")
    // Each operation allocates new strings
}
```

**After:**

```go
func (c *Categorizer) categorizeByMapping(tx Transaction) (Category, bool) {
    // Pre-allocate builder with estimated capacity
    var builder strings.Builder
    builder.Grow(len(tx.PartyName)) // Avoid reallocations
    
    // Single pass normalization to minimize allocations
    for _, r := range strings.ToLower(tx.PartyName) {
        if r != ' ' && r != '-' {
            builder.WriteRune(r)
        }
    }
    normalized := builder.String()
    
    // Use the normalized string for mapping lookup
}
```

**Performance Impact:**

- Reduces string allocations by 60-80% in categorization hot path
- Single-pass processing instead of multiple string operations
- Pre-allocated capacity prevents buffer reallocations

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

### 3. Pre-allocation and Capacity Management

**Before:**

```go
var transactions []models.Transaction
for _, entry := range entries {
    transactions = append(transactions, convertEntry(entry))
}

// Maps without size hints
mappings := make(map[string]string)
for _, item := range items {
    mappings[item.Key] = item.Value
}
```

**After:**

```go
// Pre-allocate slice with known capacity
transactions := make([]models.Transaction, 0, len(entries))
for _, entry := range entries {
    transactions = append(transactions, convertEntry(entry))
}

// Pre-allocate maps with size hints to reduce rehashing
mappings := make(map[string]string, len(items))
for _, item := range items {
    mappings[item.Key] = item.Value
}

// For large datasets, consider batch processing
const batchSize = 1000
if len(entries) > batchSize {
    // Process in batches to control memory usage
    for i := 0; i < len(entries); i += batchSize {
        end := i + batchSize
        if end > len(entries) {
            end = len(entries)
        }
        batch := entries[i:end]
        processBatch(batch)
    }
}
```

**Performance Benefits:**

- Eliminates slice reallocations during growth
- Reduces map rehashing operations
- Controls memory usage for large datasets
- Improves cache locality through better memory layout

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

### 7. Complete Transaction Model Decomposition (Requirement 16)

**Package**: `internal/models`

**Decomposed Transaction Architecture:**

The Transaction model is decomposed into focused components following Single Responsibility Principle:

```go
// TransactionCore contains essential transaction data (Requirement 16.1)
type TransactionCore struct {
    ID          string    `json:"id" csv:"id"`
    Date        time.Time `json:"date" csv:"date"`
    ValueDate   time.Time `json:"value_date" csv:"value_date"`
    Amount      Money     `json:"amount" csv:"amount"`
    Description string    `json:"description" csv:"description"`
    Status      string    `json:"status" csv:"status"`
    Reference   string    `json:"reference" csv:"reference"`
}

// NewTransactionCore creates a validated core transaction
func NewTransactionCore(id string, date time.Time, amount Money, description string) (TransactionCore, error) {
    if id == "" {
        return TransactionCore{}, &validation.ValidationError{
            Field: "id", Reason: "transaction ID cannot be empty",
        }
    }
    if date.IsZero() {
        return TransactionCore{}, &validation.ValidationError{
            Field: "date", Reason: "transaction date is required",
        }
    }
    return TransactionCore{
        ID: id, Date: date, ValueDate: date, Amount: amount, Description: description,
        Status: constants.StatusCompleted,
    }, nil
}

// Party represents a transaction party with validation (Requirement 16.2)
type Party struct {
    Name string `json:"name" yaml:"name"`
    IBAN string `json:"iban" yaml:"iban"`
}

// NewParty creates a validated Party instance
func NewParty(name, iban string) (Party, error) {
    if strings.TrimSpace(name) == "" {
        return Party{}, &validation.ValidationError{
            Field:  "name",
            Reason: "party name cannot be empty",
        }
    }
    
    if iban != "" && !validation.IsValidIBAN(iban) {
        return Party{}, &validation.ValidationError{
            Field:  "iban",
            Reason: fmt.Sprintf("invalid IBAN format: %s", iban),
        }
    }
    
    return Party{Name: name, IBAN: iban}, nil
}

// Enhanced Money with comprehensive validation and operations (Requirement 16.3)
type Money struct {
    Amount   decimal.Decimal `json:"amount" yaml:"amount"`
    Currency string          `json:"currency" yaml:"currency"`
}

// NewMoney creates a validated Money instance
func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
    if !validation.IsValidCurrency(currency) {
        return Money{}, &validation.ValidationError{
            Field:  "currency",
            Reason: fmt.Sprintf("invalid currency code: %s", currency),
        }
    }
    
    if amount.LessThan(validation.MinAmount) || amount.GreaterThan(validation.MaxAmount) {
        return Money{}, &validation.ValidationError{
            Field:  "amount",
            Reason: fmt.Sprintf("amount %s outside valid range", amount.String()),
        }
    }
    
    return Money{Amount: amount, Currency: currency}, nil
}

// Add safely adds two Money values of the same currency
func (m Money) Add(other Money) (Money, error) {
    if m.Currency != other.Currency {
        return Money{}, fmt.Errorf("cannot add different currencies: %s and %s", m.Currency, other.Currency)
    }
    return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}

// TransactionWithParties adds party information (Requirement 16.4)
type TransactionWithParties struct {
    TransactionCore
    Payer     Party                `json:"payer" csv:"payer"`
    Payee     Party                `json:"payee" csv:"payee"`
    Direction TransactionDirection `json:"direction" csv:"direction"`
}

// NewTransactionWithParties creates a transaction with party information
func NewTransactionWithParties(core TransactionCore, payer, payee Party, direction TransactionDirection) TransactionWithParties {
    return TransactionWithParties{
        TransactionCore: core,
        Payer:          payer,
        Payee:          payee,
        Direction:      direction,
    }
}

// CategorizedTransaction adds categorization data (Requirement 16.4)
type CategorizedTransaction struct {
    TransactionWithParties
    Category string `json:"category" csv:"category"`
    Type     string `json:"type" csv:"type"`
    Fund     string `json:"fund" csv:"fund"`
}

// NewCategorizedTransaction creates a categorized transaction
func NewCategorizedTransaction(withParties TransactionWithParties, category, txType, fund string) CategorizedTransaction {
    return CategorizedTransaction{
        TransactionWithParties: withParties,
        Category:              category,
        Type:                  txType,
        Fund:                  fund,
    }
}

// Transaction maintains backward compatibility through composition (Requirement 16.5)
type Transaction struct {
    CategorizedTransaction
    
    // Additional fields for specific formats (backward compatibility)
    BookkeepingNumber string          `json:"bookkeeping_number,omitempty" csv:"bookkeeping_number"`
    RemittanceInfo    string          `json:"remittance_info,omitempty" csv:"remittance_info"`
    PartyIBAN         string          `json:"party_iban,omitempty" csv:"party_iban"`
    Investment        string          `json:"investment,omitempty" csv:"investment"`
    NumberOfShares    int             `json:"number_of_shares,omitempty" csv:"number_of_shares"`
    Fees              Money           `json:"fees,omitempty" csv:"fees"`
    EntryReference    string          `json:"entry_reference,omitempty" csv:"entry_reference"`
    AccountServicer   string          `json:"account_servicer,omitempty" csv:"account_servicer"`
    BankTxCode        string          `json:"bank_tx_code,omitempty" csv:"bank_tx_code"`
    OriginalAmount    Money           `json:"original_amount,omitempty" csv:"original_amount"`
    ExchangeRate      decimal.Decimal `json:"exchange_rate,omitempty" csv:"exchange_rate"`
    
    // Tax-related fields
    AmountExclTax Money           `json:"amount_excl_tax,omitempty" csv:"amount_excl_tax"`
    AmountTax     Money           `json:"amount_tax,omitempty" csv:"amount_tax"`
    TaxRate       decimal.Decimal `json:"tax_rate,omitempty" csv:"tax_rate"`
    
    // Legacy compatibility fields (deprecated but maintained)
    PartyName   string  `json:"party_name,omitempty" csv:"party_name"`
    AmountFloat float64 `json:"amount_float,omitempty" csv:"amount_float"`
}

// Backward compatibility methods (Requirements 14.4, 4.5)
func (t *Transaction) GetPayee() Party {
    if t.Direction == DirectionDebit {
        return t.Payee
    }
    return t.Payer
}

func (t *Transaction) GetPayer() Party {
    if t.Direction == DirectionDebit {
        return t.Payer
    }
    return t.Payee
}

// GetAmountAsFloat provides backward compatibility (deprecated)
// Deprecated: Use Amount.Amount.InexactFloat64() for decimal precision
func (t *Transaction) GetAmountAsFloat() float64 {
    f, _ := t.Amount.Amount.Float64()
    return f
}
```

### 8. Comprehensive Validation Framework (Requirement 17)

**Package**: `internal/validation`

**Validation Infrastructure:**

```go
// ValidationError provides structured validation error information (Requirement 17.4)
type ValidationError struct {
    Field   string                 `json:"field"`
    Reason  string                 `json:"reason"`
    Context map[string]interface{} `json:"context,omitempty"`
    Source  string                 `json:"source,omitempty"`
}

func (e *ValidationError) Error() string {
    if e.Source != "" {
        return fmt.Sprintf("validation failed in %s for field '%s': %s", e.Source, e.Field, e.Reason)
    }
    return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Reason)
}

// Validator interface for composable validation
type Validator interface {
    Validate() error
}

// ValidationContext provides context for validation operations
type ValidationContext struct {
    Source     string // e.g., "parser", "user_input", "api"
    FilePath   string // for file-based validation
    LineNumber int    // for line-specific errors
    Operation  string // e.g., "transaction_creation", "categorization"
}

// Core validation functions (Requirements 17.1, 17.2, 17.3)

// IsValidIBAN validates IBAN format using check digits
func IsValidIBAN(iban string) bool {
    // Remove spaces and convert to uppercase
    iban = strings.ToUpper(strings.ReplaceAll(iban, " ", ""))
    
    // Basic length and format check
    if len(iban) < 15 || len(iban) > 34 {
        return false
    }
    
    // Country code validation (first 2 characters must be letters)
    if !isAlpha(iban[0]) || !isAlpha(iban[1]) {
        return false
    }
    
    // Check digits validation (characters 3-4 must be digits)
    if !isDigit(iban[2]) || !isDigit(iban[3]) {
        return false
    }
    
    // MOD-97 validation algorithm
    return validateIBANChecksum(iban)
}

// IsValidCurrency validates ISO 4217 currency codes
func IsValidCurrency(currency string) bool {
    validCurrencies := map[string]bool{
        "CHF": true, "EUR": true, "USD": true, "GBP": true,
        "JPY": true, "CAD": true, "AUD": true, "SEK": true,
        "NOK": true, "DKK": true, "PLN": true, "CZK": true,
        // Add more as needed
    }
    return validCurrencies[strings.ToUpper(currency)]
}

// ValidateAmount checks if amount is within reasonable ranges
func ValidateAmount(amount decimal.Decimal, currency string) error {
    // Define reasonable limits per currency
    limits := map[string]struct{ min, max decimal.Decimal }{
        "CHF": {decimal.NewFromFloat(-1000000), decimal.NewFromFloat(1000000)},
        "EUR": {decimal.NewFromFloat(-1000000), decimal.NewFromFloat(1000000)},
        "USD": {decimal.NewFromFloat(-1000000), decimal.NewFromFloat(1000000)},
    }
    
    limit, exists := limits[currency]
    if !exists {
        // Default limits for unknown currencies
        limit = limits["EUR"]
    }
    
    if amount.LessThan(limit.min) || amount.GreaterThan(limit.max) {
        return &ValidationError{
            Field:  "amount",
            Reason: fmt.Sprintf("amount %s outside valid range [%s, %s] for currency %s", 
                amount.String(), limit.min.String(), limit.max.String(), currency),
            Context: map[string]interface{}{
                "currency": currency,
                "min":      limit.min.String(),
                "max":      limit.max.String(),
            },
        }
    }
    
    return nil
}

// ValidateDate checks if date is within reasonable ranges
func ValidateDate(date time.Time, fieldName string) error {
    now := time.Now()
    minDate := now.AddDate(-10, 0, 0) // 10 years ago
    maxDate := now.AddDate(1, 0, 0)   // 1 year in future
    
    if date.Before(minDate) || date.After(maxDate) {
        return &ValidationError{
            Field:  fieldName,
            Reason: fmt.Sprintf("date %s outside valid range [%s, %s]", 
                date.Format("2006-01-02"), minDate.Format("2006-01-02"), maxDate.Format("2006-01-02")),
            Context: map[string]interface{}{
                "date":     date.Format(time.RFC3339),
                "min_date": minDate.Format(time.RFC3339),
                "max_date": maxDate.Format(time.RFC3339),
            },
        }
    }
    
    return nil
}

// TransactionValidator provides comprehensive transaction validation (Requirement 17.1)
type TransactionValidator struct {
    context ValidationContext
}

func NewTransactionValidator(ctx ValidationContext) *TransactionValidator {
    return &TransactionValidator{context: ctx}
}

func (tv *TransactionValidator) ValidateTransaction(tx Transaction) error {
    var errors []error
    
    // Validate core fields
    if tx.ID == "" {
        errors = append(errors, &ValidationError{
            Field: "id", Reason: "transaction ID cannot be empty", Source: tv.context.Source,
        })
    }
    
    if tx.Date.IsZero() {
        errors = append(errors, &ValidationError{
            Field: "date", Reason: "transaction date is required", Source: tv.context.Source,
        })
    } else if err := ValidateDate(tx.Date, "date"); err != nil {
        errors = append(errors, err)
    }
    
    // Validate amount and currency
    if err := ValidateAmount(tx.Amount.Amount, tx.Amount.Currency); err != nil {
        errors = append(errors, err)
    }
    
    // Validate parties
    if tx.Payer.Name == "" && tx.Payee.Name == "" {
        errors = append(errors, &ValidationError{
            Field: "parties", Reason: "at least one party (payer or payee) must be specified", Source: tv.context.Source,
        })
    }
    
    if tx.Payer.IBAN != "" && !IsValidIBAN(tx.Payer.IBAN) {
        errors = append(errors, &ValidationError{
            Field: "payer.iban", Reason: fmt.Sprintf("invalid IBAN format: %s", tx.Payer.IBAN), Source: tv.context.Source,
        })
    }
    
    if tx.Payee.IBAN != "" && !IsValidIBAN(tx.Payee.IBAN) {
        errors = append(errors, &ValidationError{
            Field: "payee.iban", Reason: fmt.Sprintf("invalid IBAN format: %s", tx.Payee.IBAN), Source: tv.context.Source,
        })
    }
    
    // Return aggregated errors
    if len(errors) > 0 {
        return &ValidationErrors{Errors: errors}
    }
    
    return nil
}

// ValidationErrors aggregates multiple validation errors
type ValidationErrors struct {
    Errors []error `json:"errors"`
}

func (ve *ValidationErrors) Error() string {
    if len(ve.Errors) == 1 {
        return ve.Errors[0].Error()
    }
    return fmt.Sprintf("validation failed with %d errors: %s", len(ve.Errors), ve.Errors[0].Error())
}

func (ve *ValidationErrors) Unwrap() []error {
    return ve.Errors
}

// Helper functions
func isAlpha(c byte) bool {
    return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isDigit(c byte) bool {
    return c >= '0' && c <= '9'
}

func validateIBANChecksum(iban string) bool {
    // Move first 4 characters to end
    rearranged := iban[4:] + iban[:4]
    
    // Replace letters with numbers (A=10, B=11, ..., Z=35)
    var numStr strings.Builder
    for _, c := range rearranged {
        if c >= 'A' && c <= 'Z' {
            numStr.WriteString(fmt.Sprintf("%d", int(c-'A'+10)))
        } else {
            numStr.WriteByte(byte(c))
        }
    }
    
    // Calculate MOD 97
    return calculateMod97(numStr.String()) == 1
}

func calculateMod97(numStr string) int {
    remainder := 0
    for _, digit := range numStr {
        remainder = (remainder*10 + int(digit-'0')) % 97
    }
    return remainder
}

// Constants for validation
var (
    MinAmount = decimal.NewFromFloat(-1000000)
    MaxAmount = decimal.NewFromFloat(1000000)
)
```

### 9. Advanced Performance Optimizations (Requirement 18)

**Package**: `internal/processing`

**Concurrent Processing Architecture:**

```go
// ConcurrentProcessor handles parallel processing of transactions (Requirement 18.2)
type ConcurrentProcessor struct {
    workerCount int
    batchSize   int
    logger      logging.Logger
    pool        *sync.Pool // Object pool for reusable objects
}

// NewConcurrentProcessor creates a processor with optimal worker count
func NewConcurrentProcessor(logger logging.Logger) *ConcurrentProcessor {
    workerCount := runtime.NumCPU()
    if workerCount > 8 {
        workerCount = 8 // Cap at 8 workers to avoid overhead
    }
    
    return &ConcurrentProcessor{
        workerCount: workerCount,
        batchSize:   100, // Configurable threshold
        logger:      logger,
        pool: &sync.Pool{
            New: func() interface{} {
                return make([]Transaction, 0, 100) // Pre-allocated slice pool
            },
        },
    }
}

// ProcessTransactions processes transactions concurrently when beneficial (Requirement 18.2)
func (cp *ConcurrentProcessor) ProcessTransactions(
    ctx context.Context,
    transactions []Transaction,
    processor func(Transaction) (Transaction, error),
) ([]Transaction, error) {
    
    // Use sequential processing for small datasets to avoid overhead
    if len(transactions) < cp.batchSize {
        return cp.processSequential(transactions, processor)
    }
    
    cp.logger.Info("Using concurrent processing",
        logging.Field{Key: "transaction_count", Value: len(transactions)},
        logging.Field{Key: "worker_count", Value: cp.workerCount})
    
    return cp.processConcurrent(ctx, transactions, processor)
}

// processConcurrent uses worker pools for large datasets
func (cp *ConcurrentProcessor) processConcurrent(
    ctx context.Context,
    transactions []Transaction,
    processor func(Transaction) (Transaction, error),
) ([]Transaction, error) {
    
    // Pre-allocate result slice with known capacity (Requirement 18.3)
    processedTxs := make([]Transaction, len(transactions))
    
    // Create channels for work distribution
    jobs := make(chan indexedTransaction, cp.workerCount*2) // Buffered for better throughput
    results := make(chan indexedResult, cp.workerCount*2)
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < cp.workerCount; i++ {
        wg.Add(1)
        go cp.worker(ctx, &wg, jobs, results, processor)
    }
    
    // Send jobs
    go func() {
        defer close(jobs)
        for i, tx := range transactions {
            select {
            case jobs <- indexedTransaction{index: i, transaction: tx}:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    // Wait for completion
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Collect results in order
    var errors []error
    for result := range results {
        if result.err != nil {
            errors = append(errors, fmt.Errorf("transaction %d: %w", result.index, result.err))
        } else {
            processedTxs[result.index] = result.transaction
        }
    }
    
    if len(errors) > 0 {
        return processedTxs, &BatchProcessingError{Errors: errors}
    }
    
    return processedTxs, nil
}

// worker processes transactions in a goroutine
func (cp *ConcurrentProcessor) worker(
    ctx context.Context,
    wg *sync.WaitGroup,
    jobs <-chan indexedTransaction,
    results chan<- indexedResult,
    processor func(Transaction) (Transaction, error),
) {
    defer wg.Done()
    
    for {
        select {
        case job, ok := <-jobs:
            if !ok {
                return
            }
            
            tx, err := processor(job.transaction)
            results <- indexedResult{
                index:       job.index,
                transaction: tx,
                err:         err,
            }
            
        case <-ctx.Done():
            return
        }
    }
}

// String Operations Optimization (Requirement 18.1)
type OptimizedStringProcessor struct {
    builderPool *sync.Pool
}

func NewOptimizedStringProcessor() *OptimizedStringProcessor {
    return &OptimizedStringProcessor{
        builderPool: &sync.Pool{
            New: func() interface{} {
                return &strings.Builder{}
            },
        },
    }
}

// NormalizePartyName optimizes string operations to minimize allocations
func (osp *OptimizedStringProcessor) NormalizePartyName(name string) string {
    if name == "" {
        return ""
    }
    
    // Get builder from pool
    builder := osp.builderPool.Get().(*strings.Builder)
    defer func() {
        builder.Reset()
        osp.builderPool.Put(builder)
    }()
    
    // Pre-allocate capacity based on input length (Requirement 18.1)
    builder.Grow(len(name))
    
    // Single-pass normalization to minimize allocations
    for _, r := range strings.ToLower(name) {
        if r != ' ' && r != '-' && r != '.' {
            builder.WriteRune(r)
        }
    }
    
    return builder.String()
}

// BatchProcessor handles large dataset processing with memory management
type BatchProcessor struct {
    batchSize int
    processor *ConcurrentProcessor
    logger    logging.Logger
}

func NewBatchProcessor(batchSize int, logger logging.Logger) *BatchProcessor {
    return &BatchProcessor{
        batchSize: batchSize,
        processor: NewConcurrentProcessor(logger),
        logger:    logger,
    }
}

// ProcessLargeDataset processes datasets in batches to control memory usage (Requirement 18.2)
func (bp *BatchProcessor) ProcessLargeDataset(
    ctx context.Context,
    transactions []Transaction,
    processor func(Transaction) (Transaction, error),
) ([]Transaction, error) {
    
    if len(transactions) <= bp.batchSize {
        return bp.processor.ProcessTransactions(ctx, transactions, processor)
    }
    
    bp.logger.Info("Processing large dataset in batches",
        logging.Field{Key: "total_transactions", Value: len(transactions)},
        logging.Field{Key: "batch_size", Value: bp.batchSize})
    
    // Pre-allocate result slice (Requirement 18.3)
    results := make([]Transaction, 0, len(transactions))
    
    // Process in batches
    for i := 0; i < len(transactions); i += bp.batchSize {
        end := i + bp.batchSize
        if end > len(transactions) {
            end = len(transactions)
        }
        
        batch := transactions[i:end]
        batchResults, err := bp.processor.ProcessTransactions(ctx, batch, processor)
        if err != nil {
            return results, fmt.Errorf("batch %d-%d failed: %w", i, end, err)
        }
        
        results = append(results, batchResults...)
        
        // Check for cancellation between batches
        select {
        case <-ctx.Done():
            return results, ctx.Err()
        default:
        }
    }
    
    return results, nil
}

// Performance monitoring and metrics
type PerformanceMetrics struct {
    ProcessingTime    time.Duration `json:"processing_time"`
    TransactionCount  int           `json:"transaction_count"`
    ThroughputPerSec  float64       `json:"throughput_per_sec"`
    MemoryAllocations int64         `json:"memory_allocations"`
    WorkerCount       int           `json:"worker_count"`
}

// MeasurePerformance wraps processing with performance measurement (Requirement 18.5)
func MeasurePerformance(
    ctx context.Context,
    transactions []Transaction,
    processor func(context.Context, []Transaction) ([]Transaction, error),
) ([]Transaction, PerformanceMetrics, error) {
    
    var memBefore, memAfter runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&memBefore)
    
    start := time.Now()
    results, err := processor(ctx, transactions)
    duration := time.Since(start)
    
    runtime.ReadMemStats(&memAfter)
    
    metrics := PerformanceMetrics{
        ProcessingTime:    duration,
        TransactionCount:  len(transactions),
        ThroughputPerSec:  float64(len(transactions)) / duration.Seconds(),
        MemoryAllocations: int64(memAfter.Mallocs - memBefore.Mallocs),
        WorkerCount:       runtime.NumCPU(),
    }
    
    return results, metrics, err
}

type indexedTransaction struct {
    index       int
    transaction Transaction
}

type indexedResult struct {
    index       int
    transaction Transaction
    err         error
}

// BatchProcessingError aggregates errors from concurrent processing
type BatchProcessingError struct {
    Errors []error
}

func (e *BatchProcessingError) Error() string {
    return fmt.Sprintf("batch processing failed with %d errors", len(e.Errors))
}
```

### 10. Enhanced Error Context and Debugging (Requirement 19)

**Package**: `internal/errors`

**Enhanced Error Context Design:**

```go
import (
    "regexp"
    "strings"
    "sync"
    "time"
)

// ContextualError provides rich context for debugging (Requirement 19.1, 19.3)
type ContextualError struct {
    Operation     string                 `json:"operation"`
    Component     string                 `json:"component"`
    FilePath      string                 `json:"file_path,omitempty"`
    LineNumber    int                    `json:"line_number,omitempty"`
    DataSnippet   string                 `json:"data_snippet,omitempty"` // Truncated for security
    Context       map[string]interface{} `json:"context,omitempty"`
    Underlying    error                  `json:"-"`
    Timestamp     time.Time              `json:"timestamp"`
    BatchInfo     *BatchContext          `json:"batch_info,omitempty"`
}

// BatchContext provides batch processing context (Requirement 19.4)
type BatchContext struct {
    BatchID       string `json:"batch_id"`
    TotalItems    int    `json:"total_items"`
    ProcessedItems int   `json:"processed_items"`
    CurrentItem   int    `json:"current_item"`
    Progress      float64 `json:"progress"`
}

func NewContextualError(operation, component string) *ContextualError {
    return &ContextualError{
        Operation: operation,
        Component: component,
        Timestamp: time.Now(),
        Context:   make(map[string]interface{}),
    }
}

func (e *ContextualError) WithFile(filePath string, lineNumber int) *ContextualError {
    e.FilePath = filePath
    e.LineNumber = lineNumber
    return e
}

func (e *ContextualError) WithDataSnippet(data string) *ContextualError {
    // Truncate and sanitize data snippet for security (Requirement 19.5)
    const maxSnippetLength = 100
    if len(data) > maxSnippetLength {
        e.DataSnippet = data[:maxSnippetLength] + "..."
    } else {
        e.DataSnippet = data
    }
    // Remove potential sensitive information
    e.DataSnippet = sanitizeDataSnippet(e.DataSnippet)
    return e
}

func (e *ContextualError) WithContext(key string, value interface{}) *ContextualError {
    e.Context[key] = value
    return e
}

func (e *ContextualError) WithBatch(batchCtx *BatchContext) *ContextualError {
    e.BatchInfo = batchCtx
    return e
}

func (e *ContextualError) WithUnderlying(err error) *ContextualError {
    e.Underlying = err
    return e
}

func (e *ContextualError) Error() string {
    var parts []string
    parts = append(parts, fmt.Sprintf("%s failed in %s", e.Operation, e.Component))
    
    if e.FilePath != "" {
        parts = append(parts, fmt.Sprintf("file: %s", e.FilePath))
        if e.LineNumber > 0 {
            parts = append(parts, fmt.Sprintf("line: %d", e.LineNumber))
        }
    }
    
    if e.BatchInfo != nil {
        parts = append(parts, fmt.Sprintf("batch: %s (item %d/%d, %.1f%%)", 
            e.BatchInfo.BatchID, e.BatchInfo.CurrentItem, e.BatchInfo.TotalItems, e.BatchInfo.Progress*100))
    }
    
    if e.DataSnippet != "" {
        parts = append(parts, fmt.Sprintf("data: %s", e.DataSnippet))
    }
    
    if e.Underlying != nil {
        parts = append(parts, fmt.Sprintf("error: %v", e.Underlying))
    }
    
    return strings.Join(parts, ", ")
}

func (e *ContextualError) Unwrap() error {
    return e.Underlying
}

// sanitizeDataSnippet removes potential sensitive information
func sanitizeDataSnippet(data string) string {
    // Remove potential account numbers, IBANs, etc.
    // This is a basic implementation - extend as needed
    patterns := []string{
        `\b[A-Z]{2}\d{2}[A-Z0-9]{4}\d{7}[A-Z0-9]{1,23}\b`, // IBAN pattern
        `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,        // Credit card pattern
    }
    
    result := data
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        result = re.ReplaceAllString(result, "[REDACTED]")
    }
    
    return result
}

// ErrorCollector aggregates errors during batch operations (Requirement 19.2)
type ErrorCollector struct {
    errors    []error
    mu        sync.Mutex
    batchInfo *BatchContext
}

func NewErrorCollector(batchCtx *BatchContext) *ErrorCollector {
    return &ErrorCollector{
        errors:    make([]error, 0),
        batchInfo: batchCtx,
    }
}

func (ec *ErrorCollector) Add(err error) {
    if err == nil {
        return
    }
    
    ec.mu.Lock()
    defer ec.mu.Unlock()
    
    // Enhance error with batch context if it's a ContextualError
    if contextErr, ok := err.(*ContextualError); ok && ec.batchInfo != nil {
        contextErr.WithBatch(ec.batchInfo)
    }
    
    ec.errors = append(ec.errors, err)
}

func (ec *ErrorCollector) HasErrors() bool {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return len(ec.errors) > 0
}

func (ec *ErrorCollector) Errors() []error {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    return append([]error(nil), ec.errors...)
}

func (ec *ErrorCollector) ErrorSummary() string {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    
    if len(ec.errors) == 0 {
        return "no errors"
    }
    
    errorCounts := make(map[string]int)
    for _, err := range ec.errors {
        if contextErr, ok := err.(*ContextualError); ok {
            key := fmt.Sprintf("%s:%s", contextErr.Component, contextErr.Operation)
            errorCounts[key]++
        } else {
            errorCounts["other"]++
        }
    }
    
    var summary []string
    for errType, count := range errorCounts {
        summary = append(summary, fmt.Sprintf("%s: %d", errType, count))
    }
    
    return fmt.Sprintf("total errors: %d (%s)", len(ec.errors), strings.Join(summary, ", "))
}

// DebugTracer provides execution tracing for complex operations (Requirement 19.5)
type DebugTracer struct {
    enabled   bool
    logger    logging.Logger
    traces    []TraceEntry
    mu        sync.Mutex
    maxTraces int
}

type TraceEntry struct {
    Timestamp time.Time              `json:"timestamp"`
    Operation string                 `json:"operation"`
    Component string                 `json:"component"`
    Duration  time.Duration          `json:"duration"`
    Context   map[string]interface{} `json:"context,omitempty"`
    Success   bool                   `json:"success"`
    ErrorMsg  string                 `json:"error_msg,omitempty"`
}

func NewDebugTracer(logger logging.Logger, enabled bool) *DebugTracer {
    return &DebugTracer{
        enabled:   enabled,
        logger:    logger,
        traces:    make([]TraceEntry, 0),
        maxTraces: 1000, // Limit memory usage
    }
}

func (dt *DebugTracer) Trace(operation, component string, fn func() error) error {
    if !dt.enabled {
        return fn()
    }
    
    start := time.Now()
    err := fn()
    duration := time.Since(start)
    
    entry := TraceEntry{
        Timestamp: start,
        Operation: operation,
        Component: component,
        Duration:  duration,
        Success:   err == nil,
        Context:   make(map[string]interface{}),
    }
    
    if err != nil {
        entry.ErrorMsg = err.Error()
    }
    
    dt.mu.Lock()
    dt.traces = append(dt.traces, entry)
    // Keep only recent traces to prevent memory issues
    if len(dt.traces) > dt.maxTraces {
        dt.traces = dt.traces[len(dt.traces)-dt.maxTraces:]
    }
    dt.mu.Unlock()
    
    dt.logger.Debug("Operation traced",
        logging.Field{Key: "operation", Value: operation},
        logging.Field{Key: "component", Value: component},
        logging.Field{Key: "duration", Value: duration},
        logging.Field{Key: "success", Value: err == nil})
    
    return err
}

func (dt *DebugTracer) TraceWithContext(operation, component string, context map[string]interface{}, fn func() error) error {
    if !dt.enabled {
        return fn()
    }
    
    start := time.Now()
    err := fn()
    duration := time.Since(start)
    
    entry := TraceEntry{
        Timestamp: start,
        Operation: operation,
        Component: component,
        Duration:  duration,
        Success:   err == nil,
        Context:   context,
    }
    
    if err != nil {
        entry.ErrorMsg = err.Error()
    }
    
    dt.mu.Lock()
    dt.traces = append(dt.traces, entry)
    if len(dt.traces) > dt.maxTraces {
        dt.traces = dt.traces[len(dt.traces)-dt.maxTraces:]
    }
    dt.mu.Unlock()
    
    fields := []logging.Field{
        {Key: "operation", Value: operation},
        {Key: "component", Value: component},
        {Key: "duration", Value: duration},
        {Key: "success", Value: err == nil},
    }
    
    for k, v := range context {
        fields = append(fields, logging.Field{Key: k, Value: v})
    }
    
    dt.logger.Debug("Operation traced with context", fields...)
    
    return err
}

func (dt *DebugTracer) GetTraces() []TraceEntry {
    dt.mu.Lock()
    defer dt.mu.Unlock()
    return append([]TraceEntry(nil), dt.traces...)
}

func (dt *DebugTracer) GetTraceSummary() map[string]interface{} {
    dt.mu.Lock()
    defer dt.mu.Unlock()
    
    if len(dt.traces) == 0 {
        return map[string]interface{}{"total_traces": 0}
    }
    
    var totalDuration time.Duration
    successCount := 0
    operationCounts := make(map[string]int)
    
    for _, trace := range dt.traces {
        totalDuration += trace.Duration
        if trace.Success {
            successCount++
        }
        operationCounts[trace.Operation]++
    }
    
    return map[string]interface{}{
        "total_traces":     len(dt.traces),
        "success_rate":     float64(successCount) / float64(len(dt.traces)),
        "average_duration": totalDuration / time.Duration(len(dt.traces)),
        "operation_counts": operationCounts,
    }
}

// CategorizationErrorTracker tracks categorization strategy failures (Requirement 19.2)
type CategorizationErrorTracker struct {
    strategyAttempts map[string][]StrategyAttempt
    mu              sync.RWMutex
    logger          logging.Logger
}

type StrategyAttempt struct {
    StrategyName string    `json:"strategy_name"`
    Timestamp    time.Time `json:"timestamp"`
    Success      bool      `json:"success"`
    ErrorMsg     string    `json:"error_msg,omitempty"`
    Transaction  string    `json:"transaction_id"`
}

func NewCategorizationErrorTracker(logger logging.Logger) *CategorizationErrorTracker {
    return &CategorizationErrorTracker{
        strategyAttempts: make(map[string][]StrategyAttempt),
        logger:          logger,
    }
}

func (cet *CategorizationErrorTracker) RecordAttempt(transactionID, strategyName string, success bool, err error) {
    attempt := StrategyAttempt{
        StrategyName: strategyName,
        Timestamp:    time.Now(),
        Success:      success,
        Transaction:  transactionID,
    }
    
    if err != nil {
        attempt.ErrorMsg = err.Error()
    }
    
    cet.mu.Lock()
    cet.strategyAttempts[transactionID] = append(cet.strategyAttempts[transactionID], attempt)
    cet.mu.Unlock()
    
    if !success {
        cet.logger.Warn("Categorization strategy failed",
            logging.Field{Key: "transaction_id", Value: transactionID},
            logging.Field{Key: "strategy", Value: strategyName},
            logging.Field{Key: "error", Value: err})
    }
}

func (cet *CategorizationErrorTracker) GetFailedStrategies(transactionID string) []StrategyAttempt {
    cet.mu.RLock()
    defer cet.mu.RUnlock()
    
    attempts := cet.strategyAttempts[transactionID]
    var failed []StrategyAttempt
    
    for _, attempt := range attempts {
        if !attempt.Success {
            failed = append(failed, attempt)
        }
    }
    
    return failed
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

### Decision 7: Comprehensive Validation Framework

**Rationale**:

- Catches data errors early in the pipeline
- Provides clear, actionable error messages
- Enables better user experience
- Prevents invalid data propagation

**Trade-offs**:

- Additional validation overhead
- More complex error handling
- Benefits: Data integrity, better debugging, user experience

### Decision 8: Concurrent Processing for Large Datasets

**Rationale**:

- Improves performance for large financial datasets
- Utilizes multi-core processors effectively
- Maintains order of results
- Provides cancellation support

**Trade-offs**:

- Increased complexity
- Memory overhead for coordination
- Benefits: Significant performance gains, better resource utilization

### Decision 9: Complete Transaction Model Decomposition

**Rationale**:

- Eliminates god object anti-pattern completely
- Separates concerns into focused value objects
- Enables better validation and type safety
- Improves maintainability and testability

**Trade-offs**:

- More types to manage
- Slightly more complex construction
- Benefits: Much cleaner architecture, better separation of concerns

### Decision 10: Comprehensive Validation Framework

**Rationale**:

- Catches data errors early in the pipeline
- Provides clear, actionable error messages
- Enables better user experience
- Prevents invalid data propagation

**Trade-offs**:

- Additional validation overhead
- More complex error handling
- Benefits: Data integrity, better debugging, user experience

### Decision 11: Advanced Performance Optimizations

**Rationale**:

- Enables efficient processing of large financial datasets
- Utilizes multi-core processors effectively
- Minimizes memory allocations in hot paths
- Provides measurable performance improvements

**Trade-offs**:

- Increased code complexity
- Memory overhead for coordination
- Benefits: 20%+ performance improvement, better resource utilization

### Decision 12: Enhanced Error Context and Debugging

**Rationale**:

- Significantly improves troubleshooting capabilities
- Provides structured error information with file/line context
- Enables better monitoring and alerting
- Helps with production debugging and issue resolution

**Trade-offs**:

- Slightly larger error objects
- More complex error handling code
- Benefits: Much better debugging experience, faster issue resolution

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
//   store := store.NewCategoryStore("categories.yaml", "creditors.yaml", "debtors.yaml")
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

1. **Test Coverage**: 100% for critical paths (parsing, categorization, data validation), comprehensive coverage for remaining functionality based on risk assessment
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

## Requirements Coverage Summary

This design document comprehensively addresses all 19 requirements:

**Foundation Requirements (1-7)**: Dependency injection, error handling, parser refactoring, transaction decomposition, interface segregation, logging abstraction, and constants
**Core Refactoring (8-15)**: Naming conventions, date handling, test environment removal, strategy pattern, builder pattern, performance optimization, backward compatibility, and test coverage
**Advanced Features (16-19)**: Complete transaction model decomposition, comprehensive validation framework, advanced performance optimizations, and enhanced error context/debugging

Each requirement is mapped to specific design components with detailed implementation guidance, ensuring full traceability from requirements to design decisions.
