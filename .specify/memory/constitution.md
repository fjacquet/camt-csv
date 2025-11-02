<!--
Sync Impact Report:
Version change: 1.1.1 -> 1.2.0
List of modified principles:
- I. Clear, Idiomatic Go (Expanded)
- II. Robust Error Handling (Expanded)
- III. Comprehensive Testing (NON-NEGOTIABLE) (Expanded)
- V. CLI Best Practices (Expanded)
- VI. Interface-Driven Design (Expanded)
- VII. Single Responsibility Principle (Expanded)
- VIII. Immutability by Default (Expanded)
- IX. Hybrid Categorization (Expanded)
- X. Configuration Management (Expanded)
Added sections:
- Design Patterns
- Anti-Patterns Avoided
- Performance & Resource Management
- Security
- Monitoring & Observability
- Deployment & Release
Removed sections: None
Templates requiring updates:
- /Users/fjacquet/Projects/camt-csv/.specify/templates/plan-template.md: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.specify/templates/spec-template.md: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.specify/templates/tasks-template.md: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.analyze.toml: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.checklist.toml: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.clarify.toml: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.constitution.toml: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.implement.toml: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.plan.toml: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.specify.toml: ✅ updated
- /Users/fjacquet/Projects/camt-csv/.gemini/commands/speckit.tasks.toml: ✅ updated
Follow-up TODOs: None
-->
# camt-csv Constitution

## Core Principles

### I. Clear, Idiomatic Go
Code MUST adhere to Go Proverbs, `go fmt`, `goimports`, and standard Go naming conventions (short for small scope, descriptive for larger scope, CamelCase for exported, lowercase for unexported, short receiver names). Clarity and simplicity are paramount.

### II. Robust Error Handling
Errors MUST always be checked and propagated. Use `fmt.Errorf` with `%w` for wrapping errors where appropriate (`errors.Is`, `errors.As`). Define custom error types for specific error conditions. Errors MUST be returned as the last return value. Fail-fast where appropriate, but provide graceful degradation with meaningful fallbacks.

### III. Comprehensive Testing (NON-NEGOTIABLE)
All significant logic MUST have unit tests. Table-driven tests are encouraged. Integration tests MUST be considered for CLI commands that interact with the file system or external services. End-to-end tests MUST cover complete user workflows. Tests MUST be isolated, deterministic, and maintainable, adhering to the Test Pyramid philosophy. Assertions MUST use `github.com/stretchr/testify`. Minimum 80% code coverage for all packages, with 100% for critical paths (parsing, validation). The entire test suite MUST pass before any changes are considered complete. Test-Driven Development (TDD) is strongly encouraged.

### IV. Explicit Concurrency Management
`sync.WaitGroup` and channels MUST be used for goroutine coordination. `context.Context` MUST be used for cancellation and propagating request-scoped values. Common concurrency pitfalls (race conditions, deadlocks) MUST be understood and avoided.

### V. CLI Best Practices
`spf13/cobra` and `spf13/viper` MUST be used for CLI and configuration management. Commands MUST be organized into logical subcommands. Flags MUST use kebab-case for long flags, have clear descriptions, specify default values, and clearly indicate required flags. Input/Output MUST use standard streams (`os.Stdin`, `os.Stdout`, `os.Stderr`), with informational messages to `Stdout` and errors/warnings to `Stderr`. Exit codes MUST follow standard Unix conventions (0 for success, non-zero for failure). Configuration precedence MUST be: CLI flags > Environment variables > Configuration file > Default values.

### VI. Interface-Driven Design
All parsers MUST implement a common `parser.Parser` interface (defined in `internal/parser/parser.go`) to ensure consistency, interchangeability, and polymorphism. This allows for easy addition of new financial data formats and simplifies testing and maintenance.

### VII. Single Responsibility Principle
Each component (package, function, type) MUST have a single, well-defined responsibility. For example, Parsers handle format-specific logic, Models define data structures, Categorizer handles categorization, Common provides shared utilities, and Store manages configuration and category storage. This promotes clear separation of concerns, easier debugging, and reduced coupling.

### VIII. Immutability by Default
Core data models and configuration objects MUST be designed to be immutable where possible. Modifications to data structures MUST result in new instances rather than in-place changes. `github.com/shopspring/decimal` MUST be used for financial amounts to prevent floating-point errors.

### IX. Hybrid Categorization
Transaction categorization MUST employ a three-tier hybrid system: Direct Mapping (exact matches from `database/creditors.yaml` and `database/debtors.yaml`), Keyword Matching (local rules from `database/categories.yaml` using `gopkg.in/yaml.v3`), and AI Fallback (Google Gemini AI for unknown transactions). This system prioritizes performance, learning, cost control, and privacy. CSV processing MUST use `github.com/gocarina/gocsv`.

### X. Configuration Management
A hierarchical configuration system using `spf13/viper` MUST be used, with precedence: CLI flags > Environment variables (with `CAMT_` prefix, loaded via `github.com/joho/godotenv`) > Configuration file (`~/.camt-csv/config.yaml` or `./camt-csv.yaml`) > Default values. Sensitive data (e.g., API keys) MUST be provided via environment variables only and never hardcoded or logged. Configuration MUST be validated at startup.

## Code Quality & Style

All exported types, functions, and methods MUST have godoc comments explaining *why* something is done, not just *what* it does. Structured logging (using `github.com/sirupsen/logrus`) MUST be used with appropriate levels (DEBUG, INFO, WARN, ERROR, FATAL), specified output format (JSON for machine readability, plain text for human readability), and relevant contextual information. Linters (e.g., `golangci-lint` with a defined set of checks) MUST be enforced via pre-commit hooks and CI/CD pipelines.

## Dependency Management

Go Modules MUST be used exclusively for dependency management. Dependency versions MUST be pinned to ensure reproducible builds.

## Design Patterns

The project MUST leverage appropriate design patterns to enhance maintainability and extensibility, including but not limited to: Strategy Pattern (for different parser implementations), Adapter Pattern (for interface bridging), Factory Pattern (for object creation), and Template Method Pattern (for common logic with customizations). Anti-patterns such as God Objects, Tight Coupling, Magic Numbers/Strings, and Premature Optimization MUST be avoided.

## Performance & Resource Management

Performance optimization MUST be driven by profiling, focusing on minimizing allocations, streaming large files, and efficient data structures. Resources (file handles, memory) MUST be managed properly with prompt cleanup. Benchmarking MUST be used for critical paths.

## Security

All user input (flags, arguments, config values) MUST be validated. Sensitive data MUST never be hardcoded and MUST be handled securely via environment variables or secure configuration management. File permissions MUST be appropriate (e.g., 0600 for config files, 0755 for binaries). Network security policies (e.g., HTTPS only) and vulnerability scanning MUST be implemented.

## Monitoring & Observability

Structured logging (using `github.com/sirupsen/logrus`) MUST be used for all significant operations. Application and system metrics (e.g., Prometheus) MUST be collected to track performance, success/failure rates, and resource utilization. Health checks MUST be implemented to monitor application status and dependencies.

## Deployment & Release

Semantic Versioning (MAJOR.MINOR.PATCH) MUST be followed for releases. The release process MUST be automated via CI/CD pipelines (e.g., GitHub Actions) for cross-platform builds and Docker images. Deployment strategies (standalone binary, container, Kubernetes) MUST be documented, including environment-specific configuration and secret management.

## Governance

This Constitution supersedes all other project practices. Amendments MUST be documented, approved by core maintainers, and include a migration plan if backward incompatible. All pull requests and code reviews MUST verify compliance with these principles. Complexity MUST be justified and aligned with the principle of simplicity. The `docs/coding-standards.md` file provides runtime development guidance.

**Version**: 1.2.0 | **Ratified**: 2025-10-12 | **Last Amended**: 2025-10-12