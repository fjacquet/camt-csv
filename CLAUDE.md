# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build the application
go build

# Run all tests
go test ./...

# Run tests with race detection
go test -v -race ./...

# Run tests with coverage
go test -v -coverprofile=coverage.txt -covermode=atomic ./...

# Run a single test
go test -v -run TestFunctionName ./path/to/package

# Lint (requires golangci-lint)
golangci-lint run --timeout=5m

# Security scan (requires gosec)
gosec -exclude=G304 ./...
```

## Architecture Overview

This is a Go CLI tool that converts financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV) into standardized CSV with AI-powered categorization.

### CAMT File Format (ISO 20022)

The camt parser handles **CAMT.053** (Bank to Customer Statement) files:

- Namespace: `urn:iso:std:iso:20022:tech:xsd:camt.053.001.02`
- Standard: ISO 20022
- Structure defined in: `internal/models/iso20022.go`

**Supported CAMT Types:**
| Type | Description | Supported |
|------|-------------|-----------|
| CAMT.052 | Bank to Customer Account Report | No |
| CAMT.053 | Bank to Customer Statement | Yes (v001.02) |
| CAMT.054 | Bank to Customer Debit/Credit Notification | No |

**Known Limitations:**

- Only version 001.02 tested (newer versions may have additional fields)
- No strict namespace validation (will attempt to parse any XML with matching structure)
- Swiss bank-specific extensions may not be fully supported

### Key Design Patterns

**Parser Factory Pattern**: Parsers implement segregated interfaces in `internal/parser/parser.go`:

```go
type Parser interface {
    Parse(r io.Reader) ([]Transaction, error)
}

type Validator interface {
    ValidateFormat(filePath string) (bool, error)
}

type CSVConverter interface {
    ConvertToCSV(inputFile, outputFile string) error
}

type LoggerConfigurable interface {
    SetLogger(logger logging.Logger)
}

type CategorizerConfigurable interface {
    SetCategorizer(categorizer models.TransactionCategorizer)
}

type BatchConverter interface {
    BatchConvert(inputDir, outputDir string) (int, error)
}

// FullParser combines all capabilities
type FullParser interface {
    Parser
    Validator
    CSVConverter
    LoggerConfigurable
    CategorizerConfigurable
    BatchConverter
}
```

New parsers are registered in `internal/factory/factory.go`. **Important**: CLI commands should get parsers from the DI Container (`root.GetContainer().GetParser()`), not directly from the factory, to ensure categorizers are properly wired.

**Three-Tier Categorization** (`internal/categorizer/`):

1. Direct mapping - exact match from `database/creditors.yaml` / `database/debitors.yaml`
2. Keyword matching - rules from `database/categories.yaml`
3. AI fallback - Gemini API via `AIClient` interface (testable abstraction)

AI categorizations are auto-learned and saved back to YAML files.

### Directory Structure

- `cmd/` - Cobra CLI commands (camt, pdf, batch, categorize, revolut, selma, debit, revolut-investment)
- `internal/` - Core application logic:
  - `*parser/` packages - Format-specific parsers with `adapter.go` implementing the interface
  - `categorizer/` - Transaction categorization with AI integration
  - `models/` - Core data structures (`Transaction`, `Category`, `Parser` interface)
  - `config/` - Viper-based hierarchical configuration
  - `store/` - YAML category database management
  - `common/` - Shared CSV utilities
- `database/` - YAML configuration files for categorization rules

### Configuration Hierarchy

Configuration loads in order (later overrides earlier):

1. Config file: `~/.camt-csv/camt-csv.yaml` or `.camt-csv/config.yaml`
2. Environment variables (see mapping below)
3. CLI flags: `--log-level`, `--ai-enabled`, etc.

**Environment Variable Mapping:**

| Config Key | Environment Variable | CLI Flag |
|------------|---------------------|----------|
| `log.level` | `CAMT_LOG_LEVEL` | `--log-level` |
| `ai.enabled` | `CAMT_AI_ENABLED` | `--ai-enabled` |
| `ai.model` | `CAMT_AI_MODEL` | - |
| `ai.api_key` | `GEMINI_API_KEY` | - |

Note: The `.env` file is auto-loaded from the current directory.

### Testing Conventions

- Use `t.TempDir()` for file system tests
- Set `TEST_MODE=true` to disable real AI API calls
- Use `SetTestCategoryStore()` to inject mock stores in categorizer tests
- Each parser has `_test.go` with table-driven tests

### Adding a New Parser

1. Create package in `internal/{name}parser/`
2. Implement core parsing in `{name}parser.go`
3. Create adapter implementing `parser.FullParser` in `adapter.go`
4. Register in `internal/factory/factory.go`
5. Add CLI command in `cmd/{name}/convert.go`
6. Wire command in `main.go`

## Coding Principles

### KISS - Keep It Simple, Stupid

Simplicity is the ultimate sophistication. Always prefer the simplest solution that works.

```go
// BAD - over-engineered
type TransactionProcessorFactory interface {
    CreateProcessor(config ProcessorConfig) TransactionProcessor
}
type TransactionProcessor interface {
    Process(tx Transaction) (ProcessedTransaction, error)
}

// GOOD - simple and direct
func ProcessTransaction(tx Transaction) (Transaction, error) {
    // Direct implementation
}
```

**Guidelines:**

- Don't add abstraction until you need it (Rule of Three)
- Avoid premature optimization
- If a junior developer can't understand it in 5 minutes, simplify it
- One function = one responsibility
- Prefer flat over nested code

### DRY - Don't Repeat Yourself

Every piece of knowledge should have a single, authoritative representation.

```go
// BAD - repeated logic
func ParseCAMTDate(s string) time.Time {
    t, _ := time.Parse("2006-01-02", s)
    return t
}
func ParseRevolutDate(s string) time.Time {
    t, _ := time.Parse("2006-01-02", s)  // Duplicated!
    return t
}

// GOOD - single source of truth
const DateFormat = "2006-01-02"
func ParseDate(s string) (time.Time, error) {
    return time.Parse(DateFormat, s)
}
```

**Guidelines:**

- Extract common logic into shared functions (`internal/common/`)
- Use constants for repeated values (`internal/models/constants.go`)
- Single source of truth for business logic
- But: Don't DRY prematurely - wait for 3 repetitions (Rule of Three)
- Acceptable duplication: test code, simple one-liners

### Functional Programming Guidelines

This codebase follows functional programming principles where applicable:

**1. No Global Mutable State**

```go
// BAD - global mutable state
var log = logrus.New()
func SetLogger(l *logrus.Logger) { log = l }

// GOOD - dependency injection
func NewParser(logger logging.Logger) *Parser {
    return &Parser{logger: logger}
}
```

**2. Immutability**

```go
// BAD - mutable struct fields
type Container struct {
    Logger logging.Logger  // Can be modified
}

// GOOD - private fields with getters
type Container struct {
    logger logging.Logger  // Immutable after creation
}
func (c *Container) GetLogger() logging.Logger { return c.logger }
```

**3. Pure Functions (where possible)**

```go
// BAD - modifies external state
func ProcessTransaction(tx *Transaction) {
    tx.Category = "Food"  // Mutates input
}

// GOOD - returns new value
func CategorizeTransaction(tx Transaction) Transaction {
    result := tx  // Copy
    result.Category = "Food"
    return result
}
```

**4. Constants Over Variables**

```go
// BAD - mutable delimiter
var Delimiter rune = ','

// GOOD - constant
const DefaultDelimiter rune = ','
```

**5. Configure at Construction**

```go
// BAD - SetX() mutator pattern
parser := NewParser()
parser.SetLogger(logger)
parser.SetCategorizer(cat)

// GOOD - constructor injection
parser := NewParser(logger, categorizer)
```

### Dependency Injection

All dependencies should flow through the `Container` (`internal/container/`):

```go
// Create container with all dependencies
container, err := container.NewContainer(cfg)

// Access dependencies via getters
logger := container.GetLogger()
parser, _ := container.GetParser(container.CAMT)
categorizer := container.GetCategorizer()
```

### Interface Design

Follow Interface Segregation Principle:

- Small, focused interfaces (`Parser`, `Validator`, `CSVConverter`)
- Compose into larger interfaces when needed (`FullParser`)
- Use `models.TransactionCategorizer` for categorization (type-safe)

## Changelog Management

**IMPORTANT**: Update `CHANGELOG.md` for every significant change.

### When to Update

Update the changelog when you:

- Add new features or commands
- Fix bugs
- Make breaking changes
- Change configuration options
- Modify public APIs or interfaces
- Add/remove dependencies
- Make security-related changes

### How to Update

1. Add entries under `## [Unreleased]` section
2. Use the appropriate category:

   - **Added** - new features
   - **Changed** - changes in existing functionality
   - **Deprecated** - soon-to-be removed features
   - **Removed** - removed features
   - **Fixed** - bug fixes
   - **Security** - vulnerability fixes

3. Write entries in imperative mood: "Add feature" not "Added feature"
4. Reference issues/PRs when relevant: "Fix parsing error (#123)"

### Release Process

When creating a release:

1. Change `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`
2. Add new empty `## [Unreleased]` section above
3. Update comparison links at bottom of file
4. Follow semver: breaking=major, features=minor, fixes=patch
