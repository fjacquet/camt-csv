# ADR-008: Logging Abstraction Layer Implementation

## Status

Accepted - Implemented

## Context

The codebase previously had direct dependencies on the logrus logging framework throughout the application, making it difficult to:

1. Test components that perform logging operations
2. Change logging frameworks in the future
3. Provide consistent logging patterns across parsers
4. Inject different logging configurations for different components

This tight coupling to a specific logging framework violated dependency inversion principles and made the codebase less maintainable and testable.

## Decision

We have implemented a comprehensive logging abstraction layer with the following components:

### 1. Logger Interface (`internal/logging/logger.go`)

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithError(err error) Logger
    WithField(key string, value interface{}) Logger
    WithFields(fields ...Field) Logger
    Fatal(msg string, fields ...Field)
    Fatalf(msg string, args ...interface{})
}

type Field struct {
    Key   string
    Value interface{}
}
```

### 2. LogrusAdapter Implementation (`internal/logging/logrus_adapter.go`)

- Wraps logrus.Logger to implement our Logger interface
- Provides constructor functions for different configurations
- Supports both JSON and text formatting
- Includes backward compatibility methods

### 3. Dependency Injection Through BaseParser

- All parsers embed `BaseParser` which manages logger injection
- Consistent logger access through `GetLogger()` method
- Constructor pattern ensures proper logger initialization

### 4. Structured Logging Pattern

- Use of `Field` struct for key-value pairs
- Consistent field naming across the application
- Context-rich log messages with relevant metadata

## Consequences

### Positive

1. **Improved Testability**: Components can be tested with mock loggers
2. **Framework Independence**: Can change logging frameworks without modifying business logic
3. **Consistent Patterns**: All parsers use the same logging approach through BaseParser
4. **Structured Logging**: Consistent field-based logging across the application
5. **Dependency Injection**: Clean separation of concerns with injected dependencies
6. **Backward Compatibility**: Existing logrus usage patterns still work during migration

### Negative

1. **Additional Abstraction**: Slight increase in code complexity
2. **Performance Overhead**: Minimal overhead from interface calls
3. **Migration Effort**: Gradual migration of existing direct logrus usage

## Implementation Details

### Parser Integration

All parsers now embed BaseParser and receive logger through constructor:

```go
type MyParser struct {
    parser.BaseParser
}

func NewMyParser(logger logging.Logger) *MyParser {
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}

func (p *MyParser) Parse(r io.Reader) ([]models.Transaction, error) {
    p.GetLogger().Info("Starting parse operation",
        logging.Field{Key: "parser", Value: "MyParser"})
    // implementation
}
```

### Testing Support

Mock logger implementation for testing:

```go
type MockLogger struct {
    Entries []LogEntry
}

func (m *MockLogger) Info(msg string, fields ...logging.Field) {
    m.Entries = append(m.Entries, LogEntry{
        Level:   "INFO",
        Message: msg,
        Fields:  fields,
    })
}
```

### Configuration

Logger creation with different configurations:

```go
// JSON format for production
logger := logging.NewLogrusAdapter("info", "json")

// Text format for development
logger := logging.NewLogrusAdapter("debug", "text")
```

## Alternatives Considered

1. **Direct Logrus Usage**: Rejected due to tight coupling and testing difficulties
2. **Standard Library log**: Rejected due to lack of structured logging support
3. **Other Logging Frameworks**: Considered but logrus provides good balance of features and performance

## Related ADRs

- ADR-001: Parser Interface Standardization
- ADR-003: Functional Programming Adoption
- ADR-004: Configuration Management Strategy

## References

- Go logging best practices
- Dependency injection patterns in Go
- Interface segregation principle
- Clean Architecture principles