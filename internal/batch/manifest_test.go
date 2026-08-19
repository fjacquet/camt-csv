package batch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitCode_AllSuccess(t *testing.T) {
	manifest := &BatchManifest{
		TotalFiles:       5,
		SuccessCount:     5,
		FailureCount:     0,
		TransactionCount: 50,
		Results:          []BatchResult{},
	}

	exitCode := manifest.ExitCode()
	assert.Equal(t, 0, exitCode, "All success should return exit code 0")
}

// A run where every file parsed successfully but none produced a single
// transaction (e.g. the wrong parser was pinned via --from: it validates
// the format but extracts nothing) must not report a clean exit. The
// manifest would otherwise say "3/3 files succeeded" with no CSV on disk.
func TestExitCode_SuccessfulFilesButZeroTransactions(t *testing.T) {
	manifest := &BatchManifest{
		TotalFiles:       3,
		SuccessCount:     3,
		FailureCount:     0,
		TransactionCount: 0,
		Results:          []BatchResult{},
	}

	exitCode := manifest.ExitCode()
	assert.Equal(t, 2, exitCode, "successful parses yielding zero transactions must not exit 0")
}

func TestExitCode_AllFailed(t *testing.T) {
	manifest := &BatchManifest{
		TotalFiles:   5,
		SuccessCount: 0,
		FailureCount: 5,
		Results:      []BatchResult{},
	}

	exitCode := manifest.ExitCode()
	assert.Equal(t, 2, exitCode, "All failed should return exit code 2")
}

func TestExitCode_PartialSuccess(t *testing.T) {
	manifest := &BatchManifest{
		TotalFiles:       8,
		SuccessCount:     5,
		FailureCount:     3,
		TransactionCount: 30,
		Results:          []BatchResult{},
	}

	exitCode := manifest.ExitCode()
	assert.Equal(t, 1, exitCode, "Partial success should return exit code 1")
}

func TestExitCode_NoFiles(t *testing.T) {
	manifest := &BatchManifest{
		TotalFiles:   0,
		SuccessCount: 0,
		FailureCount: 0,
		Results:      []BatchResult{},
	}

	exitCode := manifest.ExitCode()
	assert.Equal(t, 2, exitCode, "No files should be treated as failure (exit code 2)")
}

func TestWriteManifest_ValidJSON(t *testing.T) {
	// Create a temporary directory for test output
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, ".manifest.json")

	// Create a manifest with sample data
	now := time.Now()
	manifest := &BatchManifest{
		TotalFiles:       3,
		SuccessCount:     2,
		FailureCount:     1,
		TransactionCount: 30,
		Results: []BatchResult{
			{
				FilePath:    "/path/to/file1.xml",
				FileName:    "file1.xml",
				Success:     true,
				Error:       "",
				RecordCount: 10,
			},
			{
				FilePath:    "/path/to/file2.xml",
				FileName:    "file2.xml",
				Success:     true,
				Error:       "",
				RecordCount: 20,
			},
			{
				FilePath:    "/path/to/file3.xml",
				FileName:    "file3.xml",
				Success:     false,
				Error:       "validation_failed",
				RecordCount: 0,
			},
		},
		Duration:    5 * time.Second,
		ProcessedAt: now,
	}

	// Write manifest to file
	err := manifest.WriteManifest(manifestPath)
	require.NoError(t, err, "WriteManifest should not return an error")

	// Verify file exists
	assert.FileExists(t, manifestPath, "Manifest file should exist")

	// Read and verify JSON structure
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "Should be able to read manifest file")

	// Unmarshal to verify it's valid JSON
	var decoded BatchManifest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err, "Manifest should be valid JSON")

	// Verify the decoded manifest matches the original
	assert.Equal(t, manifest.TotalFiles, decoded.TotalFiles)
	assert.Equal(t, manifest.SuccessCount, decoded.SuccessCount)
	assert.Equal(t, manifest.FailureCount, decoded.FailureCount)
	assert.Equal(t, manifest.TransactionCount, decoded.TransactionCount)
	assert.Len(t, decoded.Results, 3)

	// Verify indentation (check that JSON is pretty-printed)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "\n", "JSON should be indented with newlines")
	assert.Contains(t, jsonStr, "  ", "JSON should be indented with spaces")
}

func TestSummary_FormatsCorrectly(t *testing.T) {
	testCases := []struct {
		name         string
		manifest     *BatchManifest
		expectedText string
	}{
		{
			name: "All success",
			manifest: &BatchManifest{
				TotalFiles:   5,
				SuccessCount: 5,
				FailureCount: 0,
			},
			expectedText: "5/5 files succeeded",
		},
		{
			name: "Partial success",
			manifest: &BatchManifest{
				TotalFiles:   8,
				SuccessCount: 5,
				FailureCount: 3,
			},
			expectedText: "5/8 files succeeded",
		},
		{
			name: "All failed",
			manifest: &BatchManifest{
				TotalFiles:   3,
				SuccessCount: 0,
				FailureCount: 3,
			},
			expectedText: "0/3 files succeeded",
		},
		{
			name: "No files",
			manifest: &BatchManifest{
				TotalFiles:   0,
				SuccessCount: 0,
				FailureCount: 0,
			},
			expectedText: "0/0 files succeeded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := tc.manifest.Summary()
			assert.Equal(t, tc.expectedText, summary)
		})
	}
}
