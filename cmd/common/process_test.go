package common_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFullParser implements parser.FullParser for testing
type MockFullParser struct {
	mock.Mock
	ValidateResult bool
	ValidateError  error
	ParseResult    []models.Transaction
	ParseError     error
	logger         logging.Logger
}

func (m *MockFullParser) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	args := m.Called(ctx, r)
	return args.Get(0).([]models.Transaction), args.Error(1)
}

func (m *MockFullParser) SetLogger(logger logging.Logger) {
	m.Called(logger)
	m.logger = logger
}

func (m *MockFullParser) SetCategorizer(categorizer models.TransactionCategorizer) {
	m.Called(categorizer)
}

func (m *MockFullParser) ValidateFormat(file string) (bool, error) {
	args := m.Called(file)
	return args.Bool(0), args.Error(1)
}

// Test error constants
func TestErrInvalidFormat(t *testing.T) {
	assert.Equal(t, "file is not in a valid format", common.ErrInvalidFormat.Error())
	assert.True(t, errors.Is(common.ErrInvalidFormat, common.ErrInvalidFormat))
}

// Test that the original mock implementations still work
func TestMockFullParser_ImplementsInterface(t *testing.T) {
	parser := &MockFullParser{}

	// Setup expectations for all method calls
	mockLogger := &logging.MockLogger{}
	parser.On("SetLogger", mockLogger).Return()
	parser.On("SetCategorizer", mock.Anything).Return()
	parser.On("ValidateFormat", "test.xml").Return(false, nil)
	parser.On("Parse", mock.Anything, mock.Anything).Return([]models.Transaction{}, nil)

	// Test SetLogger
	parser.SetLogger(mockLogger)
	assert.NotNil(t, parser.logger)

	// Test SetCategorizer
	parser.SetCategorizer(nil)

	// Test ValidateFormat
	valid, err := parser.ValidateFormat("test.xml")
	assert.NoError(t, err)
	assert.False(t, valid)

	// Test Parse
	txns, err := parser.Parse(context.Background(), nil)
	assert.NoError(t, err)
	assert.Empty(t, txns)

	parser.AssertExpectations(t)
}

// TestMockFullParser_WithErrors tests error scenarios
func TestMockFullParser_WithErrors(t *testing.T) {
	parser := &MockFullParser{
		ValidateError: assert.AnError,
		ParseError:    assert.AnError,
	}

	// Setup expectations for error scenarios
	parser.On("ValidateFormat", "test.xml").Return(false, assert.AnError)
	parser.On("Parse", mock.Anything, mock.Anything).Return([]models.Transaction{}, assert.AnError)

	_, err := parser.ValidateFormat("test.xml")
	assert.Error(t, err)

	_, err = parser.Parse(context.Background(), nil)
	assert.Error(t, err)

	parser.AssertExpectations(t)
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
