# Deprecation Timeline and Breaking Changes

## Overview

This document outlines the deprecation timeline for CAMT-CSV v2.0 refactoring and provides guidance for migrating away from deprecated APIs.

## Deprecation Policy

CAMT-CSV follows semantic versioning with the following deprecation policy:

- **Minor versions (v2.x)**: Deprecated APIs are marked with warnings but remain functional
- **Major versions (v3.0)**: Deprecated APIs are removed entirely
- **Deprecation period**: Minimum 6 months between deprecation warning and removal

## Current Status (v2.0.0)

### ✅ Already Removed (v2.0.0)

The following functions and methods were removed in v2.0.0:

#### Global Singleton Functions
- `categorizer.GetDefaultCategorizer()` → Use `container.NewContainer()` and access `Categorizer`
- `categorizer.CategorizeTransaction()` → Use `categorizer.Categorize()` with context
- `config.GetGlobalConfig()` → Use `config.Load()` with dependency injection
- `config.GetCSVDelimiter()` → Access through configuration object
- `config.GetLogLevel()` → Access through configuration object
- `config.IsAIEnabled()` → Access through configuration object
- `factory.GetParser()` → Use `factory.GetParserWithLogger()` or container
- `logging.GetLogger()` → Use `logging.NewLogrusAdapter()` with dependency injection

#### Deprecated Methods
- `CategoryStore.LoadDebitorMappings()` → Use `LoadDebtorMappings()`
- `CategoryStore.SaveDebitorMappings()` → Use `SaveDebtorMappings()`
- `DirectMappingStrategy.UpdateDebitorMapping()` → Use `UpdateDebtorMapping()`

### ⚠️ Currently Deprecated (Will be removed in v3.0.0)

The following functions are marked as deprecated and will be removed in v3.0.0:

#### Transaction Backward Compatibility Methods
```go
// Deprecated: Use GetCounterparty() or access fields directly
func (t *Transaction) GetPayee() string

// Deprecated: Use GetCounterparty() or access fields directly  
func (t *Transaction) GetPayer() string

// Deprecated: Use GetAmountAsDecimal() for precise calculations
func (t *Transaction) GetAmountAsFloat() float64

// Deprecated: Use GetDebitAsDecimal() for precise calculations
func (t *Transaction) GetDebitAsFloat() float64

// Deprecated: Use GetCreditAsDecimal() for precise calculations
func (t *Transaction) GetCreditAsFloat() float64

// Deprecated: Use GetFeesAsDecimal() for precise calculations
func (t *Transaction) GetFeesAsFloat() float64

// Deprecated: Use TransactionBuilder pattern
func (t *Transaction) ToBuilder() *TransactionBuilder

// Deprecated: Use TransactionBuilder.WithPayer()
func (t *Transaction) SetPayerInfo(name, iban string)

// Deprecated: Use TransactionBuilder.WithPayee()
func (t *Transaction) SetPayeeInfo(name, iban string)

// Deprecated: Use TransactionBuilder.WithAmountFromFloat()
func (t *Transaction) SetAmountFromFloat(amount float64, currency string)
```

#### Internal Deprecated Methods
```go
// Deprecated: Replaced by AIStrategy
func (c *Categorizer) categorizeWithGemini(transaction Transaction) (models.Category, error)
```

#### Legacy Command Functions
```go
// Deprecated: Use ProcessFile with parser.FullParser
func ProcessFileLegacy(parser models.Parser, inputFile, outputFile string, validate bool, log logging.Logger)

// Deprecated: Use container-based categorizer
func SaveMappings(log *logrus.Logger)
```

#### Configuration Functions
```go
// Deprecated: Use GetContainer().GetConfig()
func GetConfig() *config.Config
```

## Migration Guide

### From Global Singletons to Dependency Injection

**Before (Removed in v2.0.0):**
```go
// This no longer works
categorizer := categorizer.GetDefaultCategorizer()
config := config.GetGlobalConfig()
```

**After (Current approach):**
```go
// Load configuration
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Create container with dependencies
container, err := container.NewContainer(cfg)
if err != nil {
    log.Fatal(err)
}

// Use dependencies from container
categorizer := container.GetCategorizer()
logger := container.GetLogger()
```

### From Float64 to Decimal Operations

**Before (Deprecated, will be removed in v3.0.0):**
```go
amount := transaction.GetAmountAsFloat()
if amount > 100.0 {
    // Process large transaction
}
```

**After (Recommended):**
```go
amount := transaction.GetAmountAsDecimal()
threshold := decimal.NewFromFloat(100.0)
if amount.GreaterThan(threshold) {
    // Process large transaction
}
```

### From Direct Transaction Construction to Builder Pattern

**Before (Deprecated, will be removed in v3.0.0):**
```go
tx := models.Transaction{
    Date:        parseDate(dateStr),
    Amount:      parseAmount(amountStr),
    Description: description,
}
tx.SetPayerInfo("John Doe", "CH1234567890")
```

**After (Recommended):**
```go
tx, err := models.NewTransactionBuilder().
    WithDate(dateStr).
    WithAmountFromString(amountStr, "CHF").
    WithDescription(description).
    WithPayer("John Doe", "CH1234567890").
    AsDebit().
    Build()
```

## Removal Timeline

### v2.1.0 (Planned - Q2 2025)
- Add runtime deprecation warnings for remaining deprecated methods
- Enhanced migration documentation
- Automated migration tools

### v2.2.0 (Planned - Q3 2025)
- Final deprecation warnings
- Migration guide updates
- Performance improvements for new APIs

### v3.0.0 (Planned - Q4 2025)
- **BREAKING**: Remove all deprecated methods and functions
- **BREAKING**: Remove backward compatibility layers
- Clean up internal APIs
- Performance optimizations

## Automated Migration

### Detection Script

Use this script to detect deprecated API usage in your code:

```bash
#!/bin/bash
# detect_deprecated.sh

echo "Scanning for deprecated API usage..."

# Check for deprecated transaction methods
grep -r "GetAmountAsFloat\|GetDebitAsFloat\|GetCreditAsFloat\|GetFeesAsFloat" . --include="*.go" && echo "Found deprecated float methods"

# Check for deprecated transaction construction
grep -r "SetPayerInfo\|SetPayeeInfo\|SetAmountFromFloat" . --include="*.go" && echo "Found deprecated setter methods"

# Check for deprecated global functions (should not exist in v2.0+)
grep -r "GetDefaultCategorizer\|GetGlobalConfig" . --include="*.go" && echo "Found removed global functions - update required"

echo "Scan complete. See migration guide for replacement patterns."
```

### Migration Checklist

- [ ] Replace all `GetAmountAsFloat()` calls with `GetAmountAsDecimal()`
- [ ] Replace all `SetPayerInfo()` calls with `TransactionBuilder.WithPayer()`
- [ ] Replace all `SetPayeeInfo()` calls with `TransactionBuilder.WithPayee()`
- [ ] Replace all `SetAmountFromFloat()` calls with `TransactionBuilder.WithAmountFromFloat()`
- [ ] Update transaction construction to use `TransactionBuilder` pattern
- [ ] Replace global singleton usage with dependency injection container
- [ ] Update test code to use mock dependencies instead of global state

## Support

For migration assistance:
1. Review the [Migration Guide](MIGRATION_GUIDE_V2.md)
2. Check the [Developer Guide](developer-guide.md) for new patterns
3. Create an issue on GitHub with your specific migration questions

## Version Compatibility Matrix

| Version | Global Singletons | Deprecated Methods | Builder Pattern | Container DI |
|---------|-------------------|-------------------|-----------------|--------------|
| v1.x    | ✅ Available      | ✅ Available      | ❌ Not Available | ❌ Not Available |
| v2.0    | ❌ Removed        | ⚠️ Deprecated     | ✅ Available    | ✅ Available |
| v2.1    | ❌ Removed        | ⚠️ With Warnings  | ✅ Available    | ✅ Available |
| v3.0    | ❌ Removed        | ❌ Removed        | ✅ Available    | ✅ Available |

**Legend:**
- ✅ Available and supported
- ⚠️ Available but deprecated (warnings)
- ❌ Removed/Not available