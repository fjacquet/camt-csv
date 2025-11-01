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

- Logging abstraction layer (`logging.Logger` interface) decouples application from specific frameworks
- Dependency injection of logger instances through constructors
- Structured logging with consistent field names from `internal/logging/constants.go`
- Different log levels (Debug, Info, Warn, Error) for different scenarios
- Context-rich log messages with relevant metadata using `WithField`, `WithFields`, and `WithError`
- Default implementation uses `LogrusAdapter` wrapping logrus

**Benefits**:

- Easy troubleshooting and debugging
- Production monitoring capabilities
- Audit trail for financial data processing
- Improved testability with mock loggers
- Flexibility to change logging implementations without modifying business logic

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

**Principle**: Errors should be handled gracefully with clear communication to users.

**Implementation**:

- Custom error types for different failure scenarios
- Detailed error messages with context
- Partial success handling (process what you can)
- Resource cleanup in error scenarios

**Benefits**:

- Better user experience
- System resilience
- Easier troubleshooting

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
5. **Error Handling**: Use custom error types (`InvalidFormatError`, `DataExtractionError`)
6. **Testing**: Include comprehensive tests with mock dependencies
7. **Documentation**: Document format-specific considerations and usage examples

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
    // implementation
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
