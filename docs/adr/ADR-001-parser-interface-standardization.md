# ADR-001: Parser Interface Standardization

## Status

Accepted

## Context

The CAMT-CSV project supports multiple financial file formats (CAMT.053 XML, PDF statements, Revolut CSV, Selma CSV). Initially, each parser had different method signatures and behaviors, making it difficult to:

1. Add new parsers consistently
2. Test parsers uniformly
3. Maintain code quality across parsers
4. Provide consistent user experience

## Decision

We will standardize all parsers to implement a common `Parser` interface:

```go
type Parser interface {
    ParseFile(filePath string) ([]models.Transaction, error)
    ValidateFormat(filePath string) (bool, error)
    ConvertToCSV(inputFile, outputFile string) error
    WriteToCSV(transactions []models.Transaction, csvFile string) error
    SetLogger(logger *logrus.Logger)
}
```

## Consequences

### Positive

- **Consistency**: All parsers behave identically from the caller's perspective
- **Testability**: Uniform test patterns across all parsers
- **Maintainability**: Easier to add new parsers following established patterns
- **Polymorphism**: Can treat all parsers uniformly in the CLI layer
- **Documentation**: Single interface to document and understand

### Negative

- **Refactoring Cost**: Required significant changes to existing parsers
- **Interface Constraints**: Some parser-specific optimizations may be limited
- **Backward Compatibility**: Breaking changes to internal APIs

### Mitigation Strategies

- Implemented changes incrementally, one parser at a time
- Maintained comprehensive test coverage during refactoring
- Used adapter pattern where necessary to fit existing code into new interface

## Implementation Notes

- All parsers now validate format before parsing
- Error handling is consistent across parsers
- Logging is standardized using structured logging with context
- CSV output format is unified across all parsers

## Related Decisions

- ADR-002: Hybrid categorization approach
- ADR-003: Functional programming adoption

## Date

2024-12-19

## Authors

- Development Team
