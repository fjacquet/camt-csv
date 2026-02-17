# API Specifications - CAMT-CSV Project

## Overview

This document provides formal specifications for all APIs, interfaces, and contracts within the CAMT-CSV project. These specifications serve as the foundation for specification-driven development, ensuring consistency and reliability across all components.

## Parser Interface Specification

### Core Parser Interface

```go
type Parser interface {
    Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error)
}
```

#### Parse Method Specification

**Signature**: `Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error)`

**Purpose**: Extract financial transactions from a source reader

**Preconditions**:

- `ctx` MUST be a valid `context.Context`
- `r` MUST be a valid `io.Reader`
- The data from the reader MUST be in the format supported by the specific parser

**Postconditions**:

- Returns slice of valid `Transaction` objects on success
- Returns empty slice (not nil) if no transactions found
- Returns error if data format is invalid or I/O error occurs
- All returned transactions MUST have valid `Date`, `Amount`, and `Currency` fields

**Error Conditions**:

```go
// Invalid format
InvalidFormatError{Reason: string}

// Parse error with context
ParseError{Line: int, Reason: string}
```

**Implementation Requirements**:

- MUST handle malformed data gracefully
- MUST use structured logging with relevant context
- MUST NOT be responsible for closing the reader

## Data Model Specifications

### Transaction Model

```go
type Transaction struct {
    BookkeepingNumber string          `csv:"BookkeepingNumber"`
    Status            string          `csv:"Status"`
    Date              time.Time       `csv:"Date"`
    ValueDate         time.Time       `csv:"ValueDate"`
    Name              string          `csv:"Name"`
    PartyName         string          `csv:"PartyName"`
    PartyIBAN         string          `csv:"PartyIBAN"`
    Description       string          `csv:"Description"`
    RemittanceInfo    string          `csv:"RemittanceInfo"`
    Amount            decimal.Decimal `csv:"Amount"`
    CreditDebit       string          `csv:"CreditDebit"`
    DebitFlag         bool            `csv:"IsDebit"`
    Debit             decimal.Decimal `csv:"Debit"`
    Credit            decimal.Decimal `csv:"Credit"`
    Currency          string          `csv:"Currency"`
    Product           string          `csv:"Product"`
    AmountExclTax     decimal.Decimal `csv:"AmountExclTax"`
    AmountTax         decimal.Decimal `csv:"AmountTax"`
    TaxRate           decimal.Decimal `csv:"TaxRate"`
    Recipient         string          `csv:"Recipient"`
    Investment        string          `csv:"InvestmentType"`
    Number            string          `csv:"Number"`
    Category          string          `csv:"Category"`
    Type              string          `csv:"Type"`
    Fund              string          `csv:"Fund"`
    NumberOfShares    int             `csv:"NumberOfShares"`
    Fees              decimal.Decimal `csv:"Fees"`
    IBAN              string          `csv:"IBAN"`
    EntryReference    string          `csv:"EntryReference"`
    Reference         string          `csv:"Reference"`
    AccountServicer   string          `csv:"AccountServicer"`
    BankTxCode        string          `csv:"BankTxCode"`
    OriginalCurrency  string          `csv:"OriginalCurrency"`
    OriginalAmount    decimal.Decimal `csv:"OriginalAmount"`
    ExchangeRate      decimal.Decimal `csv:"ExchangeRate"`
    // Internal fields (not exported to CSV)
    Payee string `csv:"-"`
    Payer string `csv:"-"`
}
```

#### Validation Rules

**Required Fields**:

- `Date`: Must be valid `time.Time`
- `Amount`: Must be valid decimal, can be negative
- `Currency`: Must be valid ISO 4217 currency code
- `CreditDebit`: Must be either "CRDT" or "DBIT"

**Format Constraints**:

- IBAN fields: Must pass IBAN validation if present
- Date fields: Stored as `time.Time`, exported to CSV as DD.MM.YYYY
- Decimal fields: Must use `decimal.Decimal` for precision
- Internal fields (Payee, Payer): Not exported to CSV (csv:"-")

**Business Rules**:

- If `CreditDebit` is "DBIT", `Debit` should equal `Amount`
- If `CreditDebit` is "CRDT", `Credit` should equal `Amount`
- `AmountExclTax + AmountTax` should equal `Amount` when tax fields are used

**Note**: Helper methods (UpdateDebitCreditAmounts, UpdateNameFromParties, ToBuilder, etc.) were removed in v2.0.0. Transactions are now immutable value objects. Parsers construct transactions with all fields set correctly during parsing.

## Categorization Service Specification

### TransactionCategorizer Interface

```go
type TransactionCategorizer interface {
    Categorize(ctx context.Context, partyName string, isDebtor bool, amount, date, info string) (Category, error)
}
```

**Parameters**:

- `ctx`: Context for cancellation and timeouts
- `partyName`: Name of the party (creditor or debtor)
- `isDebtor`: true if this is a debit transaction
- `amount`: Transaction amount as string
- `date`: Transaction date as string
- `info`: Additional information (description, remittance info)

### OutputFormatter Interface

```go
type OutputFormatter interface {
    Header() []string
    Format(transactions []models.Transaction) ([][]string, error)
    Delimiter() rune
}
```

**Methods**:

- `Header()`: Returns CSV header row
- `Format()`: Converts transactions to CSV rows
- `Delimiter()`: Returns CSV delimiter character (e.g., ';', ',')

### Parser Interfaces

```go
type Parser interface {
    Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error)
}

type Validator interface {
    ValidateFormat(filePath string) (bool, error)
}

type CSVConverter interface {
    ConvertToCSV(ctx context.Context, inputFile, outputFile string) error
}

type LoggerConfigurable interface {
    SetLogger(logger logging.Logger)
}

type CategorizerConfigurable interface {
    SetCategorizer(categorizer models.TransactionCategorizer)
}

type BatchConverter interface {
    BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error)
}

type FullParser interface {
    Parser
    Validator
    CSVConverter
    LoggerConfigurable
    CategorizerConfigurable
    BatchConverter
}
```

**Categorization Algorithm**:

1. **Direct Mapping**: Check exact match in creditor/debtor mappings
2. **Keyword Matching**: Match against category keywords
3. **AI Fallback**: Use Gemini API if enabled and previous steps fail

**Rate Limiting**: AI calls limited by `GEMINI_REQUESTS_PER_MINUTE`

**Learning**: When auto-learn is enabled, AI categorizations are saved directly to main mapping files. When disabled, suggestions are saved to staging files (`staging_creditors.yaml`/`staging_debtors.yaml`) for manual review via `StagingStoreInterface`

## Configuration Specification

### Configuration Loading Order

1.  **Command-line flags** (highest priority)
2.  **Environment variables**
3.  **Configuration file** (`~/.camt-csv/camt-csv.yaml`)
4.  **Default values** (lowest priority)

### Configuration Options

| YAML Key (`camt-csv.yaml`) | Environment Variable | CLI Flag | Description | Default |
| :--- | :--- | :--- | :--- | :--- |
| `log.level` | `CAMT_LOG_LEVEL` | `--log-level` | Logging verbosity | `info` |
| `log.format` | `CAMT_LOG_FORMAT` | `--log-format` | Log output format (`text`, `json`) | `text` |
| `csv.delimiter` | `CAMT_CSV_DELIMITER` | `--csv-delimiter` | CSV output delimiter | `,` |
| `ai.enabled` | `CAMT_AI_ENABLED` | `--ai-enabled` | Enable/disable AI categorization | `false` |
| `ai.model` | `CAMT_AI_MODEL` | - | Gemini model for categorization | `gemini-2.0-flash` |
| `ai.api_key` | `GEMINI_API_KEY` | - | API key for Gemini | - |

## Error Handling Specification

### Error Types Hierarchy

```go
// Base error interface
type CAMTError interface {
    error
    Code() string
    Context() map[string]interface{}
}

// File operation errors
type FileNotFoundError struct {
    Path string
}

type FilePermissionError struct {
    Path      string
    Operation string
}

// Format validation errors
type InvalidFormatError struct {
    Path     string
    Expected string
    Actual   string
}

// Parsing errors
type ParseError struct {
    Path   string
    Line   int
    Column int
    Reason string
}

// Configuration errors
type ConfigurationError struct {
    Key     string
    Value   string
    Reason  string
}

// External service errors
type ExternalServiceError struct {
    Service string
    Code    int
    Message string
}
```

### Error Handling Patterns

**Wrapping Errors**:

```go
if err != nil {
    return fmt.Errorf("failed to parse CAMT file %s: %w", filePath, err)
}
```

**Custom Error Creation**:

```go
if !isValidFormat {
    return InvalidFormatError{
        Path:     filePath,
        Expected: "CAMT.053 XML",
        Actual:   detectedFormat,
    }
}
```

**Error Context**:

```go
log.WithError(err).WithFields(logrus.Fields{
    "file":   filePath,
    "parser": "camt",
    "line":   lineNumber,
}).Error("Parse error occurred")
```

## Integration Specifications

### File System Integration

**File Operations Contract**:

- All file operations MUST use proper resource cleanup
- Large files MUST be processed in streaming fashion
- Temporary files MUST use atomic rename pattern
- File permissions MUST be set appropriately (0644 for files, 0755 for directories)

**Example Pattern**:

```go
func processFile(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer func() {
        if err := file.Close(); err != nil {
            log.Warnf("Failed to close file: %v", err)
        }
    }()
    
    // Process file...
}
```

### External API Integration

**Gemini AI Integration Contract**:

- MUST respect rate limiting
- MUST handle API errors gracefully
- MUST provide fallback behavior
- MUST validate API responses

**Rate Limiting Implementation**:

```go
type RateLimiter struct {
    requests chan struct{}
    ticker   *time.Ticker
}

func (rl *RateLimiter) Allow() bool {
    select {
    case <-rl.requests:
        return true
    default:
        return false
    }
}
```

## Testing Specifications

### Unit Test Requirements

**Coverage Requirements**:

- Minimum 80% code coverage for all packages
- 100% coverage for critical paths (parsing, validation)
- All public methods MUST have tests

**Test Structure**:

```go
func TestParser_ParseFile(t *testing.T) {
    tests := []struct {
        name        string
        inputFile   string
        expectError bool
        expectCount int
        setupFunc   func() string // Returns temp file path
        cleanupFunc func(string)  // Cleanup temp file
    }{
        // Test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation...
        })
    }
}
```

### Integration Test Requirements

**File Processing Tests**:

- Test with real sample files
- Test with malformed input
- Test with large files (performance)
- Test with edge cases (empty files, single transaction)

**External Service Tests**:

- Mock external APIs in tests
- Test rate limiting behavior
- Test error handling and fallbacks

## Performance Specifications

### Performance Requirements

**File Processing**:

- MUST process files up to 100MB without excessive memory usage
- MUST complete validation in under 1 second for typical files
- MUST handle 10,000+ transactions efficiently

**Memory Usage**:

- MUST NOT load entire file into memory for large files
- MUST use streaming processing where possible
- MUST clean up resources promptly

**Benchmarking Requirements**:

```go
func BenchmarkParser_ParseFile(b *testing.B) {
    parser := NewParser()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.ParseFile("testdata/large_file.xml")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Security Specifications

### Input Validation

**File Path Validation**:

- MUST prevent directory traversal attacks
- MUST validate file extensions where appropriate
- MUST check file permissions before processing

**Data Sanitization**:

- MUST sanitize all user input
- MUST validate numeric inputs
- MUST escape output where necessary

### Sensitive Data Handling

**Configuration**:

- API keys MUST be loaded from environment variables
- MUST NOT log sensitive configuration values
- MUST use secure defaults

**File Handling**:

- Generated files MUST have appropriate permissions
- Temporary files MUST be cleaned up
- MUST NOT expose sensitive data in error messages

## Compliance Specifications

### Data Format Compliance

**ISO 20022 Compliance**:

- CAMT.053 parsing MUST follow ISO 20022 standard
- Currency codes MUST be ISO 4217 compliant
- Date formats MUST be consistent

**CSV Output Compliance**:

- MUST follow RFC 4180 CSV standard
- MUST handle special characters properly
- MUST provide consistent field ordering

This specification document serves as the authoritative reference for all API contracts and integration patterns within the CAMT-CSV project.
