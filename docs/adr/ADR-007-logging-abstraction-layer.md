# ADR-007: Logging Abstraction Layer

## Status

Accepted

## Context

The codebase previously used a global `logrus.Logger` instance accessed through package-level functions in `internal/logging`. This approach had several drawbacks:

1. **Global Mutable State**: The global logger made testing difficult and created hidden dependencies
2. **Tight Coupling**: Direct dependency on logrus throughout the codebase made it hard to change logging implementations
3. **Testing Challenges**: Tests couldn't easily mock or verify logging behavior without affecting global state
4. **Concurrency Issues**: Global state can lead to race conditions in concurrent code

## Decision

We have implemented a logging abstraction layer with the following components:

### Logger Interface

A `logging.Logger` interface defines the contract for all logging operations:

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

### LogrusAdapter Implementation

A `LogrusAdapter` struct implements the `Logger` interface by wrapping `logrus.Logger`:

- `NewLogrusAdapter(level, format string)` creates a new logger with specified configuration
- `NewLogrusAdapterFromLogger(*logrus.Logger)` wraps an existing logrus instance
- All methods delegate to the underlying logrus logger

### Dependency Injection

Components receive logger instances through constructors rather than accessing global variables:

```go
// Before (anti-pattern)
func NewParser() *Parser {
    return &Parser{
        logger: logging.GetLogger(), // Global access
    }
}

// After (recommended)
func NewParser(logger logging.Logger) *Parser {
    return &Parser{
        logger: logger,
    }
}
```

### Standardized Field Names

Constants in `internal/logging/constants.go` define standard field names for structured logging:

```go
const (
    FieldFile          = "file_path"
    FieldCategory      = "category"
    FieldError         = "error"
    // ... etc
)
```

## Consequences

### Positive

1. **Improved Testability**: Tests can inject mock loggers and verify logging behavior
2. **Decoupling**: Application code depends on the interface, not the implementation
3. **Flexibility**: Easy to swap logging implementations (e.g., switch from logrus to zap or slog)
4. **No Global State**: Eliminates global mutable state and associated concurrency issues
5. **Better Testing**: Each test can have its own logger instance without interference

### Negative

1. **Migration Effort**: Existing code needs to be updated to use dependency injection
2. **Slightly More Verbose**: Constructors need to accept logger parameters
3. **Backward Compatibility**: Deprecated `GetLogger()` function maintained for transition period

### Neutral

1. **Learning Curve**: Developers need to understand the abstraction pattern
2. **Consistency**: Requires discipline to use the interface consistently

## Implementation Notes

### Phase 1 (Complete)

- ✅ Created `logging.Logger` interface
- ✅ Implemented `LogrusAdapter`
- ✅ Added comprehensive unit tests
- ✅ Defined standardized field constants
- ✅ Updated documentation

### Phase 2 (Planned)

- Migrate parsers to use dependency injection
- Update CLI commands to create and inject loggers
- Remove global logger usage throughout codebase
- Eventually deprecate and remove `GetLogger()` function

## References

- Requirements: 6.1, 6.2, 6.3, 6.4, 6.5
- Tasks: 1, 1.1 in `.kiro/specs/code-quality-refactoring/tasks.md`
- Related ADRs: ADR-001 (Parser Interface Standardization)

## Date

2025-11-01
