# ADR-003: Functional Programming Adoption

## Status

Accepted

## Context

Financial data processing requires high reliability, predictability, and testability. Traditional imperative programming approaches can lead to:

1. **Side Effects**: Difficult to track state changes
2. **Testing Complexity**: Hard to isolate and test individual components
3. **Concurrency Issues**: Shared mutable state creates race conditions
4. **Debugging Difficulty**: Complex state interactions are hard to reason about
5. **Maintainability**: Tightly coupled code is hard to modify

## Decision

We will adopt functional programming principles throughout the CAMT-CSV codebase:

### Core Principles

1. **Pure Functions**: Functions with no side effects that always return the same output for the same input
2. **Immutability**: Data structures should not be modified after creation
3. **Function Composition**: Build complex operations from simple, composable functions
4. **Separation of Pure and Impure Code**: Isolate side effects to specific boundaries

### Implementation Guidelines

```go
// ✅ Pure function - no side effects, deterministic
func calculateTax(amount decimal.Decimal, rate decimal.Decimal) decimal.Decimal {
    return amount.Mul(rate)
}

// ✅ Pure transformation function
func transformTransaction(tx models.Transaction, rules []TransformRule) models.Transaction {
    result := tx // Copy, don't modify original
    for _, rule := range rules {
        result = rule.Apply(result)
    }
    return result
}

// ✅ Separate pure logic from I/O
func ProcessFile(filePath string) error {
    // Impure: file I/O
    data, err := readFile(filePath)
    if err != nil {
        return err
    }
    
    // Pure: data transformation
    transactions := parseTransactions(data)
    categorized := categorizeTransactions(transactions)
    csvData := formatAsCSV(categorized)
    
    // Impure: file I/O
    return writeFile(outputPath, csvData)
}
```

## Consequences

### Positive

- **Testability**: Pure functions are easy to test with predictable inputs/outputs
- **Reliability**: No hidden side effects reduce bugs
- **Concurrency**: Immutable data is naturally thread-safe
- **Composability**: Small functions can be combined to build complex operations
- **Reasoning**: Code behavior is easier to understand and predict
- **Debugging**: Isolated functions are easier to debug

### Negative

- **Performance**: Copying data instead of mutation can be slower
- **Memory Usage**: Immutable operations may use more memory
- **Learning Curve**: Team needs to adapt to functional thinking
- **Go Limitations**: Go is not a purely functional language

### Mitigation Strategies

- Use functional principles where they provide clear benefits
- Allow controlled mutability in performance-critical sections
- Provide training and code review to reinforce functional patterns
- Use Go's strengths (interfaces, composition) to support functional design

## Implementation Examples

### Transaction Processing Pipeline

```go
type TransactionProcessor func([]models.Transaction) []models.Transaction

func ProcessTransactions(transactions []models.Transaction, processors ...TransactionProcessor) []models.Transaction {
    result := transactions
    for _, processor := range processors {
        result = processor(result)
    }
    return result
}

// Usage
processed := ProcessTransactions(
    rawTransactions,
    ValidateTransactions,
    NormalizeAmounts,
    CategorizeTransactions,
    SortByDate,
)
```

### Error Handling with Functional Patterns

```go
type Result[T any] struct {
    Value T
    Error error
}

func (r Result[T]) Map(f func(T) T) Result[T] {
    if r.Error != nil {
        return r
    }
    return Result[T]{Value: f(r.Value), Error: nil}
}

func (r Result[T]) FlatMap(f func(T) Result[T]) Result[T] {
    if r.Error != nil {
        return r
    }
    return f(r.Value)
}

// Usage
result := ParseFile(filePath).
    Map(ValidateTransactions).
    Map(CategorizeTransactions).
    FlatMap(WriteToCSV)
```

### Configuration as Pure Functions

```go
type Config struct {
    LogLevel    string
    CSVDelimiter string
    AIEnabled   bool
}

func LoadConfig() Config {
    return Config{
        LogLevel:     getEnvWithDefault("LOG_LEVEL", "info"),
        CSVDelimiter: getEnvWithDefault("CSV_DELIMITER", ","),
        AIEnabled:    getBoolEnv("USE_AI_CATEGORIZATION"),
    }
}

func WithLogLevel(config Config, level string) Config {
    return Config{
        LogLevel:     level,
        CSVDelimiter: config.CSVDelimiter,
        AIEnabled:    config.AIEnabled,
    }
}
```

## Boundaries Between Pure and Impure Code

### Pure Code (Core Business Logic)

- Transaction parsing and validation
- Categorization algorithms
- Data transformations
- Calculations and formatting

### Impure Code (I/O and Side Effects)

- File system operations
- Network requests (AI API)
- Logging
- Configuration loading
- Database operations

## Testing Strategy

```go
func TestCategorizeTransaction(t *testing.T) {
    // Pure function testing - no mocks needed
    tests := []struct {
        name     string
        tx       models.Transaction
        rules    []CategoryRule
        expected string
    }{
        {
            name: "grocery transaction",
            tx:   models.Transaction{Description: "MIGROS ZURICH"},
            rules: []CategoryRule{{Pattern: "MIGROS", Category: "Groceries"}},
            expected: "Groceries",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CategorizeTransaction(tt.tx, tt.rules)
            assert.Equal(t, tt.expected, result.Category)
        })
    }
}
```

## Related Decisions

- ADR-001: Parser interface standardization
- ADR-002: Hybrid categorization approach
- ADR-004: Configuration management strategy

## Date

2024-12-19

## Authors

- Development Team
