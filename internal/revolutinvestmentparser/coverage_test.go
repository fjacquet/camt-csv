package revolutinvestmentparser

import (
	"context"
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

// --- ConvertToCSV tests ---

func TestConvertToCSV_ValidFile(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputFile := writeTestFile(t, tmpDir, "input.csv", validCSVContent())
	outputFile := filepath.Join(tmpDir, "output.csv")

	err := adapter.ConvertToCSV(context.Background(), inputFile, outputFile)
	require.NoError(t, err)

	info, err := os.Stat(outputFile)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "output CSV should not be empty")
}

func TestConvertToCSV_NonexistentInput(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	outputFile := filepath.Join(tmpDir, "output.csv")

	err := adapter.ConvertToCSV(context.Background(), "/nonexistent/path/file.csv", outputFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error opening input file")
}

func TestConvertToCSV_InvalidContent(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputFile := writeTestFile(t, tmpDir, "bad.csv", "not,a,valid,revolut,investment,csv,file,content")
	outputFile := filepath.Join(tmpDir, "output.csv")

	err := adapter.ConvertToCSV(context.Background(), inputFile, outputFile)
	require.Error(t, err)
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

func TestBatchConvert_ValidDirectory(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	writeTestFile(t, inputDir, "file1.csv", validCSVContent())
	writeTestFile(t, inputDir, "file2.csv", validCSVContent())

	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Verify output files exist
	_, err = os.Stat(filepath.Join(outputDir, "file1.csv"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "file2.csv"))
	assert.NoError(t, err)
}

func TestBatchConvert_EmptyDirectory(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestBatchConvert_NonexistentDirectory(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	outputDir := filepath.Join(tmpDir, "output")

	count, err := adapter.BatchConvert(context.Background(), "/nonexistent/dir", outputDir)
	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "failed to read input directory")
}

func TestBatchConvert_SkipsNonCSVFiles(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	writeTestFile(t, inputDir, "valid.csv", validCSVContent())
	writeTestFile(t, inputDir, "readme.txt", "not a csv file")
	writeTestFile(t, inputDir, "data.json", `{"key": "value"}`)

	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should only convert the CSV file")
}

func TestBatchConvert_SkipsInvalidCSVFiles(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	writeTestFile(t, inputDir, "valid.csv", validCSVContent())
	writeTestFile(t, inputDir, "empty.csv", "")

	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should skip the empty/invalid CSV file")
}

func TestBatchConvert_SkipsSubdirectories(t *testing.T) {
	adapter := newTestAdapter()
	tmpDir := t.TempDir()

	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))
	require.NoError(t, os.MkdirAll(filepath.Join(inputDir, "subdir"), 0750))

	writeTestFile(t, inputDir, "valid.csv", validCSVContent())

	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
