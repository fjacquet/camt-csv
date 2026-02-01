package logging

import (
	"fmt"
	"strings"
	"sync"
)

// NewMockLogger creates a new MockLogger instance for testing.
func NewMockLogger() *MockLogger {
	entries := make([]LogEntry, 0)
	return &MockLogger{
		entries:       &entries,
		pendingError:  nil,
		pendingFields: nil,
	}
}

// MockLogger is a mock implementation of the Logger interface for testing.
// It captures log entries for verification in tests.
// It is thread-safe for concurrent use.
//
// Note: entries is a pointer to a slice, which allows child loggers created
// via WithError/WithFields to share the same log entry collection with the
// parent logger, while maintaining independent pending fields and errors.
type MockLogger struct {
	mu            sync.RWMutex
	entries       *[]LogEntry // Shared across parent and child loggers
	pendingError  error
	pendingFields []Field
}

// Entries returns all captured log entries.
// Deprecated: Use GetEntries() instead for thread-safe access.
func (m *MockLogger) Entries() []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.entries == nil {
		return nil
	}
	return *m.entries
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
	m.ensureEntriesInitialized()
	allFields := append(m.pendingFields, fields...)
	*m.entries = append(*m.entries, LogEntry{
		Level:   "DEBUG",
		Message: msg,
		Fields:  allFields,
		Error:   m.pendingError,
	})
}

// ensureEntriesInitialized ensures the entries pointer is initialized.
// This handles cases where MockLogger is created via struct literal instead of NewMockLogger.
// Must be called with mu.Lock held.
func (m *MockLogger) ensureEntriesInitialized() {
	if m.entries == nil {
		entries := make([]LogEntry, 0)
		m.entries = &entries
	}
}

// Info logs an info-level message with optional fields.
func (m *MockLogger) Info(msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureEntriesInitialized()
	allFields := append(m.pendingFields, fields...)
	*m.entries = append(*m.entries, LogEntry{
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
	m.ensureEntriesInitialized()
	allFields := append(m.pendingFields, fields...)
	*m.entries = append(*m.entries, LogEntry{
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
	m.ensureEntriesInitialized()
	allFields := append(m.pendingFields, fields...)
	*m.entries = append(*m.entries, LogEntry{
		Level:   "ERROR",
		Message: msg,
		Fields:  allFields,
		Error:   m.pendingError,
	})
}

// WithError returns a new logger with an error field attached.
// The child logger shares the same entries collection but has independent pending error.
func (m *MockLogger) WithError(err error) Logger {
	m.mu.Lock() // Need write lock to potentially initialize entries
	defer m.mu.Unlock()
	m.ensureEntriesInitialized() // Ensure entries are initialized before sharing
	return &MockLogger{
		entries:       m.entries, // Share the same entries pointer
		pendingError:  err,
		pendingFields: m.pendingFields,
	}
}

// WithField returns a new logger with a single field attached.
func (m *MockLogger) WithField(key string, value interface{}) Logger {
	return m.WithFields(Field{Key: key, Value: value})
}

// WithFields returns a new logger with multiple fields attached.
// The child logger shares the same entries collection but has independent pending fields.
func (m *MockLogger) WithFields(fields ...Field) Logger {
	m.mu.Lock() // Need write lock to potentially initialize entries
	defer m.mu.Unlock()
	m.ensureEntriesInitialized() // Ensure entries are initialized before sharing
	allFields := append(m.pendingFields, fields...)
	return &MockLogger{
		entries:       m.entries, // Share the same entries pointer
		pendingError:  m.pendingError,
		pendingFields: allFields,
	}
}

// Fatal logs a fatal-level message and exits the program.
// In the mock implementation, we don't actually exit.
func (m *MockLogger) Fatal(msg string, fields ...Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureEntriesInitialized()
	allFields := append(m.pendingFields, fields...)
	*m.entries = append(*m.entries, LogEntry{
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
	m.ensureEntriesInitialized()
	*m.entries = append(*m.entries, LogEntry{
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
	if m.entries == nil {
		return nil
	}
	return *m.entries
}

// GetEntriesByLevel returns all log entries of a specific level.
func (m *MockLogger) GetEntriesByLevel(level string) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.entries == nil {
		return nil
	}
	var entries []LogEntry
	for _, entry := range *m.entries {
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
	if m.entries != nil {
		*m.entries = []LogEntry{}
	}
}

// HasEntry checks if a log entry with the given level and message exists.
func (m *MockLogger) HasEntry(level, message string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.entries == nil {
		return false
	}
	for _, entry := range *m.entries {
		if entry.Level == level && entry.Message == message {
			return true
		}
	}
	return false
}

// VerifyFatalLog checks if at least one FATAL log entry contains the expected message substring.
// Returns true if found, false otherwise.
// If verification fails and entries exist, prints all log entries for debugging.
func (m *MockLogger) VerifyFatalLog(expectedMessage string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.entries == nil {
		return false
	}
	for _, entry := range *m.entries {
		if entry.Level == "FATAL" && strings.Contains(entry.Message, expectedMessage) {
			return true
		}
	}
	return false
}

// VerifyFatalLogWithDebug is like VerifyFatalLog but prints all entries if verification fails.
// Useful for debugging test failures.
func (m *MockLogger) VerifyFatalLogWithDebug(expectedMessage string) bool {
	found := m.VerifyFatalLog(expectedMessage)
	if !found && m.entries != nil && len(*m.entries) > 0 {
		fmt.Println("VerifyFatalLog failed. All log entries:")
		for i, entry := range *m.entries {
			fmt.Printf("  [%d] %s: %s", i, entry.Level, entry.Message)
			if entry.Error != nil {
				fmt.Printf(" (error: %v)", entry.Error)
			}
			if len(entry.Fields) > 0 {
				fmt.Printf(" %+v", entry.Fields)
			}
			fmt.Println()
		}
	}
	return found
}
