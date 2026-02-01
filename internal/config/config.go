// Package config provides functionality for loading and accessing environment variables.
//
// All deprecated functions and global state have been removed.
// Use the Viper-based configuration from viper.go:
//   - InitializeConfig() to load configuration
//   - Config struct fields for type-safe access
//   - container.GetLogger() for logging (via dependency injection)
package config

