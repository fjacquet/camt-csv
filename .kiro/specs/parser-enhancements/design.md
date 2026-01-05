# Design Document: Parser Enhancements

## Overview

This design implements three focused enhancements to the camt-csv parser system:

1. **Batch Aggregation**: Modify the batch processor to consolidate multiple files from the same bank account into single output files
2. **PDF Categorization**: Integrate the existing categorization system into the PDF parser
3. **Selma Categorization**: Integrate the existing categorization system into the Selma parser

The design follows KISS, DRY, and functional programming principles by leveraging existing infrastructure without reinventing functionality.

## Architecture

### Current Architecture Analysis

The existing system has:

- **Batch Processor** (`cmd/batch/batch.go`): Currently processes each file individually
- **Categorization System** (`internal/categorizer/categorizer.go`): Well-established three-tier system (direct mapping → keyword matching → AI fallback)
- **Parser Interface** (`internal/parser/parser.go`): Defines `CategorizerConfigurable` interface for categorization integration
- **PDF Parser** (`internal/pdfparser/pdfparser.go`): Parses PDF files but lacks categorization
- **Selma Parser** (`internal/selmaparser/selmaparser.go`): Parses Selma CSV files but lacks categorization

### Enhancement Strategy

The design leverages existing patterns and interfaces:

1. **Reuse Categorizer Interface**: Both PDF and Selma parsers will implement `CategorizerConfigurable` interface
2. **Extend Batch Logic**: Modify batch processor to group files by account and aggregate transactions
3. **Maintain Consistency**: Ensure all parsers produce identical CSV output format

## Components and Interfaces

### 1. Account Identifier Extraction

**Component**: `AccountIdentifier` utility
**Location**: `internal/common/account.go`

```go
type AccountIdentifier struct {
    ID     string
    Source string // "filename", "content", "default"
}

func ExtractAccountFromCAMTFilename(filename string) AccountIdentifier
func ExtractAccountFromPDFContent(transactions []models.Transaction) AccountIdentifier
func ExtractAccountFromSelmaContent(transactions []models.Transaction) AccountIdentifier
func SanitizeAccountID(accountID string) string
```

### 2. Batch Aggregation Engine

**Component**: `BatchAggregator`
**Location**: `internal/batch/aggregator.go`

```go
type FileGroup struct {
    AccountID    string
    Files        []string
    DateRange    DateRange
}

type BatchAggregator struct {
    logger logging.Logger
}

func (ba *BatchAggregator) GroupFilesByAccount(files []string) ([]FileGroup, error)
func (ba *BatchAggregator) AggregateTransactions(group FileGroup) ([]models.Transaction, error)
func (ba *BatchAggregator) GenerateOutputFilename(accountID string, dateRange DateRange) string
```

### 3. Enhanced PDF Parser

**Modification**: Add categorization to existing `PDFParser`
**Location**: `internal/pdfparser/pdfparser.go`

```go
type PDFParser struct {
    extractor   PDFExtractor
    categorizer models.TransactionCategorizer
    logger      logging.Logger
}

// Implement CategorizerConfigurable interface
func (p *PDFParser) SetCategorizer(categorizer models.TransactionCategorizer)
```

### 4. Enhanced Selma Parser

**Modification**: Add categorization to existing `SelmaParser`
**Location**: `internal/selmaparser/selmaparser.go`

```go
type SelmaParser struct {
    categorizer models.TransactionCategorizer
    logger      logging.Logger
}

// Implement CategorizerConfigurable interface
func (s *SelmaParser) SetCategorizer(categorizer models.TransactionCategorizer)
```

## Data Models

### Account Grouping

```go
type DateRange struct {
    Start time.Time
    End   time.Time
}

func (dr DateRange) String() string // Returns "2025-04-01_2025-06-30" format
func (dr DateRange) Merge(other DateRange) DateRange
```

### Categorization Statistics

```go
type CategorizationStats struct {
    Total         int
    Successful    int
    Failed        int
    Uncategorized int
}

func (cs CategorizationStats) LogSummary(logger logging.Logger, parserType string)
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Account-based File Aggregation

*For any* set of CAMT files with the same account number in their filenames, batch processing should produce exactly one consolidated output file per unique account number
**Validates: Requirements 1.1, 4.2**

### Property 2: Consolidated File Naming Convention

*For any* account identifier and date range, the consolidated output filename should follow the format "{account_id}*{start_date}*{end_date}.csv" with filesystem-safe characters
**Validates: Requirements 1.2, 7.1, 7.3**

### Property 3: Chronological Transaction Ordering

*For any* set of transactions from multiple files being aggregated, the final output should be sorted chronologically by transaction date
**Validates: Requirements 1.3**

### Property 4: Duplicate Transaction Preservation

*For any* duplicate transactions found across multiple input files, all transactions should be included in the output and warnings should be logged
**Validates: Requirements 1.4**

### Property 5: Source File Metadata Inclusion

*For any* aggregated output file, the header should contain a comment listing all source files that were merged
**Validates: Requirements 1.5**

### Property 6: PDF Categorization Integration

*For any* PDF transaction, the categorization system should be applied using the same three-tier strategy (direct mapping → keyword matching → AI fallback) as other parsers
**Validates: Requirements 2.1, 5.3**

### Property 7: Selma Categorization Integration

*For any* Selma transaction, the categorization system should be applied using the same three-tier strategy (direct mapping → keyword matching → AI fallback) as other parsers
**Validates: Requirements 3.1, 5.3**

### Property 8: Consistent CSV Output Format

*For any* parser type (CAMT, PDF, Selma, Revolut), the CSV output should contain identical column headers including category and subcategory columns
**Validates: Requirements 2.3, 3.3, 4.1**

### Property 9: Categorization Fallback Behavior

*For any* transaction that cannot be categorized by any strategy, the category should be set to "Uncategorized"
**Validates: Requirements 2.4, 3.4**

### Property 10: Categorization Statistics Logging

*For any* parser processing completion, categorization statistics (successful/failed/uncategorized counts) should be logged
**Validates: Requirements 2.5, 3.5**

### Property 11: Configuration Consistency

*For any* parser using categorization, the same YAML configuration files and AI settings should be used as existing parsers
**Validates: Requirements 5.2**

### Property 12: Auto-learning Mechanism Consistency

*For any* successful categorization result, the auto-learning mechanism should save the mapping to YAML files using the same pattern as existing parsers
**Validates: Requirements 5.4**

### Property 13: Account Identification from Filenames

*For any* CAMT filename following the pattern "CAMT.053_{account}*{dates}*{sequence}.csv", the account number should be correctly extracted
**Validates: Requirements 6.1**

### Property 14: Date Range Calculation

*For any* set of files with overlapping date ranges, the consolidated filename should use the overall date range spanning all input files
**Validates: Requirements 7.2**

### Property 15: Output Directory Organization

*For any* batch processing operation with a specified output directory, all consolidated files should be placed in that directory
**Validates: Requirements 7.4**

## Error Handling

### Account Identification Errors

- **Missing Account Info**: Use filename as fallback identifier
- **Invalid Filename Pattern**: Log warning and use base filename
- **Multiple Accounts in File**: Use first valid account, log warning

### Categorization Errors

- **Categorizer Not Set**: Log warning, skip categorization
- **Categorization Failure**: Assign "Uncategorized", log details
- **Configuration Missing**: Use default settings, log warning

### Aggregation Errors

- **File Read Errors**: Skip file, log error, continue with others
- **Parse Errors**: Skip malformed transactions, log details
- **Output Write Errors**: Fail fast with descriptive error

## Testing Strategy

### Unit Tests

- Account identifier extraction from various filename patterns
- Date range merging and formatting
- Categorization statistics calculation and logging
- Error handling for edge cases (missing files, malformed data)

### Property-Based Tests

Each correctness property will be implemented as a property-based test with minimum 100 iterations:

- **Property 1-15**: Generate random input data and verify universal properties hold
- **File Aggregation**: Generate multiple files with same/different accounts
- **Categorization**: Generate transactions with various party names and amounts
- **Output Format**: Verify consistent CSV structure across all parsers

### Integration Tests

- End-to-end batch processing with multiple file types
- PDF and Selma parsers with categorization enabled
- Cross-parser CSV output format validation
- Auto-learning mechanism verification

### Test Configuration

- Use Go's built-in testing framework with `testify` assertions
- Property tests run with minimum 100 iterations per property
- Each test tagged with format: **Feature: parser-enhancements, Property {number}: {property_text}**
- Mock external dependencies (filesystem, AI client) for unit tests
- Use real categorization YAML files for integration tests
