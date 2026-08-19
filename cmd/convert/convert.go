// Package convert provides a format-detecting conversion command.
package convert

import (
	"context"
	"fmt"
	"os"
	"strings"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"

	"github.com/spf13/cobra"
)

// Cmd represents the convert command.
var Cmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert a statement to CSV",
	Long: `Convert a statement, or a directory of statements, to CSV.

Each file is offered to every parser in turn, and the first one that
recognizes it performs the conversion. This works for every supported format:
CAMT.053 XML, PDF statements, and the Revolut, Revolut Crypto, Revolut
Investment, Selma and Visa Debit CSV exports.

Use --from to pin a specific format instead of auto-detecting it — required
when a file's format cannot be told apart from another, or when detection
guesses wrong.

When the input is a directory, every file in it is read and their
transactions are merged into a single, date-sorted output CSV, plus a
.manifest.json run report beside it.`,
	Run: runConvert,
}

func init() { common.RegisterConvertFlags(Cmd) }

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
	recursive, _ := cmd.Flags().GetBool("recursive")
	from, _ := cmd.Flags().GetString("from")

	resolve, err := common.ResolverFor(appContainer, from)
	if err != nil {
		logger.Fatalf("%v", err)
	}

	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		logger.Fatalf("Error accessing input path: %v", err)
	}

	if fileInfo.IsDir() {
		if outputPath == "" {
			logger.Fatal("--output flag is required when processing a folder. Use -o or --output to specify the output file.")
		}
		convertDirectory(ctx, appContainer, resolve, inputPath, outputPath, logger, format, recursive)
		return
	}

	p, err := resolve(inputPath)
	if err != nil {
		types := container.DetectionOrder()
		names := make([]string, len(types))
		for i, t := range types {
			names[i] = string(t)
		}
		logger.Fatalf("Could not determine the format of %s. Supported formats: %s. "+
			"Use --from to specify it explicitly.", inputPath, strings.Join(names, ", "))
	}

	common.ProcessFile(ctx, p, inputPath, outputPath, root.SharedFlags.Validate, root.Log, appContainer, format)
	root.Log.Info("Conversion completed successfully!")
}

// convertDirectory merges every file under inputDir into the single CSV named
// by outputPath (generating a name inside it when outputPath is an existing
// directory), plus a run report beside that CSV.
func convertDirectory(ctx context.Context, appContainer *container.Container, resolve batch.ParserResolver,
	inputPath, outputPath string, logger logging.Logger, format string, recursive bool) {

	outputFile, err := common.ResolveOutputFile(inputPath, outputPath)
	if err != nil {
		logger.Fatalf("%v", err)
	}

	outFormatter, err := appContainer.GetFormatterRegistry().Get(format)
	if err != nil {
		logger.Fatalf("Invalid output format '%s': valid formats are standard, icompta, jumpsoft", format)
	}

	processor := batch.NewBatchProcessor(resolve, logger, outFormatter, recursive)

	manifest, err := processor.ProcessDirectory(ctx, inputPath, outputFile)
	if err != nil {
		logger.WithError(err).Fatal("Batch conversion failed")
		return
	}

	manifestPath := batch.ManifestPathFor(outputFile)

	logger.Info(fmt.Sprintf("Batch complete: %d/%d files succeeded",
		manifest.SuccessCount, manifest.TotalFiles))

	if manifest.FailureCount > 0 {
		logger.Warn(fmt.Sprintf("%d files failed (see %s for details)",
			manifest.FailureCount, manifestPath))
	}

	if manifest.ExitCode() != 0 {
		root.SetExitCode(manifest.ExitCode())
	}
}
