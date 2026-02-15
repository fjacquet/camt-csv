# Domain Pitfalls: Revolut Parser Enhancements

**Domain:** Go CLI financial converter (CSV/XML → standardized output with multi-format support)
**Researched:** 2026-02-15
**Scope:** Adding transaction-type intelligence, CSV output breaking change, iCompta integration, AI auto-learn safety, multi-currency handling, batch processing
**Confidence:** MEDIUM (WebSearch + codebase analysis; lacks field-tested integration wisdom)

---

## Critical Pitfalls

Mistakes that cause rewrites, data corruption, or silent failures in production.

### Pitfall 1: CSV Breaking Change Without Migration Path

**What goes wrong:**
Moving from 4-column to 35-column CSV output breaks every downstream user depending on the old format. Users with scripts, importers, or automated workflows fail silently — their systems read the new CSV without error but misinterpret columns, corrupting data.

**Why it happens:**

- Columns shift: `Description,Amount,Category,Date` → `BookkeepingNumber,Status,Date,ValueDate,Name,PartyName,...`
- Position-based parsing in user scripts reads wrong columns
- First ~5 columns may still "look right" (date-ish, amount-ish), hiding errors until reconciliation
- No version field in CSV to signal change

**Consequences:**

- Amounts misaligned with descriptions (critical for accounting)
- Categories lost or misplaced
- Dates parse to wrong column (user's script looks for col 4, finds new col 8)
- Users detect issue weeks later during reconciliation → data already corrupted
- Trust eroded; complaint volume spikes

**Prevention:**

- Phase breaking change over 2 releases:
  - **v1.2:** New format as `--output-format new`, old as default
  - **v2.0:** New as default, old deprecated with warning
- Add CSV header indicating version: `# CSV-FORMAT-VERSION=2`
- Provide migration script: `camt-csv migrate-csv old.csv new.csv`
- Document extensively in CHANGELOG and CLI help
- Emit warning during conversion: "Using legacy 4-column format. Use `--output-format new` for 35-column"

**Detection:**

- Version bump (major) explicitly signals breaking change
- User reports: "amounts don't match" or "categories are in wrong column"
- Downstream tool crashes because column count changed
- CSV structure tests fail when old parsers try new format

**Phase flag:** MUST address in Phase 1 (Planning). Implement versioning before adding columns.

---

### Pitfall 2: Multi-Currency Exchange Rate Rounding Cascade

**What goes wrong:**
Revolut exports exchange rates to 4-6 decimals. User applies rate in CHF conversion, system rounds to 2 decimals. Later, AI categorizer learns from rounded amount. User applies same tx to another account with microscopically different rate → amount diverges. Reconciliation fails with "unexplained 0.03 difference" that cascades across 100+ paired transactions.

**Why it happens:**

- Float precision degrades over multiple calculations
- `decimal.Decimal` exists in codebase but rounding rules undefined
- Paired transactions across 2 currency files (CHF-EUR, EUR-USD) each apply rounding independently
- AI auto-learn stores rounded values → learns wrong pattern
- No audit trail of rounding decisions

**Consequences:**

- 0.01-0.05 CHF discrepancies across paired transactions
- User cannot reconcile because system shows balanced but bank shows ±0.02
- AI categorizer learns "EUR transfers are 0.02 less than stated" — wrong precedent
- Multi-currency reconciliation audit becomes manual nightmare
- Trust in data integrity drops; user rejects automated categorization

**Prevention:**

- Always use `decimal.Decimal` for financial amounts (codebase already does)
- Document rounding rules in comments:

  ```go
  // Round exchange-based amounts to 2 decimals (CHF precision)
  // before storing or auto-learning.
  // Original 6-decimal rate preserved in ExchangeRate field for audit.
  converted := amount.Mul(rate).RoundBank(2)
  ```

- Pair transactions BEFORE rounding:
  - Read both currency files into memory
  - Match CHF↔EUR pairs by date, amount, merchant
  - Round both symmetrically using same rate
- For AI auto-learn: Store `OriginalAmount` (pre-rounding) so categorizer doesn't learn drift
- Add rounding audit fields:

  ```go
  Transaction{
    OriginalAmount: decimal.NewFromString("100.567890"),
    Amount: decimal.NewFromString("100.57"),  // Rounded
    ExchangeRate: decimal.NewFromString("0.925432"),
    RoundingApplied: "banker's rounding to 2 decimals"
  }
  ```

**Detection:**

- Reconciliation test: sum of exports vs. sum of output differs by >0.01 per 1000 txs
- Paired transaction test: CHF amount + EUR amount (at rate) != expected total
- AI auto-learn test: observed amount drift >0.01 CHF after 50+ categorizations
- Audit: compare OriginalAmount vs. Amount; if drift accumulates, rounding cascade underway

**Phase flag:** Phase 2 (Multi-Currency Handling). Requires rate-pairing logic before auto-learn in Phase 4.

---

### Pitfall 3: YAML Auto-Learn Database Corruption Under Concurrent Writes

**What goes wrong:**
Multiple batch processes run in parallel, each learning a new category from AI. Both read categories.yaml (shared rule set), both add new rules, both write back. Last write wins; first process's learning is lost. Then during next run, different categories merge inconsistently. Over time, YAML becomes jumbled with duplicates, orphaned rules, or conflicting mappings.

**Why it happens:**

- `store.go` reads/writes YAML without file locking
- Batch process 1: reads 50 rules, learns "Spotify → Music"
- Batch process 2: reads 50 rules, learns "Spotify → Entertainment"
- Process 1 writes 51 rules (with Music)
- Process 2 writes 51 rules (with Entertainment, losing Music rule)
- No atomic write; YAML truncate-and-rewrite is not atomic on ext4/APFS
- Concurrent map updates if same categorizer instance shared

**Consequences:**

- Lost categorization rules (first process's learning vanishes)
- Conflicting rules for same merchant (user sees different categories on reruns)
- YAML file becomes partially corrupted (malformed on disk)
- Race detector catches `map concurrent read/write` panic in test mode
- User reruns batch, gets different results → loses trust in categorization

**Prevention:**

- **File-level locking** (required):
  - Use `flock(2)` or equivalent; Go `os.Mkdir` for temp lock dir, `defer os.RemoveAll`
  - Wrap all `store.yaml` read/write in lock

  ```go
  lock := acquireFileLock(dbPath)
  defer lock.Release()
  rules, _ := LoadRules()
  // ... modify rules ...
  SaveRules(rules)  // Safe now
  ```

- **In-memory safety** (if Container shared across goroutines):
  - DI Container creates ONE categorizer per app lifetime
  - Categorizer holds `sync.RWMutex` on category store
  - All reads/writes through mutex

  ```go
  type Categorizer struct {
    mu sync.RWMutex
    rules map[string]string
  }
  func (c *Categorizer) Learn(desc, cat string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.rules[desc] = cat
    // ... write back to YAML with lock held
  }
  ```

- **Batch safety**:
  - Batch process: load rules once at start
  - Apply all categorizations in-memory
  - Write results (categorized txs) to output CSV
  - Write learned rules LAST, after all reads complete
  - Use single worker pool (MaxConcurrentOperations = 10) to limit parallel learners
- **Testing**:
  - Add `go test -race -run TestBatchConcurrency` to CI
  - Test: 3 batch processes on same file simultaneously
  - Assert: no data loss, no corruption, consistent categorization

**Detection:**

- YAML won't parse (`yaml.Unmarshal` errors)
- Race detector catches `concurrent map read`
- Categorization differs between reruns (same tx, different category)
- Learning output differs by run (rules missing from expected set)

**Phase flag:** Phase 3 (Batch Processing) and Phase 4 (AI Auto-Learn). File locking REQUIRED before any multi-process feature. Add integration test with concurrent batch runs.

---

### Pitfall 4: Transaction Type Semantic Expansion Breaks Existing Parsing

**What goes wrong:**
Current Revolut parser has flat structure: parse each row as CSV → convert to Transaction. Now adding transaction type intelligence: detect "Transfer to CHF Vacances" as special semantically (not just string matching). You add Type field parsing, introduce type enum (PAYMENT, TRANSFER, EXCHANGE, INVESTMENT, etc.). Existing hardcoded special case `"To CHF Vacances" → Vacances category` breaks when new type parsing runs first and changes Description field upstream.

**Why it happens:**

- Current code: parse CSV → check description for hardcoded string → apply category
- New code flow: parse CSV → detect type → **transform description** → check for special case
- Type detection consumes/modifies raw description (extracts type info)
- Existing hardcoded check now fails because description no longer contains "To CHF Vacances"
- Test suite passes (new logic works), but integration test on real Revolut CSV fails

**Consequences:**

- Revolut transactions with type information lose special handling
- "To CHF Vacances" → no longer categorized as Vacances
- User data shows: same merchant, different categories before/after upgrade
- Backward compat broken for users with rules based on old descriptions

**Prevention:**

- **Preserve raw description**:

  ```go
  type Transaction{
    Description string // Original, unmodified
    DescriptionParsed string // After type extraction
    Type string // Extracted type: TRANSFER, PAYMENT, etc.
  }
  ```

  - Type parsing extracts info INTO Type field, NOT into Description
  - Special case logic checks original Description, not parsed
- **Phase type detection**: Add type detection AFTER special case handling, not before

  ```go
  // BAD: type detection first
  rawDesc := row.Description
  t.Type = detectType(rawDesc) // Modifies description
  if specialCase(t.Description) { ... } // Now fails!

  // GOOD: special cases first
  if specialCase(row.Description) {
    applySpecialLogic()
  }
  // THEN detect type (non-destructive)
  t.Type = detectType(row.Description) // Only reads, doesn't modify
  ```

- **Test coverage**:
  - Add test: "Revolut special case 'To CHF Vacances' still categorizes as Vacances after type detection"
  - Table-driven test with known Revolut descriptions → verify Description unchanged
- **Configuration for type detection**:
  - Make type detection optional (feature flag):

    ```toml
    [features]
    revolut_type_detection = false  # Default off until verified
    ```

**Detection:**

- Regression test fails: "Special case merchants (e.g., 'Vacances') uncategorized"
- User reports: same category data, different outputs before/after upgrade
- Integration test on real Revolut CSV: transactions with known types now missing category

**Phase flag:** Phase 2 (Revolut Type Intelligence). MUST preserve raw descriptions and test special cases before merge.

---

### Pitfall 5: iCompta Import Silent Data Loss (Amount Field Type Mismatch)

**What goes wrong:**
iCompta expects amount fields as TEXT (not float), per docs. Exporter outputs `100.50` as string. Importer reads from Go CSV output with 35 columns. If amount is exported as `100.50` but iCompta's parser expects `100,50` (comma decimal), or if field count mismatches, iCompta silently skips malformed rows or imports with wrong amount.

**Why it happens:**

- iCompta CSV docs specify TEXT fields, not strict format
- Users import in different locales: DE uses `,`, EN uses `.`
- Go CSV outputs with `,` delimiter, iCompta may expect `;`
- Column count: new 35-column format, iCompta importer mapped to old 4-column positions
- No validation in iCompta on import (bad data doesn't error, just skips)
- User notices discrepancy only during reconciliation

**Consequences:**

- Transactions silently skipped in iCompta (amount field parsing failed)
- User sees fewer transactions in iCompta than in export file
- Amounts incorrect (wrong decimal separator interpreted)
- Date parsing fails (iCompta uses MM/dd/yyyy by default, not DD.MM.YYYY)
- User data integrity compromised; trust in tool lost

**Prevention:**

- **Explicit iCompta output format**:
  - Add `--output-format icompta` mode
  - Format rules (from iCompta docs):
    - Amounts as TEXT: `"100.50"` (never float)
    - Dates as TEXT: `"DD/MM/YYYY"` (per iCompta default, case-sensitive)
    - Delimiter: `,` (comma)
    - Quote all text fields
    - Category must match iCompta's hierarchical structure (e.g., "Income > Salary")

  ```go
  if format == "icompta" {
    tx.Category = mapToIComptaHierarchy(tx.Category)
    date = tx.Date.Format("02/01/2006") // DD/MM/YYYY
    amount = strconv.FormatFloat(tx.Amount, 'f', 2, 64) // Explicit 2 decimals
  }
  ```

- **Validation**:
  - Test iCompta import: export CSV, manually import in iCompta, verify counts
  - Add integration test: export 50 txs, count in CSV, count in exported iCompta file
  - Assert: no silent skips, all amounts match
- **Documentation**:
  - Explicit iCompta setup guide in CLAUDE.md
  - Example: `camt-csv revolut --input X.csv --output icompta.csv --output-format icompta`
  - Note: manual mapping of merchant → iCompta category required (no auto-learn for iCompta)

**Detection:**

- User imports CSV to iCompta, sees fewer rows than in original file
- Amounts off by decimal separator (100.50 read as 10050)
- Dates unrecognized (shown as blank or default)
- iCompta import wizard reports "X rows skipped due to format error"

**Phase flag:** Phase 3 (iCompta Output). Requires explicit format handling, NOT inferred. Must test with real iCompta importer.

---

### Pitfall 6: Batch Processing Partial Failure Without Rollback

**What goes wrong:**
Batch process converts 200 Revolut files to standardized CSV. File 150 fails (corrupt data). Batch continues, produces 199 output files. User sees outputs, assumes all 200 succeeded, imports into iCompta. One missing month of data goes unnoticed until reconciliation. No indication which file failed. Logging shows error, but user may not have reviewed logs.

**Why it happens:**

- Worker pool pattern in Go (as documented) processes files concurrently, collects errors
- One file error doesn't stop batch (by design, for robustness)
- Output files written immediately (not staged)
- No manifest or summary at end: "Processed 199/200, failed: [file150.csv]"
- User checks output directory, sees 199 files, assumes success

**Consequences:**

- Silent data loss (one month missing)
- User detects issue weeks later
- Incomplete dataset imported into finance app
- Reconciliation effort; financial reporting delayed
- User loses confidence in batch processing

**Prevention:**

- **Manifest file** (required):
  - Batch process writes summary file after all workers complete:

  ```json
  {
    "timestamp": "2025-02-15T10:00:00Z",
    "total": 200,
    "succeeded": 199,
    "failed": 1,
    "failed_files": [
      {
        "input": "revolut_march_2025.csv",
        "error": "CSV parsing error: invalid date format on row 45"
      }
    ]
  }
  ```

- **Exit code**:
  - `camt-csv batch ... && echo "All succeeded"` → exit 0 only if all succeeded
  - Exit non-zero if any failed (even if some succeeded)
  - User script can check exit code
- **User output**:
  - Print to stdout at end:

    ```
    Batch complete: 200 processed, 199 succeeded, 1 failed
    Failed files:
      - revolut_march_2025.csv: CSV parsing error (row 45)
    See batch-result.json for details
    ```

- **Optional: Staging**:
  - Write outputs to temp directory first
  - Only move to final directory if all succeeded (all-or-nothing semantics)
  - Prevents partial output from being used accidentally

**Detection:**

- Batch completes but manifest shows failures
- Exit code non-zero despite some files processed
- User script detects manifest and halts further processing
- File count mismatch: input files != output files

**Phase flag:** Phase 3 (Batch Processing). Must implement before release; non-negotiable for data integrity.

---

### Pitfall 7: Feature Flag Inconsistency Across Processors

**What goes wrong:**
Config enables AI categorization: `ai.enabled = true`. Revolut parser reads config at startup, initializes AI client. Batch processor runs 10 workers, each loading config independently. Worker 5 reads config after user edits it to `ai.enabled = false`, but Workers 1-4 already started with `true`. Mixed behavior: some txs categorized with AI, others not. No indication which is which. User reruns batch, gets different categorizations.

**Why it happens:**

- Config loaded at container creation (v1.1 pattern)
- No config refresh for long-running batch processes
- Feature flags read directly from config, not through explicit interface
- No "feature flag version" or snapshot at batch start
- User edits config file mid-batch; changes take effect randomly across workers

**Consequences:**

- Inconsistent categorization within single batch output
- Same transaction, different category on rerun
- Difficult to debug (some workers follow old rules, some new)
- Auto-learn stores inconsistent training data
- User has no way to reproduce results

**Prevention:**

- **Snapshot config at batch start**:

  ```go
  func BatchConvert(inputDir, outputDir string) error {
    // Capture config snapshot ONCE at start
    configSnapshot := container.GetConfig().Clone()

    // Pass snapshot to all workers
    for i := 0; i < numWorkers; i++ {
      go workerProcess(files[i], configSnapshot) // Not container.GetConfig()
    }
  }
  ```

- **Feature flag interface** (not direct config reads):

  ```go
  type FeatureFlags interface {
    IsAICategorizer() bool
    IsTypeDetection() bool
  }

  // Pass flags to all categorizers, not config
  categorizer := container.GetCategorizer()
  categorizer.SetFeatureFlags(flags) // Frozen at batch start
  ```

- **Manifest logs feature state**:

  ```json
  {
    "features_at_start": {
      "ai_enabled": true,
      "type_detection": false
    }
  }
  ```

- **Testing**:
  - Test: Batch with `ai.enabled = true`, verify all txs categorized via AI
  - Test: Concurrent batch with feature flag change mid-batch, verify inconsistency (red test), then fix

**Detection:**

- Categorizations differ within single batch output
- Same merchant, different categories in output CSV
- Re-running same batch with no input changes produces different results
- Log analysis: some workers show AI categorization, others don't

**Phase flag:** Phase 3 (Batch Processing) and Phase 4 (AI Auto-Learn). Config snapshot required for reproducibility.

---

## Moderate Pitfalls

Mistakes that cause rework but not data loss or complete system failure.

### Pitfall 8: Paired Transaction Matching Complexity in Exchange Handling

**What goes wrong:**
User exports Revolut in 3 currencies: CHF, EUR, GBP. Exchange CHF→EUR creates paired entries:

- CHF file: "Exchange -100 CHF → 92 EUR"
- EUR file: "Exchange +92 EUR ← from CHF"

Batch process needs to match pairs, apply exchange rates consistently. But if files arrive in wrong order, or amounts differ by 0.01 due to rounding, matching fails. Transaction appears in both file outputs (duplicate) or is categorized twice (once per currency view).

**Why it happens:**

- Matching logic assumes exact amount match; rounding differences break matching
- File processing order matters; if EUR processed before CHF, EUR already categorized, CHF can't link back
- Timestamp differences: Revolut shows slightly different times in each currency file
- No deduplication across files; user must manually identify duplicates

**Consequences:**

- Duplicate transactions in consolidated export
- Exchange transaction appears in both CHF and EUR categories (double-counting)
- User has to manually deduplicate
- Exchange tracking (for tax purposes) becomes unreliable

**Prevention:**

- **Delayed categorization**:
  - Parse all files first (don't categorize yet)
  - Match paired transactions across files (by date, amount range, merchant)
  - Mark paired transactions: `PairedWith: "eur_123"` field
  - THEN categorize, using pairing info to avoid double-counting
- **Amount tolerance**:
  - Match if amounts within 0.05 CHF (rounding tolerance)
  - Use rate-adjusted comparison: `CHF_amount * rate ≈ EUR_amount` (within 0.01)

  ```go
  func matchPair(chf, eur decimal.Decimal, rate decimal.Decimal) bool {
    adjusted := chf.Mul(rate).RoundBank(2)
    return adjusted.Sub(eur).Abs().LessThanOrEqual(decimal.NewFromFloat(0.01))
  }
  ```

- **Deduplication**:
  - After all files parsed, mark one side of pair as "source", other as "duplicate"
  - Output only source (deduplicated)
  - Log which transactions were marked duplicate
- **Testing**:
  - Test: 3-currency export, manually verify pairing
  - Test: rounding differences (100.567 CHF → 92.02 EUR), ensure matching works

**Detection:**

- CSV has duplicate transactions (same date, merchant, opposite amounts)
- User reports: "Exchange transaction shows in both CHF and EUR categories"
- Consolidated export total differs from bank balance by exchange difference

**Phase flag:** Phase 2 (Multi-Currency Handling). Pair-matching logic REQUIRED before batch processing.

---

### Pitfall 9: iCompta Category Hierarchy Mismatch

**What goes wrong:**
Standard categories: "Shopping", "Food", "Groceries". iCompta hierarchies: "Expenses > Food > Groceries". CSV export lists categories flatly. iCompta importer tries to match "Shopping" to hierarchy, fails or picks wrong parent. User manually remaps all 50 transactions after import.

**Why it happens:**

- iCompta uses 3-level hierarchies; standardized export uses flat categories
- No transformation layer from flat → hierarchical
- iCompta docs show examples but no authoritative list
- User's custom iCompta categories differ from default

**Consequences:**

- Post-import remapping required (manual work)
- Wrong parent category assigned (categorization seems grouped wrong)
- User won't use auto-import for future batches

**Prevention:**

- **Category mapping file**:
  - `database/icompta_categories.yaml`:

    ```yaml
    Shopping: "Expenses > Shopping"
    Groceries: "Expenses > Food > Groceries"
    Salary: "Income > Salary"
    ```

  - If no mapping found, use default parent: "Expenses > {Category}"
- **Validation**:
  - Before iCompta output, verify all categories have mappings
  - Warn if unmapped category found
  - Add to mapping file with default
- **Documentation**:
  - CLAUDE.md: "Setting up iCompta category mapping"
  - Example: import iCompta's category list, auto-generate mapping

**Detection:**

- iCompta import wizard shows category mismatches
- User needs to manually reassign transactions
- Log shows unmapped categories

**Phase flag:** Phase 3 (iCompta Output). Establish category mapping before implementing export format.

---

### Pitfall 10: Type Detection Over-Eagerness (Precision Loss)

**What goes wrong:**
Type detection tries to infer PAYMENT, TRANSFER, EXCHANGE from description. Rule: if description contains amount + currency pair, it's an EXCHANGE. But "Spotify EUR 12.99" matches (currency mentioned) → misclassified as EXCHANGE, loses PAYMENT type. Auto-categorizer later expects EXCHANGE pattern, doesn't find it. Category inference breaks.

**Why it happens:**

- Regex too broad: looks for any `currency1 + currency2`
- EXCHANGE detection runs before PAYMENT detection
- Once type assigned, not reconsidered
- No confidence score; ambiguous cases forced into wrong type

**Consequences:**

- Type misclassification for merchants mentioning currencies
- Categorization breaks downstream (expects different type fields)
- User sees "Spotify: Exchange" (wrong) instead of "Spotify: Payment"

**Prevention:**

- **Ordered detection by confidence**:

  ```go
  detectors := []TypeDetector{
    DetectInvestment(), // Highest confidence (keywords: buy, sell)
    DetectTransfer(),   // High confidence (keywords: transfer, wire)
    DetectExchange(),   // Medium confidence (2 currencies, no account context)
    DetectPayment(),    // Default (catchall)
  }

  for _, detector := range detectors {
    if confidence := detector(desc); confidence > threshold {
      return detector.Type()
    }
  }
  ```

- **Confidence scoring**:

  ```go
  type TypeDetection struct {
    Type string
    Confidence float64 // 0.0-1.0
    Reason string
  }
  ```

- **Fallback**: if no high-confidence match, assign PAYMENT (default)
- **Testing**:
  - Test: "Spotify EUR 12.99" → PAYMENT (not EXCHANGE)
  - Test: "CHF 100 → EUR 92" → EXCHANGE
  - Test: ambiguous cases default to PAYMENT

**Detection:**

- Type misclassifications in test suite (e.g., Spotify as EXCHANGE)
- Categorization precision drops after type detection feature added

**Phase flag:** Phase 2 (Revolut Type Intelligence). Type detection accuracy critical before release.

---

## Minor Pitfalls

### Pitfall 11: CSV Quote Handling in Descriptions with Commas

**What goes wrong:**
Description: `John "Johnny" Doe's Pizza Place`. CSV output quotes it: `"John ""Johnny"" Doe's Pizza Place"`. iCompta importer reads it back as: `John "Johnny" Doe's Pizza Place` (double-quotes consumed correctly) OR `John ""Johnny"" Doe's` (truncated at quote edge). Parsing is fragile if edge cases not tested.

**Prevention:**

- Use standard CSV library (Go's `encoding/csv`) for all output
- Test: descriptions with quotes, commas, newlines, special chars
- Manual test: round-trip CSV (export → import → verify) with edge-case descriptions

---

### Pitfall 12: Locale-Dependent Amount Parsing in Batch

**What goes wrong:**
User's system locale is DE (German), uses `,` as decimal separator. Batch process parses "100,50" as one hundred point fifty (decimal.Decimal), but user's importer in iCompta (also German locale) expects `100,50` and reads it as one hundred fifty. Amounts triple.

**Prevention:**

- Always use explicit decimal format (don't rely on locale)
- Parse/format with explicit `.` separator in Go (use `decimal.NewFromString`)
- Document: amounts in output always use `.` decimal, regardless of system locale

---

### Pitfall 13: Feature Flag Persistence Across Container Reloads

**What goes wrong:**
User sets `--ai-enabled=false` on first run. Config is cached in Container (singleton). User stops tool, edits config to `ai.enabled = true`. User reruns. Container is recreated, loads new config, but old singleton Container with old config still referenced by dangling goroutines. Behavior inconsistent.

**Prevention:**

- Container is stateless wrapper around config (no global mutable state)
- Config reloaded on each CLI command entry
- No persistent singletons across invocations
- (This codebase already follows this pattern; just ensure batch processors don't cache old Container)

---

## Phase-Specific Warnings

| Phase | Topic | Likely Pitfall | Mitigation | Why |
|-------|-------|---|---|---|
| 1: Planning | CSV Breaking Change | Silent data corruption in user scripts | Implement versioning, migration script, deprecation warning | Users won't notice until reconciliation |
| 2: Revolut Type | Description Preservation | Special case handling breaks | Keep raw Description unchanged, test special cases | New parsing can destroy context old code relied on |
| 2: Revolut Type | Type Detection Accuracy | Misclassification (PAYMENT as EXCHANGE) | High-confidence detection, fallback to PAYMENT | Downstream categorization depends on correct type |
| 2: Multi-Currency | Exchange Rate Rounding | Cascade of 0.01-0.05 CHF drifts | Use decimal.Decimal, store original amounts, pair before rounding | Rounding compounds; affects AI learning |
| 2: Multi-Currency | Paired Transaction Matching | Duplicates across currency files | Match pairs before categorizing, deduplicate | Exchange txs appear in multiple files |
| 3: Batch Processing | Partial Failure | Silent data loss (missed month) | Manifest file, exit codes, user output | User won't detect missing data until later |
| 3: Batch Processing | Concurrency in YAML | Data corruption, rule loss | File locking, in-memory mutex | Multiple workers writing same file simultaneously |
| 3: iCompta Output | Amount Format | Silent skip in iCompta importer | Explicit iCompta format mode, amount as TEXT | iCompta validates badly; wrong format skipped silently |
| 3: iCompta Output | Category Hierarchy | Manual remapping after import | Category mapping file, validation | iCompta hierarchies don't match flat export |
| 4: AI Auto-Learn | Feature Flag Inconsistency | Inconsistent categorizations in one batch | Config snapshot at batch start | Workers see different config mid-batch |
| 4: AI Auto-Learn | Database Corruption | YAML overwrites, rule loss | File locking + mutex, integration test with concurrent processes | Concurrent writes to YAML corrupt file |

---

## Integration Pitfalls (Cross-Phase)

### Issue: AI Learning on Incorrectly Rounded Data

**Setup:** Phase 2 (exchange rounding) + Phase 4 (AI auto-learn)

If Phase 2 rounds exchange amounts before Phase 4 learns, categorizer trains on rounded values, learns incorrect patterns. Later, when user sees original Revolut amount (100.567 CHF), categorizer can't match it (it learned 100.57).

**Prevention:** Phase 2 MUST store OriginalAmount unchanged. Phase 4 learns from OriginalAmount, not rounded Amount.

---

### Issue: Batch Manifest Doesn't Reflect Feature Flags

**Setup:** Phase 3 (batch) + Phase 4 (AI learning)

Manifest says "200 succeeded" but doesn't log which ones used AI vs. keyword matching. User reruns with different AI flag, gets different results, unsure which manifest corresponds to which feature state.

**Prevention:** Manifest includes `features_snapshot` and feature-level categorization stats (e.g., "195 AI, 5 keyword").

---

## Success Criteria for Pitfall Prevention

By end of research/planning:

- [ ] CSV breaking change has explicit migration path (2-release deprecation)
- [ ] Multi-currency rounding strategy documented (use decimal, store originals)
- [ ] Paired transaction matching algorithm outlined (before categorize)
- [ ] File locking strategy for YAML writes specified
- [ ] Batch manifest format design completed
- [ ] Feature flag snapshot approach defined
- [ ] iCompta format mode (separate from standard) planned
- [ ] Category hierarchy mapping structure chosen
- [ ] Type detection accuracy requirements stated (edge cases listed)
- [ ] Integration test plan covers: concurrent batch + YAML, iCompta import, exchange matching

## Sources

- [Backward Compatibility in Schema Evolution: Guide](https://www.dataexpert.io/blog/backward-compatibility-schema-evolution-guide)
- [How to handle money and currency conversions – Software Engineering Tips](http://www.yacoset.com/how-to-handle-currency-conversions/)
- [Decimal precision in currency and pricing | Microsoft Learn](https://learn.microsoft.com/en-us/dynamics365/sales/decimal-precision-currency-pricing)
- [Mastering Monetary Operations: Navigating Currency Handling in Java](https://www.altimetrik.com/blog/modeling-money-in-java-pitfalls-solutions)
- [iCompta | Your accounts with ease](https://www.icompta-app.com/help.php)
- [Mastering the Go Worker Pattern for Efficient Parallel Processing](https://medium.com/@divyankpandey9806/mastering-the-go-worker-pattern-for-efficient-parallel-processing-f5e9ca482446)
- [Concurrency in Go: Race Conditions, Deadlocks, and Common Pitfalls](https://medium.com/@nagarjun_nagesh/concurrency-in-go-race-conditions-deadlocks-and-common-pitfalls-52243faf1a2f)
- [AI-Powered Progressive Delivery: How Intelligent Feature Flags Are Redefining Software Releases in 2026](https://azati.ai/blog/ai-powered-progressive-delivery-feature-flags-2026/)
