// Package config provides functionality for loading and accessing environment variables.
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

// GetGeminiAPIKey retrieves the Gemini API key from environment variables
func GetGeminiAPIKey() string {
	return GetEnv("GEMINI_API_KEY", "")
}
