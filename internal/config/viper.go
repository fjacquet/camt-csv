// Package config provides Viper-based hierarchical configuration management
package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Log struct {
		Level  string `mapstructure:"level" yaml:"level"`
		Format string `mapstructure:"format" yaml:"format"`
	} `mapstructure:"log" yaml:"log"`

	CSV struct {
		Delimiter      string `mapstructure:"delimiter" yaml:"delimiter"`
		DateFormat     string `mapstructure:"date_format" yaml:"date_format"`
		IncludeHeaders bool   `mapstructure:"include_headers" yaml:"include_headers"`
		QuoteAll       bool   `mapstructure:"quote_all" yaml:"quote_all"`
	} `mapstructure:"csv" yaml:"csv"`

	AI struct {
		Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
		Model             string `mapstructure:"model" yaml:"model"`
		RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
		TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
		FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
		APIKey            string `mapstructure:"api_key" yaml:"-"` // Never serialize API key
	} `mapstructure:"ai" yaml:"ai"`

	Data struct {
		Directory     string `mapstructure:"directory" yaml:"directory"`
		BackupEnabled bool   `mapstructure:"backup_enabled" yaml:"backup_enabled"`
	} `mapstructure:"data" yaml:"data"`

	Categorization struct {
		AutoLearn           bool    `mapstructure:"auto_learn" yaml:"auto_learn"`
		ConfidenceThreshold float64 `mapstructure:"confidence_threshold" yaml:"confidence_threshold"`
		CaseSensitive       bool    `mapstructure:"case_sensitive" yaml:"case_sensitive"`
	} `mapstructure:"categorization" yaml:"categorization"`

	Parsers struct {
		CAMT struct {
			StrictValidation bool `mapstructure:"strict_validation" yaml:"strict_validation"`
		} `mapstructure:"camt" yaml:"camt"`
		PDF struct {
			OCREnabled bool `mapstructure:"ocr_enabled" yaml:"ocr_enabled"`
		} `mapstructure:"pdf" yaml:"pdf"`
		Revolut struct {
			DateFormatDetection bool `mapstructure:"date_format_detection" yaml:"date_format_detection"`
		} `mapstructure:"revolut" yaml:"revolut"`
	} `mapstructure:"parsers" yaml:"parsers"`

	Constitution struct {
		FilePaths []string `mapstructure:"file_paths" yaml:"file_paths"`
	} `mapstructure:"constitution" yaml:"constitution"`
}

// InitializeConfig initializes Viper configuration with hierarchical loading
func InitializeConfig() (*Config, error) {
	v := viper.New()

	// 1. Set defaults
	setDefaults(v)

	// 2. Config file locations
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("$HOME/.camt-csv")
	v.AddConfigPath(".camt-csv")
	v.AddConfigPath(".")

	// 3. Environment variables
	v.SetEnvPrefix("CAMT")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 4. Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Log the error but don't fail - continue with defaults and env vars
			fmt.Printf("Warning: error reading config file %s: %v\n", v.ConfigFileUsed(), err)
		}
		// Config file not found or invalid is OK, we'll use defaults and env vars
	}

	// 5. Handle special case for API key (always from env, not prefixed)
	if err := v.BindEnv("ai.api_key", "GEMINI_API_KEY"); err != nil {
		fmt.Printf("Warning: failed to bind GEMINI_API_KEY environment variable: %v\n", err)
	}

	// Bind constitution file paths from environment variable
	if err := v.BindEnv("constitution.file_paths", "CAMT_CONSTITUTION_FILE_PATHS"); err != nil {
		fmt.Printf("Warning: failed to bind CAMT_CONSTITUTION_FILE_PATHS environment variable: %v\n", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 6. Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	// CSV defaults
	v.SetDefault("csv.delimiter", ",")
	v.SetDefault("csv.date_format", "DD.MM.YYYY")
	v.SetDefault("csv.include_headers", true)
	v.SetDefault("csv.quote_all", false)

	// AI defaults
	v.SetDefault("ai.enabled", false)
	v.SetDefault("ai.model", "gemini-2.0-flash")
	v.SetDefault("ai.requests_per_minute", 10)
	v.SetDefault("ai.timeout_seconds", 30)
	v.SetDefault("ai.fallback_category", "Uncategorized")

	// Data defaults
	v.SetDefault("data.directory", "")
	v.SetDefault("data.backup_enabled", true)

	// Categorization defaults
	v.SetDefault("categorization.auto_learn", true)
	v.SetDefault("categorization.confidence_threshold", 0.8)
	v.SetDefault("categorization.case_sensitive", false)

	// Parser defaults
	v.SetDefault("parsers.camt.strict_validation", true)
	v.SetDefault("parsers.pdf.ocr_enabled", false)
	v.SetDefault("parsers.revolut.date_format_detection", true)

	// Constitution defaults
	v.SetDefault("constitution.file_paths", []string{})
}

// validateConfig validates the configuration values
func validateConfig(config *Config) error {
	// Validate log level
	if _, err := logrus.ParseLevel(config.Log.Level); err != nil {
		return fmt.Errorf("invalid log level: %s", config.Log.Level)
	}

	// Validate log format
	if config.Log.Format != "text" && config.Log.Format != "json" {
		return fmt.Errorf("invalid log format: %s (must be 'text' or 'json')", config.Log.Format)
	}

	// Validate CSV delimiter
	if len(config.CSV.Delimiter) != 1 {
		return fmt.Errorf("CSV delimiter must be a single character, got: %s", config.CSV.Delimiter)
	}

	// Validate AI configuration
	if config.AI.Enabled {
		if config.AI.APIKey == "" {
			return fmt.Errorf("GEMINI_API_KEY required when AI is enabled")
		}

		if config.AI.RequestsPerMinute < 1 || config.AI.RequestsPerMinute > 1000 {
			return fmt.Errorf("ai.requests_per_minute must be between 1 and 1000, got: %d", config.AI.RequestsPerMinute)
		}

		if config.AI.TimeoutSeconds < 1 || config.AI.TimeoutSeconds > 300 {
			return fmt.Errorf("ai.timeout_seconds must be between 1 and 300, got: %d", config.AI.TimeoutSeconds)
		}
	}

	// Validate confidence threshold
	if config.Categorization.ConfidenceThreshold < 0.0 || config.Categorization.ConfidenceThreshold > 1.0 {
		return fmt.Errorf("categorization.confidence_threshold must be between 0.0 and 1.0, got: %f", config.Categorization.ConfidenceThreshold)
	}

	return nil
}

// ConfigureLoggingFromConfig configures logging based on the Config struct
func ConfigureLoggingFromConfig(config *Config) *logrus.Logger {
	logger := logrus.New()

	// Parse and set log level
	logLevel, err := logrus.ParseLevel(strings.ToLower(config.Log.Level))
	if err != nil {
		logger.Warnf("Invalid log level '%s', using 'info'", config.Log.Level)
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Configure log format
	if strings.ToLower(config.Log.Format) == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return logger
}
