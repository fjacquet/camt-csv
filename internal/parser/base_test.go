package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBaseParser_WriteToCSV(t *testing.T) {
	t.Run("writes transactions to CSV successfully", func(t *testing.T) {
		// Create temporary directory
		tempDir := t.TempDir()
		csvFile := filepath.Join(tempDir, "test_output.csv")
		
		// Create mock logger and base parser
		mockLog := &mockLogger{}
		baseParser := NewBaseParser(mockLog)
		
		// Create test transactions
		transactions := []models.Transaction{
			{
				EntryReference: "test-1",
				Date:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				ValueDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Amount:        decimal.NewFromFloat(100.50),
				Currency:      "CHF",
				Description:   "Test transaction 1",
				CreditDebit:   models.TransactionTypeCredit,
			},
			{
				EntryReference: "test-2",
				Date:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				ValueDate:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				Amount:        decimal.NewFromFloat(-50.25),
				Currency:      "CHF",
				Description:   "Test transaction 2",
				CreditDebit:   models.TransactionTypeDebit,
			},
		}
		
		// Write to CSV
		err := baseParser.WriteToCSV(transactions, csvFile)
		
		// Verify no error
		require.NoError(t, err)
		
		// Verify file was created
		assert.FileExists(t, csvFile)
		
		// Verify logger was called
		assert.Contains(t, mockLog.messages, "INFO: Writing transactions to CSV using common writer")
		
		// Verify file content (basic check)
		content, err := os.ReadFile(csvFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test-1")
		assert.Contains(t, string(content), "test-2")
		assert.Contains(t, string(content), "Test transaction 1")
		assert.Contains(t, string(content), "Test transaction 2")
	})
	
	t.Run("handles nil transactions", func(t *testing.T) {
		tempDir := t.TempDir()
		csvFile := filepath.Join(tempDir, "test_output.csv")
		
		mockLog := &mockLogger{}
		baseParser := NewBaseParser(mockLog)
		
		err := baseParser.WriteToCSV(nil, csvFile)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot write nil transactions to CSV")
	})
	
	t.Run("handles empty transactions slice", func(t *testing.T) {
		tempDir := t.TempDir()
		csvFile := filepath.Join(tempDir, "test_output.csv")
		
		mockLog := &mockLogger{}
		baseParser := NewBaseParser(mockLog)
		
		transactions := []models.Transaction{}
		
		err := baseParser.WriteToCSV(transactions, csvFile)
		
		require.NoError(t, err)
		assert.FileExists(t, csvFile)
	})
}

func TestBaseParser_InterfaceCompliance(t *testing.T) {
	t.Run("implements LoggerConfigurable interface", func(t *testing.T) {
		var _ LoggerConfigurable = &BaseParser{}
	})
}