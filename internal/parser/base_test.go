package parser

import (
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
)

// mockLogger implements the logging.Logger interface for testing
type mockLogger struct {
	messages []string
}

func (m *mockLogger) Debug(msg string, fields ...logging.Field) {
	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, fields ...logging.Field) {
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, fields ...logging.Field) {
	m.messages = append(m.messages, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, fields ...logging.Field) {
	m.messages = append(m.messages, "ERROR: "+msg)
}

func (m *mockLogger) WithError(err error) logging.Logger {
	return m
}

func (m *mockLogger) WithField(key string, value interface{}) logging.Logger {
	return m
}

func (m *mockLogger) WithFields(fields ...logging.Field) logging.Logger {
	return m
}

func (m *mockLogger) Fatal(msg string, fields ...logging.Field) {
	m.messages = append(m.messages, "FATAL: "+msg)
}

func (m *mockLogger) Fatalf(msg string, args ...interface{}) {
	m.messages = append(m.messages, "FATAL: "+msg)
}

func TestNewBaseParser(t *testing.T) {
	t.Run("with provided logger", func(t *testing.T) {
		mockLog := &mockLogger{}
		baseParser := NewBaseParser(mockLog)

		assert.NotNil(t, baseParser.logger)
		assert.Equal(t, mockLog, baseParser.logger)
	})

	t.Run("with nil logger uses default", func(t *testing.T) {
		baseParser := NewBaseParser(nil)

		assert.NotNil(t, baseParser.logger)
		// Should use a default logger (not nil)
		assert.NotNil(t, baseParser.GetLogger())
	})
}

func TestBaseParser_SetLogger(t *testing.T) {
	t.Run("sets new logger", func(t *testing.T) {
		baseParser := NewBaseParser(nil)
		mockLog := &mockLogger{}

		baseParser.SetLogger(mockLog)

		assert.Equal(t, mockLog, baseParser.logger)
	})

	t.Run("ignores nil logger", func(t *testing.T) {
		mockLog := &mockLogger{}
		baseParser := NewBaseParser(mockLog)
		originalLogger := baseParser.logger

		baseParser.SetLogger(nil)

		assert.Equal(t, originalLogger, baseParser.logger)
	})
}

func TestBaseParser_GetLogger(t *testing.T) {
	mockLog := &mockLogger{}
	baseParser := NewBaseParser(mockLog)

	logger := baseParser.GetLogger()

	assert.Equal(t, mockLog, logger)
}

func TestBaseParser_InterfaceCompliance(t *testing.T) {
	t.Run("implements LoggerConfigurable interface", func(t *testing.T) {
		var _ LoggerConfigurable = &BaseParser{}
	})
}
