// Package batch provides functionality for batch processing and aggregation of financial files
package batch

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// BatchResult represents the result of processing a single file
type BatchResult struct {
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	Success     bool   `json:"success"`
	Error       string `json:"error"`        // Only populated if Success=false
	RecordCount int    `json:"record_count"` // Number of transactions extracted
}

// BatchManifest aggregates results from a batch operation
type BatchManifest struct {
	TotalFiles   int           `json:"total_files"`
	SuccessCount int           `json:"success_count"`
	FailureCount int           `json:"failure_count"`
	Results      []BatchResult `json:"results"`
	Duration     time.Duration `json:"duration"`
	ProcessedAt  time.Time     `json:"processed_at"`
}

// ExitCode returns the exit code based on batch processing results.
// Returns 0 if all files succeeded, 2 if all files failed or no files processed, 1 if partial success.
func (m *BatchManifest) ExitCode() int {
	// Treat no files as failure
	if m.TotalFiles == 0 {
		return 2
	}
	if m.FailureCount == 0 {
		return 0 // All success
	}
	if m.SuccessCount == 0 {
		return 2 // All failed
	}
	return 1 // Partial success
}

// WriteManifest serializes the manifest to JSON and writes it to the specified file path.
// The JSON is formatted with indentation for human readability.
func (m *BatchManifest) WriteManifest(filePath string) error {
	// Marshal with indentation for readability
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write to file with appropriate permissions
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return nil
}

// Summary returns a human-readable summary of the batch processing results.
// Format: "X/Y files succeeded"
func (m *BatchManifest) Summary() string {
	return fmt.Sprintf("%d/%d files succeeded", m.SuccessCount, m.TotalFiles)
}
