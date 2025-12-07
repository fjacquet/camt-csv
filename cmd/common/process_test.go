package common_test

import (
	"io"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
)

// MockFullParser implements parser.FullParser for testing
type MockFullParser struct {
	ValidateResult bool
	ValidateError  error
	ConvertError   error
	ParseResult    []models.Transaction
	ParseError     error
	logger         logging.Logger
}

func (m *MockFullParser) Parse(r io.Reader) ([]models.Transaction, error) {
	return m.ParseResult, m.ParseError
}

func (m *MockFullParser) ConvertToCSV(inputFile, outputFile string) error {
	return m.ConvertError
}

func (m *MockFullParser) SetLogger(logger logging.Logger) {
	m.logger = logger
}

func (m *MockFullParser) SetCategorizer(categorizer interface{}) {}

func (m *MockFullParser) ValidateFormat(file string) (bool, error) {
	return m.ValidateResult, m.ValidateError
}

// TestMockFullParser_ImplementsInterface ensures our mock implements the required interface
func TestMockFullParser_ImplementsInterface(t *testing.T) {
	parser := &MockFullParser{}

	// Test SetLogger
	mockLogger := &logging.MockLogger{}
	parser.SetLogger(mockLogger)
	assert.NotNil(t, parser.logger)

	// Test SetCategorizer
	parser.SetCategorizer(nil)

	// Test ValidateFormat
	valid, err := parser.ValidateFormat("test.xml")
	assert.NoError(t, err)
	assert.False(t, valid)

	// Test ConvertToCSV
	err = parser.ConvertToCSV("input.xml", "output.csv")
	assert.NoError(t, err)

	// Test Parse
	txns, err := parser.Parse(nil)
	assert.NoError(t, err)
	assert.Empty(t, txns)
}

// TestMockFullParser_WithErrors tests error scenarios
func TestMockFullParser_WithErrors(t *testing.T) {
	parser := &MockFullParser{
		ValidateError: assert.AnError,
		ConvertError:  assert.AnError,
		ParseError:    assert.AnError,
	}

	_, err := parser.ValidateFormat("test.xml")
	assert.Error(t, err)

	err = parser.ConvertToCSV("input.xml", "output.csv")
	assert.Error(t, err)

	_, err = parser.Parse(nil)
	assert.Error(t, err)
}

// TestMockLogger_CapturesEntries tests that the mock logger captures entries
func TestMockLogger_CapturesEntries(t *testing.T) {
	logger := &logging.MockLogger{}

	logger.Info("test message")
	logger.Warn("warning message")
	logger.Error("error message")
	logger.Fatalf("fatal: %s", "critical error")

	entries := logger.GetEntries()
	assert.Len(t, entries, 4)
	assert.True(t, logger.HasEntry("INFO", "test message"))
	assert.True(t, logger.HasEntry("WARN", "warning message"))
	assert.True(t, logger.HasEntry("ERROR", "error message"))
	assert.True(t, logger.HasEntry("FATAL", "fatal: critical error"))
}
