// Package config provides functionality for loading and accessing environment variables.
// This file maintains backward compatibility while the new Viper-based system is being adopted.
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
	// Global logger instance that should be used across the application
	Logger = logrus.New()
	// Global config instance for new Viper-based configuration
	globalConfig *Config
	configOnce   sync.Once
)

// ConfigureLogging sets up logging based on environment variables and returns the configured logger
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

// LoadEnv loads environment variables from .env file if it exists
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

// GetEnv retrieves an environment variable with a fallback value if not set
func GetEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}

// MustGetEnv retrieves an environment variable or panics if not set
func MustGetEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		Logger.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// GetGeminiAPIKey returns the Gemini API key from environment variables
func GetGeminiAPIKey() string {
	return GetEnv("GEMINI_API_KEY", "")
}

// GetGlobalConfig returns the global configuration instance, initializing it if necessary
// Deprecated: Use dependency injection with container.NewContainer instead.
// This function will be removed in v2.0.0.
//
// Migration example:
//   // Old way (deprecated)
//   config := config.GetGlobalConfig()
//
//   // New way (recommended)
//   container, err := container.NewContainer(config)
//   if err != nil {
//       log.Fatal(err)
//   }
//   config := container.GetConfig()
func GetGlobalConfig() *Config {
	configOnce.Do(func() {
		var err error
		globalConfig, err = InitializeConfig()
		if err != nil {
			Logger.Fatalf("Failed to initialize configuration: %v", err)
		}
		// Update global logger with new configuration
		Logger = ConfigureLoggingFromConfig(globalConfig)
	})
	return globalConfig
}

// InitializeGlobalConfig explicitly initializes the global configuration
// This is useful for testing or when you want to ensure config is loaded early
func InitializeGlobalConfig() error {
	var err error
	globalConfig, err = InitializeConfig()
	if err != nil {
		return err
	}
	Logger = ConfigureLoggingFromConfig(globalConfig)
	return nil
}

// Legacy compatibility functions - these will be deprecated in future versions

// GetCSVDelimiter returns the CSV delimiter from configuration
// Deprecated: Use dependency injection with container.NewContainer instead.
// This function will be removed in v2.0.0.
func GetCSVDelimiter() string {
	config := GetGlobalConfig()
	return config.CSV.Delimiter
}

// GetLogLevel returns the log level from configuration
// Deprecated: Use dependency injection with container.NewContainer instead.
// This function will be removed in v2.0.0.
func GetLogLevel() string {
	config := GetGlobalConfig()
	return config.Log.Level
}

// GetLogFormat returns the log format from configuration
// Deprecated: Use dependency injection with container.NewContainer instead.
// This function will be removed in v2.0.0.
func GetLogFormat() string {
	config := GetGlobalConfig()
	return config.Log.Format
}

// IsAIEnabled returns whether AI categorization is enabled
// Deprecated: Use dependency injection with container.NewContainer instead.
// This function will be removed in v2.0.0.
func IsAIEnabled() bool {
	config := GetGlobalConfig()
	return config.AI.Enabled
}

// GetAIModel returns the AI model name
// Deprecated: Use dependency injection with container.NewContainer instead.
// This function will be removed in v2.0.0.
func GetAIModel() string {
	config := GetGlobalConfig()
	return config.AI.Model
}

// GetAIRequestsPerMinute returns the AI requests per minute limit
// Deprecated: Use dependency injection with container.NewContainer instead.
// This function will be removed in v2.0.0.
func GetAIRequestsPerMinute() int {
	config := GetGlobalConfig()
	return config.AI.RequestsPerMinute
}
