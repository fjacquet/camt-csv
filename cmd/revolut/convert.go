// Package revolut handles Revolut statement conversion commands
package revolut

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/formatter"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut command
var Cmd = &cobra.Command{
	Use:   "revolut",
	Short: "Convert Revolut CSV to CSV",
	Long:  `Convert Revolut CSV statements to CSV format.`,
	Run:   revolutFunc,
}

func init() { common.RegisterFormatFlags(Cmd) }

func revolutFunc(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()
	root.Log.Info("Revolut convert command called")

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

	p, err := appContainer.GetParser(container.Revolut)
	if err != nil {
		logger.Fatalf("Error getting Revolut parser: %v", err)
	}

	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		logger.Fatalf("Error accessing input path: %v", err)
	}

	if fileInfo.IsDir() && outputPath == "" {
		logger.Fatalf("--output flag is required when processing a folder. Use -o or --output to specify the output directory.")
	}

	if fileInfo.IsDir() {
		batchConvert(ctx, p, inputPath, outputPath, logger, format, dateFormat)
	} else {
		common.ProcessFile(ctx, p, inputPath, outputPath, root.SharedFlags.Validate, root.Log, appContainer, format, dateFormat)
		root.Log.Info("Revolut to CSV conversion completed successfully!")
	}
}

// batchConvert processes all files in a directory using BatchProcessor with formatter
func batchConvert(ctx context.Context, p any, inputDir, outputDir string,
	logger logging.Logger, format string, _ string) {

	fullParser, ok := p.(parser.FullParser)
	if !ok {
		logger.Error("Parser does not support batch conversion")
		os.Exit(1)
	}

	formatterReg := formatter.NewFormatterRegistry()
	outFormatter, err := formatterReg.Get(format)
	if err != nil {
		logger.WithError(err).Error("Invalid format",
			logging.Field{Key: "format", Value: format})
		os.Exit(1)
	}

	processor := batch.NewBatchProcessor(fullParser, logger, outFormatter)

	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputDir)
	if err != nil {
		logger.WithError(err).Error("Batch conversion failed")
		os.Exit(1)
	}

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
