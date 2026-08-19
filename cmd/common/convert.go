package common

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// RunConvert is the shared handler for the eight format-specific convert
// commands. It pins the parser to parserType rather than detecting it, and is
// scheduled for removal alongside those commands.
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
			logger.Fatal("--output flag is required when processing a folder. Use -o or --output to specify the output file.")
		}
		FolderConvert(ctx, p, inputPath, outputPath, logger, format, recursive)
	} else {
		ProcessFile(ctx, p, inputPath, outputPath, root.SharedFlags.Validate, root.Log, appContainer, format)
		root.Log.Info(name + " to CSV conversion completed successfully!")
	}
}

// FolderConvert converts every file under inputDir into the single CSV at
// outputFile, using BatchProcessor with formatter support. This is the only
// directory-processing path for every parser.
//
// Parameters:
//   - ctx: context for cancellation
//   - p: parser (must implement parser.FullParser)
//   - inputDir: path to directory containing input files
//   - outputFile: path to the single consolidated CSV to write
//   - logger: structured logger
//   - format: output format name ("standard", "icompta" or "jumpsoft")
//   - recursive: also process files in subdirectories of inputDir
func FolderConvert(ctx context.Context, p any, inputDir, outputFile string, logger logging.Logger, format string, recursive bool) {
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

	manifest, err := processor.ProcessDirectory(ctx, inputDir, outputFile)
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
		exitFn(manifest.ExitCode())
	}
}

// ResolverFor builds the ParserResolver for one run. With from empty, each file
// is offered to every validator in turn; with from set, the named parser is
// pinned for every file.
func ResolverFor(c *container.Container, from string) (batch.ParserResolver, error) {
	if from == "" {
		return func(filePath string) (parser.FullParser, error) {
			p, _, err := c.DetectParser(filePath)
			if err != nil {
				return nil, batch.ErrNoParser
			}
			return p, nil
		}, nil
	}

	p, err := c.GetParser(container.ParserType(from))
	if err != nil {
		return nil, fmt.Errorf("unknown input format %q: valid values are %s",
			from, strings.Join(parserTypeNames(), ", "))
	}
	return batch.PinnedResolver(p), nil
}

// ResolveOutputFile turns the --output value into the single CSV path a run
// writes.
//
// A path naming an existing directory gets a file generated inside it, named
// after the input — the convenience the PDF command used to offer.
//
// An output under the input directory is refused: a later --recursive run would
// read its own output back as input. Relying on no validator ever accepting our
// own CSV is not a guarantee worth depending on.
func ResolveOutputFile(inputPath, outputPath string) (string, error) {
	resolved := outputPath
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		resolved = filepath.Join(outputPath, filepath.Base(filepath.Clean(inputPath))+".csv")
	}

	inputInfo, err := os.Stat(inputPath)
	if err != nil || !inputInfo.IsDir() {
		return resolved, nil
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return resolved, nil
	}
	absOutput, err := filepath.Abs(resolved)
	if err != nil {
		return resolved, nil
	}

	rel, err := filepath.Rel(absInput, filepath.Dir(absOutput))
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("--output must not be inside the input directory %s: "+
			"a later --recursive run would read the output back as input", inputPath)
	}

	return resolved, nil
}
