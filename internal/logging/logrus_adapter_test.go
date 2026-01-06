package logging

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogrusAdapter(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		format      string
		expectLevel logrus.Level
	}{
		{
			name:        "debug level with text format",
			level:       "debug",
			format:      "text",
			expectLevel: logrus.DebugLevel,
		},
		{
			name:        "info level with json format",
			level:       "info",
			format:      "json",
			expectLevel: logrus.InfoLevel,
		},
		{
			name:        "warn level with text format",
			level:       "warn",
			format:      "text",
			expectLevel: logrus.WarnLevel,
		},
		{
			name:        "error level with json format",
			level:       "error",
			format:      "json",
			expectLevel: logrus.ErrorLevel,
		},
		{
			name:        "invalid level defaults to info",
			level:       "invalid",
			format:      "text",
			expectLevel: logrus.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogrusAdapter(tt.level, tt.format)
			require.NotNil(t, logger)

			adapter, ok := logger.(*LogrusAdapter)
			require.True(t, ok, "logger should be a LogrusAdapter")
			assert.Equal(t, tt.expectLevel, adapter.logger.Level)

			// Check formatter type
			if tt.format == "json" {
				_, ok := adapter.logger.Formatter.(*logrus.JSONFormatter)
				assert.True(t, ok, "formatter should be JSONFormatter")
			} else {
				_, ok := adapter.logger.Formatter.(*logrus.TextFormatter)
				assert.True(t, ok, "formatter should be TextFormatter")
			}
		})
	}
}

func TestNewLogrusAdapterFromLogger(t *testing.T) {
	t.Run("with existing logger", func(t *testing.T) {
		existingLogger := logrus.New()
		existingLogger.SetLevel(logrus.DebugLevel)

		logger := NewLogrusAdapterFromLogger(existingLogger)
		require.NotNil(t, logger)

		adapter, ok := logger.(*LogrusAdapter)
		require.True(t, ok)
		assert.Equal(t, existingLogger, adapter.logger)
	})

	t.Run("with nil logger creates new one", func(t *testing.T) {
		logger := NewLogrusAdapterFromLogger(nil)
		require.NotNil(t, logger)

		adapter, ok := logger.(*LogrusAdapter)
		require.True(t, ok)
		assert.NotNil(t, adapter.logger)
	})
}

func TestLogrusAdapter_LoggingMethods(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(Logger, string, ...Field)
		level   logrus.Level
		message string
		fields  []Field
	}{
		{
			name:    "Debug with fields",
			logFunc: func(l Logger, msg string, fields ...Field) { l.Debug(msg, fields...) },
			level:   logrus.DebugLevel,
			message: "debug message",
			fields:  []Field{{Key: "key1", Value: "value1"}},
		},
		{
			name:    "Info with fields",
			logFunc: func(l Logger, msg string, fields ...Field) { l.Info(msg, fields...) },
			level:   logrus.InfoLevel,
			message: "info message",
			fields:  []Field{{Key: "key2", Value: "value2"}},
		},
		{
			name:    "Warn with fields",
			logFunc: func(l Logger, msg string, fields ...Field) { l.Warn(msg, fields...) },
			level:   logrus.WarnLevel,
			message: "warn message",
			fields:  []Field{{Key: "key3", Value: "value3"}},
		},
		{
			name:    "Error with fields",
			logFunc: func(l Logger, msg string, fields ...Field) { l.Error(msg, fields...) },
			level:   logrus.ErrorLevel,
			message: "error message",
			fields:  []Field{{Key: "key4", Value: "value4"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create logger with buffer to capture output
			logrusLogger := logrus.New()
			var buf bytes.Buffer
			logrusLogger.SetOutput(&buf)
			logrusLogger.SetLevel(logrus.DebugLevel)
			logrusLogger.SetFormatter(&logrus.TextFormatter{
				DisableTimestamp: true,
			})

			logger := NewLogrusAdapterFromLogger(logrusLogger)

			// Call the logging method
			tt.logFunc(logger, tt.message, tt.fields...)

			// Verify output contains message
			output := buf.String()
			assert.Contains(t, output, tt.message)

			// Verify output contains field key and value
			if len(tt.fields) > 0 {
				assert.Contains(t, output, tt.fields[0].Key)
			}
		})
	}
}

func TestLogrusAdapter_WithError(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.ErrorLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)
	testErr := errors.New("test error")

	loggerWithError := logger.WithError(testErr)
	loggerWithError.Error("error occurred")

	output := buf.String()
	assert.Contains(t, output, "error occurred")
	assert.Contains(t, output, "test error")
}

func TestLogrusAdapter_WithField(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.InfoLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)

	loggerWithField := logger.WithField("user", "john")
	loggerWithField.Info("user action")

	output := buf.String()
	assert.Contains(t, output, "user action")
	assert.Contains(t, output, "user")
	assert.Contains(t, output, "john")
}

func TestLogrusAdapter_WithFields(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.InfoLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)

	fields := []Field{
		{Key: "user", Value: "john"},
		{Key: "action", Value: "login"},
		{Key: "ip", Value: "192.168.1.1"},
	}

	loggerWithFields := logger.WithFields(fields...)
	loggerWithFields.Info("user logged in")

	output := buf.String()
	assert.Contains(t, output, "user logged in")
	assert.Contains(t, output, "user")
	assert.Contains(t, output, "john")
	assert.Contains(t, output, "action")
	assert.Contains(t, output, "login")
}

func TestConvertFields(t *testing.T) {
	fields := []Field{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: 42},
		{Key: "key3", Value: true},
	}

	logrusFields := convertFields(fields)

	assert.Len(t, logrusFields, 3)
	assert.Equal(t, "value1", logrusFields["key1"])
	assert.Equal(t, 42, logrusFields["key2"])
	assert.Equal(t, true, logrusFields["key3"])
}

func TestConvertFields_Empty(t *testing.T) {
	fields := []Field{}
	logrusFields := convertFields(fields)
	assert.Len(t, logrusFields, 0)
}

// TestGetLogger removed - we no longer use global logger functions
// All loggers are now injected through constructors

func TestLogrusAdapter_ChainedCalls(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.InfoLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)
	testErr := errors.New("test error")

	// Chain multiple WithField calls
	logger.
		WithField("user", "alice").
		WithField("action", "delete").
		WithError(testErr).
		Error("operation failed")

	output := buf.String()
	assert.Contains(t, output, "operation failed")
	assert.Contains(t, output, "user")
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "action")
	assert.Contains(t, output, "delete")
	assert.Contains(t, output, "test error")
}

func TestFieldConstants(t *testing.T) {
	// Verify field constants are defined (from constants.go)
	assert.Equal(t, "file_path", FieldFile)
	assert.Equal(t, "count", FieldCount)
	assert.Equal(t, "input_file", FieldInputFile)
	assert.Equal(t, "output_file", FieldOutputFile)
	assert.Equal(t, "delimiter", FieldDelimiter)
	assert.Equal(t, "error", FieldError)
	assert.Equal(t, "category", FieldCategory)
	assert.Equal(t, "parser", FieldParser)
	assert.Equal(t, "transaction_id", FieldTransactionID)
}

func TestLogrusAdapter_ImplementsInterface(t *testing.T) {
	var _ Logger = (*LogrusAdapter)(nil)
}

func TestLogrusAdapter_BackwardCompatibilityMethods(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.DebugLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)
	adapter := logger.(*LogrusAdapter)

	// Test Infof
	adapter.Infof("info message with %s", "formatting")
	output := buf.String()
	assert.Contains(t, output, "info message with formatting")

	// Reset buffer
	buf.Reset()

	// Test Errorf
	adapter.Errorf("error message with %d", 42)
	output = buf.String()
	assert.Contains(t, output, "error message with 42")

	// Reset buffer
	buf.Reset()

	// Test Warnf
	adapter.Warnf("warn message with %v", true)
	output = buf.String()
	assert.Contains(t, output, "warn message with true")

	// Reset buffer
	buf.Reset()

	// Test Debugf
	adapter.Debugf("debug message with %s", "debug")
	output = buf.String()
	assert.Contains(t, output, "debug message with debug")
}

func TestLogrusAdapter_FatalMethods(t *testing.T) {
	// Note: We can't easily test Fatal methods as they call os.Exit
	// This test just verifies the methods exist and can be called
	// In a real scenario, these would terminate the program
	
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.FatalLevel)
	
	logger := NewLogrusAdapterFromLogger(logrusLogger)
	adapter := logger.(*LogrusAdapter)

	// Verify methods exist (compilation test)
	assert.NotNil(t, adapter.Fatal)
	assert.NotNil(t, adapter.Fatalf)
}

func TestLogrusAdapter_MultipleFields(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.InfoLevel)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})

	logger := NewLogrusAdapterFromLogger(logrusLogger)

	fields := []Field{
		{Key: "string_field", Value: "test"},
		{Key: "int_field", Value: 123},
		{Key: "bool_field", Value: false},
		{Key: "float_field", Value: 3.14},
		{Key: "nil_field", Value: nil},
	}

	logger.Info("test message", fields...)

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "string_field")
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "int_field")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "bool_field")
	assert.Contains(t, output, "false")
}

func TestLogrusAdapter_EmptyFields(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.InfoLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)

	// Test with no fields
	logger.Info("message without fields")

	output := buf.String()
	assert.Contains(t, output, "message without fields")
}

func TestLogrusAdapter_LevelFiltering(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.WarnLevel) // Only warn and above
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)

	// These should not appear in output
	logger.Debug("debug message")
	logger.Info("info message")

	// These should appear
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogrusAdapter_NestedWithCalls(t *testing.T) {
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.InfoLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logger := NewLogrusAdapterFromLogger(logrusLogger)
	testErr := errors.New("nested error")

	// Test nested With calls
	nestedLogger := logger.
		WithField("level1", "value1").
		WithFields(Field{Key: "level2", Value: "value2"}).
		WithError(testErr).
		WithField("level3", "value3")

	nestedLogger.Info("nested logging test")

	output := buf.String()
	assert.Contains(t, output, "nested logging test")
	assert.Contains(t, output, "level1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "level2")
	assert.Contains(t, output, "value2")
	assert.Contains(t, output, "level3")
	assert.Contains(t, output, "value3")
	assert.Contains(t, output, "nested error")
}

func TestAllFieldConstants(t *testing.T) {
	// Test all field constants are properly defined
	constants := map[string]string{
		FieldFile:          "file_path",
		FieldParser:        "parser",
		FieldTransactionID: "transaction_id",
		FieldCategory:      "category",
		FieldReason:        "reason",
		FieldOperation:     "operation",
		FieldStatus:        "status",
		FieldError:         "error",
		FieldDuration:      "duration_ms",
		FieldCount:         "count",
		FieldDelimiter:     "delimiter",
		FieldInputFile:     "input_file",
		FieldOutputFile:    "output_file",
	}

	for constant, expected := range constants {
		assert.Equal(t, expected, constant, "Field constant should match expected value")
	}
}

func TestLogrusAdapter_JSONFormat(t *testing.T) {
	logger := NewLogrusAdapter("info", "json")
	adapter := logger.(*LogrusAdapter)

	// Verify JSON formatter is set
	_, ok := adapter.logger.Formatter.(*logrus.JSONFormatter)
	assert.True(t, ok, "Should use JSON formatter")
}

func TestLogrusAdapter_TextFormat(t *testing.T) {
	logger := NewLogrusAdapter("info", "text")
	adapter := logger.(*LogrusAdapter)

	// Verify Text formatter is set
	textFormatter, ok := adapter.logger.Formatter.(*logrus.TextFormatter)
	assert.True(t, ok, "Should use Text formatter")
	assert.True(t, textFormatter.FullTimestamp, "Should have full timestamp enabled")
}

func TestLogrusAdapter_InvalidLevelHandling(t *testing.T) {
	// Capture logrus warnings
	logrusLogger := logrus.New()
	var buf bytes.Buffer
	logrusLogger.SetOutput(&buf)

	// This should trigger a warning and default to info level
	logger := NewLogrusAdapter("invalid-level", "text")
	adapter := logger.(*LogrusAdapter)

	assert.Equal(t, logrus.InfoLevel, adapter.logger.Level, "Should default to info level for invalid input")
}

func TestField_Structure(t *testing.T) {
	field := Field{
		Key:   "test_key",
		Value: "test_value",
	}

	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, "test_value", field.Value)
}

func TestConvertFields_VariousTypes(t *testing.T) {
	fields := []Field{
		{Key: "string", Value: "text"},
		{Key: "int", Value: 42},
		{Key: "float", Value: 3.14},
		{Key: "bool", Value: true},
		{Key: "nil", Value: nil},
		{Key: "slice", Value: []string{"a", "b"}},
		{Key: "map", Value: map[string]int{"x": 1}},
	}

	logrusFields := convertFields(fields)

	assert.Len(t, logrusFields, 7)
	assert.Equal(t, "text", logrusFields["string"])
	assert.Equal(t, 42, logrusFields["int"])
	assert.Equal(t, 3.14, logrusFields["float"])
	assert.Equal(t, true, logrusFields["bool"])
	assert.Nil(t, logrusFields["nil"])
	assert.Equal(t, []string{"a", "b"}, logrusFields["slice"])
	assert.Equal(t, map[string]int{"x": 1}, logrusFields["map"])
}
