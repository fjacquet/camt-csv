package convert

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/formatter"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// revolutSampleCSV is copied from internal/container/detect_test.go — its
// constants are unexported and in another package, so duplicating the one
// row this file needs is correct here.
const revolutSampleCSV = `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2026-03-15 10:00:00,2026-03-15 10:00:00,Coop Pronto,-24.50,0.00,CHF,COMPLETED,975.50`

// newTestContainer builds a container with AI disabled so tests never reach
// a network.
func newTestContainer(t *testing.T) *container.Container {
	t.Helper()
	t.Setenv("TEST_MODE", "true")

	cfg := &config.Config{}
	cfg.Log.Level = "error"
	cfg.Log.Format = "text"

	c, err := container.NewContainer(cfg)
	require.NoError(t, err)
	return c
}

// mustResolve returns the parser one run would use for path, failing the test
// if resolution errors.
func mustResolve(t *testing.T, c *container.Container, from, path string) parser.FullParser {
	t.Helper()
	resolve, err := resolverFor(c, from, c.GetLogger())
	require.NoError(t, err)
	res, err := resolve(path)
	require.NoError(t, err)
	return res.Parser
}

// --from must offer exactly the registered parser types, so the flag cannot
// drift from what the container can actually build.
func TestResolverFor_AcceptsEveryDetectableType(t *testing.T) {
	c := newTestContainer(t)

	for _, pt := range container.DetectionOrder() {
		resolve, err := resolverFor(c, string(pt), c.GetLogger())
		require.NoError(t, err, "--from %s must be accepted", pt)

		res, err := resolve("/irrelevant/path.csv")
		require.NoError(t, err)
		assert.NotNil(t, res.Parser)
	}
}

func TestResolverFor_RejectsUnknownFormat(t *testing.T) {
	c := newTestContainer(t)

	_, err := resolverFor(c, "postbank", c.GetLogger())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "postbank")
	assert.Contains(t, err.Error(), "camt", "the error must list the valid values")
}

// --from is an escape hatch, not a filter: it pins the parser even for a file
// the detector would have routed elsewhere, so a wrong pin fails loudly rather
// than silently falling back to the right parser.
func TestResolverFor_PinBypassesDetection(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	revolutFile := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(revolutFile, []byte(revolutSampleCSV), 0600))

	resolve, err := resolverFor(c, "selma", c.GetLogger())
	require.NoError(t, err)

	res, err := resolve(revolutFile)
	require.NoError(t, err, "the pin resolves regardless of content")

	valid, _ := res.Parser.ValidateFormat(revolutFile)
	assert.False(t, valid, "the pinned Selma parser must not accept a Revolut file")
}

// The negative half above (a wrong pin rejects the file) proves nothing about
// whether the pin actually selected the parser named by --from: an
// implementation that ignored `from` and always pinned CAMT would pass it too
// (CAMT's ValidateFormat also rejects a Revolut CSV). This is the positive
// half: pinning "revolut" against a genuine Revolut file must select a parser
// that accepts it.
func TestResolverFor_PinSelectsNamedParser(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	revolutFile := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(revolutFile, []byte(revolutSampleCSV), 0600))

	resolve, err := resolverFor(c, "revolut", c.GetLogger())
	require.NoError(t, err)

	res, err := resolve(revolutFile)
	require.NoError(t, err)

	valid, err := res.Parser.ValidateFormat(revolutFile)
	require.NoError(t, err)
	assert.True(t, valid, "pinning --from revolut must select a parser that accepts a Revolut file")
}

// With no --from, resolution falls back to detection and an unrecognized file
// is reported as such rather than guessed at.
func TestResolverFor_DetectsWhenUnset(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	unknown := filepath.Join(dir, "mystery.csv")
	require.NoError(t, os.WriteFile(unknown, []byte("a,b,c\n1,2,3\n"), 0600))

	resolve, err := resolverFor(c, "", c.GetLogger())
	require.NoError(t, err)

	_, err = resolve(unknown)
	assert.ErrorIs(t, err, batch.ErrNoParser)
}

// convert's entire premise is auto-detection, so a successful detection must
// tell the user what was detected — the message the old DetectParser call
// site used to log directly. ResolverFor's detecting closure is now the only
// code that sees both the file path and the ParserType DetectParser chose, so
// it is the only place that can log it, for both the single-file and the
// directory path.
func TestResolverFor_LogsDetectedFormat(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	revolutFile := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(revolutFile, []byte(revolutSampleCSV), 0600))

	mockLogger := logging.NewMockLogger()
	resolve, err := resolverFor(c, "", mockLogger)
	require.NoError(t, err)

	_, err = resolve(revolutFile)
	require.NoError(t, err)

	found := false
	for _, entry := range mockLogger.GetEntriesByLevel("INFO") {
		if entry.Message != "Detected input format" {
			continue
		}
		for _, f := range entry.Fields {
			if f.Key == "format" && f.Value == "revolut" {
				found = true
			}
		}
	}
	assert.True(t, found, "detecting a file must log its detected format")
}

// A pinned --from logs nothing: the format was already named on the command
// line, so a per-file "detected" message would be noise, not information.
func TestResolverFor_PinnedDoesNotLogDetection(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	revolutFile := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(revolutFile, []byte(revolutSampleCSV), 0600))

	mockLogger := logging.NewMockLogger()
	resolve, err := resolverFor(c, "revolut", mockLogger)
	require.NoError(t, err)

	_, err = resolve(revolutFile)
	require.NoError(t, err)

	assert.False(t, mockLogger.HasEntry("INFO", "Detected input format"),
		"a pinned --from must not log a detection message")
}

// A single-file conversion's outcome is its exit code. A one-entry run report
// adds nothing, so none is written.
func TestConvert_SingleFileWritesNoManifest(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(input, []byte(revolutSampleCSV), 0600))
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	err := processFileWithErrorFormatted(context.Background(),
		mustResolve(t, c, "", input), input, outputFile, false, c.GetLogger(), c, "standard")

	require.NoError(t, err)
	assert.FileExists(t, outputFile)
	assert.NoFileExists(t, batch.ManifestPathFor(outputFile))
}

// -o names a file. Pointing it at an existing directory is a convenience the
// PDF command used to offer, and it survives: the folder being read names the
// output written inside it.
func TestResolveOutputFile_ExistingDirectoryGetsGeneratedName(t *testing.T) {
	inputDir := filepath.Join(t.TempDir(), "releves-2024")
	require.NoError(t, os.MkdirAll(inputDir, 0750))
	outputDir := t.TempDir()

	got, err := resolveOutputFile(inputDir, outputDir)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "releves-2024.csv"), got)
}

// A trailing separator on -o must be read as "this is a directory" even when
// the directory does not exist yet: os.Stat can't tell existing-dir from
// not-yet-created-dir, and without this signal the generated name stays
// "outdir/" verbatim. ProcessDirectory would then os.MkdirAll(filepath.Dir(
// "outdir/")) — creating outdir as a directory — and only fail once every
// file has already been parsed, trying to write the CSV into it ("is a
// directory").
func TestResolveOutputFile_TrailingSeparatorIsTreatedAsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "releves-2024")
	require.NoError(t, os.MkdirAll(inputDir, 0750))
	outputDir := filepath.Join(tempDir, "outdir") + string(filepath.Separator)

	got, err := resolveOutputFile(inputDir, outputDir)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tempDir, "outdir", "releves-2024.csv"), got)
	assert.NoDirExists(t, filepath.Join(tempDir, "outdir"),
		"resolving the output path must not itself create the directory")
}

// A file input pointed at an existing directory must not double the
// extension: the directory-naming branch used to run before the
// input-is-a-directory check, so a file input named "statement.csv" got
// ".csv" appended on top of its own extension, yielding "statement.csv.csv".
func TestResolveOutputFile_FileInputIntoExistingDirectoryDoesNotDoubleExtension(t *testing.T) {
	inputDir := t.TempDir()
	inputFile := filepath.Join(inputDir, "statement.csv")
	require.NoError(t, os.WriteFile(inputFile, []byte("x"), 0600))
	outputDir := t.TempDir()

	got, err := resolveOutputFile(inputFile, outputDir)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "statement.csv"), got,
		"a file input's own name must be reused verbatim, not have .csv appended again")
}

// The output-under-input guard used to apply only to directory inputs,
// leaving a single-file conversion free to seed its own directory:
// `convert -i samples/revolut.csv -o samples/out.csv` would write a CSV that
// a later `convert -i samples/ --recursive` reads back as input. The guard
// now covers file inputs too, refusing output in the file's own directory.
func TestResolveOutputFile_FileInputBesideItselfIsRefused(t *testing.T) {
	inputDir := t.TempDir()
	inputFile := filepath.Join(inputDir, "statement.csv")
	require.NoError(t, os.WriteFile(inputFile, []byte("x"), 0600))
	outputPath := filepath.Join(inputDir, "out.csv")

	_, err := resolveOutputFile(inputFile, outputPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "inside the input directory")
}

func TestResolveOutputFile_PlainPathIsUnchanged(t *testing.T) {
	inputDir := t.TempDir()
	want := filepath.Join(t.TempDir(), "out.csv")

	got, err := resolveOutputFile(inputDir, want)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// filepath.Rel between the input directory and the output's directory decides
// the guard, and its result is not a simple string prefix: "." (write
// straight into the input) and a true subdirectory are refused; ".." (the
// input's own parent) and a sibling directory that merely shares a string
// prefix with the input's name are both allowed. A naive rewrite that drops
// the `rel != ".."` clause or loosens the separator-qualified HasPrefix check
// would pass the first two subtests here while silently breaking the parent
// or sibling case — this table exists so nobody "fixes" filepath.Rel into a
// naive string prefix later without a test catching it.
func TestResolveOutputFile_GuardsAgainstInputOverlap(t *testing.T) {
	tests := []struct {
		name       string
		outputFrom func(inputDir string) string
		wantErr    bool
	}{
		{
			name:       "direct into input directory is refused",
			outputFrom: func(inputDir string) string { return filepath.Join(inputDir, "out.csv") },
			wantErr:    true,
		},
		{
			name: "nested subdirectory of input is refused",
			outputFrom: func(inputDir string) string {
				return filepath.Join(inputDir, "sub", "out.csv")
			},
			wantErr: true,
		},
		{
			name: "input's own parent directory is allowed",
			outputFrom: func(inputDir string) string {
				return filepath.Join(filepath.Dir(inputDir), "out.csv")
			},
			wantErr: false,
		},
		{
			name: "sibling directory sharing a string prefix with input is allowed",
			outputFrom: func(inputDir string) string {
				// inputDir is ".../data"; "database" shares the string
				// prefix "data" but is a genuine sibling, not a subdirectory.
				return filepath.Join(filepath.Dir(inputDir), filepath.Base(inputDir)+"base", "out.csv")
			},
			wantErr: false,
		},
		{
			// A subdirectory of input literally named "..xyz" makes
			// filepath.Rel return "..xyz" — a bare HasPrefix(rel, "..") (no
			// separator) misreads that as "starts with the up-one-level
			// token" and would wrongly allow writing back into input. This
			// is the one case that actually distinguishes the real
			// separator-qualified check from the naive rewrite the review
			// warned about; the parent/sibling cases above pass under either.
			name: "subdirectory whose name merely starts with .. is refused",
			outputFrom: func(inputDir string) string {
				return filepath.Join(inputDir, "..xyz", "out.csv")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputDir := filepath.Join(t.TempDir(), "data")
			require.NoError(t, os.MkdirAll(inputDir, 0750))

			outputPath := tt.outputFrom(inputDir)

			_, err := resolveOutputFile(inputDir, outputPath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "inside the input directory")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// convert takes its input/output via -i/-o, not positional args. Without a
// cobra.Args validator, `convert statement.pdf` is silently accepted (the
// argument is just discarded) and fails deep inside with a confusing "Error
// accessing input path: stat : no such file or directory" (the empty,
// unset -i) instead of cobra's own clear error naming the problem.
func TestCmd_RejectsPositionalArgs(t *testing.T) {
	require.NotNil(t, Cmd.Args, "convert must reject positional arguments")
	assert.NoError(t, Cmd.Args(Cmd, []string{}), "no positional args must still be accepted")
	assert.Error(t, Cmd.Args(Cmd, []string{"statement.pdf"}),
		"a stray positional argument must be rejected, not silently discarded")
}

func TestCmd_IsRegisteredWithFlags(t *testing.T) {
	assert.Equal(t, "convert", Cmd.Use)
	assert.NotNil(t, Cmd.Flags().Lookup("format"), "convert must accept --format")
	assert.NotNil(t, Cmd.Flags().Lookup("recursive"), "convert must accept --recursive")
	assert.NotNil(t, Cmd.Flags().Lookup("from"), "convert must accept --from")
	assert.NotNil(t, Cmd.Flags().Lookup("keep-payments"), "convert must accept --keep-payments")

	dateFormat := Cmd.Flags().Lookup("date-format")
	require.NotNil(t, dateFormat, "convert must accept --date-format")
	assert.NotEmpty(t, dateFormat.Deprecated, "--date-format must be marked deprecated")
}

// Every detectable format named in DetectionOrder must appear in the
// command's help text, generated rather than hand-maintained so a new parser
// (or one merely forgotten, as viseca was) can't silently go missing from it.
func TestCmd_LongListsEveryDetectableFormat(t *testing.T) {
	for _, pt := range container.DetectionOrder() {
		assert.Contains(t, Cmd.Long, string(pt), "Long must mention every detectable format")
	}
}

// convertDirectory success path: two files pinned to the Revolut parser merge
// into one output CSV plus a manifest, with no FATAL logged.
func TestConvertDirectory_Success(t *testing.T) {
	c := newTestContainer(t)
	inputDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "a.csv"), []byte(revolutSampleCSV), 0600))
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	mockLogger := logging.NewMockLogger()
	resolve, err := resolverFor(c, "revolut", mockLogger)
	require.NoError(t, err)

	convertDirectory(context.Background(), c, resolve, inputDir, outputFile, mockLogger, "standard", false)

	assert.Empty(t, mockLogger.GetEntriesByLevel("FATAL"))
	// Outputs carry the account they hold; a.csv names none, so its rows go
	// to the "unknown" CSV.
	assert.FileExists(t, batch.AccountOutputPathFor(outputFile, "unknown"))
	assert.FileExists(t, batch.ManifestPathFor(outputFile))
}

// An invalid --format must stop before any batch work happens. Without the
// early return after logger.Fatalf, execution falls through to
// batch.NewBatchProcessor with a nil formatter (which silently substitutes
// StandardFormatter) and writes output anyway — the exact "invalid format
// gets ignored" bug this test pins against.
func TestConvertDirectory_InvalidFormatWritesNothing(t *testing.T) {
	c := newTestContainer(t)
	inputDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "a.csv"), []byte(revolutSampleCSV), 0600))
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	mockLogger := logging.NewMockLogger()
	resolve, err := resolverFor(c, "revolut", mockLogger)
	require.NoError(t, err)

	convertDirectory(context.Background(), c, resolve, inputDir, outputFile, mockLogger, "not-a-format", false)

	require.NotEmpty(t, mockLogger.GetEntriesByLevel("FATAL"))
	assert.NoFileExists(t, outputFile, "an invalid format must not silently fall back to writing standard output")
}

// -o pointed at the input directory must stop before any batch work happens,
// for the same reason as the invalid-format case above. Asserting only that
// outputPath itself was never written is too weak here: without the early
// return, ResolveOutputFile's error leaves outputFile == "", and the batch
// still runs and fails a beat later trying to write to that empty path,
// producing a second FATAL that a looser assertion could mistake for this
// one. The absence of "Starting batch processing" is what actually pins the
// guard stopping things before BatchProcessor ever starts.
func TestConvertDirectory_OutputUnderInputWritesNothing(t *testing.T) {
	c := newTestContainer(t)
	inputDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "a.csv"), []byte(revolutSampleCSV), 0600))
	outputPath := filepath.Join(inputDir, "out.csv")

	mockLogger := logging.NewMockLogger()
	resolve, err := resolverFor(c, "revolut", mockLogger)
	require.NoError(t, err)

	convertDirectory(context.Background(), c, resolve, inputDir, outputPath, mockLogger, "standard", false)

	fatalEntries := mockLogger.GetEntriesByLevel("FATAL")
	require.NotEmpty(t, fatalEntries)
	assert.Contains(t, fatalEntries[0].Message, "inside the input directory")
	assert.NoFileExists(t, outputPath)
	assert.False(t, mockLogger.HasEntry("INFO", "Starting batch processing"),
		"the guard must stop before BatchProcessor starts, not merely before it succeeds")
}

// convertDirectory's non-zero exit codes must reach root.ExitCode() — the
// value main() actually exits with — via root.SetExitCode rather than a
// route that bypasses it.
func TestConvertDirectory_RecordsExitCodeThroughSeam(t *testing.T) {
	root.ResetExitCode()
	defer root.ResetExitCode()

	c := newTestContainer(t)
	inputDir := t.TempDir() // empty: zero files processed -> ExitCode() == 2
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	mockLogger := logging.NewMockLogger()
	resolve, err := resolverFor(c, "revolut", mockLogger)
	require.NoError(t, err)

	convertDirectory(context.Background(), c, resolve, inputDir, outputFile, mockLogger, "standard", false)

	assert.Equal(t, 2, root.ExitCode())
}

// The behavior the pdf command used to own — a directory of PDFs consolidated
// into one chronologically sorted CSV — must survive its deletion. This is the
// regression guard for that removal.
//
// samples/pdf holds exactly one file, so consolidating that directory as-is
// merges nothing and asserting only "at least one file converted" would pass
// even if BatchProcessor's directory merge were broken. Two independent
// copies of the same sample, under different names, and an assertion that the
// consolidated transaction count is exactly double a single parse, is what
// actually exercises the merge rather than only the single-file path.
func TestConvert_PDFDirectoryConsolidates(t *testing.T) {
	// Skipped where poppler-utils is absent: the PDF parser shells out to pdftotext.
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed")
	}

	c := newTestContainer(t)
	resolve, err := resolverFor(c, "pdf", c.GetLogger())
	require.NoError(t, err)

	src, err := os.ReadFile("../../samples/pdf/viseca.pdf")
	require.NoError(t, err)

	inputDir := t.TempDir()
	statementA := filepath.Join(inputDir, "statement-a.pdf")
	require.NoError(t, os.WriteFile(statementA, src, 0600))                                 // #nosec G703 -- test fixture, path built from t.TempDir() and a literal name
	require.NoError(t, os.WriteFile(filepath.Join(inputDir, "statement-b.pdf"), src, 0600)) // #nosec G703 -- test fixture, path built from t.TempDir() and a literal name

	// Reference count: how many transactions the sample parses to on its own.
	res, err := resolve(statementA)
	require.NoError(t, err)
	f, err := os.Open(statementA)
	require.NoError(t, err)
	singleFileTransactions, err := res.Parser.Parse(context.Background(), f)
	require.NoError(t, f.Close())
	require.NoError(t, err)
	require.NotEmpty(t, singleFileTransactions, "the sample PDF must parse to at least one transaction")

	outputFile := filepath.Join(t.TempDir(), "consolidated.csv")
	bp := batch.NewBatchProcessor(resolve, c.GetLogger(), formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.Equal(t, 2, manifest.SuccessCount, "both copies must convert")
	assert.Equal(t, 2*len(singleFileTransactions), manifest.TransactionCount,
		"the consolidated output must carry both files' transactions merged, not just one")
	assert.FileExists(t, batch.AccountOutputPathFor(outputFile, "unknown"))
}

// With several CSVs written per run, the output path the user typed is no
// longer the path their rows are in. The command has to name each file it
// wrote, or the only record of where the transactions went is the manifest.
func TestConvertDirectory_LogsEachAccountOutput(t *testing.T) {
	c := newTestContainer(t)
	inputDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(inputDir, "a.csv"), []byte(revolutSampleCSV), 0600))
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	mockLogger := logging.NewMockLogger()
	resolve, err := resolverFor(c, "revolut", mockLogger)
	require.NoError(t, err)

	convertDirectory(context.Background(), c, resolve, inputDir, outputFile, mockLogger, "standard", false)

	written := batch.AccountOutputPathFor(outputFile, "unknown")
	found := false
	for _, entry := range mockLogger.GetEntriesByLevel("INFO") {
		for _, f := range entry.Fields {
			if f.Key == "path" && f.Value == written {
				found = true
			}
		}
	}
	assert.True(t, found, "the run must log the path of every CSV it wrote")
}
