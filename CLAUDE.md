# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Using Makefile (recommended)
make build            # Build the application
make test             # Run all tests
make test-race        # Run tests with race detector
make coverage         # Generate HTML coverage report
make lint             # Run golangci-lint
make security         # Run gosec security scan
make sbom             # Generate SBOM (CycloneDX format)
make all              # Lint, test, and build
make install-tools    # Install dev tools (golangci-lint, gosec, cyclonedx-gomod)

# Direct commands (for specific cases)
go test -v -run TestFunctionName ./path/to/package  # Single test
go test -v -coverprofile=coverage.txt ./...         # Coverage profile
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

> **Detailed patterns and examples**: See `.claude/skills/golang-expert/` for comprehensive Go patterns including functional programming, interface design, testing, concurrency, error handling, and performance optimization.

### Core Principles

1. **KISS** - Prefer the simplest solution. No abstraction until needed (Rule of Three).
2. **DRY** - Single source of truth. Extract after 3 repetitions.
3. **No Global Mutable State** - Use dependency injection via `Container`.
4. **Immutability** - Private fields with getters, return new values.
5. **Pure Functions** - Same input = same output, no side effects.
6. **Interface Segregation** - Small, focused interfaces composed when needed.

### Dependency Injection

All dependencies flow through the `Container` (`internal/container/`):

```go
container, err := container.NewContainer(cfg)
logger := container.GetLogger()
parser, _ := container.GetParser(container.CAMT)
categorizer := container.GetCategorizer()
```

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
