package logging

import (
	"github.com/sirupsen/logrus"
)

// LogrusAdapter adapts logrus.Logger to implement our Logger interface.
// This allows us to use logrus as the underlying logging implementation
// while keeping the rest of the codebase decoupled from it.
type LogrusAdapter struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// NewLogrusAdapter creates a new LogrusAdapter with the specified log level and format.
//
// Parameters:
//   - level: Log level as string ("debug", "info", "warn", "error")
//   - format: Log format as string ("json" or "text")
//
// Returns a Logger interface implementation backed by logrus.
func NewLogrusAdapter(level, format string) Logger {
	logger := logrus.New()

	// Parse and set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logger.Warnf("Invalid log level '%s', using 'info'", level)
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set log format
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return &LogrusAdapter{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}
}

// NewLogrusAdapterFromLogger creates a LogrusAdapter from an existing logrus.Logger.
// This is useful for maintaining compatibility with existing code.
func NewLogrusAdapterFromLogger(logger *logrus.Logger) Logger {
	if logger == nil {
		logger = logrus.New()
	}
	return &LogrusAdapter{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}
}

// Debug logs a debug-level message with optional fields
func (l *LogrusAdapter) Debug(msg string, fields ...Field) {
	l.entry.WithFields(convertFields(fields)).Debug(msg)
}

// Info logs an info-level message with optional fields
func (l *LogrusAdapter) Info(msg string, fields ...Field) {
	l.entry.WithFields(convertFields(fields)).Info(msg)
}

// Warn logs a warning-level message with optional fields
func (l *LogrusAdapter) Warn(msg string, fields ...Field) {
	l.entry.WithFields(convertFields(fields)).Warn(msg)
}

// Error logs an error-level message with optional fields
func (l *LogrusAdapter) Error(msg string, fields ...Field) {
	l.entry.WithFields(convertFields(fields)).Error(msg)
}

// WithError returns a new logger with an error field attached
func (l *LogrusAdapter) WithError(err error) Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithError(err),
	}
}

// WithField returns a new logger with a single field attached
func (l *LogrusAdapter) WithField(key string, value interface{}) Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields returns a new logger with multiple fields attached
func (l *LogrusAdapter) WithFields(fields ...Field) Logger {
	return &LogrusAdapter{
		logger: l.logger,
		entry:  l.entry.WithFields(convertFields(fields)),
	}
}

// convertFields converts our Field slice to logrus.Fields map
func convertFields(fields []Field) logrus.Fields {
	logrusFields := make(logrus.Fields, len(fields))
	for _, field := range fields {
		logrusFields[field.Key] = field.Value
	}
	return logrusFields
}

// Fatal logs a fatal-level message and exits the program
func (l *LogrusAdapter) Fatal(msg string, fields ...Field) {
	l.entry.WithFields(convertFields(fields)).Fatal(msg)
}

// Fatalf logs a fatal-level message with formatting and exits the program
func (l *LogrusAdapter) Fatalf(msg string, args ...interface{}) {
	l.entry.Fatalf(msg, args...)
}

// Convenience methods for backward compatibility with old logging style
func (l *LogrusAdapter) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

func (l *LogrusAdapter) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

func (l *LogrusAdapter) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

func (l *LogrusAdapter) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}
