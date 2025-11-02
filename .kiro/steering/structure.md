# Project Structure

## Directory Layout

```
camt-csv/
├── cmd/                    # CLI command implementations
│   ├── root/              # Root cobra command
│   ├── camt/              # CAMT.053 XML conversion command
│   ├── pdf/               # PDF conversion command
│   ├── revolut/           # Revolut CSV conversion command
│   ├── revolut-investment/# Revolut investment conversion command
│   ├── selma/             # Selma CSV conversion command
│   ├── debit/             # Generic debit CSV conversion command
│   ├── batch/             # Batch processing command
│   ├── categorize/        # Categorization command
│   ├── review/            # Codebase review command
│   └── common/            # Shared command utilities
├── internal/              # Private application code (not importable externally)
│   ├── models/            # Core data structures (Transaction, etc.)
│   ├── parser/            # Parser interface and factory
│   ├── camtparser/        # CAMT.053 XML parser implementation
│   ├── pdfparser/         # PDF parser implementation
│   ├── revolutparser/     # Revolut CSV parser implementation
│   ├── revolutinvestmentparser/ # Revolut investment parser
│   ├── selmaparser/       # Selma CSV parser implementation
│   ├── debitparser/       # Generic debit CSV parser implementation
│   ├── categorizer/       # Transaction categorization logic
│   ├── store/             # YAML data store (categories, mappings)
│   ├── config/            # Configuration management (Viper)
│   ├── common/            # Shared utilities (CSV I/O)
│   ├── currencyutils/     # Currency parsing and formatting
│   ├── dateutils/         # Date parsing and formatting
│   ├── fileutils/         # File system utilities
│   ├── logging/           # Centralized logging setup
│   ├── textutils/         # Text extraction utilities
│   ├── xmlutils/          # XML processing utilities
│   ├── validation/        # Input validation
│   ├── parsererror/       # Custom error types
│   ├── report/            # Report generation
│   ├── reviewer/          # Code review functionality
│   └── scanner/           # File scanning utilities
├── database/              # Configuration data files
│   ├── categories.yaml    # Category definitions and keywords
│   ├── creditors.yaml     # Creditor-to-category mappings
│   └── debtors.yaml       # Debtor-to-category mappings
├── samples/               # Sample input files for testing
│   ├── camt053/          # CAMT.053 XML samples
│   ├── pdf/              # PDF samples
│   ├── revolut/          # Revolut CSV samples
│   └── selma/            # Selma CSV samples
├── docs/                  # Documentation
│   ├── adr/              # Architecture Decision Records
│   ├── user-guide.md     # User documentation
│   ├── codebase_documentation.md
│   ├── design-principles.md
│   ├── coding-standards.md
│   └── operations.md     # Deployment and operations guide
├── specs/                 # Feature specifications (Speckit workflow)
├── .kiro/                # Kiro IDE configuration
│   └── steering/         # AI assistant steering rules
├── main.go               # Application entry point
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
├── .golangci.yml         # Linter configuration
└── README.md             # Project overview
```

## Architecture Patterns

### Parser Architecture

All parsers implement the `parser.Parser` interface:

```go
type Parser interface {
    Parse(r io.Reader) ([]models.Transaction, error)
}
```

**Adding a New Parser:**
1. Create package in `internal/<format>parser/`
2. Implement `Parser` interface
3. Create `adapter.go` with constructor
4. Add tests in `<format>parser_test.go`
5. Register in parser factory (if using factory pattern)
6. Add CLI command in `cmd/<format>/`

### Categorization Flow

1. **Direct Mapping**: Check `creditors.yaml` / `debtors.yaml` for exact matches
2. **Keyword Matching**: Search `categories.yaml` for keyword patterns
3. **AI Fallback**: Query Gemini API (if enabled)
4. **Auto-Learning**: Save successful AI categorizations to YAML files

### Configuration Hierarchy

Priority order (highest to lowest):
1. CLI flags (`--log-level debug`)
2. Environment variables (`LOG_LEVEL=debug`)
3. Config file (`~/.camt-csv/config.yaml`)
4. Default values

## Code Organization Principles

### Separation of Concerns

- **`cmd/`**: CLI interface, flag parsing, command orchestration
- **`internal/`**: Core business logic, pure functions where possible
- **`database/`**: Configuration data (YAML files)

### Pure vs Impure Functions

- **Pure Core**: `models/`, `currencyutils/`, `dateutils/`, categorization logic
- **Impure Layer**: File I/O, API calls, logging (confined to specific packages)

### Naming Conventions

- **Packages**: Short, lowercase, no underscores (e.g., `camtparser`, `dateutils`)
- **Files**: Lowercase with underscores for tests (e.g., `parser.go`, `parser_test.go`)
- **Exported**: CamelCase starting with uppercase
- **Unexported**: camelCase starting with lowercase
- **Interfaces**: Often named without "Interface" suffix (e.g., `Parser`, not `ParserInterface`)

### Testing Structure

- Test files alongside source: `parser.go` → `parser_test.go`
- Use `testdata/` subdirectories for test fixtures
- Mock external dependencies (file system, API calls)
- Table-driven tests for multiple scenarios
- Use `t.TempDir()` for temporary test files

## File Naming Patterns

- Main implementation: `<package>.go` (e.g., `camtparser.go`)
- Tests: `<package>_test.go`
- Adapters: `adapter.go` (constructor and setup)
- Helpers: `<package>_helpers.go`
- Constants: `constants.go`
- Errors: `errors.go`

## Import Organization

Standard Go import order:
1. Standard library
2. External dependencies
3. Internal packages

Example:
```go
import (
    "fmt"
    "io"
    
    "github.com/sirupsen/logrus"
    "github.com/spf13/cobra"
    
    "fjacquet/camt-csv/internal/models"
    "fjacquet/camt-csv/internal/parser"
)
```
