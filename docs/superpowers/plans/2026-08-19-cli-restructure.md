# CLI Restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse ten root CLI commands into two (`convert`, `categorize`) and make a directory input always produce one consolidated CSV.

**Architecture:** `BatchProcessor` stops resolving one fixed parser and stops writing one CSV per file. It gains a `ParserResolver` seam — a `func(path) (parser.FullParser, error)` supplied by the caller, satisfied either by container auto-detection or by a parser pinned with `--from`. Per-file work becomes parse-only, returning `[]models.Transaction` plus a `BatchResult` for the manifest; the accumulated transactions are consolidated by the (currently dead) `BatchAggregator` and written once.

**Tech Stack:** Go 1.24+, cobra v1.10.2, shopspring/decimal, testify, logrus.

**Spec:** `docs/adr/ADR-021-cli-command-restructure.md`

## Global Constraints

- Every numeric `Transaction` field is `shopspring/decimal`. Never `int`, never `float`.
- Never use `math/rand` — Semgrep blocks it (CWE-338).
- Output dates are always `models.DateFormatCSV` (`DD.MM.YYYY`). `--date-format` stays registered and deprecated; it has no effect.
- Tests use `t.TempDir()` for filesystem work and `TEST_MODE=true` to suppress real AI calls.
- Never assert on `Transaction.Number` — `TransactionBuilder` mints a fresh UUID per run (`builder.go:24`). `BookkeepingNumber` is the stable one.
- `TestIComptaHeaderCoversPluginMappings` must stay green: the iCompta formatter's column names are a contract, and a renamed column breaks imports silently.
- `TestDetectParser_ValidatorsDoNotOverlap` must stay green and becomes load-bearing: detection is now the only routing path.
- Run `make test` (`go test -race -coverprofile=coverage.txt -covermode=atomic ./...`) and `make lint` (`golangci-lint run --timeout=5m`) before each commit.
- Target release is **v3.0.0**; the change is deliberately breaking and ships no compatibility aliases.
- Branch: `feat/cli-restructure`. `main` is protected — integrate via PR.

## Reference: what exists today

Reading these before starting saves an hour of confusion.

- `internal/container/container.go:30-37` — the eight `ParserType` constants: `camt`, `pdf`, `revolut`, `revolut-investment`, `revolut-crypto`, `selma`, `debit`, `viseca`.
- `internal/container/detect.go` — `DetectParser(path) (parser.FullParser, ParserType, error)` and `DetectionOrder() []ParserType`.
- `internal/batch/processor.go` — `BatchProcessor`, `ProcessDirectory`, `discoverFiles`, `outputPathFor`, `processFile`.
- `internal/batch/manifest.go` — `BatchResult`, `BatchManifest`, `ExitCode()`, `WriteManifest()`.
- `internal/batch/aggregator.go` — `BatchAggregator` with private `sortTransactionsChronologically` and `detectAndLogDuplicates`. Dead in production; only caller is `internal/integration/cross_parser_test.go:166`.
- `internal/common/csv.go:88` — `WriteTransactionsToCSVWithFormatter(txs []models.Transaction, csvFile string, logger logging.Logger, formatter formatter.OutputFormatter, delimiter rune) error`.
- `internal/container/detect_test.go:110` — inline CSV fixture constants (`revolutCSV`, `selmaCSV`, `debitCSV`, `visecaCSV`, …). They are unexported and live in package `container`; tests in other packages copy the ones they need rather than importing them.

---

### Task 1: Export consolidation from BatchAggregator

Today the chronological sort and the duplicate report are private methods reachable only through `AggregateTransactions`, which takes a `FileGroup` and swallows per-file parse errors with `continue`. `BatchProcessor` needs the sort and the duplicate report but must keep its own per-file error capture for the manifest. Export the useful half, and refactor the existing entry point onto it so there is one implementation.

**Files:**
- Modify: `internal/batch/aggregator.go`
- Test: `internal/batch/aggregator_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `func (ba *BatchAggregator) Consolidate(transactions []models.Transaction, label string) []models.Transaction` — sorts in place, logs potential duplicates, returns the same slice. `label` names the batch in duplicate warnings.

- [ ] **Step 1: Write the failing test**

Append to `internal/batch/aggregator_test.go`:

```go
// Consolidate is what BatchProcessor calls once every file in a batch has been
// parsed. It must order the merged set by date regardless of the order files
// were read in, and must never drop a transaction: two identical purchases on
// the same day are a real thing, so duplicates are reported, not removed.
func TestConsolidate_SortsAndKeepsDuplicates(t *testing.T) {
	logger := logging.NewMockLogger()
	agg := batch.NewBatchAggregator(logger)

	mar := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	jan := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)

	input := []models.Transaction{
		{BookkeepingNumber: "c", Date: mar, ValueDate: mar, Amount: decimal.NewFromInt(30), PartyName: "Migros"},
		{BookkeepingNumber: "a", Date: jan, ValueDate: jan, Amount: decimal.NewFromInt(10), PartyName: "Coop"},
		{BookkeepingNumber: "b", Date: feb, ValueDate: feb, Amount: decimal.NewFromInt(20), PartyName: "SBB"},
		{BookkeepingNumber: "b2", Date: feb, ValueDate: feb, Amount: decimal.NewFromInt(20), PartyName: "SBB"},
	}

	got := agg.Consolidate(input, "releves-2024")

	require.Len(t, got, 4, "Consolidate must not drop the duplicate")

	var dates []time.Time
	for _, tx := range got {
		dates = append(dates, tx.Date)
	}
	assert.Equal(t, []time.Time{jan, feb, feb, mar}, dates, "must be ordered by date")

	// BookkeepingNumber is the stable identifier; Number is a fresh UUID per run.
	assert.Equal(t, "a", got[0].BookkeepingNumber)
	assert.Equal(t, "c", got[3].BookkeepingNumber)
}

// An empty batch is a normal outcome (a directory of unreadable files), not an
// error: Consolidate must return cleanly rather than panic on the empty slice.
func TestConsolidate_EmptyInput(t *testing.T) {
	logger := logging.NewMockLogger()
	agg := batch.NewBatchAggregator(logger)

	got := agg.Consolidate(nil, "empty")

	assert.Empty(t, got)
}
```

Ensure the file's import block has `time`, `github.com/shopspring/decimal`, `fjacquet/camt-csv/internal/models`, `fjacquet/camt-csv/internal/logging`, `fjacquet/camt-csv/internal/batch`, testify's `assert` and `require`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/batch/ -run TestConsolidate -v`
Expected: FAIL — `agg.Consolidate undefined (type *batch.BatchAggregator has no field or method Consolidate)`

- [ ] **Step 3: Write minimal implementation**

In `internal/batch/aggregator.go`, add:

```go
// Consolidate orders a merged transaction set chronologically and reports
// potential duplicates without removing any.
//
// Merging a directory of mixed formats is exactly where overlaps appear — a
// Viseca PDF statement alongside the Viseca CSV export of the same month. They
// are reported rather than deduplicated because a similarity heuristic would
// eventually erase two genuine identical purchases made on the same day;
// iCompta runs its own duplicate detection at import.
//
// label names the batch in the duplicate warnings so a user reading the log
// knows which run they refer to.
func (ba *BatchAggregator) Consolidate(transactions []models.Transaction, label string) []models.Transaction {
	ba.sortTransactionsChronologically(transactions)
	ba.detectAndLogDuplicates(transactions, label)
	return transactions
}
```

Then refactor `AggregateTransactions` so the two calls it currently makes become one call to `Consolidate` — there must be exactly one implementation of "sort then report duplicates". Replace:

```go
	// Sort transactions chronologically by date
	ba.sortTransactionsChronologically(allTransactions)

	// Log potential duplicates (but keep all transactions as per requirements)
	ba.detectAndLogDuplicates(allTransactions, group.AccountID)
```

with:

```go
	allTransactions = ba.Consolidate(allTransactions, group.AccountID)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/batch/ -v`
Expected: PASS, including the pre-existing `AggregateTransactions` tests, which now exercise `Consolidate` indirectly.

- [ ] **Step 5: Commit**

```bash
git add internal/batch/aggregator.go internal/batch/aggregator_test.go
git commit -m "refactor: export Consolidate from BatchAggregator

The chronological sort and duplicate report were reachable only through
AggregateTransactions, which swallows per-file parse errors. BatchProcessor
needs both while keeping its own error capture for the manifest."
```

---

### Task 2: Give BatchProcessor a ParserResolver seam

`BatchProcessor` is built around one fixed parser, which cannot serve a directory of mixed formats. Replace the fixed parser with a resolver function supplied by the caller. Auto-detection and a `--from` pin are then the same code path with different closures, and package `batch` gains no dependency on `container`.

This task changes only the resolution seam. Output is still one CSV per file; Task 3 changes that. Splitting them keeps each diff reviewable.

**Files:**
- Modify: `internal/batch/processor.go`
- Modify: `cmd/common/convert.go` (call site of `NewBatchProcessor`)
- Test: `internal/batch/processor_test.go`

**Interfaces:**
- Consumes: `BatchAggregator.Consolidate` exists but is not used yet.
- Produces:
  - `type ParserResolver func(filePath string) (parser.FullParser, error)`
  - `func NewBatchProcessor(resolve ParserResolver, logger logging.Logger, fmt formatter.OutputFormatter, recursive bool) *BatchProcessor`
  - `func PinnedResolver(p parser.FullParser) ParserResolver`

- [ ] **Step 1: Write the failing test**

Append to `internal/batch/processor_test.go`:

```go
// The resolver is consulted per file, which is what lets one directory hold
// several formats. A file the resolver rejects is recorded as a failure and
// must not stop the files after it.
func TestProcessDirectory_ResolvesPerFile(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	writeSample(t, inputDir, "a.csv", "ok")
	writeSample(t, inputDir, "b.csv", "unsupported")
	writeSample(t, inputDir, "c.csv", "ok")

	var asked []string
	resolve := func(filePath string) (parser.FullParser, error) {
		asked = append(asked, filepath.Base(filePath))
		if strings.HasPrefix(filepath.Base(filePath), "b") {
			return nil, batch.ErrNoParser
		}
		return &stubParser{transactions: []models.Transaction{sampleTransaction()}}, nil
	}

	bp := batch.NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputDir)

	require.NoError(t, err)
	assert.Equal(t, []string{"a.csv", "b.csv", "c.csv"}, asked, "every file must be offered to the resolver")
	assert.Equal(t, 2, manifest.SuccessCount)
	assert.Equal(t, 1, manifest.FailureCount)
	assert.Equal(t, 1, manifest.ExitCode(), "partial success")
}

// PinnedResolver is what --from produces: the same parser for every file,
// so a batch the detector misreads can be forced through one parser.
func TestPinnedResolver_AlwaysReturnsSameParser(t *testing.T) {
	p := &stubParser{}
	resolve := batch.PinnedResolver(p)

	got1, err1 := resolve("/any/path.csv")
	got2, err2 := resolve("/other/file.xml")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Same(t, p, got1)
	assert.Same(t, p, got2)
}
```

`stubParser`, `writeSample` and `sampleTransaction` are helpers. If `processor_test.go` already defines equivalents, reuse them rather than adding near-duplicates. Otherwise add:

```go
type stubParser struct {
	transactions []models.Transaction
	parseErr     error
	invalid      bool
}

func (s *stubParser) Parse(_ context.Context, _ io.Reader) ([]models.Transaction, error) {
	if s.parseErr != nil {
		return nil, s.parseErr
	}
	return s.transactions, nil
}
func (s *stubParser) ValidateFormat(string) (bool, error)             { return !s.invalid, nil }
func (s *stubParser) ConvertToCSV(context.Context, string, string) error { return nil }
func (s *stubParser) SetLogger(logging.Logger)                        {}
func (s *stubParser) SetCategorizer(models.TransactionCategorizer)    {}

var _ parser.FullParser = (*stubParser)(nil)

func writeSample(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func sampleTransaction() models.Transaction {
	return models.Transaction{
		BookkeepingNumber: "1",
		Date:              time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Amount:            decimal.NewFromInt(10),
		PartyName:         "Coop",
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/batch/ -run 'TestProcessDirectory_ResolvesPerFile|TestPinnedResolver' -v`
Expected: FAIL — `cannot use resolve (variable of type func(string) (parser.FullParser, error)) as parser.FullParser value` and `undefined: batch.PinnedResolver`, `undefined: batch.ErrNoParser`.

- [ ] **Step 3: Write minimal implementation**

In `internal/batch/processor.go`:

```go
// ParserResolver returns the parser to use for one input file.
//
// It is the seam that lets a single directory hold several formats: the CLI
// supplies either a closure over Container.DetectParser (the default) or one
// returning a parser pinned by --from. Package batch therefore needs no
// knowledge of the container or of how formats are recognized.
type ParserResolver func(filePath string) (parser.FullParser, error)

// ErrNoParser reports that no parser accepts a file. Resolvers return it so a
// batch records the file as a failure and moves on.
var ErrNoParser = errors.New("no parser recognizes this file format")

// PinnedResolver returns a ParserResolver that hands back p for every file.
// This is what --from produces: an escape hatch for a batch the detector reads
// wrongly, not a filter that selects matching files. Files p cannot read fail
// individually and are recorded in the manifest.
func PinnedResolver(p parser.FullParser) ParserResolver {
	return func(string) (parser.FullParser, error) { return p, nil }
}
```

Change the struct field `parser parser.FullParser` to `resolve ParserResolver`, and the constructor's first argument to `resolve ParserResolver`. In `processFile`, replace the two `bp.parser.` uses with a resolution step at the top:

```go
	p, err := bp.resolve(filePath)
	if err != nil {
		result.Error = fmt.Sprintf("format_not_recognized: %v", err)
		bp.logger.WithError(err).Warn("Skipping file of unrecognized format",
			logging.Field{Key: "file", Value: fileName})
		return result
	}
```

then use `p.ValidateFormat` and `p.Parse` below it. Add `errors` to the import block.

Update the sole production call site, `cmd/common/convert.go`'s `FolderConvert`: it currently asserts `p.(parser.FullParser)` then calls `NewBatchProcessor(fullParser, …)`. Keep the assertion and pass `batch.PinnedResolver(fullParser)` instead.

- [ ] **Step 4: Run tests to verify they pass**

Run: `make test`
Expected: PASS. Pre-existing `processor_test.go` tests that construct a `BatchProcessor` need their first argument wrapped in `batch.PinnedResolver(...)` — do that mechanically; the assertions themselves do not change.

- [ ] **Step 5: Commit**

```bash
git add internal/batch/processor.go internal/batch/processor_test.go cmd/common/convert.go
git commit -m "refactor: resolve a parser per file in BatchProcessor

A fixed parser cannot serve a directory of mixed formats. The new
ParserResolver seam makes auto-detection and a --from pin the same code
path, and keeps package batch free of any container dependency."
```

---

### Task 3: One consolidated CSV per directory

Make `ProcessDirectory` write a single file. Per-file work becomes parse-only; the accumulated transactions go through `Consolidate` and are written once. The manifest moves next to the output file.

**Files:**
- Modify: `internal/batch/processor.go`
- Test: `internal/batch/processor_test.go`

**Interfaces:**
- Consumes: `BatchAggregator.Consolidate` (Task 1), `ParserResolver` (Task 2).
- Produces:
  - `func (bp *BatchProcessor) ProcessDirectory(ctx context.Context, inputDir, outputFile string) (*BatchManifest, error)` — `outputFile` is now a file path, not a directory.
  - `func ManifestPathFor(outputFile string) string` — replaces the output's extension with `.manifest.json`.

- [ ] **Step 1: Write the failing test**

Append to `internal/batch/processor_test.go`:

```go
// A directory always yields one CSV. Files of different formats are merged into
// it and ordered by date, which is the whole point: drop every statement into a
// folder, get one import file.
func TestProcessDirectory_WritesSingleSortedCSV(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputFile := filepath.Join(t.TempDir(), "releves.csv")

	writeSample(t, inputDir, "march.csv", "x")
	writeSample(t, inputDir, "january.csv", "x")

	march := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	january := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	resolve := func(filePath string) (parser.FullParser, error) {
		when := march
		if strings.HasPrefix(filepath.Base(filePath), "january") {
			when = january
		}
		return &stubParser{transactions: []models.Transaction{{
			BookkeepingNumber: filepath.Base(filePath),
			Date:              when,
			ValueDate:         when,
			Amount:            decimal.NewFromInt(1),
			PartyName:         "Coop",
		}}}, nil
	}

	bp := batch.NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.Equal(t, 2, manifest.SuccessCount)
	assert.FileExists(t, outputFile)

	// Exactly one CSV, and January's row precedes March's inside it.
	body, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Less(t, strings.Index(string(body), "january.csv"),
		strings.Index(string(body), "march.csv"),
		"rows must be ordered by date, not by filename")

	entries, err := os.ReadDir(filepath.Dir(outputFile))
	require.NoError(t, err)
	var csvCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".csv") {
			csvCount++
		}
	}
	assert.Equal(t, 1, csvCount, "a directory input must produce exactly one CSV")
}

// One unreadable file out of many must not discard the rest: the CSV is written
// with what succeeded, the exit code reports partial success, and the manifest
// names the failure so the user knows what to re-run.
func TestProcessDirectory_PartialFailureStillWritesCSV(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	writeSample(t, inputDir, "good.csv", "x")
	writeSample(t, inputDir, "bad.csv", "x")

	resolve := func(filePath string) (parser.FullParser, error) {
		if strings.HasPrefix(filepath.Base(filePath), "bad") {
			return &stubParser{parseErr: errors.New("corrupt header")}, nil
		}
		return &stubParser{transactions: []models.Transaction{sampleTransaction()}}, nil
	}

	bp := batch.NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.FileExists(t, outputFile, "the successes must still be written")
	assert.Equal(t, 1, manifest.ExitCode(), "partial success")

	var failed *batch.BatchResult
	for i := range manifest.Results {
		if !manifest.Results[i].Success {
			failed = &manifest.Results[i]
		}
	}
	require.NotNil(t, failed)
	assert.Equal(t, "bad.csv", failed.FileName, "the manifest must name the failure")
}

// The manifest replaces the output extension rather than appending to it, and
// lands beside the CSV: there is no output directory to hold it any more.
func TestProcessDirectory_WritesManifestBesideOutput(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outDir := t.TempDir()
	outputFile := filepath.Join(outDir, "releves.csv")

	writeSample(t, inputDir, "a.csv", "x")
	resolve := batch.PinnedResolver(&stubParser{transactions: []models.Transaction{sampleTransaction()}})

	bp := batch.NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), false)
	_, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(outDir, "releves.manifest.json"))
	assert.NoFileExists(t, filepath.Join(outDir, ".manifest.json"), "the old fixed name must be gone")
}

func TestManifestPathFor(t *testing.T) {
	assert.Equal(t, "/out/releves.manifest.json", batch.ManifestPathFor("/out/releves.csv"))
	assert.Equal(t, "/out/noext.manifest.json", batch.ManifestPathFor("/out/noext"))
}

// --recursive is about which files are read, not about the shape of the output:
// a whole tree still lands in the one CSV.
func TestProcessDirectory_RecursiveMergesWholeTree(t *testing.T) {
	logger := logging.NewMockLogger()
	inputDir := t.TempDir()
	nested := filepath.Join(inputDir, "2024", "q1")
	require.NoError(t, os.MkdirAll(nested, 0750))

	writeSample(t, inputDir, "top.csv", "x")
	writeSample(t, nested, "deep.csv", "x")

	outputFile := filepath.Join(t.TempDir(), "all.csv")
	resolve := batch.PinnedResolver(&stubParser{transactions: []models.Transaction{sampleTransaction()}})

	bp := batch.NewBatchProcessor(resolve, logger, formatter.NewStandardFormatter(), true)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	assert.Equal(t, 2, manifest.SuccessCount, "the nested file must be read too")
	assert.FileExists(t, outputFile)

	entries, err := os.ReadDir(filepath.Dir(outputFile))
	require.NoError(t, err)
	var csvCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".csv") {
			csvCount++
		}
	}
	assert.Equal(t, 1, csvCount, "recursion must not multiply outputs")
}

// An empty directory is a failed run, not a silent success: exit code 2 tells
// the user nothing was converted.
func TestProcessDirectory_EmptyDirectoryExitsTwo(t *testing.T) {
	logger := logging.NewMockLogger()
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	bp := batch.NewBatchProcessor(batch.PinnedResolver(&stubParser{}), logger,
		formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), t.TempDir(), outputFile)

	require.NoError(t, err)
	assert.Equal(t, 2, manifest.ExitCode())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/batch/ -run TestProcessDirectory -v`
Expected: FAIL — `undefined: batch.ManifestPathFor`, and the single-CSV assertion fails because the current code writes one CSV per input.

- [ ] **Step 3: Write minimal implementation**

In `internal/batch/processor.go`:

Add the manifest path helper:

```go
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
```

Rename `processFile` to `parseFile` and change it to return the transactions alongside the result, dropping every write:

```go
// parseFile resolves a parser for filePath and returns the transactions it
// yields. It never writes: a batch produces one consolidated CSV, written once
// by ProcessDirectory after every file has been read.
//
// This method never panics and captures all errors in the returned result.
func (bp *BatchProcessor) parseFile(ctx context.Context, filePath string) ([]models.Transaction, BatchResult) {
```

Keep the existing validation, open, close and parse steps verbatim. Delete the `os.MkdirAll` and `WriteTransactionsToCSVWithFormatter` block at the end, and finish with:

```go
	result.Success = true
	result.RecordCount = len(transactions)
	return transactions, result
```

Every early return becomes `return nil, result`.

Delete `outputPathFor` and the `claimed` map entirely — output names no longer exist.

Rewrite `ProcessDirectory`:

```go
// ProcessDirectory reads every file under inputDir and writes their merged
// transactions to the single CSV at outputFile, plus a run report beside it.
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

	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("input directory does not exist: %s", inputDir)
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	files, err := bp.discoverFiles(inputDir)
	if err != nil {
		return nil, err
	}

	bp.logger.Info("Starting batch processing",
		logging.Field{Key: "input_dir", Value: inputDir},
		logging.Field{Key: "output_file", Value: outputFile},
		logging.Field{Key: "files_found", Value: len(files)})

	manifest := &BatchManifest{
		TotalFiles:  len(files),
		Results:     make([]BatchResult, 0, len(files)),
		ProcessedAt: time.Now(),
	}

	var merged []models.Transaction

	for _, filePath := range files {
		select {
		case <-ctx.Done():
			bp.logger.Warn("Batch processing cancelled",
				logging.Field{Key: "processed", Value: len(manifest.Results)},
				logging.Field{Key: "total", Value: manifest.TotalFiles})
			manifest.Duration = time.Since(startTime)
			return manifest, ctx.Err()
		default:
		}

		transactions, result := bp.parseFile(ctx, filePath)
		manifest.Results = append(manifest.Results, result)

		if result.Success {
			manifest.SuccessCount++
			merged = append(merged, transactions...)
		} else {
			manifest.FailureCount++
		}
	}

	// Consolidation is skipped for an empty batch: writing a header-only CSV
	// would look like a successful conversion of nothing.
	if len(merged) > 0 {
		merged = NewBatchAggregator(bp.logger).Consolidate(merged, filepath.Base(outputFile))

		delimiter := bp.formatter.Delimiter()
		if err := common.WriteTransactionsToCSVWithFormatter(
			merged, outputFile, bp.logger, bp.formatter, delimiter); err != nil {
			return manifest, fmt.Errorf("failed to write consolidated CSV: %w", err)
		}
	}

	manifest.Duration = time.Since(startTime)

	bp.logger.Info("Batch processing completed",
		logging.Field{Key: "total_files", Value: manifest.TotalFiles},
		logging.Field{Key: "success", Value: manifest.SuccessCount},
		logging.Field{Key: "failed", Value: manifest.FailureCount},
		logging.Field{Key: "transactions", Value: len(merged)},
		logging.Field{Key: "duration", Value: manifest.Duration.String()})

	manifestPath := ManifestPathFor(outputFile)
	if err := manifest.WriteManifest(manifestPath); err != nil {
		bp.logger.WithError(err).Warn("Failed to write manifest file",
			logging.Field{Key: "path", Value: manifestPath})
	} else {
		bp.logger.Info("Wrote batch manifest",
			logging.Field{Key: "path", Value: manifestPath})
	}

	return manifest, nil
}
```

Add `fjacquet/camt-csv/internal/models` to the imports if it is not already there.

`discoverFiles` keeps skipping hidden entries, which is what stopped a previous run's `.manifest.json` from becoming an input. The new manifest name is **not** hidden, so a second run over the same folder would try to parse `releves.manifest.json` — that is exactly what Task 5's output-under-input guard prevents at the CLI, and no validator accepts JSON, so it would be recorded as a skipped file rather than corrupt anything.

- [ ] **Step 4: Run tests to verify they pass**

Run: `make test`
Expected: PASS. Pre-existing `processor_test.go` tests asserting per-file output paths no longer describe the design — delete them, do not weaken them. `cmd/common/convert_test.go`'s `TestFolderConvert_EmptyDirectory` asserts `.manifest.json` inside the output dir; it is updated in Task 4 along with `FolderConvert` itself. If it fails here, leave it red and note it; Task 4 fixes it.

- [ ] **Step 5: Commit**

```bash
git add internal/batch/processor.go internal/batch/processor_test.go
git commit -m "feat: consolidate a directory into a single CSV

A directory input now merges into one date-sorted CSV rather than mirroring
the input tree. Per-file work is parse-only; the manifest lands beside the
output. Partial failures still write the successes."
```

---

### Task 4: Wire --from and the single-file output into the CLI

Add the input-format flag, promote `--keep-payments`, and rewrite `cmd/convert` as the sole handler. This is where the two remaining guards live: `-o` naming when given an existing directory, and the refusal to write under the input directory.

**Files:**
- Modify: `cmd/common/flags.go`
- Modify: `cmd/common/convert.go`
- Modify: `cmd/convert/convert.go`
- Test: `cmd/convert/convert_test.go`
- Test: `cmd/common/convert_test.go`

**Interfaces:**
- Consumes: `batch.ParserResolver`, `batch.PinnedResolver`, `batch.ErrNoParser`, `batch.ManifestPathFor`, the new `ProcessDirectory` signature.
- Produces:
  - `func common.RegisterConvertFlags(cmd *cobra.Command)` — replaces `RegisterFormatFlags`, adding `--from` and `--keep-payments`.
  - `func common.ResolverFor(c *container.Container, from string) (batch.ParserResolver, error)` — pinned when `from` is set, detecting otherwise.
  - `func common.ResolveOutputFile(inputPath, outputPath string) (string, error)` — applies the existing-directory naming rule and the under-input refusal.

- [ ] **Step 1: Write the failing test**

Replace the contents of `cmd/convert/convert_test.go` — `TestSameDirectory`, `TestDiscoverInputs_*` and `TestOutputPathFor_*` all test functions this task deletes. Write:

```go
// --from must offer exactly the registered parser types, so the flag cannot
// drift from what the container can actually build.
func TestResolverFor_AcceptsEveryDetectableType(t *testing.T) {
	c := newTestContainer(t)

	for _, pt := range container.DetectionOrder() {
		resolve, err := common.ResolverFor(c, string(pt))
		require.NoError(t, err, "--from %s must be accepted", pt)

		p, err := resolve("/irrelevant/path.csv")
		require.NoError(t, err)
		assert.NotNil(t, p)
	}
}

func TestResolverFor_RejectsUnknownFormat(t *testing.T) {
	c := newTestContainer(t)

	_, err := common.ResolverFor(c, "postbank")

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

	resolve, err := common.ResolverFor(c, "selma")
	require.NoError(t, err)

	p, err := resolve(revolutFile)
	require.NoError(t, err, "the pin resolves regardless of content")

	valid, _ := p.ValidateFormat(revolutFile)
	assert.False(t, valid, "the pinned Selma parser must not accept a Revolut file")
}

// With no --from, resolution falls back to detection and an unrecognized file
// is reported as such rather than guessed at.
func TestResolverFor_DetectsWhenUnset(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	unknown := filepath.Join(dir, "mystery.csv")
	require.NoError(t, os.WriteFile(unknown, []byte("a,b,c\n1,2,3\n"), 0600))

	resolve, err := common.ResolverFor(c, "")
	require.NoError(t, err)

	_, err = resolve(unknown)
	assert.ErrorIs(t, err, batch.ErrNoParser)
}

// A single-file conversion's outcome is its exit code. A one-entry run report
// adds nothing, so none is written.
func TestConvert_SingleFileWritesNoManifest(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(input, []byte(revolutSampleCSV), 0600))
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	err := common.ProcessFileWithErrorFormatted(context.Background(),
		mustResolve(t, c, "", input), input, outputFile, false, c.GetLogger(), c, "standard")

	require.NoError(t, err)
	assert.FileExists(t, outputFile)
	assert.NoFileExists(t, batch.ManifestPathFor(outputFile))
}

// mustResolve returns the parser one run would use for path, failing the test
// if resolution errors.
func mustResolve(t *testing.T, c *container.Container, from, path string) parser.FullParser {
	t.Helper()
	resolve, err := common.ResolverFor(c, from)
	require.NoError(t, err)
	p, err := resolve(path)
	require.NoError(t, err)
	return p
}

// -o names a file. Pointing it at an existing directory is a convenience the
// PDF command used to offer, and it survives: the folder being read names the
// output written inside it.
func TestResolveOutputFile_ExistingDirectoryGetsGeneratedName(t *testing.T) {
	inputDir := filepath.Join(t.TempDir(), "releves-2024")
	require.NoError(t, os.MkdirAll(inputDir, 0750))
	outputDir := t.TempDir()

	got, err := common.ResolveOutputFile(inputDir, outputDir)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "releves-2024.csv"), got)
}

func TestResolveOutputFile_PlainPathIsUnchanged(t *testing.T) {
	inputDir := t.TempDir()
	want := filepath.Join(t.TempDir(), "out.csv")

	got, err := common.ResolveOutputFile(inputDir, want)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// Writing inside the folder being read means a later --recursive run reads its
// own output back as input. Refuse it rather than rely on no validator ever
// accepting our own CSV.
func TestResolveOutputFile_RefusesOutputUnderInput(t *testing.T) {
	inputDir := t.TempDir()
	nested := filepath.Join(inputDir, "sub")
	require.NoError(t, os.MkdirAll(nested, 0750))

	_, err := common.ResolveOutputFile(inputDir, filepath.Join(nested, "out.csv"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "input")
}
```

Add a `newTestContainer` helper building a container from a default config with `TEST_MODE=true` set via `t.Setenv`, and a `revolutSampleCSV` constant. Copy the Revolut header from `internal/container/detect_test.go` — its constants are unexported and in another package, so duplicating the one you need is correct here.

Append to `cmd/common/convert_test.go`, replacing `TestFolderConvert_EmptyDirectory`'s manifest assertion:

```go
// The run report lands beside the output file now that there is no output
// directory to hold it.
func TestFolderConvert_WritesManifestBesideOutput(t *testing.T) {
	mockLogger := logging.NewMockLogger()
	inputDir := t.TempDir()
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	restore := common.SetExitFn(func(int) {})
	defer restore()

	common.FolderConvert(context.Background(), &convertMockParser{}, inputDir, outputFile,
		mockLogger, "standard", false)

	assert.FileExists(t, batch.ManifestPathFor(outputFile))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/... -v`
Expected: FAIL — `undefined: common.ResolverFor`, `undefined: common.ResolveOutputFile`.

- [ ] **Step 3: Write minimal implementation**

In `cmd/common/flags.go`, rename `RegisterFormatFlags` to `RegisterConvertFlags` and add the two flags:

```go
// RegisterConvertFlags adds the conversion flags to a command.
func RegisterConvertFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("format", "f", "",
		"Output format: icompta (iCompta-compatible), standard (29-column comma-delimited CSV), or jumpsoft (7-column Jumpsoft Money CSV). Default: icompta (overridable via CAMT_OUTPUT_FORMAT env var)")
	cmd.Flags().String("from", "",
		"Input format, bypassing auto-detection: "+strings.Join(parserTypeNames(), ", ")+
			". On a directory this pins every file to that parser; files it cannot read fail individually")
	cmd.Flags().Bool("recursive", false,
		"When the input is a directory, also process files in its subdirectories")
	// Read back by root.applyFlagOverrides; a viper.BindPFlag would be ignored
	// because InitializeConfig builds its own Viper instance.
	cmd.Flags().Bool("keep-payments", false,
		"Import the monthly Viseca card settlement rows (dropped by default because the bank statement carries the same payments)")
	cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
	// The flag has never been wired to the writers: output dates are always
	// rendered with models.DateFormatCSV. Keep it registered so existing
	// invocations do not break, but tell the user it does nothing.
	if err := cmd.Flags().MarkDeprecated("date-format", "it has no effect; output dates are always DD.MM.YYYY"); err != nil {
		panic(err)
	}
}

// parserTypeNames lists the detectable formats for flag help and error messages.
func parserTypeNames() []string {
	types := container.DetectionOrder()
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	return names
}
```

Move `parserTypeNames` here from `cmd/convert/convert.go` and delete it there.

Add to `cmd/common/convert.go`:

```go
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
```

Change `FolderConvert`'s `outputDir` parameter to `outputFile` and pass `batch.PinnedResolver(fullParser)`; its doc comment should say it writes one CSV. Delete `RunConvert` entirely — Task 5 removes its last callers, and leaving it would keep eight dead packages compiling.

Rewrite `cmd/convert/convert.go`'s `runConvert` to: read `--from`, build the resolver via `ResolverFor`, `os.Stat` the input, and branch. The directory branch calls `ResolveOutputFile` then a `BatchProcessor` built from the resolver. The single-file branch resolves one parser and calls `common.ProcessFile`. Delete `convertDirectory`, `discoverInputs`, `outputPathFor`, `sameDirectory` and `parserTypeNames` from that file — every one is now duplicated by `BatchProcessor` or `cmd/common`.

Update the command's `Long` text: it currently promises "every file in it is detected and converted independently" and points at the format-specific commands. Both statements become false.

- [ ] **Step 4: Run tests to verify they pass**

Run: `make test && make lint`
Expected: PASS. The eight format command packages still compile because they call `common.RegisterFormatFlags` — rename those call sites to `RegisterConvertFlags` mechanically, or delete the packages now as Task 5 does.

- [ ] **Step 5: Commit**

```bash
git add cmd/common/flags.go cmd/common/convert.go cmd/convert/convert.go cmd/convert/convert_test.go cmd/common/convert_test.go
git commit -m "feat: add --from and single-file output to convert

--from pins a parser for a batch the detector reads wrongly. -o now always
names a file, generating a name inside an existing directory, and refuses to
sit under the input directory."
```

---

### Task 5: Delete the eight format commands

**Files:**
- Delete: `cmd/camt/`, `cmd/pdf/`, `cmd/selma/`, `cmd/viseca/`, `cmd/revolut/`, `cmd/revolut-crypto/`, `cmd/revolut-investment/`, `cmd/debit/`
- Modify: `main.go:52-61`
- Modify: `cmd/root/root.go`
- Test: `cmd/convert/convert_test.go`, `cmd/root/root_test.go`

**Interfaces:**
- Consumes: everything from Tasks 3 and 4.
- Produces: no new symbols.

- [ ] **Step 1: Write the failing test**

Append to `cmd/convert/convert_test.go`:

```go
// The behavior the pdf command used to own — a directory of PDFs consolidated
// into one chronologically sorted CSV — must survive its deletion. This is the
// regression guard for that removal.
func TestConvert_PDFDirectoryConsolidates(t *testing.T) {
	// Skipped where poppler-utils is absent: the PDF parser shells out to pdftotext.
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed")
	}

	c := newTestContainer(t)
	inputDir := "../../samples/pdf"
	outputFile := filepath.Join(t.TempDir(), "consolidated.csv")

	resolve, err := common.ResolverFor(c, "pdf")
	require.NoError(t, err)

	bp := batch.NewBatchProcessor(resolve, c.GetLogger(), formatter.NewStandardFormatter(), false)
	manifest, err := bp.ProcessDirectory(context.Background(), inputDir, outputFile)

	require.NoError(t, err)
	require.Positive(t, manifest.SuccessCount, "at least one sample PDF must convert")
	assert.FileExists(t, outputFile)
}
```

Append to `cmd/root/root_test.go`:

```go
// The root namespace holds verbs only, grouped so the primary function and the
// diagnostic do not read as peers.
func TestRootCommand_HasOnlyConvertAndCategorize(t *testing.T) {
	var names []string
	for _, c := range root.Cmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		names = append(names, c.Name())
	}
	sort.Strings(names)

	assert.Equal(t, []string{"categorize", "convert"}, names)
}

func TestRootCommand_RemovedFormatCommandsAreGone(t *testing.T) {
	removed := []string{"camt", "pdf", "selma", "viseca", "revolut",
		"revolut-crypto", "revolut-investment", "debit"}

	for _, name := range removed {
		for _, c := range root.Cmd.Commands() {
			assert.NotEqual(t, name, c.Name(), "%s must be gone; use convert --from %s", name, name)
		}
	}
}

func TestRootCommand_GroupsSeparatePrimaryFromAccessory(t *testing.T) {
	byName := map[string]*cobra.Command{}
	for _, c := range root.Cmd.Commands() {
		byName[c.Name()] = c
	}

	require.Contains(t, byName, "convert")
	require.Contains(t, byName, "categorize")
	assert.Equal(t, "conversion", byName["convert"].GroupID)
	assert.Equal(t, "tools", byName["categorize"].GroupID)
}

// The tool has handled seven formats since long before this change; the help
// text still described it as a CAMT.053 converter.
func TestRootCommand_ShortDescriptionIsNotCAMTOnly(t *testing.T) {
	assert.NotContains(t, root.Cmd.Short, "CAMT.053 XML files to CSV",
		"the tool converts seven formats, not one")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/root/ ./cmd/convert/ -v`
Expected: FAIL — the removed commands are still registered, and `GroupID` is empty on both survivors.

- [ ] **Step 3: Write minimal implementation**

Delete the eight directories:

```bash
git rm -r cmd/camt cmd/pdf cmd/selma cmd/viseca cmd/revolut cmd/revolut-crypto cmd/revolut-investment cmd/debit
```

In `main.go`, drop the eight imports and reduce the registration block to:

```go
	root.Cmd.AddCommand(convert.Cmd)
	root.Cmd.AddCommand(categorize.Cmd)
```

In `cmd/root/root.go`, register the groups inside `Init()` before any `AddCommand` runs:

```go
	Cmd.AddGroup(
		&cobra.Group{ID: "conversion", Title: "Conversion:"},
		&cobra.Group{ID: "tools", Title: "Tools:"},
	)
```

Set `GroupID: "conversion"` on `convert.Cmd` and `GroupID: "tools"` on `categorize.Cmd` in their respective `cobra.Command` literals.

Correct the root description:

```go
		Short: "Convert bank and broker statements to CSV, with automatic transaction categorization.",
		Long: `camt-csv converts financial statements to CSV for import into accounting software.

It reads CAMT.053 XML, PDF statements, and the Revolut, Revolut Crypto, Revolut
Investment, Selma and Visa Debit CSV exports, detecting the format automatically.
Transactions are categorized from the party's name.`,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `make test && make lint && make build`
Expected: PASS, and `./camt-csv --help` shows the two groups with no format commands.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat!: collapse eight format commands into convert --from

Eight tenths of the root namespace were one concept, and mixed format nouns
with verbs. Auto-detection already covers every format; pinning one is now a
flag. Help output groups the primary function apart from the diagnostic.

BREAKING CHANGE: camt, pdf, selma, viseca, revolut, revolut-crypto,
revolut-investment and debit are removed. Use convert, with --from <format>
when detection must be bypassed."
```

---

### Task 6: Documentation and changelog

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/user-guide.md`, `docs/developer-guide.md`, `docs/architecture.md`, `docs/index.md`

**Interfaces:**
- Consumes: the finished CLI from Task 5.
- Produces: no code.

- [ ] **Step 1: Find every stale invocation**

Run: `grep -rn "camt-csv \(camt\|pdf\|selma\|viseca\|revolut\|revolut-crypto\|revolut-investment\|debit\) " --include="*.md" .`

Every hit becomes `camt-csv convert -i … -o …`, adding `--from <format>` only where the example is specifically about pinning a format.

- [ ] **Step 2: Update CHANGELOG.md**

Under `## [Unreleased]`, in imperative mood:

```markdown
### Removed

- Remove the `camt`, `pdf`, `selma`, `viseca`, `revolut`, `revolut-crypto`,
  `revolut-investment` and `debit` commands. Use `convert`, which detects the
  format, with `--from <format>` to pin one.

### Changed

- **BREAKING:** A directory input now produces a single consolidated CSV sorted
  by date, instead of one CSV per input file. `--output` always names a file;
  pointing it at an existing directory generates a name inside it.
- Write the batch run report beside the output (`releves.manifest.json`) rather
  than as a fixed `.manifest.json` inside the output directory.
- Group the help output, separating conversion from diagnostic tools.
- Correct the root command description, which described the tool as a CAMT.053
  converter three formats after it stopped being one.

### Added

- Add `--from <format>` to `convert`, pinning the parser and bypassing
  auto-detection.
- Promote `--keep-payments` from the removed `viseca` command to `convert`.

### Fixed

- Refuse an `--output` path inside the input directory, where a later
  `--recursive` run would read its own output back as input.
- Anchor the `specs/` ignore rule to the repository root; a bare pattern also
  matched nested `specs/` directories.
```

- [ ] **Step 3: Update CLAUDE.md**

In "Adding a New Parser", delete step 5 ("Add CLI command in `cmd/{name}/convert.go` — delegate to `common.RunConvert`") and step 6's wiring in `main.go`; renumber. Both describe a structure that no longer exists. Add a line to the Format Detection paragraph noting that `detectionOrder` now also defines the `--from` values, so a parser missing from it is unreachable by any route.

In the Output Formatter Registry paragraph, correct the CLI usage line: directory input produces one consolidated CSV.

- [ ] **Step 4: Verify the docs match the binary**

Run: `make build && ./camt-csv --help && ./camt-csv convert --help`
Compare against every example touched. Fix any drift.

- [ ] **Step 5: Commit**

```bash
git add CHANGELOG.md README.md CLAUDE.md docs/
git commit -m "docs: update for the two-command CLI

Every camt-csv <format> example becomes convert. The Adding a New Parser
steps lose the per-format command they no longer need."
```

---

## Verification before opening the PR

- [ ] `make lint` clean
- [ ] `make test` green, race detector included
- [ ] `make security` clean (gosec)
- [ ] `./camt-csv --help` shows exactly `convert` and `categorize`, under `Conversion:` and `Tools:`
- [ ] `./camt-csv convert -i samples/camt053 -o /tmp/out.csv` writes one CSV plus `/tmp/out.manifest.json`
- [ ] `./camt-csv convert -i samples/ -o /tmp/all.csv --recursive` merges mixed formats into one date-sorted CSV
- [ ] `./camt-csv convert -i samples/camt053 -o samples/camt053/out.csv` is refused
- [ ] `./camt-csv convert -i samples/revolut.csv -o /tmp/r.csv --from selma` fails loudly rather than falling back
- [ ] Import one output into iCompta and confirm the columns still resolve — `TestIComptaHeaderCoversPluginMappings` guards the header, but only a real import proves the mapping
- [ ] Open the PR against `main` (protected); title `feat!: collapse the CLI to convert and categorize`
