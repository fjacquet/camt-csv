package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"
)

// RecordExitCode records the process exit code requested by a batch run
// (cmd/convert's directory branch). It defers the actual os.Exit to main so
// that the root PersistentPostRun hook still runs and saves the
// creditor/debitor mappings.
//
// This used to go through a package-level function variable so tests could
// intercept it without exiting the test process. That seam's only caller was
// deleted along with common.RunConvert/FolderConvert; cmd/convert's own tests
// assert through root.ExitCode() instead, so the indirection is gone too.
func RecordExitCode(code int) {
	root.SetExitCode(code)
}

// ResolverFor builds the ParserResolver for one run. With from empty, each
// file is offered to every validator in turn, and the detected format is
// logged at INFO per file — this is the only place that can, since it is the
// only code that sees both the file path and the ParserType DetectParser
// chose. With from set, the named parser is pinned for every file and nothing
// is logged: the format was already named on the command line.
func ResolverFor(c *container.Container, from string, logger logging.Logger) (batch.ParserResolver, error) {
	if from == "" {
		return func(filePath string) (parser.FullParser, error) {
			p, parserType, err := c.DetectParser(filePath)
			if err != nil {
				return nil, batch.ErrNoParser
			}
			logger.Info("Detected input format",
				logging.Field{Key: "file", Value: filePath},
				logging.Field{Key: "format", Value: string(parserType)})
			return p, nil
		}, nil
	}

	p, err := c.GetParser(container.ParserType(from))
	if err != nil {
		return nil, fmt.Errorf("unknown input format %q: valid values are %s",
			from, strings.Join(ParserTypeNames(), ", "))
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
