# Migration Guide - CAMT-CSV v2.0

## Overview

This guide helps developers migrate from the old architecture to the new dependency injection-based architecture introduced in v2.0. While the CLI interface remains unchanged, the internal APIs have been significantly improved.

## Breaking Changes Summary

### 1. Global State Elimination

**Old (Deprecated):**
```go
// Global singleton pattern - DEPRECATED
categorizer := categorizer.GetDefaultCategorizer()
```

**New (Recommended):**
```go
// Dependency injection pattern
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}
categorizer := container.Categorizer
```

### 2. Parser Construction

**Old (Still Works):**
```go
// Direct construction without logger
parser := camtparser.NewParser()
```

**New (Recommended):**
```go
// Constructor with logger dependency
logger := logging.NewLogrusAdapter("info", "text")
parser := camtparser.NewParser(logger)
```

### 3. Configuration Access

**Old (Deprecated):**
```go
// Global configuration access - DEPRECATED
config := config.GetGlobalConfig()
```

**New (Recommended):**
```go
// Instance-based configuration
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}
// Access config through container or pass explicitly
```

## Transaction Model Backward Compatibility

### Enhanced Backward Compatibility Methods

The Transaction model now includes enhanced backward compatibility methods with direction-based logic:

**GetPayee() Method:**
```go
// For debit transactions: returns payee (who receives money)
// For credit transactions: returns payer (who sent money to us)
otherParty := tx.GetPayee()
```

**GetPayer() Method:**
```go
// For debit transactions: returns payer (account holder)
// For credit transactions: returns payee (account holder)
accountHolder := tx.GetPayer()
```

**GetAmountAsFloat() Method:**
```go
// Deprecated but still available for backward compatibility
amount := tx.GetAmountAsFloat() // May lose precision

// Recommended for new code:
amount := tx.GetAmountAsDecimal() // Precise decimal arithmetic
```

### Migration Path for Transaction Access

**Old Pattern (Still Works):**
```go
// Legacy access patterns continue to work
payee := tx.GetPayee()
payer := tx.GetPayer()
amount := tx.GetAmountAsFloat()
```

**New Pattern (Recommended):**
```go
// Direct field access for clarity
payee := tx.Payee
payer := tx.Payer
amount := tx.GetAmountAsDecimal()

// Or use counterparty for "other party" logic
counterparty := tx.GetCounterparty()

// Or use TransactionBuilder for new transactions
tx, err := models.NewTransactionBuilder().
    WithPayer("John Doe", "CH1234567890").
    WithPayee("Acme Corp", "CH0987654321").
    WithAmountFromFloat(100.50, "CHF").
    AsDebit().
    Build()
```
```go
// Load configuration explicitly
config, err := config.Load()
if err != nil {
    log.Fatal(err)
}
```

## Migration Steps

### Step 1: Update Imports

Add new imports for dependency injection:

```go
import (
    "fjacquet/camt-csv/internal/container"
    "fjacquet/camt-csv/internal/logging"
    // ... other imports
)
```

### Step 2: Replace Global Singletons

**Before:**
```go
func processTransactions(transactions []models.Transaction) error {
    categorizer := categorizer.GetDefaultCategorizer()
    
    for _, tx := range transactions {
        categorized, err := categorizer.CategorizeTransaction(tx)
        if err != nil {
            return err
        }
        // process categorized transaction
    }
    return nil
}
```

**After:**
```go
func processTransactions(transactions []models.Transaction, cat *categorizer.Categorizer) error {
    ctx := context.Background()
    
    for _, tx := range transactions {
        category, err := cat.Categorize(ctx, tx)
        if err != nil {
            return err
        }
        tx.Category = category.Name
        // process categorized transaction
    }
    return nil
}

// In main or setup function:
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}

err = processTransactions(transactions, container.Categorizer)
```

### Step 3: Update Parser Usage

**Before:**
```go
func parseFile(filePath string) ([]models.Transaction, error) {
    parser := camtparser.NewParser()
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    return parser.Parse(file)
}
```

**After:**
```go
func parseFile(filePath string, logger logging.Logger) ([]models.Transaction, error) {
    parser := camtparser.NewParser(logger)
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    return parser.Parse(file)
}

// Or use container:
func parseFileWithContainer(filePath string, container *container.Container) ([]models.Transaction, error) {
    parser, err := container.GetParser(parser.CAMT)
    if err != nil {
        return nil, err
    }
    
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    return parser.Parse(file)
}
```

### Step 4: Update Error Handling

**Before:**
```go
if err != nil {
    log.Printf("Error: %v", err)
    return err
}
```

**After:**
```go
if err != nil {
    // Use proper error inspection
    var parseErr *parsererror.ParseError
    if errors.As(err, &parseErr) {
        logger.Error("Parse error occurred",
            logging.Field{Key: "parser", Value: parseErr.Parser},
            logging.Field{Key: "field", Value: parseErr.Field})
    }
    return fmt.Errorf("processing failed: %w", err)
}
```

### Step 5: Update Transaction Construction

**Before:**
```go
tx := models.Transaction{
    Date:        parseDate(dateStr),
    Amount:      parseAmount(amountStr),
    Description: description,
    PartyName:   partyName,
}
```

**After (Recommended):**
```go
tx, err := models.NewTransactionBuilder().
    WithDate(dateStr).
    WithAmount(parseAmount(amountStr), currency).
    WithDescription(description).
    WithPayer(partyName, partyIBAN).
    AsDebit().
    Build()
if err != nil {
    return fmt.Errorf("transaction construction failed: %w", err)
}
```

## Configuration File Migration

### Old Format (Still Supported)

Environment variables and CLI flags continue to work:

```bash
export GEMINI_API_KEY=your_key
export LOG_LEVEL=debug
./camt-csv camt -i file.xml -o output.csv
```

### New Format (Recommended)

Create `~/.camt-csv/config.yaml`:

```yaml
log:
  level: "debug"
  format: "json"
csv:
  delimiter: ","
  include_headers: true
ai:
  enabled: true
  model: "gemini-2.0-flash"
  api_key: "your_key_here"  # Or use GEMINI_API_KEY env var
```

## File Renaming

### Debtor Configuration File

**Old:** `database/debitors.yaml`
**New:** `database/debtors.yaml`

**Migration:**
```bash
# Rename your existing file
mv database/debitors.yaml database/debtors.yaml
```

The application maintains backward compatibility with the old filename.

## Testing Migration

### Old Testing Pattern

**Before:**
```go
func TestParser(t *testing.T) {
    parser := camtparser.NewParser()
    // Test without proper dependency injection
}
```

**After:**
```go
func TestParser(t *testing.T) {
    mockLogger := &logging.MockLogger{}
    parser := camtparser.NewParser(mockLogger)
    
    // Test with injected dependencies
    result, err := parser.Parse(strings.NewReader(testData))
    assert.NoError(t, err)
    assert.Len(t, result, expectedCount)
    
    // Verify logging behavior
    assert.Contains(t, mockLogger.Entries, "Starting parse operation")
}
```

### Mock Dependencies

Use the provided mock implementations:

```go
// Mock logger for testing
mockLogger := &logging.MockLogger{}

// Mock AI client for testing
mockAI := &categorizer.MockAIClient{
    CategorizeFunc: func(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
        tx.Category = "TestCategory"
        return tx, nil
    },
}

// Mock category store for testing
mockStore := &store.MockCategoryStore{
    Categories: []models.CategoryConfig{
        {Name: "TestCategory", Keywords: []string{"test"}},
    },
}
```

## Deprecation Timeline

### v2.0.0 (Current)
- New architecture introduced
- Old APIs marked as deprecated with warnings
- Full backward compatibility maintained

### v2.1.0 (Planned)
- Deprecation warnings added to logs
- Migration examples in documentation

### v3.0.0 (Future)
- Deprecated APIs removed
- Breaking changes for internal APIs only
- CLI interface remains unchanged

## Common Migration Issues

### Issue 1: Import Errors

**Problem:**
```
cannot find package "fjacquet/camt-csv/internal/container"
```

**Solution:**
Update your go.mod and run:
```bash
go mod tidy
```

### Issue 2: Logger Not Available

**Problem:**
```go
// This won't work anymore
parser.logger.Info("message")
```

**Solution:**
```go
// Use GetLogger() method
parser.GetLogger().Info("message")
```

### Issue 3: Global Config Access

**Problem:**
```go
// Deprecated global access
config := config.GetGlobalConfig()
```

**Solution:**
```go
// Load config explicitly
config, err := config.Load()
if err != nil {
    log.Fatal(err)
}
```

## Benefits of Migration

### Improved Testability
- All dependencies can be mocked
- No global state interference
- Isolated unit tests

### Better Error Handling
- Structured error types with context
- Proper error wrapping and inspection
- Detailed error messages

### Enhanced Logging
- Framework-agnostic logging interface
- Structured logging with fields
- Configurable log levels and formats

### Performance Improvements
- Lazy initialization of expensive resources
- String operation optimizations
- Pre-allocation for better memory usage

## Getting Help

### Documentation
- [Architecture Documentation](architecture.md)
- [Design Principles](design-principles.md)
- [User Guide](user-guide.md)

### Code Examples
- Check the `cmd/` directory for updated CLI implementations
- Review test files for dependency injection patterns
- See `internal/container/container_test.go` for container usage

### Migration Support
- Create an issue on GitHub for migration questions
- Include your current code patterns and desired outcome
- Reference this migration guide in your issue

## Conclusion

The new architecture provides significant improvements in maintainability, testability, and performance while maintaining full backward compatibility for CLI users. The migration primarily affects internal API usage and provides a cleaner, more robust foundation for future development.

Take advantage of the deprecation period to gradually migrate your code, and don't hesitate to reach out for support during the transition.