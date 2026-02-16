package batch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"
)

// BatchProcessor handles standardized batch processing for any parser
type BatchProcessor struct {
	parser parser.FullParser
	logger logging.Logger
}

// NewBatchProcessor creates a new BatchProcessor instance that wraps the provided parser.
// The processor will use the parser for validation, parsing, and CSV writing operations.
func NewBatchProcessor(p parser.FullParser, logger logging.Logger) *BatchProcessor {
	return &BatchProcessor{
		parser: p,
		logger: logger,
	}
}

// ProcessDirectory processes all files in inputDir and writes converted files to outputDir.
// Returns a manifest (never nil) containing results for each file processed.
// Individual file failures are captured in the manifest, not returned as errors.
// An error is returned only for configuration or permission issues with the directories.
func (bp *BatchProcessor) ProcessDirectory(ctx context.Context, inputDir, outputDir string) (*BatchManifest, error) {
	startTime := time.Now()

	// Validate input directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("input directory does not exist: %s", inputDir)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Discover files to process
	files := bp.discoverFiles(inputDir)

	bp.logger.Info("Starting batch processing",
		logging.Field{Key: "input_dir", Value: inputDir},
		logging.Field{Key: "output_dir", Value: outputDir},
		logging.Field{Key: "files_found", Value: len(files)})

	// Initialize manifest
	manifest := &BatchManifest{
		TotalFiles:   len(files),
		SuccessCount: 0,
		FailureCount: 0,
		Results:      make([]BatchResult, 0, len(files)),
		ProcessedAt:  time.Now(),
	}

	// Process each file sequentially
	for _, filePath := range files {
		// Check for cancellation
		select {
		case <-ctx.Done():
			bp.logger.Warn("Batch processing cancelled",
				logging.Field{Key: "processed", Value: len(manifest.Results)},
				logging.Field{Key: "total", Value: manifest.TotalFiles})
			manifest.Duration = time.Since(startTime)
			return manifest, ctx.Err()
		default:
		}

		result := bp.processFile(ctx, filePath, outputDir)
		manifest.Results = append(manifest.Results, result)

		if result.Success {
			manifest.SuccessCount++
		} else {
			manifest.FailureCount++
		}
	}

	// Calculate duration
	manifest.Duration = time.Since(startTime)

	bp.logger.Info("Batch processing completed",
		logging.Field{Key: "total_files", Value: manifest.TotalFiles},
		logging.Field{Key: "success", Value: manifest.SuccessCount},
		logging.Field{Key: "failed", Value: manifest.FailureCount},
		logging.Field{Key: "duration", Value: manifest.Duration.String()})

	// Always write manifest to output directory
	manifestPath := filepath.Join(outputDir, ".manifest.json")
	if err := manifest.WriteManifest(manifestPath); err != nil {
		bp.logger.WithError(err).Warn("Failed to write manifest file",
			logging.Field{Key: "path", Value: manifestPath})
	} else {
		bp.logger.Info("Wrote batch manifest",
			logging.Field{Key: "path", Value: manifestPath})
	}

	return manifest, nil
}

// discoverFiles returns a sorted list of processable files in the given directory.
// Only returns files in the top-level directory (not recursive).
// Skips hidden files (starting with '.') and directories.
func (bp *BatchProcessor) discoverFiles(inputDir string) []string {
	var files []string

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		bp.logger.WithError(err).Error("Failed to read input directory",
			logging.Field{Key: "dir", Value: inputDir})
		return files
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			bp.logger.Debug("Skipping hidden file",
				logging.Field{Key: "file", Value: entry.Name()})
			continue
		}

		filePath := filepath.Join(inputDir, entry.Name())
		files = append(files, filePath)
	}

	// Sort files alphabetically for consistent ordering
	sort.Strings(files)

	return files
}

// processFile processes a single file and returns a BatchResult.
// This method never panics and captures all errors in the returned result.
func (bp *BatchProcessor) processFile(ctx context.Context, filePath, outputDir string) BatchResult {
	fileName := filepath.Base(filePath)

	bp.logger.Info("Processing file",
		logging.Field{Key: "file", Value: fileName})

	result := BatchResult{
		FilePath:    filePath,
		FileName:    fileName,
		Success:     false,
		Error:       "",
		RecordCount: 0,
	}

	// Step 1: Validate format
	isValid, err := bp.parser.ValidateFormat(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("validation_error: %v", err)
		bp.logger.WithError(err).Warn("Validation error",
			logging.Field{Key: "file", Value: fileName})
		return result
	}

	if !isValid {
		result.Error = "validation_failed"
		bp.logger.Warn("Invalid format",
			logging.Field{Key: "file", Value: fileName})
		return result
	}

	// Step 2: Open and parse file
	file, err := os.Open(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("open_error: %v", err)
		bp.logger.WithError(err).Warn("Failed to open file",
			logging.Field{Key: "file", Value: fileName})
		return result
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			bp.logger.WithError(closeErr).Warn("Failed to close file",
				logging.Field{Key: "file", Value: fileName})
		}
	}()

	transactions, err := bp.parser.Parse(ctx, file)
	if err != nil {
		result.Error = err.Error()
		bp.logger.WithError(err).Warn("Parse error",
			logging.Field{Key: "file", Value: fileName})
		return result
	}

	// Step 3: Generate output filename (preserve basename, change extension to .csv)
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	outputFileName := baseName + ".csv"
	outputPath := filepath.Join(outputDir, outputFileName)

	// Step 4: Write CSV
	if err := common.ExportTransactionsToCSVWithLogger(transactions, outputPath, bp.logger); err != nil {
		result.Error = fmt.Sprintf("write_error: %v", err)
		bp.logger.WithError(err).Warn("Failed to write CSV",
			logging.Field{Key: "file", Value: fileName},
			logging.Field{Key: "output", Value: outputFileName})
		return result
	}

	// Success!
	result.Success = true
	result.RecordCount = len(transactions)

	bp.logger.Info("Successfully processed file",
		logging.Field{Key: "file", Value: fileName},
		logging.Field{Key: "records", Value: result.RecordCount},
		logging.Field{Key: "output", Value: outputFileName})

	return result
}
