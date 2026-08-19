package batch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/formatter"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"
)

// ParserResolver returns the parser to use for one input file.
//
// It is the seam that lets a single directory hold several formats: the CLI
// supplies either a closure over Container.DetectParser (the default) or one
// returning a parser pinned by --from. Package batch therefore needs no
// knowledge of the container or of how formats are recognized.
type ParserResolver func(filePath string) (parser.FullParser, error)

// ErrNoParser reports that no parser accepts a file. Resolvers return it so a
// batch records the file as a failure and moves on.
var ErrNoParser = errors.New("no parser recognizes this file format")

// PinnedResolver returns a ParserResolver that hands back p for every file.
// This is what --from produces: an escape hatch for a batch the detector reads
// wrongly, not a filter that selects matching files. Files p cannot read fail
// individually and are recorded in the manifest.
func PinnedResolver(p parser.FullParser) ParserResolver {
	return func(string) (parser.FullParser, error) { return p, nil }
}

// BatchProcessor handles standardized batch processing for any parser
type BatchProcessor struct {
	resolve   ParserResolver
	logger    logging.Logger
	formatter formatter.OutputFormatter
	recursive bool
}

// NewBatchProcessor creates a new BatchProcessor instance that resolves a
// parser per file via resolve. The processor uses the resolved parser for
// validation, parsing, and CSV writing operations.
// If fmt is nil, a StandardFormatter will be used by default for backward compatibility.
//
// recursive is fixed here rather than settable afterwards: ProcessDirectory
// reads it, so a later change could alter a run already under way.
func NewBatchProcessor(resolve ParserResolver, logger logging.Logger, fmt formatter.OutputFormatter, recursive bool) *BatchProcessor {
	// Default to StandardFormatter if formatter is nil (backward compatibility)
	if fmt == nil {
		fmt = formatter.NewStandardFormatter()
	}

	return &BatchProcessor{
		resolve:   resolve,
		logger:    logger,
		formatter: fmt,
		recursive: recursive,
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
	files, err := bp.discoverFiles(inputDir)
	if err != nil {
		return nil, err
	}

	// Output names are handed out as files are processed so that collisions can
	// be detected; see outputPathFor.
	claimed := make(map[string]bool, len(files))

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

		result := bp.processFile(ctx, filePath, bp.outputPathFor(inputDir, filePath, outputDir, claimed))
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
// Hidden files and directories (names starting with '.') are always skipped.
// Subdirectories are descended into only when the processor is recursive.
//
// A directory that cannot be read is an error rather than an empty result: a
// truncated work list would otherwise produce a manifest reporting success for
// every file it happened to find, with no sign that others were missed.
func (bp *BatchProcessor) discoverFiles(inputDir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", inputDir, err)
	}

	for _, entry := range entries {
		// Hidden entries are skipped whether they are files or directories:
		// .git and .manifest.json are never inputs.
		if strings.HasPrefix(entry.Name(), ".") {
			bp.logger.Debug("Skipping hidden entry",
				logging.Field{Key: "name", Value: entry.Name()})
			continue
		}

		entryPath := filepath.Join(inputDir, entry.Name())

		if entry.IsDir() {
			if bp.recursive {
				nested, err := bp.discoverFiles(entryPath)
				if err != nil {
					return nil, err
				}
				files = append(files, nested...)
			}
			continue
		}

		files = append(files, entryPath)
	}

	// Sort for consistent, reproducible ordering across runs.
	sort.Strings(files)

	return files, nil
}

// outputPathFor maps an input file to its output path, mirroring the input
// tree under outputDir and replacing the extension with .csv.
//
// Mirroring matters: a flat output directory would send inputDir/jan/statement.xml
// and inputDir/feb/statement.xml to the same file, and the second conversion
// would overwrite the first while the manifest reported both as successful.
//
// Two inputs in the same directory differing only by extension — statement.pdf
// and statement.csv — still collide. claimed tracks the paths already handed
// out so those get the source extension folded into the name instead of
// silently replacing one another.
func (bp *BatchProcessor) outputPathFor(inputDir, filePath, outputDir string, claimed map[string]bool) string {
	relPath, err := filepath.Rel(inputDir, filePath)
	if err != nil {
		// Fall back to the bare name; Rel only fails on inputs we did not walk.
		relPath = filepath.Base(filePath)
	}

	ext := filepath.Ext(relPath)
	candidate := filepath.Join(outputDir, strings.TrimSuffix(relPath, ext)+".csv")

	if claimed[candidate] {
		disambiguated := filepath.Join(outputDir,
			strings.TrimSuffix(relPath, ext)+"-"+strings.TrimPrefix(ext, ".")+".csv")
		bp.logger.Info("Output name already taken, disambiguating with the source extension",
			logging.Field{Key: "file", Value: filePath},
			logging.Field{Key: "output", Value: disambiguated})
		candidate = disambiguated
	}

	claimed[candidate] = true
	return candidate
}

// processFile converts filePath and writes the result to outputPath.
// This method never panics and captures all errors in the returned result.
func (bp *BatchProcessor) processFile(ctx context.Context, filePath, outputPath string) BatchResult {
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

	p, err := bp.resolve(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("format_not_recognized: %v", err)
		bp.logger.WithError(err).Warn("Skipping file of unrecognized format",
			logging.Field{Key: "file", Value: fileName})
		return result
	}

	// Step 1: Validate format
	isValid, err := p.ValidateFormat(filePath)
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
	file, err := os.Open(filePath) // #nosec G304 -- CLI tool requires user-provided file paths
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

	transactions, err := p.Parse(ctx, file)
	if err != nil {
		result.Error = err.Error()
		bp.logger.WithError(err).Warn("Parse error",
			logging.Field{Key: "file", Value: fileName})
		return result
	}

	// Step 3: Write CSV using formatter. Mirroring the input tree can introduce
	// subdirectories that do not exist under the output root yet.
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil {
		result.Error = fmt.Sprintf("output_dir_error: %v", err)
		bp.logger.WithError(err).Warn("Failed to create output directory",
			logging.Field{Key: "file", Value: fileName})
		return result
	}

	delimiter := bp.formatter.Delimiter()
	if err := common.WriteTransactionsToCSVWithFormatter(
		transactions, outputPath, bp.logger, bp.formatter, delimiter); err != nil {
		result.Error = fmt.Sprintf("write_error: %v", err)
		bp.logger.WithError(err).Warn("Failed to write CSV",
			logging.Field{Key: "file", Value: fileName},
			logging.Field{Key: "output", Value: outputPath})
		return result
	}

	// Success!
	result.Success = true
	result.RecordCount = len(transactions)

	bp.logger.Info("Successfully processed file",
		logging.Field{Key: "file", Value: fileName},
		logging.Field{Key: "records", Value: result.RecordCount},
		logging.Field{Key: "output", Value: outputPath})

	return result
}
