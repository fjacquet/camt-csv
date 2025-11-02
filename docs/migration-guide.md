# Migration Guide

## Overview

This guide helps you migrate from older versions of CAMT-CSV to the latest version, which includes significant architectural improvements, new features, and some breaking changes.

## Version 2.0.0 Migration

### Breaking Changes

#### 1. Debtor File Naming Convention

**Change**: The debtor mapping file has been renamed from `debitors.yaml` to `debtors.yaml` to follow standard English spelling.

**Migration Steps**:

1. **Rename your existing file**:
   ```bash
   mv database/debitors.yaml database/debtors.yaml
   ```

2. **Update configuration references** (if using custom paths):
   ```yaml
   # Old configuration
   categorization:
     debitors_file: "path/to/debitors.yaml"
   
   # New configuration
   categorization:
     debtors_file: "path/to/debtors.yaml"
   ```

**Backward Compatibility**: The application will continue to work with `debitors.yaml` for now, but you'll see deprecation warnings. Update your files to avoid future issues.

#### 2. Global Singleton Removal

**Change**: Global singleton functions have been removed in favor of dependency injection.

**Old Code**:
```go
// This no longer works
categorizer := categorizer.GetDefaultCategorizer()
result := categorizer.CategorizeTransaction(tx)
```

**New Code**:
```go
// Use dependency injection
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}

result, err := container.Categorizer.Categorize(ctx, tx)
if err != nil {
    log.Fatal(err)
}
```

#### 3. Configuration Structure Changes

**Change**: Configuration structure has been updated for better organization.

**Old Configuration** (`~/.camt-csv/config.yaml`):
```yaml
log_level: "info"
csv_delimiter: ","
ai_enabled: true
gemini_api_key: "your-key"
```

**New Configuration**:
```yaml
log:
  level: "info"
  format: "text"
csv:
  delimiter: ","
  include_headers: true
ai:
  enabled: true
  model: "gemini-2.0-flash"
  # API key should be set via environment variable
```

**Migration Steps**:

1. **Update configuration file structure**:
   ```bash
   # Backup old config
   cp ~/.camt-csv/config.yaml ~/.camt-csv/config.yaml.backup
   
   # Update to new structure
   cat > ~/.camt-csv/config.yaml << EOF
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
   ```

2. **Set API key via environment variable**:
   ```bash
   export GEMINI_API_KEY=your_api_key_here
   ```

#### 4. Error Handling Changes

**Change**: Custom error types provide more detailed context.

**Old Error Handling**:
```go
if err != nil {
    log.Printf("Error: %v", err)
}
```

**New Error Handling**:
```go
if err != nil {
    var parseErr *parsererror.ParseError
    if errors.As(err, &parseErr) {
        log.Printf("Parse error in %s field %s with value '%s': %v", 
            parseErr.Parser, parseErr.Field, parseErr.Value, parseErr.Err)
    } else {
        log.Printf("Error: %v", err)
    }
}
```

### New Features

#### 1. Structured Logging

**Feature**: Framework-agnostic logging with structured fields.

**Usage**:
```go
logger := logging.NewLogrusAdapter("info", "json")
logger.Info("Processing transaction",
    logging.Field{Key: "file", Value: filename},
    logging.Field{Key: "count", Value: len(transactions)})
```

#### 2. Transaction Builder Pattern

**Feature**: Fluent API for constructing transactions.

**Usage**:
```go
tx, err := models.NewTransactionBuilder().
    WithDate("2025-01-15").
    WithAmount(decimal.NewFromFloat(100.50), "CHF").
    WithPayer("John Doe", "CH1234567890").
    WithPayee("Acme Corp", "CH0987654321").
    AsDebit().
    Build()
```

#### 3. Revolut Investment Parser

**Feature**: Dedicated parser for Revolut investment transactions.

**Usage**:
```bash
./camt-csv revolut-investment -i investment_export.csv -o processed.csv
```

#### 4. Enhanced PDF Parser

**Feature**: Improved PDF parsing with dependency injection for better testability.

**Benefits**:
- More reliable extraction
- Better error messages
- Easier testing and debugging

### Deprecated Features

#### 1. Global Configuration Functions

**Deprecated**:
```go
config := config.GetGlobalConfig()  // Deprecated
```

**Replacement**:
```go
cfg, err := config.LoadConfig()
if err != nil {
    log.Fatal(err)
}
```

#### 2. Direct Parser Factory Usage

**Deprecated**:
```go
parser, err := factory.GetParser(factory.CAMT)  // Deprecated
```

**Replacement**:
```go
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}

parser, err := container.GetParser(factory.CAMT)
if err != nil {
    log.Fatal(err)
}
```

## Version 1.x to 2.0 Migration Checklist

### Pre-Migration Steps

- [ ] **Backup your data**: Create backups of configuration files and YAML mappings
- [ ] **Test current setup**: Ensure your current version works correctly
- [ ] **Review custom configurations**: Note any custom settings or file paths

### Migration Steps

1. **Update binary**:
   ```bash
   # Download new version
   git pull origin main
   go build -o camt-csv
   ```

2. **Rename debtor file**:
   ```bash
   mv database/debitors.yaml database/debtors.yaml
   ```

3. **Update configuration**:
   ```bash
   # Update config file structure (see examples above)
   nano ~/.camt-csv/config.yaml
   ```

4. **Set environment variables**:
   ```bash
   export GEMINI_API_KEY=your_api_key_here
   ```

5. **Test basic functionality**:
   ```bash
   ./camt-csv camt -i samples/camt053/sample.xml -o test_output.csv
   ```

6. **Verify categorization**:
   ```bash
   # Check that categories are working correctly
   head -10 test_output.csv
   ```

### Post-Migration Verification

- [ ] **Test all parsers**: Verify each parser type works correctly
- [ ] **Check categorization**: Ensure transactions are categorized properly
- [ ] **Verify batch processing**: Test batch operations if you use them
- [ ] **Review logs**: Check for any deprecation warnings

## Common Migration Issues

### Issue 1: Configuration Not Found

**Error**: `config file not found`

**Solution**:
```bash
# Ensure config directory exists
mkdir -p ~/.camt-csv

# Create basic config
cat > ~/.camt-csv/config.yaml << EOF
log:
  level: "info"
  format: "text"
csv:
  delimiter: ","
ai:
  enabled: false
EOF
```

### Issue 2: Debtor File Not Found

**Error**: `failed to load debtors: file not found`

**Solution**:
```bash
# Check if old file exists and rename it
if [ -f database/debitors.yaml ]; then
    mv database/debitors.yaml database/debtors.yaml
    echo "Renamed debitors.yaml to debtors.yaml"
fi
```

### Issue 3: API Key Not Working

**Error**: `API key not configured`

**Solution**:
```bash
# Set environment variable
export GEMINI_API_KEY=your_actual_api_key

# Or add to your shell profile
echo 'export GEMINI_API_KEY=your_actual_api_key' >> ~/.bashrc
source ~/.bashrc
```

### Issue 4: Permission Errors

**Error**: `permission denied`

**Solution**:
```bash
# Fix file permissions
chmod 600 ~/.camt-csv/config.yaml
chmod 644 database/*.yaml
```

## Rollback Procedure

If you encounter issues and need to rollback:

1. **Restore old binary**:
   ```bash
   git checkout v1.x.x  # Replace with your previous version
   go build -o camt-csv
   ```

2. **Restore configuration**:
   ```bash
   cp ~/.camt-csv/config.yaml.backup ~/.camt-csv/config.yaml
   ```

3. **Restore debtor file**:
   ```bash
   mv database/debtors.yaml database/debitors.yaml
   ```

## Getting Help

### Troubleshooting Steps

1. **Enable debug logging**:
   ```bash
   ./camt-csv --log-level debug camt -i input.xml -o output.csv
   ```

2. **Check configuration**:
   ```bash
   ./camt-csv --help
   ```

3. **Verify file permissions**:
   ```bash
   ls -la ~/.camt-csv/
   ls -la database/
   ```

### Support Resources

- **Documentation**: Check the [User Guide](user-guide.md) for detailed usage instructions
- **Architecture**: Review the [Architecture Documentation](architecture.md) for technical details
- **Development**: See the [Developer Guide](developer-guide.md) for contributing
- **Issues**: Report bugs on the GitHub issue tracker

### Migration Support

If you encounter issues during migration:

1. **Create an issue** with:
   - Your current version
   - Migration steps attempted
   - Error messages
   - Configuration files (redacted)

2. **Include debug output**:
   ```bash
   ./camt-csv --log-level debug [command] 2>&1 | tee migration-debug.log
   ```

3. **Provide sample data** (if possible):
   - Anonymized input files
   - Expected vs actual output

## Future Migrations

### Staying Updated

1. **Follow semantic versioning**: 
   - Patch versions (x.x.1): Bug fixes, safe to update
   - Minor versions (x.1.x): New features, backward compatible
   - Major versions (2.x.x): Breaking changes, follow migration guide

2. **Read release notes**: Always review CHANGELOG.md before updating

3. **Test in development**: Test new versions with your data before production use

### Best Practices

- **Regular backups**: Keep backups of configuration and mapping files
- **Version pinning**: Pin to specific versions in production environments
- **Gradual rollout**: Test new versions with subset of data first
- **Monitor deprecations**: Address deprecation warnings promptly

This migration guide ensures a smooth transition to the latest version while maintaining your existing workflows and data.