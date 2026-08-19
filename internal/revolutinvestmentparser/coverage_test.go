package revolutinvestmentparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validCSVContent returns a valid Revolut investment CSV string for test fixtures.
func validCSVContent() string {
	return `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2025-05-30T10:31:02.786456Z,,CASH TOP-UP,,,€454,EUR,1.0722
2025-05-30T10:31:05.452Z,2B7K,BUY - MARKET,39.81059277,€11.40,€454,EUR,1.0722`
}

// writeTestFile creates a file with the given content inside dir and returns its path.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0600)
	require.NoError(t, err)
	return path
}

func newTestAdapter() *Adapter {
	logger := logging.NewLogrusAdapter("info", "text")
	return NewAdapter(logger)
}

// --- ValidateFormat tests ---

func TestValidateFormat_ValidCSV(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputFile := writeTestFile(t, tmpDir, "valid.csv", validCSVContent())

	valid, err := adapter.ValidateFormat(inputFile)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestValidateFormat_NonexistentFile(t *testing.T) {
	adapter := newTestAdapter()

	valid, err := adapter.ValidateFormat("/nonexistent/path/file.csv")
	require.Error(t, err)
	assert.False(t, valid)
}

func TestValidateFormat_EmptyFile(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputFile := writeTestFile(t, tmpDir, "empty.csv", "")

	valid, err := adapter.ValidateFormat(inputFile)
	require.NoError(t, err)
	assert.False(t, valid, "empty file should not validate as valid CSV")
}

// --- BatchConvert tests ---
