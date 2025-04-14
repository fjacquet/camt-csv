// Package config provides functionality for loading and accessing environment variables.
package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var (
	once sync.Once
	log  = logrus.New()
)

// LoadEnv loads environment variables from .env file if it exists
func LoadEnv() {
	once.Do(func() {
		// Try to find .env file in current directory
		envFile := ".env"
		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			// Try to find .env in parent directory (project root)
			workDir, err := os.Getwd()
			if err == nil {
				parentEnvFile := filepath.Join(filepath.Dir(workDir), ".env")
				if _, err := os.Stat(parentEnvFile); err == nil {
					envFile = parentEnvFile
				}
			}
		}

		err := godotenv.Load(envFile)
		if err != nil {
			log.Warnf("Error loading .env file: %v", err)
			log.Info("Continuing with existing environment variables")
		} else {
			log.Infof("Environment variables loaded from %s", envFile)
		}
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
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// GetGeminiAPIKey retrieves the Gemini API key from environment variables
func GetGeminiAPIKey() string {
	return GetEnv("GEMINI_API_KEY", "")
}
