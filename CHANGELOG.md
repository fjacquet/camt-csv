# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Development Infrastructure**:
  - `CLAUDE.md` - AI-assisted development guidance with updated parser interface documentation
  - `Dockerfile` - Multi-stage Alpine container build for containerized deployments
  - `Makefile` - Development commands (build, test, lint, coverage, security)
  - `codecov.yml` - Code coverage configuration and thresholds

- **Test Coverage Improvements**:
  - Add tests for `cmd/batch` package
  - Add tests for `cmd/common` package with mock parser implementation
  - Add tests for `internal/fileutils` package
  - Add tests for `internal/textutils` package
  - Add tests for `internal/validation` package

### Changed

- Update `CLAUDE.md` to reflect refactored parser interface (segregated interfaces, new factory location)
- Update dependencies: cobra v1.10.2, golang.org/x/net v0.47.0, golang.org/x/sys v0.38.0, golang.org/x/text v0.31.0

### Deprecated

- **Legacy Configuration Functions** (`internal/config/config.go`):
  - `LoadEnv()` - Use `InitializeConfig()` instead
  - `GetEnv()` - Use `Config` struct fields instead
  - `MustGetEnv()` - Use `Config` struct with validation instead
  - `GetGeminiAPIKey()` - Use `Config.AI.APIKey` instead
  - `ConfigureLogging()` - Use `ConfigureLoggingFromConfig()` instead
  - `InitializeGlobalConfig()` - Use `InitializeConfig()` with DI container instead
  - Global `Logger` variable - Use `container.GetLogger()` instead
  - All deprecated functions will be removed in v3.0.0

### Fixed

- Remove redundant `config.LoadEnv()` call in `cmd/categorize/categorize.go`
- Fix unchecked `file.Close()` return values in `internal/fileutils/fileutils_test.go`
- Fix SLSA workflow: update Go version to 1.24, upgrade to slsa-github-generator v2.0.0
- Add missing `.slsa-goreleaser.yml` configuration for SLSA provenance builds

## [2.0.0] - 2025-11-02

### Added

- **Dependency Injection Architecture**: Complete refactoring to use dependency injection pattern
  - New `Container` type for managing all application dependencies
  - Elimination of global mutable state for better testability
  - All parsers now receive dependencies through constructors

- **Interface Segregation for Parsers**: 
  - Segregated parser interfaces (`Parser`, `Validator`, `CSVConverter`, `LoggerConfigurable`)
  - `BaseParser` foundation providing common functionality to all parsers
  - Composition-based architecture eliminating code duplication

- **Framework-Agnostic Logging**:
  - New `logging.Logger` interface for structured logging abstraction
  - `LogrusAdapter` implementation with dependency injection support
  - Structured logging with `Field` type for key-value pairs

- **Transaction Builder Pattern**:
  - Fluent API for constructing complex transactions with validation
  - Type-safe transaction creation with sensible defaults
  - Automatic field population and validation at build time

- **Strategy Pattern for Categorization**:
  - Modular categorization strategies (`DirectMappingStrategy`, `KeywordStrategy`, `AIStrategy`)
  - Priority-based strategy execution with independent testing
  - Extensible architecture for adding new categorization methods

- **Comprehensive Error Handling**:
  - Custom error types (`ParseError`, `ValidationError`, `CategorizationError`, `DataExtractionError`)
  - Detailed error context with parser, field, and value information
  - Proper error wrapping with `fmt.Errorf` and `%w` verb

- **Performance Optimizations**:
  - String operations optimization with `strings.Builder` and pre-allocation
  - Lazy initialization for expensive resources (AI client)
  - Pre-allocated slices and maps with capacity hints

- **Constants-Based Design**:
  - Comprehensive constants in `internal/models/constants.go`
  - Elimination of magic strings and numbers throughout codebase
  - Type-safe transaction directions and status values

- **Enhanced Documentation**:
  - Comprehensive architecture documentation
  - Developer guide with patterns and best practices
  - Migration guide for upgrading from v1.x
  - Godoc comments for all public APIs

### Changed

- **File Naming Convention**: `debitors.yaml` renamed to `debtors.yaml` for standard English spelling
- **Configuration Structure**: Hierarchical YAML structure with nested sections (`log`, `csv`, `ai`)
- **Date Handling**: Internal use of `time.Time` instead of strings with proper CSV marshaling
- **Parser Architecture**: All parsers now embed `BaseParser` and use dependency injection
- **Error Messages**: More detailed and structured error information with custom types

### Removed

- **Global Singleton Functions** (BREAKING CHANGES):
  - `categorizer.GetDefaultCategorizer()` - Use `container.NewContainer()` and access `Categorizer`
  - `categorizer.CategorizeTransaction()` - Use `categorizer.Categorize()` with context
  - `categorizer.UpdateDebitorCategory()` - Use `categorizer.UpdateDebtorMapping()`
  - `categorizer.UpdateCreditorCategory()` - Use `categorizer.UpdateCreditorMapping()`
  - `config.GetGlobalConfig()` - Use `config.LoadConfig()` with dependency injection
  - `config.GetCSVDelimiter()` - Access through configuration object
  - `config.GetLogLevel()` - Access through configuration object
  - `config.IsAIEnabled()` - Access through configuration object
  - `factory.GetParser()` - Use `factory.GetParserWithLogger()` or container
  - `logging.GetLogger()` - Use `logging.NewLogrusAdapter()` with dependency injection

- **Deprecated Methods** (BREAKING CHANGES):
  - `CategoryStore.LoadDebitorMappings()` - Use `LoadDebtorMappings()`
  - `CategoryStore.SaveDebitorMappings()` - Use `SaveDebtorMappings()`
  - `DirectMappingStrategy.UpdateDebitorMapping()` - Use `UpdateDebtorMapping()`
  - `Transaction.GetAmountAsFloat()` - Use `Transaction.Amount.Float64()` or decimal operations
  - `Transaction.GetPayee()` - Access `Transaction.Payee` field directly
  - `Transaction.GetPayer()` - Access `Transaction.Payer` field directly
  - Legacy transaction conversion methods - Use `TransactionBuilder` for new transactions

- **Internal Deprecated Methods**:
  - `Categorizer.categorizeWithGemini()` - Replaced by `AIStrategy`

### Deprecation Timeline

See [DEPRECATION_TIMELINE.md](docs/DEPRECATION_TIMELINE.md) for complete deprecation schedule and migration guidance.

**Current Status (v2.0.0):**
- ✅ Global singleton functions removed
- ⚠️ Transaction backward compatibility methods deprecated (removal in v3.0.0)
- ✅ New dependency injection architecture available
- ✅ TransactionBuilder pattern available
  - `Categorizer.categorizeByCreditorMapping()` - Replaced by `DirectMappingStrategy`
  - `Categorizer.categorizeByDebitorMapping()` - Replaced by `DirectMappingStrategy`
  - `Categorizer.categorizeLocallyByKeywords()` - Replaced by `KeywordStrategy`

### Migration Guide

#### Configuration Migration

**Old configuration** (`~/.camt-csv/config.yaml`):
```yaml
log_level: "info"
csv_delimiter: ","
ai_enabled: true
```

**New configuration**:
```yaml
log:
  level: "info"
  format: "text"
csv:
  delimiter: ","
ai:
  enabled: true
  model: "gemini-2.0-flash"
```

#### Code Migration

**Old code**:
```go
// Global singleton usage (removed)
categorizer := categorizer.GetDefaultCategorizer()
result := categorizer.CategorizeTransaction(tx)
```

**New code**:
```go
// Dependency injection
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}
result, err := container.Categorizer.Categorize(ctx, tx)
```

#### File Migration

```bash
# Rename debtor mapping file
mv database/debitors.yaml database/debtors.yaml
```

### Security

- Improved input validation with custom error types
- Better error message sanitization
- Proper file permissions constants usage

### Performance

- Reduced memory allocations in hot paths
- Optimized string operations with pre-allocation
- Lazy initialization of expensive resources

### Testing

- Achieved 80%+ test coverage
- Mock dependencies for all external interactions
- Integration tests for end-to-end workflows
- Benchmark tests for performance-critical paths

---

## [1.x.x] - Previous Versions

See Git history for changes in previous versions.

### Breaking Changes Summary

This major version (2.0.0) removes all deprecated global singleton functions and methods that were marked for removal. The new architecture is based on dependency injection and provides better testability, maintainability, and performance.

**Key Migration Steps:**
1. Update configuration file structure
2. Rename `debitors.yaml` to `debtors.yaml`
3. Replace global function calls with dependency injection pattern
4. Update error handling to use new custom error types
5. Use `TransactionBuilder` for creating new transactions

For detailed migration instructions, see [docs/migration-guide.md](docs/migration-guide.md).