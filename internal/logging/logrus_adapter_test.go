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
		name     string
		logFunc  func(Logger, string, ...Field)
		level    logrus.Level
		message  string
		fields   []Field
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
