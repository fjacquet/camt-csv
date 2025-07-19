# API Specifications - CAMT-CSV Project

## Overview

This document provides formal specifications for all APIs, interfaces, and contracts within the CAMT-CSV project. These specifications serve as the foundation for specification-driven development, ensuring consistency and reliability across all components.

## Parser Interface Specification

### Core Parser Interface

```go
type Parser interface {
    ParseFile(filePath string) ([]models.Transaction, error)
    ValidateFormat(filePath string) (bool, error)
    ConvertToCSV(inputFile, outputFile string) error
    WriteToCSV(transactions []models.Transaction, csvFile string) error
    SetLogger(logger *logrus.Logger)
}
```

#### ParseFile Method Specification

**Signature**: `ParseFile(filePath string) ([]models.Transaction, error)`

**Purpose**: Extract financial transactions from a source file

**Preconditions**:

- `filePath` MUST be a valid file path string
- File MUST exist and be readable
- File MUST be in the format supported by the specific parser

**Postconditions**:

- Returns slice of valid `Transaction` objects on success
- Returns empty slice (not nil) if no transactions found
- Returns error if file format is invalid or I/O error occurs
- All returned transactions MUST have valid `Date`, `Amount`, and `Currency` fields

**Error Conditions**:

```go
// File not found
FileNotFoundError{Path: string}

// Invalid format
InvalidFormatError{Path: string, Expected: string, Actual: string}

// Parse error with context
ParseError{Path: string, Line: int, Reason: string}
```

**Implementation Requirements**:

- MUST validate file format before parsing
- MUST handle malformed data gracefully
- MUST use structured logging with file context
- MUST close file handles properly (defer pattern)

#### ValidateFormat Method Specification

**Signature**: `ValidateFormat(filePath string) (bool, error)`

**Purpose**: Check if file conforms to expected format without full parsing

**Preconditions**:

- `filePath` MUST be a valid file path string

**Postconditions**:

- Returns `(true, nil)` for valid format
- Returns `(false, nil)` for invalid format (not an error condition)
- Returns `(false, error)` only for I/O errors

**Performance Requirements**:

- MUST complete validation in O(1) time relative to file size
- SHOULD read minimal data necessary for validation
- MUST NOT load entire file into memory

#### ConvertToCSV Method Specification

**Signature**: `ConvertToCSV(inputFile, outputFile string) error`

**Purpose**: End-to-end conversion from source format to CSV

**Preconditions**:

- `inputFile` MUST exist and be readable
- `outputFile` directory MUST be writable
- Parser MUST support the input file format

**Postconditions**:

- Creates valid CSV file at `outputFile` path
- CSV MUST use configured delimiter (from `common.Delimiter`)
- CSV MUST include standard headers
- Creates output directory if it doesn't exist

**Atomicity Guarantee**:

- On failure, MUST NOT leave partial output file
- SHOULD use temporary file and atomic rename

#### WriteToCSV Method Specification

**Signature**: `WriteToCSV(transactions []models.Transaction, csvFile string) error`

**Purpose**: Export transaction slice to CSV format

**Preconditions**:

- `transactions` can be empty but MUST NOT be nil
- `csvFile` directory MUST be writable

**Postconditions**:

- Creates valid CSV file with headers
- Empty input produces CSV with headers only
- Uses configured CSV delimiter
- All numeric values formatted consistently

**Data Integrity Requirements**:

- Decimal amounts MUST preserve precision
- Dates MUST be in DD.MM.YYYY format
- Currency codes MUST be ISO 4217 compliant

#### SetLogger Method Specification

**Signature**: `SetLogger(logger *logrus.Logger)`

**Purpose**: Configure logging for parser instance

**Preconditions**:

- `logger` can be nil (disables logging)

**Postconditions**:

- All parser operations use provided logger
- Logger propagated to internal components
- Structured logging with consistent field names

## Data Model Specifications

### Transaction Model

```go
type Transaction struct {
    // Core Fields (REQUIRED)
    Date            string          `json:"date" validate:"required,date_format=DD.MM.YYYY"`
    Amount          decimal.Decimal `json:"amount" validate:"required"`
    Currency        string          `json:"currency" validate:"required,iso4217"`
    CreditDebit     string          `json:"credit_debit" validate:"required,oneof=CRDT DBIT"`
    
    // Identification Fields
    BookkeepingNumber string `json:"bookkeeping_number"`
    EntryReference    string `json:"entry_reference"`
    Reference         string `json:"reference"`
    
    // Party Information
    Name              string `json:"name"`
    PartyName         string `json:"party_name"`
    PartyIBAN         string `json:"party_iban" validate:"omitempty,iban"`
    
    // Transaction Details
    Description       string          `json:"description"`
    Category          string          `json:"category"`
    Type              string          `json:"type"`
    Status            string          `json:"status"`
    
    // Financial Details
    Debit             decimal.Decimal `json:"debit"`
    Credit            decimal.Decimal `json:"credit"`
    AmountExclTax     decimal.Decimal `json:"amount_excl_tax"`
    AmountTax         decimal.Decimal `json:"amount_tax"`
    TaxRate           decimal.Decimal `json:"tax_rate"`
    Fees              decimal.Decimal `json:"fees"`
    
    // Investment Fields
    Investment        string          `json:"investment"`
    Fund              string          `json:"fund"`
    NumberOfShares    float64         `json:"number_of_shares"`
    
    // Additional Fields
    ValueDate         string          `json:"value_date" validate:"omitempty,date_format=DD.MM.YYYY"`
    IBAN              string          `json:"iban" validate:"omitempty,iban"`
    BankTxCode        string          `json:"bank_tx_code"`
    AccountServicer   string          `json:"account_servicer"`
    
    // Legacy Compatibility
    Payer             string          `json:"payer"`
    Payee             string          `json:"payee"`
    Recipient         string          `json:"recipient"`
    DebitFlag         bool            `json:"debit_flag"`
    
    // Multi-currency Support
    OriginalCurrency  string          `json:"original_currency" validate:"omitempty,iso4217"`
    OriginalAmount    decimal.Decimal `json:"original_amount"`
    ExchangeRate      decimal.Decimal `json:"exchange_rate"`
}
```

#### Validation Rules

**Required Fields**:

- `Date`: Must be in DD.MM.YYYY format
- `Amount`: Must be valid decimal, can be negative
- `Currency`: Must be valid ISO 4217 currency code
- `CreditDebit`: Must be either "CRDT" or "DBIT"

**Format Constraints**:

- IBAN fields: Must pass IBAN validation if present
- Date fields: Must be parseable as DD.MM.YYYY
- Decimal fields: Must use `decimal.Decimal` for precision

**Business Rules**:

- If `CreditDebit` is "DBIT", `Debit` should equal `Amount`
- If `CreditDebit` is "CRDT", `Credit` should equal `Amount`
- `AmountExclTax + AmountTax` should equal `Amount` when tax fields are used

#### Helper Methods Specification

```go
// UpdateDebitCreditAmounts ensures Debit/Credit fields match CreditDebit indicator
func (t *Transaction) UpdateDebitCreditAmounts()

// UpdateNameFromParties sets Name field based on transaction direction
func (t *Transaction) UpdateNameFromParties()

// Validate performs comprehensive validation of all fields
func (t *Transaction) Validate() error
```

## Categorization Service Specification

### Categorizer Interface

```go
type Categorizer interface {
    CategorizeTransaction(tx CategorizeTransaction) (*Category, error)
    UpdateCreditorCategory(creditor, category string) error
    UpdateDebitorCategory(debitor, category string) error
    SetTestCategoryStore(store *CategoryStore)
}
```

### CategorizeTransaction Specification

**Input**: `CategorizeTransaction` struct

```go
type CategorizeTransaction struct {
    PartyName   string
    IsDebtor    bool
    Description string
    Amount      decimal.Decimal
    Currency    string
}
```

**Algorithm**:

1. **Direct Mapping**: Check exact match in creditor/debitor mappings
2. **Keyword Matching**: Match against category keywords
3. **AI Fallback**: Use Gemini API if enabled and previous steps fail

**Rate Limiting**: AI calls limited by `GEMINI_REQUESTS_PER_MINUTE`

**Learning**: Successful AI categorizations automatically saved to mappings

## Configuration Specification

### Environment Variables

| Variable | Type | Default | Validation | Description |
|----------|------|---------|------------|-------------|
| `GEMINI_API_KEY` | string | "" | non-empty if AI enabled | Google AI API key |
| `GEMINI_MODEL` | string | "gemini-2.0-flash" | valid model name | Gemini model identifier |
| `GEMINI_REQUESTS_PER_MINUTE` | int | 10 | 1-1000 | API rate limit |
| `USE_AI_CATEGORIZATION` | bool | false | true/false | Enable AI categorization |
| `LOG_LEVEL` | string | "info" | trace/debug/info/warn/error | Logging verbosity |
| `LOG_FORMAT` | string | "text" | text/json | Log output format |
| `CSV_DELIMITER` | string | "," | single character | CSV field separator |
| `DATA_DIR` | string | "" | valid directory path | Custom data directory |

### Configuration Loading Order

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file** (`~/.camt-csv/config.yaml`)
4. **Default values** (lowest priority)

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
