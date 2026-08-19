package common

import (
	"context"
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

// exitFn records the process exit code requested by a batch run. It defers the
// actual os.Exit to main so that the root PersistentPostRun hook still runs and
// saves the creditor/debitor mappings. Replaced in tests.
var exitFn = root.SetExitCode

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
	recursive, _ := cmd.Flags().GetBool("recursive")

	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	if format == "" {
		format = appContainer.GetConfig().Output.Format
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
		FolderConvert(ctx, p, inputPath, outputPath, logger, format, recursive)
	} else {
		ProcessFile(ctx, p, inputPath, outputPath, root.SharedFlags.Validate, root.Log, appContainer, format)
		root.Log.Info(name + " to CSV conversion completed successfully!")
	}
}

// FolderConvert processes all files in a directory using BatchProcessor with formatter support.
// This is the single directory-processing path for every parser.
//
// Parameters:
//   - ctx: context for cancellation
//   - p: parser (must implement parser.FullParser)
//   - inputDir: path to directory containing input files
//   - outputDir: path to output directory (will be created if absent)
//   - logger: structured logger
//   - format: output format name ("standard", "icompta" or "jumpsoft")
//   - recursive: also process files in subdirectories of inputDir
func FolderConvert(ctx context.Context, p any, inputDir, outputDir string, logger logging.Logger, format string, recursive bool) {
	// Resolve formatter
	formatterReg := formatter.NewFormatterRegistry()
	outFormatter, err := formatterReg.Get(format)
	if err != nil {
		logger.Fatalf("Invalid output format '%s': valid formats are standard, icompta, jumpsoft", format)
		return // unreachable in production (logger.Fatal exits), but enables testing with mock logger
	}

	// Assert parser to FullParser
	fullParser, ok := p.(parser.FullParser)
	if !ok {
		logger.Fatal("Parser does not support batch conversion")
		return // unreachable in production, but enables testing with mock logger
	}

	// Create and run the batch processor
	processor := batch.NewBatchProcessor(batch.PinnedResolver(fullParser), logger, outFormatter, recursive)

	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
	if err != nil {
		logger.WithError(err).Fatal("Batch conversion failed")
		return
	}

	// ProcessDirectory already wrote the manifest; we only need its path to
	// point the user at it.
	manifestPath := filepath.Join(outputDir, ".manifest.json")

	logger.Info(fmt.Sprintf("Batch complete: %d/%d files succeeded",
		manifest.SuccessCount, manifest.TotalFiles))

	if manifest.FailureCount > 0 {
		logger.Warn(fmt.Sprintf("%d files failed (see %s for details)",
			manifest.FailureCount, manifestPath))
	}

	if manifest.ExitCode() != 0 {
		exitFn(manifest.ExitCode())
	}
}
