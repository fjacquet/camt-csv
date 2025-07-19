# Implementation Plan: Full Configuration Management

## Overview

This document outlines the detailed implementation plan for upgrading the CAMT-CSV project's configuration management from the current simple environment variable approach to a full Viper-based hierarchical configuration system as specified in ADR-004.

## Current State Analysis

### What We Have

- Simple environment variable loading with `godotenv`
- Basic logging configuration in `internal/config/config.go`
- Direct `os.Getenv()` calls scattered throughout the codebase
- `.env` file support for development

### What We Need

- Viper-based hierarchical configuration (CLI flags → env vars → config file → defaults)
- YAML configuration file support (`~/.camt-csv/config.yaml`)
- Structured configuration objects with type safety
- Centralized configuration management
- Backward compatibility with existing `.env` approach

## Implementation Phases

### Phase 1: Foundation Setup (Days 1-2)

#### 1.1 Dependencies and Structure

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Add Viper dependency to `go.mod`
- [ ] Create new configuration structure in `internal/config/`
- [ ] Define Go structs matching ADR-004 YAML schema
- [ ] Create configuration loading hierarchy

**Files to Create/Modify**:

```
internal/config/
├── config.go          # Main config struct and loading logic
├── viper.go           # Viper initialization and hierarchy setup
├── defaults.go        # Default configuration values
└── validation.go      # Configuration validation
```

**Deliverables**:

- Configuration structs defined
- Viper initialization framework
- Default values established

#### 1.2 Core Configuration Structure

**Estimated Time**: 4 hours

**Configuration Schema** (based on ADR-004):

```go
type Config struct {
    Log struct {
        Level  string `mapstructure:"level" yaml:"level"`
        Format string `mapstructure:"format" yaml:"format"`
    } `mapstructure:"log" yaml:"log"`
    
    CSV struct {
        Delimiter  string `mapstructure:"delimiter" yaml:"delimiter"`
        DateFormat string `mapstructure:"date_format" yaml:"date_format"`
    } `mapstructure:"csv" yaml:"csv"`
    
    AI struct {
        Enabled            bool   `mapstructure:"enabled" yaml:"enabled"`
        Model              string `mapstructure:"model" yaml:"model"`
        RequestsPerMinute  int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
        FallbackCategory   string `mapstructure:"fallback_category" yaml:"fallback_category"`
    } `mapstructure:"ai" yaml:"ai"`
    
    Data struct {
        Directory string `mapstructure:"directory" yaml:"directory"`
    } `mapstructure:"data" yaml:"data"`
    
    Categorization struct {
        AutoLearn            bool    `mapstructure:"auto_learn" yaml:"auto_learn"`
        ConfidenceThreshold  float64 `mapstructure:"confidence_threshold" yaml:"confidence_threshold"`
    } `mapstructure:"categorization" yaml:"categorization"`
}
```

### Phase 2: Configuration Loading Implementation (Days 2-3)

#### 2.1 Viper Integration

**Estimated Time**: 6 hours

**Tasks**:

- [ ] Implement hierarchical configuration loading
- [ ] Set up config file discovery (`~/.camt-csv/config.yaml`, `./camt-csv.yaml`)
- [ ] Environment variable mapping with prefixes
- [ ] Default value initialization

**Key Implementation Points**:

```go
// Hierarchy: CLI flags → env vars → config file → defaults
func InitializeConfig() (*Config, error) {
    v := viper.New()
    
    // 1. Set defaults
    setDefaults(v)
    
    // 2. Config file locations
    v.SetConfigName("camt-csv")
    v.SetConfigType("yaml")
    v.AddConfigPath("$HOME/.camt-csv")
    v.AddConfigPath(".")
    
    // 3. Environment variables
    v.SetEnvPrefix("CAMT_CSV")
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    // 4. Read config file
    if err := v.ReadInConfig(); err != nil {
        // Handle missing config file gracefully
    }
    
    // 5. Bind CLI flags (done in CLI setup)
    
    var config Config
    return &config, v.Unmarshal(&config)
}
```

#### 2.2 Backward Compatibility

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Maintain `.env` file support alongside Viper
- [ ] Create migration utilities
- [ ] Environment variable mapping for existing variables

**Compatibility Strategy**:

- Keep existing `.env` loading as fallback
- Map existing env vars to new config structure
- Provide migration warnings/guidance

### Phase 3: CLI Integration (Days 3-4)

#### 3.1 Root Command Updates

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Update `cmd/root/root.go` to use Viper configuration
- [ ] Add global flags for common configuration options
- [ ] Integrate config loading into command initialization

**Global Flags to Add**:

```go
// Global flags that affect all commands
rootCmd.PersistentFlags().String("config", "", "config file path")
rootCmd.PersistentFlags().String("log-level", "", "log level (trace, debug, info, warn, error)")
rootCmd.PersistentFlags().String("log-format", "", "log format (text, json)")
rootCmd.PersistentFlags().String("data-dir", "", "data directory path")
```

#### 3.2 Subcommand Updates

**Estimated Time**: 6 hours

**Tasks**:

- [ ] Update each subcommand to support relevant flags
- [ ] Add command-specific configuration options
- [ ] Ensure flag binding to Viper configuration

**Commands to Update**:

- `cmd/camt/` - Add CAMT-specific flags
- `cmd/pdf/` - Add PDF parsing flags
- `cmd/revolut/` - Add Revolut-specific flags
- `cmd/selma/` - Add Selma-specific flags
- `cmd/categorize/` - Add categorization flags
- `cmd/batch/` - Add batch processing flags

### Phase 4: Parser Integration (Days 4-5)

#### 4.1 Parser Configuration Updates

**Estimated Time**: 6 hours

**Tasks**:

- [ ] Update all parsers to use centralized configuration
- [ ] Replace direct `os.Getenv()` calls with config access
- [ ] Add configuration injection to parser constructors

**Parser Updates Required**:

```go
// Before: Direct environment variable access
apiKey := os.Getenv("GEMINI_API_KEY")

// After: Configuration-based access
apiKey := config.AI.APIKey
```

**Files to Update**:

- `internal/camtparser/`
- `internal/pdfparser/`
- `internal/revolutparser/`
- `internal/selmaparser/`
- `internal/categorizer/`

#### 4.2 Configuration Injection

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Add configuration parameters to parser constructors
- [ ] Update parser interfaces to accept configuration
- [ ] Modify main application to pass configuration to parsers

### Phase 5: Testing and Validation (Days 5-6)

#### 5.1 Configuration Testing

**Estimated Time**: 6 hours

**Tasks**:

- [ ] Create comprehensive configuration tests
- [ ] Test all configuration sources (flags, env vars, files, defaults)
- [ ] Test configuration precedence hierarchy
- [ ] Test backward compatibility scenarios

**Test Scenarios**:

```go
func TestConfigurationHierarchy(t *testing.T) {
    tests := []struct {
        name           string
        configFile     string
        envVars        map[string]string
        cliFlags       []string
        expectedValue  interface{}
    }{
        {
            name: "CLI flag overrides all",
            configFile: "log:\n  level: info",
            envVars: map[string]string{"CAMT_CSV_LOG_LEVEL": "debug"},
            cliFlags: []string{"--log-level", "error"},
            expectedValue: "error",
        },
        // More test cases...
    }
}
```

#### 5.2 Integration Testing

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Test all CLI commands with new configuration system
- [ ] Test parser functionality with configuration injection
- [ ] Test migration scenarios from old to new system

### Phase 6: Documentation and Migration (Days 6-7)

#### 6.1 Documentation Updates

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Update user guide with new configuration options
- [ ] Create configuration reference documentation
- [ ] Update CLI help text and examples
- [ ] Create migration guide for existing users

**Documentation Files to Update**:

- `docs/user-guide.md`
- `docs/configuration-reference.md` (new)
- `docs/migration-guide.md` (new)
- `README.md`

#### 6.2 Migration Tools

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Create `.env` to YAML conversion utility
- [ ] Add configuration validation command
- [ ] Create configuration template generator

### Phase 7: Final Integration and Testing (Days 7-8)

#### 7.1 End-to-End Testing

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Full application testing with new configuration system
- [ ] Performance testing to ensure no regression
- [ ] User acceptance testing scenarios

#### 7.2 Deployment Preparation

**Estimated Time**: 4 hours

**Tasks**:

- [ ] Update build and deployment scripts
- [ ] Create configuration examples for different environments
- [ ] Prepare release notes and changelog

## Risk Mitigation Strategies

### Breaking Changes

**Risk**: Existing users may face configuration issues
**Mitigation**:

- Maintain backward compatibility with `.env` files
- Provide clear migration documentation
- Add deprecation warnings for old configuration methods

### Implementation Complexity

**Risk**: Viper integration may introduce bugs
**Mitigation**:

- Comprehensive testing at each phase
- Incremental rollout with feature flags
- Extensive validation and error handling

### Performance Impact

**Risk**: Configuration loading may slow application startup
**Mitigation**:

- Benchmark configuration loading performance
- Implement configuration caching where appropriate
- Optimize file discovery and parsing

## Success Criteria

### Functional Requirements

- [ ] All configuration options from ADR-004 are supported
- [ ] Hierarchical configuration precedence works correctly
- [ ] Backward compatibility with existing `.env` approach
- [ ] All CLI commands support relevant configuration flags

### Non-Functional Requirements

- [ ] Configuration loading adds <100ms to application startup
- [ ] All existing functionality continues to work
- [ ] Test coverage maintains >80% for configuration code
- [ ] Documentation is complete and accurate

### User Experience

- [ ] Clear migration path for existing users
- [ ] Intuitive CLI flag naming and organization
- [ ] Helpful error messages for configuration issues
- [ ] Comprehensive examples and documentation

## Rollback Plan

If issues arise during implementation:

1. **Phase-by-phase rollback**: Each phase is self-contained and can be reverted
2. **Feature flags**: Use build tags or runtime flags to enable/disable new configuration system
3. **Parallel systems**: Run old and new configuration systems in parallel during transition
4. **Quick fixes**: Maintain ability to quickly patch critical configuration issues

## Timeline Summary

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| 1 | 2 days | Foundation setup, config structures |
| 2 | 1 day | Viper integration, loading hierarchy |
| 3 | 1 day | CLI integration, global flags |
| 4 | 1 day | Parser updates, config injection |
| 5 | 1 day | Testing and validation |
| 6 | 1 day | Documentation and migration tools |
| 7 | 1 day | Final integration and testing |

**Total Estimated Duration**: 8 days

## Next Steps

1. **Get approval** for this implementation plan
2. **Set up development branch** for configuration management work
3. **Begin Phase 1** with foundation setup
4. **Regular check-ins** after each phase completion
5. **User testing** before final deployment

This plan provides a structured, low-risk approach to implementing the full configuration management system while maintaining backward compatibility and ensuring a smooth transition for existing users.
