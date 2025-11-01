# Go CLI Project Coding Standards Document

## 1. Introduction**

* **Purpose:** This document outlines the coding standards for Go CLI projects to ensure consistency, readability, maintainability, and optimal performance. Adherence to these standards is mandatory for all contributions.
* **Scope:** These standards apply to all Go CLI applications developed within our organization/team.
* **Guiding Principles:** Clarity, Simplicity, Idiomatic Go, Testability, Performance (where critical).

## 2. General Go Best Practices (Foundation)

* **Go Proverbs:** Encourage developers to internalize the Go Proverbs (e.g., "Clear is better than clever," "Errors are values").
* **`go fmt` and `goimports`:** Mandate the use of `go fmt` for code formatting and `goimports` for managing imports. Integrate these into pre-commit hooks or CI/CD pipelines.
* **Naming Conventions:**
  * **Package Names:** Short, concise, all lowercase, no underscores. Should reflect the package's purpose (e.g., `cmd`, `config`, `utils`).
  * **Variable Names:** Short when scope is small (e.g., `i` for loop counter), descriptive when scope is larger (e.g., `httpClient`).
  * **Function/Method Names:** CamelCase. Exported names start with an uppercase letter, unexported with lowercase.
  * **Receiver Names:** Short, typically one or two letters, consistent across methods for a given type (e.g., `(c *Config)`, `(s *Service)`).
* **Error Handling:**
  * Always check errors. Don't ignore them.
  * Propagate errors up the call stack until they can be handled meaningfully.
  * Use `fmt.Errorf` with `%w` for wrapping errors where appropriate (`errors.Is`, `errors.As`).
  * **Custom Error Types**: Use standardized error types from `internal/parsererror/`:
    - `ParseError`: General parsing failures with parser, field, and value context
    - `ValidationError`: Format validation failures with file path and reason
    - `CategorizationError`: Transaction categorization failures with strategy context
    - `InvalidFormatError`: Files not matching expected format with content snippets
    - `DataExtractionError`: Field extraction failures with raw data context
  * **Error Context**: Provide detailed context in error messages to aid troubleshooting
  * **Graceful Degradation**: Log warnings for recoverable issues, return errors for unrecoverable ones
  * Return errors as the last return value.
* **Concurrency:**
  * Favor `sync.WaitGroup` for waiting on goroutines.
  * Use channels for communication between goroutines.
  * Understand and avoid common concurrency pitfalls (race conditions, deadlocks).
  * Use `context.Context` for cancellation and propagating request-scoped values.
* **Documentation:**
  * All exported types, functions, and methods must have godoc comments.
  * Comments should explain *why* something is done, not just *what* it does.
  * Document complex algorithms or tricky parts of the code.
* **Testing:**
  * Write unit tests for all significant logic.
  * Use `go test` and follow standard testing patterns.
  * Table-driven tests are encouraged for multiple test cases.
  * Consider integration tests for CLI commands that interact with the file system or external services (mocking where appropriate).

## 3. CLI-Specific Guidelines

* **Command Structure:**
  * **Cobra/Viper:** Strongly recommend or mandate the use of `spf13/cobra` for command-line interfaces and `spf13/viper` for configuration management.
  * **Subcommands:** Organize complex CLIs into logical subcommands (e.g., `mycli config`, `mycli build`, `mycli deploy`).
  * **Root Command:** Define a clear root command with a concise `Short` and `Long` description.
* **Flags:**
  * **Naming:** Use kebab-case for long flags (e.g., `--file-path`, `--dry-run`). Short flags should be a single character where sensible (e.g., `-f`, `-d`).
  * **Descriptions:** Provide clear and helpful descriptions for all flags.
  * **Defaults:** Specify default values for flags where applicable.
  * **Required Flags:** Clearly indicate which flags are required.
* **Input/Output:**
  * **Standard Streams:** Use `os.Stdin`, `os.Stdout`, `os.Stderr` for I/O.
  * **User Feedback:**
    * Informational messages to `Stdout`.
    * Errors and warnings to `Stderr`.
    * Use consistent logging (see Logging section).
  * **Exit Codes:** Use standard Unix exit codes (0 for success, non-zero for failure).
* **Configuration:**
  * **Viper:** Use Viper for loading configuration from files, environment variables, and flags.
  * **Order of Precedence:** Document the order of precedence for configuration sources (e.g., flag > env var > config file > default).
  * **Default Locations:** Define standard locations for config files (e.g., `~/.mycli.yaml`, `/etc/mycli/config.yaml`).
* **Context:**
  * Pass `context.Context` to functions and methods that might perform long-running operations or interact with external services, allowing for cancellation (e.g., `Ctrl+C`).
* **Prompts/Interactivity:**
  * If the CLI requires interactive prompts, use a dedicated library (e.g., `survey` or `go-prompt`) for a consistent user experience.
* **Versioning:**
  * Integrate versioning information (e.g., using `ldflags` during build) so `mycli --version` or `mycli version` displays the current version, build date, and commit hash.

## 4. Code Organization and Structure

* **Project Layout:** Follow a standard Go project layout (e.g., similar to `github.com/golang-standards/project-layout`):
  * `cmd/your-cli-name/main.go`: Main entry point for the CLI.
  * `internal/`: Private application and library code.
  * `pkg/`: Library code that can be used by external applications.
  * `api/`: If applicable, API definitions.
  * `build/`: Build scripts, Dockerfiles.
  * `configs/`: Example configuration files.
  * `docs/`: Documentation.
  * `test/`: Integration tests or test data.
* **Modularity:** Break down logic into small, focused packages and functions. Avoid large, monolithic files.
* **Separation of Concerns:** Separate CLI parsing logic from core business logic.
  * `cmd` package should handle flag parsing and command execution.
  * Core logic should reside in `internal` or `pkg` packages.
* **Parser Architecture**: Follow the standardized parser architecture:
  - **Segregated Interfaces**: Use `Parser`, `Validator`, `CSVConverter`, `LoggerConfigurable` interfaces
  - **BaseParser Embedding**: All parsers should embed `parser.BaseParser` for common functionality
  - **Constructor Pattern**: Accept logger through constructor: `NewMyParser(logger logging.Logger)`
  - **Interface Implementation**: Implement only the interfaces your parser needs
* **Constants Usage**: Use constants from `internal/models/constants.go` instead of magic strings:
  ```go
  transaction.CreditDebit = models.TransactionTypeDebit  // Not "DBIT"
  transaction.Category = models.CategoryUncategorized   // Not "Uncategorized"
  ```
* **AIClient Interface**: When interacting with external AI services, define an `AIClient` interface to decouple the application from the specific AI provider. This allows for easier testing and swapping of AI backends.

## 5. Logging

* **Logging Abstraction:** Use the `logging.Logger` interface from `internal/logging` to decouple application code from specific logging frameworks. This enables easier testing and flexibility in choosing logging implementations.
* **Dependency Injection:** Components should receive logger instances through constructors rather than using global variables. This improves testability and eliminates global state.
* **BaseParser Integration:** All parsers should embed `parser.BaseParser` which provides logger management through `SetLogger()` and `GetLogger()` methods.
* **Implementation:** The default implementation uses `LogrusAdapter` which wraps `logrus.Logger`. Create loggers using `logging.NewLogrusAdapter(level, format)`.
* **Structured Logging:** Use the `logging.Field` struct for key-value pairs instead of string formatting:
  ```go
  logger.Info("Processing transaction",
      logging.Field{Key: "file", Value: filename},
      logging.Field{Key: "count", Value: len(transactions)})
  ```
* **Levels:** Use appropriate logging levels (DEBUG, INFO, WARN, ERROR, FATAL).
* **Output Format:** Support both JSON (for machine readability) and text (for human readability) formats.
* **Contextual Logging:** Use the `WithField`, `WithFields`, and `WithError` methods to add context to log messages.
* **Testing:** Use mock logger implementations in tests rather than depending on concrete logging frameworks.
* **Constructor Pattern:** All components requiring logging should accept a logger in their constructor:
  ```go
  func NewMyParser(logger logging.Logger) *MyParser {
      return &MyParser{
          BaseParser: parser.NewBaseParser(logger),
      }
  }
  ```

## 6. Dependency Management

* **Go Modules:** Use Go Modules exclusively for dependency management.
* **Pinning Versions:** Pin dependency versions to ensure reproducible builds.
* **Vendoring:** Discuss whether to vendor dependencies (e.g., for air-gapped environments) or rely on `go.mod`/`go.sum`.

## 7. Performance Considerations

* **Minimize Allocations:** Be mindful of unnecessary memory allocations, especially in performance-critical paths.
* **Benchmarking:** Use `go test -bench` for performance-critical functions.
* **Profiling:** Use `pprof` to identify performance bottlenecks when necessary.

## 8. Security

* **Input Validation:** Always validate user input (flags, arguments, config values).
* **Sensitive Data:** Never hardcode sensitive information. Use environment variables or secure configuration management.
* **Permissions:** Ensure generated files or directories have appropriate permissions.

## 9. Tooling and Automation

* **Linters:** Recommend or mandate specific linters (e.g., `golangci-lint` with a defined set of checks).
* **Pre-commit Hooks:** Suggest using tools like `pre-commit` to automate `go fmt`, `goimports`, and linting before commits.
* **CI/CD:** Integrate these checks into the CI/CD pipeline to enforce standards.
* **Speckit Workflow**: Utilize the `speckit` command suite (`spec`, `plan`, `tasks`, `analyze`, `implement`) to streamline the software development lifecycle, from specification to implementation.

## 10. Review Process

* **Code Reviews:** All code must undergo a thorough code review. Reviewers should ensure adherence to these standards.

---

## Implementation Steps

1. **Choose Core Libraries:** Decide on the primary libraries for CLI parsing (Cobra is almost a de-facto standard) and configuration (Viper).
2. **Define Linter Rules:** Set up a `.golangci.yml` file with the specific linters and rules you want to enforce.
3. **Template Project:** Consider creating a template Go CLI project that already adheres to these standards, making it easier for new projects to start correctly.
4. **Integration with CI/CD:** Automate the enforcement of these standards in your CI/CD pipeline.
5. **Training and Communication:** Clearly communicate these standards to the development team and provide training as needed.
6. **Living Document:** Treat this as a living document. Review and update it periodically based on new Go features, community best practices, and project needs.

This comprehensive document would provide a strong foundation for developing consistent, high-quality Go CLI projects.
