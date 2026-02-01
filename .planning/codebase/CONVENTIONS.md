# Coding Conventions

**Analysis Date:** 2026-02-01

## Naming Patterns

**Files:**
- Package files named `{name}.go` (e.g., `adapter.go`, `categorizer.go`)
- Test files use `*_test.go` suffix (e.g., `adapter_test.go`)
- Internal test files use `*_internal_test.go` suffix for white-box testing
- Parser adapters consistently named `adapter.go` within each parser package
- Directories for specialized parsers: `{name}parser/` (e.g., `camtparser/`, `pdfparser/`, `debitparser/`)

**Functions:**
- PascalCase for exported functions: `NewAdapter()`, `Parse()`, `ConvertToCSV()`, `ValidateFormat()`
- camelCase for unexported functions: `extractPartyNameFromDescription()`, `cleanPaymentMethodPrefixes()`, `setTransactionTypeFromDescription()`
- Helper functions prefixed with `is` or `set` for predicates/setters: `isIBANFormat()`, `isASCII()`
- Command handler functions use `Func` suffix: `pdfFunc()`, `batchFunc()`

**Variables:**
- camelCase for local variables and parameters: `inputDir`, `outputDir`, `xmlData`, `partyName`
- UPPERCASE for constants: `testXMLContent`, `FieldFile`, `FieldParser`
- Single-letter variables only in loops and tight scopes: `for i := range entries`, `func(entry *models.Entry)`
- Receiver names are single letters: `func (a *Adapter)`, `func (c *Categorizer)`, `func (p *ConcurrentProcessor)`

**Types:**
- PascalCase for exported types: `Transaction`, `Category`, `Adapter`, `Categorizer`, `Logger`
- Struct field names exported as PascalCase: `Amount`, `Currency`, `PartyName`, `RemittanceInfo`
- Interface names end in `-er`: `Parser`, `Validator`, `Logger`, `Categorizer`
- Type aliases for constants: `type TransactionType string`, `type FindingStatus string`

**Package Names:**
- All lowercase, no hyphens: `camtparser`, `categorizer`, `logging`, `common`
- Descriptive plural for utility packages: `models`, `parser` (as interface package)
- Specialized prefixes for related functionality: `{format}parser` pattern

## Code Style

**Formatting:**
- Tool: `gofmt` (enabled in golangci-lint config)
- Line width: Default Go conventions (no hard limit enforced)
- Indentation: Tabs (Go standard)

**Linting:**
- Tool: `golangci-lint` v2 with `.golangci.yml`
- Active linters: `errcheck`, `ineffassign`, `unused`, `gosec`
- Disabled linters: `staticcheck`, `govet`, `misspell`
- Security scanner: `gosec` with whitelist for false positives (G101, G204, G304)

**Error Wrapping:**
- Use `fmt.Errorf("...: %w", err)` for error wrapping to preserve error chain
- Custom error types defined as: `var ErrInvalidFormat = errors.New("file is not in a valid format")`
- Error handling in Parse methods returns sentinel errors for validation
- Log errors before wrapping: `a.GetLogger().WithError(err).Warn("Failed to...")`

## Import Organization

**Order:**
1. Standard library: `import ("context", "encoding/xml", "fmt", "io", "os", ...)`
2. External packages: `import ("github.com/...", "golang.org/x/...")`
3. Internal packages: `import ("fjacquet/camt-csv/cmd/...", "fjacquet/camt-csv/internal/...")`

**Path Aliases:**
- No aliases used; full import paths always used
- Subdirectories referenced as `fjacquet/camt-csv/cmd/...` or `fjacquet/camt-csv/internal/...`

**Common Imports Pattern:**
```go
// Typical file structure
package {name}

import (
    // Standard lib
    "context"
    "errors"
    "fmt"
    "io"
    "os"

    // Internal
    "fjacquet/camt-csv/internal/logging"
    "fjacquet/camt-csv/internal/models"
    "fjacquet/camt-csv/internal/parser"

    // External
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

## Error Handling

**Patterns:**
- Functions return `(result, error)` tuple: `Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error)`
- Errors wrapped with context: `return nil, fmt.Errorf("error reading from reader: %w", err)`
- Sentinel errors defined at package level: `var ErrInvalidFormat = errors.New("...")`
- Error type checking with `errors.Is()`: `if err := p.ValidateFormat(inputFile); err != nil`
- Fallback transaction creation on parse error (not panic):
  ```go
  fallback, _ := models.NewTransactionBuilder().
      WithDatetime(parsedBookingDate).
      Build()
  transaction = fallback
  ```
- Errors logged with context fields before being returned

**Log Error Style:**
```go
a.GetLogger().WithError(err).Warn("Failed to categorize transaction",
    logging.Field{Key: "party", Value: catPartyName})
```

## Logging

**Framework:** Logrus adapter via `internal/logging/logrus_adapter.go`

**Logger Interface:**
- Use injected logger from DI container: `container.GetLogger()`
- All packages accept logger via constructor or setter: `NewAdapter(logger logging.Logger)`
- Logger interface defines: `Debug()`, `Info()`, `Warn()`, `Error()`, `Fatal()`, `WithError()`, `WithField()`, `WithFields()`

**Structured Field Names (Constants):**
- Defined in `internal/logging/constants.go`
- Examples: `FieldFile`, `FieldParser`, `FieldTransactionID`, `FieldCategory`, `FieldCount`
- Custom fields created as: `logging.Field{Key: "custom_key", Value: someValue}`

**Patterns:**
- Info level for successful operations: `a.GetLogger().Info("Writing transactions to CSV file", logging.Field{Key: "count", Value: len(transactions)})`
- Warn level for recoverable issues: `a.GetLogger().Warn("Failed to close file", logging.Field{Key: "error", Value: err})`
- Error level for significant issues: `a.GetLogger().WithError(err).Error("Failed to convert file", logging.Field{Key: "file", Value: inputFile})`
- Debug level for detailed tracing: `a.GetLogger().WithFields(...).Debug("Transaction categorized successfully")`
- Fatal level for unrecoverable errors: `log.Fatalf("%v", err)`

## Comments

**When to Comment:**
- Document exported types and functions: `// Adapter implements the models.Parser interface for CAMT.053 XML files.`
- Explain non-obvious logic in functions
- Mark intentional bypasses of lint rules: `// #nosec G304 -- CLI tool requires user-provided file paths`
- Document deprecated functionality with migration instructions:
  ```go
  // Deprecated: Use container.GetLogger() instead for dependency injection.
  // This will be removed in v3.0.0.
  ```
- Explain complex parsing logic for XML structures

**Comment Style:**
- Single-line comments for simple explanations: `// Extract PartyName from Description if it starts with specific prefixes`
- Multi-line comments for complex logic
- Package-level comments placed at package declaration:
  ```go
  // Package categorizer provides functionality to categorize transactions using multiple methods:
  // 1. Direct seller-to-category mapping from a YAML database
  // 2. Local keyword-based categorization from rules in a YAML file
  // 3. AI-based categorization using Gemini model as a fallback
  ```

**Documentation Comments:**
- Exported functions documented with GoDoc style:
  ```go
  // NewLogrusAdapter creates a new LogrusAdapter with the specified log level and format.
  //
  // Parameters:
  //   - level: Log level as string ("debug", "info", "warn", "error")
  //   - format: Log format as string ("json" or "text")
  //
  // Returns a Logger interface implementation backed by logrus.
  func NewLogrusAdapter(level, format string) Logger {
  ```

## Function Design

**Size:** Small, single-responsibility functions preferred
- Complex parsing logic broken into helpers: `extractPartyNameFromDescription()`, `cleanPaymentMethodPrefixes()`
- Average function length: 20-80 lines with clear cohesion

**Parameters:**
- Use context as first parameter for cancellation support: `func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error)`
- Avoid parameter objects; use explicit parameters for clarity
- Use builder pattern for complex object construction: `models.NewTransactionBuilder().WithAmount(...).WithPartyName(...).Build()`

**Return Values:**
- Always return errors as last value: `(result, error)`
- Multiple return values used for result + error pairs
- Use named return values only when useful for documentation
- Return empty slices, not nil: `return []Transaction{}, nil` instead of `nil`

## Module Design

**Exports:**
- Only public interfaces and essential structs exported
- Unexported helpers kept private to packages
- Factory functions use `New` prefix: `NewAdapter()`, `NewCategorizer()`, `NewContainer()`

**Barrel Files:**
- Not used; each package imports specific types directly
- No `__init__.go` pattern

**Dependency Injection:**
- All external dependencies injected through constructors or setters
- Global state minimized; use container for singletons
- Interfaces used for dependencies: `Parser`, `Logger`, `Categorizer`
- Setters for optional configuration: `SetLogger()`, `SetCategorizer()`

**Interface Segregation:**
- Small, focused interfaces composed when needed
- Example from `internal/parser/parser.go`:
  ```go
  type Parser interface {
      Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error)
  }
  type Validator interface {
      ValidateFormat(filePath string) (bool, error)
  }
  type FullParser interface {
      Parser
      Validator
      CSVConverter
      LoggerConfigurable
      CategorizerConfigurable
      BatchConverter
  }
  ```

## Concurrency

**Patterns:**
- Context-based cancellation for all long-running operations
- Worker pool pattern for concurrent processing: `ConcurrentProcessor` with `runtime.NumCPU()` workers
- Sequential processing threshold (100 entries) for small datasets; concurrent for larger
- Atomic counters for thread-safe progress tracking: `sync/atomic.AddInt64()`

**Locking:**
- Sync.Mutex used in critical sections (e.g., debtor/creditor map updates)
- Defer-based unlock pattern: `defer c.lock.Unlock()` after `c.lock.Lock()`

## Deprecation

**Pattern:**
- Mark deprecated items with comment and removal version
- Provide migration guidance in deprecation notice
- Example:
  ```go
  // Deprecated: Use InitializeConfig() with dependency injection instead.
  // This function will be removed in v3.0.0.
  //
  // Migration:
  //   // Old: logger := config.ConfigureLogging()
  //   // New: cfg, _ := config.InitializeConfig()
  func ConfigureLogging() *logrus.Logger {
  ```

---

*Convention analysis: 2026-02-01*
