// Package convert provides a format-detecting conversion command.
package convert

import (
	"context"
	"fmt"
	"os"
	"strings"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"

	"github.com/spf13/cobra"
)

// Cmd represents the convert command.
var Cmd = &cobra.Command{
	Use:     "convert",
	Short:   "Convert a statement to CSV",
	GroupID: "conversion",
	// The input/output paths are the -i/-o flags, not positional args: a
	// natural but wrong invocation like `convert statement.pdf` would
	// otherwise be silently accepted and fail deep inside with a
	// confusing "Error accessing input path: stat : no such file or
	// directory" (the empty, unset -i) rather than cobra's own clear
	// "unknown command" error.
	Args: cobra.NoArgs,
	Run:  runConvert,
}

func init() {
	registerConvertFlags(Cmd)

	// Built from parserTypeNames() — the same list DetectionOrder and
	// --from's help text draw from — so this text can't drift out of sync
	// with what the command actually accepts, the way its hand-written
	// predecessor (which never mentioned viseca) did.
	Cmd.Long = fmt.Sprintf(`Convert a statement, or a directory of statements, to CSV.

Each file is offered to every parser in turn, and the first one that
recognizes it performs the conversion. Supported formats: %s.

Use --from to pin a specific format instead of auto-detecting it — required
when a file's format cannot be told apart from another, or when detection
guesses wrong.

When the input is a directory, every file in it is read and their
transactions are grouped by the account named in each file name, then
written date-sorted to one CSV per account: -o releves.csv produces
releves_54293249.csv, releves_53153547.csv, and so on. Files whose names
carry no account number are written together to releves_unknown.csv. A
single releves.manifest.json run report names every CSV written.`,
		strings.Join(parserTypeNames(), ", "))
}

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

	resolve, err := resolverFor(appContainer, from, logger)
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

	res, err := resolve(inputPath)
	if err != nil {
		logger.Fatalf("Could not determine the format of %s. Supported formats: %s. "+
			"Use --from to specify it explicitly.", inputPath, strings.Join(parserTypeNames(), ", "))
	}

	resolvedOutput, err := resolveOutputFile(inputPath, outputPath)
	if err != nil {
		logger.Fatalf("%v", err)
	}

	processFile(ctx, res.Parser, inputPath, resolvedOutput, root.SharedFlags.Validate, root.Log, appContainer, format)
	root.Log.Info("Conversion completed successfully!")
}

// convertDirectory reads every file under inputDir and writes one CSV per
// account it finds, named after the CSV at outputPath (generating that name
// inside it when outputPath is an existing directory): out.csv becomes
// out_54293249.csv, out_unknown.csv, and so on, plus one run report beside
// them naming each.
func convertDirectory(ctx context.Context, appContainer *container.Container, resolve batch.ParserResolver,
	inputPath, outputPath string, logger logging.Logger, format string, recursive bool) {

	outputFile, err := resolveOutputFile(inputPath, outputPath)
	if err != nil {
		logger.Fatalf("%v", err)
		return // unreachable in production (logger.Fatal exits), but enables testing with mock logger
	}

	outFormatter, err := appContainer.GetFormatterRegistry().Get(format)
	if err != nil {
		logger.Fatalf("Invalid output format '%s': valid formats are standard, icompta, jumpsoft", format)
		return // unreachable in production (logger.Fatal exits), but enables testing with mock logger
	}

	processor := batch.NewBatchProcessor(resolve, logger, outFormatter, recursive)

	manifest, err := processor.ProcessDirectory(ctx, inputPath, outputFile)
	if err != nil {
		logger.WithError(err).Fatal("Batch conversion failed")
		return
	}

	manifestPath := batch.ManifestPathFor(outputFile)

	// Each account's rows are in their own CSV, so the path the user typed
	// is never the path their transactions ended up in: name every file.
	for _, account := range manifest.Accounts {
		logger.Info("Wrote account CSV",
			logging.Field{Key: "account", Value: account.Account},
			logging.Field{Key: "path", Value: account.OutputFile},
			logging.Field{Key: "transactions", Value: account.TransactionCount})
	}

	logger.Info(fmt.Sprintf("Batch complete: %d/%d files succeeded",
		manifest.SuccessCount, manifest.TotalFiles))

	if manifest.FailureCount > 0 {
		logger.Warn(fmt.Sprintf("%d files failed (see %s for details)",
			manifest.FailureCount, manifestPath))
	}

	if exitCode := manifest.ExitCode(); exitCode != 0 {
		root.SetExitCode(exitCode)
	}
}
