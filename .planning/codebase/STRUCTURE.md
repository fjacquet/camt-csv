# Codebase Structure

**Analysis Date:** 2026-02-01

## Directory Layout

```
camt-csv/
├── cmd/                           # CLI command implementations
│   ├── root/                      # Root command and container setup
│   ├── camt/                      # CAMT.053 conversion command
│   ├── pdf/                       # PDF statement conversion command
│   ├── revolut/                   # Revolut CSV conversion command
│   ├── revolut-investment/        # Revolut investment CSV conversion command
│   ├── selma/                     # Selma CSV conversion command
│   ├── debit/                     # Debit statement conversion command
│   ├── batch/                     # Batch directory conversion command
│   ├── categorize/                # Manual categorization command
│   ├── common/                    # Shared command processing logic
│   ├── review/                    # Code review/compliance command
│   ├── analyze/                   # Code analysis command
│   ├── implement/                 # Implementation planning command
│   └── tasks/                     # Task management command
├── internal/                      # Core application logic
│   ├── parser/                    # Parser interfaces and base implementation
│   ├── camtparser/                # CAMT.053 XML parser implementation
│   ├── pdfparser/                 # PDF statement parser implementation
│   ├── revolutparser/             # Revolut CSV parser implementation
│   ├── revolutinvestmentparser/   # Revolut investment CSV parser
│   ├── selmaparser/               # Selma CSV parser implementation
│   ├── debitparser/               # Debit statement parser implementation
│   ├── categorizer/               # Multi-strategy transaction categorizer
│   ├── models/                    # Core data structures (Transaction, Category, etc.)
│   ├── container/                 # Dependency injection container
│   ├── config/                    # Configuration management (Viper-based)
│   ├── store/                     # YAML file persistence for categories
│   ├── logging/                   # Logging abstraction (LogrusAdapter)
│   ├── parsererror/               # Custom error types
│   ├── common/                    # Shared CSV utilities (WriteTransactionsToCSV, etc.)
│   ├── batch/                     # Batch processing utilities
│   ├── validation/                # Data validation utilities
│   ├── dateutils/                 # Date parsing and formatting
│   ├── currencyutils/             # Currency conversion utilities
│   ├── textutils/                 # Text manipulation utilities
│   ├── xmlutils/                  # XML parsing utilities
│   ├── fileutils/                 # File system utilities
│   ├── scanner/                   # File scanning utilities
│   ├── integration/               # External API integration (Gemini AI)
│   ├── reviewer/                  # Code review and compliance checking
│   ├── git/                       # Git operations utilities
│   └── report/                    # Report generation utilities
├── database/                      # YAML configuration files (not generated)
│   ├── categories.yaml            # Transaction category definitions with keywords
│   ├── creditors.yaml             # Party name → category direct mappings (auto-learned)
│   └── debtors.yaml               # Party name → category direct mappings (auto-learned)
├── samples/                       # Sample input files for testing
│   ├── camt053/                   # Example CAMT.053 XML files
│   ├── pdf/                       # Example PDF statement files
│   ├── revolut/                   # Example Revolut CSV files
│   ├── selma/                     # Example Selma CSV files
│   └── debit/                     # Example debit statement files
├── docs/                          # Documentation
│   ├── architecture.md
│   ├── user-guide.md
│   ├── developer-guide.md
│   ├── design-principles.md
│   └── adr/                       # Architecture Decision Records
├── specs/                         # Specification documents and requirements
├── scripts/                       # Utility scripts
├── work/                          # Working directory for scratch files (not committed)
├── temp/                          # Temporary files (not committed)
├── main.go                        # Application entry point
├── Makefile                       # Build and test commands
├── go.mod                         # Go module dependencies
├── go.sum                         # Go dependency checksums
├── CHANGELOG.md                   # Release notes and version history
├── CLAUDE.md                      # Claude Code instructions for this project
├── README.md                      # Project overview
└── Dockerfile                     # Container image definition
```

## Directory Purposes

**cmd/ - CLI Commands:**
- Purpose: Expose application functionality through command-line interface using Cobra
- Contains: Command definitions, flag parsing, user-facing error messages
- Key files: Root command orchestration in `cmd/root/root.go`
- Convention: Each format has a directory with `convert.go` containing the command handler

**internal/parser/ - Parser Interfaces & Base:**
- Purpose: Define segregated parser interfaces and provide base implementation
- Key files: `parser.go` (interfaces), `base.go` (BaseParser for embedding)
- Pattern: All parsers embed BaseParser and implement segregated interfaces from here

**internal/{format}parser/ - Format-Specific Parsers:**
- Purpose: Transform format-specific input into standardized Transaction model
- Structure: `adapter.go` (implements FullParser), `{format}parser.go` (core logic), `adapter_test.go`
- Pattern: Adapter embeds BaseParser, implements Parse() with format-specific logic
- Examples: `internal/camtparser/adapter.go`, `internal/revolutparser/adapter.go`

**internal/categorizer/ - Categorization Engine:**
- Purpose: Apply multi-strategy transaction categorization with auto-learning
- Key files:
  - `categorizer.go` - Orchestrator that runs strategies in priority order
  - `strategy.go` - Strategy interface definition
  - `direct_mapping.go` - Exact name lookup strategy
  - `keyword.go` - Substring matching strategy
  - `semantic_strategy.go` - AI-based categorization strategy
  - `gemini_client.go` - Gemini API integration
- Pattern: Strategies implement common interface, executed in sequence until match

**internal/models/ - Core Data Structures:**
- Purpose: Define domain models used throughout application
- Key files:
  - `transaction.go` - Transaction model (64 fields for various formats)
  - `categorizer.go` - Category model and TransactionCategorizer interface
  - `models.go` - Compliance/review related models
- Convention: Export models with many fields for flexibility across parser formats

**internal/container/ - Dependency Injection:**
- Purpose: Centralized wiring of all dependencies at application startup
- Key file: `container.go` - Creates logger, config, store, categorizer, all parsers
- Pattern: Immutable Container with private fields, public getter methods
- Usage: Get from root command via `root.GetContainer()`

**internal/config/ - Configuration Management:**
- Purpose: Load application configuration from multiple sources with hierarchy
- Key files:
  - `viper.go` - Viper-based configuration loading
  - `config.go` - Configuration struct definition and deprecated legacy functions
- Sources: YAML file → environment variables → CLI flags (CLI overrides)
- Usage: Loaded in root.Init() and passed to Container

**internal/store/ - Persistence:**
- Purpose: Load and save category configurations to YAML files
- Key file: `store.go` - CategoryStore for finding and loading YAML files
- Files managed: `database/categories.yaml`, `database/creditors.yaml`, `database/debtors.yaml`
- Pattern: FindConfigFile() searches standard locations (current dir, ./database/, ~/.config/)

**internal/logging/ - Structured Logging:**
- Purpose: Abstract logging implementation behind interface for testability
- Key files: `logrus_adapter.go` (implements Logger interface), `logger.go` (interface)
- Pattern: LogrusAdapter wraps sirupsen/logrus, exposes Logger interface
- Usage: Passed to components via SetLogger() method

**internal/parsererror/ - Error Types:**
- Purpose: Custom error types with structured context for better error handling
- Types:
  - ParseError - Parser failures with field/value context
  - ValidationError - Format validation failures
  - InvalidFormatError - Detailed format mismatch descriptions
  - DataExtractionError - Field extraction failures
  - CategorizationError - Categorization strategy failures
- Pattern: All implement error interface with Unwrap() for error chain inspection

**database/ - Configuration Data:**
- Purpose: Store category definitions and auto-learned mappings
- Files (not generated, committed):
  - `categories.yaml` - Category definitions with matching keywords
  - `creditors.yaml` - Auto-learned creditor name → category mappings (generated)
  - `debtors.yaml` - Auto-learned debtor name → category mappings (generated)
- Pattern: Saved by categorizer.SaveCreditorsToYAML() in root command's PersistentPostRun

## Key File Locations

**Entry Points:**
- `main.go` (line 1-89) - Binary entry point, loads .env, calls root.Init(), executes CLI
- `cmd/root/root.go` - Root command definition, container initialization, shared flags

**Configuration & DI:**
- `internal/config/viper.go` - Config loading with Viper hierarchy
- `internal/container/container.go` - Dependency injection setup, parser registration

**Interfaces & Abstractions:**
- `internal/parser/parser.go` - Segregated parser interfaces (Parser, Validator, CSVConverter, etc.)
- `internal/models/categorizer.go` - TransactionCategorizer interface
- `internal/models/transaction.go` - Transaction model with 64 exported fields
- `internal/parser/base.go` - BaseParser for embedding in concrete parsers

**Core Parsers (by file count & complexity):**
- `internal/camtparser/` - CAMT.053 XML parser (most complex, ISO 20022 standard)
- `internal/pdfparser/` - PDF text extraction parser
- `internal/revolutparser/` - Revolut CSV parser
- `internal/selmaparser/` - Selma CSV parser
- `internal/debitparser/` - Debit statement parser
- `internal/revolutinvestmentparser/` - Investment statement parser

**Categorization:**
- `internal/categorizer/categorizer.go` - Multi-strategy orchestrator
- `internal/categorizer/direct_mapping.go` - Exact name lookup
- `internal/categorizer/keyword.go` - Keyword-based matching
- `internal/categorizer/semantic_strategy.go` - AI categorization
- `internal/categorizer/gemini_client.go` - Gemini API client

**Testing & Mocks:**
- Each internal package has `*_test.go` files
- Mock implementations: `container/container_test.go`, `store/mock.go`
- Test data: `internal/categorizer/testdata/`

## Naming Conventions

**Files:**
- Go source: `{descriptor}.go` (e.g., `transaction.go`, `categorizer.go`, `adapter.go`)
- Tests: `{descriptor}_test.go` (e.g., `transaction_test.go`, `adapter_test.go`)
- Acceptance/special tests: `{descriptor}_internal_test.go` (e.g., `convert_internal_test.go`)
- Configuration: YAML files in `database/` and user's `~/.camt-csv/`
- Documentation: `*.md` in `docs/` and root directory

**Directories:**
- Internal packages: All lowercase with no underscores (e.g., `camtparser`, `revolutparser`, `categorizer`)
- Format prefixes: `{format}parser` (e.g., `camtparser`, `pdfparser`, `revolutparser`)
- Utility packages: Descriptive names (e.g., `dateutils`, `textutils`, `xmlutils`)

**Types (Go structs/interfaces):**
- Public: PascalCase (e.g., `Transaction`, `Categorizer`, `BaseParser`)
- Interfaces: PascalCase with descriptive names (e.g., `Parser`, `FullParser`, `Validator`, `TransactionCategorizer`)
- Private: camelCase (e.g., `logger`, `categorizer`, `strategies`)

**Functions/Methods:**
- Public: PascalCase (e.g., `NewAdapter()`, `Parse()`, `Categorize()`, `GetLogger()`)
- Getter pattern: `Get{Name}()` (e.g., `GetLogger()`, `GetCategorizer()`, `GetParser()`)
- Setter pattern: `Set{Name}()` (e.g., `SetLogger()`, `SetCategorizer()`)
- Constructor pattern: `New{Type}()` (e.g., `NewContainer()`, `NewCategoryStore()`)
- Private: camelCase (e.g., `parseTransaction()`, `normalizeString()`)

**Constants:**
- Public: ALL_CAPS with underscores (e.g., `CAMT`, `PDF`, `Revolut` for ParserType)
- Actually: Most constants use their descriptive names (e.g., `ErrInvalidFormat`, `OverallStatusCompliant`)

**Packages:**
- Lowercase single word (e.g., `models`, `container`, `categorizer`)
- Multiple concepts: use shorter name (e.g., `camtparser` not `camt_parser`)

## Where to Add New Code

**New Parser Format:**
- Create: `internal/{format}parser/` directory
- Files needed:
  - `{format}parser.go` - Core parsing logic
  - `adapter.go` - Implements parser.FullParser interface, embeds BaseParser
  - `adapter_test.go` - Unit tests for adapter
  - `{format}parser_test.go` - Unit tests for parsing logic
- Register: Add to `internal/container/container.go` in `NewContainer()`
- Command: Create `cmd/{format}/convert.go` with Cobra command handler
- Wire: Add to `main.go` init() via `root.Cmd.AddCommand()`

**New Categorization Strategy:**
- Create: File in `internal/categorizer/` named `{strategy_name}.go`
- Implement: `CategorizationStrategy` interface from `internal/categorizer/strategy.go`
- Register: Add to strategy slice in `internal/categorizer/categorizer.go`
- Pattern: See `direct_mapping.go`, `keyword.go`, `semantic_strategy.go` as examples

**New Utility Package:**
- Create: `internal/{concept}utils/` directory (e.g., `newutils/`)
- Convention: Keep focused on single responsibility
- Export: Public functions with clear names
- Test: Include `_test.go` with table-driven tests

**New CLI Command:**
- Create: `cmd/{command}/` directory with at least one `*.go` file
- Pattern: Create Cobra Command and assign to `Cmd` variable
- Wire: In `main.go` init(), add `root.Cmd.AddCommand({command}.Cmd)`
- Flags: Use `root.SharedFlags` for common flags (input, output) or define new persistent flags
- Container: Get from root with `root.GetContainer()`

**Adding Fields to Transaction Model:**
- Edit: `internal/models/transaction.go`
- Pattern: Add struct tag with `csv:"FieldName"` for CSV output
- Use: `csv:"-"` to exclude from CSV output (e.g., `Payee`, `Payer` kept for backward compatibility)
- Parsers: Update relevant parsers in `{format}parser.go` to populate new fields
- Tests: Add to test data in appropriate parser test files

**New Logging:**
- Pattern: Inject logger via constructor or SetLogger() method
- Usage: `logger.Info("message")`, `logger.WithError(err).Warn("message")`
- Structured fields: `logger.WithField("key", value).Debug("message")`
- Never: Use `log` package directly or global loggers

**New Configuration Options:**
- Edit: `internal/config/config.go` - Add to Config struct
- YAML: Add section to `internal/config/viper.go` configuration loading
- Environment: Add env var mapping in viper.SetDefault() or BindEnv()
- CLI flags: Add to root command flags in `cmd/root/root.go`
- Documentation: Update `CLAUDE.md` with new options

## Special Directories

**database/ - Configuration & Mappings:**
- Purpose: Store category definitions and learned mappings
- Generated: `creditors.yaml` and `debtors.yaml` are generated/auto-learned
- Committed: `categories.yaml` is committed (canonical definitions)
- Behavior: Categorizer.SaveCreditorsToYAML() and SaveDebitorsToYAML() update these files
- Location: Can be in project directory or user's `~/.camt-csv/` (store.FindConfigFile searches)

**samples/ - Example Files:**
- Purpose: Reference input files for each format for testing and documentation
- Committed: Yes, checked into version control
- Usage: Developers use for manual testing, documentation shows examples
- Subdirectories: `camt053/`, `pdf/`, `revolut/`, `selma/`, `debit/`

**docs/ - Documentation:**
- Purpose: User guides, developer guides, architecture records
- ADR: `docs/adr/` contains Architecture Decision Records
- Committed: Yes, documentation is versioned
- Format: Markdown files with clear structure

**specs/ - Specification Documents:**
- Purpose: Technical specifications and requirements
- Contains: Implementation contracts, feature specs, issue details
- Format: Markdown and YAML
- Usage: References for development

**temp/ and work/ - Scratch Directories:**
- Purpose: Temporary working files during development
- Committed: NO - these are .gitignored
- Cleanup: Safe to delete, regenerated as needed
- Usage: Debug output, test scripts, scratch notes

**vendor/ - Dependencies (if used):**
- Purpose: Would contain pinned Go module dependencies
- Currently: Not present, using go.mod/go.sum instead
- Behavior: `go mod download` fetches to GOPATH

---

*Structure analysis: 2026-02-01*
