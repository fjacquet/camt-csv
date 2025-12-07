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

### Key Design Patterns

**Parser Factory Pattern**: All parsers implement `models.Parser` interface defined in `internal/models/parser.go`:
```go
type Parser interface {
    Parse(r io.Reader) ([]Transaction, error)
    ConvertToCSV(inputFile, outputFile string) error
    WriteToCSV(transactions []Transaction, csvFile string) error
    SetLogger(logger *logrus.Logger)
    ValidateFormat(file string) (bool, error)
    BatchConvert(inputDir, outputDir string) (int, error)
}
```

New parsers are registered in `internal/parser/factory.go` via `GetParser(parserType ParserType)`.

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
2. Environment variables: `CAMT_LOG_LEVEL`, `GEMINI_API_KEY`, etc.
3. CLI flags: `--log-level`, `--ai-enabled`, etc.

### Testing Conventions

- Use `t.TempDir()` for file system tests
- Set `TEST_MODE=true` to disable real AI API calls
- Use `SetTestCategoryStore()` to inject mock stores in categorizer tests
- Each parser has `_test.go` with table-driven tests

### Adding a New Parser

1. Create package in `internal/{name}parser/`
2. Implement core parsing in `{name}parser.go`
3. Create adapter implementing `models.Parser` in `adapter.go`
4. Register in `internal/parser/factory.go`
5. Add CLI command in `cmd/{name}/convert.go`
6. Wire command in `main.go`

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
