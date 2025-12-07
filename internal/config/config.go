// Package config provides functionality for loading and accessing environment variables.
//
// DEPRECATION NOTICE: The functions in this file (config.go) are deprecated.
// Use the Viper-based configuration from viper.go instead:
//   - LoadEnv() -> Use InitializeConfig() from viper.go
//   - GetEnv() -> Use Config struct fields
//   - MustGetEnv() -> Use Config struct fields with validation
//   - GetGeminiAPIKey() -> Use Config.AI.APIKey
//   - ConfigureLogging() -> Use ConfigureLoggingFromConfig()
//   - Logger (global) -> Use container.GetLogger()
//
// These functions will be removed in v3.0.0.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var (
	once sync.Once

	// Logger is a global logger instance.
	//
	// Deprecated: Use container.GetLogger() instead for dependency injection.
	// Global mutable state is an anti-pattern. This will be removed in v3.0.0.
	Logger = logrus.New()

	// globalConfig is the global config instance.
	// Deprecated: Use InitializeConfig() with dependency injection instead.
	globalConfig *Config
)

// ConfigureLogging sets up logging based on environment variables and returns the configured logger.
//
// Deprecated: Use ConfigureLoggingFromConfig() with a Config struct instead.
// This function will be removed in v3.0.0.
//
// Migration:
//
//	// Old:
//	logger := config.ConfigureLogging()
//
//	// New:
//	cfg, _ := config.InitializeConfig()
//	logger := config.ConfigureLoggingFromConfig(cfg)
func ConfigureLogging() *logrus.Logger {
	// Configure log level
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "info" // Default log level
	}

	// Parse the log level
	logLevel, err := logrus.ParseLevel(strings.ToLower(logLevelStr))
	if err != nil {
		Logger.Warnf("Invalid log level '%s', using 'info'", logLevelStr)
		logLevel = logrus.InfoLevel
	}
	Logger.SetLevel(logLevel)

	// Configure log format
	logFormat := os.Getenv("LOG_FORMAT")
	if strings.ToLower(logFormat) == "json" {
		Logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		// Default to text formatter
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return Logger
}

// LoadEnv loads environment variables from .env file if it exists.
//
// Deprecated: Use InitializeConfig() instead, which handles all configuration loading.
// This function will be removed in v3.0.0.
//
// Migration:
//
//	// Old:
//	config.LoadEnv()
//
//	// New:
//	cfg, err := config.InitializeConfig()
//	// cfg contains all configuration from env vars and config files
func LoadEnv() {
	once.Do(func() {
		// Try to find .env file in current directory
		envFile := ".env"
		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			// Try to find .env in parent directory (project root)
			envFile = filepath.Join("..", ".env")
			if _, err := os.Stat(envFile); os.IsNotExist(err) {
				Logger.Info("No .env file found, using environment variables")
				return
			}
		}

		// Load .env file
		err := godotenv.Load(envFile)
		if err != nil {
			Logger.Warnf("Error loading .env file: %v", err)
			return
		}
		Logger.Infof("Loaded environment variables from %s", envFile)

		// Configure logging after loading environment variables
		ConfigureLogging()
	})
}

// GetEnv retrieves an environment variable with a fallback value if not set.
//
// Deprecated: Use Config struct fields instead. Access configuration via
// InitializeConfig() which provides type-safe access to all settings.
// This function will be removed in v3.0.0.
//
// Migration:
//
//	// Old:
//	value := config.GetEnv("SOME_KEY", "default")
//
//	// New:
//	cfg, _ := config.InitializeConfig()
//	value := cfg.SomeField // Use appropriate Config field
func GetEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}

// MustGetEnv retrieves an environment variable or panics if not set.
//
// Deprecated: Use Config struct with validation instead. InitializeConfig()
// validates required fields and returns errors rather than panicking.
// This function will be removed in v3.0.0.
//
// Migration:
//
//	// Old:
//	apiKey := config.MustGetEnv("API_KEY")
//
//	// New:
//	cfg, err := config.InitializeConfig()
//	if err != nil { log.Fatal(err) }
//	apiKey := cfg.AI.APIKey
func MustGetEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		Logger.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// GetGeminiAPIKey returns the Gemini API key from environment variables.
//
// Deprecated: Use Config.AI.APIKey instead.
// This function will be removed in v3.0.0.
//
// Migration:
//
//	// Old:
//	apiKey := config.GetGeminiAPIKey()
//
//	// New:
//	cfg, _ := config.InitializeConfig()
//	apiKey := cfg.AI.APIKey
func GetGeminiAPIKey() string {
	return GetEnv("GEMINI_API_KEY", "")
}

// InitializeGlobalConfig explicitly initializes the global configuration.
//
// Deprecated: Use InitializeConfig() and dependency injection instead.
// This function maintains global state which is an anti-pattern.
// This function will be removed in v3.0.0.
//
// Migration:
//
//	// Old:
//	err := config.InitializeGlobalConfig()
//
//	// New:
//	cfg, err := config.InitializeConfig()
//	container, err := container.NewContainer(cfg)
func InitializeGlobalConfig() error {
	var err error
	globalConfig, err = InitializeConfig()
	if err != nil {
		return err
	}
	Logger = ConfigureLoggingFromConfig(globalConfig)
	return nil
}
