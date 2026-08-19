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
	Account     string `json:"account"`      // Account this file's rows were written to
}

// AccountSummary names one CSV a batch wrote and the account it holds.
//
// A directory of bank downloads covers several accounts, each written to its
// own file, so "where did my rows go" is no longer answerable from the output
// path alone — this section of the manifest is the run's only record of it.
type AccountSummary struct {
	Account          string `json:"account"`
	OutputFile       string `json:"output_file"`
	TransactionCount int    `json:"transaction_count"`
}

// BatchManifest aggregates results from a batch operation
type BatchManifest struct {
	TotalFiles   int `json:"total_files"`
	SuccessCount int `json:"success_count"`
	FailureCount int `json:"failure_count"`
	// Transactions parsed across every account, whether or not each
	// account's CSV was written: a run whose write failed still reports what
	// it read, and ExitCode's zero-transaction check stays a statement about
	// parsing. Per-CSV counts are in Accounts.
	TransactionCount int              `json:"transaction_count"`
	Accounts         []AccountSummary `json:"accounts"` // One entry per CSV actually written, ordered by account
	Results          []BatchResult    `json:"results"`
	Duration         time.Duration    `json:"duration"`
	ProcessedAt      time.Time        `json:"processed_at"`
}

// ExitCode returns the exit code based on batch processing results.
// Returns 0 if all files succeeded and transactions were produced, 2 if all
// files failed, no files were processed, or every file that "succeeded"
// still produced zero transactions (e.g. the wrong parser was pinned via
// --from and validated but extracted nothing), 1 if partial success.
func (m *BatchManifest) ExitCode() int {
	// Treat no files as failure
	if m.TotalFiles == 0 {
		return 2
	}
	// A run that read files and converted nothing is materially a failed
	// run, even though every individual parse reported success: a manifest
	// summary of "3/3 files succeeded" with no output on disk is a silent
	// success on a failed conversion.
	if m.SuccessCount > 0 && m.TransactionCount == 0 {
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
