# Configuration Migration Guide

This guide helps you migrate from the legacy environment variable configuration system to the new hierarchical Viper-based configuration management introduced in ADR-004.

## Overview

The CAMT-CSV project has migrated from simple environment variables to a comprehensive hierarchical configuration system using Viper. This provides:

- **Hierarchical Configuration**: Defaults → Config Files → Environment Variables → CLI Flags
- **Multiple Config Formats**: YAML, JSON, TOML support
- **Better Validation**: Comprehensive configuration validation
- **Improved Flexibility**: Easier to manage complex configurations
- **Backward Compatibility**: Existing environment variables continue to work

## Migration Steps

### 1. Current Environment Variables (Still Supported)

All existing environment variables continue to work for backward compatibility:

```bash
# Legacy environment variables (still supported)
export LOG_LEVEL=debug
export LOG_FORMAT=json
export CSV_DELIMITER="|"
export GEMINI_API_KEY=your_api_key_here
export USE_AI_CATEGORIZATION=true
export GEMINI_MODEL=gemini-2.0-flash
export GEMINI_REQUESTS_PER_MINUTE=20
```

### 2. New Configuration File (Recommended)

Create a configuration file at `~/.camt-csv/config.yaml` or `./camt-csv.yaml`:

```yaml
# ~/.camt-csv/config.yaml
log:
  level: "debug"
  format: "json"

csv:
  delimiter: "|"
  date_format: "DD.MM.YYYY"
  include_headers: true
  quote_all: false

ai:
  enabled: true
  model: "gemini-2.0-flash"
  requests_per_minute: 20
  timeout_seconds: 30
  fallback_category: "Uncategorized"

data:
  directory: ""
  backup_enabled: true

categorization:
  auto_learn: true
  confidence_threshold: 0.8
  case_sensitive: false

parsers:
  camt:
    strict_validation: true
  pdf:
    ocr_enabled: false
  revolut:
    date_format_detection: true
```

### 3. New CLI Flags

Use the new CLI flags for one-time configuration overrides:

```bash
# New CLI flags
camt-csv --log-level debug --log-format json --ai-enabled --csv-delimiter "|"
```

## Configuration Precedence

The configuration system follows this precedence order (highest to lowest):

1. **CLI Flags** (highest priority)
2. **Environment Variables** (with `CAMT_` prefix)
3. **Configuration Files** (`~/.camt-csv/config.yaml` or `./camt-csv.yaml`)
4. **Defaults** (lowest priority)

## Environment Variable Mapping

| Legacy Environment Variable | New Config Path | CLI Flag | Default |
|----------------------------|-----------------|----------|---------|
| `LOG_LEVEL` | `log.level` | `--log-level` | `info` |
| `LOG_FORMAT` | `log.format` | `--log-format` | `text` |
| `CSV_DELIMITER` | `csv.delimiter` | `--csv-delimiter` | `,` |
| `GEMINI_API_KEY` | `ai.api_key` | N/A | `""` |
| `USE_AI_CATEGORIZATION` | `ai.enabled` | `--ai-enabled` | `false` |
| `GEMINI_MODEL` | `ai.model` | N/A | `gemini-2.0-flash` |
| `GEMINI_REQUESTS_PER_MINUTE` | `ai.requests_per_minute` | N/A | `10` |

## New Environment Variables (with CAMT_ prefix)

The new system uses the `CAMT_` prefix for environment variables:

```bash
# New prefixed environment variables
export CAMT_LOG_LEVEL=debug
export CAMT_LOG_FORMAT=json
export CAMT_CSV_DELIMITER="|"
export CAMT_AI_ENABLED=true
export CAMT_AI_MODEL=gemini-2.0-flash
export CAMT_AI_REQUESTS_PER_MINUTE=20

# Special case: API key remains unprefixed for security
export GEMINI_API_KEY=your_api_key_here
```

## Migration Strategies

### Strategy 1: Gradual Migration (Recommended)

1. **Keep existing environment variables** - they continue to work
2. **Add configuration file** for new settings and better organization
3. **Gradually move settings** from environment variables to config file
4. **Use CLI flags** for temporary overrides

### Strategy 2: Full Migration

1. **Create comprehensive config file** with all your settings
2. **Remove environment variables** (except `GEMINI_API_KEY`)
3. **Test thoroughly** to ensure all settings work correctly

### Strategy 3: Hybrid Approach

1. **Use config file** for static settings (log format, CSV settings, etc.)
2. **Keep environment variables** for deployment-specific settings
3. **Use CLI flags** for debugging and testing

## Configuration Validation

The new system includes comprehensive validation:

- **Log Level**: Must be valid logrus level (debug, info, warn, error)
- **Log Format**: Must be "text" or "json"
- **CSV Delimiter**: Must be a single character
- **AI Settings**: Validates API key presence, rate limits, timeouts
- **Confidence Threshold**: Must be between 0.0 and 1.0

## Testing Your Configuration

Test your configuration with the `--help` flag to see all available options:

```bash
camt-csv --help
```

Test specific configurations:

```bash
# Test with debug logging
camt-csv --log-level debug

# Test with JSON format
camt-csv --log-format json

# Test with custom delimiter
camt-csv --csv-delimiter "|"

# Test AI categorization
camt-csv --ai-enabled
```

## Troubleshooting

### Configuration File Not Found

This is normal and expected. The system will use defaults and environment variables.

### Invalid Configuration Values

The system will report validation errors with specific details:

```
Failed to initialize configuration: invalid configuration: invalid log level: invalid_level
```

### Environment Variable Issues

- Ensure `GEMINI_API_KEY` is set if using AI categorization
- Use `CAMT_` prefix for new environment variables
- Legacy environment variables (without prefix) still work

### CLI Flag Issues

- Use `--help` to see all available flags
- Boolean flags don't require values: `--ai-enabled` (not `--ai-enabled=true`)
- String flags require values: `--log-level debug`

## Examples

### Development Setup

```yaml
# ~/.camt-csv/config.yaml
log:
  level: "debug"
  format: "text"

csv:
  delimiter: ","
  include_headers: true

ai:
  enabled: false  # Disable AI for faster development

categorization:
  auto_learn: true
  confidence_threshold: 0.7
```

### Production Setup

```yaml
# ~/.camt-csv/config.yaml
log:
  level: "info"
  format: "json"

csv:
  delimiter: ","
  include_headers: true
  quote_all: true

ai:
  enabled: true
  model: "gemini-2.0-flash"
  requests_per_minute: 10
  timeout_seconds: 30

data:
  backup_enabled: true

categorization:
  auto_learn: false  # Disable learning in production
  confidence_threshold: 0.8
```

### CI/CD Setup

```bash
# Use environment variables for CI/CD
export CAMT_LOG_LEVEL=warn
export CAMT_LOG_FORMAT=json
export CAMT_AI_ENABLED=false
export GEMINI_API_KEY=${SECRET_GEMINI_API_KEY}

# Run with overrides
camt-csv batch --input ./data --output ./results
```

## Benefits of Migration

1. **Better Organization**: All configuration in one place
2. **Environment-Specific Configs**: Different configs for dev/staging/prod
3. **Validation**: Catch configuration errors early
4. **Documentation**: Self-documenting configuration files
5. **Flexibility**: Easy to add new configuration options
6. **CLI Integration**: Override any setting from command line

## Support

If you encounter issues during migration:

1. Check the validation error messages for specific guidance
2. Use `--log-level debug` to see detailed configuration loading information
3. Verify your configuration file syntax with a YAML validator
4. Test with minimal configuration first, then add complexity

The migration is designed to be seamless with full backward compatibility. Take your time and migrate at your own pace.
