// Package logging provides a logging abstraction layer that decouples the application
// from specific logging frameworks. This allows for easier testing and flexibility
// in choosing logging implementations.
package logging

// Logger defines the interface for structured logging throughout the application.
// Implementations should provide structured logging with support for fields and error context.
type Logger interface {
	// Debug logs a debug-level message with optional fields
	Debug(msg string, fields ...Field)
	
	// Info logs an info-level message with optional fields
	Info(msg string, fields ...Field)
	
	// Warn logs a warning-level message with optional fields
	Warn(msg string, fields ...Field)
	
	// Error logs an error-level message with optional fields
	Error(msg string, fields ...Field)
	
	// WithError returns a new logger with an error field attached
	WithError(err error) Logger
	
	// WithField returns a new logger with a single field attached
	WithField(key string, value interface{}) Logger
	
	// WithFields returns a new logger with multiple fields attached
	WithFields(fields ...Field) Logger
	
	// Fatal logs a fatal-level message and exits the program
	Fatal(msg string, fields ...Field)
	
	// Fatalf logs a fatal-level message with formatting and exits the program
	Fatalf(msg string, args ...interface{})
}

// Field represents a key-value pair for structured logging.
// Fields provide context to log messages without cluttering the message text.
type Field struct {
	Key   string
	Value interface{}
}
