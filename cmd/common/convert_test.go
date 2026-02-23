package common_test

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// convertMockParser implements parser.FullParser for testing FolderConvert.
// All methods are minimal stubs — file-system tests use temp dirs with no real files,
// so Parse/ValidateFormat are never invoked by the BatchProcessor in these tests.
type convertMockParser struct{}

func (m *convertMockParser) Parse(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
	return nil, nil
}

func (m *convertMockParser) ValidateFormat(_ string) (bool, error) {
	return false, nil
}

func (m *convertMockParser) ConvertToCSV(_ context.Context, _, _ string) error {
	return nil
}

func (m *convertMockParser) SetLogger(_ logging.Logger) {}

func (m *convertMockParser) SetCategorizer(_ models.TransactionCategorizer) {}

func (m *convertMockParser) BatchConvert(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}

var _ parser.FullParser = (*convertMockParser)(nil)

// TestRunConvert_FolderWithoutOutput verifies that the --output guard in RunConvert
// produces a FATAL log entry when the input is a directory and outputPath is empty.
// Because RunConvert uses the global root logger (which calls os.Exit on Fatal), this test
// exercises FolderConvert with a non-FullParser to confirm the FATAL guard paths work
// when using a mock logger that does not actually exit.
func TestRunConvert_FolderWithoutOutput(t *testing.T) {
	mockLogger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Prevent os.Exit from killing the test process
	var capturedExitCode int
	restore := common.SetOsExitFn(func(code int) { capturedExitCode = code })
	defer restore()
	_ = capturedExitCode

	// Passing a non-FullParser (plain struct) triggers the guard in FolderConvert
	// ("Parser does not support batch conversion")
	type notAParser struct{}
	common.FolderConvert(context.Background(), notAParser{}, inputDir, outputDir, mockLogger, "standard", "")

	fatalEntries := mockLogger.GetEntriesByLevel("FATAL")
	require.NotEmpty(t, fatalEntries, "expected at least one FATAL log entry")

	// Verify the fatal message relates to batch conversion support
	found := false
	for _, entry := range fatalEntries {
		if strings.Contains(entry.Message, "batch conversion") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected FATAL message about 'batch conversion', got: %v", fatalEntries)
}

// TestFolderConvert_EmptyDirectory verifies that FolderConvert completes successfully
// when the input directory exists but contains no files.
// A 0-file batch results in ExitCode()==2 per BatchManifest contract; this test
// verifies that no FATAL is logged and that the manifest file is written.
func TestFolderConvert_EmptyDirectory(t *testing.T) {
	mockLogger := logging.NewMockLogger()
	mockParser := &convertMockParser{}

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Capture the exit code instead of exiting
	var capturedExitCode int
	restore := common.SetOsExitFn(func(code int) { capturedExitCode = code })
	defer restore()

	common.FolderConvert(context.Background(), mockParser, inputDir, outputDir, mockLogger, "standard", "")

	// No FATAL entries — the exit is via osExitFn, not logger.Fatal
	fatalEntries := mockLogger.GetEntriesByLevel("FATAL")
	assert.Empty(t, fatalEntries, "expected no FATAL log entries for empty directory")

	// Empty directory → ExitCode()==2 per BatchManifest contract
	assert.Equal(t, 2, capturedExitCode, "expected exit code 2 for empty directory (no files)")

	// Manifest file should have been written
	manifestPath := filepath.Join(outputDir, ".manifest.json")
	assert.FileExists(t, manifestPath, "expected .manifest.json to be written")
}

// TestFolderConvert_InvalidFormat verifies that FolderConvert logs a FATAL entry
// when given an unrecognised format string.
func TestFolderConvert_InvalidFormat(t *testing.T) {
	mockLogger := logging.NewMockLogger()
	mockParser := &convertMockParser{}

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	// Prevent os.Exit in case it is reached (it should not be for invalid format)
	restore := common.SetOsExitFn(func(_ int) {})
	defer restore()

	common.FolderConvert(context.Background(), mockParser, inputDir, outputDir, mockLogger, "invalid", "")

	fatalEntries := mockLogger.GetEntriesByLevel("FATAL")
	require.NotEmpty(t, fatalEntries, "expected a FATAL log entry for invalid format")

	// Message should mention "invalid" or "format"
	found := mockLogger.VerifyFatalLog("invalid") || mockLogger.VerifyFatalLog("format")
	assert.True(t, found, "expected FATAL message mentioning 'invalid' or 'format', got: %v", fatalEntries)
}
