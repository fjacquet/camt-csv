package logging

import (
	"fmt"
	"sync"
)

// NewMockLogger creates a new MockLogger instance for testing.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Entries: []LogEntry{},
	}
}

// MockLogger is a mock implementation of the Logger interface for testing.
// It captures log entries for verification in tests.
// It is thread-safe for concurrent use.
type MockLogger struct {
	mu            sync.RWMutex
	Entries       []LogEntry
	pendingError  error
	pendingFields []Field
}

// LogEntry represents a single log entry captured by MockLogger.
type LogEntry struct {
	Level   string
	Message string
	Fields  []Field
	Error   error
}

// Debug logs a debug-level message with optional fields.
func (m *MockLogger) Debug(msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	allFields := append(m.pendingFields, fields...)
	m.Entries = append(m.Entries, LogEntry{
		Level:   "DEBUG",
		Message: msg,
		Fields:  allFields,
		Error:   m.pendingError,
	})
}

// Info logs an info-level message with optional fields.
func (m *MockLogger) Info(msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	allFields := append(m.pendingFields, fields...)
	m.Entries = append(m.Entries, LogEntry{
		Level:   "INFO",
		Message: msg,
		Fields:  allFields,
		Error:   m.pendingError,
	})
}

// Warn logs a warning-level message with optional fields.
func (m *MockLogger) Warn(msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	allFields := append(m.pendingFields, fields...)
	m.Entries = append(m.Entries, LogEntry{
		Level:   "WARN",
		Message: msg,
		Fields:  allFields,
		Error:   m.pendingError,
	})
}

// Error logs an error-level message with optional fields.
func (m *MockLogger) Error(msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	allFields := append(m.pendingFields, fields...)
	m.Entries = append(m.Entries, LogEntry{
		Level:   "ERROR",
		Message: msg,
		Fields:  allFields,
		Error:   m.pendingError,
	})
}

// WithError returns a new logger with an error field attached.
func (m *MockLogger) WithError(err error) Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Copy Entries slice to prevent state sharing between parent and child loggers
	entriesCopy := make([]LogEntry, len(m.Entries))
	copy(entriesCopy, m.Entries)
	return &MockLogger{
		Entries:       entriesCopy,
		pendingError:  err,
		pendingFields: m.pendingFields,
	}
}

// WithField returns a new logger with a single field attached.
func (m *MockLogger) WithField(key string, value interface{}) Logger {
	return m.WithFields(Field{Key: key, Value: value})
}

// WithFields returns a new logger with multiple fields attached.
func (m *MockLogger) WithFields(fields ...Field) Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	allFields := append(m.pendingFields, fields...)
	// Copy Entries slice to prevent state sharing between parent and child loggers
	entriesCopy := make([]LogEntry, len(m.Entries))
	copy(entriesCopy, m.Entries)
	return &MockLogger{
		Entries:       entriesCopy,
		pendingError:  m.pendingError,
		pendingFields: allFields,
	}
}

// Fatal logs a fatal-level message and exits the program.
// In the mock implementation, we don't actually exit.
func (m *MockLogger) Fatal(msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	allFields := append(m.pendingFields, fields...)
	m.Entries = append(m.Entries, LogEntry{
		Level:   "FATAL",
		Message: msg,
		Fields:  allFields,
		Error:   m.pendingError,
	})
}

// Fatalf logs a fatal-level message with formatting and exits the program.
// In the mock implementation, we don't actually exit.
func (m *MockLogger) Fatalf(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Entries = append(m.Entries, LogEntry{
		Level:   "FATAL",
		Message: fmt.Sprintf(msg, args...),
		Fields:  m.pendingFields,
		Error:   m.pendingError,
	})
}

// GetEntries returns all captured log entries.
func (m *MockLogger) GetEntries() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Entries
}

// GetEntriesByLevel returns all log entries of a specific level.
func (m *MockLogger) GetEntriesByLevel(level string) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var entries []LogEntry
	for _, entry := range m.Entries {
		if entry.Level == level {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Clear removes all captured log entries.
func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Entries = []LogEntry{}
}

// HasEntry checks if a log entry with the given level and message exists.
func (m *MockLogger) HasEntry(level, message string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, entry := range m.Entries {
		if entry.Level == level && entry.Message == message {
			return true
		}
	}
	return false
}
