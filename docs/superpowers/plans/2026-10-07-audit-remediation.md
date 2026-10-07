# Audit Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cut AI calls and YAML churn, remove dead/duplicated code, and close the CSV formula-injection gap, without changing amounts or row counts of any conversion.

**Architecture:** Six sequential PRs (A → F), each on its own branch from an up-to-date `main`, each merged before the next starts. Categorization changes concentrate in `internal/categorizer` (one place that records results: cache, learning, staging) and `internal/common/categorization_helper.go` (the single per-parser loop, later batch-aware).

**Tech Stack:** Go 1.27.1, testify (assert/require/mock), shopspring/decimal, gocsv, yaml.v3, golangci-lint, GitHub PRs (main is protected).

**Spec:** `docs/superpowers/specs/2026-10-07-audit-remediation-design.md`

## Global Constraints

- Money and quantities stay `shopspring/decimal`; never float/int.
- Never use `math/rand` (Semgrep blocks it).
- `icompta` formatter output must stay byte-identical (iCompta resolves columns by name; `TestIComptaHeaderCoversPluginMappings` must keep passing).
- Numeric and date CSV columns are never formula-escaped (`-180.00` must stay `-180.00`).
- Tests use `t.TempDir()`; `TEST_MODE=true` for any CLI run; never touch `~/Desktop/ic25.cdb` or `database/*.yaml` from tests.
- Every PR: `make test` and `make lint` green, CHANGELOG entry under `## [Unreleased]`, PR via `gh pr create`, commit messages end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`, PR bodies end with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- The user's uncommitted `database/creditors.yaml` and `database/debtors.yaml` are never staged (`git add` explicit paths only).
- Regression check per PR (Task R): convert `work/in/revolut` with `TEST_MODE=true` before and after; row count and amount sum identical.

## Review Focus

1. **Party name with only whitespace** reaching the batch/AI path — expected: Uncategorized, never an AI call. Test in Task D2.
2. **AI answer differing only in case or accents-free spelling** (`courses` vs `Courses`) — expected: accepted, canonical spelling returned. Test in Task B3.
3. **Staging file missing or corrupt at startup** — expected: staged tier empty, run continues, flush recreates the file. Test in Task B4.
4. **Same party as debtor and creditor in one batch** — expected: two separate cache entries and two answers. Test in Task D2.
5. **Text cell that is already apostrophe-escaped (`'=x`)** — expected: escaped again so it round-trips. Test in Task E1.

---

## Task R: Regression baseline (run before PR A, and at the end of every PR)

**Files:** none (scratch output only).

- [ ] **Step 1: Capture baseline on main**

```bash
cd /Users/fjacquet/Projects/camt-csv
git checkout main && git pull
make build
mkdir -p /tmp/camt-regress && rm -f /tmp/camt-regress/*
TEST_MODE=true CAMT_AI_ENABLED=false ./camt-csv convert -i work/in/revolut -o /tmp/camt-regress/base.csv
git checkout -- database/creditors.yaml database/debtors.yaml 2>/dev/null; git stash list >/dev/null
```

Note: if the convert run modified `database/*.yaml`, restore ONLY the lines it added (the user has uncommitted edits there); compare `git diff --stat database/` before and after the run.

- [ ] **Step 2: Record row counts and sums**

```bash
python3 -I - /tmp/camt-regress <<'EOF'
import csv,sys,glob
from decimal import Decimal as D
for f in sorted(glob.glob(sys.argv[1]+'/base_*.csv')):
    rows=list(csv.DictReader(open(f,encoding='utf-8'),delimiter=';'))
    print(f.split('/')[-1], len(rows), sum(D(r['Amount']) for r in rows))
EOF
```

Expected today: `base_revolut-chf.csv 161 229.35`, `base_revolut-eur.csv 2 1.50`.

- [ ] **Step 3: After each PR**, rerun with output `after.csv` and the same script on `after_*.csv`. Counts and sums must match Step 2.

---

# PR A — dead code and silent CAMT fallback

Branch: `git checkout main && git pull && git checkout -b chore/remove-dead-code`

### Task A1: Delete compliance scaffolding and constitution loader

**Files:**
- Delete: `internal/models/models.go`, `internal/parser/constitution.go`, `internal/parser/constitution_test.go`
- Modify: `internal/config/viper.go` (remove `Constitution` struct field ~90-92, `BindEnv("constitution.file_paths", ...)` ~149-150, default ~220-221)
- Modify: `docs/architecture.md:181` (drop the `constitution.go` line)

- [ ] **Step 1: Confirm no production references**

```bash
for s in NewComplianceReport ComplianceReport ConstitutionPrinciple ConstitutionLoader NewConstitutionLoader Constitution; do
  echo "== $s"; grep -rnw "$s" --include='*.go' . | grep -v _test.go
done
```
Expected: only definitions in the files being deleted, plus `Constitution` in `internal/config/viper.go`.

- [ ] **Step 2: Delete files and config keys**

```bash
git rm internal/models/models.go internal/parser/constitution.go internal/parser/constitution_test.go
```
Then edit `internal/config/viper.go`: remove the `Constitution struct { FilePaths []string ... }` field from the config struct, the `BindEnv("constitution.file_paths", "CAMT_CONSTITUTION_FILE_PATHS")` call, and the `SetDefault("constitution.file_paths", ...)` call. Remove the `constitution.go` line from `docs/architecture.md`.

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./internal/models/ ./internal/parser/ ./internal/config/`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add -A internal/models internal/parser internal/config docs/architecture.md
git commit -m "chore: remove unused compliance scaffolding and constitution config

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task A2: Trim batch aggregator and legacy account extraction

**Files:**
- Modify: `internal/batch/aggregator.go` — keep only `BatchAggregator`, `NewBatchAggregator`, `Consolidate`, `sortTransactionsChronologically`, `detectAndLogDuplicates`, `arePotentialDuplicates`; delete `DateRange` (+`String`, `Merge`), `FileGroup`, `GroupFilesByAccount`, `extractDateRangeFromFilename`, `AggregateTransactions`, `GenerateOutputFilename`, `GenerateSourceFileHeader`, `CalculateDateRangeFromTransactions`; drop now-unused imports (`fmt`, `path/filepath`, `time`, `common`).
- Modify: `internal/batch/aggregator_test.go` — keep `TestProperty_ChronologicalTransactionOrdering`, `TestProperty_DuplicateTransactionPreservation`, `TestConsolidate_SortsAndKeepsDuplicates`, `TestDetectAndLogDuplicates_NonAdjacentSameDateDuplicatesStillFound`, `TestConsolidate_EmptyInput` and helpers `cryptoRandIntn`, `cryptoRandFloat64`, `cryptoShuffle`; delete every other test and helper.
- Modify: `internal/integration/cross_parser_test.go` — delete `TestBatchProcessingWithMixedFileTypes`, `createTestCAMTFile`, `mockParser` (type + method), and the `batch` import.
- Modify: `internal/common/account.go` — delete `AccountIdentifier`, `camtFilenamePattern`, `ExtractAccountFromCAMTFilename`, `ExtractAccountFromFilename`; edit the comments at ~120-125 and ~146-150 that mention them so they describe `AccountKeyFromFilename` on its own.
- Modify: `internal/common/account_test.go` — delete `TestExtractAccountFromCAMTFilename`, `TestProperty_AccountIdentificationFromFilenames`, `TestProperty_FilenameAccountExtraction`, `TestExtractAccountFromFilename`, helpers `cryptoRandFloat32`, `generateRandomDigits`, `generateRandomDate`, `generateRandomFilename`, `generateRandomString`; fix the comment above `TestAccountKeyFromFilename`.

- [ ] **Step 1: Confirm references**

```bash
for s in GroupFilesByAccount AggregateTransactions GenerateOutputFilename GenerateSourceFileHeader CalculateDateRangeFromTransactions ExtractAccountFromFilename ExtractAccountFromCAMTFilename AccountIdentifier; do
  echo "== $s"; grep -rnw "$s" --include='*.go' . | grep -v _test.go
done
```
Expected: only definitions (and `ExtractAccountFromFilename` inside `GroupFilesByAccount`).

- [ ] **Step 2: Apply the deletions listed above.**

- [ ] **Step 3: Build, vet, test**

Run: `go build ./... && go vet ./internal/batch/ ./internal/common/ ./internal/integration/ && go test ./internal/batch/ ./internal/common/ ./internal/integration/`
Expected: PASS, no "imported and not used".

- [ ] **Step 4: Commit**

```bash
git add internal/batch internal/common/account.go internal/common/account_test.go internal/integration
git commit -m "chore: remove unused batch aggregation and filename account helpers

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task A3: Remove dead CSV and debit-parser entry points

**Files:**
- Modify: `internal/common/csv.go` — delete `ReadCSVFile`, `WriteTransactionsToCSV`, `WriteTransactionsToCSVWithLogger`.
- Delete: `internal/common/csv_test.go` (only tests the deleted functions).
- Modify: `internal/common/coverage_test.go` — delete `TestReadCSVFile_*` (≈200-233) and the `WriteTransactionsToCSVWithLogger` tests (≈231-284, ≈412-430).
- Modify: `internal/integration/cross_parser_test.go` — `TestCrossParserConsistency` calls `common.WriteTransactionsToCSVWithLogger` (≈49-55): replace those calls with `common.WriteTransactionsToCSVWithFormatter(txs, path, logger, formatter.NewStandardFormatter(), ',')` (import `fjacquet/camt-csv/internal/formatter`).
- Modify: `internal/debitparser/debitparser.go` — delete `ParseFile`, `ParseFileWithLogger`; drop the `common` import if unused.
- Modify: `internal/debitparser/debitparser_test.go` — delete `TestParseFile`, `TestParseFileWithLogger`, `TestParseFileWithInvalidFile`.
- Modify: `docs/architecture.md:161` — replace the `common.WriteTransactionsToCSV(...)` example with `common.WriteTransactionsToCSVWithFormatter(...)`.

- [ ] **Step 1: Confirm references**

```bash
for s in ReadCSVFile WriteTransactionsToCSV WriteTransactionsToCSVWithLogger ParseFileWithLogger; do
  echo "== $s"; grep -rnw "$s" --include='*.go' . | grep -v _test.go
done
grep -rnw "ParseFile" --include='*.go' internal/debitparser cmd | grep -v _test.go
```
Expected: only the definitions and their mutual calls.

- [ ] **Step 2: Apply the deletions and the test rewrite.**

- [ ] **Step 3: Test**

Run: `go build ./... && go test ./internal/common/ ./internal/debitparser/ ./internal/integration/`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add -A internal/common internal/debitparser internal/integration docs/architecture.md
git commit -m "chore: remove unused CSV writers and debit ParseFile entry points

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task A4: Remove test-only methods

**Files:**
- Modify: `internal/models/transaction.go` — delete `GetAmountAsDecimal` (≈93-100), `SetAmountFromDecimal` (≈102-105), `SetFeesFromDecimal` (≈122-125).
- Modify: `internal/models/transaction_test.go` — delete `TestGetAmountAsDecimal` and the two subtests in `TestTransaction_UncoveredMethods` (≈138-145, ≈183-190).
- Modify: `internal/models/builder_test.go` — replace each `tx.GetAmountAsDecimal()` (≈652, 653, 753, 767, 778, 787) with `tx.Amount`.
- Modify: `internal/categorizer/categorizer.go` — delete exported wrappers `UpdateDebitorCategory` and `UpdateCreditorCategory` (keep the unexported ones).
- Modify: `internal/categorizer/categorizer_di_test.go` — delete `TestCategorizer_UpdateMethods`.
- Modify: `internal/categorizer/keyword.go` — delete `ReloadCategories` and `loadCategories`; remove the `store` field; rename the constructor parameter to `_ CategoryStoreInterface` (keeps all call sites compiling).
- Modify: `internal/categorizer/keyword_test.go` — delete `TestKeywordStrategy_ReloadCategories`.
- Modify: `internal/categorizer/direct_mapping.go` — delete `ReloadMappings`; remove the `store` field; constructor parameter becomes `_ CategoryStoreInterface`.
- Modify: `internal/categorizer/direct_mapping_test.go` — delete `TestDirectMappingStrategy_ReloadMappings` and `TestDirectMappingStrategy_ReloadMappings_RaceCondition`.
- Modify: `docs/developer-guide.md` (≈268, 305, 1109, 1116) — replace `GetAmountAsDecimal()` with `.Amount` and drop the `SetAmountFromDecimal()` mention.

- [ ] **Step 1: Confirm references**

```bash
for s in GetAmountAsDecimal SetAmountFromDecimal SetFeesFromDecimal UpdateDebitorCategory UpdateCreditorCategory ReloadCategories ReloadMappings; do
  echo "== $s"; grep -rnw "$s" --include='*.go' . | grep -v _test.go
done
```
Expected: only definitions.

- [ ] **Step 2: Apply the changes.**

- [ ] **Step 3: Test and lint**

Run: `go test ./internal/models/ ./internal/categorizer/ && make lint`
Expected: PASS, `0 issues`.

- [ ] **Step 4: Commit**

```bash
git add internal/models internal/categorizer docs/developer-guide.md
git commit -m "chore: remove test-only methods from models and categorizer

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task A5: Skip CAMT entries that cannot be built

**Files:**
- Modify: `internal/camtparser/entry_mapping.go` (`entryToTransaction`, ≈12-106)
- Modify: `internal/camtparser/adapter.go` (`Parse`, ≈60-74)
- Test: `internal/camtparser/camtparser_test.go`

**Interfaces:**
- Produces: `func (a *Adapter) entryToTransaction(entry camtEntry, statementAccount string) (models.Transaction, bool)` — `false` means skip.

- [ ] **Step 1: Write the failing test** (append to `camtparser_test.go`)

```go
// An entry the builder rejects (here: no currency) used to come out as a
// zero-valued row. It must be skipped, and the rest of the statement kept.
func TestParse_SkipsEntryThatCannotBeBuilt(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02"><BkToCstmrStmt><Stmt>
<Acct><Id><IBAN>CH1700767000K54293249</IBAN></Id></Acct>
<Ntry><Amt>10.00</Amt><CdtDbtInd>DBIT</CdtDbtInd><Sts>BOOK</Sts><BookgDt><Dt>2026-04-15</Dt></BookgDt><ValDt><Dt>2026-04-15</Dt></ValDt></Ntry>
<Ntry><Amt Ccy="CHF">120.00</Amt><CdtDbtInd>DBIT</CdtDbtInd><Sts>BOOK</Sts><BookgDt><Dt>2026-04-16</Dt></BookgDt><ValDt><Dt>2026-04-16</Dt></ValDt></Ntry>
</Stmt></BkToCstmrStmt></Document>`

	logger := logging.NewMockLogger()
	adapter := NewAdapter(logger)
	transactions, err := adapter.Parse(context.Background(), strings.NewReader(doc))
	require.NoError(t, err)

	require.Len(t, transactions, 1, "the unbuildable entry is skipped, the other kept")
	assert.Equal(t, "120", transactions[0].Amount.Abs().String())
	assert.True(t, logger.HasEntry("WARN", "Skipping CAMT entry that cannot be built"))
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/camtparser/ -run TestParse_SkipsEntryThatCannotBeBuilt -v`
Expected: FAIL (`Len 2` or missing WARN entry). If `HasEntry` uses a different level string, check `internal/logging/mock.go:234` and adjust the level literal only.

- [ ] **Step 3: Implement**

In `entry_mapping.go`, change the signature and replace the fallback block:

```go
// entryToTransaction maps one CAMT.053 statement entry onto a Transaction.
// It reports false when the builder rejects the entry; the caller skips it
// rather than emitting a row with invented or zero values.
func (a *Adapter) entryToTransaction(entry camtEntry, statementAccount string) (models.Transaction, bool) {
```

```go
	transaction, err := builder.Build()
	if err != nil {
		a.GetLogger().WithError(err).Warn("Skipping CAMT entry that cannot be built",
			logging.Field{Key: "account_servicer_ref", Value: entry.AccountServicer.Ref},
			logging.Field{Key: "reference", Value: reference})
		return models.Transaction{}, false
	}
```

and change the final `return transaction` to `return transaction, true`.

In `adapter.go` `Parse`, replace the categorize-and-append line:

```go
			transaction, ok := a.entryToTransaction(entry, statementAccount)
			if !ok {
				continue
			}
			transaction = a.categorizeTransaction(ctx, transaction)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/camtparser/ -v`
Expected: PASS (existing `TestCAMTParser_ErrorScenarios` subtests only assert inside `if len(transactions) > 0`).

- [ ] **Step 5: CHANGELOG** — under `## [Unreleased]`:
  - `### Removed`: `- Remove unused compliance scaffolding, the constitution config keys, legacy batch aggregation and filename helpers, and test-only methods.`
  - `### Fixed`: `- Skip CAMT entries the transaction builder rejects instead of writing an empty row for them.`

- [ ] **Step 6: Commit, regression, PR**

```bash
git add internal/camtparser CHANGELOG.md
git commit -m "fix(camt): skip entries that cannot be built instead of emitting empty rows

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
make test && make lint
```
Run Task R Step 3. Then:
```bash
git push -u origin chore/remove-dead-code
gh pr create --base main --title "chore: remove dead code; skip unbuildable CAMT entries" --body-file <(printf '%s\n\n%s\n' "Spec: docs/superpowers/specs/2026-10-07-audit-remediation-design.md (PR A). Includes the spec and plan." "🤖 Generated with [Claude Code](https://claude.com/claude-code)")
```
Before the first commit of this branch, bring the spec and plan along: `git checkout docs/audit-remediation-spec -- docs/superpowers/specs/2026-10-07-audit-remediation-design.md docs/superpowers/plans/2026-10-07-audit-remediation.md` and commit them as `docs: audit remediation spec and plan`. Wait for CI green, merge with `gh pr merge --merge --delete-branch` (if it returns 502, check `gh pr view --json state`).

---

# PR B — AI cost and AI answer validation

Branch: `git checkout main && git pull && git checkout -b perf/categorizer-ai-cost`

### Task B1: One cache-key function, negative caching

**Files:**
- Modify: `internal/categorizer/categorizer.go`
- Test: `internal/categorizer/categorizer_cost_test.go` (create)

**Interfaces:**
- Produces: `func cacheKey(partyName string, isDebtor bool) string` (package-private), used by B2-B5 and D2.
- Produces test double `countingAI` (in `categorizer_cost_test.go`, package `categorizer`) used by B2-B5 and D2.

- [ ] **Step 1: Write the failing test** (`internal/categorizer/categorizer_cost_test.go`)

```go
package categorizer

import (
	"context"
	"sync"
	"testing"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingAI answers from a fixed map (lowercased party -> category) and counts calls.
type countingAI struct {
	mu      sync.Mutex
	answers map[string]string
	calls   int
}

func (c *countingAI) Categorize(_ context.Context, tx models.Transaction) (models.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	tx.Category = c.answers[strings.ToLower(strings.TrimSpace(tx.PartyName))]
	if tx.Category == "" {
		tx.Category = models.CategoryUncategorized
	}
	return tx, nil
}

func (c *countingAI) GetEmbedding(context.Context, string) ([]float32, error) { return nil, nil }

func (c *countingAI) callCount() int { c.mu.Lock(); defer c.mu.Unlock(); return c.calls }

func newCostCategorizer(t *testing.T, ai *countingAI, autoLearn bool) (*Categorizer, *store.MockCategoryStore) {
	t.Helper()
	st := &store.MockCategoryStore{
		Categories: []models.CategoryConfig{
			{Name: "Courses", Keywords: []string{"MIGROS"}},
			{Name: "Abonnements"},
			{Name: "Restaurants"},
		},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}
	c := NewCategorizer(ai, nil, st, testLogger(), autoLearn, 0.70)
	t.Cleanup(c.Shutdown)
	return c, st
}

func TestCategorize_UncategorizedIsCachedForTheRun(t *testing.T) {
	ai := &countingAI{answers: map[string]string{}}
	c, _ := newCostCategorizer(t, ai, false)

	for i := 0; i < 3; i++ {
		got, err := c.Categorize(context.Background(), "Mystery Shop", true, "-10", "2026-01-01", "")
		require.NoError(t, err)
		assert.Equal(t, models.CategoryUncategorized, got.Name)
	}
	assert.Equal(t, 1, ai.callCount(), "an unknown party reaches the AI once per run")
}
```
Add `"strings"` to the imports.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/categorizer/ -run TestCategorize_UncategorizedIsCachedForTheRun -v`
Expected: FAIL, `expected 1, actual 3`.

- [ ] **Step 3: Implement** in `categorizer.go`

Add:
```go
// cacheKey is the one normalization used for the in-run cache, so lookups,
// stores and invalidations always agree.
func cacheKey(partyName string, isDebtor bool) string {
	return fmt.Sprintf("%s|%v", strings.ToLower(strings.TrimSpace(partyName)), isDebtor)
}
```
In `categorizeTransaction` replace the inline `cacheKey := fmt.Sprintf(...)` with `key := cacheKey(transaction.PartyName, transaction.IsDebtor)` and use `key` in the lookup and store. Store every found result (remove the `if category.Name != "" && category.Name != models.CategoryUncategorized` guard), and before the final `return models.Category{Name: models.CategoryUncategorized, ...}` store that result too:

```go
	uncategorized := models.Category{
		Name:        models.CategoryUncategorized,
		Description: "No categorization strategy succeeded",
	}
	c.batchCacheMu.Lock()
	c.batchCache[key] = uncategorized
	c.batchCacheMu.Unlock()
	return uncategorized, nil
```
In `updateDebitorCategory` / `updateCreditorCategory` replace the inline key with `cacheKey(partyName, true)` / `cacheKey(partyName, false)`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/categorizer/ -v`
Expected: PASS. If an existing test asserted that Uncategorized is NOT cached, update its expectation to the new rule and say so in the commit message.

- [ ] **Step 5: Commit**

```bash
git add internal/categorizer/categorizer.go internal/categorizer/categorizer_cost_test.go
git commit -m "perf(categorizer): cache uncategorized results for the run

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task B2: Learn only from AI, save once per run

**Files:**
- Modify: `internal/categorizer/categorizer.go` (`Categorize`)
- Test: `internal/categorizer/categorizer_cost_test.go`

**Interfaces:**
- Consumes: `countingAI`, `newCostCategorizer` (B1).
- Produces: `func (c *Categorizer) recordLearning(partyName string, isDebtor bool, category models.Category)` — used by D2.

- [ ] **Step 1: Write the failing test**

```go
// countingStore wraps MockCategoryStore and counts saves.
type countingStore struct {
	*store.MockCategoryStore
	creditorSaves, debtorSaves int
}

func (s *countingStore) SaveCreditorMappings(m map[string]string) error {
	s.creditorSaves++
	return s.MockCategoryStore.SaveCreditorMappings(m)
}

func (s *countingStore) SaveDebtorMappings(m map[string]string) error {
	s.debtorSaves++
	return s.MockCategoryStore.SaveDebtorMappings(m)
}

func TestCategorize_AutoLearnOnlyFromAIAndNoSavePerTransaction(t *testing.T) {
	ai := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	st := &countingStore{MockCategoryStore: &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Courses", Keywords: []string{"MIGROS"}}, {Name: "Abonnements"}},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{"coop": "Courses"},
	}}
	c := NewCategorizer(ai, nil, st, testLogger(), true, 0.70)
	t.Cleanup(c.Shutdown)
	ctx := context.Background()

	_, _ = c.Categorize(ctx, "Coop", true, "-5", "2026-01-01", "")       // direct
	_, _ = c.Categorize(ctx, "MIGROS Lausanne", true, "-5", "2026-01-01", "") // keyword
	_, _ = c.Categorize(ctx, "Kiro", true, "-19", "2026-01-01", "")      // ai

	assert.Zero(t, st.debtorSaves, "nothing is written during the run")
	require.NoError(t, c.SaveDebitorsToYAML())
	assert.Equal(t, 1, st.debtorSaves, "one write at the end")

	saved := st.MockCategoryStore.DebtorMappings
	assert.Equal(t, "Abonnements", saved["kiro"], "AI answers are learned")
	assert.NotContains(t, saved, "migros lausanne", "keyword hits are not learned")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/categorizer/ -run TestCategorize_AutoLearnOnlyFromAI -v`
Expected: FAIL (`debtorSaves` > 0, `migros lausanne` present).

- [ ] **Step 3: Implement** — replace the body of `Categorize` after `category, err := c.categorizeTransaction(ctx, transaction)` with:

```go
	if err == nil {
		c.recordLearning(partyName, isDebtor, category)
	}
	return category, err
}

// recordLearning keeps what the AI taught us. Only AI answers are worth
// keeping: direct and keyword hits are already in the YAML files. With
// auto-learn on, the mapping is updated in memory and saved once when the
// command ends (cmd/root finalize); with it off, the answer is staged.
func (c *Categorizer) recordLearning(partyName string, isDebtor bool, category models.Category) {
	if category.Source != "ai" || category.Name == "" || category.Name == models.CategoryUncategorized {
		return
	}
	if !c.isAutoLearnEnabled {
		c.saveStagingSuggestion(partyName, isDebtor, category.Name)
		return
	}
	c.logger.WithFields(
		logging.Field{Key: "party", Value: partyName},
		logging.Field{Key: "category", Value: category.Name},
		logging.Field{Key: "debtor", Value: isDebtor},
	).Info("Auto-learning mapping from AI")
	if isDebtor {
		c.updateDebitorCategory(partyName, category.Name)
	} else {
		c.updateCreditorCategory(partyName, category.Name)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/categorizer/ ./cmd/... -v 2>&1 | tail -30`
Expected: PASS. Existing tests asserting a save per call must be updated to call `SaveCreditorsToYAML`/`SaveDebitorsToYAML` before asserting.

- [ ] **Step 5: Commit**

```bash
git add internal/categorizer
git commit -m "perf(categorizer): learn only AI answers and save mappings once per run

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task B3: Accept only AI answers that name a known category

**Files:**
- Modify: `internal/categorizer/categorizer.go` (struct, `NewCategorizer`, `categorizeTransaction`)
- Test: `internal/categorizer/categorizer_cost_test.go`

**Interfaces:**
- Produces: `func (c *Categorizer) canonicalAICategory(name string) (string, bool)` — used by D2.

- [ ] **Step 1: Write the failing test**

```go
func TestCategorize_AIAnswerMustBeAKnownCategory(t *testing.T) {
	ai := &countingAI{answers: map[string]string{
		"evil":  `=HYPERLINK("http://x")`,
		"lunch": "restaurants", // wrong case, still known
	}}
	c, _ := newCostCategorizer(t, ai, false)
	ctx := context.Background()

	got, err := c.Categorize(ctx, "Evil", true, "-1", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, models.CategoryUncategorized, got.Name)

	got, err = c.Categorize(ctx, "Lunch", true, "-1", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Restaurants", got.Name, "canonical spelling from categories.yaml")
}

func TestCategorize_NoCategoriesLoadedSkipsValidation(t *testing.T) {
	ai := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	st := &store.MockCategoryStore{CreditorMappings: map[string]string{}, DebtorMappings: map[string]string{}}
	c := NewCategorizer(ai, nil, st, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	got, err := c.Categorize(context.Background(), "Kiro", true, "-1", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got.Name, "an empty categories.yaml must not reject every answer")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/categorizer/ -run 'TestCategorize_AIAnswerMustBe|TestCategorize_NoCategoriesLoaded' -v`
Expected: first test FAIL (`=HYPERLINK...` accepted, `restaurants` returned lowercase).

- [ ] **Step 3: Implement**

Add a field to `Categorizer`: `knownCategories map[string]string // lowercased name -> canonical name from categories.yaml`. In `NewCategorizer`, after categories load:

```go
	c.knownCategories = make(map[string]string, len(c.categories))
	for _, cat := range c.categories {
		c.knownCategories[strings.ToLower(strings.TrimSpace(cat.Name))] = cat.Name
	}
```
Add:
```go
// canonicalAICategory accepts an AI answer only if it names a category from
// categories.yaml, returning that category's own spelling. A model can be
// steered by transaction text into answering anything, and an accepted answer
// is written to the CSV and possibly learned for good. With no categories
// loaded there is nothing to check against, so the answer is kept.
func (c *Categorizer) canonicalAICategory(name string) (string, bool) {
	if len(c.knownCategories) == 0 {
		return name, true
	}
	canonical, ok := c.knownCategories[strings.ToLower(strings.TrimSpace(name))]
	return canonical, ok
}
```
In `categorizeTransaction`, inside `if found {`, before logging/caching:

```go
		if category.Source == "ai" {
			canonical, ok := c.canonicalAICategory(category.Name)
			if !ok {
				c.logger.WithFields(
					logging.Field{Key: "party", Value: transaction.PartyName},
					logging.Field{Key: "answer", Value: category.Name},
				).Warn("Rejected AI category not in categories.yaml")
				category = models.Category{Name: models.CategoryUncategorized, Source: "ai"}
			} else {
				category.Name = canonical
			}
		}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/categorizer/ -v 2>&1 | tail -20`
Expected: PASS. Existing tests whose mock AI returns names absent from their mock store's categories (e.g. "MockCategory", "AI Test Category") either use a store with no categories (validation skipped) or must add that name to the store's `Categories`.

- [ ] **Step 5: Commit**

```bash
git add internal/categorizer
git commit -m "fix(categorizer): accept only AI answers naming a known category

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task B4: Staged tier and one staging write per run

**Files:**
- Modify: `internal/categorizer/store_interface.go` (`StagingStoreInterface`)
- Modify: `internal/store/staging.go`, `internal/store/staging_test.go`
- Create: `internal/categorizer/staged.go`
- Modify: `internal/categorizer/categorizer.go` (`SetStagingStore`, `saveStagingSuggestion`, new `FlushStaging`)
- Modify: `cmd/root/root.go` (`finalize`)
- Test: `internal/categorizer/categorizer_cost_test.go`, `internal/store/staging_test.go`

**Interfaces:**
- Produces:
  ```go
  type StagingStoreInterface interface {
  	LoadSuggestions() (creditors, debtors map[string]string, err error)
  	MergeSuggestions(creditors, debtors map[string]string) error
  }
  func NewStagedStrategy(creditors, debtors map[string]string, logger logging.Logger) *StagedStrategy // Name() "Staged", Source "staged"
  func (c *Categorizer) FlushStaging() error
  ```
- Removed: `AppendCreditorSuggestion`, `AppendDebtorSuggestion`.

- [ ] **Step 1: Write the failing store test** (`internal/store/staging_test.go`, add)

```go
func TestStagingStore_MergeKeepsExistingAndLoadReadsBoth(t *testing.T) {
	dir := t.TempDir()
	s := NewStagingStore(filepath.Join(dir, "sc.yaml"), filepath.Join(dir, "sd.yaml"))

	require.NoError(t, s.MergeSuggestions(map[string]string{"employer": "Salaire"}, map[string]string{"kiro": "Abonnements"}))
	require.NoError(t, s.MergeSuggestions(nil, map[string]string{"Ifolor": "Loisirs"}))

	creditors, debtors, err := s.LoadSuggestions()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"employer": "Salaire"}, creditors)
	assert.Equal(t, map[string]string{"kiro": "Abonnements", "ifolor": "Loisirs"}, debtors)
}

func TestStagingStore_LoadMissingOrCorruptIsEmpty(t *testing.T) {
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "sd.yaml")
	require.NoError(t, os.WriteFile(corrupt, []byte(":\n- not a map"), 0o600))
	s := NewStagingStore(filepath.Join(dir, "missing.yaml"), corrupt)

	creditors, debtors, err := s.LoadSuggestions()
	require.NoError(t, err)
	assert.Empty(t, creditors)
	assert.Empty(t, debtors)
}
```
Replace any existing `TestStagingStore_Append*` tests with these (the Append methods are removed).

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/store/ -run TestStagingStore -v`
Expected: FAIL to compile (`MergeSuggestions` undefined).

- [ ] **Step 3: Implement the store** — in `internal/store/staging.go` replace `AppendCreditorSuggestion`, `AppendDebtorSuggestion`, `appendSuggestion` with:

```go
// LoadSuggestions reads both staging files. A missing or corrupt file reads
// as empty: staging is a convenience, never a reason to stop a run.
func (s *StagingStore) LoadSuggestions() (map[string]string, map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(s.creditorsFile), s.read(s.debtorsFile), nil
}

// MergeSuggestions adds suggestions to the staging files, keeping what is
// already there, with one read and one write per file.
func (s *StagingStore) MergeSuggestions(creditors, debtors map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.merge(s.creditorsFile, creditors); err != nil {
		return err
	}
	return s.merge(s.debtorsFile, debtors)
}

func (s *StagingStore) read(filePath string) map[string]string {
	mappings := make(map[string]string)
	data, err := os.ReadFile(s.resolvePath(filePath)) // #nosec G304 -- path constructed internally
	if err != nil {
		return mappings
	}
	if yaml.Unmarshal(data, &mappings) != nil {
		return make(map[string]string)
	}
	return mappings
}

func (s *StagingStore) merge(filePath string, suggestions map[string]string) error {
	if len(suggestions) == 0 {
		return nil
	}
	mappings := s.read(filePath)
	for party, category := range suggestions {
		mappings[strings.ToLower(party)] = category
	}
	resolvedPath := s.resolvePath(filePath)
	if err := os.MkdirAll(filepath.Dir(resolvedPath), models.PermissionDirectory); err != nil {
		return fmt.Errorf("error creating staging directory: %w", err)
	}
	data, err := yaml.Marshal(mappings)
	if err != nil {
		return fmt.Errorf("error marshaling staging suggestions: %w", err)
	}
	if err := os.WriteFile(resolvedPath, data, models.PermissionNonSecretFile); err != nil {
		return fmt.Errorf("error writing staging file %s: %w", resolvedPath, err)
	}
	return nil
}
```
Update `StagingStoreInterface` in `internal/categorizer/store_interface.go` to the two methods above.

- [ ] **Step 4: Run store tests** — `go test ./internal/store/ -v` → PASS.

- [ ] **Step 5: Write the failing categorizer test** (`categorizer_cost_test.go`)

```go
// memStaging is an in-memory StagingStoreInterface that counts merges.
type memStaging struct {
	creditors, debtors map[string]string
	merges             int
}

func (m *memStaging) LoadSuggestions() (map[string]string, map[string]string, error) {
	return m.creditors, m.debtors, nil
}

func (m *memStaging) MergeSuggestions(cr, db map[string]string) error {
	m.merges++
	for k, v := range cr {
		m.creditors[strings.ToLower(k)] = v
	}
	for k, v := range db {
		m.debtors[strings.ToLower(k)] = v
	}
	return nil
}

func TestCategorize_StagedSuggestionsAnswerTheNextRun(t *testing.T) {
	staging := &memStaging{creditors: map[string]string{}, debtors: map[string]string{}}
	ctx := context.Background()

	// Run 1: the AI answers; the suggestion is staged once, at the end.
	ai1 := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	c1, _ := newCostCategorizer(t, ai1, false)
	c1.SetStagingStore(staging)
	_, _ = c1.Categorize(ctx, "Kiro", true, "-19", "2026-01-01", "")
	_, _ = c1.Categorize(ctx, "MIGROS Lausanne", true, "-5", "2026-01-01", "")
	assert.Zero(t, staging.merges, "nothing written during the run")
	require.NoError(t, c1.FlushStaging())
	assert.Equal(t, 1, staging.merges)
	assert.Equal(t, map[string]string{"kiro": "Abonnements"}, staging.debtors, "only AI answers are staged")

	// Run 2: a fresh categorizer finds Kiro in staging and never calls the AI.
	ai2 := &countingAI{answers: map[string]string{}}
	c2, _ := newCostCategorizer(t, ai2, false)
	c2.SetStagingStore(staging)
	got, err := c2.Categorize(ctx, "KIRO", true, "-19", "2026-02-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got.Name)
	assert.Equal(t, "staged", got.Source)
	assert.Zero(t, ai2.callCount())
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/categorizer/ -run TestCategorize_StagedSuggestions -v`
Expected: FAIL to compile (`FlushStaging` undefined).

- [ ] **Step 7: Implement the staged tier** — create `internal/categorizer/staged.go`:

```go
package categorizer

import (
	"context"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// StagedStrategy answers from AI suggestions staged by earlier runs. They are
// unreviewed, so they rank below the user's own mappings and keywords, but
// above the semantic and AI tiers that would otherwise pay to find the same
// answer again. It never writes: promotion to the mapping files stays manual.
type StagedStrategy struct {
	creditors map[string]string
	debtors   map[string]string
	logger    logging.Logger
}

// NewStagedStrategy builds the tier from lowercased party-name maps.
func NewStagedStrategy(creditors, debtors map[string]string, logger logging.Logger) *StagedStrategy {
	return &StagedStrategy{creditors: creditors, debtors: debtors, logger: logger}
}

// Name returns the strategy name for logs.
func (s *StagedStrategy) Name() string { return "Staged" }

// Categorize looks the party up in the staged suggestions for its direction.
func (s *StagedStrategy) Categorize(_ context.Context, tx Transaction) (models.Category, bool, error) {
	key := strings.ToLower(strings.TrimSpace(tx.PartyName))
	if key == "" {
		return models.Category{}, false, nil
	}
	mappings := s.creditors
	if tx.IsDebtor {
		mappings = s.debtors
	}
	name, ok := mappings[key]
	if !ok || name == "" {
		return models.Category{}, false, nil
	}
	return models.Category{
		Name:        name,
		Description: categoryDescriptionFromName(name),
		Confidence:  0.85,
		Source:      "staged",
	}, true, nil
}
```

In `categorizer.go` add fields `pendingStagedCreditors, pendingStagedDebtors map[string]string` and `stagingMu sync.Mutex`; initialize both maps in `NewCategorizer`. Replace `SetStagingStore` and `saveStagingSuggestion`, and add `FlushStaging`:

```go
// SetStagingStore wires the staging files: their suggestions become the
// staged tier (after direct and keyword), and new AI answers are buffered
// for FlushStaging.
func (c *Categorizer) SetStagingStore(staging StagingStoreInterface) {
	c.stagingStore = staging
	creditors, debtors, err := staging.LoadSuggestions()
	if err != nil {
		c.logger.WithError(err).Warn("Failed to load staged suggestions")
		return
	}
	staged := NewStagedStrategy(creditors, debtors, c.logger)
	for i, strategy := range c.strategies {
		if _, ok := strategy.(*KeywordStrategy); ok {
			c.strategies = append(c.strategies[:i+1], append([]CategorizationStrategy{staged}, c.strategies[i+1:]...)...)
			return
		}
	}
	c.strategies = append([]CategorizationStrategy{staged}, c.strategies...)
}

func (c *Categorizer) saveStagingSuggestion(partyName string, isDebtor bool, categoryName string) {
	if c.stagingStore == nil {
		return
	}
	c.stagingMu.Lock()
	defer c.stagingMu.Unlock()
	if isDebtor {
		c.pendingStagedDebtors[strings.ToLower(partyName)] = categoryName
	} else {
		c.pendingStagedCreditors[strings.ToLower(partyName)] = categoryName
	}
}

// FlushStaging writes the run's AI suggestions to the staging files in one
// merge. It is safe to call more than once.
func (c *Categorizer) FlushStaging() error {
	if c.stagingStore == nil {
		return nil
	}
	c.stagingMu.Lock()
	creditors, debtors := c.pendingStagedCreditors, c.pendingStagedDebtors
	c.pendingStagedCreditors, c.pendingStagedDebtors = map[string]string{}, map[string]string{}
	c.stagingMu.Unlock()
	if len(creditors) == 0 && len(debtors) == 0 {
		return nil
	}
	return c.stagingStore.MergeSuggestions(creditors, debtors)
}
```

In `cmd/root/root.go` `finalize`, after the debitor save:

```go
	if err := categorizerInstance.FlushStaging(); err != nil {
		Log.WithError(err).Warn("Failed to save staged AI suggestions")
	}
```

- [ ] **Step 8: Run tests**

Run: `go test ./internal/categorizer/ ./internal/store/ ./internal/container/ ./cmd/... 2>&1 | tail -20`
Expected: PASS. Fix any existing test that used the removed `Append*` methods by switching to `MergeSuggestions`/`memStaging`.

- [ ] **Step 9: Commit**

```bash
git add internal/categorizer internal/store cmd/root/root.go
git commit -m "perf(categorizer): reuse staged AI suggestions and write staging once per run

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task B5: CHANGELOG, regression, PR

- [ ] **Step 1: CHANGELOG** under `## [Unreleased]`:
  - `### Changed`:
    - `- Reuse staged AI suggestions as a categorization tier (after direct mappings and keywords), so a merchant the AI already classified is not sent to it again on later runs.`
    - `- Learn and stage only AI answers, and write mapping and staging files once per run instead of once per transaction.`
    - `- Cache uncategorized results for the run, so an unknown merchant reaches the AI once.`
  - `### Security`: `- Accept an AI category only if it names a category from categories.yaml.`
- [ ] **Step 2:** `make test && make lint`, Task R Step 3.
- [ ] **Step 3:** commit CHANGELOG, push, `gh pr create --title "perf(categorizer): cut AI calls and YAML writes; validate AI answers"`, wait for CI, merge.

---

# PR C — one categorization loop

Branch: `git checkout main && git pull && git checkout -b refactor/shared-categorization-loop`

### Task C1: Party-name function in the shared helper

**Files:**
- Modify: `internal/common/categorization_helper.go`
- Test: `internal/common/categorization_helper_test.go`

**Interfaces:**
- Produces:
  ```go
  func DefaultPartyName(tx models.Transaction) string
  func ProcessTransactionsWithPartyName(ctx context.Context, transactions []models.Transaction, logger logging.Logger, categorizer models.TransactionCategorizer, parserType string, partyName func(models.Transaction) string) ([]models.Transaction, error)
  // ProcessTransactionsWithCategorizationStats keeps its signature and calls ProcessTransactionsWithPartyName with DefaultPartyName.
  ```

- [ ] **Step 1: Write the failing test** (append to `categorization_helper_test.go`; it already defines a testify `MockCategorizer`)

```go
func TestDefaultPartyName_FallsBackToDescription(t *testing.T) {
	tx := models.Transaction{Description: "PMT CARTE RATP", CreditDebit: models.TransactionTypeDebit}
	assert.Equal(t, "PMT CARTE RATP", DefaultPartyName(tx))

	tx.Payee = "RATP"
	assert.Equal(t, "RATP", DefaultPartyName(tx), "a real party wins over the description")
}

func TestProcessTransactionsWithPartyName_UsesGivenFunction(t *testing.T) {
	m := new(MockCategorizer)
	m.On("Categorize", mock.Anything, "CLEANED", false, "10", "2026-04-15", "desc").
		Return(models.Category{Name: "Courses"}, nil)
	tx := models.Transaction{
		PartyName:   "RAW",
		Description: "desc",
		Amount:      decimal.NewFromInt(10),
		Date:        time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
	}

	out, err := ProcessTransactionsWithPartyName(context.Background(), []models.Transaction{tx}, nil, m, "Test",
		func(models.Transaction) string { return "CLEANED" })
	require.NoError(t, err)
	assert.Equal(t, "Courses", out[0].Category)
	m.AssertExpectations(t)
}
```
Add imports as needed (`decimal`, `time`, `mock`).

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/common/ -run 'TestDefaultPartyName|TestProcessTransactionsWithPartyName' -v`
Expected: FAIL to compile.

- [ ] **Step 3: Implement** in `categorization_helper.go`:

```go
// DefaultPartyName picks who a transaction is with: the payee or payer, then
// the other party fields, and finally the description, which is all some
// formats (Visa debit exports) carry.
func DefaultPartyName(tx models.Transaction) string {
	for _, name := range []string{tx.GetPartyName(), tx.PartyName, tx.Name, tx.Recipient, tx.Description} {
		if name != "" {
			return name
		}
	}
	return ""
}

// ProcessTransactionsWithCategorizationStats categorizes with DefaultPartyName.
func ProcessTransactionsWithCategorizationStats(
	ctx context.Context,
	transactions []models.Transaction,
	logger logging.Logger,
	categorizer models.TransactionCategorizer,
	parserType string,
) ([]models.Transaction, error) {
	return ProcessTransactionsWithPartyName(ctx, transactions, logger, categorizer, parserType, DefaultPartyName)
}
```
Rename the existing function body to `ProcessTransactionsWithPartyName(..., partyName func(models.Transaction) string)` (keep the existing doc comment on it), replace the "Attempt categorization" block that computes `partyName` with `party := partyName(tx)` (rename uses accordingly), and replace every `"Uncategorized"` literal with `models.CategoryUncategorized`.

- [ ] **Step 4: Run tests** — `go test ./internal/common/ ./internal/visecaparser/ ./internal/selmaparser/ ./internal/pdfparser/ -v 2>&1 | tail -15` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/common
git commit -m "refactor(common): let the categorization helper take a party-name function

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task C2: Debit, Revolut, Revolut-crypto, Revolut-investment use the helper

**Files:**
- Modify: `internal/debitparser/debitparser.go` (`ParseWithCategorizer` ≈63-120)
- Modify: `internal/revolutparser/revolutparser.go` (`ParseWithCategorizer` ≈78-161)
- Modify: `internal/revolutcryptoparser/revolutcryptoparser.go` (≈148-209)
- Modify: `internal/revolutinvestmentparser/revolutinvestmentparser.go` (≈90-142)
- Modify: `internal/revolutinvestmentparser/revolutinvestmentparser_test.go` (≈159, ≈177)

**Interfaces:**
- Consumes: `common.ProcessTransactionsWithCategorizationStats` (C1).

- [ ] **Step 1: Update the two investment expectations first (they are the failing tests for the accepted behavior change)**

In `revolutinvestmentparser_test.go` lines ≈159 and ≈177, change the date argument `"30.05.2025"` to `"2025-05-30"` and the info argument `""` to `mock.Anything`.

Run: `go test ./internal/revolutinvestmentparser/ -run TestParseWithCategorizer -v`
Expected: FAIL (the parser still sends `30.05.2025`).

- [ ] **Step 2: Replace each inline loop**

In each of the four parsers, delete the `if categorizer != nil { ... } else { ... }` block inside the row loop (keep conversion, skipping and `ctx.Err()` checks), and after the loop, before the final log/return, add:

```go
	transactions, err := common.ProcessTransactionsWithCategorizationStats(ctx, transactions, logger, categorizer, "<Name>")
	if err != nil {
		return nil, err
	}
```
with `<Name>` = `"Debit"`, `"Revolut"`, `"RevolutCrypto"`, `"RevolutInvestment"`. In `revolutparser.go`, call it on `transactions` before `postProcessTransactions(transactions)` (same order as today). Import `fjacquet/camt-csv/internal/common` where missing. If `transactions` and `err` are both already declared in that scope, use `=` instead of `:=`; the compiler error "no new variables on left side of :=" tells you which.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/debitparser/ ./internal/revolutparser/ ./internal/revolutcryptoparser/ ./internal/revolutinvestmentparser/ -v 2>&1 | grep -E '^(---|ok|FAIL)'`
Expected: PASS. `debitparser` `TestParseWithCategorizer` still gets "Transport" because `DefaultPartyName` falls back to the description.

- [ ] **Step 4: Commit**

```bash
git add internal/debitparser internal/revolutparser internal/revolutcryptoparser internal/revolutinvestmentparser
git commit -m "refactor(parsers): debit and Revolut parsers use the shared categorization loop

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task C3: CAMT uses the helper with its own party function

**Files:**
- Modify: `internal/camtparser/adapter.go` (`Parse`, delete `categorizeTransaction`)
- Test: `internal/camtparser/camtparser_test.go`

- [ ] **Step 1: Write the failing test**

```go
type recordingCategorizer struct {
	parties, dates, infos []string
}

func (r *recordingCategorizer) Categorize(_ context.Context, party string, _ bool, _, date, info string) (models.Category, error) {
	r.parties = append(r.parties, party)
	r.dates = append(r.dates, date)
	r.infos = append(r.infos, info)
	return models.Category{Name: "Courses"}, nil
}

func TestParse_CategorizesWithCleanedPartyAndISODate(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02"><BkToCstmrStmt><Stmt>
<Acct><Id><IBAN>CH1700767000K54293249</IBAN></Id></Acct>
<Ntry><Amt Ccy="CHF">12.50</Amt><CdtDbtInd>DBIT</CdtDbtInd><Sts>BOOK</Sts><BookgDt><Dt>2026-04-15</Dt></BookgDt><ValDt><Dt>2026-04-15</Dt></ValDt>
<AddtlNtryInf>PMT CARTE Migros Lausanne</AddtlNtryInf></Ntry>
</Stmt></BkToCstmrStmt></Document>`

	rec := &recordingCategorizer{}
	adapter := NewAdapter(logging.NewMockLogger())
	adapter.SetCategorizer(rec)
	txs, err := adapter.Parse(context.Background(), strings.NewReader(doc))
	require.NoError(t, err)
	require.Len(t, txs, 1)
	require.Len(t, rec.parties, 1)
	assert.NotContains(t, rec.parties[0], "PMT CARTE", "payment-method prefix is stripped before categorizing")
	assert.Equal(t, "2026-04-15", rec.dates[0])
	assert.Equal(t, "Courses", txs[0].Category)
}
```

- [ ] **Step 2: Run** — `go test ./internal/camtparser/ -run TestParse_CategorizesWithCleaned -v` → FAIL on the date (`15.04.2026`).

- [ ] **Step 3: Implement** — in `Parse`, append transactions without categorizing (`transactions = append(transactions, transaction)`), and after both loops:

```go
	return common.ProcessTransactionsWithPartyName(ctx, transactions, a.GetLogger(), a.GetCategorizer(), "CAMT", camtPartyName)
}

// camtPartyName is who a CAMT entry is with: the resolved party, else the
// entry text, without the card/transfer prefix the bank puts in front.
func camtPartyName(tx models.Transaction) string {
	party := tx.PartyName
	if party == "" {
		party = tx.Description
	}
	if party == "" {
		party = tx.RemittanceInfo
	}
	return cleanPaymentMethodPrefixes(party)
}
```
Delete `categorizeTransaction`. Import `fjacquet/camt-csv/internal/common`; drop `dateutils` if unused.

- [ ] **Step 4: Run** — `go test ./internal/camtparser/ ./internal/batch/ ./cmd/... -v 2>&1 | grep -E '^(---|ok|FAIL)'` → PASS.

- [ ] **Step 5: Commit, CHANGELOG, regression, PR**

CHANGELOG `### Changed`: `- All parsers share one categorization loop: the categorizer now gets ISO dates and the transaction description for debit, Revolut, Revolut crypto, Revolut investment and CAMT statements, and a transaction with no party is categorized from its description.`

```bash
git add internal/camtparser CHANGELOG.md
git commit -m "refactor(camt): use the shared categorization loop

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
make test && make lint
```
Task R Step 3 (categories may differ; counts and sums must not). Push, `gh pr create --title "refactor: one categorization loop for all parsers"`, CI, merge.

---

# PR D — batched AI requests

Branch: `git checkout main && git pull && git checkout -b perf/batch-ai-requests`

### Task D1: Batch prompt and JSON parsing in the AI clients

**Files:**
- Modify: `internal/categorizer/ai_common.go` (split prompt; add `categorizeBatch`, `parseBatchAnswer`)
- Modify: `internal/categorizer/ai_client.go` (add `BatchAIClient`)
- Modify: `internal/categorizer/gemini_client.go`, `internal/categorizer/openrouter_client.go`
- Test: `internal/categorizer/ai_batch_test.go` (create)

**Interfaces:**
- Produces:
  ```go
  type BatchAIClient interface {
  	CategorizeBatch(ctx context.Context, transactions []models.Transaction) (map[string]string, error)
  }
  // keys: lowercased, trimmed party names; values: cleanCategory'd answers
  func parseBatchAnswer(raw string) (map[string]string, error)
  ```

- [ ] **Step 1: Write the failing tests** (`internal/categorizer/ai_batch_test.go`)

```go
package categorizer

import (
	"context"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBatchAnswer(t *testing.T) {
	raw := "```json\n{\"Kiro\": \"Abonnements\", \" Migros \": \"**Courses**\"}\n```"
	got, err := parseBatchAnswer(raw)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"kiro": "Abonnements", "migros": "Courses"}, got)

	_, err = parseBatchAnswer("I cannot help with that")
	assert.Error(t, err)
}

func TestBaseClientCategorizeBatch_OneRequestForManyParties(t *testing.T) {
	b := newBaseAIClient("test", testLogger(), "key", 60)
	var prompts []string
	complete := func(_ context.Context, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		return `{"Kiro": "Abonnements", "Migros": "Courses"}`, nil
	}
	txs := []models.Transaction{{PartyName: "Kiro"}, {PartyName: "Migros"}}

	got, err := b.categorizeBatch(context.Background(), txs, complete)
	require.NoError(t, err)
	assert.Len(t, prompts, 1)
	assert.Contains(t, prompts[0], "Kiro")
	assert.Contains(t, prompts[0], "Migros")
	assert.Equal(t, "Abonnements", got["kiro"])
	assert.True(t, strings.Contains(prompts[0], "JSON"))
}
```

- [ ] **Step 2: Run** — `go test ./internal/categorizer/ -run 'TestParseBatchAnswer|TestBaseClientCategorizeBatch' -v` → FAIL to compile.

- [ ] **Step 3: Implement**

In `ai_common.go`, split `buildCategorizationPrompt`: move everything in its format string up to (not including) the line `TRANSACTION TO CATEGORIZE:` into `const categorizationPreamble = \`...\`` (verbatim text, no format verbs — if the preamble contains `%`, double it is NOT needed in a const; just keep it as is and remove from the Sprintf). Rewrite:

```go
func buildCategorizationPrompt(transaction models.Transaction) string {
	return categorizationPreamble + fmt.Sprintf(`TRANSACTION TO CATEGORIZE:

Party: %s
Description: %s
Amount: %s CHF

Category:`, transaction.PartyName, transaction.Description, transaction.Amount.String())
}

func buildBatchCategorizationPrompt(transactions []models.Transaction) string {
	var b strings.Builder
	b.WriteString(categorizationPreamble)
	b.WriteString("TRANSACTIONS TO CATEGORIZE (one per line: party | description | amount CHF):\n\n")
	for _, tx := range transactions {
		fmt.Fprintf(&b, "%s | %s | %s\n", tx.PartyName, tx.Description, tx.Amount.String())
	}
	b.WriteString("\nAnswer with ONLY a JSON object mapping each party name, exactly as written above, to one category from the list. No other text.\n")
	return b.String()
}

// parseBatchAnswer reads the JSON object a batch prompt asks for, tolerating
// a fenced code block or text around it. Keys are normalized like the cache.
func parseBatchAnswer(raw string) (map[string]string, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object in batch answer")
	}
	var answers map[string]string
	if err := json.Unmarshal([]byte(raw[start:end+1]), &answers); err != nil {
		return nil, fmt.Errorf("invalid JSON in batch answer: %w", err)
	}
	out := make(map[string]string, len(answers))
	for party, category := range answers {
		out[strings.ToLower(strings.TrimSpace(party))] = cleanCategory(category)
	}
	return out, nil
}

// categorizeBatch asks for many parties in one request: one rate-limiter
// token, one prompt. A party missing from the answer is simply absent from
// the map; the caller decides what to do with it.
func (b *baseAIClient) categorizeBatch(ctx context.Context, transactions []models.Transaction, complete completeFn) (map[string]string, error) {
	if b.apiKey == "" || len(transactions) == 0 {
		return map[string]string{}, nil
	}
	if err := b.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait cancelled: %w", err)
	}
	raw, err := b.completeWithRetry(ctx, buildBatchCategorizationPrompt(transactions), complete)
	if err != nil {
		return nil, err
	}
	return parseBatchAnswer(raw)
}
```
Add `"encoding/json"` to imports. In `ai_client.go` add the `BatchAIClient` interface (doc: optional; implemented by the Gemini and OpenRouter clients). In each client:

```go
// CategorizeBatch categorizes many transactions in one request.
func (c *GeminiClient) CategorizeBatch(ctx context.Context, transactions []models.Transaction) (map[string]string, error) {
	return c.categorizeBatch(ctx, transactions, c.complete)
}
```
(same for `*OpenRouterClient`).

- [ ] **Step 4: Run** — `go test ./internal/categorizer/ -v 2>&1 | grep -E '^(---|ok|FAIL)'` → PASS (existing prompt tests still pass because the single prompt text is unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/categorizer
git commit -m "feat(categorizer): batch prompt and JSON answer parsing in AI clients

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task D2: Categorizer.CategorizeBatch

**Files:**
- Modify: `internal/models/categorizer.go` (add `CategorizeRequest`, `BatchCategorizer`)
- Create: `internal/categorizer/batch.go`
- Test: `internal/categorizer/ai_batch_test.go`

**Interfaces:**
- Consumes: `cacheKey`, `recordLearning`, `canonicalAICategory` (B1-B3), `BatchAIClient` (D1).
- Produces:
  ```go
  // internal/models
  type CategorizeRequest struct {
  	PartyName string
  	IsDebtor  bool
  	Amount    string
  	Date      string
  	Info      string
  }
  type BatchCategorizer interface {
  	CategorizeBatch(ctx context.Context, requests []CategorizeRequest) ([]Category, error)
  }
  // internal/categorizer
  const aiBatchSize = 25
  func (c *Categorizer) CategorizeBatch(ctx context.Context, requests []models.CategorizeRequest) ([]models.Category, error)
  ```

- [ ] **Step 1: Write the failing tests** (append to `ai_batch_test.go`)

```go
// batchingAI implements AIClient and BatchAIClient; it can drop parties or
// return garbage to exercise the fallbacks.
type batchingAI struct {
	countingAI
	batchCalls int
	drop       map[string]bool
	garbage    bool
}

func (b *batchingAI) CategorizeBatch(_ context.Context, txs []models.Transaction) (map[string]string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.batchCalls++
	if b.garbage {
		return nil, fmt.Errorf("invalid JSON in batch answer")
	}
	out := map[string]string{}
	for _, tx := range txs {
		key := strings.ToLower(strings.TrimSpace(tx.PartyName))
		if !b.drop[key] {
			out[key] = b.answers[key]
		}
	}
	return out, nil
}

func batchRequests(parties ...string) []models.CategorizeRequest {
	reqs := make([]models.CategorizeRequest, 0, len(parties))
	for _, p := range parties {
		reqs = append(reqs, models.CategorizeRequest{PartyName: p, IsDebtor: true, Amount: "-1", Date: "2026-01-01"})
	}
	return reqs
}

func newBatchCategorizer(t *testing.T, ai *batchingAI) *Categorizer {
	t.Helper()
	st := &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Courses", Keywords: []string{"MIGROS"}}, {Name: "Abonnements"}},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}
	c := NewCategorizer(ai, nil, st, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)
	return c
}

func TestCategorizeBatch_SixtyPartiesThreeCalls(t *testing.T) {
	ai := &batchingAI{countingAI: countingAI{answers: map[string]string{}}}
	parties := make([]string, 60)
	for i := range parties {
		parties[i] = fmt.Sprintf("Shop %02d", i)
		ai.answers[strings.ToLower(parties[i])] = "Abonnements"
	}
	c := newBatchCategorizer(t, ai)

	got, err := c.CategorizeBatch(context.Background(), batchRequests(parties...))
	require.NoError(t, err)
	require.Len(t, got, 60)
	assert.Equal(t, 3, ai.batchCalls)
	assert.Zero(t, ai.callCount(), "no single calls when every party is answered")
	assert.Equal(t, "Abonnements", got[59].Name)
}

func TestCategorizeBatch_LocalTiersAndCacheBeforeAI(t *testing.T) {
	ai := &batchingAI{countingAI: countingAI{answers: map[string]string{"kiro": "Abonnements"}}}
	c := newBatchCategorizer(t, ai)

	got, err := c.CategorizeBatch(context.Background(), batchRequests("MIGROS Lausanne", "Kiro", "kiro ", "   "))
	require.NoError(t, err)
	assert.Equal(t, "Courses", got[0].Name, "keyword tier")
	assert.Equal(t, "Abonnements", got[1].Name)
	assert.Equal(t, "Abonnements", got[2].Name, "same party, one AI question")
	assert.Equal(t, models.CategoryUncategorized, got[3].Name, "blank party never reaches the AI")
	assert.Equal(t, 1, ai.batchCalls)
}

func TestCategorizeBatch_SamePartyBothDirectionsAreSeparate(t *testing.T) {
	ai := &batchingAI{countingAI: countingAI{answers: map[string]string{"twint": "Abonnements"}}}
	c := newBatchCategorizer(t, ai)
	reqs := []models.CategorizeRequest{
		{PartyName: "Twint", IsDebtor: true},
		{PartyName: "Twint", IsDebtor: false},
	}

	got, err := c.CategorizeBatch(context.Background(), reqs)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, 2, ai.batchCalls, "debtor and creditor are asked separately")
}

func TestCategorizeBatch_MissingPartyFallsBackToSingleCall(t *testing.T) {
	ai := &batchingAI{
		countingAI: countingAI{answers: map[string]string{"kiro": "Abonnements", "ifolor": "Abonnements"}},
		drop:       map[string]bool{"ifolor": true},
	}
	c := newBatchCategorizer(t, ai)

	got, err := c.CategorizeBatch(context.Background(), batchRequests("Kiro", "Ifolor"))
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got[1].Name)
	assert.Equal(t, 1, ai.callCount(), "only the missing party gets a single call")
}

func TestCategorizeBatch_GarbageFallsBackForWholeChunk(t *testing.T) {
	ai := &batchingAI{countingAI: countingAI{answers: map[string]string{"kiro": "Abonnements", "ifolor": "Abonnements"}}, garbage: true}
	c := newBatchCategorizer(t, ai)

	got, err := c.CategorizeBatch(context.Background(), batchRequests("Kiro", "Ifolor"))
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got[0].Name)
	assert.Equal(t, 2, ai.callCount())
}
```
Imports to add: `"fmt"`, `"fjacquet/camt-csv/internal/store"`.

Why two calls in `TestCategorizeBatch_SamePartyBothDirectionsAreSeparate`: a batch answer is keyed by party name only, so one chunk never mixes debtors and creditors; the implementation below sends one chunk per direction.

- [ ] **Step 2: Run** — `go test ./internal/categorizer/ -run TestCategorizeBatch -v` → FAIL to compile.

- [ ] **Step 3: Implement** — add the two types to `internal/models/categorizer.go` (with doc comments). Create `internal/categorizer/batch.go`:

```go
package categorizer

import (
	"context"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// aiBatchSize is how many parties one AI request carries.
const aiBatchSize = 25

// CategorizeBatch categorizes many transactions, asking the AI once per chunk
// of parties the local tiers could not place instead of once per party.
// Results are in request order and go through the same cache, validation,
// learning and staging as single requests.
func (c *Categorizer) CategorizeBatch(ctx context.Context, requests []models.CategorizeRequest) ([]models.Category, error) {
	results := make([]models.Category, len(requests))
	pending := map[string][]int{} // cache key -> request indexes
	var order []string            // first-seen order of pending keys

	for i, req := range requests {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tx := Transaction{PartyName: req.PartyName, IsDebtor: req.IsDebtor, Amount: req.Amount, Date: req.Date, Info: req.Info}
		if category, ok := c.categorizeWithoutAI(ctx, tx); ok {
			results[i] = category
			continue
		}
		key := cacheKey(req.PartyName, req.IsDebtor)
		if _, seen := pending[key]; !seen {
			order = append(order, key)
		}
		pending[key] = append(pending[key], i)
	}

	batcher, canBatch := c.aiClient.(BatchAIClient)
	for _, isDebtor := range []bool{true, false} {
		var keys []string
		for _, key := range order {
			if requests[pending[key][0]].IsDebtor == isDebtor {
				keys = append(keys, key)
			}
		}
		for start := 0; start < len(keys); start += aiBatchSize {
			chunk := keys[start:min(start+aiBatchSize, len(keys))]
			if err := c.answerChunk(ctx, batcher, canBatch, requests, pending, chunk); err != nil {
				return nil, err
			}
		}
	}

	for key, idxs := range pending {
		c.batchCacheMu.RLock()
		category := c.batchCache[key]
		c.batchCacheMu.RUnlock()
		for _, i := range idxs {
			results[i] = category
		}
	}
	return results, nil
}

// categorizeWithoutAI answers from the cache or the local tiers, reporting
// false when only the AI is left. A blank party is answered as Uncategorized.
func (c *Categorizer) categorizeWithoutAI(ctx context.Context, tx Transaction) (models.Category, bool) {
	if strings.TrimSpace(tx.PartyName) == "" {
		return models.Category{Name: models.CategoryUncategorized, Description: "No party name provided"}, true
	}
	key := cacheKey(tx.PartyName, tx.IsDebtor)
	c.batchCacheMu.RLock()
	cached, ok := c.batchCache[key]
	c.batchCacheMu.RUnlock()
	if ok {
		return cached, true
	}
	for _, strategy := range c.strategies {
		if _, isAI := strategy.(*AIStrategy); isAI {
			continue
		}
		category, found, err := strategy.Categorize(ctx, tx)
		if err != nil || !found {
			continue
		}
		c.storeInCache(key, category)
		return category, true
	}
	return models.Category{}, false
}

// answerChunk asks the AI about one chunk of same-direction parties, then
// falls back to single requests for any party the answer left out.
func (c *Categorizer) answerChunk(ctx context.Context, batcher BatchAIClient, canBatch bool, requests []models.CategorizeRequest, pending map[string][]int, chunk []string) error {
	answers := map[string]string{}
	if canBatch {
		txs := make([]models.Transaction, 0, len(chunk))
		for _, key := range chunk {
			req := requests[pending[key][0]]
			txs = append(txs, models.Transaction{PartyName: req.PartyName, Description: req.Info, Amount: models.ParseAmount(req.Amount)})
		}
		got, err := batcher.CategorizeBatch(ctx, txs)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			c.logger.WithError(err).Warn("Batch AI categorization failed, falling back to single requests",
				logging.Field{Key: "parties", Value: len(chunk)})
		} else {
			answers = got
		}
	}

	for _, key := range chunk {
		req := requests[pending[key][0]]
		var category models.Category
		if answer, ok := answers[strings.ToLower(strings.TrimSpace(req.PartyName))]; ok {
			category = c.aiResult(req.PartyName, answer)
		} else {
			tx := Transaction{PartyName: req.PartyName, IsDebtor: req.IsDebtor, Amount: req.Amount, Date: req.Date, Info: req.Info}
			category = c.singleAI(ctx, tx)
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
		}
		c.storeInCache(key, category)
		c.recordLearning(req.PartyName, req.IsDebtor, category)
	}
	return nil
}

// aiResult turns a raw AI answer into a validated category.
func (c *Categorizer) aiResult(partyName, answer string) models.Category {
	if answer == "" || answer == models.CategoryUncategorized {
		return models.Category{Name: models.CategoryUncategorized, Source: "ai"}
	}
	canonical, ok := c.canonicalAICategory(answer)
	if !ok {
		c.logger.WithFields(
			logging.Field{Key: "party", Value: partyName},
			logging.Field{Key: "answer", Value: answer},
		).Warn("Rejected AI category not in categories.yaml")
		return models.Category{Name: models.CategoryUncategorized, Source: "ai"}
	}
	return models.Category{Name: canonical, Description: categoryDescriptionFromName(canonical), Confidence: 0.8, Source: "ai"}
}

// singleAI asks the AI tier alone about one party.
func (c *Categorizer) singleAI(ctx context.Context, tx Transaction) models.Category {
	for _, strategy := range c.strategies {
		ai, ok := strategy.(*AIStrategy)
		if !ok {
			continue
		}
		category, found, err := ai.Categorize(ctx, tx)
		if err != nil || !found {
			return models.Category{Name: models.CategoryUncategorized, Source: "ai"}
		}
		return c.aiResult(tx.PartyName, category.Name)
	}
	return models.Category{Name: models.CategoryUncategorized}
}

func (c *Categorizer) storeInCache(key string, category models.Category) {
	c.batchCacheMu.Lock()
	c.batchCache[key] = category
	c.batchCacheMu.Unlock()
}
```
Add `"strings"` to the imports. Also, in `categorizer.go` `categorizeTransaction`, replace the two inline cache-store blocks with `c.storeInCache(key, ...)`, and replace the B3 validation block with `category = c.aiResult(transaction.PartyName, category.Name)` when `category.Source == "ai"` (keeps one validation path).

- [ ] **Step 4: Run** — `go test ./internal/categorizer/ -race -v 2>&1 | grep -E '^(---|ok|FAIL)'` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/models/categorizer.go internal/categorizer
git commit -m "feat(categorizer): categorize in batches, one AI request per 25 parties

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task D3: The shared loop uses batches when available

**Files:**
- Modify: `internal/common/categorization_helper.go` (`ProcessTransactionsWithPartyName`)
- Test: `internal/common/categorization_helper_test.go`

- [ ] **Step 1: Write the failing test**

```go
type fakeBatchCategorizer struct {
	batchCalls, singleCalls int
}

func (f *fakeBatchCategorizer) Categorize(context.Context, string, bool, string, string, string) (models.Category, error) {
	f.singleCalls++
	return models.Category{Name: "Courses"}, nil
}

func (f *fakeBatchCategorizer) CategorizeBatch(_ context.Context, reqs []models.CategorizeRequest) ([]models.Category, error) {
	f.batchCalls++
	out := make([]models.Category, len(reqs))
	for i := range reqs {
		out[i] = models.Category{Name: "Courses"}
	}
	return out, nil
}

func TestProcessTransactions_UsesBatchCategorizer(t *testing.T) {
	f := &fakeBatchCategorizer{}
	txs := []models.Transaction{
		{PartyName: "A", Amount: decimal.NewFromInt(-1)},
		{PartyName: "B", Amount: decimal.NewFromInt(-2)},
		{PartyName: "C", Category: "Investissements"}, // parser-set, untouched
		{},                                         // no party
	}
	out, err := ProcessTransactionsWithCategorizationStats(context.Background(), txs, nil, f, "Test")
	require.NoError(t, err)
	assert.Equal(t, 1, f.batchCalls)
	assert.Zero(t, f.singleCalls)
	assert.Equal(t, []string{"Courses", "Courses", "Investissements", models.CategoryUncategorized},
		[]string{out[0].Category, out[1].Category, out[2].Category, out[3].Category})
}
```

- [ ] **Step 2: Run** — `go test ./internal/common/ -run TestProcessTransactions_UsesBatchCategorizer -v` → FAIL (`batchCalls` 0).

- [ ] **Step 3: Implement** — in `ProcessTransactionsWithPartyName`, keep the first loop's skip rules (nil categorizer, parser-set category, no party → Uncategorized) but, when `categorizer` implements `models.BatchCategorizer`, collect a `models.CategorizeRequest` and the transaction index instead of calling `Categorize`; after the loop:

```go
	if len(batchIdx) > 0 {
		categories, err := batcher.CategorizeBatch(ctx, batchReqs)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			logger.WithError(err).Warn("Batch categorization failed",
				logging.Field{Key: "parser_type", Value: parserType})
			for _, i := range batchIdx {
				processedTransactions[i].Category = models.CategoryUncategorized
				stats.IncrementFailed()
			}
		} else {
			for n, i := range batchIdx {
				name := categories[n].Name
				if name == "" || name == models.CategoryUncategorized {
					processedTransactions[i].Category = models.CategoryUncategorized
					stats.IncrementUncategorized()
				} else {
					processedTransactions[i].Category = name
					stats.IncrementSuccessful()
				}
			}
		}
	}
```
with, before the loop:
```go
	batcher, canBatch := categorizer.(models.BatchCategorizer)
	var batchReqs []models.CategorizeRequest
	var batchIdx []int
```
and, in the loop where `Categorize` was called, when `canBatch`:
```go
		if canBatch {
			batchReqs = append(batchReqs, models.CategorizeRequest{
				PartyName: party, IsDebtor: tx.IsDebit(), Amount: tx.Amount.String(),
				Date: tx.Date.Format("2006-01-02"), Info: tx.Description,
			})
			batchIdx = append(batchIdx, i)
			continue
		}
```
(`categorizer.(models.BatchCategorizer)` on a nil interface returns `ok=false`, so the nil-categorizer path is unchanged.)

- [ ] **Step 4: Run** — `go test ./... 2>&1 | grep -E 'FAIL|^ok' | grep FAIL` → no output.

- [ ] **Step 5: CHANGELOG, regression, PR**

CHANGELOG `### Changed`: `- Ask the AI about up to 25 merchants per request instead of one; a merchant missing from the answer is asked again on its own.`

```bash
git add internal/common CHANGELOG.md
git commit -m "perf: parsers categorize through batched AI requests

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
make test && make lint
```
Task R Step 3. Push, `gh pr create --title "perf: batch AI categorization requests"`, CI, merge.

---

# PR E — formula injection

Branch: `git checkout main && git pull && git checkout -b fix/csv-formula-injection`

### Task E1: `internal/csvsafe` package

**Files:**
- Create: `internal/csvsafe/csvsafe.go`, `internal/csvsafe/csvsafe_test.go`
- Modify: `internal/icompta/report.go` (use `csvsafe`, escape every text column), `internal/icompta/report_test.go`

**Interfaces:**
- Produces: `func Escape(s string) string`, `func Unescape(s string) string`, `func NeedsEscape(s string) bool` in package `csvsafe`.

- [ ] **Step 1: Write the failing test** (`internal/csvsafe/csvsafe_test.go`)

```go
package csvsafe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeRoundTrip(t *testing.T) {
	for _, s := range []string{"", "Migros", `=HYPERLINK("x")`, "+1", "-2", "@SUM(A1)", "\tx", "'=already", "'plain"} {
		assert.Equal(t, s, Unescape(Escape(s)), s)
	}
	assert.Equal(t, `'=HYPERLINK("x")`, Escape(`=HYPERLINK("x")`))
	assert.Equal(t, "''=already", Escape("'=already"), "an escaped-looking value is escaped again")
	assert.Equal(t, "Migros", Escape("Migros"))
}
```

- [ ] **Step 2: Run** — `go test ./internal/csvsafe/ -v` → FAIL (package missing).

- [ ] **Step 3: Implement** — `internal/csvsafe/csvsafe.go`: package doc `// Package csvsafe neutralises CSV cells a spreadsheet would evaluate as formulas.` then move `formulaTriggers`, `needsEscape`, `escapeCell`, `unescapeCell` from `internal/icompta/report.go` verbatim, renamed `NeedsEscape`, `Escape`, `Unescape` (exported doc comments). In `report.go` delete them, import `fjacquet/camt-csv/internal/csvsafe`, and write every text column escaped:

```go
		rec := []string{
			csvsafe.Escape(row.SplitID), csvsafe.Escape(row.Date), csvsafe.Escape(row.Name), row.Amount,
			csvsafe.Escape(row.OldCategory), csvsafe.Escape(row.NewCategory),
			row.Tier, string(row.Decision), csvsafe.Escape(row.Reason), apply,
		}
```
and in `ReadReport` read `SplitID: csvsafe.Unescape(rec[0]), Date: csvsafe.Unescape(rec[1])` plus the existing `Unescape` calls. Add a case to `report_test.go`'s round-trip test with `Date: "=1+1"` and `SplitID: "@x"`.

- [ ] **Step 4: Run** — `go test ./internal/csvsafe/ ./internal/icompta/ -v 2>&1 | grep -E '^(---|ok|FAIL)'` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/csvsafe internal/icompta
git commit -m "fix(icompta): escape every text column of the recategorize report

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

### Task E2: Escape free text in standard and jumpsoft output

**Files:**
- Modify: `internal/formatter/standard.go`, `internal/formatter/jumpsoft.go`
- Test: `internal/formatter/formatter_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestFormattersEscapeFormulasExceptICompta(t *testing.T) {
	tx := createTestTransaction()
	tx.Description = `=HYPERLINK("http://evil","x")`
	tx.Name = "@SUM(A1)"
	tx.Amount = decimal.NewFromFloat(-180)

	std, err := NewStandardFormatter().Format([]models.Transaction{tx})
	require.NoError(t, err)
	assert.Equal(t, `'=HYPERLINK("http://evil","x")`, std[0][6])
	assert.Equal(t, "'@SUM(A1)", std[0][3])
	assert.Equal(t, "-180.00", std[0][8], "amounts are never escaped")

	js, err := NewJumpsoftFormatter().Format([]models.Transaction{tx})
	require.NoError(t, err)
	assert.Equal(t, `'=HYPERLINK("http://evil","x")`, js[0][1])
	assert.Equal(t, "-180.00", js[0][2])

	ic, err := NewIComptaFormatter().Format([]models.Transaction{tx})
	require.NoError(t, err)
	assert.Equal(t, `=HYPERLINK("http://evil","x")`, ic[0][3], "iCompta output stays raw")
}
```
(If the iCompta constructor has another name, use the one in `internal/formatter/icompta.go`; Description is column 3 there.)

- [ ] **Step 2: Run** — `go test ./internal/formatter/ -run TestFormattersEscapeFormulas -v` → FAIL.

- [ ] **Step 3: Implement**

`standard.go`:
```go
// standardTextColumns are the free-text columns of the standard layout, which
// carry bank or merchant text a spreadsheet could evaluate as a formula.
var standardTextColumns = []int{3, 4, 6, 7, 16, 17, 18} // Name, PartyName, Description, RemittanceInfo, Category, Type, Fund
```
and in `Format`, after `row, err := tx.MarshalCSV()`:
```go
		for _, col := range standardTextColumns {
			row[col] = csvsafe.Escape(row[col])
		}
```
`jumpsoft.go`: wrap `description`, `category`, `txType`, `notes` with `csvsafe.Escape(...)` in the appended row. Import `fjacquet/camt-csv/internal/csvsafe` in both (it imports nothing internal, so no cycle).

- [ ] **Step 4: Run** — `go test ./internal/formatter/ -v 2>&1 | grep -E '^(---|ok|FAIL)'` → PASS (including `TestIComptaHeaderCoversPluginMappings`).

- [ ] **Step 5: CHANGELOG, regression, PR**

CHANGELOG `### Security`: `- Escape text cells that a spreadsheet would evaluate as formulas in the standard and jumpsoft CSV formats and in every text column of the recategorize report. iCompta output is unchanged.`

```bash
git add internal/formatter CHANGELOG.md
git commit -m "fix(formatter): escape formula-like text in standard and jumpsoft output

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
make test && make lint
```
Task R Step 3 (icompta format, must be identical). Push, `gh pr create --title "fix: CSV formula injection in standard and jumpsoft output"`, CI, merge.

---

# PR F — housekeeping

Branch: `git checkout main && git pull && git checkout -b chore/housekeeping`

### Task F1: Remove the no-op gocsv init

**Files:** Modify `internal/common/csv.go` (delete `init()` ≈22-25; keep `const Delimiter`).

- [ ] **Step 1:** `grep -rn 'csv:"[^"]*[,;]' --include='*.go' .` → expect no output (no multi-name tags).
- [ ] **Step 2:** delete the `init()`; drop the `gocsv` import from `csv.go` if now unused.
- [ ] **Step 3:** `go test ./internal/common/ ./internal/debitparser/ ./internal/revolutparser/` → PASS.
- [ ] **Step 4:** commit `chore(common): remove no-op gocsv TagSeparator init`.

### Task F2: PDF extractor cleanup and one pdftotext run per file

**Files:**
- Modify: `internal/pdfparser/pdfparser_helpers.go` (delete `var extractTextFromPDF`; `getDefaultLogger()` at ≈143 → the in-scope `logger`)
- Modify: `internal/pdfparser/extractor.go` (`RealPDFExtractor.ExtractText` calls `extractTextFromPDFImpl` directly; add `cachingExtractor`)
- Modify: `internal/pdfparser/adapter.go` (`NewAdapter` wraps the extractor)
- Test: `internal/pdfparser/pdfparser_test.go` (reuse `callCountingExtractor` ≈315-325)

**Interfaces:**
- Produces: `func newCachingExtractor(inner PDFExtractor) PDFExtractor`

- [ ] **Step 1: Write the failing test**

```go
func TestCachingExtractor_SameBytesExtractedOnce(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.pdf"), filepath.Join(dir, "copy.pdf")
	require.NoError(t, os.WriteFile(a, []byte("%PDF-1.4 same"), 0o600))
	require.NoError(t, os.WriteFile(b, []byte("%PDF-1.4 same"), 0o600))

	inner := &callCountingExtractor{text: "hello"}
	ex := newCachingExtractor(inner)

	t1, err := ex.ExtractText(a)
	require.NoError(t, err)
	t2, err := ex.ExtractText(b)
	require.NoError(t, err)
	assert.Equal(t, "hello", t1)
	assert.Equal(t, t1, t2)
	assert.Equal(t, 1, inner.calls, "a temp copy of the same PDF is not extracted twice")
}
```
(Adjust field names to `callCountingExtractor`'s actual fields at pdfparser_test.go:315-325.)

- [ ] **Step 2: Run** — FAIL (undefined `newCachingExtractor`).

- [ ] **Step 3: Implement** in `extractor.go`:

```go
// cachingExtractor runs the inner extractor once per distinct PDF content.
// Format detection extracts the original file, then Parse extracts a temp
// copy of the same bytes; keying by content hash makes the second a hit.
type cachingExtractor struct {
	inner PDFExtractor
	mu    sync.Mutex
	texts map[[sha256.Size]byte]string
}

func newCachingExtractor(inner PDFExtractor) PDFExtractor {
	return &cachingExtractor{inner: inner, texts: map[[sha256.Size]byte]string{}}
}

func (c *cachingExtractor) ExtractText(pdfPath string) (string, error) {
	data, err := os.ReadFile(pdfPath) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return c.inner.ExtractText(pdfPath)
	}
	sum := sha256.Sum256(data)
	c.mu.Lock()
	text, ok := c.texts[sum]
	c.mu.Unlock()
	if ok {
		return text, nil
	}
	text, err = c.inner.ExtractText(pdfPath)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.texts[sum] = text
	c.mu.Unlock()
	return text, nil
}
```
In `NewAdapter`, after defaulting a nil extractor: `extractor = newCachingExtractor(extractor)`. Delete `var extractTextFromPDF`; make `RealPDFExtractor.ExtractText` call `extractTextFromPDFImpl(pdfPath)`. At ≈143 replace `getDefaultLogger()` with `logger`.

- [ ] **Step 4: Run** — `go test ./internal/pdfparser/ -v 2>&1 | grep -E '^(---|ok|FAIL)'` → PASS.
- [ ] **Step 5: Commit** `perf(pdf): extract each PDF once across detection and parsing`.

### Task F3: Split pdfparser_helpers.go and drop the local min

**Files:**
- Create: `internal/pdfparser/extract.go` (`getDefaultLogger`, `extractTextFromPDFImpl`)
- Create: `internal/pdfparser/viseca_pdf.go` (`parseVisecaTransactionsWithCategorizer`, `formatDate`)
- Create: `internal/pdfparser/text.go` (`extractAmount`, `cleanDescription`, `extractPayee`, `extractMerchant`, `preProcessText`, `containsMerchantIdentifier`, `containsAmount`)
- Modify: `internal/pdfparser/pdfparser_helpers.go` (keeps regex vars, `parseTransactionsWithCategorizer`, `finalizeTransactionWithCategorizer`, `sortTransactions`, `deduplicateTransactions`, `determineCreditDebit`)
- Modify: `internal/pdfparser/pdfparser.go` (delete the orphaned doc comment ≈90-98)
- Modify: `internal/pdfparser/pdfparser_test.go` (delete `TestMinFunction`)

- [ ] **Step 1:** Move functions verbatim (cut/paste, each new file gets only the imports it uses). Delete the local `func min` (≈797-802); the Go 1.21+ builtin `min` keeps lines ≈281, 425, 656 compiling unchanged.
- [ ] **Step 2:** `go build ./... && go vet ./internal/pdfparser/ && go test ./internal/pdfparser/` → PASS.
- [ ] **Step 3:** Commit `refactor(pdf): split pdfparser helpers by concern`.

### Task F4: Lazy semantic warm-up that the first lookup waits for

**Files:**
- Modify: `internal/categorizer/semantic_strategy.go` (constructor, `Categorize`, `Shutdown`)
- Modify: `internal/categorizer/semantic_lifecycle_test.go` (tests that assume the constructor starts warm-up)

- [ ] **Step 1: Write the failing test** (in `semantic_lifecycle_test.go`, reuse `countingEmbedder`)

```go
func TestSemanticStrategy_WarmupIsLazy(t *testing.T) {
	emb := &countingEmbedder{}
	s := NewSemanticStrategyWithCache(emb, testLogger(), []models.CategoryConfig{{Name: "Courses"}}, 0.7, nil)
	t.Cleanup(s.Shutdown)
	assert.Zero(t, emb.callCount(), "building the strategy embeds nothing")

	_, _, _ = s.Categorize(context.Background(), Transaction{PartyName: "Migros"})
	assert.NotZero(t, emb.callCount(), "the first lookup warms up and waits for it")
}
```
(Use `countingEmbedder`'s actual counter accessor; if it has a `release` channel that blocks, close it before `Categorize` or use a non-blocking instance.)

- [ ] **Step 2: Run** — FAIL (constructor already embedded).

- [ ] **Step 3: Implement** — add `warmupOnce sync.Once` to `SemanticStrategy`. Move the goroutine start out of the constructor into:

```go
// startWarmup embeds the categories in the background, once.
func (s *SemanticStrategy) startWarmup() {
	s.warmupOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.mu.Lock()
		s.cancelWarmup = cancel
		s.warmupDone = make(chan struct{})
		done := s.warmupDone
		s.mu.Unlock()
		go func() {
			defer close(done)
			s.initializeEmbeddings(ctx, s.categories)
		}()
	})
}
```
In `Categorize`, after the `s.client == nil` check:
```go
	s.startWarmup()
	s.mu.RLock()
	done := s.warmupDone
	s.mu.RUnlock()
	select {
	case <-done:
	case <-ctx.Done():
		return models.Category{}, false, ctx.Err()
	}
```
then the existing `initialized` check (still skips if warm-up was cancelled or failed). `Shutdown`: read `cancelWarmup`/`warmupDone` under `s.mu`; return if nil. Update older lifecycle tests that expected warm-up at construction to call `Categorize` first.

- [ ] **Step 4: Run** — `go test ./internal/categorizer/ -race -v 2>&1 | grep -E '^(---|ok|FAIL)'` → PASS.
- [ ] **Step 5: Commit** `perf(categorizer): start semantic warm-up on first use and wait for it`.

### Task F5: Small duplications

**Files:**
- Modify: `internal/revolutinvestmentparser/revolutinvestmentparser.go` (7 blocks ≈190-321)
- Create: `internal/common/columns.go`, `internal/common/columns_test.go`
- Modify: header checks in `internal/debitparser/debitparser.go` (≈296-308), `internal/revolutparser/revolutparser.go` (≈371-386), `internal/selmaparser/selmaparser.go` (≈244-290), `internal/visecaparser/visecaparser.go` (≈318-348), `internal/revolutcryptoparser/adapter.go` (≈45-61)
- Create: `internal/parser/errors.go`; Modify: `internal/container/detect.go:29-30`, `internal/batch/processor.go:43-45`

**Interfaces:**
- Produces: `func MissingColumn(header []string, required ...string) string`, `var parser.ErrFormatNotRecognized`.

- [ ] **Step 1: Write the failing tests**

`internal/common/columns_test.go`:
```go
func TestMissingColumn(t *testing.T) {
	header := []string{"﻿Date", " Amount ", "Currency"}
	assert.Equal(t, "", MissingColumn(header, "Date", "Amount", "Currency"))
	assert.Equal(t, "State", MissingColumn(header, "Date", "State"))
}
```
`internal/batch/processor_test.go` (add):
```go
func TestErrNoParserIsTheSharedSentinel(t *testing.T) {
	assert.ErrorIs(t, ErrNoParser, parser.ErrFormatNotRecognized)
}
```

- [ ] **Step 2: Run** — FAIL (undefined).

- [ ] **Step 3: Implement**

`internal/common/columns.go`:
```go
package common

import "strings"

// MissingColumn returns the first required column absent from a CSV header,
// or "" when all are present. Header names are compared trimmed and without
// a UTF-8 byte-order mark, which spreadsheet exports often prepend.
func MissingColumn(header []string, required ...string) string {
	present := make(map[string]bool, len(header))
	for _, name := range header {
		present[strings.TrimSpace(strings.TrimPrefix(name, "﻿"))] = true
	}
	for _, name := range required {
		if !present[name] {
			return name
		}
	}
	return ""
}
```
In each listed validator, replace the map-building + loop with `if missing := common.MissingColumn(header, required...); missing != "" { <existing missing-column action, using missing as the column name> }`. Keep each parser's existing return values and log messages exactly (selma still returns its `ValidationError`, viseca still `false, nil`). revolut-investment keeps its positional check.

`internal/parser/errors.go`:
```go
package parser

import "errors"

// ErrFormatNotRecognized is returned when no parser accepts a file.
var ErrFormatNotRecognized = errors.New("no parser recognizes this file format")
```
`detect.go`: `var ErrFormatNotRecognized = parser.ErrFormatNotRecognized`. `processor.go`: `var ErrNoParser = parser.ErrFormatNotRecognized`.

revolutinvestmentparser — add and use:
```go
// parseDecimalField parses one numeric column, or reports which one failed.
func parseDecimalField(field, raw, what string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, &parsererror.DataExtractionError{
			FilePath:       "(from reader)",
			FieldName:      field,
			RawDataSnippet: raw,
			Msg:            fmt.Sprintf("failed to parse %s: %v", what, err),
		}
	}
	return value, nil
}
```
Each block becomes, e.g. for BUY quantity:
```go
		if row.Quantity != "" {
			quantity, err := parseDecimalField("Quantity", row.Quantity, "quantity")
			if err != nil {
				return models.Transaction{}, err
			}
			builder = builder.WithNumberOfShares(quantity)
		}
```
Keep each block's `what` text identical to its current message prefix (`quantity`, `price per share`, `total amount`, `dividend amount`, `cash top-up amount`, `amount`) and its existing `cleanAmountString` call on the raw value where it has one (`RawDataSnippet` stays the uncleaned `row.X` — pass the cleaned string to parse and keep the snippet raw by adding a `snippet` param only if a test checks it; otherwise snippet = the parsed string).

- [ ] **Step 4: Run** — `go test ./... 2>&1 | grep FAIL` → no output; `make lint` → 0 issues.
- [ ] **Step 5: Commit** `refactor: share CSV column check, decimal field parsing and format sentinel`.

### Task F6: CHANGELOG, regression, PR, release

- [ ] **Step 1: CHANGELOG** `### Changed`: `- Run pdftotext once per PDF, start semantic embedding warm-up only when a command categorizes, and share CSV header checks across parsers.`
- [ ] **Step 2:** `make test && make lint`, Task R Step 3, push, `gh pr create --title "chore: housekeeping from the audit"`, CI, merge.
- [ ] **Step 3: Release v4.2.0** — branch `release/v4.2.0`: insert `## [4.2.0] - <today>` under `## [Unreleased]` in CHANGELOG, PR, merge, then on main: `git tag -a v4.2.0 -m v4.2.0 && git push origin v4.2.0`; watch the Release workflow to success. Check `git tag --sort=-v:refname | head -1` first so 4.2.0 is next.
