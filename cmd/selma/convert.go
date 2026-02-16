// Package selma handles Selma statement conversion commands
package selma

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"

	"github.com/spf13/cobra"
)

// Cmd represents the selma command
var Cmd = &cobra.Command{
	Use:   "selma",
	Short: "Convert Selma CSV to CSV",
	Long:  `Convert Selma CSV statements to CSV format.`,
	Run:   selmaFunc,
}

func init() {
	Cmd.Flags().StringP("format", "f", "standard",
		"Output format: standard (35-column CSV) or icompta (iCompta-compatible)")
	Cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
}

func selmaFunc(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()
	root.Log.Info("Selma convert command called")

	inputPath := root.SharedFlags.Input
	outputPath := root.SharedFlags.Output

	logger.Infof("Input: %s", inputPath)
	logger.Infof("Output: %s", outputPath)

	// Get format flags
	format, _ := cmd.Flags().GetString("format")
	dateFormat, _ := cmd.Flags().GetString("date-format")

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container
	p, err := appContainer.GetParser(container.Selma)
	if err != nil {
		logger.Fatalf("Error getting Selma parser: %v", err)
	}

	// Check if input is directory or file
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		logger.Fatalf("Error accessing input path: %v", err)
	}

	if fileInfo.IsDir() {
		// Directory mode - batch conversion
		batchConvert(ctx, p, inputPath, outputPath, logger)
	} else {
		// File mode - single file conversion
		common.ProcessFile(ctx, p, inputPath, outputPath, root.SharedFlags.Validate, root.Log, appContainer, format, dateFormat)
		root.Log.Info("Selma to CSV conversion completed successfully!")
	}
}

// batchConvert processes all files in a directory using BatchConvert
func batchConvert(ctx context.Context, p interface{}, inputDir, outputDir string, logger logging.Logger) {
	// Cast to BatchConverter interface
	batchConverter, ok := p.(interface {
		BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error)
	})
	if !ok {
		logger.Error("Parser does not support batch conversion")
		os.Exit(1)
	}

	count, err := batchConverter.BatchConvert(ctx, inputDir, outputDir)
	if err != nil {
		logger.WithError(err).Error("Batch conversion failed")
		os.Exit(1)
	}

	// Load manifest to get exit code
	manifestPath := filepath.Join(outputDir, ".manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		logger.WithError(err).Warn("Could not read manifest")
		// Fallback: exit 0 if count > 0, else exit 1
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

	// Log summary
	logger.Info(fmt.Sprintf("Batch complete: %d/%d files succeeded", manifest.SuccessCount, manifest.TotalFiles))

	if manifest.FailureCount > 0 {
		logger.Warn(fmt.Sprintf("%d files failed (see %s for details)", manifest.FailureCount, manifestPath))
	}

	// Exit with semantic code
	if manifest.ExitCode() != 0 {
		os.Exit(manifest.ExitCode())
	}
}
