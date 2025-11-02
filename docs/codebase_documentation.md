# CAMT-CSV Project Documentation

This document provides a comprehensive overview of the `camt-csv` project, detailing its purpose, architecture, core functionalities, adherence to functional programming principles, testing strategy, and dependency management.

## 1. Project Overview

The `camt-csv` project is a command-line interface (CLI) application designed to convert various financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV, Debit CSV) into a standardized CSV format. A key feature is its intelligent transaction categorization, which employs a hybrid approach combining local keyword matching with AI-based classification using the Gemini model.

**Key Features:**

* **Multi-format Conversion:** Supports CAMT.053 XML, PDF (including Viseca credit card statements), Revolut CSV, Revolut Investment CSV, Selma CSV, and generic Debit CSV.
* **Transaction Categorization:** Hybrid approach using local YAML-based keyword matching and AI (Gemini) as a fallback.
* **CLI Interface:** Modular command structure for various operations (camt, pdf, batch, categorize, revolut-investment).
* **Hierarchical Configuration:** Viper-based configuration system supporting config files, environment variables, and CLI flags with full backward compatibility.
* **Extensible Parser Architecture:** Standardized interface for easy addition of new data sources.

**High-Level Architecture:**

The project follows a clean separation of concerns:

* **`cmd/`**: Contains the entry points for the CLI commands (e.g., `camt`, `pdf`, `batch`). Each command typically orchestrates calls to the `internal/` packages.
* **`internal/`**: Houses the core application logic, divided into specialized packages. This directory adheres to Go best practices, ensuring its contents are not importable by external projects, promoting encapsulation. Key architectural improvements include dependency injection, logging abstraction, and elimination of global state.
* **`database/`**: Stores YAML configuration files for categorization rules (categories, creditors, debtors).
* **`samples/`**: Provides example input files for various formats.

```bash
camt-csv/
├── cmd/               # CLI command definitions
├── internal/          # Application-specific packages (pure core & infrastructure)
├── database/          # Configuration data files
└── samples/           # Sample files for testing
```

### 2. Core Functional Principles

The `camt-csv` project, particularly its `internal` packages, demonstrates a strong adherence to functional programming principles, as outlined in the `GEMINI.md` guidelines:

* **Pure Functions:** Many functions, especially within `internal/models`, `internal/currencyutils`, and `internal/dateutils`, are pure. For example, `models.FormatDate` and `currencyutils.StandardizeAmount` take inputs and return outputs without side effects, always producing the same output for the same input.
* **Immutability:** Data structures like `models.Transaction` are designed to be passed by value or explicitly copied when modifications are needed, although Go's struct behavior means they are not strictly immutable by default. The `Update...` methods on `Transaction` modify the receiver, which is a common Go idiom, but the overall design encourages treating `Transaction` objects as value types.
* **Function Composition:** The `parser.DefaultParser` and `common.GeneralizedConvertToCSV` exemplify composition. `GeneralizedConvertToCSV` takes `parseFunc` and `validateFunc` as arguments, composing them to create a complete conversion pipeline.
* **Strict Separation of Core Logic and Side Effects:
  * **Pure Core:** Packages like `internal/models`, `internal/currencyutils`, `internal/dateutils`, and parts of `internal/categorizer` (e.g., `categorizeLocallyByKeywords`) contain pure logic.
  * **Impure Layer (Side Effects):** Interactions with the file system (`os.ReadFile`, `os.WriteFile`), external APIs (Gemini in `internal/categorizer`), and logging (`logrus`) are confined to specific functions or packages, typically within the `internal/infrastructure` (though not explicitly named as such, `internal/store` and parts of `internal/categorizer` handle this) or the top-level `cmd/` packages. For instance, `categorizer.categorizeWithGemini` handles the external API call, while `categorizer.categorizeTransaction` orchestrates the pure and impure steps.
* **Higher-Order Functions:** While not explicitly using `map`, `filter`, `reduce` in a functional library sense (as Go's standard library doesn't heavily promote this style), the design of functions accepting other functions (like `common.GeneralizedConvertToCSV`) achieves a similar compositional benefit.

### 3. Key Modules and Their Responsibilities

* **`cmd/`**: 
  * `analyze/analyze.go`: CLI command for codebase analysis and compliance checking.
  * `batch/batch.go`: Handles batch conversion of multiple files.
  * `camt/convert.go`: CLI command for CAMT.053 XML conversion.
  * `categorize/categorize.go`: CLI command for categorizing transactions.
  * `common/process.go`: Common processing logic for CLI commands.
  * `debit/convert.go`: CLI command for Debit CSV conversion.
  * `implement/implement.go`: CLI command for implementing development tasks.
  * `pdf/convert.go`: CLI command for PDF statement conversion.
  * `review/review.go`: CLI command for codebase compliance review.
  * `revolut/convert.go`: CLI command for Revolut CSV conversion.
  * `revolut-investment/convert.go`: CLI command for Revolut investment CSV conversion.
  * `root/root.go`: Defines the root Cobra command for the CLI.
  * `selma/convert.go`: CLI command for Selma CSV conversion.
  * `tasks/tasks.go`: CLI command for task management and tracking.

* **`main.go`**: Main entry point for the CLI application.

* **`internal/`**: 
  * **`camtparser/`**: Parses CAMT.053 XML files. Embeds `BaseParser` and implements the `parser.Parser` interface.
  * **`categorizer/`**: Core logic for transaction categorization. Manages local keyword matching, creditor/debtor mappings, and integrates with the Gemini AI for fallback categorization. It also handles rate limiting for AI calls.
  * **`common/`**: Provides shared utilities, including CSV reading/writing (`csv.go`) and a generalized conversion function (`GeneralizedConvertToCSV`).
  * **`config/`**: Handles application configuration using Viper, supporting hierarchical configuration from files, environment variables, and CLI flags.
  * **`currencyutils/`**: Utility functions for currency parsing, formatting, and calculations (e.g., tax amounts).
  * **`dateutils/`**: Utility functions for date parsing, formatting, and business day calculations.
  * **`debitparser/`**: Parses generic debit CSV files. Embeds `BaseParser` and implements the `parser.Parser` interface.
  * **`fileutils/`**: General file system utilities.
  * **`logging/`**: Framework-agnostic logging abstraction layer with `Logger` interface and `LogrusAdapter` implementation. Provides structured logging with `Field` struct for key-value pairs, enabling dependency injection and easier testing with mock loggers.
  * **`models/`**: Defines the core data structures, most notably the `models.Transaction` struct, which represents a standardized financial transaction. It includes methods for amount parsing, date formatting, and deriving transaction properties. Contains comprehensive constants in `constants.go` to eliminate magic strings and numbers throughout the codebase.
  * **`parser/`**: Defines segregated parser interfaces following Interface Segregation Principle (`Parser`, `Validator`, `CSVConverter`, `LoggerConfigurable`, `FullParser`) and provides the `BaseParser` foundation that all parsers embed for common functionality including logging and CSV writing.
  * **`parsererror/`**: Defines comprehensive custom error types for parsing operations, including `ParseError`, `ValidationError`, `CategorizationError`, `InvalidFormatError`, and `DataExtractionError` with proper error wrapping and context.
  * **`pdfparser/`**: Parses PDF bank statements with dependency injection for PDF extraction. Embeds `BaseParser` and implements the `parser.Parser` interface. Includes specialized logic for Viseca credit card statements.
  * **`revolutparser/`**: Parses Revolut CSV export files. Embeds `BaseParser` and implements the `parser.Parser` interface.
  * **`revolutinvestmentparser/`**: Parses Revolut investment transaction CSV files. Embeds `BaseParser` and implements the `parser.Parser` interface. Handles investment-specific transaction types like BUY, DIVIDEND, and CASH TOP-UP.
  * **`selmaparser/`**: Parses Selma investment CSV files, with specific logic for handling investment-related transactions and stamp duty. Embeds `BaseParser` and implements the `parser.Parser` interface.
  * **`store/`**: Manages loading and saving categorization data (categories, creditors, debtors) from YAML files.
  * **`textutils/`**: Utilities for text extraction and manipulation.
  * **`xmlutils/`**: Utilities for XML processing (e.g., XPath, constants).

* **`database/`**: 
  * `categories.yaml`: Defines custom transaction categories and associated keywords for local matching.
  * `creditors.yaml`: Stores mappings from creditor names to categories.
  * `debtors.yaml`: Stores mappings from debtor names to categories.

### 4. Standardized Parser Architecture

The project employs a highly standardized parser architecture built around segregated interfaces and a common base implementation, making it easy to add new financial data sources while eliminating code duplication.

**Core Parser Interfaces:**

The architecture is built on several segregated interfaces defined in `internal/parser/parser.go`:

* **`Parser`**: Core parsing interface with `Parse(r io.Reader) ([]models.Transaction, error)` method
* **`Validator`**: Interface for format validation with `ValidateFormat(filePath string) (bool, error)` method  
* **`CSVConverter`**: Interface for CSV conversion with `ConvertToCSV(inputFile, outputFile string) error` method
* **`LoggerConfigurable`**: Interface for logger management with `SetLogger(logger logging.Logger)` method
* **`FullParser`**: Composite interface combining all capabilities for parsers that need complete functionality

**BaseParser Foundation:**

All parser implementations embed the `BaseParser` struct from `internal/parser/base.go`, which provides:

* **Common Logger Management**: Implements `LoggerConfigurable` interface with `SetLogger()` and `GetLogger()` methods
* **Shared CSV Writing**: Provides `WriteToCSV()` method using the common CSV writer from `internal/common`
* **Consistent Initialization**: `NewBaseParser(logger)` constructor ensures proper setup
* **Framework-Agnostic Logging**: Uses `logging.Logger` interface for dependency injection

**Parser Implementation Pattern:**

```go
type MyParser struct {
    parser.BaseParser
    // parser-specific fields
}

func NewMyParser(logger logging.Logger) *MyParser {
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}

func (p *MyParser) Parse(r io.Reader) ([]models.Transaction, error) {
    p.GetLogger().Info("Starting parse operation")
    // implementation using structured logging
}
```

This architecture eliminates code duplication while maintaining the flexibility to add parser-specific functionality. All concrete parser implementations (e.g., `camtparser`, `pdfparser`, `revolutparser`, `revolutinvestmentparser`, `selmaparser`, `debitparser`) follow this pattern.

**Error Handling:**

The parser architecture includes comprehensive error handling with custom error types defined in `internal/parsererror/`:

* **`ParseError`**: For general parsing failures with context
* **`ValidationError`**: For format validation failures  
* **`InvalidFormatError`**: For files that don't match expected format
* **`DataExtractionError`**: For failures extracting specific data fields
* **`CategorizationError`**: For transaction categorization failures

**Constants and Magic String Elimination:**

All parsers use constants from `internal/models/constants.go` instead of magic strings:

* Transaction types: `TransactionTypeDebit`, `TransactionTypeCredit`
* Categories: `CategoryUncategorized`, `CategorySalary`, etc.
* Bank codes: `BankCodeCashWithdrawal`, `BankCodePOS`, etc.
* File permissions: `PermissionConfigFile`, `PermissionDirectory`

### 5. Transaction Categorization

Transaction categorization is a core feature, implemented in `internal/categorizer/`. It uses a multi-stage, hybrid approach:

1. **Direct Mapping (Exact Match):** The system first checks `creditorMappings` and `debitorMappings` (loaded from `database/creditors.yaml` and `database/debitors.yaml`). These provide exact, case-insensitive matches for known payees/payers to specific categories. This is the fastest and most efficient method for recurring transactions.
2. **Local Keyword Matching:** If no direct mapping is found, the `categorizeLocallyByKeywords` function attempts to match transaction descriptions and party names against keywords defined in `database/categories.yaml`. This method is also fast and avoids API calls.
3. **AI-Based Categorization (Fallback):** If local matching fails and AI categorization is enabled (`ai.enabled: true` in your configuration), the system falls back to using the Google Gemini API.
    * A prompt is constructed with transaction details and a list of allowed categories.
    * The Gemini model (`ai.model`, default `gemini-2.0-flash`) is queried.
    * A rate limiter (`ai.requests_per_minute`) is implemented to prevent exceeding API quotas.
    * Successful AI categorizations are *automatically saved* to the `creditors.yaml` or `debitors.yaml` files, effectively "learning" new mappings and reducing future AI calls for similar transactions.

**Customization:** Users can customize categories and keyword rules by editing `database/categories.yaml`. New creditor/debitor mappings are automatically learned and saved by the application.

### 6. Testing Strategy

The project employs a comprehensive testing strategy using Go's built-in `testing` package and the `stretchr/testify/assert` library for assertions.

* **Unit Tests:** Each significant package (e.g., `camtparser`, `categorizer`, `common`, `currencyutils`, `dateutils`, `debitparser`, `models`, `pdfparser`, `revolutparser`, `selmaparser`, `store`) has its own `_test.go` file containing unit tests.
* **Test Setup and Teardown:** Tests frequently use `t.TempDir()` to create temporary directories for test files, ensuring isolation and automatic cleanup (`defer os.RemoveAll(tempDir)`).
* **Mocking:**
  * **File System:** Tests often create mock input files (`os.WriteFile`) to simulate real data without relying on external resources.
  * **External Dependencies:** For the `categorizer` package, `os.Setenv("TEST_MODE", "true")` is used to disable actual Gemini API calls during tests, ensuring tests are fast and deterministic. A `SetTestCategoryStore` function allows injecting a mock `store.CategoryStore` to control data loading.
  * **PDF Extraction:** The `pdfparser_test.go` uses a `mockPDFExtraction()` function to prevent actual `pdftotext` calls, returning predefined mock transactions.
* **Assertions:** The `github.com/stretchr/testify/assert` library is used extensively for clear and concise assertions (e.g., `assert.NoError`, `assert.True`, `assert.Equal`, `assert.Contains`).
* **Non-Regression Testing:** The presence of numerous test files across all core functionalities indicates a strong emphasis on non-regression testing. Running `go test ./...` from the project root would execute the entire suite, ensuring existing features are not broken by new changes.
* **Logger Configuration:** Many test files include an `init()` function or `SetLogger` calls to configure `logrus` for debug output during testing, aiding in debugging test failures.

### 7. Dependency Management

The project uses Go Modules for dependency management, ensuring reproducible builds and efficient package operations.

* **`go.mod` and `go.sum`**: These files define the project's direct and transitive dependencies, managed by Go Modules.
* **Centralized Dependencies**: Dependencies are declared in `go.mod`, ensuring a centralized and version-controlled list.
* **Reproducible Builds**: `go.sum` ensures that the exact versions of all dependencies are used, leading to reproducible builds across different environments.

This documentation provides a comprehensive understanding of the `camt-csv` project, its design principles, and implementation details.
