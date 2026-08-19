package batch

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/formatter"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFullParser implements parser.FullParser for testing
type mockFullParser struct {
	validateFunc func(filePath string) (bool, error)
	parseFunc    func(ctx context.Context, r io.Reader) ([]models.Transaction, error)
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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

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
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

	// Execute
	result := processor.processFile(context.Background(), testFile, outputDir)

	// Assert
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "write_error")
	assert.Equal(t, 0, result.RecordCount)
}

func TestBatchProcessorWithFormatter(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Create a test file
	testFile := filepath.Join(inputDir, "test.xml")
	require.NoError(t, os.WriteFile(testFile, []byte("test data"), 0644)) // #nosec G306 -- test file

	// Setup mock parser
	mockParser := newMockParser()
	mockParser.validateFunc = func(filePath string) (bool, error) {
		return true, nil
	}
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		// Create a test transaction with known data
		tx, _ := models.NewTransactionBuilder().
			WithDatetime(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)).
			WithAmount(decimal.NewFromFloat(100.50), "CHF").
			WithDescription("Test payment").
			Build()
		return []models.Transaction{tx}, nil
	}

	logger := logging.NewLogrusAdapter("error", "text")

	// Test with IComptaFormatter
	icomptaFormatter := &testIComptaFormatter{}
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, icomptaFormatter, false)

	// Execute
	ctx := context.Background()
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, 1, manifest.SuccessCount)
	assert.Equal(t, 0, manifest.FailureCount)

	// Verify output CSV was created
	csvPath := filepath.Join(outputDir, "test.csv")
	assert.FileExists(t, csvPath)

	// Read CSV file and verify delimiter and column count
	content, err := os.ReadFile(csvPath)
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")
	require.Greater(t, len(lines), 1, "CSV should have at least header and one data row")

	// Verify semicolon delimiter is used
	headerLine := lines[0]
	assert.Contains(t, headerLine, ";", "CSV should use semicolon delimiter")

	// Verify 10 columns (iCompta format)
	headerFields := strings.Split(headerLine, ";")
	assert.Equal(t, 10, len(headerFields), "iCompta format should have 10 columns")

	// Verify data row uses semicolon
	if len(lines) > 1 && lines[1] != "" {
		dataFields := strings.Split(lines[1], ";")
		assert.Equal(t, 10, len(dataFields), "Data row should have 10 fields")
	}
}

// testIComptaFormatter is a minimal test implementation of OutputFormatter
// that mimics iCompta format (semicolon delimiter, 10 columns)
type testIComptaFormatter struct{}

func (f *testIComptaFormatter) Header() []string {
	return []string{
		"Date",
		"Description",
		"Category",
		"Debit",
		"Credit",
		"Currency",
		"Account",
		"Status",
		"Reference",
		"Notes",
	}
}

func (f *testIComptaFormatter) Format(transactions []models.Transaction) ([][]string, error) {
	rows := make([][]string, 0, len(transactions))
	for _, tx := range transactions {
		row := []string{
			tx.Date.Format("02.01.2006"),
			tx.Description,
			tx.Category,
			tx.Debit.String(),
			tx.Credit.String(),
			tx.Currency,
			tx.IBAN,
			tx.Status,
			tx.Reference,
			tx.RemittanceInfo,
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *testIComptaFormatter) Delimiter() rune {
	return ';'
}

// By default only the top level of the input directory is processed.
func TestDiscoverFiles_NonRecursiveByDefault(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "top.csv"), []byte("x"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "nested.csv"), []byte("x"), 0600))

	bp := NewBatchProcessor(PinnedResolver(newMockParser()), logging.NewLogrusAdapter("error", "text"), nil, false)

	files, err := bp.discoverFiles(dir)
	require.NoError(t, err)

	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join(dir, "top.csv"), files[0])
}

// With recursion enabled, nested files are found at any depth.
func TestDiscoverFiles_Recursive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "top.csv"), []byte("x"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub", "deeper"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "nested.csv"), []byte("x"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "deeper", "deep.csv"), []byte("x"), 0600))

	bp := NewBatchProcessor(PinnedResolver(newMockParser()), logging.NewLogrusAdapter("error", "text"), nil, true)

	files, err := bp.discoverFiles(dir)
	require.NoError(t, err)

	require.Len(t, files, 3)
	assert.Contains(t, files, filepath.Join(dir, "top.csv"))
	assert.Contains(t, files, filepath.Join(dir, "sub", "nested.csv"))
	assert.Contains(t, files, filepath.Join(dir, "sub", "deeper", "deep.csv"))
}

// Hidden directories must be skipped even when recursing: .git and a previous
// run's output are never inputs.
func TestDiscoverFiles_RecursiveSkipsHiddenDirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "visible.csv"), []byte("x"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".hidden"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden", "secret.csv"), []byte("x"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".manifest.json"), []byte("{}"), 0600))

	bp := NewBatchProcessor(PinnedResolver(newMockParser()), logging.NewLogrusAdapter("error", "text"), nil, true)

	files, err := bp.discoverFiles(dir)
	require.NoError(t, err)

	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join(dir, "visible.csv"), files[0])
}

// Two statements with the same basename in different subdirectories must not
// overwrite each other. Before the output path mirrored the input tree, the
// second conversion silently replaced the first while the manifest reported
// both as successful.
func TestProcessDirectory_RecursiveDoesNotOverwriteSameBasename(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(inputDir, "jan"), 0750))
	require.NoError(t, os.MkdirAll(filepath.Join(inputDir, "feb"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "jan", "statement.csv"), []byte("x"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "feb", "statement.csv"), []byte("x"), 0600))

	mockParser := newMockParser()
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return createTestTransactions(3), nil
	}

	processor := NewBatchProcessor(PinnedResolver(mockParser), logging.NewLogrusAdapter("error", "text"), nil, true)

	manifest, err := processor.ProcessDirectory(context.Background(), inputDir, outputDir)
	require.NoError(t, err)

	assert.Equal(t, 2, manifest.SuccessCount)
	assert.FileExists(t, filepath.Join(outputDir, "jan", "statement.csv"))
	assert.FileExists(t, filepath.Join(outputDir, "feb", "statement.csv"))
}

// Inputs in one directory that differ only by extension would map to the same
// output name. The source extension is folded in rather than one result
// replacing the other.
func TestProcessDirectory_DisambiguatesSameStemDifferentExtension(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "statement.csv"), []byte("x"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "statement.xml"), []byte("x"), 0600))

	mockParser := newMockParser()
	mockParser.parseFunc = func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
		return createTestTransactions(3), nil
	}

	processor := NewBatchProcessor(PinnedResolver(mockParser), logging.NewLogrusAdapter("error", "text"), nil, false)

	manifest, err := processor.ProcessDirectory(context.Background(), inputDir, outputDir)
	require.NoError(t, err)

	require.Equal(t, 2, manifest.SuccessCount)

	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)

	var produced []string
	for _, e := range entries {
		if e.Name() != ".manifest.json" {
			produced = append(produced, e.Name())
		}
	}
	assert.Len(t, produced, 2, "each input must produce its own output: %v", produced)
}

// A directory that cannot be read must fail the run rather than yield a short
// work list, which would produce a manifest reporting success for everything it
// happened to find.
func TestProcessDirectory_UnreadableSubdirectoryIsAnError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "top.csv"), []byte("x"), 0600))

	locked := filepath.Join(inputDir, "locked")
	require.NoError(t, os.MkdirAll(locked, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(locked, "hidden.csv"), []byte("x"), 0600))
	require.NoError(t, os.Chmod(locked, 0000))
	// Restore permissions so t.TempDir cleanup can remove the tree.
	t.Cleanup(func() { _ = os.Chmod(locked, 0700) }) // #nosec G302 -- test fixture directory, restored so cleanup can run

	processor := NewBatchProcessor(PinnedResolver(newMockParser()), logging.NewLogrusAdapter("error", "text"), nil, true)

	_, err := processor.ProcessDirectory(context.Background(), inputDir, outputDir)

	require.Error(t, err, "an unreadable subdirectory must not be silently skipped")
	assert.Contains(t, err.Error(), "failed to read directory")
}

// The resolver is consulted per file, which is what lets one directory hold
// several formats. A file the resolver rejects is recorded as a failure and
// must not stop the files after it.
func TestProcessDirectory_ResolvesPerFile(t *testing.T) {
	logger := logging.NewLogrusAdapter("error", "text")
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeSample(t, inputDir, "a.csv", "ok")
	writeSample(t, inputDir, "b.csv", "unsupported")
	writeSample(t, inputDir, "c.csv", "ok")

	var asked []string
	resolve := func(filePath string) (parser.FullParser, error) {
		asked = append(asked, filepath.Base(filePath))
		if strings.HasPrefix(filepath.Base(filePath), "b") {
			return nil, ErrNoParser
		}
		p := newMockParser()
		p.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
			return createTestTransactions(1), nil
		}
		return p, nil
	}

	bp := NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputDir)

	require.NoError(t, err)
	assert.Equal(t, []string{"a.csv", "b.csv", "c.csv"}, asked, "every file must be offered to the resolver")
	assert.Equal(t, 2, manifest.SuccessCount)
	assert.Equal(t, 1, manifest.FailureCount)
	assert.Equal(t, 1, manifest.ExitCode(), "partial success")
}

// PinnedResolver is what --from produces: the same parser for every file,
// so a batch the detector misreads can be forced through one parser.
func TestPinnedResolver_AlwaysReturnsSameParser(t *testing.T) {
	p := newMockParser()
	resolve := PinnedResolver(p)

	got1, err1 := resolve("/any/path.csv")
	got2, err2 := resolve("/other/file.xml")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Same(t, p, got1)
	assert.Same(t, p, got2)
}

func writeSample(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}
