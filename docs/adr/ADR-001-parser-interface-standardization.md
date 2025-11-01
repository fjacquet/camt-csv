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

We will standardize all parsers using a segregated interface architecture with a common foundation:

**Core Interfaces:**
```go
type Parser interface {
    Parse(r io.Reader) ([]models.Transaction, error)
}

type Validator interface {
    ValidateFormat(r io.Reader) error
}

type CSVConverter interface {
    ConvertToCSV(transactions []models.Transaction, csvFile string) error
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

**BaseParser Foundation:**
All parsers embed a `BaseParser` struct that provides common functionality:
- Logger management (implements `LoggerConfigurable`)
- CSV writing capability (implements `CSVConverter`)
- Consistent initialization patterns

**Implementation Pattern:**
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
```

## Consequences

### Positive

- **Consistency**: All parsers behave identically from the caller's perspective
- **Code Reuse**: BaseParser eliminates duplication of common functionality (logging, CSV writing)
- **Testability**: Uniform test patterns across all parsers with shared test utilities
- **Maintainability**: Easier to add new parsers following established patterns
- **Polymorphism**: Can treat all parsers uniformly in the CLI layer
- **Segregated Interfaces**: Parsers can implement only the interfaces they need
- **Documentation**: Clear interface contracts to document and understand

### Negative

- **Refactoring Cost**: Required significant changes to existing parsers
- **Interface Constraints**: Some parser-specific optimizations may be limited
- **Backward Compatibility**: Breaking changes to internal APIs

### Mitigation Strategies

- Implemented changes incrementally, one parser at a time
- Maintained comprehensive test coverage during refactoring
- Used adapter pattern where necessary to fit existing code into new interface

## Implementation Notes

- All parsers embed `BaseParser` for common functionality
- Segregated interfaces allow parsers to implement only needed capabilities
- Error handling uses custom error types (`InvalidFormatError`, `DataExtractionError`)
- Logging is standardized using abstraction layer with structured logging
- CSV output format is unified across all parsers via shared `WriteToCSV` method
- Dependency injection pattern used for testability (e.g., PDF extractor interface)

## Related Decisions

- ADR-002: Hybrid categorization approach
- ADR-003: Functional programming adoption

## Date

2024-12-19

## Authors

- Development Team
