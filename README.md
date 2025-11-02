# CAMT-CSV

> Convert financial statements to CSV with intelligent transaction categorization

CAMT-CSV is a powerful command-line tool that converts various financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV, and more) into standardized CSV files with AI-powered transaction categorization.

[![Go CI](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml/badge.svg)](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/fjacquet/camt-csv/graph/badge.svg?token=ST9KKUV81N)](https://codecov.io/gh/fjacquet/camt-csv)

## ✨ Key Features

- **Multi-format Support**: Extensible parser architecture with segregated interfaces (`Parser`, `Validator`, `CSVConverter`, `LoggerConfigurable`) and `BaseParser` foundation for easy addition of new financial statement formats.
- **Smart Categorization**: Three-tier hybrid approach using direct mapping, keyword matching, and AI fallback with strategy pattern implementation (`DirectMappingStrategy`, `KeywordStrategy`, `AIStrategy`).
- **Dependency Injection Architecture**: Clean architecture with explicit dependencies through `Container` pattern, eliminating global state and improving testability.
- **Framework-Agnostic Logging**: Structured logging abstraction (`logging.Logger` interface) with `LogrusAdapter` implementation for flexible logging backends and dependency injection.
- **Transaction Builder Pattern**: Fluent API for constructing complex transactions with validation, type safety, and enhanced backward compatibility methods with direction-based logic (`GetPayee()`, `GetPayer()`, `GetCounterparty()`).
- **Comprehensive Error Handling**: Custom error types (`ParseError`, `ValidationError`, `CategorizationError`, `InvalidFormatError`, `DataExtractionError`) with detailed context and proper error wrapping.
- **Hierarchical Configuration**: Viper-based config system supporting files, environment variables, and CLI flags with full backward compatibility.
- **Investment Support**: Dedicated parser for Revolut investment transactions (BUY, DIVIDEND, CASH TOP-UP) with specialized categorization.
- **Batch Processing**: Handle multiple files at once with automatic format detection.
- **Performance Optimized**: String operations optimization with `strings.Builder`, lazy initialization with `sync.Once`, and pre-allocation for efficient processing.
- **Constants-Based Design**: Complete elimination of magic strings and numbers through comprehensive constants in `internal/models/constants.go`.
- **Comprehensive Testing**: Risk-based testing strategy with 100% coverage for critical paths (parsing, validation, categorization) and comprehensive coverage elsewhere.

## 🚀 Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
go build

# For PDF support, install poppler:
# macOS: brew install poppler
# Ubuntu: apt-get install poppler-utils
```

### Basic Usage

```bash
# Convert a file using a specific parser
./camt-csv <parser-type> -i input.file -o output.csv

# Example: Convert a CAMT.053 file
./camt-csv camt -i statement.xml -o processed.csv

# Example: Convert Revolut investment transactions
./camt-csv revolut-investment -i investments.csv -o processed.csv

# Batch process multiple files
./camt-csv batch -i input_dir/ -o output_dir/

# Perform codebase compliance review
./camt-csv review /path/to/your/project/src/file.go /path/to/your/project/src/module/ \
  --constitution-files .camt-csv/constitution.yaml \
  --principles "GO-001,GO-002" \
  --output-format json \
  --output-file compliance_report.json \
  --git-ref main
```

### Configuration

CAMT-CSV supports hierarchical configuration with multiple options:

```bash
# Option 1: Configuration file (recommended)
mkdir -p ~/.camt-csv
cat > ~/.camt-csv/camt-csv.yaml << EOF
log:
  level: "info"
  format: "text"
csv:
  delimiter: ","
  include_headers: true
ai:
  enabled: true
  model: "gemini-2.0-flash"
EOF

# Option 2: Environment variables (backward compatible)
export GEMINI_API_KEY=your_api_key_here
export CAMT_LOG_LEVEL=debug
export CAMT_AI_ENABLED=true

# Option 3: CLI flags (temporary overrides)
./camt-csv --log-level debug --ai-enabled camt -i file.xml -o output.csv
```

See [Configuration Migration Guide](docs/configuration-migration-guide.md) for complete details.

## 📚 Documentation

- **[User Guide](docs/user-guide.md)** - Complete usage guide with examples and troubleshooting
- **[Codebase Documentation](docs/codebase_documentation.md)** - Technical architecture and development details
- **[Design Principles](docs/design-principles.md)** - Core design philosophy and patterns

## 🏗️ Architecture & Supported Formats

### Parser Architecture

All parsers follow a standardized architecture built on dependency injection and interface segregation:

```go
// Segregated interfaces for specific capabilities
type Parser interface {
    Parse(r io.Reader) ([]models.Transaction, error)
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

type FullParser interface {
    Parser
    Validator
    CSVConverter
    LoggerConfigurable
}

// BaseParser provides common functionality
type MyParser struct {
    parser.BaseParser  // Embedded for logging and CSV writing
    // parser-specific fields
}

func NewMyParser(logger logging.Logger) *MyParser {
    return &MyParser{
        BaseParser: parser.NewBaseParser(logger),
    }
}
```

### Dependency Injection Container

The application uses a centralized container for dependency management:

```go
// Create container with all dependencies
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}

// Access components through the container
parser, err := container.GetParser(container.CAMT)
categorizer := container.GetCategorizer()
logger := container.GetLogger()

// All dependencies are explicitly injected
```

### Transaction Builder Pattern

Complex transactions are constructed using the builder pattern:

```go
tx, err := models.NewTransactionBuilder().
    WithDate("2025-01-15").
    WithAmount(decimal.NewFromFloat(100.50), "CHF").
    WithPayer("John Doe", "CH1234567890").
    WithPayee("Acme Corp", "CH0987654321").
    AsDebit().
    Build()
```

### Supported Formats

| Parser Type | Description | Key Features |
|---|---|---|
| `camt` | ISO 20022 bank statements (CAMT.053 XML) | Multi-currency support, complete transaction details, party information |
| `pdf` | PDF bank statements (Viseca, generic) | Text extraction with dependency injection, specialized Viseca parsing |
| `revolut` | Revolut app CSV exports | Transaction state handling, fee processing, currency conversion |
| `revolut-investment` | Revolut investment transactions | BUY/DIVIDEND/CASH TOP-UP categorization, share tracking |
| `selma` | Investment platform CSV data | Investment categorization, stamp duty association |
| `debit` | Generic CSV debit transactions | Flexible column mapping, date format detection |

## 🤖 Smart Categorization

CAMT-CSV uses a sophisticated three-tier strategy pattern for transaction categorization:

### Strategy Pattern Implementation

The categorization system uses the Strategy pattern for modular, testable categorization:

```go
type CategorizationStrategy interface {
    Categorize(ctx context.Context, tx Transaction) (Category, bool, error)
    Name() string
}

type Categorizer struct {
    strategies []CategorizationStrategy
    store      *store.CategoryStore
    logger     logging.Logger
}

// Strategies are executed in priority order:
// 1. DirectMappingStrategy - Exact name matches (fastest)
// 2. KeywordStrategy - Pattern matching (local)  
// 3. AIStrategy - Gemini AI fallback (optional)

// Usage with dependency injection
categorizer := container.GetCategorizer()
category, err := categorizer.CategorizeTransaction(transaction)
```

### Categorization Flow

1. **Direct Mapping Strategy** - Instant recognition from `creditors.yaml`/`debtors.yaml` using exact name matching
2. **Keyword Strategy** - Local pattern matching from `categories.yaml` using configurable keyword rules
3. **AI Strategy** - Gemini AI for unknown transactions with auto-learning and rate limiting

### Dependency Injection

The categorizer uses dependency injection for all components:

```go
// Container manages all dependencies
container, err := container.NewContainer(config)
categorizer := container.Categorizer

// Or create directly with dependencies
categorizer := categorizer.NewCategorizer(store, aiClient, logger)
```

### Migration Note

**Important**: Starting from version 2.0.0, the debtor mapping file has been renamed from `debitors.yaml` to `debtors.yaml` to follow standard English spelling conventions. If you have an existing `debitors.yaml` file, please rename it to `debtors.yaml`. The application will continue to work with the old filename for backward compatibility, but it's recommended to update your files.

**Enhanced Backward Compatibility**: Version 2.0.0 introduces enhanced backward compatibility methods for the Transaction model:

- `GetPayee()` - Returns appropriate party based on transaction direction (payee for debits, payer for credits)
- `GetPayer()` - Returns appropriate party based on transaction direction (payer for debits, payee for credits)  
- `GetCounterparty()` - Always returns the "other party" in the transaction (recommended for new code)
- `GetAmountAsFloat()` - Continues to work but is deprecated in favor of `GetAmountAsDecimal()`

These methods ensure existing code continues to work while providing a clear migration path to the new architecture. See the [Migration Guide](docs/MIGRATION_GUIDE_V2.md) for detailed migration instructions.

## 🛠️ Development

```bash
# Run tests
go test ./...

# Build for production
go build -ldflags="-s -w"

# View help
./camt-csv --help
```

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

---

**Need help?** Check the [User Guide](docs/user-guide.md) for detailed instructions and examples.
