package batch

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

// mockFullParser implements parser.FullParser for testing
type mockFullParser struct {
	validateFunc func(filePath string) (bool, error)
	parseFunc    func(ctx context.Context, r io.Reader) ([]models.Transaction, error)
	batchFunc    func(ctx context.Context, inputDir, outputDir string) (int, error)
	logger       logging.Logger
	categorizer  models.TransactionCategorizer
	shouldFailOn map[string]string // filename -> error message
	recordCounts map[string]int    // filename -> record count
}

func newMockParser() *mockFullParser {
	return &mockFullParser{
		shouldFailOn: make(map[string]string),
		recordCounts: make(map[string]int),
	}
}

func (m *mockFullParser) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, r)
	}
	// Default: return empty transactions
	return []models.Transaction{}, nil
}

func (m *mockFullParser) ValidateFormat(filePath string) (bool, error) {
	if m.validateFunc != nil {
		return m.validateFunc(filePath)
	}
	// Default: all files are valid
	return true, nil
}

func (m *mockFullParser) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
	return errors.New("not implemented in mock")
}

func (m *mockFullParser) SetLogger(logger logging.Logger) {
	m.logger = logger
}

func (m *mockFullParser) SetCategorizer(categorizer models.TransactionCategorizer) {
	m.categorizer = categorizer
}

func (m *mockFullParser) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
	if m.batchFunc != nil {
		return m.batchFunc(ctx, inputDir, outputDir)
	}
	return 0, errors.New("not implemented in mock")
}

// Helper to create test transactions
func createTestTransactions(count int) []models.Transaction {
	transactions := make([]models.Transaction, count)
	for i := range count {
		tx, _ := models.NewTransactionBuilder().
			WithDatetime(time.Now()).
			WithAmount(decimal.NewFromInt(int64(100+i)), "CHF").
			WithDescription("Test transaction").
			Build()
		transactions[i] = tx
	}
	return transactions
}

func TestProcessDirectory_AllSuccess(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Create 3 test files
	testFiles := []string{"file1.xml", "file2.xml", "file3.xml"}
	for _, name := range testFiles {
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, name), []byte("test data"), 0644)) // #nosec G306 -- test file
	}

	// Setup mock parser
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		return true, nil
	}
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return createTestTransactions(10), nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	ctx := context.Background()
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, 3, manifest.TotalFiles)
	assert.Equal(t, 3, manifest.SuccessCount)
	assert.Equal(t, 0, manifest.FailureCount)
	assert.Equal(t, 0, manifest.ExitCode())
	assert.Len(t, manifest.Results, 3)

	// Verify all results are successful
	for _, result := range manifest.Results {
		assert.True(t, result.Success)
		assert.Equal(t, "", result.Error)
		assert.Equal(t, 10, result.RecordCount)
	}

	// Verify CSV files were created
	for _, name := range testFiles {
		csvName := name[:len(name)-4] + ".csv"
		csvPath := filepath.Join(outputDir, csvName)
		assert.FileExists(t, csvPath)
	}
}

func TestProcessDirectory_PartialSuccess(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Create 3 test files
	testFiles := []string{"valid1.xml", "invalid.xml", "valid2.xml"}
	for _, name := range testFiles {
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, name), []byte("test data"), 0644)) // #nosec G306 -- test file
	}

	// Setup mock parser - second file fails validation
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		if filepath.Base(filePath) == "invalid.xml" {
			return false, nil
		}
		return true, nil
	}
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return createTestTransactions(5), nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	ctx := context.Background()
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, 3, manifest.TotalFiles)
	assert.Equal(t, 2, manifest.SuccessCount)
	assert.Equal(t, 1, manifest.FailureCount)
	assert.Equal(t, 1, manifest.ExitCode()) // Partial success
	assert.Len(t, manifest.Results, 3)

	// Verify failure result (files sorted alphabetically: invalid.xml, valid1.xml, valid2.xml)
	failedResult := manifest.Results[0] // invalid.xml is first file (alphabetical)
	assert.False(t, failedResult.Success)
	assert.Equal(t, "validation_failed", failedResult.Error)
	assert.Equal(t, 0, failedResult.RecordCount)
}

func TestProcessDirectory_AllFailed(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Create 3 test files
	testFiles := []string{"file1.xml", "file2.xml", "file3.xml"}
	for _, name := range testFiles {
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, name), []byte("test data"), 0644)) // #nosec G306 -- test file
	}

	// Setup mock parser - all files fail validation
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		return false, nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	ctx := context.Background()
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, 3, manifest.TotalFiles)
	assert.Equal(t, 0, manifest.SuccessCount)
	assert.Equal(t, 3, manifest.FailureCount)
	assert.Equal(t, 2, manifest.ExitCode()) // All failed
	assert.Len(t, manifest.Results, 3)

	// Verify all results failed
	for _, result := range manifest.Results {
		assert.False(t, result.Success)
		assert.Equal(t, "validation_failed", result.Error)
	}
}

func TestProcessDirectory_EmptyDirectory(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// No files created - empty directory

	mockParser := newMockParser()
	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	ctx := context.Background()
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, 0, manifest.TotalFiles)
	assert.Equal(t, 0, manifest.SuccessCount)
	assert.Equal(t, 0, manifest.FailureCount)
	assert.Equal(t, 2, manifest.ExitCode()) // Empty directory treated as failure
	assert.Len(t, manifest.Results, 0)
}

func TestProcessDirectory_WritesManifest(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Create 1 test file
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "test.xml"), []byte("test data"), 0644)) // #nosec G306 -- test file

	// Setup mock parser
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		return true, nil
	}
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return createTestTransactions(3), nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	ctx := context.Background()
	_, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
	require.NoError(t, err)

	// Assert manifest file was created
	manifestPath := filepath.Join(outputDir, ".manifest.json")
	assert.FileExists(t, manifestPath)

	// Verify manifest content
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest BatchManifest
	err = json.Unmarshal(data, &manifest)
	require.NoError(t, err)

	assert.Equal(t, 1, manifest.TotalFiles)
	assert.Equal(t, 1, manifest.SuccessCount)
}

func TestProcessDirectory_ContinuesOnError(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Create 3 test files
	testFiles := []string{"file1.xml", "file2.xml", "file3.xml"}
	for _, name := range testFiles {
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, name), []byte("test data"), 0644)) // #nosec G306 -- test file
	}

	// Setup mock parser - first file fails, rest succeed
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		if filepath.Base(filePath) == "file1.xml" {
			return false, nil
		}
		return true, nil
	}
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return createTestTransactions(5), nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	ctx := context.Background()
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, 3, manifest.TotalFiles)
	assert.Equal(t, 2, manifest.SuccessCount)
	assert.Equal(t, 1, manifest.FailureCount)

	// Verify that file1 failed but file2 and file3 succeeded
	assert.False(t, manifest.Results[0].Success) // file1.xml failed
	assert.True(t, manifest.Results[1].Success)  // file2.xml succeeded
	assert.True(t, manifest.Results[2].Success)  // file3.xml succeeded
}

func TestProcessFile_ValidationFailure(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	testFile := filepath.Join(inputDir, "test.xml")
	require.NoError(t, os.WriteFile(testFile, []byte("test data"), 0644)) // #nosec G306 -- test file

	// Setup mock parser with validation failure
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		return false, nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	result := processor.processFile(context.Background(), testFile, outputDir)

	// Assert
	assert.False(t, result.Success)
	assert.Equal(t, "validation_failed", result.Error)
	assert.Equal(t, 0, result.RecordCount)
}

func TestProcessFile_ParseError(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	testFile := filepath.Join(inputDir, "test.xml")
	require.NoError(t, os.WriteFile(testFile, []byte("test data"), 0644)) // #nosec G306 -- test file

	// Setup mock parser with parse error
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		return true, nil
	}
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return nil, errors.New("parse error: invalid XML structure")
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	result := processor.processFile(context.Background(), testFile, outputDir)

	// Assert
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "parse error")
	assert.Equal(t, 0, result.RecordCount)
}

func TestProcessFile_WriteError(t *testing.T) {
	// Setup - create read-only output directory to force write error
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))
	require.NoError(t, os.MkdirAll(outputDir, 0444)) // #nosec G301 -- test directory (intentionally read-only)

	testFile := filepath.Join(inputDir, "test.xml")
	require.NoError(t, os.WriteFile(testFile, []byte("test data"), 0644)) // #nosec G306 -- test file

	// Setup mock parser
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		return true, nil
	}
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return createTestTransactions(5), nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(mockParser, logger)

	// Execute
	result := processor.processFile(context.Background(), testFile, outputDir)

	// Assert
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "write_error")
	assert.Equal(t, 0, result.RecordCount)
}
