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
)

// ConfigureLogging sets up logging based on environment variables and returns the configured logger.
// It configures both the log level (from LOG_LEVEL env var) and format (from LOG_FORMAT env var).
//
// Supported log levels: debug, info, warn, error, fatal, panic
// Supported log formats: json, text (default)
//
// Returns:
//   - *logrus.Logger: Configured logger instance ready for use
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
// It searches for .env files in the current directory and parent directory.
// This function is safe to call multiple times as it uses sync.Once internally.
//
// The function automatically configures logging after loading environment variables.
// If no .env file is found, it continues without error, relying on system environment variables.
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
// This is a utility function for safely accessing environment variables with defaults.
//
// Parameters:
//   - key: The environment variable name to retrieve
//   - fallback: The default value to return if the environment variable is not set
//
// Returns:
//   - string: The environment variable value or the fallback value
func GetEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}

// MustGetEnv retrieves an environment variable or panics if not set.
// This function should be used for required environment variables where the application
// cannot continue without the value.
//
// Parameters:
//   - key: The environment variable name to retrieve
//
// Returns:
//   - string: The environment variable value
//
// Panics:
//   - If the environment variable is not set or is empty
func MustGetEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		Logger.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// GetGeminiAPIKey returns the Gemini API key from environment variables.
// This is a convenience function for accessing the GEMINI_API_KEY environment variable.
// Returns an empty string if the API key is not set.
//
// Returns:
//   - string: The Gemini API key or empty string if not configured
func GetGeminiAPIKey() string {
	return GetEnv("GEMINI_API_KEY", "")
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
