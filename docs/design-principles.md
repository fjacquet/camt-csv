# Design Principles - CAMT-CSV Project

## Overview

The CAMT-CSV project is built on a foundation of solid software engineering principles that prioritize maintainability, extensibility, and reliability. This document outlines the core design principles that guide the development and evolution of this financial data processing system.

## Core Design Principles

### 1. **Interface-Driven Design with Segregated Interfaces**

**Principle**: All parsers implement segregated interfaces that separate concerns and provide only the capabilities they need.

**Implementation**:

- **Core Interfaces**: `Parser`, `Validator`, `CSVConverter`, `LoggerConfigurable`, `FullParser`
- **BaseParser Foundation**: All parsers embed `BaseParser` struct for common functionality
- **Composition over Inheritance**: Parsers compose interfaces rather than inheriting from large base classes
- **Single Responsibility per Interface**: Each interface has one clear purpose

**Example**:
```go
type MyParser struct {
    parser.BaseParser  // Provides logging and CSV writing
    // parser-specific fields
}
```

**Benefits**:

- Easy to add new financial data formats following established patterns
- Eliminates code duplication through BaseParser
- Consistent API across all parsers with shared functionality
- Simplified testing with common test utilities
- Interface segregation principle compliance

### 2. **Single Responsibility Principle**

**Principle**: Each component has a single, well-defined responsibility.

**Implementation**:

- **Parsers**: Handle format-specific parsing logic
- **Models**: Define data structures and validation
- **Categorizer**: Handles transaction categorization logic
- **Common**: Provides shared utilities and CSV writing
- **Store**: Manages configuration and category storage

**Benefits**:

- Clear separation of concerns
- Easier debugging and testing
- Reduced coupling between components

### 3. **Dependency Injection & Inversion of Control**

**Principle**: Dependencies are injected rather than hard-coded, allowing for flexibility and testability.

**Implementation**:

- **Logger Injection**: All parsers receive logger through `BaseParser` constructor
- **Interface Dependencies**: Components depend on `logging.Logger` interface rather than concrete implementations
- **PDF Extractor Injection**: PDF parser uses `PDFExtractor` interface for testability
- **Categorizer Injection**: Transaction classification through injected categorizer
- **Store Injection**: Configuration management through injected store
- **Test-specific Injection**: Mock dependencies in test suites

**Example**:
```go
func NewMyParser(logger logging.Logger) *MyParser {
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}
```

**Benefits**:

- Improved testability with mock dependencies
- Elimination of global mutable state
- Runtime configuration flexibility
- Cleaner separation between components
- Easier unit testing without shared state
- Consistent dependency management through BaseParser

### 4. **Fail-Fast with Graceful Degradation**

**Principle**: Detect errors early but provide meaningful fallbacks when possible.

**Implementation**:

- Comprehensive input validation at parser entry points
- Early return on invalid file formats
- Graceful handling of malformed data with logging
- Default values for missing optional fields

**Benefits**:

- Better user experience with clear error messages
- System stability under adverse conditions
- Easier debugging with detailed logging

### 5. **Immutable Data Structures**

**Principle**: Core data models are designed to be immutable where possible.

**Implementation**:

- Transaction models with read-only fields after creation
- Decimal types for financial amounts (preventing floating-point errors)
- Configuration objects that don't change after initialization

**Benefits**:

- Thread safety
- Predictable behavior
- Reduced bugs from unexpected state changes

### 6. **Comprehensive Logging & Observability**

**Principle**: All significant operations are logged with appropriate detail levels using a framework-agnostic abstraction.

**Implementation**:

- **Logging Abstraction Layer**: `logging.Logger` interface decouples application from specific frameworks
- **Dependency Injection**: Logger instances injected through constructors via `BaseParser`
- **Structured Logging**: Consistent field names using `logging.Field` struct for key-value pairs
- **Multiple Log Levels**: Debug, Info, Warn, Error, Fatal with appropriate usage
- **Context-Rich Messages**: Metadata using `WithField`, `WithFields`, and `WithError` methods
- **Default Implementation**: `LogrusAdapter` wrapping logrus with JSON and text formatters
- **Test Support**: Mock logger implementations for unit testing

**Example**:
```go
logger.Info("Processing transaction", 
    logging.Field{Key: "file", Value: filename},
    logging.Field{Key: "count", Value: len(transactions)})
```

**Benefits**:

- Easy troubleshooting and debugging with structured data
- Production monitoring capabilities
- Audit trail for financial data processing
- Improved testability with mock loggers
- Flexibility to change logging implementations without modifying business logic
- Consistent logging patterns across all parsers through `BaseParser`

### 7. **Test-Driven Quality Assurance**

**Principle**: Comprehensive testing ensures reliability and prevents regressions.

**Implementation**:

- Unit tests for all parser packages
- Integration tests for end-to-end workflows
- Test data that covers edge cases and error conditions
- Consistent test structure across all packages

**Benefits**:

- High confidence in code changes
- Documentation through test examples
- Regression prevention

### 8. **Configuration Over Convention**

**Principle**: Behavior should be configurable rather than hard-coded.

**Implementation**:

- YAML-based configuration files for categories and mappings
- Configurable CSV delimiters and output formats
- Environment-specific settings
- Runtime parser selection

**Benefits**:

- Adaptability to different environments
- User customization without code changes
- Easy deployment across different setups

### 9. **Error Handling & Recovery**

**Principle**: Errors should be handled gracefully with clear communication to users using standardized error types.

**Implementation**:

- **Custom Error Types**: Comprehensive error types in `internal/parsererror/`
  - `ParseError`: General parsing failures with parser, field, and value context
  - `ValidationError`: Format validation failures with file path and reason
  - `CategorizationError`: Transaction categorization failures with strategy context
  - `InvalidFormatError`: Files not matching expected format with content snippets
  - `DataExtractionError`: Field extraction failures with raw data context
- **Error Wrapping**: Proper error context using `fmt.Errorf` with `%w` verb
- **Error Inspection**: Use of `errors.Is` and `errors.As` for error type checking
- **Graceful Degradation**: Log warnings for recoverable issues, return errors for unrecoverable ones
- **Resource Cleanup**: Proper cleanup in error scenarios with `defer` statements

**Example**:
```go
if err != nil {
    return nil, &parsererror.ParseError{
        Parser: "CAMT",
        Field:  "amount",
        Value:  rawValue,
        Err:    err,
    }
}
```

**Benefits**:

- Better user experience with detailed error context
- System resilience through graceful degradation
- Easier troubleshooting with structured error information
- Consistent error handling patterns across all parsers

### 10. **Performance & Resource Management**

**Principle**: Efficient resource usage and performance optimization where needed.

**Implementation**:

- Streaming file processing for large datasets
- Proper resource cleanup (file handles, memory)
- Efficient data structures (decimal for financial calculations)
- Lazy loading where appropriate

**Benefits**:

- Scalability for large files
- Reduced memory footprint
- Better system resource utilization

## Design Patterns Used

### Strategy Pattern

- Different parser implementations for different file formats
- Pluggable categorization strategies

### Adapter Pattern

- CAMT parser adapter to bridge different XML parsing approaches
- Common interface adaptation for different data sources

### Factory Pattern

- Parser creation based on file type detection
- Configuration object creation

### Template Method Pattern

- Common CSV writing logic with format-specific customizations
- Shared validation patterns with format-specific rules

## Anti-Patterns Avoided

### God Objects

- No single class handles multiple responsibilities
- Clear separation between parsing, validation, and output

### Tight Coupling

- Interfaces used to decouple components
- Dependency injection prevents hard dependencies

### Magic Numbers/Strings

- Constants defined for configuration values
- Enum-like patterns for status codes and types

### Premature Optimization

- Focus on correctness first, then performance
- Profiling-driven optimization decisions

## Evolution Guidelines

### Adding New Parsers

1. **Create Parser Package**: Create `internal/<format>parser/` directory
2. **Embed BaseParser**: Struct should embed `parser.BaseParser` for common functionality
3. **Implement Interfaces**: Implement `parser.Parser` interface (minimum requirement)
4. **Constructor Pattern**: Use `NewMyParser(logger logging.Logger)` constructor accepting logger
5. **Error Handling**: Use custom error types from `internal/parsererror/`
6. **Constants Usage**: Use constants from `internal/models/constants.go` instead of magic strings
7. **Structured Logging**: Use injected logger with structured fields
8. **Testing**: Include comprehensive tests with mock dependencies
9. **Documentation**: Document format-specific considerations and usage examples

**Example Structure**:
```go
type MyParser struct {
    parser.BaseParser
    // format-specific fields
}

func NewMyParser(logger logging.Logger) *MyParser {
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}

func (p *MyParser) Parse(r io.Reader) ([]models.Transaction, error) {
    p.GetLogger().Info("Starting parse operation")
    
    // Use constants instead of magic strings
    transaction.CreditDebit = models.TransactionTypeDebit
    
    // Use custom error types with context
    if err != nil {
        return nil, &parsererror.ParseError{
            Parser: "MyParser",
            Field:  "amount",
            Value:  rawValue,
            Err:    err,
        }
    }
    
    return transactions, nil
}
```

### Extending Functionality

1. Consider impact on existing interfaces
2. Maintain backward compatibility
3. Add configuration options rather than hard-coding behavior
4. Update all relevant documentation

### Performance Improvements

1. Profile before optimizing
2. Maintain correctness while improving performance
3. Consider memory vs. speed trade-offs
4. Document performance characteristics

## Conclusion

These design principles have created a robust, maintainable, and extensible financial data processing system. By adhering to these principles, the codebase remains clean, testable, and adaptable to changing requirements while ensuring the reliability needed for financial data processing.
