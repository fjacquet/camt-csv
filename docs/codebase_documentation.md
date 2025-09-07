# Mailtag Project Documentation

This document provides a comprehensive overview of the `camt-csv` project, detailing its purpose, architecture, core functionalities, adherence to functional programming principles, testing strategy, and dependency management.

## 1. Project Overview

The `camt-csv` project is a command-line interface (CLI) application designed to convert various financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV, Debit CSV) into a standardized CSV format. A key feature is its intelligent transaction categorization, which employs a hybrid approach combining local keyword matching with AI-based classification using the Gemini model.

**Key Features:**

* **Multi-format Conversion:** Supports CAMT.053 XML, PDF (including Viseca credit card statements), Revolut CSV, Revolut Investment CSV, Selma CSV, and generic Debit CSV.
* **Transaction Categorization:** Hybrid approach using local YAML-based keyword matching and AI (Gemini) as a fallback.
* **CLI Interface:** Modular command structure for various operations (convert, batch, categorize, revolut-investment).
* **Hierarchical Configuration:** Viper-based configuration system supporting config files, environment variables, and CLI flags with full backward compatibility.
* **Extensible Parser Architecture:** Standardized interface for easy addition of new data sources.

**High-Level Architecture:**

The project follows a clean separation of concerns:

* **`cmd/`**: Contains the entry points for the CLI commands (e.g., `camt`, `pdf`, `batch`). Each command typically orchestrates calls to the `internal/` packages.
* **`internal/`**: Houses the core application logic, divided into specialized packages. This directory adheres to Go best practices, ensuring its contents are not importable by external projects, promoting encapsulation.
* **`database/`**: Stores YAML configuration files for categorization rules (categories, creditors, debitors).
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
* **Strict Separation of Core Logic and Side Effects:**
  * **Pure Core:** Packages like `internal/models`, `internal/currencyutils`, `internal/dateutils`, and parts of `internal/categorizer` (e.g., `categorizeLocallyByKeywords`) contain pure logic.
  * **Impure Layer (Side Effects):** Interactions with the file system (`os.ReadFile`, `os.WriteFile`), external APIs (Gemini in `internal/categorizer`), and logging (`logrus`) are confined to specific functions or packages, typically within the `internal/infrastructure` (though not explicitly named as such, `internal/store` and parts of `internal/categorizer` handle this) or the top-level `cmd/` packages. For instance, `categorizer.categorizeWithGemini` handles the external API call, while `categorizer.categorizeTransaction` orchestrates the pure and impure steps.
* **Higher-Order Functions:** While not explicitly using `map`, `filter`, `reduce` in a functional library sense (as Go's standard library doesn't heavily promote this style), the design of functions accepting other functions (like `common.GeneralizedConvertToCSV`) achieves a similar compositional benefit.

### 3. Key Modules and Their Responsibilities

* **`cmd/`**:
  * `batch/batch.go`: Handles batch conversion of multiple files.
  * `camt/convert.go`: CLI command for CAMT.053 XML conversion.
  * `camt-csv/main.go`: Main entry point for the CLI application.
  * `categorize/categorize.go`: CLI command for categorizing transactions.
  * `common/process.go`: Common processing logic for CLI commands.
  * `debit/convert.go`: CLI command for Debit CSV conversion.
  * `pdf/convert.go`: CLI command for PDF statement conversion.
  * `revolut/convert.go`: CLI command for Revolut CSV conversion.
  * `root/root.go`: Defines the root Cobra command for the CLI.
  * `selma/convert.go`: CLI command for Selma CSV conversion.

* **`internal/`**:
  * **`camtparser/`**: Parses CAMT.053 XML files. Implements the `parser.Parser` interface.
  * **`categorizer/`**: Core logic for transaction categorization. Manages local keyword matching, creditor/debitor mappings, and integrates with the Gemini AI for fallback categorization. It also handles rate limiting for AI calls.
  * **`common/`**: Provides shared utilities, including CSV reading/writing (`csv.go`) and a generalized conversion function (`GeneralizedConvertToCSV`).
  * **`config/`**: Handles application configuration, primarily loading environment variables.
  * **`currencyutils/`**: Utility functions for currency parsing, formatting, and calculations (e.g., tax amounts).
  * **`dateutils/`**: Utility functions for date parsing, formatting, and business day calculations.
  * **`debitparser/`**: Parses generic debit CSV files. Implements the `parser.Parser` interface.
  * **`fileutils/`**: General file system utilities.
  * **`logging/`**: Centralized logging setup using `logrus`.
  * **`models/`**: Defines the core data structures, most notably the `models.Transaction` struct, which represents a standardized financial transaction. It includes methods for amount parsing, date formatting, and deriving transaction properties.
  * **`parser/`**: Defines the `parser.Parser` interface, which all specific parsers must implement, ensuring a consistent API. It also provides a `DefaultParser` for common functionalities.
  * **`parsererror/`**: Defines custom error types for parsing operations.
  * **`pdfparser/`**: Parses PDF bank statements, including specialized logic for Viseca credit card statements. Implements the `parser.Parser` interface.
  * **`revolutparser/`**: Parses Revolut CSV export files. Implements the `parser.Parser` interface.
  * **`selmaparser/`**: Parses Selma investment CSV files, with specific logic for handling investment-related transactions and stamp duty. Implements the `parser.Parser` interface.
  * **`store/`**: Manages loading and saving categorization data (categories, creditors, debitors) from YAML files.
  * **`textutils/`**: Utilities for text extraction and manipulation.
  * **`xmlutils/`**: Utilities for XML processing (e.g., XPath, constants).

* **`database/`**:
  * `categories.yaml`: Defines custom transaction categories and associated keywords for local matching.
  * `creditors.yaml`: Stores mappings from creditor names to categories.
  * `debitors.yaml`: Stores mappings from debitor names to categories.

### 4. Standardized Parser Architecture

The project employs a highly standardized parser architecture, making it easy to add new financial data sources. This is achieved through the `parser.Parser` interface defined in `internal/parser/parser.go`.

**`parser.Parser` Interface:**

All concrete parser implementations (e.g., `camtparser`, `pdfparser`, `revolutparser`, `selmaparser`, `debitparser`) must implement the following methods:

* `ParseFile(filePath string) ([]models.Transaction, error)`: Parses a source file and extracts a slice of `models.Transaction` objects.
* `ValidateFormat(filePath string) (bool, error)`: Checks if a given file adheres to the expected format for that parser.
* `ConvertToCSV(inputFile, outputFile string) error`: A convenience method that combines parsing and writing to CSV.
* `WriteToCSV(transactions []models.Transaction, csvFile string) error`: Writes a slice of `models.Transaction` objects to a CSV file.
* `SetLogger(logger *logrus.Logger)`: Allows injecting a custom logger for the parser.

The `parser.DefaultParser` struct provides common implementations for `WriteToCSV` and `ConvertToCSV`, promoting code reuse and consistency. The `common.GeneralizedConvertToCSV` function further abstracts the conversion flow, accepting generic `parseFunc` and `validateFunc` arguments.

### 5. Transaction Categorization

Transaction categorization is a core feature, implemented in `internal/categorizer/`. It uses a multi-stage, hybrid approach:

1. **Direct Mapping (Exact Match):** The system first checks `creditorMappings` and `debitorMappings` (loaded from `database/creditors.yaml` and `database/debitors.yaml`). These provide exact, case-insensitive matches for known payees/payers to specific categories. This is the fastest and most efficient method for recurring transactions.
2. **Local Keyword Matching:** If no direct mapping is found, the `categorizeLocallyByKeywords` function attempts to match transaction descriptions and party names against keywords defined in `database/categories.yaml`. This method is also fast and avoids API calls.
3. **AI-Based Categorization (Fallback):** If local matching fails and AI categorization is enabled (`USE_AI_CATEGORIZATION=true` in `.env`), the system falls back to using the Google Gemini API.
    * A prompt is constructed with transaction details and a list of allowed categories.
    * The Gemini model (`GEMINI_MODEL`, default `gemini-2.0-flash`) is queried.
    * A rate limiter (`GEMINI_REQUESTS_PER_MINUTE`) is implemented to prevent exceeding API quotas.
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

The project uses `uv` for dependency management, as specified in `GEMINI.md`. This ensures reproducible builds and efficient package operations.

* **`go.mod` and `go.sum`**: These files define the project's direct and transitive dependencies, managed by Go Modules.
* **`uv`**: While `go.mod` and `go.sum` are Go's native dependency management, the `GEMINI.md` explicitly states `uv` for all package management operations. This implies that `uv` would be used to interact with the Go module system (e.g., `uv run go mod tidy`, `uv run go build`).
* **`pyproject.toml`**: Although this is a Go project, the `GEMINI.md` mentions `pyproject.toml` for defining dependencies. This suggests a potential future integration with Python tools or a general guideline for projects that might involve Python. For this specific Go project, `go.mod` is the primary dependency manifest.
* **Centralized Dependencies**: Dependencies are declared in `go.mod`, ensuring a centralized and version-controlled list.
* **Reproducible Builds**: `go.sum` ensures that the exact versions of all dependencies are used, leading to reproducible builds across different environments.

This documentation provides a comprehensive understanding of the `camt-csv` project, its design principles, and implementation details.
