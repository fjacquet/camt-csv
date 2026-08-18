# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Stop the Viseca import double-counting the monthly card payment. Viseca's export carries a `Votre paiement - Merci` row for every statement settlement, which is the same money movement as the `Viseca Payment Services SA` debit on the bank statement; with both imported, an accounting tool that books the bank debit as a transfer credits the card account twice. On a real 26-month export that drifted the card balance up by 49,625.50 CHF. Settlement rows are now dropped, recognised by an empty `MerchantName`, an issuer-credit amount and the localised payment descriptor (fr/de/en/it) — a refund or a merchant row without a merchant name is unaffected. Pass `--keep-payments` (or `parsers.viseca.keep_payments: true`) to import them, for a setup that does not import the bank side.
- Apply command flags that shadow configuration. `InitializeConfig` builds its own Viper instance, so the `viper.BindPFlag` calls registered against the global one were never read; flags are now copied onto the loaded config before the container is built.
- Stop commands that end early losing learned category mappings. Anything that terminated a run before it unwound to the root `PersistentPostRun` hook — `FolderConvert` calling `os.Exit` on a non-zero run manifest (any failed file, an all-failed run, or an empty input directory), and the fifteen-odd `Fatal`/`Fatalf` call sites across `cmd/` — skipped the save of `creditors.yaml`/`debtors.yaml` and left the embedding warm-up issuing API calls. The save now also runs as a logrus exit handler, which covers every fatal path, and the batch exit code is recorded and applied by `main` after `Execute` returns.
- Apply the `backup.*` configuration. `CategoryStore.SetBackupConfig` had no production caller, so `backup.enabled`, `backup.directory` and `backup.timestamp_format` were silently ignored and the store always used its built-in defaults. The container now wires them from the loaded config. An empty `timestamp_format` keeps the default rather than collapsing every backup onto one filename.

### Added

- Add backup retention. Every mapping save wrote a timestamped `.backup` file and nothing ever removed them, so `database/` accumulated hundreds of copies. The newest `backup.retention` backups per file are kept (default 10); set it to `0` to keep all of them.

## [2.6.0] - 2026-08-18

### Fixed

- Stop `InvestmentType` being emitted on transactions that are not investments. `Transaction.Investment` is back-filled from `Type` for every parser, so an ordinary Revolut card payment carries `Investment="CARD_PAYMENT"`; the three `CSV-Revolut*` plugins map `investmentTransactionInfo.type` to that column, so all 221 rows of a Revolut export would have imported into iCompta as investment transactions. The column is now emitted only for rows that name a security.
- Stop the `icompta` output format discarding every investment and identity field. It emitted a fixed 10-column projection that had no `NumberOfShares`, `Fund`, `InvestmentType` or `Fees`, so Selma conversions lost share quantities entirely, and no parser's `ValueDate` or `CreditDebit` ever reached iCompta even though all eleven configured import plugins reference them. The header now carries ten further columns; the original ten keep their names and positions, so existing plugins resolve unchanged.
- Stop truncating fractional share counts. `Transaction.NumberOfShares` was an `int`, so the Selma parser's `int(sharesFloat)` and the Revolut Investment parser's `int(quantity.IntPart())` silently discarded any fraction — a holding of `0.4523` units was recorded as `0`, and a Revolut buy of `39.81059277` shares as `39`. Shares are now `decimal.Decimal`, matching every other numeric field on `Transaction`.
- Populate `BookkeepingNumber`, which no parser set despite being read back in the Selma stamp-duty pass. The CAMT parser now sources it from the account servicer reference and the Selma parser from the `Bookkeeping No.` column, giving iCompta a stable `externalID` to deduplicate re-imported statements. Revolut and the Viseca PDF statements have no identifier in their source data and are left empty.

- Stop an unreadable Selma share count discarding the whole trade. The share count is cosmetic next to the money movement, so the row is now kept with zero shares and a warning naming the offending value.

### Added

- Add a `viseca` command and parser for the CSV transaction export from the Viseca One portal. It reads structured fields instead of recovering them from laid-out PDF text, so it carries the merchant name, the foreign-currency original and exchange rate, and `TransactionId` — a stable per-transaction identifier the PDF statements never had, which becomes the `BookkeepingNumber` iCompta uses to deduplicate re-imported statements. The `pdf` command and parser are unchanged and remain the route for historical statements.
- Add `TestIComptaHeaderCoversPluginMappings` and `TestIComptaPluginsUseFormatterDelimiter`, which assert that every column and separator the iCompta import plugins reference is actually produced by the formatter. iCompta resolves columns by name and silently drops a mapping that finds no matching column, so this class of loss was previously invisible.
- Add `docs/icompta-plugin-setup.md` covering plugin configuration, safe editing of the iCompta document, and per-parser `externalID` coverage.
- Add `TransactionBuilder.WithBookkeepingNumber`.

### Changed

- Regenerate `.planning/reference/icompta-import-plugins.txt` from the iCompta document. The checked-in copy was stale: it recorded the wrong separator for `CSV-Selma` and `CSV-Revolut-CHF` and omitted five plugins.

## [2.5.0] - 2026-08-15

### Added

- Add a `convert` command that detects the input format instead of requiring you to name it. Each parser is asked in turn whether it recognizes the file and the first one that does performs the conversion, covering CAMT.053 XML, PDF, and the Revolut, Revolut Crypto, Revolut Investment, Selma and Visa Debit CSV exports. Pointed at a directory it detects each file independently, so a folder holding a mix of formats converts in one pass; unrecognized files are skipped with a warning rather than guessed at.
- Add `Categorizer.Shutdown` and `Container.Close`, called from the command post-run hook, so an in-flight embedding warm-up is cancelled at shutdown instead of continuing to issue rate-limited API calls. The warm-up now checks for cancellation between categories.
- Add a `--recursive` flag to the directory-processing commands, so a tree of statements can be converted in one run. Hidden files and directories are skipped at every level.

### Changed

- Fix `BatchProcessor` recursion being settable after construction: `SetRecursive` is replaced by a constructor parameter, since `ProcessDirectory` reads it and a late change could alter a run already under way.
- Replace the package-level `geminiAPIBaseURL`, which existed only so tests could redirect the client, with a `baseURL` constructor parameter matching `OpenRouterClient`. The detection order and the Revolut Investment header list become functions returning fresh slices, so no caller can alter them for the next one.
- Split `camtparser/adapter.go` (648 lines, with a single 380-line `Parse`) into three files: `camt053_schema.go` for the CAMT.053 XML types, which were declared inline inside `Parse`; `entry_mapping.go` for the entry-to-`Transaction` mapping and its helpers; and a 156-line `adapter.go` that now only drives the decode-map-categorize loop. Output is byte-identical, verified against nine real CAMT.053 files. Package coverage rises from 85.5% to 87.0%.
- Replace the copy-pasted `cmd/revolut` convert handler with `common.RunConvert`, the shared path already used by camt, selma, debit, revolut-crypto, and revolut-investment. Behaviour is unchanged; the command drops from 118 to 21 lines.
- Deprecate the `--date-format` flag. It was registered and threaded through five functions but never read by any writer — output dates have always been `DD.MM.YYYY`. The flag is still accepted so existing invocations keep working, and now reports that it has no effect.
- Unify `GeminiClient` and `OpenRouterClient` onto a shared `baseAIClient`. The two were near-forks: an identical 165-line categorization prompt, an identical 70-entry synonym table, and an identical `cleanCategory` were maintained twice, so a fix to one silently left the other behind. Prompt construction, response cleaning, rate limiting, retry/backoff and credential gating now have a single implementation; each provider supplies only its own HTTP call. The prompt text, synonym table and cleaning behaviour are unchanged.

### Fixed

- Fix `--recursive` destroying converted output. Output files were named from the input basename alone, so a recursive run over `jan/statement.xml` and `feb/statement.xml` wrote both to one `statement.csv` — the second silently replacing the first while the manifest reported both as successful. Output now mirrors the input tree. Inputs in one directory differing only by extension take the source extension into their name rather than colliding.
- Fix `convert` accepting `--recursive` without acting on it; nested files were never converted.
- Fix a directory that cannot be read being treated as empty. `discoverFiles` logged the failure and returned a short list, so the manifest reported success for whatever it happened to find. It is now an error.
- Fix cancellation being missed on rows that fail to convert, and a cancelled `Categorize` being recorded as an ordinary categorization failure. Every parser now checks for cancellation at the top of its row loop and returns `ctx.Err()` instead of filing the remaining transactions as Uncategorized.
- Fix the semantic categorization tier being silently disabled, and a data race, when `ai.provider` is `openrouter`. The container built the categorizer with the chat client and then swapped in the embedding client via `SetEmbeddingClient`, while a warm-up goroutine started by the constructor was already reading that field — an unsynchronized write confirmed by `go test -race`. Worse, that warm-up ran against `OpenRouterClient`, whose `GetEmbedding` always errors: it logged a warning per category and then marked the tier initialized with an empty embedding map, so tier 3 returned nothing while the startup log reported `Semantic tier: active`. `NewCategorizer` now takes the chat and embedding clients as separate parameters and wires both before any goroutine starts; `SetEmbeddingClient` is removed.
- Fix `revolutinvestmentparser.ValidateFormat` accepting any readable CSV. It only checked that a header row could be parsed, so it claimed Revolut, Selma and Visa Debit exports as its own, and `--validate` on those files passed against the wrong parser. It now verifies the eight expected column names, matching the check the parser itself already performed.
- Fix `OpenRouterClient` not retrying request timeouts. Its `isRetryableError` was missing the `os.IsTimeout` check that `GeminiClient` had, so a timed-out OpenRouter request failed outright instead of being retried. Both providers now share one retry policy.
- Stop writing `.manifest.json` twice per batch run — `BatchProcessor.ProcessDirectory` already writes it, and `FolderConvert` was immediately rewriting the same file.
- Add missing `#nosec G304` justifications in `batch/processor.go` and `pdfparser.go`, matching the convention used at every other file-open site; `gosec` now reports zero issues.
- Propagate `context.Context` through categorization. `common.ProcessTransactionsWithCategorizationStats` hardcoded `context.Background()`, and six of the seven parsers accepted a `ctx` on `Parse` and discarded it, so cancelling a run (Ctrl-C, a deadline, a cancelled batch) had no effect once categorization started — a several-thousand-transaction AI run could not be interrupted. Every parser now threads its `ctx` to `Categorize`, and the categorization loops check for cancellation between transactions and return `ctx.Err()` instead of a partially categorized slice.

### Removed

- Reduce `models/iso20022.go` from 427 lines to 32, and its tests from 824 lines to 108. Despite describing most of CAMT.053, the file had a single production use: `ISO20022Parser.ValidateFormat` unmarshalled a candidate file into `ISO20022Document` and checked that at least one statement came back. All ten helper methods on `models.Entry` (`GetPayer`, `GetPayee`, `GetIBAN`, `BuildDescription` and the rest) had no caller outside their own tests. This was not duplication of `camtparser/camt053_schema.go` in the DRY sense — the two answer different questions — but dead weight around a yes/no check. Validation verdicts are unchanged across 23 files, and CAMT conversion output is byte-identical.

- Remove `BatchConverter` from the `parser.FullParser` interface and drop all seven adapter `BatchConvert` implementations. The method had no callers: directory processing has gone through `batch.BatchProcessor` since it was introduced, and the seven implementations had silently diverged (Selma returned `not implemented`, Visa Debit delegated to a legacy helper, the rest hand-rolled incompatible loops). Parsers now parse; `batch.BatchProcessor` handles directories.
- Remove the legacy package-level `debitparser.BatchConvert` and `debitparser.BatchConvertWithLogger` helpers, whose only caller was the deleted adapter method.


## [2.4.0] - 2026-04-06

### Added

- Add `revolut-crypto` command to parse Revolut Crypto account statement CSV exports (French locale numbers and dates, supports `Achat` and `Récompense de staking` transaction types)
- Add `revolut` parser support for French-localized CSV exports (headers and values normalized to English before parsing)

### Fixed

- Fix `revolut-investment` parser failing on amounts with ISO 4217 currency prefix (e.g. `USD 2.84`)
- Fix `revolut` command ignoring `output.format` config when `--format` flag is not passed

- Fix all parsers outputting only positive amounts — debit transactions now correctly have negative amounts in CSV output (CAMT, PDF, Revolut, Visa Debit, Selma, Revolut Investment). The sign is applied in `TransactionBuilder.Build()` based on debit/credit direction.
- Fix AI rate limiter dropping requests instead of throttling — replace non-blocking `Allow()` with blocking `Wait(ctx)` in both OpenRouter and Gemini clients so batch processing naturally paces requests at the configured rate instead of rejecting them
- Increase rate limiter burst size from 1 to `requestsPerMinute` to allow natural request bursts within the configured limit
- Fix PDF consolidation failing when `--output` is a directory — auto-generate filename from input directory name (e.g., `-o work/out/viseca` now writes `viseca.csv` inside that directory)

### Changed

- Increase default `requests_per_minute` from 5 to 60 in config

## [2.3.2] - 2026-03-24

### Fixed

- Fix Selma parser losing `Name` field — Fund ISIN now mapped to `PartyName` for trade, dividend, withholding_tax transactions; `Selma` used for fees and transfers
- Fix Selma trade descriptions — `trade` replaced with `Buy <ISIN>` or `Sell <ISIN>` based on amount sign
- Fix Selma withholding_tax category — now correctly set to `Impôts` internally without relying on AI
- Fix `cleanCategory` to extract `**text**` from anywhere in a single-line verbose AI response (not just at string edges)
- Fix `categorization_helper` to preserve categories set by parser-internal logic — skips external categorizer when category already determined

## [2.3.1] - 2026-03-24

### Fixed

- Fix AI response parsing — `cleanCategory` now handles verbose multi-line responses and markdown bold formatting (`**Category**`) returned by some models (e.g., `mistral-small-2603`); previously the full explanation was stored as the category value
- Fix `math/rand` usage in `GeminiClient` retry jitter — replaced with time-based jitter (`time.Now().UnixNano()`) to resolve Semgrep CWE-338 warning; retry jitter is not security-sensitive
- Fix config test expectations for API key error message — updated to match current wording after v1.6 unified key refactor
- Add `OPENROUTER_API_KEY` env var as dedicated fallback for OpenRouter users (between `CAMT_AI_API_KEY` and `GEMINI_API_KEY`)

### Changed

- Lower default `ai.requests_per_minute` from 20 to 5 for cost control on personal finance workloads

### Added

- Add `ai.provider` config field (default `gemini`, supports `openrouter`) to select AI backend; validated at startup
- Add `ai.base_url` config field (default empty) to override provider endpoint for OpenAI-compatible providers
- Add `CAMT_AI_API_KEY` as unified API key env var; `GEMINI_API_KEY` retained as backward-compatible fallback
- Add `OpenRouterClient` implementing `AIClient` interface — enables any OpenRouter-hosted model (e.g., `mistralai/mistral-small-2603`) for transaction categorization via OpenAI-compatible chat/completions API
- Add split chat/embedding client architecture — OpenRouter handles chat categorization (tier 4), Gemini handles embeddings (tier 3) when `GEMINI_API_KEY` is set alongside OpenRouter
- Add `Categorizer.SetEmbeddingClient()` method for independent embedding provider wiring
- Add provider and semantic tier status logging at startup (e.g., "AI provider: openrouter", "Semantic tier: active (Gemini embeddings)")
- Add `categorization.semantic_threshold` config key (default 0.70, env `CAMT_CATEGORIZATION_SEMANTIC_THRESHOLD`) to tune semantic matching sensitivity
- Add persistent embedding cache (`~/.camt-csv/embedding_cache.json`) — eliminates ~50 Gemini API calls on startup when categories haven't changed
- Add in-batch deduplication cache — identical party names within a single run are categorized once and reused, reducing redundant API calls by ~90%
- Add per-transaction embedding cache within SemanticStrategy — avoids duplicate Gemini embedding calls for the same party name
- Add near-miss logging for semantic scores just below threshold (debug level) to help tune threshold

### Changed

- Refactor `NewGeminiClient` to accept `apiKey` as constructor parameter instead of reading `GEMINI_API_KEY` from environment — enables multi-provider key management
- Refactor container AI wiring to use provider-based switch with split chat/embedding clients
- Merge hardcoded keyword patterns from Go source into `database/categories.yaml` — single source of truth for all keyword rules
- Remove `categorizeWithHardcodedPatterns` method from KeywordStrategy — all keyword matching now driven by YAML configuration

### Fixed

- Fix `ai.model` config key being ignored — GeminiClient now uses the configured model instead of reading `GEMINI_MODEL` env directly
- Fix `ai.timeout_seconds` config key being ignored — HTTP client timeout is now configurable instead of hardcoded to 30s
- Fix non-deterministic keyword matching — hardcoded merchant patterns used Go map iteration (random order), causing less-specific keywords to sometimes win over more-specific ones

## [1.5.1] - 2026-03-02

### Added

- Configure default output format via `CAMT_OUTPUT_FORMAT` environment variable (or `.env` file); `--format` flag still overrides when passed explicitly

## [1.5.0] - 2026-03-02

### Added

- Add `--format jumpsoft` output option to all parser commands (camt, pdf, revolut, selma, debit, revolut-investment), producing 7-column comma-delimited CSV (Date, Description, Amount, Currency, Category, Type, Notes) compatible with Jumpsoft Money import

## [1.4.0] - 2026-02-23

### Added

- Add `FolderConvert` to `cmd/common/convert.go`: modern batch path using `BatchProcessor` with formatter support for camt, debit, selma, and revolut-investment parsers
- Add unit tests for folder routing in `cmd/common/convert_test.go` (empty dir, invalid format, non-FullParser guard)
- Add SBOM generation to GoReleaser release pipeline using syft (CycloneDX JSON format, per-archive and source)

### Changed

- Change default output format from `standard` to `icompta` — iCompta-compatible semicolon-delimited output requires no `--format` flag
- `RunConvert` now logs a fatal error and exits when `--input` is a directory and `--output` is not set
- `RunConvert` delegates to `FolderConvert` (instead of `BatchConvertLegacy`) when folder input is given with `--output`
- Add `osExitFn` package variable to `cmd/common` for testable exit-code handling in `FolderConvert`
- All 6 parser commands (camt, debit, revolut, revolut-investment, selma, pdf) now accept file or folder as input automatically — no separate batch command needed (Input Auto-Detection)
- Folder input requires `--output` flag; clear error message is shown if omitted
- PDF folder mode always consolidates all PDFs into a single CSV (removed `--batch` flag)
- Non-PDF folder mode (camt, debit, revolut, revolut-investment, selma) outputs one CSV per input file using BatchProcessor with formatter support

### Removed

- Remove `batch` subcommand — all parser commands accept folder input directly via auto-detected folder mode
- Remove deprecated `BatchConvertLegacy` internal function from cmd/common
- Remove `--batch` flag from `pdf` command (folder mode now always consolidates)
- Remove `pdfBatchConvert` function from `cmd/pdf/convert.go` (superseded by consolidation-only folder mode)

## [2.3.0] - 2026-02-21

### Fixed

- Skip output file generation when 0 transactions are parsed (all parsers and modes)

## [2.2.3] - 2026-02-21

### Fixed

- Skip output file generation when 0 transactions are parsed (all parsers and modes)

## [2.2.2] - 2026-02-21

### Changed

- Simplify string normalization: replace hand-rolled ASCII fast-path with `strings.ToLower`/`strings.ToUpper`
- Make CSV writers non-mutating: copy transaction slices before modifying derived fields
- Make strategy constructors pure: `NewDirectMappingStrategy` and `NewKeywordStrategy` accept pre-loaded data instead of performing I/O
- Extract shared `RunConvert` handler and `RegisterFormatFlags` helper for CLI commands (-327 lines)
- Extract shared `ConvertToCSVDefault` into `BaseParser` for adapter DRY (-40 lines)

### Fixed

- Race condition in `MockLogger` lazy initialization using `sync.Once`

### Removed

- Delete dead `models.Parser` interface (replaced by `parser.FullParser`)
- Delete unused `TransactionParty` and `CategorizerTransaction` types from models
- Delete unused `Config` interface, `aiFactory` field, and `SetAIClientFactory` from categorizer
- Delete thin wrapper functions `ExportTransactionsToCSV` and `ExportTransactionsToCSVWithLogger`
- Delete dead `loadMappings` method from `DirectMappingStrategy`

## [2.2.1] - 2026-02-21

### Changed

- Simplify CI: merge duplicate test runs, remove Go 1.23 matrix, use `go-version-file: go.mod`
- Upgrade `actions/setup-go@v4` to `@v5` across all workflows
- Remove duplicate SBOM generation from CI (SLSA workflow handles it for releases)
- Improve test coverage from 64.3% to 70.5%

### Removed

- Dead code cleanup: remove 112+ unreachable functions detected by `deadcode` (golang.org/x/tools)
- Remove unused utility packages: `currencyutils`, `fileutils`, `validation`, `textutils`, `xmlutils`, `factory`
- Remove unused model types: `Money`, `Party` (and their tests)
- Remove `strategy_result.go` from categorizer
- Remove dead ISO20022 methods (`ParseFile`, `extractTransactions`, `entryToTransaction`, `categorizeTransactions`, `ConvertToCSV`, `CreateEmptyCSVFile`) and `concurrent_processor.go`
- Remove dead standalone parser wrappers (`Parse`, `WriteToCSV`, `ConvertToCSV`, `BatchConvert` + `*WithLogger` variants) from all parsers
- Remove dead builder methods (`WithBookkeepingNumber`, `WithAmountFromFloat`, `WithBankTxCode`, `WithFeesFromFloat`, `WithIBAN`, `Reset`, `Clone`)
- Remove dead container methods (`GetParsers`, `GetConfig`, `GetStore`, `GetAIClient`, `Close`)
- Remove dead utility functions from `dateutils`, `git`, `common`, `formatter`, `categorizer`, `parsererror`

## [2.2.0] - 2026-02-17

### Added

- Multi-arch Docker images published to `ghcr.io/fjacquet/camt-csv` on release
- Homebrew tap: `brew tap fjacquet/homebrew-tap && brew install camt-csv`

### Fixed

- Resolve all open gosec code scanning alerts (G117, G204, G702, G703, G704)
- Sanitize git ref inputs to prevent command injection (G204)

## [2.1.0] - 2026-02-17

### Added

- GoReleaser workflow for automated multi-platform releases on tag push
- `--version` flag now shows version, commit, and build date injected via ldflags
- AI categorization staging: when `--auto-learn` is off, AI suggestions are saved to `database/staging_creditors.yaml` and `database/staging_debtors.yaml` for manual review instead of being discarded
- Staging configuration: `staging.enabled`, `staging.creditors_file`, `staging.debtors_file`

### Changed

- Makefile LDFLAGS aligned with GoReleaser: `main.version`, `main.commit`, `main.date`
- Dockerfile updated with version ldflags injection and Alpine 3.21

### Removed

- Unused AI tool configs: `.specify/`, `.kiro/`, `.gemini/`, `specs/`

## [2.0.0] - 2026-02-17

### Removed

- **Deprecated Transaction methods**: `GetPayee()`, `GetPayer()`, `GetAmountAsFloat()`, `GetDebitAsFloat()`, `GetCreditAsFloat()`, `GetFeesAsFloat()`, `ToBuilder()`, `SetPayerInfo()`, `SetPayeeInfo()`, `SetAmountFromFloat()` — use `GetCounterparty()`, `*AsDecimal()` variants, and `TransactionBuilder` instead
- **Deprecated process functions**: `ProcessFileLegacy()`, `ProcessFileLegacyWithError()`, `SaveMappings()` — use `ProcessFileWithError()` and container-based categorizer
- **Deprecated root command function**: `GetConfig()` — use `GetContainer().GetConfig()`
- **Deprecated mock logger field**: `Entries` on `MockLogger` — use `GetEntries()`
- **Deprecated XPath constants**: `Camt053XPaths` var — use `DefaultCamt053XPaths()`

### Changed

- **README.md slimmed down** from ~287 lines to ~107 lines; detailed content moved to docs site
- **Documentation restructured**: 7 stale docs deleted (3 migration guides, deprecation timeline, 3 meta-docs), migration content consolidated into developer guide
- **All documentation updated** to reflect current codebase:
  - `api-specifications.md`: Corrected Transaction struct, Categorizer interface, added OutputFormatter
  - `architecture.md`: Added SemanticStrategy (4th categorization tier), Formatter system, context.Context
  - `design-principles.md`: Added Registry Pattern, updated method signatures
  - `developer-guide.md`: Fixed interface signatures, added v1.x→v2.0.0 migration guide
  - `user-guide.md`: Complete configuration reference with all flags, env vars, and YAML options
  - `testing-strategy.md`: Removed stale `SetTestCategoryStore()` refs, added DI-based testing patterns
  - `operations.md`: Removed fictional Prometheus/health check sections, updated CI/CD pipeline
  - `coding-standards.md`: Added context.Context convention, strategy pattern reference

## [1.3.0] - 2026-02-16

### Changed

- **Standard CSV Format Trimmed to 29 Columns** (Phase 10): Remove 6 redundant/dead fields from standard CSV output
  - Remove BookkeepingNumber (never populated by any parser)
  - Remove IsDebit/DebitFlag (redundant with CreditDebit field)
  - Remove Debit (derived from Amount + CreditDebit)
  - Remove Credit (derived from Amount + CreditDebit)
  - Remove Recipient (redundant with Name/PartyName)
  - Remove AmountTax (never populated by any parser)
  - Update MarshalCSV/UnmarshalCSV for 29-column format
  - Update StandardFormatter header from 35 to 29 columns
  - Fix hardcoded header in common/csv.go WriteTransactionsToCSVWithLogger

### Added

- **End-to-End Format Tests** (Phase 11): Integration tests verifying both output formats
  - TestEndToEndConversion_StandardFormat validates 29-column standard CSV output
  - TestEndToEndConversion_iComptaFormat validates 10-column semicolon-delimited output
  - Explicit 29-column assertions in TestCrossParserConsistency

## [1.2.0] - 2026-02-16

### Added

- **Output Formatter Framework** (Phase 5): Pluggable CSV output formatting with strategy pattern
  - `OutputFormatter` interface with `StandardFormatter` (29-column, comma-delimited) and `iComptaFormatter` (10-column, semicolon-delimited, dd.MM.yyyy dates)
  - `FormatterRegistry` for managing formatters by name
  - `--format` CLI flag on all parser commands (camt, pdf, revolut, selma, debit, revolut-investment)
  - `ProcessFile()` refactored to use `Parse()` + `WriteTransactionsToCSVWithFormatter` pipeline
  - DI container exposes `FormatterRegistry` for consistent formatter access

- **Product Field** (Phase 6): Transaction model expanded from 34 to 35 columns
  - New `Product` field (Current/Savings) positioned after Currency
  - Builder pattern updated with `WithProduct()` method
  - All formatters and CSV writers updated for 35-column output

- **Revolut Investment Parser Enhancements** (Phase 6): Complete transaction type coverage
  - SELL transaction support (credit/incoming money from sales)
  - CUSTODY_FEE transaction support (debit/outgoing fees with fee tracking)
  - Batch conversion mode for investment CSV files

- **Revolut Parser Field Population** (Phase 6): Full 35-field standardized output
  - All transaction fields populated via builder pattern
  - Exchange transactions preserve OriginalAmount/OriginalCurrency metadata
  - Product field populated from source data
  - REVERTED and PENDING transactions logged when skipped

- **Batch Infrastructure** (Phase 7): Universal batch processing with error reporting
  - Reusable `BatchProcessor` using composition pattern (wraps any parser)
  - `BatchManifest` with JSON serialization and semantic exit codes (0=all success, 1=partial, 2=all failed)
  - PDF parser `--batch` flag for individual file conversion mode
  - All 6 CLI commands generate `.manifest.json` and exit with semantic codes
  - Batch processing continues after individual file failures

- **AI Safety Controls** (Phase 8): Safety gates for AI categorization
  - `--auto-learn` flag controls auto-save of AI categorizations (default: OFF)
  - Gemini API rate limiting via token bucket (configurable RPM, default 10)
  - Exponential backoff with jitter for transient API failures (429, 503, timeouts)
  - Confidence metadata on all categorizations (direct=1.0, keyword=0.95, semantic=0.90, AI=0.8-0.9)
  - Pre-save audit logging with confidence scores and source strategy

### Previously added

- **PDF Directory Consolidation**: The `pdf` command now accepts a directory as input, consolidating all PDF files into a single CSV output
  - Single file mode: `camt-csv pdf -i file.pdf -o output.csv` (existing behavior)
  - Directory mode: `camt-csv pdf -i pdf_dir/ -o consolidated.csv` (new feature)
  - Automatically detects and processes all PDF files in the specified directory
  - Consolidates all transactions from multiple PDFs into a single chronologically-sorted CSV
  - Failed PDFs are skipped with warning logs (graceful degradation)
  - Context cancellation support for interrupting long operations with Ctrl+C
  - Validation flag (`--validate`) applies to each PDF file individually
  - Case-insensitive PDF file detection (.pdf, .PDF, .Pdf all supported)
  - No source file metadata in output - pure transaction data for clean consolidation

- **Development Infrastructure**:

  - `CLAUDE.md` - AI-assisted development guidance with coding principles (KISS, DRY, FP)
  - `Dockerfile` - Multi-stage Alpine container build for containerized deployments
  - `Makefile` - Development commands (build, test, lint, coverage, security)
  - `codecov.yml` - Code coverage configuration and thresholds
  - `plan.md` - Senior architect review with action plan and production readiness checklist

- **Documentation**:

  - CAMT.053 ISO 20022 format documentation in CLAUDE.md
  - Coding principles: KISS, DRY, Functional Programming guidelines
  - Dependency injection patterns and interface design guidelines

- **Category YAML Backup System**: Automatic timestamped backups of category mapping files before auto-learn overwrites
  - Backup enabled by default (`backup.enabled: true` in config)
  - Configurable backup directory (defaults to same directory as original file)
  - Configurable timestamp format (default: `YYYYMMDD_HHMMSS`)
  - Atomic behavior: failed backup prevents save, protecting original file
  - Supports both creditor and debtor mapping files

- **Test Coverage Improvements**:
  - Add tests for `cmd/batch` package
  - Add tests for `cmd/common` package with mock parser implementation
  - Add tests for `internal/fileutils` package
  - Add tests for `internal/textutils` package
  - Add tests for `internal/validation` package
  - Add nil container error verification tests for camt, debit, and pdf commands
  - Add 8 concurrent processing edge case tests: context cancellation (before, during, inflight), race conditions, partial result data integrity
  - Add 14 PDF Viseca format detection edge case tests: partial markers, false positives, ambiguous formats
  - Add 20+ error message validation tests across all parsers (CAMT, Debit, Revolut, Selma, PDF) verifying file path and field context
  - Add 5 category backup tests: backup creation, custom location, failure prevention, disabled mode, multiple timestamps
  - Add `MockLogger.VerifyFatalLog()` and `VerifyFatalLogWithDebug()` helper methods for test verification

### Removed

- **Deprecated Config Functions**: Remove `LoadEnv()`, `GetEnv()`, `MustGetEnv()`, `GetGeminiAPIKey()`, `ConfigureLogging()`, `InitializeGlobalConfig()` from `internal/config/config.go`; all configuration now flows through Viper/Container
- **Global Mutable State**: Remove `Logger`, `globalConfig`, and `sync.Once` globals from config package; all state flows through DI container
- **Fallback Categorizer Creation**: Remove silent fallback in `PersistentPostRun` that bypassed dependency injection; nil container now logs warning and returns early

### Security

- **No Credential Logging**: API key values never appear in log output at any level; only presence/absence is logged
- **Secure Temp Files**: All temporary files use `os.CreateTemp()` with random naming; no predictable temp file paths
- **File Permission Standardization**: Non-secret files (YAML category mappings, CSV output) use 0644; secrets use 0600; directories use 0750

### Changed

- **PDF Parser Temp File Consolidation**: Replace individual temp file with single temp directory (`os.MkdirTemp`) for all PDF processing; cleanup uses single `os.RemoveAll` call
- **PDF Parser ExtractText Optimization**: Eliminate duplicate `ExtractText` call (was called twice: once for validation, once for extraction)
- **Categorize Command Init**: Replace `panic(err)` with graceful error handling; Cobra framework handles missing required flags at runtime

### Fixed

- **PDF Debug File Leak**: Remove `debug_pdf_extract.txt` file that accumulated in working directory after PDF parsing
- **PDF Context Propagation**: `Parse()` and `ParseWithExtractor()` now accept and propagate `context.Context` instead of discarding it for `context.Background()`
- **PDF Temp File Cleanup**: Consolidate two separate defer blocks into single close-then-remove defer for correct cleanup ordering
- **MockLogger State Isolation**: `WithError()` and `WithFields()` now create properly isolated instances using shared pointer pattern; tests can verify specific log messages at correct levels
- **Context Propagation**: Fix context propagation throughout application for proper cancellation support
  - CLI commands now extract context from `cmd.Context()` and propagate through all layers
  - Parser interfaces now accept `context.Context` parameter for cancellation and timeout support
  - `TransactionCategorizer.Categorize()` now accepts `context.Context` for AI operations
  - `ConcurrentProcessor` properly handles context cancellation with partial result returns
  - Fix index out of range panic in `ConcurrentProcessor.processConcurrent()` when context cancelled
  - Add context cancellation tests for sequential and concurrent processing paths
  - Enables graceful shutdown with Ctrl+C in batch operations
- **Race Condition in DirectMappingStrategy**: Fix race condition in `ReloadMappings()` method
  - Build new mappings outside lock to eliminate vulnerability window
  - Atomic pointer swap ensures readers never see empty maps during reload
  - Add concurrent test verifying no race conditions with `-race` detector
  - All tests pass with race detector: `make test-race` clean
- **Batch Command Categorization**: Fix batch command not categorizing transactions
  - Batch command now uses DI container instead of factory directly
  - Transactions are now properly categorized using all 3 tiers (direct mapping, keyword, AI)

### Changed

- **BREAKING: Parser Interfaces**: Add `context.Context` parameter to all parser methods
  - `Parser.Parse()` now requires `context.Context` as first parameter
  - `CSVConverter.ConvertToCSV()` now requires `context.Context` as first parameter
  - `BatchConverter.BatchConvert()` now requires `context.Context` as first parameter
  - Custom parser implementations must be updated to accept context
- **BREAKING: TransactionCategorizer Interface**: Add `context.Context` parameter to categorization
  - `TransactionCategorizer.Categorize()` now requires `context.Context` as first parameter
  - Enables proper cancellation propagation through AI categorization calls
- **BatchConverter Interface**: Add `BatchConverter` interface to `parser.FullParser`
  - Follows Interface Segregation Principle (ISP)
  - Enables batch conversion through the DI container
- Update `CLAUDE.md` to reflect refactored parser interface (segregated interfaces, new factory location)
- Update dependencies: cobra v1.10.2, golang.org/x/net v0.47.0, golang.org/x/sys v0.38.0, golang.org/x/text v0.31.0
- **Type-safe Categorizer Interface**: Replace `interface{}` with `models.TransactionCategorizer` for compile-time type safety
  - Add `TransactionCategorizer` interface to models package
  - Update `CategorizerConfigurable` interface to use type-safe signature
  - Modify `Categorize` method to include auto-learning functionality
- **Immutable Container**: Make `Container` struct fields private for immutability
  - All fields now accessed through getter methods only
  - Add `GetParsers()` method returning a copy of the parser map
  - Prevents accidental modification after initialization

### Deprecated

- **Legacy Configuration Functions** (`internal/config/config.go`):
  - `LoadEnv()` - Use `InitializeConfig()` instead
  - `GetEnv()` - Use `Config` struct fields instead
  - `MustGetEnv()` - Use `Config` struct with validation instead
  - `GetGeminiAPIKey()` - Use `Config.AI.APIKey` instead
  - `ConfigureLogging()` - Use `ConfigureLoggingFromConfig()` instead
  - `InitializeGlobalConfig()` - Use `InitializeConfig()` with DI container instead
  - Global `Logger` variable - Use `container.GetLogger()` instead
  - All deprecated functions will be removed in v3.0.0

### Fixed

- Remove redundant `config.LoadEnv()` call in `cmd/categorize/categorize.go`
- Fix unchecked `file.Close()` return values in `internal/fileutils/fileutils_test.go`
- Fix SLSA workflow: update Go version to 1.24, upgrade to slsa-github-generator v2.0.0
- Add missing `.slsa-goreleaser.yml` configuration for SLSA provenance builds

## [2.0.0] - 2025-11-02

### Added

- **Dependency Injection Architecture**: Complete refactoring to use dependency injection pattern

  - New `Container` type for managing all application dependencies
  - Elimination of global mutable state for better testability
  - All parsers now receive dependencies through constructors

- **Interface Segregation for Parsers**:

  - Segregated parser interfaces (`Parser`, `Validator`, `CSVConverter`, `LoggerConfigurable`)
  - `BaseParser` foundation providing common functionality to all parsers
  - Composition-based architecture eliminating code duplication

- **Framework-Agnostic Logging**:

  - New `logging.Logger` interface for structured logging abstraction
  - `LogrusAdapter` implementation with dependency injection support
  - Structured logging with `Field` type for key-value pairs

- **Transaction Builder Pattern**:

  - Fluent API for constructing complex transactions with validation
  - Type-safe transaction creation with sensible defaults
  - Automatic field population and validation at build time

- **Strategy Pattern for Categorization**:

  - Modular categorization strategies (`DirectMappingStrategy`, `KeywordStrategy`, `AIStrategy`)
  - Priority-based strategy execution with independent testing
  - Extensible architecture for adding new categorization methods

- **Comprehensive Error Handling**:

  - Custom error types (`ParseError`, `ValidationError`, `CategorizationError`, `DataExtractionError`)
  - Detailed error context with parser, field, and value information
  - Proper error wrapping with `fmt.Errorf` and `%w` verb

- **Performance Optimizations**:

  - String operations optimization with `strings.Builder` and pre-allocation
  - Lazy initialization for expensive resources (AI client)
  - Pre-allocated slices and maps with capacity hints

- **Constants-Based Design**:

  - Comprehensive constants in `internal/models/constants.go`
  - Elimination of magic strings and numbers throughout codebase
  - Type-safe transaction directions and status values

- **Enhanced Documentation**:
  - Comprehensive architecture documentation
  - Developer guide with patterns and best practices
  - Migration guide for upgrading from v1.x
  - Godoc comments for all public APIs

### Changed

- **File Naming Convention**: `debitors.yaml` renamed to `debtors.yaml` for standard English spelling
- **Configuration Structure**: Hierarchical YAML structure with nested sections (`log`, `csv`, `ai`)
- **Date Handling**: Internal use of `time.Time` instead of strings with proper CSV marshaling
- **Parser Architecture**: All parsers now embed `BaseParser` and use dependency injection
- **Error Messages**: More detailed and structured error information with custom types

### Removed

- **Global Singleton Functions** (BREAKING CHANGES):

  - `categorizer.GetDefaultCategorizer()` - Use `container.NewContainer()` and access `Categorizer`
  - `categorizer.CategorizeTransaction()` - Use `categorizer.Categorize()` with context
  - `categorizer.UpdateDebitorCategory()` - Use `categorizer.UpdateDebtorMapping()`
  - `categorizer.UpdateCreditorCategory()` - Use `categorizer.UpdateCreditorMapping()`
  - `config.GetGlobalConfig()` - Use `config.LoadConfig()` with dependency injection
  - `config.GetCSVDelimiter()` - Access through configuration object
  - `config.GetLogLevel()` - Access through configuration object
  - `config.IsAIEnabled()` - Access through configuration object
  - `factory.GetParser()` - Use `factory.GetParserWithLogger()` or container
  - `logging.GetLogger()` - Use `logging.NewLogrusAdapter()` with dependency injection

- **Deprecated Methods** (BREAKING CHANGES):

  - `CategoryStore.LoadDebitorMappings()` - Use `LoadDebtorMappings()`
  - `CategoryStore.SaveDebitorMappings()` - Use `SaveDebtorMappings()`
  - `DirectMappingStrategy.UpdateDebitorMapping()` - Use `UpdateDebtorMapping()`
  - `Transaction.GetAmountAsFloat()` - Use `Transaction.Amount.Float64()` or decimal operations
  - `Transaction.GetPayee()` - Access `Transaction.Payee` field directly
  - `Transaction.GetPayer()` - Access `Transaction.Payer` field directly
  - Legacy transaction conversion methods - Use `TransactionBuilder` for new transactions

- **Internal Deprecated Methods**:
  - `Categorizer.categorizeWithGemini()` - Replaced by `AIStrategy`

### Deprecation Timeline

See [DEPRECATION_TIMELINE.md](docs/DEPRECATION_TIMELINE.md) for complete deprecation schedule and migration guidance.

**Current Status (v2.0.0):**

- ✅ Global singleton functions removed
- ⚠️ Transaction backward compatibility methods deprecated (removal in v3.0.0)
- ✅ New dependency injection architecture available
- ✅ TransactionBuilder pattern available
  - `Categorizer.categorizeByCreditorMapping()` - Replaced by `DirectMappingStrategy`
  - `Categorizer.categorizeByDebitorMapping()` - Replaced by `DirectMappingStrategy`
  - `Categorizer.categorizeLocallyByKeywords()` - Replaced by `KeywordStrategy`

### Migration Guide

#### Configuration Migration

**Old configuration** (`~/.camt-csv/config.yaml`):

```yaml
log_level: "info"
csv_delimiter: ","
ai_enabled: true
```

**New configuration**:

```yaml
log:
  level: "info"
  format: "text"
csv:
  delimiter: ","
ai:
  enabled: true
  model: "gemini-2.0-flash"
```

#### Code Migration

**Old code**:

```go
// Global singleton usage (removed)
categorizer := categorizer.GetDefaultCategorizer()
result := categorizer.CategorizeTransaction(tx)
```

**New code**:

```go
// Dependency injection
container, err := container.NewContainer(config)
if err != nil {
    log.Fatal(err)
}
result, err := container.Categorizer.Categorize(ctx, tx)
```

#### File Migration

```bash
# Rename debtor mapping file
mv database/debitors.yaml database/debtors.yaml
```

### Security

- Improved input validation with custom error types
- Better error message sanitization
- Proper file permissions constants usage

### Performance

- Reduced memory allocations in hot paths
- Optimized string operations with pre-allocation
- Lazy initialization of expensive resources

### Testing

- Achieved 80%+ test coverage
- Mock dependencies for all external interactions
- Integration tests for end-to-end workflows
- Benchmark tests for performance-critical paths

---

## [1.x.x] - Previous Versions

See Git history for changes in previous versions.

### Breaking Changes Summary

This major version (2.0.0) removes all deprecated global singleton functions and methods that were marked for removal. The new architecture is based on dependency injection and provides better testability, maintainability, and performance.

**Key Migration Steps:**

1. Update configuration file structure
2. Rename `debitors.yaml` to `debtors.yaml`
3. Replace global function calls with dependency injection pattern
4. Update error handling to use new custom error types
5. Use `TransactionBuilder` for creating new transactions

For detailed migration instructions, see [docs/migration-guide.md](docs/migration-guide.md).
