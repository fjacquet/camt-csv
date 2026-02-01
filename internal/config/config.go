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
	"sync"

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

