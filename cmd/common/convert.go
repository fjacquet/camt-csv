package common

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/formatter"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"

	"github.com/spf13/cobra"
)

// RunConvert is the shared handler for all convert commands.
// It handles: get logger, get container, get parser, stat input, branch to batch or single-file.
// When input is a directory:
//   - If --output is not set, it logs a fatal error and exits.
//   - If --output is set, it delegates to FolderConvert (modern BatchProcessor path).
func RunConvert(cmd *cobra.Command, _ []string, parserType container.ParserType, name string) {
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()
	root.Log.Info(name + " convert command called")

	inputPath := root.SharedFlags.Input
	outputPath := root.SharedFlags.Output

	logger.Infof("Input: %s", inputPath)
	logger.Infof("Output: %s", outputPath)

	format, _ := cmd.Flags().GetString("format")
	dateFormat, _ := cmd.Flags().GetString("date-format")

	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	p, err := appContainer.GetParser(parserType)
	if err != nil {
		logger.Fatalf("Error getting %s parser: %v", name, err)
	}

	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		logger.Fatalf("Error accessing input path: %v", err)
	}

	if fileInfo.IsDir() {
		if outputPath == "" {
			logger.Fatal("--output flag is required when processing a folder. Use -o or --output to specify the output directory.")
		}
		FolderConvert(ctx, p, inputPath, outputPath, logger, format, dateFormat)
	} else {
		ProcessFile(ctx, p, inputPath, outputPath, root.SharedFlags.Validate, root.Log, appContainer, format, dateFormat)
		root.Log.Info(name + " to CSV conversion completed successfully!")
	}
}

// FolderConvert processes all files in a directory using the modern BatchProcessor with formatter support.
// It replaces the legacy BatchConvertLegacy path for CAMT, debit, selma, and revolut-investment parsers
// when called from RunConvert.
//
// Parameters:
//   - ctx: context for cancellation
//   - p: parser (must implement parser.FullParser)
//   - inputDir: path to directory containing input files
//   - outputDir: path to output directory (will be created if absent)
//   - logger: structured logger
//   - format: output format name ("standard" or "icompta")
//   - dateFormat: date format string (reserved for future use)
func FolderConvert(ctx context.Context, p interface{}, inputDir, outputDir string, logger logging.Logger, format string, _ string) {
	// Resolve formatter
	formatterReg := formatter.NewFormatterRegistry()
	outFormatter, err := formatterReg.Get(format)
	if err != nil {
		logger.Fatalf("Invalid output format '%s': valid formats are standard, icompta", format)
		return // unreachable in production (logger.Fatal exits), but enables testing with mock logger
	}

	// Assert parser to FullParser
	fullParser, ok := p.(parser.FullParser)
	if !ok {
		logger.Fatal("Parser does not support batch conversion")
		return // unreachable in production, but enables testing with mock logger
	}

	// Create and run the batch processor
	processor := batch.NewBatchProcessor(fullParser, logger, outFormatter)

	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
	if err != nil {
		logger.WithError(err).Fatal("Batch conversion failed")
		return
	}

	// Write manifest (processor already writes it, but we refresh for the log message)
	manifestPath := filepath.Join(outputDir, ".manifest.json")
	if err := manifest.WriteManifest(manifestPath); err != nil {
		logger.WithError(err).Warn("Failed to write manifest")
	}

	logger.Info(fmt.Sprintf("Batch complete: %d/%d files succeeded",
		manifest.SuccessCount, manifest.TotalFiles))

	if manifest.FailureCount > 0 {
		logger.Warn(fmt.Sprintf("%d files failed (see %s for details)",
			manifest.FailureCount, manifestPath))
	}

	if manifest.ExitCode() != 0 {
		os.Exit(manifest.ExitCode())
	}
}

// BatchConvertLegacy processes all files in a directory using the BatchConvert interface
// with manifest-based exit codes. Used by CAMT, Selma, Debit, and RevolutInvestment.
//
// Deprecated: Use FolderConvert instead. This function is retained for compatibility
// and will be removed in Phase 13 with the full batch cleanup.
func BatchConvertLegacy(ctx context.Context, p interface{}, inputDir, outputDir string, logger logging.Logger) {
	batchConverter, ok := p.(interface {
		BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error)
	})
	if !ok {
		logger.Fatal("Parser does not support batch conversion")
	}

	count, err := batchConverter.BatchConvert(ctx, inputDir, outputDir)
	if err != nil {
		logger.WithError(err).Error("Batch conversion failed")
		os.Exit(1)
	}

	manifestPath := filepath.Join(outputDir, ".manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		logger.WithError(err).Warn("Could not read manifest")
		if count == 0 {
			os.Exit(1)
		}
		return
	}

	var manifest batch.BatchManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		logger.WithError(err).Warn("Could not parse manifest")
		if count == 0 {
			os.Exit(1)
		}
		return
	}

	logger.Info(fmt.Sprintf("Batch complete: %d/%d files succeeded",
		manifest.SuccessCount, manifest.TotalFiles))

	if manifest.FailureCount > 0 {
		logger.Warn(fmt.Sprintf("%d files failed (see %s for details)",
			manifest.FailureCount, manifestPath))
	}

	if manifest.ExitCode() != 0 {
		os.Exit(manifest.ExitCode())
	}
}
