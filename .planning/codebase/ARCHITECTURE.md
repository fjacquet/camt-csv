# Architecture

**Analysis Date:** 2026-02-01

## Pattern Overview

**Overall:** Layered architecture with dependency injection container and interface-segregated components

**Key Characteristics:**
- Segregated parser interfaces following Interface Segregation Principle (ISP)
- Dependency injection via centralized Container for wiring dependencies
- Strategy pattern for transaction categorization with multiple strategies applied in priority order
- Factory pattern for creating and managing parser instances
- Command pattern via Cobra CLI for exposing functionality
- Context propagation throughout for cancellation and timeout support

## Layers

**Presentation (CLI):**
- Purpose: Expose application functionality through command-line interface
- Location: `cmd/` directory with subcommands (camt, pdf, revolut, selma, debit, revolut-investment, batch, categorize, etc.)
- Contains: Cobra commands, flag definitions, command handlers
- Depends on: Container, Parser interfaces, Common processing logic
- Used by: End users through CLI

**Application/Orchestration:**
- Purpose: Coordinate execution of parsers and handle common processing workflows
- Location: `cmd/common/process.go` and command handlers
- Contains: File processing pipelines, validation logic, error handling orchestration
- Depends on: Parser interfaces, Logger, Categorizer
- Used by: CLI commands

**Parser Layer:**
- Purpose: Transform various input formats into standardized Transaction model
- Location: `internal/{format}parser/` packages (camtparser, pdfparser, revolutparser, selmaparser, debitparser, revolutinvestmentparser)
- Contains: Format-specific parsing logic, validation, CSV conversion
- Depends on: Models (Transaction), Logger, Categorizer, Parser interfaces
- Used by: Application layer and Container

**Categorization Layer:**
- Purpose: Apply multi-strategy transaction categorization with auto-learning
- Location: `internal/categorizer/`
- Contains: Strategy implementations (DirectMapping, Keyword, AI/Semantic), categorizer orchestrator
- Depends on: Models, Store, Logger, AI client (optional)
- Used by: Parsers during transaction processing

**Persistence Layer:**
- Purpose: Load and save category configurations and mappings
- Location: `internal/store/store.go`
- Contains: YAML file I/O for categories.yaml, creditors.yaml, debtors.yaml
- Depends on: Models, File system
- Used by: Categorizer, Container initialization

**Configuration Layer:**
- Purpose: Load and manage application configuration from files, environment, CLI flags
- Location: `internal/config/` with Viper-based hierarchical configuration
- Contains: Config struct, Viper initialization, logging configuration
- Depends on: Viper library, environment variables
- Used by: Container, root command initialization

**Support Layers:**
- Logging: `internal/logging/` - LogrusAdapter abstraction for structured logging
- Models: `internal/models/` - Core Transaction and Category data structures
- Utilities: `internal/{dateutils, currencyutils, textutils, validation, xmlutils, fileutils}` - Reusable helpers
- Error Handling: `internal/parsererror/` - Custom error types with context

## Data Flow

**File Conversion Flow:**

1. **User invokes command** → `camt --input file.xml --output file.csv`
2. **root.Init()** → Loads configuration (Viper), initializes Container with DI
3. **Command handler** → Gets parser from container: `appContainer.GetParser(container.CAMT)`
4. **Process file** → Calls `parser.ConvertToCSV(ctx, inputFile, outputFile)`:
   - Parse input file to Transaction slice
   - For each transaction, call categorizer.Categorize()
   - Write transactions to CSV using common.WriteTransactionsToCSV()
5. **Post-processing** → Root command's PersistentPostRun saves creditor/debtor mappings to YAML
6. **Output** → CSV file written to disk with categorized transactions

**Categorization Strategy Selection:**

1. **Direct Mapping** → Check creditors.yaml/debtors.yaml for exact party name match
2. **Keyword Matching** → Check categories.yaml rules for substring matches in party name/description
3. **Semantic/AI** → If enabled and no match found, query Gemini API for categorization
4. **Auto-Learning** → Save new mappings discovered via AI back to appropriate YAML file

**Configuration Hierarchy (later overrides earlier):**

1. Default values in Config struct
2. Config file: `~/.camt-csv/config.yaml` or `.camt-csv/config.yaml`
3. Environment variables: `CAMT_LOG_LEVEL`, `CAMT_AI_ENABLED`, `GEMINI_API_KEY`, etc.
4. CLI flags: `--log-level`, `--ai-enabled`, etc.

**State Management:**

- **Immutable Container** → Once created, Container is read-only via getter methods
- **Immutable Config** → Configuration is loaded once at startup and not modified
- **Categorizer Mutations** → Category mappings are saved to disk at command end via PersistentPostRun hook
- **No Global Mutable State** → All state flows through dependency injection

## Key Abstractions

**Parser Interface Hierarchy (internal/parser/parser.go):**

- `Parser` - Core parsing capability: `Parse(ctx, reader) ([]Transaction, error)`
- `Validator` - Format validation: `ValidateFormat(path) (bool, error)`
- `CSVConverter` - File-to-CSV conversion: `ConvertToCSV(ctx, inputFile, outputFile) error`
- `LoggerConfigurable` - Logger injection: `SetLogger(logger)`
- `CategorizerConfigurable` - Categorizer injection: `SetCategorizer(categorizer)`
- `BatchConverter` - Batch processing: `BatchConvert(ctx, inputDir, outputDir) (int, error)`
- `FullParser` - Composition of all above interfaces

**Example:** `internal/camtparser/adapter.go` embeds `parser.BaseParser` and implements FullParser

**TransactionCategorizer Interface (internal/models/categorizer.go):**

- Purpose: Abstraction for categorization strategies
- Method: `Categorize(ctx, partyName, isDebtor, amount, date, info string) (Category, error)`
- Implementations: `internal/categorizer/Categorizer` (multi-strategy orchestrator)

**Container (internal/container/container.go):**

- Purpose: Centralized dependency injection registry
- Immutable after creation with private fields
- Provides getters: `GetParser(type)`, `GetLogger()`, `GetConfig()`, `GetCategorizer()`, `GetStore()`
- Initializes all parsers with their dependencies in NewContainer()

**Categorization Strategies (internal/categorizer/):**

- `CategorizationStrategy` interface: `Execute(ctx, transaction) (StrategyResult, error)`
- Implementations:
  - `DirectMappingStrategy` - Exact name lookup in YAML
  - `KeywordStrategy` - Substring matching in keywords lists
  - `SemanticStrategy` - AI-based categorization via Gemini
- Orchestrator applies in priority order, stops at first success

## Entry Points

**main() → `main.go` line 84-89:**
- Location: `/Users/fjacquet/Projects/camt-csv/main.go`
- Triggers: Binary execution with any CLI command
- Responsibilities: Load .env, configure logging, initialize root command, execute Cobra

**init() → `main.go` line 22-43:**
- Location: `/Users/fjacquet/Projects/camt-csv/main.go`
- Triggers: Before main() when binary is loaded
- Responsibilities: Register all subcommands (camt, pdf, revolut, etc.), call root.Init()

**root.Init() → `cmd/root/root.go` line 169-201:**
- Location: `/Users/fjacquet/Projects/camt-csv/cmd/root/root.go`
- Triggers: Called during init()
- Responsibilities: Define persistent flags, bind to Viper, add management commands

**Cmd.PersistentPreRun → `cmd/root/root.go` line 50-63:**
- Location: `/Users/fjacquet/Projects/camt-csv/cmd/root/root.go`
- Triggers: Before ANY subcommand execution
- Responsibilities: Initialize configuration via Viper, create Container with DI

**Cmd.PersistentPostRun → `cmd/root/root.go` line 64-93:**
- Location: `/Users/fjacquet/Projects/camt-csv/cmd/root/root.go`
- Triggers: After ANY subcommand execution (success or failure with hook still running)
- Responsibilities: Save discovered mappings to creditors.yaml and debtors.yaml

**Subcommand Handlers → `cmd/{format}/convert.go`:**
- Location: `cmd/camt/convert.go`, `cmd/pdf/convert.go`, etc.
- Triggers: User runs `camt-csv camt --input X --output Y`
- Responsibilities: Get parser from container, call common.ProcessFile(), handle user output

## Error Handling

**Strategy:** Structured custom error types with context, Unwrap() support for error inspection

**Custom Error Types (internal/parsererror/):**

- `ParseError` - Parsing failures with parser name, field, value
- `ValidationError` - Format validation failures with reason
- `CategorizationError` - Categorization strategy failures
- `InvalidFormatError` - Detailed format mismatch with expected/actual
- `DataExtractionError` - Field extraction failures with raw data snippet

**Patterns:**

- All custom errors implement `error` interface with `Error()` method
- All provide `Unwrap()` for error chain inspection with `errors.Is()` and `errors.As()`
- Errors propagated up with context preserved (parser name, file path, etc.)
- Fatal errors logged and exit code returned via `log.Fatalf()` in command handlers

**Example:** `cmd/common/process.go` returns `ProcessFileWithError()` for testable error handling, wraps with fmt.Errorf()

## Cross-Cutting Concerns

**Logging:**
- Framework: Logrus via LogrusAdapter abstraction
- Approach: Structured logging with fields, passed as Logger interface
- Initialization: `config.ConfigureLoggingFromConfig()` sets level/format from Config
- Usage: Commands get logger via `container.GetLogger()` or root global `root.Log`
- Fields: `logging.Field{Key: "...", Value: ...}` for structured data

**Configuration:**
- Framework: Viper with hierarchical loading
- Sources: YAML file → environment variables → CLI flags (CLI overrides)
- Initialization: `config.InitializeConfig()` called in PersistentPreRun
- Access: Via Config struct fields or container.GetConfig()
- Environment vars: `CAMT_LOG_LEVEL`, `GEMINI_API_KEY`, `CAMT_AI_ENABLED`, etc.

**Context Propagation:**
- Pattern: Context passed through call chains for cancellation/timeout
- Usage: `Parse(ctx, reader)`, `Categorize(ctx, ...)`, `ConvertToCSV(ctx, ...)`
- Benefit: Enables graceful cancellation of long-running operations (e.g., AI API calls)

**Dependency Injection:**
- Container pattern centralizes wiring at startup
- All components receive dependencies via constructor parameters or SetXxx() methods
- Immutable Container ensures consistent dependency graph throughout execution
- Enables easy testing via mock injection and interface-based design

---

*Architecture analysis: 2026-02-01*
