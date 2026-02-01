# Codebase Concerns

**Analysis Date:** 2026-02-01

## Tech Debt

**Deprecated Configuration System:**
- Issue: Legacy configuration code in `internal/config/config.go` marked for removal in v3.0.0. Multiple deprecated functions coexist with new Viper-based system (`MustGetEnv()`, `LoadEnv()`, `GetEnv()`).
- Files: `internal/config/config.go` (entire file), `cmd/root/root.go` (lines 80-91)
- Impact: Duplicated configuration logic, inconsistent error handling patterns. New code uses dependency injection while legacy code can panic.
- Fix approach: Complete migration from deprecated functions to Viper/Container pattern. Remove `MustGetEnv()` and direct env access from all command code.

**Fallback Categorizer Creation in PersistentPostRun:**
- Issue: `cmd/root/root.go` lines 80-91 creates a new categorizer instance with hardcoded paths when container is nil, bypassing dependency injection.
- Files: `cmd/root/root.go` (lines 80-91)
- Impact: Creates unnecessary object allocations, uses hardcoded database paths instead of configuration, defeats DI container benefits.
- Fix approach: Remove fallback logic. If container fails to initialize, let the error propagate instead of silently creating unmanaged objects.

**MockLogger State Management Issues:**
- Issue: `internal/logging/mock.go` - `WithError()` and `WithFields()` methods create new MockLogger instances but share the original Entries slice. Pending state (pendingError, pendingFields) is not properly isolated.
- Files: `internal/logging/mock.go` (lines 86-110), usage in `internal/categorizer/categorizer_strategy_test.go` (lines 176, 232, 280)
- Impact: Tests skip logging verification due to these issues (explicit TODOs). Can lead to test pollution where logging state leaks between test cases.
- Fix approach: Redesign MockLogger to properly isolate state. Consider making each chained call fully independent or using a different pattern for capturing logs.

**Large Monolithic PDF Parsing Function:**
- Issue: `internal/pdfparser/pdfparser_helpers.go` is 783 lines with deeply nested parsing logic. Multiple concerns (format detection, transaction extraction, categorization) mixed in single file.
- Files: `internal/pdfparser/pdfparser_helpers.go`
- Impact: Difficult to modify safely, high cognitive load, increased bug risk. Hard to test individual parsing strategies independently.
- Fix approach: Refactor into separate strategy implementations (VisecaStrategy, StandardPDFStrategy, etc.). Use strategy pattern to handle different PDF formats.

---

## Known Bugs

**Debug File Accumulation:**
- Symptoms: `debug_pdf_extract.txt` is created in current working directory for every PDF parsed (line 96-100). File persists across runs, can accumulate or be committed to git.
- Files: `internal/pdfparser/pdfparser.go` (lines 95-103)
- Trigger: Parse any PDF file via CLI
- Workaround: None. Manually delete debug file or redirect to temp directory.
- Root cause: Hard-coded path to current working directory instead of using OS temp directory.

**MockLogger Logging Verification Skipped:**
- Symptoms: Tests cannot verify that specific log messages were emitted at correct levels.
- Files: `internal/categorizer/categorizer_strategy_test.go` (lines 176, 232, 280)
- Trigger: Run categorizer tests
- Workaround: None. Tests only verify final result, not logging behavior.
- Root cause: MockLogger implementation does not properly track which logger instance emitted messages when using method chaining (WithError/WithFields).

**Context Loss in Library Functions:**
- Symptoms: Cancellation signals from caller are ignored in PDF parsing.
- Files: `internal/pdfparser/pdfparser.go` (line 30)
- Trigger: Call `ParseWithExtractor()` - it immediately creates a Background context instead of using passed context.
- Workaround: Call `ParseWithExtractorAndCategorizer()` directly with proper context.
- Root cause: Function signature accepts context but discards it for Background context.

---

## Security Considerations

**API Credentials in Debug Logs:**
- Risk: GEMINI_API_KEY is logged at debug level in categorization attempt messages.
- Files: `internal/categorizer/gemini_client.go` (lines 105-109, 131-134)
- Current mitigation: Only at debug level (not default), field value is transaction details not the key itself.
- Recommendations: Remove all credential logging entirely. If needed for troubleshooting, log only presence/absence of key, not its value. Use structured logging that can be redacted.

**Temporary File Predictable Naming:**
- Risk: `pdfparser_helpers.go` line 86 creates temp files with pattern `{inputPath}.txt`. Predictable names in shared directories could allow race conditions or temp file hijacking.
- Files: `internal/pdfparser/pdfparser_helpers.go` (line 86)
- Current mitigation: None - relies on directory permissions only.
- Recommendations: Use `os.CreateTemp()` for random naming. Don't create temp files with predictable names in any directory.

**Hardcoded File Permissions:**
- Risk: Debug file created with 0600 (read/write owner only) but database YAML files also use 0600. If accidentally used in shared directories, could cause permission issues.
- Files: Multiple places using `0600`
- Current mitigation: Only used with files in user-controlled directories.
- Recommendations: Use 0644 for files meant to be shared, keep 0600 for secrets. Document intent clearly.

---

## Performance Bottlenecks

**Concurrent Processor Result Buffering:**
- Problem: `internal/camtparser/concurrent_processor.go` line 72 creates buffered result channel with capacity of `len(entries)`. For large CAMT files with thousands of entries, this holds all results in memory before final slice creation.
- Files: `internal/camtparser/concurrent_processor.go` (line 72)
- Cause: Channel must buffer all results before collecting. No streaming or early output possible.
- Improvement path: Use unbuffered channel and collect results in order as they arrive. For large files (>10k entries), consider streaming output to file instead of loading all in memory.

**PDF Text Extraction Creates Double Temp Files:**
- Problem: `internal/pdfparser/pdfparser.go` creates `os.CreateTemp()` (line 40), while `pdfparser_helpers.go` creates additional `.txt` temp file (line 86). Two temporary files created per PDF.
- Files: `internal/pdfparser/pdfparser.go` (lines 40, 57), `internal/pdfparser/pdfparser_helpers.go` (lines 85-103)
- Cause: Design does not consolidate temporary file handling.
- Improvement path: Use single temporary directory or file. Pass file path through instead of relying on pdftotext's implicit file creation.

**getDefaultLogger() Creates Logger Each Call:**
- Problem: `internal/pdfparser/pdfparser_helpers.go` line 76 creates new logger instance every time it's called (fallback for nil logger).
- Files: `internal/pdfparser/pdfparser_helpers.go` (lines 75-78), called from line 104, 130, 138, 146.
- Cause: No caching or singleton pattern.
- Improvement path: Cache the default logger or ensure logger is always passed. Add metrics to track fallback usage.

---

## Fragile Areas

**Panic in CLI Initialization:**
- Files: `cmd/categorize/categorize.go` (line 27)
- Why fragile: Uses `panic(err)` in `init()` function. Any error marking required flag causes immediate panic instead of graceful error message.
- Safe modification: Replace with `log.Fatal()` or handle error properly. Note: `init()` functions should generally not call `log.Fatal()` either; prefer returning error from a setup function.
- Test coverage: Tests exist (`cmd/categorize/categorize_test.go`) but don't cover the panic case because tests bypass init().

**Temporary File Cleanup with Multiple Defer Blocks:**
- Files: `internal/pdfparser/pdfparser.go` (lines 44-55)
- Why fragile: Two defer blocks clean up the same temporary file. If first defer succeeds in removal, second defer's `tempFile.Close()` happens after file is deleted (not critical but suggests unclear resource ownership).
- Safe modification: Use single defer that closes then removes, or track whether cleanup already happened.
- Test coverage: `internal/pdfparser/pdfparser_test.go` tests basic functionality but not temp file cleanup edge cases.

**String Parsing for Amounts and Dates:**
- Files: `internal/pdfparser/pdfparser_helpers.go` (entire file), `internal/models/transaction.go` (ParseAmount function)
- Why fragile: Heavy use of regex and string manipulation for financial data. Swiss format (DD.MM.YY, Swiss decimal formatting) is error-prone to parse.
- Safe modification: Add comprehensive test coverage for edge cases (different date formats, currency symbols, amounts with spaces, decimal separators). Consider using date/amount parsing libraries.
- Test coverage: Large test file (`internal/pdfparser/pdfparser_test.go` - 1180 lines) but covers happy path more than edge cases.

**Category Store YAML File I/O:**
- Files: `internal/store/store.go`
- Why fragile: Concurrent reads/writes to YAML files without locking. If categorizer auto-learns categories while file is being saved, could corrupt YAML.
- Safe modification: Add file-level locking (sync.RWMutex) or use atomic writes (write to temp, then rename).
- Test coverage: `internal/store/store_test.go` (493 lines) but does not test concurrent access patterns.

---

## Scaling Limits

**Concurrent Processing Threshold Too Low:**
- Current capacity: Sequential processing for <100 entries, concurrent for >=100.
- Limit: At 100-150 entries, concurrent overhead (goroutine creation, channel buffering) may not be worth the benefit. Real concurrency benefit appears only at 500+ entries.
- Scaling path: Benchmark to find optimal threshold (likely 500+). Make threshold configurable. Adjust worker count based on file size and system load.

**Result Channel Buffering:**
- Current capacity: Channels buffer all results in memory (line 72).
- Limit: Large files (>100k transactions) would allocate hundreds of MB for result channel alone.
- Scaling path: Stream results to file incrementally instead of buffering. Use bounded queue with backpressure.

**PDF Parsing Memory Usage:**
- Current capacity: Reads entire PDF as string in memory (line 106 in pdfparser.go).
- Limit: Very large PDF files (>100MB) could cause OOM on systems with limited RAM.
- Scaling path: Stream PDF text processing line-by-line instead of loading entire text. Use io.Reader patterns throughout.

---

## Dependencies at Risk

**External PDF Text Extraction Tool (pdftotext):**
- Risk: Code calls external `pdftotext` command via `exec.Command()` (line 90, pdfparser_helpers.go). If tool not installed, PDF parsing silently fails with generic error.
- Impact: Users without pdftotext cannot use PDF functionality. Error message doesn't guide them to install it.
- Migration plan: Consider Go PDF libraries (pdfio, pdfium-go) that don't require external tools. Or detect missing pdftotext and provide helpful error message.

**Gemini API Availability:**
- Risk: AI categorization completely depends on Google Gemini API availability and quota. Outages block categorization, no built-in retry with backoff.
- Impact: If Gemini API is down, all transactions fall back to keyword/direct mapping. No warning to user about degraded AI availability.
- Migration plan: Add circuit breaker pattern for Gemini calls. Implement exponential backoff for retries. Cache successful categorizations. Allow configurable fallback behavior.

---

## Test Coverage Gaps

**Nil Container Handling:**
- What's not tested: Many commands log fatal errors when container is nil but don't actually verify the log output. Tests use comments like "we're testing it doesn't panic" rather than verifying correct behavior.
- Files: `cmd/camt/camt_test.go` (lines 63-100), `cmd/debit/debit_test.go` (lines 62-94), `cmd/pdf/pdf_test.go` (lines 63-95)
- Risk: Silent failures where fatal errors are logged but not actually propagated. Commands might appear to work when they actually failed initialization.
- Priority: High - initialization failures are critical to catch.

**Concurrent Processing Edge Cases:**
- What's not tested: Race conditions, context cancellation mid-processing, partial result handling if context cancels with workers still running.
- Files: `internal/camtparser/concurrent_processor_test.go`
- Risk: Data loss or corruption under high concurrency or cancellation scenarios.
- Priority: High - concurrency bugs are hard to reproduce and debug.

**PDF Format Detection:**
- What's not tested: Edge cases in Viseca format detection (line 123-149, pdfparser_helpers.go). What if file has some but not all Viseca markers? What if markers appear in transaction description?
- Files: `internal/pdfparser/pdfparser_helpers.go`
- Risk: Wrong parser strategy selected, transactions parsed incorrectly.
- Priority: Medium - affects subset of users with certain PDF formats.

**Error Wrapping and Message Quality:**
- What's not tested: Most functions return `fmt.Errorf(...%w...)` but tests don't verify error chain depth or message clarity. Users see generic "error parsing" without context about which file or field failed.
- Files: Multiple parser packages
- Risk: User confusion when parsing fails. Hard to debug file-specific issues.
- Priority: Medium - affects user experience but not data integrity.

---

## Missing Critical Features

**No Dry-Run Mode:**
- Problem: CLI has no way to preview changes before committing YAML files. Categorizer saves mappings immediately and there's no confirmation step.
- Blocks: Safe categorization workflow. Users can accidentally corrupt category mappings.

**No Batch Error Recovery:**
- Problem: Directory consolidation (pdf/convert.go line 143) silently skips invalid files. No option to retry, no summary of failures, no error report file.
- Blocks: Reliable batch processing of mixed-quality PDF files.

**No Backup of Category Mappings:**
- Problem: YAML files are overwritten without backup. If auto-learned category is wrong, user must manually fix YAML (no undo).
- Blocks: Confidence in using AI categorization feature. Risk of data loss if wrong categories are learned.

---

## Architecture Issues

**Inconsistent Error Handling Patterns:**
- Issue: Some functions return error, others log.Fatal, others log.Warn and continue. No clear strategy for different error severities.
- Files: Throughout codebase
- Impact: Unpredictable behavior. Some errors cause exit, others are silently ignored. Tests struggle to verify error handling.
- Fix approach: Define error hierarchy - which errors should exit immediately vs retry vs continue. Update all error-handling code to follow pattern. Document in CONVENTIONS.md.

**Global Mutable State in Deprecated Code:**
- Issue: `internal/config/config.go` lines 32 and 36 define global logger and config variables. While marked deprecated, code may still reference them.
- Files: `internal/config/config.go` (lines 32, 36)
- Impact: Thread safety concerns, test isolation issues, makes dependency injection harder to enforce.
- Fix approach: Trace all usage of globalConfig, Logger, and deprecated functions. Migrate all call sites to DI container. Remove globals entirely.

---

*Concerns audit: 2026-02-01*
