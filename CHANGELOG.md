# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.3.0] - 2026-02-21

### Fixed

- Skip output file generation when 0 transactions are parsed (all parsers and modes)

## [2.2.3] - 2026-02-21

### Fixed

- Skip output file generation when 0 transactions are parsed (all parsers and modes)

## [2.2.2] - 2026-02-21

### Changed

- Simplify string normalization: replace hand-rolled ASCII fast-path with `strings.ToLower`/`strings.ToUpper`
- Make CSV writers non-mutating: copy transaction slices before modifying derived fields
- Make strategy constructors pure: `NewDirectMappingStrategy` and `NewKeywordStrategy` accept pre-loaded data instead of performing I/O
- Extract shared `RunConvert` handler and `RegisterFormatFlags` helper for CLI commands (-327 lines)
- Extract shared `ConvertToCSVDefault` into `BaseParser` for adapter DRY (-40 lines)

### Fixed

- Race condition in `MockLogger` lazy initialization using `sync.Once`

### Removed

- Delete dead `models.Parser` interface (replaced by `parser.FullParser`)
- Delete unused `TransactionParty` and `CategorizerTransaction` types from models
- Delete unused `Config` interface, `aiFactory` field, and `SetAIClientFactory` from categorizer
- Delete thin wrapper functions `ExportTransactionsToCSV` and `ExportTransactionsToCSVWithLogger`
- Delete dead `loadMappings` method from `DirectMappingStrategy`

## [2.2.1] - 2026-02-21

### Changed

- Simplify CI: merge duplicate test runs, remove Go 1.23 matrix, use `go-version-file: go.mod`
- Upgrade `actions/setup-go@v4` to `@v5` across all workflows
- Remove duplicate SBOM generation from CI (SLSA workflow handles it for releases)
- Improve test coverage from 64.3% to 70.5%

### Removed

- Dead code cleanup: remove 112+ unreachable functions detected by `deadcode` (golang.org/x/tools)
- Remove unused utility packages: `currencyutils`, `fileutils`, `validation`, `textutils`, `xmlutils`, `factory`
- Remove unused model types: `Money`, `Party` (and their tests)
- Remove `strategy_result.go` from categorizer
- Remove dead ISO20022 methods (`ParseFile`, `extractTransactions`, `entryToTransaction`, `categorizeTransactions`, `ConvertToCSV`, `CreateEmptyCSVFile`) and `concurrent_processor.go`
- Remove dead standalone parser wrappers (`Parse`, `WriteToCSV`, `ConvertToCSV`, `BatchConvert` + `*WithLogger` variants) from all parsers
- Remove dead builder methods (`WithBookkeepingNumber`, `WithAmountFromFloat`, `WithBankTxCode`, `WithFeesFromFloat`, `WithIBAN`, `Reset`, `Clone`)
- Remove dead container methods (`GetParsers`, `GetConfig`, `GetStore`, `GetAIClient`, `Close`)
- Remove dead utility functions from `dateutils`, `git`, `common`, `formatter`, `categorizer`, `parsererror`

## [2.2.0] - 2026-02-17

### Added

- Multi-arch Docker images published to `ghcr.io/fjacquet/camt-csv` on release
- Homebrew tap: `brew tap fjacquet/homebrew-tap && brew install camt-csv`

### Fixed

- Resolve all open gosec code scanning alerts (G117, G204, G702, G703, G704)
- Sanitize git ref inputs to prevent command injection (G204)

## [2.1.0] - 2026-02-17

### Added

- GoReleaser workflow for automated multi-platform releases on tag push
- `--version` flag now shows version, commit, and build date injected via ldflags
- AI categorization staging: when `--auto-learn` is off, AI suggestions are saved to `database/staging_creditors.yaml` and `database/staging_debtors.yaml` for manual review instead of being discarded
- Staging configuration: `staging.enabled`, `staging.creditors_file`, `staging.debtors_file`

### Changed

- Makefile LDFLAGS aligned with GoReleaser: `main.version`, `main.commit`, `main.date`
- Dockerfile updated with version ldflags injection and Alpine 3.21

### Removed

- Unused AI tool configs: `.specify/`, `.kiro/`, `.gemini/`, `specs/`

## [2.0.0] - 2026-02-17

### Removed

- **Deprecated Transaction methods**: `GetPayee()`, `GetPayer()`, `GetAmountAsFloat()`, `GetDebitAsFloat()`, `GetCreditAsFloat()`, `GetFeesAsFloat()`, `ToBuilder()`, `SetPayerInfo()`, `SetPayeeInfo()`, `SetAmountFromFloat()` — use `GetCounterparty()`, `*AsDecimal()` variants, and `TransactionBuilder` instead
- **Deprecated process functions**: `ProcessFileLegacy()`, `ProcessFileLegacyWithError()`, `SaveMappings()` — use `ProcessFileWithError()` and container-based categorizer
- **Deprecated root command function**: `GetConfig()` — use `GetContainer().GetConfig()`
- **Deprecated mock logger field**: `Entries` on `MockLogger` — use `GetEntries()`
- **Deprecated XPath constants**: `Camt053XPaths` var — use `DefaultCamt053XPaths()`

### Changed

- **README.md slimmed down** from ~287 lines to ~107 lines; detailed content moved to docs site
- **Documentation restructured**: 7 stale docs deleted (3 migration guides, deprecation timeline, 3 meta-docs), migration content consolidated into developer guide
- **All documentation updated** to reflect current codebase:
  - `api-specifications.md`: Corrected Transaction struct, Categorizer interface, added OutputFormatter
  - `architecture.md`: Added SemanticStrategy (4th categorization tier), Formatter system, context.Context
  - `design-principles.md`: Added Registry Pattern, updated method signatures
  - `developer-guide.md`: Fixed interface signatures, added v1.x→v2.0.0 migration guide
  - `user-guide.md`: Complete configuration reference with all flags, env vars, and YAML options
  - `testing-strategy.md`: Removed stale `SetTestCategoryStore()` refs, added DI-based testing patterns
  - `operations.md`: Removed fictional Prometheus/health check sections, updated CI/CD pipeline
  - `coding-standards.md`: Added context.Context convention, strategy pattern reference

## [1.3.0] - 2026-02-16

### Changed

- **Standard CSV Format Trimmed to 29 Columns** (Phase 10): Remove 6 redundant/dead fields from standard CSV output
  - Remove BookkeepingNumber (never populated by any parser)
  - Remove IsDebit/DebitFlag (redundant with CreditDebit field)
  - Remove Debit (derived from Amount + CreditDebit)
  - Remove Credit (derived from Amount + CreditDebit)
  - Remove Recipient (redundant with Name/PartyName)
  - Remove AmountTax (never populated by any parser)
  - Update MarshalCSV/UnmarshalCSV for 29-column format
  - Update StandardFormatter header from 35 to 29 columns
  - Fix hardcoded header in common/csv.go WriteTransactionsToCSVWithLogger

### Added

- **End-to-End Format Tests** (Phase 11): Integration tests verifying both output formats
  - TestEndToEndConversion_StandardFormat validates 29-column standard CSV output
  - TestEndToEndConversion_iComptaFormat validates 10-column semicolon-delimited output
  - Explicit 29-column assertions in TestCrossParserConsistency

## [1.2.0] - 2026-02-16

### Added

- **Output Formatter Framework** (Phase 5): Pluggable CSV output formatting with strategy pattern
  - `OutputFormatter` interface with `StandardFormatter` (29-column, comma-delimited) and `iComptaFormatter` (10-column, semicolon-delimited, dd.MM.yyyy dates)
  - `FormatterRegistry` for managing formatters by name
  - `--format` CLI flag on all parser commands (camt, pdf, revolut, selma, debit, revolut-investment)
  - `ProcessFile()` refactored to use `Parse()` + `WriteTransactionsToCSVWithFormatter` pipeline
  - DI container exposes `FormatterRegistry` for consistent formatter access

- **Product Field** (Phase 6): Transaction model expanded from 34 to 35 columns
  - New `Product` field (Current/Savings) positioned after Currency
  - Builder pattern updated with `WithProduct()` method
  - All formatters and CSV writers updated for 35-column output

- **Revolut Investment Parser Enhancements** (Phase 6): Complete transaction type coverage
  - SELL transaction support (credit/incoming money from sales)
  - CUSTODY_FEE transaction support (debit/outgoing fees with fee tracking)
  - Batch conversion mode for investment CSV files

- **Revolut Parser Field Population** (Phase 6): Full 35-field standardized output
  - All transaction fields populated via builder pattern
  - Exchange transactions preserve OriginalAmount/OriginalCurrency metadata
  - Product field populated from source data
  - REVERTED and PENDING transactions logged when skipped

- **Batch Infrastructure** (Phase 7): Universal batch processing with error reporting
  - Reusable `BatchProcessor` using composition pattern (wraps any parser)
  - `BatchManifest` with JSON serialization and semantic exit codes (0=all success, 1=partial, 2=all failed)
  - PDF parser `--batch` flag for individual file conversion mode
  - All 6 CLI commands generate `.manifest.json` and exit with semantic codes
  - Batch processing continues after individual file failures

- **AI Safety Controls** (Phase 8): Safety gates for AI categorization
  - `--auto-learn` flag controls auto-save of AI categorizations (default: OFF)
  - Gemini API rate limiting via token bucket (configurable RPM, default 10)
  - Exponential backoff with jitter for transient API failures (429, 503, timeouts)
  - Confidence metadata on all categorizations (direct=1.0, keyword=0.95, semantic=0.90, AI=0.8-0.9)
  - Pre-save audit logging with confidence scores and source strategy

### Previously added

- **PDF Directory Consolidation**: The `pdf` command now accepts a directory as input, consolidating all PDF files into a single CSV output
  - Single file mode: `camt-csv pdf -i file.pdf -o output.csv` (existing behavior)
  - Directory mode: `camt-csv pdf -i pdf_dir/ -o consolidated.csv` (new feature)
  - Automatically detects and processes all PDF files in the specified directory
  - Consolidates all transactions from multiple PDFs into a single chronologically-sorted CSV
  - Failed PDFs are skipped with warning logs (graceful degradation)
  - Context cancellation support for interrupting long operations with Ctrl+C
  - Validation flag (`--validate`) applies to each PDF file individually
  - Case-insensitive PDF file detection (.pdf, .PDF, .Pdf all supported)
  - No source file metadata in output - pure transaction data for clean consolidation

- **Development Infrastructure**:

  - `CLAUDE.md` - AI-assisted development guidance with coding principles (KISS, DRY, FP)
  - `Dockerfile` - Multi-stage Alpine container build for containerized deployments
  - `Makefile` - Development commands (build, test, lint, coverage, security)
  - `codecov.yml` - Code coverage configuration and thresholds
  - `plan.md` - Senior architect review with action plan and production readiness checklist

- **Documentation**:

  - CAMT.053 ISO 20022 format documentation in CLAUDE.md
  - Coding principles: KISS, DRY, Functional Programming guidelines
  - Dependency injection patterns and interface design guidelines

- **Category YAML Backup System**: Automatic timestamped backups of category mapping files before auto-learn overwrites
  - Backup enabled by default (`backup.enabled: true` in config)
  - Configurable backup directory (defaults to same directory as original file)
  - Configurable timestamp format (default: `YYYYMMDD_HHMMSS`)
  - Atomic behavior: failed backup prevents save, protecting original file
  - Supports both creditor and debtor mapping files

- **Test Coverage Improvements**:
  - Add tests for `cmd/batch` package
  - Add tests for `cmd/common` package with mock parser implementation
  - Add tests for `internal/fileutils` package
  - Add tests for `internal/textutils` package
  - Add tests for `internal/validation` package
  - Add nil container error verification tests for camt, debit, and pdf commands
  - Add 8 concurrent processing edge case tests: context cancellation (before, during, inflight), race conditions, partial result data integrity
  - Add 14 PDF Viseca format detection edge case tests: partial markers, false positives, ambiguous formats
  - Add 20+ error message validation tests across all parsers (CAMT, Debit, Revolut, Selma, PDF) verifying file path and field context
  - Add 5 category backup tests: backup creation, custom location, failure prevention, disabled mode, multiple timestamps
  - Add `MockLogger.VerifyFatalLog()` and `VerifyFatalLogWithDebug()` helper methods for test verification

### Removed

- **Deprecated Config Functions**: Remove `LoadEnv()`, `GetEnv()`, `MustGetEnv()`, `GetGeminiAPIKey()`, `ConfigureLogging()`, `InitializeGlobalConfig()` from `internal/config/config.go`; all configuration now flows through Viper/Container
- **Global Mutable State**: Remove `Logger`, `globalConfig`, and `sync.Once` globals from config package; all state flows through DI container
- **Fallback Categorizer Creation**: Remove silent fallback in `PersistentPostRun` that bypassed dependency injection; nil container now logs warning and returns early

### Security

- **No Credential Logging**: API key values never appear in log output at any level; only presence/absence is logged
- **Secure Temp Files**: All temporary files use `os.CreateTemp()` with random naming; no predictable temp file paths
- **File Permission Standardization**: Non-secret files (YAML category mappings, CSV output) use 0644; secrets use 0600; directories use 0750

### Changed

- **PDF Parser Temp File Consolidation**: Replace individual temp file with single temp directory (`os.MkdirTemp`) for all PDF processing; cleanup uses single `os.RemoveAll` call
- **PDF Parser ExtractText Optimization**: Eliminate duplicate `ExtractText` call (was called twice: once for validation, once for extraction)
- **Categorize Command Init**: Replace `panic(err)` with graceful error handling; Cobra framework handles missing required flags at runtime

### Fixed

- **PDF Debug File Leak**: Remove `debug_pdf_extract.txt` file that accumulated in working directory after PDF parsing
- **PDF Context Propagation**: `Parse()` and `ParseWithExtractor()` now accept and propagate `context.Context` instead of discarding it for `context.Background()`
- **PDF Temp File Cleanup**: Consolidate two separate defer blocks into single close-then-remove defer for correct cleanup ordering
- **MockLogger State Isolation**: `WithError()` and `WithFields()` now create properly isolated instances using shared pointer pattern; tests can verify specific log messages at correct levels
- **Context Propagation**: Fix context propagation throughout application for proper cancellation support
  - CLI commands now extract context from `cmd.Context()` and propagate through all layers
  - Parser interfaces now accept `context.Context` parameter for cancellation and timeout support
  - `TransactionCategorizer.Categorize()` now accepts `context.Context` for AI operations
  - `ConcurrentProcessor` properly handles context cancellation with partial result returns
  - Fix index out of range panic in `ConcurrentProcessor.processConcurrent()` when context cancelled
  - Add context cancellation tests for sequential and concurrent processing paths
  - Enables graceful shutdown with Ctrl+C in batch operations
- **Race Condition in DirectMappingStrategy**: Fix race condition in `ReloadMappings()` method
  - Build new mappings outside lock to eliminate vulnerability window
  - Atomic pointer swap ensures readers never see empty maps during reload
  - Add concurrent test verifying no race conditions with `-race` detector
  - All tests pass with race detector: `make test-race` clean
- **Batch Command Categorization**: Fix batch command not categorizing transactions
  - Batch command now uses DI container instead of factory directly
  - Transactions are now properly categorized using all 3 tiers (direct mapping, keyword, AI)

### Changed

- **BREAKING: Parser Interfaces**: Add `context.Context` parameter to all parser methods
  - `Parser.Parse()` now requires `context.Context` as first parameter
  - `CSVConverter.ConvertToCSV()` now requires `context.Context` as first parameter
  - `BatchConverter.BatchConvert()` now requires `context.Context` as first parameter
  - Custom parser implementations must be updated to accept context
- **BREAKING: TransactionCategorizer Interface**: Add `context.Context` parameter to categorization
  - `TransactionCategorizer.Categorize()` now requires `context.Context` as first parameter
  - Enables proper cancellation propagation through AI categorization calls
- **BatchConverter Interface**: Add `BatchConverter` interface to `parser.FullParser`
  - Follows Interface Segregation Principle (ISP)
  - Enables batch conversion through the DI container
- Update `CLAUDE.md` to reflect refactored parser interface (segregated interfaces, new factory location)
- Update dependencies: cobra v1.10.2, golang.org/x/net v0.47.0, golang.org/x/sys v0.38.0, golang.org/x/text v0.31.0
- **Type-safe Categorizer Interface**: Replace `interface{}` with `models.TransactionCategorizer` for compile-time type safety
  - Add `TransactionCategorizer` interface to models package
  - Update `CategorizerConfigurable` interface to use type-safe signature
  - Modify `Categorize` method to include auto-learning functionality
- **Immutable Container**: Make `Container` struct fields private for immutability
  - All fields now accessed through getter methods only
  - Add `GetParsers()` method returning a copy of the parser map
  - Prevents accidental modification after initialization

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
