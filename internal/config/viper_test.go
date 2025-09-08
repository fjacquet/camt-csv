package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeConfig_Defaults(t *testing.T) {
	// Clear any existing environment variables
	clearTestEnvVars(t)

	config, err := InitializeConfig()
	require.NoError(t, err)

	// Test default values
	assert.Equal(t, "info", config.Log.Level)
	assert.Equal(t, "text", config.Log.Format)
	assert.Equal(t, ",", config.CSV.Delimiter)
	assert.Equal(t, "DD.MM.YYYY", config.CSV.DateFormat)
	assert.True(t, config.CSV.IncludeHeaders)
	assert.False(t, config.CSV.QuoteAll)
	assert.False(t, config.AI.Enabled)
	assert.Equal(t, "gemini-2.0-flash", config.AI.Model)
	assert.Equal(t, 10, config.AI.RequestsPerMinute)
	assert.Equal(t, 30, config.AI.TimeoutSeconds)
	assert.Equal(t, "Uncategorized", config.AI.FallbackCategory)
	assert.Equal(t, "", config.Data.Directory)
	assert.True(t, config.Data.BackupEnabled)
	assert.True(t, config.Categorization.AutoLearn)
	assert.Equal(t, 0.8, config.Categorization.ConfidenceThreshold)
	assert.False(t, config.Categorization.CaseSensitive)
	assert.True(t, config.Parsers.CAMT.StrictValidation)
	assert.False(t, config.Parsers.PDF.OCREnabled)
	assert.True(t, config.Parsers.Revolut.DateFormatDetection)
}

func TestInitializeConfig_EnvironmentVariables(t *testing.T) {
	// Clear any existing environment variables
	clearTestEnvVars(t)

	// Set test environment variables
	testEnvVars := map[string]string{
		"CAMT_LOG_LEVEL":                    "debug",
		"CAMT_LOG_FORMAT":                   "json",
		"CAMT_CSV_DELIMITER":                ";",
		"CAMT_AI_ENABLED":                   "true",
		"CAMT_AI_MODEL":                     "gemini-1.5-pro",
		"CAMT_AI_REQUESTS_PER_MINUTE":       "15",
		"CAMT_CATEGORIZATION_AUTO_LEARN":    "false",
		"CAMT_PARSERS_CAMT_STRICT_VALIDATION": "false",
		"GEMINI_API_KEY":                    "test-api-key",
	}

	for key, value := range testEnvVars {
		t.Setenv(key, value)
	}

	config, err := InitializeConfig()
	require.NoError(t, err)

	// Test environment variable overrides
	assert.Equal(t, "debug", config.Log.Level)
	assert.Equal(t, "json", config.Log.Format)
	assert.Equal(t, ";", config.CSV.Delimiter)
	assert.True(t, config.AI.Enabled)
	assert.Equal(t, "gemini-1.5-pro", config.AI.Model)
	assert.Equal(t, 15, config.AI.RequestsPerMinute)
	assert.False(t, config.Categorization.AutoLearn)
	assert.False(t, config.Parsers.CAMT.StrictValidation)
	assert.Equal(t, "test-api-key", config.AI.APIKey)
}

func TestInitializeConfig_ConfigFile(t *testing.T) {
	// Clear any existing environment variables
	clearTestEnvVars(t)

	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "camt-csv.yaml")
	
	configContent := `
log:
  level: "warn"
  format: "json"
csv:
  delimiter: "|"
  date_format: "YYYY-MM-DD"
ai:
  enabled: false
  model: "gemini-1.0-pro"
  requests_per_minute: 20
categorization:
  auto_learn: false
  confidence_threshold: 0.9
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp directory so config file is found
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	config, err := InitializeConfig()
	require.NoError(t, err)

	// Test config file values
	assert.Equal(t, "warn", config.Log.Level)
	assert.Equal(t, "json", config.Log.Format)
	assert.Equal(t, "|", config.CSV.Delimiter)
	assert.Equal(t, "YYYY-MM-DD", config.CSV.DateFormat)
	assert.False(t, config.AI.Enabled)
	assert.Equal(t, "gemini-1.0-pro", config.AI.Model)
	assert.Equal(t, 20, config.AI.RequestsPerMinute)
	assert.False(t, config.Categorization.AutoLearn)
	assert.Equal(t, 0.9, config.Categorization.ConfidenceThreshold)
}

func TestInitializeConfig_HierarchicalPrecedence(t *testing.T) {
	// Clear any existing environment variables
	clearTestEnvVars(t)

	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "camt-csv.yaml")
	
	configContent := `
log:
  level: "warn"
csv:
  delimiter: "|"
ai:
  requests_per_minute: 20
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variables that should override config file
	t.Setenv("CAMT_LOG_LEVEL", "error")
	t.Setenv("CAMT_AI_REQUESTS_PER_MINUTE", "25")
	t.Setenv("GEMINI_API_KEY", "env-api-key")

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()
	
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	config, err := InitializeConfig()
	require.NoError(t, err)

	// Test precedence: env vars should override config file
	assert.Equal(t, "error", config.Log.Level)        // env var wins
	assert.Equal(t, "|", config.CSV.Delimiter)        // config file value
	assert.Equal(t, 25, config.AI.RequestsPerMinute)  // env var wins
	assert.Equal(t, "env-api-key", config.AI.APIKey)  // env var (API key)
}

func TestValidateConfig_InvalidValues(t *testing.T) {
	tests := []struct {
		name        string
		modifyConfig func(*Config)
		expectError string
	}{
		{
			name: "invalid log level",
			modifyConfig: func(c *Config) {
				c.Log.Level = "invalid"
			},
			expectError: "invalid log level",
		},
		{
			name: "invalid log format",
			modifyConfig: func(c *Config) {
				c.Log.Format = "invalid"
			},
			expectError: "invalid log format",
		},
		{
			name: "invalid CSV delimiter",
			modifyConfig: func(c *Config) {
				c.CSV.Delimiter = "abc"
			},
			expectError: "CSV delimiter must be a single character",
		},
		{
			name: "AI enabled without API key",
			modifyConfig: func(c *Config) {
				c.AI.Enabled = true
				c.AI.APIKey = ""
			},
			expectError: "GEMINI_API_KEY required when AI is enabled",
		},
		{
			name: "invalid requests per minute",
			modifyConfig: func(c *Config) {
				c.AI.Enabled = true
				c.AI.APIKey = "test-key"
				c.AI.RequestsPerMinute = 0
			},
			expectError: "ai.requests_per_minute must be between 1 and 1000",
		},
		{
			name: "invalid timeout seconds",
			modifyConfig: func(c *Config) {
				c.AI.Enabled = true
				c.AI.APIKey = "test-key"
				c.AI.TimeoutSeconds = 0
			},
			expectError: "ai.timeout_seconds must be between 1 and 300",
		},
		{
			name: "invalid confidence threshold",
			modifyConfig: func(c *Config) {
				c.Categorization.ConfidenceThreshold = 1.5
			},
			expectError: "categorization.confidence_threshold must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Log: struct {
					Level  string `mapstructure:"level" yaml:"level"`
					Format string `mapstructure:"format" yaml:"format"`
				}{
					Level:  "info",
					Format: "text",
				},
				CSV: struct {
					Delimiter      string `mapstructure:"delimiter" yaml:"delimiter"`
					DateFormat     string `mapstructure:"date_format" yaml:"date_format"`
					IncludeHeaders bool   `mapstructure:"include_headers" yaml:"include_headers"`
					QuoteAll       bool   `mapstructure:"quote_all" yaml:"quote_all"`
				}{
					Delimiter: ",",
				},
				AI: struct {
					Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
					Model             string `mapstructure:"model" yaml:"model"`
					RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
					TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
					FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
					APIKey            string `mapstructure:"api_key" yaml:"-"`
				}{
					RequestsPerMinute: 10,
					TimeoutSeconds:    30,
				},
				Categorization: struct {
					AutoLearn           bool    `mapstructure:"auto_learn" yaml:"auto_learn"`
					ConfidenceThreshold float64 `mapstructure:"confidence_threshold" yaml:"confidence_threshold"`
					CaseSensitive       bool    `mapstructure:"case_sensitive" yaml:"case_sensitive"`
				}{
					ConfidenceThreshold: 0.8,
				},
			}

			tt.modifyConfig(config)
			err := validateConfig(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestConfigureLoggingFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		testFunc func(*testing.T, *Config)
	}{
		{
			name: "text format info level",
			config: &Config{
				Log: struct {
					Level  string `mapstructure:"level" yaml:"level"`
					Format string `mapstructure:"format" yaml:"format"`
				}{
					Level:  "info",
					Format: "text",
				},
			},
			testFunc: func(t *testing.T, config *Config) {
				logger := ConfigureLoggingFromConfig(config)
				assert.NotNil(t, logger)
				// Additional assertions could be added here to test logger configuration
			},
		},
		{
			name: "json format debug level",
			config: &Config{
				Log: struct {
					Level  string `mapstructure:"level" yaml:"level"`
					Format string `mapstructure:"format" yaml:"format"`
				}{
					Level:  "debug",
					Format: "json",
				},
			},
			testFunc: func(t *testing.T, config *Config) {
				logger := ConfigureLoggingFromConfig(config)
				assert.NotNil(t, logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, tt.config)
		})
	}
}

// Helper function to clear test environment variables
func clearTestEnvVars(t *testing.T) {
	envVars := []string{
		"CAMT_LOG_LEVEL",
		"CAMT_LOG_FORMAT",
		"CAMT_CSV_DELIMITER",
		"CAMT_CSV_DATE_FORMAT",
		"CAMT_AI_ENABLED",
		"CAMT_AI_MODEL",
		"CAMT_AI_REQUESTS_PER_MINUTE",
		"CAMT_AI_TIMEOUT_SECONDS",
		"CAMT_AI_FALLBACK_CATEGORY",
		"CAMT_DATA_DIRECTORY",
		"CAMT_DATA_BACKUP_ENABLED",
		"CAMT_CATEGORIZATION_AUTO_LEARN",
		"CAMT_CATEGORIZATION_CONFIDENCE_THRESHOLD",
		"CAMT_CATEGORIZATION_CASE_SENSITIVE",
		"CAMT_PARSERS_CAMT_STRICT_VALIDATION",
		"CAMT_PARSERS_PDF_OCR_ENABLED",
		"CAMT_PARSERS_REVOLUT_DATE_FORMAT_DETECTION",
		"GEMINI_API_KEY",
	}

	for _, envVar := range envVars {
		if err := os.Unsetenv(envVar); err != nil {
			// Log warning but continue - this is test cleanup
			fmt.Printf("Warning: failed to unset environment variable %s: %v\n", envVar, err)
		}
	}
}
