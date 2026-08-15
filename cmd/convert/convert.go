// Package convert provides a format-detecting conversion command.
package convert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"

	"github.com/spf13/cobra"
)

// Cmd represents the convert command.
var Cmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert a statement to CSV, detecting its format automatically",
	Long: `Convert a statement to CSV without naming its format.

Each parser is asked in turn whether it recognizes the file, and the first one
that does performs the conversion. This works for every supported format:
CAMT.053 XML, PDF statements, and the Revolut, Revolut Crypto, Revolut
Investment, Selma and Visa Debit CSV exports.

When the input is a directory, every file in it is detected and converted
independently, so a directory holding a mix of formats is handled in one pass.

Use the format-specific commands instead when you want a file rejected rather
than guessed at.`,
	Run: runConvert,
}

func init() { common.RegisterFormatFlags(Cmd) }

func runConvert(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()

	inputPath := root.SharedFlags.Input
	outputPath := root.SharedFlags.Output

	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "" {
		format = appContainer.GetConfig().Output.Format
	}

	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		logger.Fatalf("Error accessing input path: %v", err)
	}

	if fileInfo.IsDir() {
		if outputPath == "" {
			logger.Fatal("--output flag is required when processing a folder. Use -o or --output to specify the output directory.")
		}
		convertDirectory(ctx, appContainer, inputPath, outputPath, logger, format)
		return
	}

	p, parserType, err := appContainer.DetectParser(inputPath)
	if err != nil {
		logger.Fatalf("Could not determine the format of %s. Supported formats: %s. "+
			"Use a format-specific command if you know which one it is.",
			inputPath, strings.Join(parserTypeNames(), ", "))
	}

	logger.Info("Detected input format",
		logging.Field{Key: "file", Value: inputPath},
		logging.Field{Key: "format", Value: string(parserType)})

	common.ProcessFile(ctx, p, inputPath, outputPath, root.SharedFlags.Validate, root.Log, appContainer, format)
	root.Log.Info("Conversion completed successfully!")
}

// convertDirectory detects and converts each file in inputDir independently, so
// a directory containing several different statement formats converts in one
// pass. Unrecognized files are skipped with a warning rather than aborting.
func convertDirectory(ctx context.Context, appContainer *container.Container,
	inputDir, outputDir string, logger logging.Logger, format string) {

	// Outputs are named after their inputs, so writing into the input directory
	// would overwrite the statements being read.
	if sameDirectory(inputDir, outputDir) {
		logger.Fatal("--output must differ from --input: converted files are named after their sources and would overwrite them")
	}

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		logger.Fatalf("Error reading input directory: %v", err)
	}

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		logger.Fatalf("Error creating output directory: %v", err)
	}

	var converted, skipped, failed int

	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if err := ctx.Err(); err != nil {
			logger.Warn("Conversion cancelled",
				logging.Field{Key: "converted", Value: converted})
			return
		}

		inputFile := filepath.Join(inputDir, entry.Name())

		p, parserType, err := appContainer.DetectParser(inputFile)
		if err != nil {
			logger.Warn("Skipping file of unrecognized format",
				logging.Field{Key: "file", Value: entry.Name()})
			skipped++
			continue
		}

		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		outputFile := filepath.Join(outputDir, base+".csv")

		if err := common.ProcessFileWithErrorFormatted(ctx, p, inputFile, outputFile,
			root.SharedFlags.Validate, logger, appContainer, format); err != nil {
			logger.WithError(err).Warn("Failed to convert file",
				logging.Field{Key: "file", Value: entry.Name()})
			failed++
			continue
		}

		logger.Info("Converted file",
			logging.Field{Key: "file", Value: entry.Name()},
			logging.Field{Key: "format", Value: string(parserType)})
		converted++
	}

	logger.Info(fmt.Sprintf("Convert complete: %d converted, %d skipped, %d failed",
		converted, skipped, failed))
}

// sameDirectory reports whether two paths refer to the same directory,
// resolving them so that "." and "./out/.." style arguments are compared
// correctly. Paths that cannot be resolved are treated as different: the
// output directory may not exist yet.
func sameDirectory(a, b string) bool {
	resolvedA, err := filepath.Abs(a)
	if err != nil {
		return false
	}
	resolvedB, err := filepath.Abs(b)
	if err != nil {
		return false
	}
	return filepath.Clean(resolvedA) == filepath.Clean(resolvedB)
}

// parserTypeNames lists the detectable formats for error messages.
func parserTypeNames() []string {
	types := container.DetectionOrder()
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	return names
}
