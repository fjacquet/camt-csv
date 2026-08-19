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

// A context cancelled mid-batch must stop ProcessDirectory rather than run to
// completion: the returned error is ctx.Err(), and the manifest records fewer
// results than TotalFiles because later files were never reached. This is the
// only test of ProcessDirectory's ctx.Done() branch (processor.go's
// select/default at the top of the per-file loop) — the equivalent coverage
// that used to live in cmd/pdf's TestConsolidatePDFDirectory_ContextCancellation
// was deleted along with that package and had no replacement here.
func TestProcessDirectory_ContextCancellationStopsBatch(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	require.NoError(t, os.MkdirAll(inputDir, 0750))

	// Three files, sorted alphabetically by discoverFiles: cancellation
	// during the first file's parse must prevent the second and third from
	// ever being processed.
	testFiles := []string{"a.xml", "b.xml", "c.xml"}
	for _, name := range testFiles {
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, name), []byte("test data"), 0644)) // #nosec G306 -- test file
	}

	ctx, cancel := context.WithCancel(context.Background())

	mockParser := newMockParser()
	mockParser.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
		cancel() // Cancel as soon as the first file starts parsing.
		return createTestTransactions(1), nil
	}

	logger := logging.NewLogrusAdapter("error", "text")
	processor := NewBatchProcessor(PinnedResolver(mockParser), logger, nil, false)

	outputFile := filepath.Join(tempDir, "out.csv")

	// Execute
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputFile)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	require.NotNil(t, manifest)
	assert.Equal(t, 3, manifest.TotalFiles)
	assert.Less(t, len(manifest.Results), manifest.TotalFiles,
		"cancellation must stop the batch before every file is processed")
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
	transactions, result := processor.parseFile(context.Background(), testFile)

	// Assert
	assert.False(t, result.Success)
	assert.Equal(t, "validation_failed", result.Error)
	assert.Equal(t, 0, result.RecordCount)
	assert.Nil(t, transactions)
}

func TestProcessFile_ParseError(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
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
	transactions, result := processor.parseFile(context.Background(), testFile)

	// Assert
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "parse error")
	assert.Equal(t, 0, result.RecordCount)
	assert.Nil(t, transactions)
}

// Custom formatters (delimiter, column set) still apply to the single
// consolidated output; this is no longer per-file so the output path is the
// caller-supplied file rather than a name mirrored from the input.
func TestBatchProcessorWithFormatter(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputFile := filepath.Join(tempDir, "output", "out.csv")
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
	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputFile)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, 1, manifest.SuccessCount)
	assert.Equal(t, 0, manifest.FailureCount)

	// Verify the single consolidated CSV was created at the requested path
	assert.FileExists(t, outputFile)

	// Read CSV file and verify delimiter and column count
	content, err := os.ReadFile(outputFile)
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
	outputFile := filepath.Join(t.TempDir(), "out.csv")

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
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

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

// sampleTransaction returns a valid, minimal transaction with a fixed date.
// time.Now() (as createTestTransactions uses) cannot express sort order
// between fixtures, so callers that care about ordering need an explicit date.
func sampleTransaction() models.Transaction {
	tx, err := models.NewTransactionBuilder().
		WithDatetime(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)).
		WithAmount(decimal.NewFromInt(42), "CHF").
		WithDescription("Sample transaction").
		Build()
	if err != nil {
		panic(err)
	}
	return tx
}

// A directory always yields one CSV. Files of different formats are merged into
// it and ordered by date, which is the whole point: drop every statement into a
// folder, get one import file.
func TestProcessDirectory_WritesSingleSortedCSV(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputFile := filepath.Join(t.TempDir(), "releves.csv")

	// Filenames are chosen so lexicographic discovery order ("a-march.csv"
	// before "b-january.csv") is the OPPOSITE of chronological order. If
	// Consolidate were skipped, or silently dropped, the rows would either
	// appear in filename order or January's row would be missing entirely
	// — both of which this test must catch, not paper over.
	writeSample(t, inputDir, "a-march.csv", "x")
	writeSample(t, inputDir, "b-january.csv", "x")

	march := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	january := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	resolve := func(filePath string) (parser.FullParser, error) {
		when := march
		desc := "march"
		if strings.HasPrefix(filepath.Base(filePath), "b-january") {
			when, desc = january, "january"
		}
		p := newMockParser()
		p.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
			tx, err := models.NewTransactionBuilder().
				WithDatetime(when).
				WithValueDatetime(when).
				WithAmount(decimal.NewFromInt(1), "CHF").
				WithDescription(desc).
				WithPartyName("Coop").
				Build()
			if err != nil {
				return nil, err
			}
			return []models.Transaction{tx}, nil
		}
		return p, nil
	}

	bp := NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.Equal(t, 2, manifest.SuccessCount)
	assert.Equal(t, 0, manifest.ExitCode(), "a fully successful run must exit 0")
	assert.FileExists(t, outputFile)

	// Exactly one CSV, and January's row precedes March's inside it. Both
	// indices must actually be found: strings.Index returns -1 for a
	// missing substring, and -1 < <positive> would make this assertion pass
	// even if a row went missing entirely.
	body, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	januaryIdx := strings.Index(string(body), "january")
	marchIdx := strings.Index(string(body), "march")
	require.NotEqual(t, -1, januaryIdx, "january's row must be present in the output")
	require.NotEqual(t, -1, marchIdx, "march's row must be present in the output")
	assert.Less(t, januaryIdx, marchIdx,
		"rows must be ordered by date, not by filename")

	entries, err := os.ReadDir(filepath.Dir(outputFile))
	require.NoError(t, err)
	var csvCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".csv") {
			csvCount++
		}
	}
	assert.Equal(t, 1, csvCount, "a directory input must produce exactly one CSV")
}

// One unreadable file out of many must not discard the rest: the CSV is written
// with what succeeded, the exit code reports partial success, and the manifest
// names the failure so the user knows what to re-run.
func TestProcessDirectory_PartialFailureStillWritesCSV(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	writeSample(t, inputDir, "good.csv", "x")
	writeSample(t, inputDir, "bad.csv", "x")

	resolve := func(filePath string) (parser.FullParser, error) {
		p := newMockParser()
		if strings.HasPrefix(filepath.Base(filePath), "bad") {
			p.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
				return nil, errors.New("corrupt header")
			}
			return p, nil
		}
		p.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
			return []models.Transaction{sampleTransaction()}, nil
		}
		return p, nil
	}

	bp := NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.FileExists(t, outputFile, "the successes must still be written")
	assert.Equal(t, 1, manifest.ExitCode(), "partial success")

	var failed *BatchResult
	for i := range manifest.Results {
		if !manifest.Results[i].Success {
			failed = &manifest.Results[i]
		}
	}
	require.NotNil(t, failed)
	assert.Equal(t, "bad.csv", failed.FileName, "the manifest must name the failure")
}

// The manifest replaces the output extension rather than appending to it, and
// lands beside the CSV: there is no output directory to hold it any more.
func TestProcessDirectory_WritesManifestBesideOutput(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outDir := t.TempDir()
	outputFile := filepath.Join(outDir, "releves.csv")

	writeSample(t, inputDir, "a.csv", "x")
	mockParser := newMockParser()
	mockParser.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
		return []models.Transaction{sampleTransaction()}, nil
	}
	resolve := PinnedResolver(mockParser)

	bp := NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	_, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	manifestPath := filepath.Join(outDir, "releves.manifest.json")
	assert.FileExists(t, manifestPath)
	assert.NoFileExists(t, filepath.Join(outDir, ".manifest.json"), "the old fixed name must be gone")

	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	var decoded BatchManifest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, 1, decoded.TotalFiles)
	assert.Equal(t, 1, decoded.SuccessCount)
	assert.Equal(t, 1, decoded.TransactionCount, "the run's actual transaction count must reach the file")
}

func TestManifestPathFor(t *testing.T) {
	assert.Equal(t, "/out/releves.manifest.json", ManifestPathFor("/out/releves.csv"))
	assert.Equal(t, "/out/noext.manifest.json", ManifestPathFor("/out/noext"))
}

// --recursive is about which files are read, not about the shape of the output:
// a whole tree still lands in the one CSV.
func TestProcessDirectory_RecursiveMergesWholeTree(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	nested := filepath.Join(inputDir, "2024", "q1")
	require.NoError(t, os.MkdirAll(nested, 0750))

	writeSample(t, inputDir, "top.csv", "x")
	writeSample(t, nested, "deep.csv", "x")

	outputFile := filepath.Join(t.TempDir(), "all.csv")
	mockParser := newMockParser()
	mockParser.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
		return []models.Transaction{sampleTransaction()}, nil
	}
	resolve := PinnedResolver(mockParser)

	bp := NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), true)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.Equal(t, 2, manifest.SuccessCount, "the nested file must be read too")
	assert.FileExists(t, outputFile)

	// Checking SuccessCount alone would stay green even if only the last
	// file's transactions made it into merged (e.g. an accidental
	// `merged = transactions` instead of `append`): assert the row count
	// the output actually contains, one per successfully parsed file.
	body, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	assert.Len(t, lines, 3, "header plus one row per successfully parsed file")

	entries, err := os.ReadDir(filepath.Dir(outputFile))
	require.NoError(t, err)
	var csvCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".csv") {
			csvCount++
		}
	}
	assert.Equal(t, 1, csvCount, "recursion must not multiply outputs")
}

// An empty directory is a failed run, not a silent success: exit code 2 tells
// the user nothing was converted.
func TestProcessDirectory_EmptyDirectoryExitsTwo(t *testing.T) {
	logger := logging.NewMockLogger()
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	bp := NewBatchProcessor(PinnedResolver(newMockParser()), logger,
		formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), t.TempDir(), outputFile)

	require.NoError(t, err)
	assert.Equal(t, 2, manifest.ExitCode())
}

// A CSV write failure must not swallow the manifest: the run report naming
// which files succeeded is exactly what a user needs to know what to re-run,
// and losing it precisely when the run failed would defeat ADR-021's
// partial-failure design.
func TestProcessDirectory_WriteErrorIsReturned(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outDir := t.TempDir()
	outputFile := filepath.Join(outDir, "out.csv")
	// Making outDir itself read-only would also block the manifest write
	// (same directory, same permission), which is exactly the bug this test
	// exists to catch — a false pass. Instead, pre-create the CSV path as a
	// directory: os.Create(csvFile) fails on it (EISDIR) while the manifest,
	// a differently-named file in the same otherwise-writable directory,
	// still succeeds.
	require.NoError(t, os.MkdirAll(outputFile, 0750))

	writeSample(t, inputDir, "a.csv", "x")
	mockParser := newMockParser()
	mockParser.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
		return []models.Transaction{sampleTransaction()}, nil
	}

	bp := NewBatchProcessor(PinnedResolver(mockParser), logger, formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.Error(t, err, "a CSV write failure must be returned to the caller")
	require.NotNil(t, manifest, "the manifest must still be returned on a write failure")
	assert.Equal(t, 1, manifest.SuccessCount, "the file itself parsed fine; only the final write failed")

	assert.FileExists(t, ManifestPathFor(outputFile),
		"the run report must be written even when the CSV write fails")
}

// A second run over the same folder must not treat its own previous CSV and
// manifest as inputs to convert. The old manifest name (.manifest.json) was
// hidden and always skipped; the new name is not.
func TestProcessDirectory_RepeatRunIgnoresOwnOutput(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputFile := filepath.Join(inputDir, "releves.csv")

	writeSample(t, inputDir, "a.csv", "x")
	mockParser := newMockParser()
	mockParser.parseFunc = func(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
		return []models.Transaction{sampleTransaction()}, nil
	}
	bp := NewBatchProcessor(PinnedResolver(mockParser), logger, formatter.NewStandardFormatter(), false)

	manifest1, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)
	require.NoError(t, err)
	assert.Equal(t, 1, manifest1.TotalFiles, "the first run must only see the real input file")
	assert.FileExists(t, outputFile)
	assert.FileExists(t, ManifestPathFor(outputFile))

	// Second run over the same folder: releves.csv and releves.manifest.json
	// from the first run are now sitting in inputDir alongside a.csv.
	manifest2, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)
	require.NoError(t, err)
	assert.Equal(t, 1, manifest2.TotalFiles,
		"a repeat run must not discover its own previous CSV or manifest as inputs")
	assert.Equal(t, 0, manifest2.ExitCode(), "a repeat run over unchanged input must still cleanly succeed")
}
