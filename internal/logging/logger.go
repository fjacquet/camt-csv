// Package logging provides a centralized logging facility for the entire application
package logging

import (
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	// Logger is the global logger instance for the application
	Logger *logrus.Logger

	// Ensure initialization happens only once
	once sync.Once
)

// Initialize sets up the global logger with the specified configuration
func Initialize() *logrus.Logger {
	once.Do(func() {
		Logger = logrus.New()

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
	})

	return Logger
}

// GetLogger returns the global logger instance, initializing it if necessary
func GetLogger() *logrus.Logger {
	if Logger == nil {
		Initialize()
	}
	return Logger
}

// SetLogger sets the global logger instance
func SetLogger(logger *logrus.Logger) {
	Logger = logger
}

// Debug logs a message at the debug level
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Debugf logs a formatted message at the debug level
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info logs a message at the info level
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Infof logs a formatted message at the info level
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn logs a message at the warn level
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Warnf logs a formatted message at the warn level
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error logs a message at the error level
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Errorf logs a formatted message at the error level
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal logs a message at the fatal level and then exits
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// Fatalf logs a formatted message at the fatal level and then exits
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// WithField adds a field to the log entry
func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

// WithFields adds multiple fields to the log entry
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// SetupPackageLogger creates or configures a package-level logger
// This is a helper function for transitioning existing packages to use the centralized logger
func SetupPackageLogger(pkgLogger *logrus.Logger) *logrus.Logger {
	if pkgLogger == nil {
		pkgLogger = logrus.New()
	}

	// Apply the same configuration as the global logger
	globalLogger := GetLogger()

	// Copy configuration from the global logger
	pkgLogger.SetLevel(globalLogger.GetLevel())
	pkgLogger.SetFormatter(globalLogger.Formatter)
	pkgLogger.SetOutput(globalLogger.Out)

	return pkgLogger
}

// SetAllLogLevels forces a specific log level across all loggers in the system
// This should be called early in application startup after environment variables are loaded
func SetAllLogLevels(level logrus.Level) {
	// Initialize our central logger with this level
	once.Do(func() {
		if Logger == nil {
			Logger = logrus.New()
		}

		Logger.SetLevel(level)

		// Use text formatter by default unless JSON is explicitly configured
		logFormat := os.Getenv("LOG_FORMAT")
		if strings.ToLower(logFormat) == "json" {
			Logger.SetFormatter(&logrus.JSONFormatter{})
		} else {
			Logger.SetFormatter(&logrus.TextFormatter{
				FullTimestamp: true,
			})
		}
	})

	// If loggers are already initialized, this ensures they still get the correct level
	if Logger != nil {
		Logger.SetLevel(level)
	}
}

// ParseLogLevelFromEnv reads LOG_LEVEL from environment and returns the appropriate logrus level
// It defaults to InfoLevel if not specified or invalid
func ParseLogLevelFromEnv() logrus.Level {
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		return logrus.InfoLevel
	}

	level, err := logrus.ParseLevel(strings.ToLower(logLevelStr))
	if err != nil {
		return logrus.InfoLevel
	}

	return level
}
