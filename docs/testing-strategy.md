# Testing Strategy - CAMT-CSV Project

## Overview

This document defines the comprehensive testing strategy for the CAMT-CSV project, ensuring high code quality, reliability, and maintainability through systematic testing approaches.

## Testing Philosophy

### Core Principles

1. **Test Pyramid**: More unit tests, fewer integration tests, minimal end-to-end tests
2. **Fast Feedback**: Tests should run quickly to enable rapid development cycles
3. **Isolation**: Tests should be independent and not affect each other
4. **Deterministic**: Tests should produce consistent results across environments
5. **Maintainable**: Tests should be easy to understand and modify

### Quality Gates

- **Minimum Coverage**: 80% code coverage for all packages
- **Critical Path Coverage**: 100% coverage for parsing and validation logic
- **Performance**: All tests must complete within 30 seconds
- **Reliability**: Tests must pass consistently (>99% success rate)

## Test Categories

### 1. Unit Tests

**Purpose**: Test individual functions and methods in isolation

**Scope**:

- Pure functions (data transformations, calculations)
- Business logic components
- Utility functions
- Error handling paths

**Structure**:

```go
func TestParser_ParseTransaction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected models.Transaction
        wantErr  bool
    }{
        {
            name:  "valid transaction",
            input: `<transaction>...</transaction>`,
            expected: models.Transaction{
                Date:   "01.01.2024",
                Amount: decimal.NewFromFloat(100.50),
            },
            wantErr: false,
        },
        {
            name:    "invalid XML",
            input:   `<invalid>`,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseTransaction(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected.Date, result.Date)
            assert.True(t, tt.expected.Amount.Equal(result.Amount))
        })
    }
}
```

**Coverage Requirements**:

- All public methods must have tests
- All error conditions must be tested
- Edge cases and boundary conditions must be covered

### 2. Integration Tests

**Purpose**: Test component interactions and external dependencies

**Scope**:

- Parser integration with file system
- Categorizer integration with AI services
- Configuration loading and validation
- CLI command execution

**Test Structure**:

```go
func TestCAMTParser_Integration(t *testing.T) {
    // Setup test environment
    tempDir := t.TempDir()
    testFile := filepath.Join(tempDir, "test.xml")
    outputFile := filepath.Join(tempDir, "output.csv")
    
    // Create test data
    err := os.WriteFile(testFile, []byte(validCAMTXML), 0644)
    require.NoError(t, err)
    
    // Execute integration
    parser := camtparser.New()
    err = parser.ConvertToCSV(testFile, outputFile)
    
    // Verify results
    assert.NoError(t, err)
    assert.FileExists(t, outputFile)
    
    // Verify CSV content
    content, err := os.ReadFile(outputFile)
    require.NoError(t, err)
    
    lines := strings.Split(string(content), "\n")
    assert.Contains(t, lines[0], "Date,Description,Amount")
    assert.Greater(t, len(lines), 1)
}
```

### 3. End-to-End Tests

**Purpose**: Test complete user workflows

**Scope**:

- CLI command execution
- File processing workflows
- Error handling and recovery

**Test Structure**:

```go
func TestCLI_EndToEnd(t *testing.T) {
    // Setup test environment
    tempDir := t.TempDir()
    inputFile := createTestCAMTFile(t, tempDir)
    outputFile := filepath.Join(tempDir, "output.csv")
    
    // Execute CLI command
    cmd := exec.Command("camt-csv", "convert", 
        "--input", inputFile,
        "--output", outputFile,
        "--log-level", "debug")
    
    output, err := cmd.CombinedOutput()
    
    // Verify execution
    assert.NoError(t, err, "CLI command failed: %s", output)
    assert.FileExists(t, outputFile)
    
    // Verify output quality
    transactions, err := readCSVTransactions(outputFile)
    assert.NoError(t, err)
    assert.Greater(t, len(transactions), 0)
}
```

## Testing Patterns

### 1. Table-Driven Tests

**When to Use**: Testing multiple scenarios with similar structure

```go
func TestCategorizeTransaction(t *testing.T) {
    tests := []struct {
        name        string
        description string
        partyName   string
        expected    string
    }{
        {"grocery store", "MIGROS ZURICH", "MIGROS", "Groceries"},
        {"transport", "SBB TICKET", "SBB", "Transportation"},
        {"unknown", "UNKNOWN MERCHANT", "UNKNOWN", "Uncategorized"},
    }
    
    categorizer := NewCategorizer()
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tx := models.CategorizeTransaction{
                Description: tt.description,
                PartyName:   tt.partyName,
            }
            
            category, err := categorizer.CategorizeTransaction(tx)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, category.Name)
        })
    }
}
```

### 2. Mock Testing

**When to Use**: Testing with external dependencies

```go
type MockAIService struct {
    mock.Mock
}

func (m *MockAIService) CategorizeTransaction(ctx context.Context, tx models.CategorizeTransaction) (*models.Category, error) {
    args := m.Called(ctx, tx)
    return args.Get(0).(*models.Category), args.Error(1)
}

func TestCategorizer_WithAIFallback(t *testing.T) {
    mockAI := new(MockAIService)
    mockAI.On("CategorizeTransaction", mock.Anything, mock.Anything).
        Return(&models.Category{Name: "AI_Category"}, nil)
    
    categorizer := NewCategorizer(WithAIService(mockAI))
    
    tx := models.CategorizeTransaction{Description: "Unknown Transaction"}
    category, err := categorizer.CategorizeTransaction(tx)
    
    assert.NoError(t, err)
    assert.Equal(t, "AI_Category", category.Name)
    mockAI.AssertExpectations(t)
}
```

### 3. Test Fixtures

**Purpose**: Reusable test data and setup

```go
// testdata/fixtures.go
package testdata

var ValidCAMTXML = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
    <BkToCstmrStmt>
        <!-- Valid CAMT.053 content -->
    </BkToCstmrStmt>
</Document>`

func CreateTempCAMTFile(t *testing.T, dir string) string {
    file := filepath.Join(dir, "test.xml")
    err := os.WriteFile(file, []byte(ValidCAMTXML), 0644)
    require.NoError(t, err)
    return file
}
```

## Test Data Management

### 1. Sample Files

**Location**: `samples/` directory
**Purpose**: Real-world test data for integration tests

```
samples/
├── camt/
│   ├── valid_statement.xml
│   ├── empty_statement.xml
│   └── malformed.xml
├── pdf/
│   ├── bank_statement.pdf
│   └── complex_layout.pdf
├── csv/
│   ├── revolut_sample.csv
│   └── selma_sample.csv
└── expected/
    ├── camt_expected.csv
    └── revolut_expected.csv
```

### 2. Test Data Generation

```go
func GenerateTestTransaction() models.Transaction {
    return models.Transaction{
        Date:        "01.01.2024",
        Amount:      decimal.NewFromFloat(rand.Float64() * 1000),
        Currency:    "CHF",
        CreditDebit: []string{"CRDT", "DBIT"}[rand.Intn(2)],
        Description: fmt.Sprintf("Test Transaction %d", rand.Int()),
    }
}

func GenerateTestTransactions(count int) []models.Transaction {
    transactions := make([]models.Transaction, count)
    for i := 0; i < count; i++ {
        transactions[i] = GenerateTestTransaction()
    }
    return transactions
}
```

## Performance Testing

### 1. Benchmarks

**Purpose**: Measure and track performance over time

```go
func BenchmarkParser_ParseFile(b *testing.B) {
    parser := camtparser.New()
    testFile := "testdata/large_statement.xml"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.ParseFile(testFile)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkCategorizer_CategorizeTransaction(b *testing.B) {
    categorizer := NewCategorizer()
    tx := models.CategorizeTransaction{
        Description: "MIGROS ZURICH",
        PartyName:   "MIGROS",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := categorizer.CategorizeTransaction(tx)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 2. Load Testing

**Purpose**: Test behavior under high load

```go
func TestParser_LargeFile(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping large file test in short mode")
    }
    
    // Generate large test file (10MB+)
    largeFile := generateLargeCAMTFile(t, 10000) // 10k transactions
    
    parser := camtparser.New()
    
    start := time.Now()
    transactions, err := parser.ParseFile(largeFile)
    duration := time.Since(start)
    
    assert.NoError(t, err)
    assert.Equal(t, 10000, len(transactions))
    assert.Less(t, duration, 10*time.Second, "Large file parsing took too long")
}
```

## Test Organization

### 1. Directory Structure

```
test/
├── unit/           # Unit tests (alongside source code)
├── integration/    # Integration tests
├── e2e/           # End-to-end tests
├── benchmarks/    # Performance benchmarks
├── testdata/      # Test data and fixtures
└── helpers/       # Test utilities and helpers
```

### 2. Test Naming Conventions

```go
// Unit tests: Test<Type>_<Method>_<Scenario>
func TestCAMTParser_ParseFile_ValidInput(t *testing.T) {}
func TestCAMTParser_ParseFile_InvalidFormat(t *testing.T) {}

// Integration tests: TestIntegration_<Component>_<Scenario>
func TestIntegration_Parser_FileProcessing(t *testing.T) {}

// End-to-end tests: TestE2E_<Workflow>_<Scenario>
func TestE2E_ConvertCommand_Success(t *testing.T) {}
```

## Test Execution

### 1. Local Development

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./internal/camtparser

# Run benchmarks
go test -bench=. ./...

# Run tests with race detection
go test -race ./...
```

### 2. CI/CD Pipeline

```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [1.21, 1.22]
    
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: ${{ matrix.go-version }}
    
    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## Test Quality Assurance

### 1. Test Review Checklist

- [ ] Tests are independent and can run in any order
- [ ] Tests have descriptive names explaining what they test
- [ ] Tests cover both happy path and error conditions
- [ ] Tests use appropriate assertions with helpful messages
- [ ] Tests clean up resources (temp files, connections)
- [ ] Tests run quickly (< 1 second per test)

### 2. Coverage Analysis

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage by function
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### 3. Test Maintenance

- **Regular Review**: Review and update tests when code changes
- **Flaky Test Detection**: Monitor test reliability and fix unstable tests
- **Performance Monitoring**: Track test execution time and optimize slow tests
- **Dependency Updates**: Keep test dependencies up to date

## Continuous Improvement

### 1. Metrics Tracking

- Test execution time trends
- Code coverage trends
- Test failure rates
- Performance benchmark results

### 2. Test Automation

- Automatic test generation for new parsers
- Property-based testing for data transformations
- Mutation testing to verify test quality

### 3. Team Practices

- **Test-Driven Development**: Write tests before implementation
- **Code Reviews**: Include test review in code review process
- **Knowledge Sharing**: Regular sessions on testing best practices
- **Tool Evaluation**: Continuously evaluate new testing tools and techniques

This testing strategy ensures comprehensive coverage and high quality for the CAMT-CSV project while maintaining development velocity and reliability.
