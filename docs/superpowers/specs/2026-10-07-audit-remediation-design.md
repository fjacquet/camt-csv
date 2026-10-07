# Audit remediation — design

Date: 2026-10-07. Source: whole-codebase audit (security, performance/AI cost, code
quality) run on `main` at v4.1.0. Baseline tools: govulncheck clean, semgrep clean,
gosec only G304 on CLI-supplied paths (trusted).

## Goal

Cut AI calls and YAML churn on monthly statement runs, remove dead and duplicated
code, close the CSV formula-injection gap, without changing amounts or row counts of
any conversion.

## Decisions (user, 2026-10-07)

1. Formula escaping applies to `standard` and `jumpsoft` output only; `icompta`
   output stays raw (iCompta never evaluates formulas; a `'` would pollute imported names).
2. Staged AI suggestions are reused as a read-only cache tier; promotion to
   `creditors.yaml`/`debtors.yaml` stays manual.
3. AI requests are batched, 25 parties per prompt, JSON answer.
4. Parsers unify on `common.ProcessTransactionsWithCategorizationStats` behavior:
   date `2006-01-02`, `info` = description.

## Delivery

Six PRs, merged in order A → F, each with its own tests and CHANGELOG entry.
Release v4.2.0 after F.

### PR A — dead code and silent CAMT fallback

Delete (each confirmed by a repo-wide grep showing no non-test reference):

- `internal/models/models.go` compliance types (`ComplianceReport`, `Finding`,
  `CorrectiveAction`, `NewComplianceReport`) and `internal/parser/constitution.go`.
- `internal/batch/aggregator.go`: everything not reachable from `Consolidate`
  (`GroupFilesByAccount`, `extractDateRangeFromFilename`, `AggregateTransactions`,
  `GenerateOutputFilename`, `GenerateSourceFileHeader`,
  `CalculateDateRangeFromTransactions`, and types only they use).
- `common.ExtractAccountFromFilename`, `common.ExtractAccountFromCAMTFilename`.
- `common.ReadCSVFile`, `common.WriteTransactionsToCSV`, and
  `common.WriteTransactionsToCSVWithLogger` (its only production caller is
  `WriteTransactionsToCSV`).
- The unused `constitution.*` config keys in `internal/config/viper.go`.
- `debitparser.ParseFile`, `debitparser.ParseFileWithLogger`.
- Test-only methods: `Transaction.SetAmountFromDecimal`, `SetFeesFromDecimal`,
  `GetAmountAsDecimal`; `Categorizer.UpdateCreditorCategory`,
  `UpdateDebitorCategory`; `KeywordStrategy.ReloadCategories`;
  `DirectMappingStrategy.ReloadMappings`.
- Tests covering only the deleted code.

A symbol is kept if the grep finds any production reference; the PR lists each kept
candidate and why.

`internal/camtparser/entry_mapping.go`: when `builder.Build()` fails, skip the entry,
log a Warn with its entry reference, and do not emit the fallback row. (The fallback
reuses the same date, amount and currency, so it almost always fails too and today
emits a zero-valued `Transaction{}`.) Test: an entry that cannot build produces no row and the
rest of the statement converts.

### PR B — AI cost and AI answer validation

All in `internal/categorizer` and `internal/store`.

- **Learn only from AI.** Auto-learn updates the in-memory mapping only when
  `category.Source == "ai"`; it no longer saves per transaction. The existing single
  save in `PersistentPostRun` (`cmd/root/root.go`) persists it.
- **Stage only from AI, write once.** Staging suggestions are recorded only for
  `Source == "ai"`, buffered in memory, and written once when the run ends
  (same hook that saves mappings). Existing staged entries are preserved (merge,
  not overwrite).
- **Staged tier.** New strategy `staged`, loaded read-only from the staging files at
  startup, ordered after `direct` and `keyword`, before `semantic` and `ai`. It
  answers exact (case-insensitive, trimmed) party-name matches, split by
  debtor/creditor like the direct tier. It never writes.
- **Negative caching.** The in-run cache stores Uncategorized results and AI failures
  too, so each `(party, isDebtor)` pair reaches the AI at most once per run.
- **Validate AI answers.** An AI category is accepted only if it matches a category
  name in `categories.yaml` (case-insensitive; the canonical spelling is returned).
  Otherwise the result is Uncategorized and a Warn names the rejected answer.

Tests (counting mock `AIClient`):
- second run over the same merchants makes 0 AI calls (staged tier);
- a merchant appearing 3 times with an Uncategorized answer makes 1 call;
- an answer not in `categories.yaml` becomes Uncategorized;
- auto-learn on: direct/keyword hits do not modify mappings; YAML is written once per run;
- staging: only AI answers are staged, file written once, prior entries kept.

### PR C — one categorization loop

`debitparser`, `revolutparser`, `revolutcryptoparser`, `revolutinvestmentparser` and
`camtparser` drop their inline categorize loops and call the shared helper after
building transactions. The helper takes a party-name function:
`ProcessTransactionsWithCategorizationStats` keeps its signature and uses
`DefaultPartyName` (`GetPartyName` → `PartyName` → `Name` → `Recipient` →
`Description`; the `Description` step is new and is what the debit parser needs);
`ProcessTransactionsWithPartyName` takes an explicit function. CAMT passes its own
(`PartyName` → `Description` → `RemittanceInfo`, then `cleanPaymentMethodPrefixes`), so
its output names are unchanged. The helper uses
`models.CategoryUncategorized` instead of the `"Uncategorized"` literal.

Accepted behavior change: those five parsers now send the date as `2006-01-02` and
the description as `info`, like viseca, pdf and selma.

Tests: existing parser tests pass; a recording mock categorizer shows each of the
eight parsers' Categorize calls carry the ISO date and the description.

### PR D — batched AI requests

- `models.BatchCategorizer` (optional): `CategorizeBatch(ctx, []CategorizeRequest)
  ([]Category, error)`. `Categorizer` implements it.
- `categorizer.BatchAIClient` (optional): `CategorizeBatch(ctx, []models.Transaction)
  (map[string]string, error)`, implemented in `baseAIClient` and used by the Gemini
  and OpenRouter clients.
- Flow in `common.ProcessTransactionsWithCategorizationStats`: if the categorizer
  implements `BatchCategorizer`, use it; otherwise the current per-transaction path.
- `Categorizer.CategorizeBatch`: run cache + local tiers (direct, keyword, staged,
  semantic) per request; collect distinct misses by `(party, isDebtor)`; send them in
  chunks of 25 with a prompt asking for a JSON object `{"<party>": "<category>"}`;
  validate each answer (PR B rule). Answers are matched back to requests by party
  name lowercased and trimmed (the cache key); a party missing from the answer, or an
  unparseable response, falls back to the single-request path. Results go through
  the same cache, learning and staging as single requests.
- One chunk counts as one request against `ai.requests_per_minute`.
- `recategorize` keeps per-row calls (out of scope).

Tests: 60 distinct misses → 3 AI calls; a party omitted from the JSON → 1 extra single
call; malformed JSON → every party in the chunk falls back; JSON parsing tolerates a
fenced ```json block.

### PR E — formula injection

- Move `escapeCell`/`needsEscape`/`unescapeCell` from `internal/icompta/report.go`
  to a new leaf package `internal/csvsafe` (`Escape`, `Unescape`, `NeedsEscape`);
  `internal/common` cannot host them because it imports `formatter`.
- `standard` and `jumpsoft` formatters escape free-text columns (name, party name,
  description, remittance info, recipient, category, type, fund). Numeric and date
  columns are never escaped.
- `icompta` formatter: unchanged (raw).
- `icompta/report.go`: also escape `SplitID` and `Date`.

Tests: a description `=HYPERLINK("x")` gets a leading `'` in standard and jumpsoft,
stays raw in icompta; `-180.00` amounts unchanged in all three; report escapes all
text columns and round-trips through `unescapeCell`.

### PR F — housekeeping

- Remove the `init()` setting `gocsv.TagSeparator` in `internal/common/csv.go` if no
  struct tag relies on it (verify by tests); otherwise document why it stays.
- Drop the package var `extractTextFromPDF`; use the injected `PDFExtractor`.
  Use the in-scope logger instead of `getDefaultLogger()` (pdfparser_helpers.go).
- PDF: wrap the extractor in a cache keyed by the SHA-256 of the PDF bytes, so
  `ValidateFormat` (original path) and `Parse` (temp copy) run `pdftotext` once.
- Semantic warm-up starts lazily on the first semantic lookup (so commands that never
  categorize never warm up), and that lookup waits for it, bounded by ctx, instead of
  skipping to AI.
- `revolutinvestmentparser`: local `parseDecimalField(name, raw)` replaces the 7
  repeated blocks.
- `common.MissingColumn(header []string, required ...string) string` (first missing
  name, or ""; header names trimmed and BOM-stripped) replaces the set checks in the
  debit, revolut, selma, viseca and revolut-crypto validators. revolut-investment
  checks headers by position and keeps its own check.
- One "format not recognized" sentinel, `parser.ErrFormatNotRecognized`;
  `container.ErrFormatNotRecognized` and `batch.ErrNoParser` become aliases of it.
- Split `internal/pdfparser/pdfparser_helpers.go` into `extract.go` (pdftotext I/O),
  `viseca_pdf.go`, `text.go`; rename the local `min` shadowing the builtin.

Deferred, not in this work: `root.Log`/`AppConfig`/`AppContainer` globals,
the 24 nil-logger fallbacks, `batch/processor.go` split, store Load/Save pairs.

## Verification (every PR)

- TDD for each behavior change; `make test` and `make lint` green; CI green.
- Regression: convert `work/in/revolut` (and any CAMT/PDF sample under `work/in`)
  with `TEST_MODE=true` before and after; row counts and amounts identical. For C
  and D, categories may differ but must be names from `categories.yaml`.
- No run may modify `database/*.yaml` except through the end-of-run save.

## Out of scope

New features; `recategorize` batching; changing `ai.requests_per_minute`; the
iCompta plugin configuration.
