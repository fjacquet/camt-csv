package batch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/formatter"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Resolution is the parser chosen for one input file, and whether choosing it
// already proved the file is in that parser's format.
//
// Auto-detection proves it as a side effect — DetectParser returns the first
// parser whose ValidateFormat accepts the file — so re-validating in parseFile
// would repeat the most expensive step of the slowest formats: a second
// pdftotext subprocess per PDF, a second whole-document unmarshal per CAMT.
// A pinned parser (--from) has proved nothing, so it still gets validated.
type Resolution struct {
	Parser    parser.FullParser
	Validated bool
}

// ParserResolver returns the parser to use for one input file, and whether
// resolving it already proved the format.
//
// It is the seam that lets a single directory hold several formats: the CLI
// supplies either a closure over Container.DetectParser (the default) or one
// returning a parser pinned by --from. Package batch therefore needs no
// knowledge of the container or of how formats are recognized.
type ParserResolver func(filePath string) (Resolution, error)

// ErrNoParser reports that no parser accepts a file. Resolvers return it so a
// batch records the file as a failure and moves on.
var ErrNoParser = errors.New("no parser recognizes this file format")

// PinnedResolver returns a ParserResolver that hands back p for every file.
// This is what --from produces: an escape hatch for a batch the detector reads
// wrongly, not a filter that selects matching files. Files p cannot read fail
// individually and are recorded in the manifest. Validated is always false: a
// pinned parser has proved nothing about the file yet, unlike detection.
func PinnedResolver(p parser.FullParser) ParserResolver {
	return func(string) (Resolution, error) { return Resolution{Parser: p}, nil }
}

// BatchProcessor handles standardized batch processing for any parser
type BatchProcessor struct {
	resolve   ParserResolver
	logger    logging.Logger
	formatter formatter.OutputFormatter
	recursive bool
}

// NewBatchProcessor creates a new BatchProcessor instance that resolves a
// parser per file via resolve. The processor uses the resolved parser for
// validation, parsing, and CSV writing operations.
// If fmt is nil, a StandardFormatter will be used by default for backward compatibility.
//
// recursive is fixed here rather than settable afterwards: ProcessDirectory
// reads it, so a later change could alter a run already under way.
func NewBatchProcessor(resolve ParserResolver, logger logging.Logger, fmt formatter.OutputFormatter, recursive bool) *BatchProcessor {
	// Default to StandardFormatter if formatter is nil (backward compatibility)
	if fmt == nil {
		fmt = formatter.NewStandardFormatter()
	}

	// A nil resolve is a caller bug, not something a batch run should panic
	// on mid-file: NewBatchProcessor is exported and now takes a function
	// value, so default to a resolver that fails every file individually —
	// the same outcome any other resolver rejecting a file already produces.
	if resolve == nil {
		resolve = func(string) (Resolution, error) { return Resolution{}, ErrNoParser }
	}

	return &BatchProcessor{
		resolve:   resolve,
		logger:    logger,
		formatter: fmt,
		recursive: recursive,
	}
}

// ManifestPathFor names the run report for an output file by replacing its
// extension: releves.csv -> releves.manifest.json.
//
// The report used to be a fixed .manifest.json inside the output directory.
// With a directory input now producing a single file, there is no output
// directory to hold it, and two runs writing into the same folder would
// otherwise overwrite each other's report.
func ManifestPathFor(outputFile string) string {
	return strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".manifest.json"
}

// AccountOutputPathFor names the CSV holding one account's rows, by suffixing
// the base output name: releves.csv + 54293249 -> releves_54293249.csv.
//
// Every account gets a suffix, including a lone one: a run's outputs are then
// recognizable as a set, and no file's name silently means "everything that
// was left over".
func AccountOutputPathFor(outputFile, account string) string {
	ext := filepath.Ext(outputFile)
	return strings.TrimSuffix(outputFile, ext) + "_" + account + ext
}

// unknownAccount labels rows from files whose names carry no account number.
// They are written to their own CSV rather than dropped or folded into a real
// account's file: an unrecognized name is a naming problem, not a reason to
// lose transactions.
const unknownAccount = "unknown"

// accountGroup collects one account's transactions in the order its files were
// read. Groups are kept in a slice, not just a map, so outputs and the
// manifest are ordered by account and reproducible across runs.
type accountGroup struct {
	account      string
	transactions []models.Transaction
}

// ProcessDirectory reads every file under inputDir and writes one CSV per
// account found there, named by AccountOutputPathFor, plus a single run report
// beside them.
//
// Files are attributed to an account by name (common.AccountKeyFromFilename);
// those carrying no account number are written together to the "unknown" CSV.
// Splitting is unconditional: a folder downloaded from a bank routinely covers
// several accounts, and one merged CSV mixes rows that an accounting import
// must keep apart — a mistake that is invisible in the output.
//
// Returns a manifest (never nil) containing results for each file processed.
// Individual file failures are captured in the manifest, not returned as
// errors: one unreadable file out of forty must not discard the other
// thirty-nine, so the CSV is written with what succeeded and the exit code
// reports partial success.
//
// An error is returned only for problems with the directories themselves.
func (bp *BatchProcessor) ProcessDirectory(ctx context.Context, inputDir, outputFile string) (*BatchManifest, error) {
	startTime := time.Now()

	// Validate input directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("input directory does not exist: %s", inputDir)
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Discover files to process
	files, err := bp.discoverFiles(inputDir)
	if err != nil {
		return nil, err
	}

	// A repeat run over the same folder must not ingest its own previous
	// output: the old manifest name was hidden (.manifest.json) and so
	// always skipped by discoverFiles, but the new name is not. Without
	// this, running the same convert command twice would offer the CSV and
	// manifest from run 1 to every parser as inputs for run 2, recording
	// both as format_not_recognized failures and turning an idempotent
	// command into a partial-success exit code.
	files = excludeOwnOutputs(files, outputFile)

	bp.logger.Info("Starting batch processing",
		logging.Field{Key: "input_dir", Value: inputDir},
		logging.Field{Key: "output_file", Value: outputFile},
		logging.Field{Key: "files_found", Value: len(files)})

	// Initialize manifest
	manifest := &BatchManifest{
		TotalFiles:   len(files),
		SuccessCount: 0,
		FailureCount: 0,
		Results:      make([]BatchResult, 0, len(files)),
		ProcessedAt:  time.Now(),
	}

	var groups []accountGroup
	byAccount := make(map[string]int) // account -> index into groups
	var cancelled bool

	// Process each file sequentially
	for _, filePath := range files {
		// Check for cancellation
		select {
		case <-ctx.Done():
			bp.logger.Warn("Batch processing cancelled",
				logging.Field{Key: "processed", Value: len(manifest.Results)},
				logging.Field{Key: "total", Value: manifest.TotalFiles})
			cancelled = true
		default:
		}
		if cancelled {
			break
		}

		transactions, result := bp.parseFile(ctx, filePath)

		account := common.AccountKeyFromFilename(filePath)
		if account == "" {
			account = unknownAccount
		}
		result.Account = account
		manifest.Results = append(manifest.Results, result)

		if result.Success {
			manifest.SuccessCount++
			idx, seen := byAccount[account]
			if !seen {
				groups = append(groups, accountGroup{account: account})
				idx = len(groups) - 1
				byAccount[account] = idx
			}
			groups[idx].transactions = append(groups[idx].transactions, transactions...)
		} else {
			manifest.FailureCount++
		}
	}

	// Ordered by account so the CSVs a run writes, and the manifest section
	// naming them, are the same from one run to the next regardless of the
	// order the files happened to be read in.
	sort.Slice(groups, func(i, j int) bool { return groups[i].account < groups[j].account })

	// A group holding no transactions is skipped rather than written empty:
	// WriteTransactionsToCSVWithFormatter errors on a nil slice (it only
	// no-ops on a non-nil empty one), so an empty batch would otherwise fail
	// the whole run with a spurious write error instead of the accurate
	// "no files converted" story the manifest and exit code already tell.
	//
	// On cancellation, nothing is written even if some files already
	// succeeded: a truncated statement set silently imported into accounting
	// software is worse than none. The manifest is written below regardless,
	// so the run report — which files completed, and how many transactions
	// they carried — survives even though the CSVs do not.
	//
	// Duplicate detection runs per account, not across the whole batch: two
	// accounts holding the same amount on the same day is what a transfer
	// between them looks like, and warning about it would train the user to
	// ignore the warning that matters.
	var writeErr error
	csvWritten := false
	aggregator := NewBatchAggregator(bp.logger)
	delimiter := bp.formatter.Delimiter()

	for i := range groups {
		if cancelled || len(groups[i].transactions) == 0 {
			continue
		}

		group := &groups[i]
		group.transactions = aggregator.Consolidate(group.transactions, group.account)
		accountFile := AccountOutputPathFor(outputFile, group.account)

		if err := common.WriteTransactionsToCSVWithFormatter(
			group.transactions, accountFile, bp.logger, bp.formatter, delimiter); err != nil {
			// One account's failed write must not discard the others: the
			// remaining CSVs are still written, and the first error is
			// returned so the run reports the failure.
			bp.logger.WithError(err).Error("Failed to write account CSV",
				logging.Field{Key: "account", Value: group.account},
				logging.Field{Key: "path", Value: accountFile})
			if writeErr == nil {
				writeErr = fmt.Errorf("failed to write CSV for account %s: %w", group.account, err)
			}
			continue
		}

		csvWritten = true
		manifest.Accounts = append(manifest.Accounts, AccountSummary{
			Account:          group.account,
			OutputFile:       accountFile,
			TransactionCount: len(group.transactions),
		})
	}

	// Nothing was written this run (cancelled, or every file yielded zero
	// transactions), yet a CSV from an earlier run may still be sitting at
	// outputFile — left there beside a fresh manifest reporting zero
	// transactions, it reads as this run's output. It is not deleted: an
	// unexpected zero-transaction run (most commonly --from pinned wrong) is
	// exactly the situation where silently destroying a prior good
	// conversion would be least welcome. writeErr == nil excludes the case
	// where the write itself failed partway — the file there may already be
	// a truncated new write, not a stale old one, and that failure is
	// reported separately.
	if !csvWritten && writeErr == nil {
		for _, stale := range staleOutputs(outputFile) {
			bp.logger.Warn("This run produced no transactions to write, but an earlier "+
				"run's CSV is still present at the output path and was left untouched",
				logging.Field{Key: "path", Value: stale})
		}
	}

	for _, group := range groups {
		manifest.TransactionCount += len(group.transactions)
	}
	manifest.Duration = time.Since(startTime)

	// A file that validated and parsed without error but yielded no
	// transactions is not visible anywhere else in the manifest: every
	// per-file BatchResult still reports Success=true. This is the only
	// place that can warn the user their whole batch silently converted
	// nothing, most commonly because --from pinned the wrong parser.
	if !cancelled && manifest.SuccessCount > 0 && manifest.TransactionCount == 0 {
		bp.logger.Warn("All successfully parsed files yielded zero transactions; "+
			"check that the correct parser was used (e.g. --from)",
			logging.Field{Key: "success_count", Value: manifest.SuccessCount})
	}

	bp.logger.Info("Batch processing completed",
		logging.Field{Key: "total_files", Value: manifest.TotalFiles},
		logging.Field{Key: "success", Value: manifest.SuccessCount},
		logging.Field{Key: "failed", Value: manifest.FailureCount},
		logging.Field{Key: "transactions", Value: manifest.TransactionCount},
		logging.Field{Key: "duration", Value: manifest.Duration.String()})

	// The manifest is written even when the CSV write above failed, or the
	// run was cancelled: it is the run report naming which files completed,
	// and losing it exactly when the run failed would defeat its entire
	// purpose (ADR-021).
	manifestPath := ManifestPathFor(outputFile)
	if err := manifest.WriteManifest(manifestPath); err != nil {
		bp.logger.WithError(err).Warn("Failed to write manifest file",
			logging.Field{Key: "path", Value: manifestPath})
	} else {
		bp.logger.Info("Wrote batch manifest",
			logging.Field{Key: "path", Value: manifestPath})
	}

	if cancelled {
		return manifest, ctx.Err()
	}

	return manifest, writeErr
}

// discoverFiles returns a sorted list of processable files in the given directory.
// Hidden files and directories (names starting with '.') are always skipped.
// Subdirectories are descended into only when the processor is recursive.
//
// A directory that cannot be read is an error rather than an empty result: a
// truncated work list would otherwise produce a manifest reporting success for
// every file it happened to find, with no sign that others were missed.
func (bp *BatchProcessor) discoverFiles(inputDir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", inputDir, err)
	}

	for _, entry := range entries {
		// Hidden entries are skipped whether they are files or directories:
		// .git and .manifest.json are never inputs.
		if strings.HasPrefix(entry.Name(), ".") {
			bp.logger.Debug("Skipping hidden entry",
				logging.Field{Key: "name", Value: entry.Name()})
			continue
		}

		entryPath := filepath.Join(inputDir, entry.Name())

		if entry.IsDir() {
			if bp.recursive {
				nested, err := bp.discoverFiles(entryPath)
				if err != nil {
					return nil, err
				}
				files = append(files, nested...)
			}
			continue
		}

		files = append(files, entryPath)
	}

	// Sort for consistent, reproducible ordering across runs.
	sort.Strings(files)

	return files, nil
}

// excludeOwnOutputs drops a run's own artifacts from a discovered file list:
// the manifest, and every per-account CSV a previous run wrote for this same
// output name.
//
// The account CSVs cannot be listed up front — which accounts exist is only
// known after every file has been read — so they are matched by the name shape
// AccountOutputPathFor produces: same directory, same base name, one suffix,
// same extension. Comparison is on resolved absolute paths so relative vs.
// absolute spelling of the same directory doesn't slip through.
//
// ProcessDirectory is exported, so this guard lives here rather than only at
// the CLI layer: any caller writing its output inside the directory it just
// read deserves the same protection against re-ingesting its own report and
// CSVs on a second run.
func excludeOwnOutputs(files []string, outputFile string) []string {
	outputDir := absPath(filepath.Dir(outputFile))
	ext := filepath.Ext(outputFile)
	prefix := strings.TrimSuffix(filepath.Base(outputFile), ext) + "_"
	manifest := absPath(ManifestPathFor(outputFile))

	filtered := files[:0]
	for _, f := range files {
		abs := absPath(f)
		if abs == manifest {
			continue
		}
		if filepath.Dir(abs) == outputDir && isAccountOutputName(filepath.Base(abs), prefix, ext) {
			continue
		}
		filtered = append(filtered, f)
	}
	return filtered
}

// staleOutputs lists the per-account CSVs an earlier run left at this output
// name. A run writing nothing must name them, so a fresh manifest reporting
// zero transactions is never read alongside an older run's CSVs as if they
// were this run's output.
func staleOutputs(outputFile string) []string {
	ext := filepath.Ext(outputFile)
	prefix := strings.TrimSuffix(filepath.Base(outputFile), ext) + "_"

	matches, err := filepath.Glob(strings.TrimSuffix(outputFile, ext) + "_*" + ext)
	if err != nil {
		return nil
	}

	var stale []string
	for _, m := range matches {
		if isAccountOutputName(filepath.Base(m), prefix, ext) {
			stale = append(stale, m)
		}
	}
	sort.Strings(stale)
	return stale
}

// isAccountOutputName reports whether a file name is one AccountOutputPathFor
// could have produced for this output: the base name, an underscore, then
// either an account number or the "unknown" label.
//
// Matching the prefix alone would be too greedy in both directions. A source
// file the user happens to have named releves_backup.csv would be silently
// skipped as if this command had written it, and a run that wrote nothing
// would name it in a stale-output warning, sending the user looking for a CSV
// this command never produced.
func isAccountOutputName(name, prefix, ext string) bool {
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ext) {
		return false
	}

	account := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ext)
	if account == unknownAccount {
		return true
	}
	if account == "" {
		return false
	}
	for _, r := range account {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// absPath resolves a path, falling back to a cleaned relative one when the
// working directory cannot be read.
func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

// parseFile resolves a parser for filePath and returns the transactions it
// yields. It never writes: a batch produces one consolidated CSV, written once
// by ProcessDirectory after every file has been read.
//
// This method never panics and captures all errors in the returned result.
func (bp *BatchProcessor) parseFile(ctx context.Context, filePath string) ([]models.Transaction, BatchResult) {
	fileName := filepath.Base(filePath)

	bp.logger.Info("Processing file",
		logging.Field{Key: "file", Value: fileName})

	result := BatchResult{
		FilePath:    filePath,
		FileName:    fileName,
		Success:     false,
		Error:       "",
		RecordCount: 0,
	}

	res, err := bp.resolve(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("format_not_recognized: %v", err)
		bp.logger.WithError(err).Warn("Skipping file of unrecognized format",
			logging.Field{Key: "file", Value: fileName})
		return nil, result
	}
	p := res.Parser

	// Step 1: Validate format, unless the resolver already proved it (the
	// detect path validates as a side effect of choosing the parser; a
	// pinned --from parser has not, so it is still validated here).
	if !res.Validated {
		isValid, err := p.ValidateFormat(filePath)
		if err != nil {
			result.Error = fmt.Sprintf("validation_error: %v", err)
			bp.logger.WithError(err).Warn("Validation error",
				logging.Field{Key: "file", Value: fileName})
			return nil, result
		}

		if !isValid {
			result.Error = "validation_failed"
			bp.logger.Warn("Invalid format",
				logging.Field{Key: "file", Value: fileName})
			return nil, result
		}
	}

	// Step 2: Open and parse file
	file, err := os.Open(filePath) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		result.Error = fmt.Sprintf("open_error: %v", err)
		bp.logger.WithError(err).Warn("Failed to open file",
			logging.Field{Key: "file", Value: fileName})
		return nil, result
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			bp.logger.WithError(closeErr).Warn("Failed to close file",
				logging.Field{Key: "file", Value: fileName})
		}
	}()

	transactions, err := p.Parse(ctx, file)
	if err != nil {
		result.Error = err.Error()
		bp.logger.WithError(err).Warn("Parse error",
			logging.Field{Key: "file", Value: fileName})
		return nil, result
	}

	// Success!
	result.Success = true
	result.RecordCount = len(transactions)

	bp.logger.Info("Successfully processed file",
		logging.Field{Key: "file", Value: fileName},
		logging.Field{Key: "records", Value: result.RecordCount})

	return transactions, result
}
