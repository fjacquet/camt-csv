# ADR-004: Configuration Management Strategy

## Status

Accepted

## Context

The CAMT-CSV application requires flexible configuration to support:

1. **Multiple Environments**: Development, testing, production
2. **User Preferences**: Different users have different needs
3. **Security**: Sensitive data like API keys must be handled securely
4. **Deployment**: Configuration should work in various deployment scenarios
5. **Defaults**: Sensible defaults for new users
6. **Override Hierarchy**: Clear precedence for configuration sources

## Decision

We will implement a hierarchical configuration system using **Viper** with the following precedence order (highest to lowest):

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file** (`~/.camt-csv/config.yaml` or `./camt-csv.yaml`)
4. **Default values** (lowest priority)

### Configuration Structure

```yaml
# ~/.camt-csv/config.yaml
log:
  level: "info"           # trace, debug, info, warn, error
  format: "text"          # text, json

csv:
  delimiter: ","          # CSV field separator
  date_format: "DD.MM.YYYY"

ai:
  enabled: false          # Enable AI categorization
  model: "gemini-2.0-flash"
  requests_per_minute: 10
  fallback_category: "Uncategorized"

data:
  directory: ""           # Custom data directory (empty = default)
  
categorization:
  auto_learn: true        # Save AI results to local mappings
  confidence_threshold: 0.8
```

## Consequences

### Positive

- **Flexibility**: Users can configure the application for their needs
- **Security**: Sensitive data via environment variables, not files
- **Portability**: Works across different deployment environments
- **Precedence**: Clear override hierarchy prevents confusion
- **Defaults**: Works out-of-the-box for new users
- **Validation**: Configuration is validated at startup

### Negative

- **Complexity**: Multiple configuration sources to manage
- **Documentation**: Need to document all configuration options
- **Testing**: Must test various configuration combinations
- **Migration**: Changes to configuration structure require migration

### Mitigation Strategies

- Comprehensive documentation of all configuration options
- Configuration validation with helpful error messages
- Backward compatibility for configuration changes
- Example configuration files in the repository

## Implementation Details

### Environment Variables

All configuration can be overridden with environment variables using the `CAMT_` prefix:

```bash
export CAMT_LOG_LEVEL=debug
export CAMT_CSV_DELIMITER=";"
export CAMT_AI_ENABLED=true
export GEMINI_API_KEY="your-api-key-here"
```

### Command-line Flags

Critical options available as CLI flags:

```bash
camt-csv convert --input file.xml --output file.csv --log-level debug --dry-run
```

### Configuration Loading

```go
func initConfig() {
    // Set config file locations
    viper.SetConfigName("camt-csv")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("$HOME/.camt-csv")
    viper.AddConfigPath(".")
    
    // Environment variables
    viper.SetEnvPrefix("CAMT")
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    // Set defaults
    setDefaults()
    
    // Read config file (optional)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            log.Fatalf("Error reading config file: %v", err)
        }
        log.Debug("No config file found, using defaults and environment variables")
    }
    
    // Validate configuration
    if err := validateConfig(); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }
}

func setDefaults() {
    viper.SetDefault("log.level", "info")
    viper.SetDefault("log.format", "text")
    viper.SetDefault("csv.delimiter", ",")
    viper.SetDefault("csv.date_format", "DD.MM.YYYY")
    viper.SetDefault("ai.enabled", false)
    viper.SetDefault("ai.model", "gemini-2.0-flash")
    viper.SetDefault("ai.requests_per_minute", 10)
    viper.SetDefault("ai.fallback_category", "Uncategorized")
    viper.SetDefault("categorization.auto_learn", true)
    viper.SetDefault("categorization.confidence_threshold", 0.8)
}
```

### Configuration Validation

```go
func validateConfig() error {
    // Validate log level
    if _, err := logrus.ParseLevel(viper.GetString("log.level")); err != nil {
        return fmt.Errorf("invalid log level: %s", viper.GetString("log.level"))
    }
    
    // Validate CSV delimiter
    delimiter := viper.GetString("csv.delimiter")
    if len(delimiter) != 1 {
        return fmt.Errorf("CSV delimiter must be a single character, got: %s", delimiter)
    }
    
    // Validate AI configuration
    if viper.GetBool("ai.enabled") {
        if viper.GetString("GEMINI_API_KEY") == "" {
            return fmt.Errorf("GEMINI_API_KEY required when AI is enabled")
        }
        
        rpm := viper.GetInt("ai.requests_per_minute")
        if rpm < 1 || rpm > 1000 {
            return fmt.Errorf("ai.requests_per_minute must be between 1 and 1000, got: %d", rpm)
        }
    }
    
    return nil
}
```

## Security Considerations

### Sensitive Data Handling

- **API Keys**: Must be provided via environment variables only
- **File Permissions**: Config files should be readable only by owner (0600)
- **Logging**: Never log sensitive configuration values

### Example Secure Setup

```bash
# Set API key securely
export GEMINI_API_KEY="$(cat ~/.secrets/gemini-api-key)"

# Secure config file permissions
chmod 600 ~/.camt-csv/config.yaml
```

## Configuration Schema

### Complete Configuration Reference

```yaml
# Logging configuration
log:
  level: "info"           # trace|debug|info|warn|error
  format: "text"          # text|json

# CSV output configuration  
csv:
  delimiter: ","          # Single character
  date_format: "DD.MM.YYYY"
  include_headers: true
  quote_all: false

# AI categorization
ai:
  enabled: false          # Enable AI categorization
  model: "gemini-2.0-flash"
  requests_per_minute: 10
  timeout_seconds: 30
  fallback_category: "Uncategorized"
  
# Data storage
data:
  directory: ""           # Custom data directory (empty = default)
  backup_enabled: true
  
# Categorization behavior
categorization:
  auto_learn: true        # Save AI results to mappings
  confidence_threshold: 0.8
  case_sensitive: false
  
# Parser-specific settings
parsers:
  camt:
    strict_validation: true
  pdf:
    ocr_enabled: false
  revolut:
    date_format_detection: true
```

## Migration Strategy

When configuration structure changes:

1. **Backward Compatibility**: Support old configuration keys
2. **Deprecation Warnings**: Log warnings for deprecated options
3. **Migration Tool**: Provide utility to migrate old configurations
4. **Documentation**: Update all documentation and examples

## Related Decisions

- ADR-001: Parser interface standardization
- ADR-002: Hybrid categorization approach
- ADR-003: Functional programming adoption

## Date

2024-12-19

## Authors

- Development Team
