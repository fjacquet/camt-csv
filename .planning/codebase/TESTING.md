# Testing Patterns

**Analysis Date:** 2026-02-01

## Test Framework

**Runner:**
- Go built-in `testing` package with `go test` command
- Config: No configuration files (uses defaults from go.mod: Go 1.24.2)

**Assertion Library:**
- `github.com/stretchr/testify` v1.11.1
- Primary functions: `assert.Equal()`, `assert.True()`, `assert.False()`, `assert.Error()`, `assert.NotNil()`
- Error assertion: `require.NoError()`, `assert.ErrorIs()`, `assert.Contains()`

**Run Commands:**
```bash
make test              # Run all tests with -v verbose flag
make test-race         # Run tests with race detector enabled
make coverage          # Generate coverage report (HTML)
make coverage-summary  # Show coverage summary per package
go test -v ./...       # Direct test execution
go test -v -run TestFunctionName ./path/to/package  # Single test
go test -v -coverprofile=coverage.txt ./...  # Coverage profile
```

## Test File Organization

**Location:**
- Co-located with implementation: `*_test.go` files in same directory as `.go` files
- Example: `internal/models/transaction.go` → `internal/models/transaction_test.go`
- Internal (white-box) tests use: `*_internal_test.go` suffix
- Total: 69 test files across codebase

**Naming:**
- Package name uses `_test` suffix to separate test package: `package batch_test`
- Test functions use `TestFunctionName` pattern
- Test cases grouped in tables with descriptive names
- Example from `internal/models/transaction_test.go`:
  ```go
  package models

  func TestGetAmountAsDecimal(t *testing.T) {
      testCases := []struct {
          name     string
          amount   string
          expected string
      }{
          {"SimpleAmount", "123.45", "123.45"},
          {"AmountWithComma", "123,45", "123.45"},
      }
  }
  ```

**Structure:**
```
internal/
├── camtparser/
│   ├── camtparser.go
│   ├── adapter.go
│   ├── camtparser_test.go        # Black-box tests
│   ├── camtparser_internal_test.go # White-box tests
│   └── concurrent_processor_test.go
├── models/
│   ├── models.go
│   ├── transaction.go
│   └── transaction_test.go
```

## Test Structure

**Suite Organization:**
- Flat structure; no test suites/nested classes
- Table-driven tests for multiple scenarios
- Subtests with `t.Run()` for grouping related tests

**Example Table-Driven Test:**
```go
func TestGetAmountAsDecimal(t *testing.T) {
    testCases := []struct {
        name     string
        amount   string
        expected string
    }{
        {"SimpleAmount", "123.45", "123.45"},
        {"AmountWithComma", "123,45", "123.45"},
        {"NegativeAmount", "-123.45", "-123.45"},
        {"InvalidAmount", "not-a-number", "0"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            expected, _ := decimal.NewFromString(tc.expected)
            tx := &Transaction{Amount: ParseAmount(tc.amount)}
            result := tx.GetAmountAsDecimal()
            assert.True(t, expected.Equal(result),
                "GetAmountAsDecimal() with Amount=%s should return %s, got %s",
                tc.amount, tc.expected, result.String())
        })
    }
}
```

**Patterns:**

Setup/Teardown:
```go
func TestBatchFunc_MissingInputOutput(t *testing.T) {
    // Save original state
    originalInput := root.SharedFlags.Input
    originalOutput := root.SharedFlags.Output

    // Setup test
    root.SharedFlags.Input = ""
    root.SharedFlags.Output = ""

    // Cleanup (defer runs after test)
    defer func() {
        root.SharedFlags.Input = originalInput
        root.SharedFlags.Output = originalOutput
    }()

    // Test assertions
    assert.Equal(t, "", root.SharedFlags.Input)
}
```

Temporary Directories:
```go
func TestParseFile(t *testing.T) {
    // Create temporary directory (auto-cleaned up)
    tempDir := t.TempDir()
    testFile := filepath.Join(tempDir, "test_camt.xml")

    // Write test data
    err := os.WriteFile(testFile, []byte(xmlContent), 0600)
    require.NoError(t, err)

    // Test operations
}
```

Context Testing:
```go
func TestConcurrentProcessor_ProcessTransactions_UsesSequentialForSmall(t *testing.T) {
    logger := logging.NewLogrusAdapter("info", "text")
    processor := NewConcurrentProcessor(logger)

    transactions := processor.ProcessTransactions(context.Background(), entries, simpleProcessor)
    assert.Len(t, transactions, 50)
}
```

## Mocking

**Framework:** Hand-written mock structs implementing interfaces
- No external mocking library used
- Mocks embed interface types to ensure compliance
- Use compile-time verification: `var _ parser.FullParser = (*mockFullParser)(nil)`

**Example Mock Pattern:**
```go
// mockFullParser implements parser.FullParser for testing
type mockFullParser struct {
    validateErr    error
    validateResult bool
    convertErr     error
    parseErr       error
    transactions   []models.Transaction
}

func (m *mockFullParser) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
    if m.parseErr != nil {
        return nil, m.parseErr
    }
    return m.transactions, nil
}

func (m *mockFullParser) ValidateFormat(filePath string) (bool, error) {
    return m.validateResult, m.validateErr
}

// ... other interface methods

// Compile-time interface compliance check
var _ parser.FullParser = (*mockFullParser)(nil)
```

**What to Mock:**
- External dependencies: parsers, categorizers, loggers
- File system operations in unit tests (use `t.TempDir()` for integration tests)
- HTTP clients or API calls (not present in current codebase)
- Categorizer for parser tests to isolate parsing logic

**What NOT to Mock:**
- Standard library functions (encoding/xml, io, os)
- Internal utility functions (error handling, logging)
- Builder patterns (use real builders in tests)
- Transaction construction (test with real Transaction struct)

## Fixtures and Factories

**Test Data:**
- Embedded XML constants at package level: `const testXMLContent = "<?xml version=..."`
- CSV content literals in test functions
- Table-driven test data in struct slices

**Example Fixture Pattern:**
```go
const testXMLContent = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
    <BkToCstmrStmt>
        <Stmt>
            <Ntry>
                <Amt Ccy="EUR">100</Amt>
                <CdtDbtInd>DBIT</CdtDbtInd>
                ...
            </Ntry>
        </Stmt>
    </BkToCstmrStmt>
</Document>`

func TestParseFile(t *testing.T) {
    tempDir := t.TempDir()
    testFile := filepath.Join(tempDir, "test_camt.xml")
    err := os.WriteFile(testFile, []byte(testXMLContent), 0600)
    require.NoError(t, err)
    // ... test with fixture
}
```

**Factories:**
- Logger factory: `logging.NewLogrusAdapter("info", "text")`
- Adapter factories: `NewAdapter(logger)`, `NewConcurrentProcessor(logger)`
- Builder patterns: `models.NewTransactionBuilder().With...().Build()`

**Location:**
- Fixtures embedded in test files directly
- No shared fixture package or `fixtures/` directory
- Constants defined at top of test files near usage

## Coverage

**Requirements:** No enforced coverage target
- Coverage tracking enabled with `make coverage`
- HTML report generated to `coverage.html`
- Summary view available with `make coverage-summary`

**View Coverage:**
```bash
make coverage              # Generate HTML report
go tool cover -html=coverage.txt  # View coverage report
make coverage-summary      # Terminal summary per package
```

**Patterns:**
- All critical paths tested (Parse, Validate, Convert methods)
- Error cases tested with custom error returns from mocks
- Concurrency tested with race detector: `make test-race`
- Panic recovery tested by verifying fallback behavior (no panics)

## Test Types

**Unit Tests:**
- Scope: Individual functions and methods
- Approach: Isolated with mocks for dependencies
- Examples: `TestGetAmountAsDecimal()`, `TestCreditDebitMethods()`, `TestNewConcurrentProcessor()`
- Mock dependencies: Logger, Categorizer, external parsers
- Real dependencies: Transaction structs, utility functions
- Location: `*_test.go` files

**Integration Tests:**
- Scope: Multiple components working together
- Approach: Test parser with real categorizers and YAML stores
- Examples: `TestParseFile()` parsing actual XML to transactions
- Real file I/O: Using `t.TempDir()` for isolated file operations
- Minimal mocking: Only external systems (APIs) are mocked
- Location: `*_test.go` files with integration test markers in names

**Cross-Package Tests:**
- Location: `internal/integration/cross_parser_test.go`
- Test multiple parsers with shared test data
- Verify interface compliance across implementations

**E2E Tests:**
- Framework: None currently
- CLI commands tested through unit tests on handler functions
- File conversion tested through integration tests

## Common Patterns

**Async Testing:**
```go
func TestConcurrentProcessor_ProcessTransactions_UsesConcurrentForLarge(t *testing.T) {
    logger := logging.NewLogrusAdapter("debug", "text")
    processor := NewConcurrentProcessor(logger)

    entries := make([]models.Entry, 150)
    simpleProcessor := func(entry *models.Entry) models.Transaction {
        return models.Transaction{Amount: decimal.NewFromFloat(200)}
    }

    transactions := processor.ProcessTransactions(context.Background(), entries, simpleProcessor)
    assert.Len(t, transactions, 150)
}
```

**Cancellation Testing:**
```go
func TestBatchConvert_ContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Simulate cancellation

    count, err := adapter.BatchConvert(ctx, inputDir, outputDir)
    assert.Error(t, err)
    assert.Equal(t, context.Canceled, err)
}
```

**Error Testing:**
```go
func TestPDFConvert_InvalidFormat(t *testing.T) {
    logger := logging.NewLogrusAdapter("info", "text")
    mockParser := &mockFullParser{
        validateResult: false,
        validateErr:    nil,
    }

    err := common.ProcessFileWithError(context.Background(), mockParser, "input.pdf", "output.csv", true, logger)

    assert.Error(t, err)
    assert.ErrorIs(t, err, common.ErrInvalidFormat)
}

func TestPDFConvert_ConversionError(t *testing.T) {
    mockParser := &mockFullParser{
        validateResult: true,
        convertErr:     errors.New("PDF parsing failed"),
    }

    err := common.ProcessFileWithError(context.Background(), mockParser, "input.pdf", "output.csv", true, logger)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error converting to CSV")
}
```

**Receiver/Interface Compliance Testing:**
```go
// Verify mock implements interface at compile time
var _ parser.FullParser = (*mockFullParser)(nil)
```

## Test Helpers

**Custom Loggers in Tests:**
```go
logger := logging.NewLogrusAdapter("info", "text")
logger := logging.NewLogrusAdapter("debug", "text")  // For detailed output
```

**Assertion Helpers:**
- Use `assert.*` for non-blocking assertions
- Use `require.*` for assertions that should stop test execution on failure
- Use `t.Run()` for subtests with cleanup

**Temporary Files:**
```go
tempDir := t.TempDir()  // Auto-cleaned directory
testFile := filepath.Join(tempDir, "test.xml")
os.WriteFile(testFile, []byte(content), 0600)
```

## Test Running and Debugging

**Run Specific Test:**
```bash
go test -v -run TestFunctionName ./path/to/package
```

**Run Tests in Package:**
```bash
go test -v ./internal/camtparser
```

**Run All Tests:**
```bash
make test         # Verbose
make test-race    # With race detector
```

**Debug Failing Test:**
1. Reduce test to minimal case
2. Add logging fields to understand state
3. Use `t.Logf()` for debug output
4. Run with `-v` flag for verbose output

---

*Testing analysis: 2026-02-01*
