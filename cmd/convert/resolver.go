package convert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
)

// resolverFor builds the ParserResolver for one run. With from empty, each
// file is offered to every validator in turn, and the detected format is
// logged at INFO per file — this is the only place that can, since it is the
// only code that sees both the file path and the ParserType DetectParser
// chose. With from set, the named parser is pinned for every file and nothing
// is logged: the format was already named on the command line.
func resolverFor(c *container.Container, from string, logger logging.Logger) (batch.ParserResolver, error) {
	if from == "" {
		return func(filePath string) (batch.Resolution, error) {
			p, parserType, err := c.DetectParser(filePath)
			if err != nil {
				return batch.Resolution{}, batch.ErrNoParser
			}
			logger.Info("Detected input format",
				logging.Field{Key: "file", Value: filePath},
				logging.Field{Key: "format", Value: string(parserType)})
			// DetectParser already ran ValidateFormat to pick p, so parseFile
			// does not need to run it again.
			return batch.Resolution{Parser: p, Validated: true}, nil
		}, nil
	}

	p, err := c.GetParser(container.ParserType(from))
	if err != nil {
		return nil, fmt.Errorf("unknown input format %q: valid values are %s",
			from, strings.Join(parserTypeNames(), ", "))
	}
	return batch.PinnedResolver(p), nil
}

// resolveOutputFile turns the --output value into the single CSV path a run
// writes.
//
// A path naming an existing directory, or written with a trailing separator
// (a directory that does not exist yet — os.Stat can't tell us so, and the
// alternative is discovering it only after every file has already been
// parsed, when ProcessDirectory's os.MkdirAll(filepath.Dir(...)) creates it
// and the final CSV write fails with "is a directory"), gets a file generated
// inside it, named after the input — the convenience the PDF command used to
// offer. For a directory input that is "<dir>/<input-dir-basename>.csv"; for
// a file input it is the input's own name, unchanged, so a directory -o does
// not turn "statement.csv" into "statement.csv.csv".
//
// An output under the input's directory is refused: for a directory input,
// writing inside it would let a later --recursive run read its own output
// back as input; for a file input, the equivalent hazard is writing beside
// the file, in the same directory a later directory conversion would read.
// Relying on no validator ever accepting our own CSV is not a guarantee
// worth depending on.
func resolveOutputFile(inputPath, outputPath string) (string, error) {
	inputInfo, statErr := os.Stat(inputPath)
	inputIsDir := statErr == nil && inputInfo.IsDir()

	outputIsDir := strings.HasSuffix(outputPath, string(filepath.Separator))
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		outputIsDir = true
	}

	resolved := outputPath
	if outputIsDir {
		if inputIsDir {
			resolved = filepath.Join(outputPath, filepath.Base(filepath.Clean(inputPath))+".csv")
		} else {
			resolved = filepath.Join(outputPath, filepath.Base(filepath.Clean(inputPath)))
		}
	}

	// containDir is the directory a later run must not have its input
	// re-seeded into: the input directory itself, or a file input's own
	// parent directory. Nothing to guard when inputPath could not be stat'd.
	containDir := inputPath
	if !inputIsDir {
		if statErr != nil {
			return resolved, nil
		}
		containDir = filepath.Dir(inputPath)
	}

	absInput, err := filepath.Abs(containDir)
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
			"a later --recursive run would read the output back as input", containDir)
	}

	return resolved, nil
}
