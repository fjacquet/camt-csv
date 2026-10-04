# iCompta Recategorize Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `camt-csv recategorize preview|apply`, which runs the four-tier categorizer over an iCompta database, writes a reviewable CSV report, then applies exactly that report to the database with backup and logging.

**Architecture:** A new `internal/icompta` package holds pure rules (`policy`, `selector`, `report`) and two SQLite-facing units (`store` read-only, `apply` read-write). A `Classifier` interface wraps the existing `Categorizer` so the preview loop is testable with a fake. A thin Cobra command in `cmd/recategorize` wires them through the DI container.

**Tech Stack:** Go 1.27.1, `modernc.org/sqlite` (pure Go, driver name `"sqlite"`), `shopspring/decimal`, `golang.org/x/text/unicode/norm`, Cobra, existing `logging.Logger`, testify.

**Spec:** `docs/superpowers/specs/2026-10-04-icompta-recategorize-design.md`

## Global Constraints

- Module `fjacquet/camt-csv`, Go `1.27.1` (the `go` directive in `go.mod`).
- Every numeric amount is `shopspring/decimal`, never `float` or `int`.
- No global mutable state: dependencies come from the `Container`; `internal/icompta` takes everything as parameters.
- Never use `math/rand` (Semgrep blocks it).
- SQLite driver is `modernc.org/sqlite`, opened with `sql.Open("sqlite", dsn)`. Read-only DSN is `file:<abs path>?mode=ro`, built with `net/url` so spaces are escaped.
- iCompta writes `lastModificationDate` in UTC as `YYYY-MM-DD HH:MM:SS`. Apply writes the same format.
- Tests never open the real `ic25.cdb`. Each test builds a fixture database in `t.TempDir()`.
- `CHANGELOG.md` gets entries under `## [Unreleased]`, imperative mood.
- Main is protected: work happens on `feat/icompta-recategorize`, delivered by PR.
- Commit messages end with `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`. PR bodies end with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- Run Go via `rtk`-prefixed commands where RTK has a filter (`rtk go test` is not filtered; plain `go test` is fine).

## Review Focus

Inputs and conditions the spec implies but the obvious tests would miss, most likely first. Each has a pinning test in the owning task.

1. **Empty or NULL split amount.** 2 real splits have none. Expected: no panic, falls back to the transaction amount, else zero (treated as a credit), still categorized. Pinned in Task 4.
2. **Unicode-decomposed category names.** macOS can hand back `Non Classe` + combining accent. Expected: still recognised as "unknown" and still resolvable to its `ICCategory` ID. Pinned in Tasks 2 and 4.
3. **Report round-tripped through a spreadsheet.** BOM, CRLF, `YES`/`Yes`, trailing commas on comment lines. Expected: read correctly, or fail with the offending line number, never silently skip. Pinned in Task 3.
4. **Applying the same report twice.** Expected: the second run changes nothing and logs every row as `category changed since preview`. Pinned in Task 7.
5. **Ctrl-C during a long preview.** Expected: a valid, partial report of what was done, not an empty file and not a hang. Pinned in Task 6.

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/icompta/types.go` | `Candidate`, tier constants, `Action`, `Decision`, `Proposal` |
| `internal/icompta/policy.go` | `IsUnknownCategory`, `SameCategory`, `Exclusion`, `Decide` (pure) |
| `internal/icompta/report.go` | `Row`, `Report`, `Write`, `ReadReport` (pure) |
| `internal/icompta/store.go` | read-only SQLite access: `OpenReadOnly`, `Snapshot`, `State`, `Categories` |
| `internal/icompta/preview.go` | `Classifier`, `NewCategorizerClassifier`, `Preview` loop |
| `internal/icompta/apply.go` | `Apply` with guards, backup, transaction, logging |
| `internal/categorizer/categorizer.go` | add `CategorizeLocal` |
| `cmd/recategorize/recategorize.go` | Cobra command, `preview` and `apply` subcommands |
| `main.go` | register the command |
| `docs/icompta-recategorize.md` | user guide, iCloud note |

---

### Task 1: Align the spec with what planning found

**Files:**
- Modify: `docs/superpowers/specs/2026-10-04-icompta-recategorize-design.md`

**Interfaces:**
- Consumes: none
- Produces: the spec the later tasks cite.

- [ ] **Step 1: Append the amendments section**

Append to the end of the spec file:

```markdown

## Amendments from implementation planning

1. **Real-category splits run only the local tiers.** For a split that already
   has a real category, only direct mapping and keyword can override it (policy
   table), so the networked tiers are never asked. This adds
   `Categorizer.CategorizeLocal` and keeps the AI/semantic quota for the ~720
   splits that need it. Empty or "unknown" splits run the full chain through
   `Categorizer.CategorizeTransaction`, which does not auto-learn.
2. **State marker is data-based, not file size.** `db_state` is
   `splits=<count>;split_modified=<max>;tx_modified=<max>`. File size can change
   when iCompta merely opens and closes the database, which would force a
   needless, slow re-preview. The per-row `old_category` check still protects
   every individual write.
3. **Report has a `reason` column** between `decision` and `apply`. Columns:
   `split_id,date,name,amount,old_category,new_category,tier,decision,reason,apply`.
4. **A cancelled preview writes a partial report.** Rows already decided are
   valid; `apply` handles a partial report like any other.
5. **Apply refuses a database with a non-empty `-wal` file**, because a plain
   file copy would not be a consistent backup.
6. **Category comparison is Unicode-NFC normalised**, so a decomposed accent
   in the database still matches.
```

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/specs/2026-10-04-icompta-recategorize-design.md
git commit -m "docs: amend recategorize spec with planning findings

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Domain types, selector and policy (pure)

**Files:**
- Create: `internal/icompta/types.go`
- Create: `internal/icompta/policy.go`
- Test: `internal/icompta/policy_test.go`

**Interfaces:**
- Consumes: `models.CategoryUncategorized` (`"Uncategorized"`) from `internal/models/constants.go`.
- Produces (used by Tasks 3, 4, 6, 7):
  - `type Candidate struct { SplitID, Date, Name, Payee, Comment string; Amount decimal.Decimal; CategoryID, CategoryName string; IsInvestment, IsLinked bool }`
  - `func (c Candidate) PartyName() string`, `func (c Candidate) IsDebtor() bool`
  - `const TierDirectMapping = "direct_mapping"`, `TierKeyword = "keyword"`, `TierSemantic = "semantic"`, `TierAI = "ai"`
  - `type Action string` with `ActionChange = "change"`, `ActionKeep = "keep"`, `ActionSkip = "skipped"`
  - `type Decision struct { Action Action; Reason string }`
  - `type Proposal struct { Category, Tier string }`
  - `func IsUnknownCategory(name string) bool`, `func SameCategory(a, b string) bool`
  - `func Exclusion(c Candidate) string`
  - `func Decide(current string, p Proposal) Decision`
  - `const ReasonUnchanged = "unchanged"`, `ReasonNoSuggestion = "no suggestion"`

- [ ] **Step 1: Write the failing test**

`internal/icompta/policy_test.go`:

```go
package icompta

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		proposal   Proposal
		wantAction Action
		wantReason string
	}{
		{"empty filled by ai", "", Proposal{"Alimentation", TierAI}, ActionChange, ""},
		{"unknown replaced by semantic", "Divers", Proposal{"Voiture", TierSemantic}, ActionChange, ""},
		{"real replaced by keyword", "Alimentation", Proposal{"Courses", TierKeyword}, ActionChange, ""},
		{"real replaced by direct mapping", "Alimentation", Proposal{"Courses", TierDirectMapping}, ActionChange, ""},
		{"real never replaced by ai", "Alimentation", Proposal{"Courses", TierAI}, ActionKeep, "does not override"},
		{"real never replaced by semantic", "Alimentation", Proposal{"Courses", TierSemantic}, ActionKeep, "does not override"},
		{"same category is unchanged", "Alimentation", Proposal{"Alimentation", TierDirectMapping}, ActionKeep, ReasonUnchanged},
		{"uncategorized proposal is no suggestion", "", Proposal{"Uncategorized", ""}, ActionKeep, ReasonNoSuggestion},
		{"failed ai proposal is no suggestion", "Divers", Proposal{"Uncategorized (AI)", TierAI}, ActionKeep, ReasonNoSuggestion},
		{"empty proposal is no suggestion", "Non Classé", Proposal{}, ActionKeep, ReasonNoSuggestion},
		{"decomposed accent still unknown", "Non Classé", Proposal{"Alimentation", TierAI}, ActionChange, ""},
		{"decomposed accent still same", "Séjours", Proposal{"Séjours", TierKeyword}, ActionKeep, ReasonUnchanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.current, tt.proposal)
			assert.Equal(t, tt.wantAction, got.Action)
			assert.Contains(t, got.Reason, tt.wantReason)
		})
	}
}

func TestIsUnknownCategory(t *testing.T) {
	for _, n := range []string{"", "  ", "Divers", "Non Classé", "Autre", "Uncategorized", "Uncategorized (AI)"} {
		assert.True(t, IsUnknownCategory(n), n)
	}
	for _, n := range []string{"Alimentation", "Voiture", "Épargne"} {
		assert.False(t, IsUnknownCategory(n), n)
	}
}

func TestExclusion(t *testing.T) {
	assert.Equal(t, "", Exclusion(Candidate{}))
	assert.Equal(t, "investment", Exclusion(Candidate{IsInvestment: true}))
	assert.Equal(t, "linked transfer", Exclusion(Candidate{IsLinked: true}))
	assert.Equal(t, "investment", Exclusion(Candidate{IsInvestment: true, IsLinked: true}))
}

func TestCandidate_PartyNameAndDirection(t *testing.T) {
	assert.Equal(t, "Migros", Candidate{Name: "Achat 12/04", Payee: " Migros "}.PartyName())
	assert.Equal(t, "Achat 12/04", Candidate{Name: "Achat 12/04"}.PartyName())
	assert.True(t, Candidate{Amount: decimal.RequireFromString("-8.70")}.IsDebtor())
	assert.False(t, Candidate{Amount: decimal.RequireFromString("8.70")}.IsDebtor())
	assert.False(t, Candidate{}.IsDebtor(), "zero amount is not a debit")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/icompta/ -run 'TestDecide|TestIsUnknown|TestExclusion|TestCandidate' -v`
Expected: FAIL, build error `undefined: Decide` (package does not exist yet).

- [ ] **Step 3: Write the implementation**

`internal/icompta/types.go`:

```go
// Package icompta recategorizes transactions stored in an iCompta database.
package icompta

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Categorization tiers, as reported in models.Category.Source.
const (
	TierDirectMapping = "direct_mapping"
	TierKeyword       = "keyword"
	TierSemantic      = "semantic"
	TierAI            = "ai"
)

// Action is what the policy decided for one split.
type Action string

const (
	ActionChange Action = "change"
	ActionKeep   Action = "keep"
	ActionSkip   Action = "skipped"
)

// Reasons attached to a keep decision that callers branch on.
const (
	ReasonUnchanged    = "unchanged"
	ReasonNoSuggestion = "no suggestion"
)

// Decision is the policy outcome: an action and, when it is not a plain
// change, the reason.
type Decision struct {
	Action Action
	Reason string
}

// Proposal is what the categorizer suggested for a split.
type Proposal struct {
	Category string
	Tier     string
}

// Candidate is one iCompta split with the context needed to categorize it.
type Candidate struct {
	SplitID      string
	Date         string // ICTransaction.date, YYYY-MM-DD
	Name         string // ICTransaction.name, usually the bank label
	Payee        string
	Comment      string
	Amount       decimal.Decimal
	CategoryID   string // "" when the split has no category
	CategoryName string // "" when CategoryID is empty or dangling
	IsInvestment bool
	IsLinked     bool
}

// PartyName is the payee when set, else the transaction name. In real data the
// payee is mostly empty and the name carries the bank label.
func (c Candidate) PartyName() string {
	if p := strings.TrimSpace(c.Payee); p != "" {
		return p
	}
	return strings.TrimSpace(c.Name)
}

// IsDebtor follows the repo convention (see revolutinvestmentparser): money
// out is a debtor transaction.
func (c Candidate) IsDebtor() bool {
	return c.Amount.IsNegative()
}
```

`internal/icompta/policy.go`:

```go
package icompta

import (
	"strings"

	"golang.org/x/text/unicode/norm"

	"fjacquet/camt-csv/internal/models"
)

// unknownCategories are the categories that mean "nobody decided". A split in
// one of these may be replaced by any tier.
var unknownCategories = map[string]bool{
	"Divers":             true,
	"Non Classé":         true,
	"Autre":              true,
	"Uncategorized":      true,
	"Uncategorized (AI)": true,
}

// normalize makes category names comparable: macOS can return decomposed
// accents, so compare NFC.
func normalize(s string) string {
	return norm.NFC.String(strings.TrimSpace(s))
}

// IsUnknownCategory reports whether a category name carries no decision.
func IsUnknownCategory(name string) bool {
	n := normalize(name)
	return n == "" || unknownCategories[n]
}

// SameCategory compares two category names after normalisation.
func SameCategory(a, b string) bool {
	return normalize(a) == normalize(b)
}

// Exclusion returns why a split is out of scope, or "" when it is in scope.
// Investment splits and linked transfers are legitimately uncategorised.
func Exclusion(c Candidate) string {
	switch {
	case c.IsInvestment:
		return "investment"
	case c.IsLinked:
		return "linked transfer"
	}
	return ""
}

// usable reports whether the categorizer actually suggested something.
func (p Proposal) usable() bool {
	switch normalize(p.Category) {
	case "", models.CategoryUncategorized, "Uncategorized (AI)":
		return false
	}
	return true
}

// Decide applies the overwrite rule: an unknown or empty category takes any
// suggestion; a real category is replaced only by a deterministic tier.
func Decide(current string, p Proposal) Decision {
	if !p.usable() {
		return Decision{ActionKeep, ReasonNoSuggestion}
	}
	if SameCategory(current, p.Category) {
		return Decision{ActionKeep, ReasonUnchanged}
	}
	if IsUnknownCategory(current) {
		return Decision{ActionChange, ""}
	}
	if p.Tier == TierDirectMapping || p.Tier == TierKeyword {
		return Decision{ActionChange, ""}
	}
	return Decision{ActionKeep, "tier " + p.Tier + " does not override a chosen category"}
}
```

- [ ] **Step 4: Promote the x/text dependency and run the tests**

Run: `go mod tidy && go test ./internal/icompta/ -v`
Expected: PASS. `go.mod` now lists `golang.org/x/text` as a direct requirement.

- [ ] **Step 5: Commit**

```bash
git add internal/icompta go.mod go.sum
git commit -m "feat(icompta): add overwrite policy and split selector

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Report CSV (pure)

**Files:**
- Create: `internal/icompta/report.go`
- Test: `internal/icompta/report_test.go`

**Interfaces:**
- Consumes: `Action` (Task 2).
- Produces (used by Tasks 6, 7, 8):
  - `type Row struct { SplitID, Date, Name, Amount, OldCategory, NewCategory, Tier string; Decision Action; Reason string; Apply bool }`
  - `type Report struct { DBState string; Rows []Row }`
  - `func (r Report) Write(w io.Writer) error`
  - `func ReadReport(rd io.Reader) (Report, error)`
  - For a skipped row, `Decision` is `ActionSkip` and `Reason` holds the reason; the CSV `decision` column is `skipped` and `reason` carries the text.

- [ ] **Step 1: Write the failing test**

`internal/icompta/report_test.go`:

```go
package icompta

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleReport() Report {
	return Report{
		DBState: "splits=3;split_modified=2026-10-04 19:32:27;tx_modified=2026-10-04 19:49:30",
		Rows: []Row{
			{SplitID: "S1", Date: "2026-01-02", Name: "Café \"Chez Paul\", Genève\nterrasse", Amount: "-12.50",
				OldCategory: "Divers", NewCategory: "Restaurants", Tier: TierKeyword, Decision: ActionChange, Apply: true},
			{SplitID: "S2", Date: "2026-01-03", Name: "# looks like a comment", Amount: "8.7",
				OldCategory: "", NewCategory: "", Decision: ActionKeep, Reason: ReasonNoSuggestion},
			{SplitID: "S3", Date: "2026-01-04", Name: "X", Amount: "1",
				OldCategory: "Divers", NewCategory: "Nope", Tier: TierAI, Decision: ActionSkip, Reason: "unknown category"},
		},
	}
}

func TestReport_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, sampleReport().Write(&buf))

	got, err := ReadReport(&buf)
	require.NoError(t, err)
	assert.Equal(t, sampleReport(), got)
}

// A report edited in a spreadsheet comes back with a BOM, CRLF line endings,
// trailing commas on the comment lines, and a hand-typed "YES".
func TestReadReport_SpreadsheetEdited(t *testing.T) {
	in := "\xef\xbb\xbf# camt-csv recategorize report v1,,,,,,,,,\r\n" +
		"# db_state=splits=1;split_modified=a;tx_modified=b,,,,,,,,,\r\n" +
		"split_id,date,name,amount,old_category,new_category,tier,decision,reason,apply\r\n" +
		"S1,2026-01-02,Coop,-5,Divers,Alimentation,keyword,change,,YES\r\n"

	got, err := ReadReport(strings.NewReader(in))
	require.NoError(t, err)
	assert.Equal(t, "splits=1;split_modified=a;tx_modified=b", got.DBState)
	require.Len(t, got.Rows, 1)
	assert.True(t, got.Rows[0].Apply)
	assert.Equal(t, ActionChange, got.Rows[0].Decision)
}

func TestReadReport_Errors(t *testing.T) {
	const header = "split_id,date,name,amount,old_category,new_category,tier,decision,reason,apply\n"
	tests := []struct {
		name, in, want string
	}{
		{"not a report", "a,b,c\n", "not a camt-csv recategorize report"},
		{"missing state", "# camt-csv recategorize report v1\n" + header, "db_state"},
		{"wrong header", "# camt-csv recategorize report v1\n# db_state=x\nfoo,bar\n", "header"},
		{"bad apply", "# camt-csv recategorize report v1\n# db_state=x\n" + header +
			"S1,d,n,1,a,b,keyword,change,,maybe\n", "line 4"},
		{"bad decision", "# camt-csv recategorize report v1\n# db_state=x\n" + header +
			"S1,d,n,1,a,b,keyword,explode,,yes\n", "line 4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadReport(strings.NewReader(tt.in))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/icompta/ -run 'TestReport|TestReadReport' -v`
Expected: FAIL, `undefined: Report`.

- [ ] **Step 3: Write the implementation**

`internal/icompta/report.go`:

```go
package icompta

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

const (
	reportMagic = "# camt-csv recategorize report v1"
	stateKey    = "# db_state="
)

var reportHeader = []string{
	"split_id", "date", "name", "amount", "old_category", "new_category",
	"tier", "decision", "reason", "apply",
}

// Row is one reviewed decision. Only rows with Decision == ActionChange and
// Apply == true are ever written to the database.
type Row struct {
	SplitID     string
	Date        string
	Name        string
	Amount      string
	OldCategory string
	NewCategory string
	Tier        string
	Decision    Action
	Reason      string
	Apply       bool
}

// Report is what preview writes and apply reads.
type Report struct {
	DBState string
	Rows    []Row
}

// Write encodes the report: two comment lines, then a CSV with a header.
func (r Report) Write(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s\n%s%s\n", reportMagic, stateKey, r.DBState); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(reportHeader); err != nil {
		return err
	}
	for _, row := range r.Rows {
		apply := "no"
		if row.Apply {
			apply = "yes"
		}
		rec := []string{
			row.SplitID, row.Date, row.Name, row.Amount, row.OldCategory, row.NewCategory,
			row.Tier, string(row.Decision), row.Reason, apply,
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ReadReport decodes a report, tolerating what a spreadsheet does to a CSV:
// a UTF-8 BOM, CRLF line endings, trailing commas on the comment lines and any
// case of yes/no.
func ReadReport(rd io.Reader) (Report, error) {
	br := bufio.NewReader(rd)
	if b, err := br.Peek(3); err == nil && bytes.Equal(b, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}

	var rep Report
	magicSeen := false
	for {
		first, err := br.Peek(1)
		if err != nil || first[0] != '#' {
			break
		}
		line, err := br.ReadString('\n')
		line = strings.TrimRight(line, ", \r\n")
		switch {
		case line == reportMagic:
			magicSeen = true
		case strings.HasPrefix(line, stateKey):
			rep.DBState = strings.TrimPrefix(line, stateKey)
		}
		if err != nil {
			break
		}
	}
	if !magicSeen {
		return Report{}, fmt.Errorf("not a camt-csv recategorize report (missing %q line)", reportMagic)
	}
	if rep.DBState == "" {
		return Report{}, fmt.Errorf("report has no db_state line")
	}

	cr := csv.NewReader(br)
	cr.FieldsPerRecord = len(reportHeader)
	records, err := cr.ReadAll()
	if err != nil {
		return Report{}, fmt.Errorf("read report rows: %w", err)
	}
	if len(records) == 0 || strings.Join(records[0], ",") != strings.Join(reportHeader, ",") {
		return Report{}, fmt.Errorf("report header is not %q", strings.Join(reportHeader, ","))
	}

	for i, rec := range records[1:] {
		lineNo := i + 4 // 2 comment lines, 1 header, 1-based
		decision := Action(rec[7])
		switch decision {
		case ActionChange, ActionKeep, ActionSkip:
		default:
			return Report{}, fmt.Errorf("line %d: decision %q is not change, keep or skipped", lineNo, rec[7])
		}
		var apply bool
		switch strings.ToLower(strings.TrimSpace(rec[9])) {
		case "yes":
			apply = true
		case "no":
		default:
			return Report{}, fmt.Errorf("line %d: apply %q is not yes or no", lineNo, rec[9])
		}
		rep.Rows = append(rep.Rows, Row{
			SplitID: rec[0], Date: rec[1], Name: rec[2], Amount: rec[3],
			OldCategory: rec[4], NewCategory: rec[5], Tier: rec[6],
			Decision: decision, Reason: rec[8], Apply: apply,
		})
	}
	return rep, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/icompta/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/icompta/report.go internal/icompta/report_test.go
git commit -m "feat(icompta): add reviewable recategorization report

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Read-only store and SQLite driver

**Files:**
- Create: `internal/icompta/store.go`
- Test: `internal/icompta/store_test.go`
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `Candidate`, `normalize` (Task 2).
- Produces (used by Tasks 6, 7, 8):
  - `type Categories struct{ ... }` with `IDByName(name string) (string, bool)`, `NameByID(id string) string`, `Len() int`
  - `type Snapshot struct { Candidates []Candidate; Categories Categories; State string }`
  - `type Store struct{ ... }`
  - `func OpenReadOnly(ctx context.Context, path string) (*Store, error)`
  - `func (s *Store) Snapshot(ctx context.Context) (Snapshot, error)`
  - `func (s *Store) State(ctx context.Context) (string, error)`
  - `func (s *Store) Close() error`
  - `func readState(ctx context.Context, q interface{ QueryRowContext(context.Context, string, ...any) *sql.Row }) (string, error)` (package-private, reused by Task 7)
  - test helper `newFixtureDB(t *testing.T) string` creating the minimal schema (reused by Task 7).

- [ ] **Step 1: Check the driver API, then add it**

Verified with Context7 (`/gitlab_cznic/sqlite`, `/websites/pkg_go_dev_modernc_org_sqlite`) on 2026-10-04: driver name `"sqlite"`; `file:<path>?mode=ro` passes through to SQLite URI handling; `_timeout=<ms>` is the validated shorthand for `PRAGMA busy_timeout`; `_txlock=immediate` is supported; `_pragma=` values run verbatim and are unvalidated, so the plan does not use them. Also verified: Cobra `MarkFlagRequired` is enforced in `execute()` only (hence the explicit checks in Task 8), `shopspring/decimal` `String()` trims trailing zeros (`-12.30` prints `-12.3`) and `RequireFromString` panics on bad input (tests only), and `norm.NFC.String(s)` composes decomposed accents.

Run: `go get modernc.org/sqlite@latest`
Expected: `go.mod` gains `modernc.org/sqlite`.

- [ ] **Step 2: Write the failing test**

`internal/icompta/store_test.go`:

```go
package icompta

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const fixtureSchema = `
CREATE TABLE ICCategory (ID TEXT UNIQUE NOT NULL, name TEXT NOT NULL);
CREATE TABLE ICTransaction (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, date TEXT NOT NULL, name TEXT NOT NULL,
	comment TEXT, amount TEXT, payee TEXT, investmentTransactionInfo TEXT);
CREATE TABLE ICTransactionSplit (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, "transaction" TEXT NOT NULL,
	amount TEXT, category TEXT, linkedSplit TEXT);
`

// newFixtureDB builds a database with the columns recategorize uses and a small
// data set covering every case the reader must handle.
func newFixtureDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ic test.cdb") // a space: the DSN must escape it
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()

	stmts := []string{
		fixtureSchema,
		`INSERT INTO ICCategory VALUES ('C-ALI','Alimentation'),('C-DIV','Divers'),('C-NC','Non Classe` + "́" + `')`,
		// T1: categorised, debit
		`INSERT INTO ICTransaction VALUES ('T1','2026-01-01','2026-01-02','COOP CITY','', '-10.50','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S1',NULL,'T1','-10.50','C-ALI','')`,
		// T2: no category, payee set
		`INSERT INTO ICTransaction VALUES ('T2',NULL,'2026-01-03','VIREMENT','ref 7','200','Employeur SA',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S2',NULL,'T2','200',NULL,NULL)`,
		// T3: investment
		`INSERT INTO ICTransaction VALUES ('T3',NULL,'2026-01-04','BUY','', '-500','','info')`,
		`INSERT INTO ICTransactionSplit VALUES ('S3',NULL,'T3','-500',NULL,NULL)`,
		// T4: linked transfer leg
		`INSERT INTO ICTransaction VALUES ('T4',NULL,'2026-01-05','TRANSFER','', '-50','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S4',NULL,'T4','-50',NULL,'S9')`,
		// T5: split amount empty, falls back to the transaction amount
		`INSERT INTO ICTransaction VALUES ('T5',NULL,'2026-01-06','SPLITLESS','', '-7.25','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S5',NULL,'T5','',NULL,NULL)`,
		// T6: both amounts empty
		`INSERT INTO ICTransaction VALUES ('T6',NULL,'2026-01-07','NOAMOUNT','', NULL,'',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S6',NULL,'T6',NULL,'C-DIV',NULL)`,
		// T7: category ID that no longer exists
		`INSERT INTO ICTransaction VALUES ('T7',NULL,'2026-01-08','DANGLING','', '-1','',NULL)`,
		`INSERT INTO ICTransactionSplit VALUES ('S7',NULL,'T7','-1','C-GONE',NULL)`,
	}
	for _, s := range stmts {
		_, err := db.Exec(s)
		require.NoError(t, err, s)
	}
	return path
}

func openFixture(t *testing.T) (*Store, string) {
	t.Helper()
	path := newFixtureDB(t)
	s, err := OpenReadOnly(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func TestStore_Snapshot(t *testing.T) {
	s, _ := openFixture(t)
	snap, err := s.Snapshot(context.Background())
	require.NoError(t, err)

	byID := map[string]Candidate{}
	for _, c := range snap.Candidates {
		byID[c.SplitID] = c
	}
	require.Len(t, byID, 7)

	s1 := byID["S1"]
	assert.Equal(t, "COOP CITY", s1.Name)
	assert.Equal(t, "Alimentation", s1.CategoryName)
	assert.True(t, s1.Amount.Equal(decimal.RequireFromString("-10.50")))
	assert.True(t, s1.IsDebtor())

	s2 := byID["S2"]
	assert.Equal(t, "Employeur SA", s2.PartyName())
	assert.Equal(t, "ref 7", s2.Comment)
	assert.Empty(t, s2.CategoryID)

	assert.True(t, byID["S3"].IsInvestment)
	assert.True(t, byID["S4"].IsLinked)

	// Review focus 1: empty split amount falls back, then falls to zero.
	assert.True(t, byID["S5"].Amount.Equal(decimal.RequireFromString("-7.25")))
	assert.True(t, byID["S6"].Amount.IsZero())
	assert.Equal(t, "Divers", byID["S6"].CategoryName)

	// A dangling category ID resolves to no name rather than failing.
	assert.Equal(t, "C-GONE", byID["S7"].CategoryID)
	assert.Empty(t, byID["S7"].CategoryName)
}

// Review focus 2: the database holds a decomposed accent; the lookup is by
// normalised name.
func TestCategories_ResolveDecomposedNames(t *testing.T) {
	s, _ := openFixture(t)
	snap, err := s.Snapshot(context.Background())
	require.NoError(t, err)

	id, ok := snap.Categories.IDByName("Non Classé") // composed
	assert.True(t, ok)
	assert.Equal(t, "C-NC", id)

	_, ok = snap.Categories.IDByName("Inexistante")
	assert.False(t, ok)
	assert.Equal(t, 3, snap.Categories.Len())
}

func TestStore_StateChangesWhenDataChanges(t *testing.T) {
	s, path := openFixture(t)
	before, err := s.State(context.Background())
	require.NoError(t, err)
	assert.Contains(t, before, "splits=7")

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`UPDATE ICTransactionSplit SET lastModificationDate='2026-10-04 19:00:00' WHERE ID='S1'`)
	require.NoError(t, err)

	after, err := s.State(context.Background())
	require.NoError(t, err)
	assert.NotEqual(t, before, after)
}

func TestOpenReadOnly_Errors(t *testing.T) {
	_, err := OpenReadOnly(context.Background(), filepath.Join(t.TempDir(), "missing.cdb"))
	require.Error(t, err)

	// A database without the expected columns fails with a clear message.
	path := filepath.Join(t.TempDir(), "other.cdb")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE ICCategory (ID TEXT)`)
	require.NoError(t, err)
	db.Close()

	_, err = OpenReadOnly(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ICCategory")
}

func TestStore_IsReadOnly(t *testing.T) {
	s, _ := openFixture(t)
	_, err := s.db.Exec(`UPDATE ICTransactionSplit SET category='C-DIV' WHERE ID='S1'`)
	require.Error(t, err, "the preview connection must not be able to write")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/icompta/ -run 'TestStore|TestCategories|TestOpenReadOnly' -v`
Expected: FAIL, `undefined: OpenReadOnly`.

- [ ] **Step 4: Write the implementation**

`internal/icompta/store.go`:

```go
package icompta

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/shopspring/decimal"
	_ "modernc.org/sqlite" // pure-Go driver, registers "sqlite"
)

// requiredColumns are the columns this package reads or writes. A database
// missing one fails fast with a clear message instead of a mid-run SQL error.
var requiredColumns = map[string][]string{
	"ICCategory":         {"ID", "name"},
	"ICTransaction":      {"ID", "lastModificationDate", "date", "name", "comment", "amount", "payee", "investmentTransactionInfo"},
	"ICTransactionSplit": {"ID", "lastModificationDate", "transaction", "amount", "category", "linkedSplit"},
}

// Categories maps iCompta category names to IDs and back. Names are compared
// NFC-normalised.
type Categories struct {
	byName map[string]string
	byID   map[string]string
}

// IDByName resolves a category name to its iCompta ID.
func (c Categories) IDByName(name string) (string, bool) {
	id, ok := c.byName[normalize(name)]
	return id, ok
}

// NameByID resolves an iCompta category ID to its name, "" when unknown.
func (c Categories) NameByID(id string) string { return c.byID[id] }

// Len is the number of categories.
func (c Categories) Len() int { return len(c.byID) }

// Snapshot is a consistent read of everything a preview needs.
type Snapshot struct {
	Candidates []Candidate
	Categories Categories
	State      string
}

// Store reads an iCompta database. It never writes.
type Store struct {
	db *sql.DB
}

// dsn builds a SQLite file URI so spaces and other characters in the path are
// escaped.
func dsn(path, query string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	u := url.URL{Scheme: "file", Path: abs}
	s := u.String()
	if query != "" {
		s += "?" + query
	}
	return s, nil
}

// OpenReadOnly opens the database read-only and checks the expected columns.
func OpenReadOnly(ctx context.Context, path string) (*Store, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	uri, err := dsn(path, "mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open iCompta database: %w", err)
	}
	if err := requireColumns(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close releases the connection.
func (s *Store) Close() error { return s.db.Close() }

func requireColumns(ctx context.Context, db *sql.DB) error {
	for table, cols := range requiredColumns {
		rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
		if err != nil {
			return fmt.Errorf("inspect %s: %w", table, err)
		}
		have := map[string]bool{}
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull, pk int
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				_ = rows.Close()
				return fmt.Errorf("inspect %s: %w", table, err)
			}
			have[name] = true
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("inspect %s: %w", table, err)
		}
		_ = rows.Close()
		for _, c := range cols {
			if !have[c] {
				return fmt.Errorf("not an iCompta database this tool understands: %s.%s is missing", table, c)
			}
		}
	}
	return nil
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// readState summarises how much the data has changed: the split count and the
// latest modification date of splits and transactions. It is deliberately not
// the file size, which can change when iCompta merely opens the file.
func readState(ctx context.Context, q rowQuerier) (string, error) {
	var splits int
	var splitMod, txMod string
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(MAX(lastModificationDate),'') FROM ICTransactionSplit`).
		Scan(&splits, &splitMod); err != nil {
		return "", fmt.Errorf("read split state: %w", err)
	}
	if err := q.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(lastModificationDate),'') FROM ICTransaction`).Scan(&txMod); err != nil {
		return "", fmt.Errorf("read transaction state: %w", err)
	}
	return fmt.Sprintf("splits=%d;split_modified=%s;tx_modified=%s", splits, splitMod, txMod), nil
}

// State returns the current data-state marker.
func (s *Store) State(ctx context.Context) (string, error) {
	return readState(ctx, s.db)
}

// Snapshot reads categories, every split and the state marker in one read
// transaction.
func (s *Store) Snapshot(ctx context.Context) (Snapshot, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, fmt.Errorf("begin read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cats, err := loadCategories(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	state, err := readState(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT s.ID, t.date, t.name, COALESCE(t.payee,''), COALESCE(t.comment,''),
       COALESCE(NULLIF(s.amount,''), t.amount, ''), COALESCE(s.category,''),
       CASE WHEN COALESCE(t.investmentTransactionInfo,'') <> '' THEN 1 ELSE 0 END,
       CASE WHEN COALESCE(s.linkedSplit,'') <> '' THEN 1 ELSE 0 END
FROM ICTransactionSplit s
JOIN ICTransaction t ON t.ID = s."transaction"
ORDER BY t.date, s.ID`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read splits: %w", err)
	}
	defer rows.Close()

	var cands []Candidate
	for rows.Next() {
		var c Candidate
		var amount string
		var inv, linked int
		if err := rows.Scan(&c.SplitID, &c.Date, &c.Name, &c.Payee, &c.Comment,
			&amount, &c.CategoryID, &inv, &linked); err != nil {
			return Snapshot{}, fmt.Errorf("scan split: %w", err)
		}
		if strings.TrimSpace(amount) != "" {
			c.Amount, err = decimal.NewFromString(strings.TrimSpace(amount))
			if err != nil {
				return Snapshot{}, fmt.Errorf("split %s: amount %q: %w", c.SplitID, amount, err)
			}
		}
		c.CategoryName = cats.NameByID(c.CategoryID)
		c.IsInvestment = inv == 1
		c.IsLinked = linked == 1
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("read splits: %w", err)
	}
	return Snapshot{Candidates: cands, Categories: cats, State: state}, nil
}

type rowsQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func loadCategories(ctx context.Context, q rowsQuerier) (Categories, error) {
	rows, err := q.QueryContext(ctx, `SELECT ID, name FROM ICCategory`)
	if err != nil {
		return Categories{}, fmt.Errorf("read categories: %w", err)
	}
	defer rows.Close()
	cats := Categories{byName: map[string]string{}, byID: map[string]string{}}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return Categories{}, fmt.Errorf("scan category: %w", err)
		}
		cats.byID[id] = name
		cats.byName[normalize(name)] = id
	}
	return cats, rows.Err()
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/icompta/ -v`
Expected: PASS. If `TestStore_IsReadOnly` does not fail on write, the DSN is not being honoured: print the URI and compare with `file:///<abs path>?mode=ro`.

- [ ] **Step 6: Commit**

```bash
git add internal/icompta/store.go internal/icompta/store_test.go go.mod go.sum
git commit -m "feat(icompta): read an iCompta database read-only

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 5: `Categorizer.CategorizeLocal`

**Files:**
- Modify: `internal/categorizer/categorizer.go` (add after `CategorizeTransaction`, around line 193)
- Test: `internal/categorizer/categorize_local_test.go`

**Interfaces:**
- Consumes: `DirectMappingStrategy`, `KeywordStrategy`, `Transaction` (existing).
- Produces: `func (c *Categorizer) CategorizeLocal(ctx context.Context, transaction Transaction) (models.Category, bool)` — runs only the tiers that need no network and reports whether one matched. Does not touch the batch cache and never auto-learns.

- [ ] **Step 1: Write the failing test**

`internal/categorizer/categorize_local_test.go`:

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

// recordingChat fails the test if the AI tier is ever asked.
type recordingChat struct {
	mu    sync.Mutex
	calls int
}

func (r *recordingChat) Categorize(_ context.Context, tx models.Transaction) (models.Transaction, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return tx, nil
}

func (r *recordingChat) GetEmbedding(context.Context, string) ([]float32, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return []float32{1, 0, 0}, nil
}

func TestCategorizeLocal(t *testing.T) {
	chat := &recordingChat{}
	c := NewCategorizer(chat, nil, &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Voiture", Keywords: []string{"GARAGE"}}},
		CreditorMappings: map[string]string{"employeur sa": "Salaire"},
		DebtorMappings:   map[string]string{"coop city": "Alimentation"},
	}, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	tests := []struct {
		name       string
		tx         Transaction
		wantFound  bool
		wantName   string
		wantSource string
	}{
		{"debtor direct mapping", Transaction{PartyName: "COOP CITY", IsDebtor: true}, true, "Alimentation", "direct_mapping"},
		{"creditor direct mapping", Transaction{PartyName: "Employeur SA"}, true, "Salaire", "direct_mapping"},
		{"keyword", Transaction{PartyName: "Garage du Lac"}, true, "Voiture", "keyword"},
		{"keyword in info", Transaction{PartyName: "X", Info: "facture garage"}, true, "Voiture", "keyword"},
		{"no local match", Transaction{PartyName: "Inconnu SA"}, false, "", ""},
		{"empty party", Transaction{PartyName: "  "}, false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := c.CategorizeLocal(context.Background(), tt.tx)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantSource, got.Source)
		})
	}

	assert.Zero(t, chat.calls, "the AI client must never be consulted by CategorizeLocal")
}

// CategorizeLocal must not poison the batch cache: a later full categorization
// of the same party still has to run every tier.
func TestCategorizeLocal_DoesNotFillBatchCache(t *testing.T) {
	c := NewCategorizer(&recordingChat{}, nil, &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Voiture", Keywords: []string{"GARAGE"}}},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	_, found := c.CategorizeLocal(context.Background(), Transaction{PartyName: "Garage du Lac"})
	require.True(t, found)

	c.batchCacheMu.RLock()
	defer c.batchCacheMu.RUnlock()
	assert.Empty(t, c.batchCache)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/categorizer/ -run TestCategorizeLocal -v`
Expected: FAIL, `c.CategorizeLocal undefined`.

- [ ] **Step 3: Write the implementation**

In `internal/categorizer/categorizer.go`, directly after `CategorizeTransaction`:

```go
// CategorizeLocal tries only the tiers that need no network, direct mapping
// and keyword matching, and reports whether one matched. It does not use the
// batch cache and never auto-learns, so it is safe to run over a whole
// database without spending embedding or AI quota.
func (c *Categorizer) CategorizeLocal(ctx context.Context, transaction Transaction) (models.Category, bool) {
	if strings.TrimSpace(transaction.PartyName) == "" {
		return models.Category{}, false
	}
	for _, strategy := range c.strategies {
		switch strategy.(type) {
		case *DirectMappingStrategy, *KeywordStrategy:
		default:
			continue
		}
		category, found, err := strategy.Categorize(ctx, transaction)
		if err != nil || !found {
			continue
		}
		return category, true
	}
	return models.Category{}, false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/categorizer/ -v -run TestCategorizeLocal`
Expected: PASS. Then `go test -race ./internal/categorizer/` to confirm nothing else moved.

- [ ] **Step 5: Commit**

```bash
git add internal/categorizer/categorizer.go internal/categorizer/categorize_local_test.go
git commit -m "feat(categorizer): add CategorizeLocal for network-free tiers

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Preview loop

**Files:**
- Create: `internal/icompta/preview.go`
- Test: `internal/icompta/preview_test.go`

**Interfaces:**
- Consumes: `Candidate`, `Exclusion`, `Decide`, `IsUnknownCategory`, `Proposal`, `Action*`, `Reason*` (Task 2); `Row`, `Report` (Task 3); `Snapshot`, `Categories` (Task 4); `categorizer.Transaction`, `Categorizer.CategorizeTransaction`, `Categorizer.CategorizeLocal` (Task 5); `logging.Logger`.
- Produces (used by Task 8):
  - `type Classifier interface { Full(ctx context.Context, tx categorizer.Transaction) (models.Category, error); Local(ctx context.Context, tx categorizer.Transaction) (models.Category, bool) }`
  - `func NewCategorizerClassifier(c *categorizer.Categorizer) Classifier`
  - `func Preview(ctx context.Context, snap Snapshot, cl Classifier, log logging.Logger) Report`

- [ ] **Step 1: Write the failing test**

`internal/icompta/preview_test.go`:

```go
package icompta

import (
	"context"
	"errors"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClassifier answers from maps keyed by party name and records which method
// was used, so tests can assert the networked tiers are spared.
type fakeClassifier struct {
	full      map[string]models.Category
	fullErr   map[string]error
	local     map[string]models.Category
	fullCalls []string
	localCall []string
	onFull    func() // lets a test cancel mid-run
}

func (f *fakeClassifier) Full(_ context.Context, tx categorizer.Transaction) (models.Category, error) {
	f.fullCalls = append(f.fullCalls, tx.PartyName)
	if f.onFull != nil {
		f.onFull()
	}
	if err := f.fullErr[tx.PartyName]; err != nil {
		return models.Category{}, err
	}
	if c, ok := f.full[tx.PartyName]; ok {
		return c, nil
	}
	return models.Category{Name: models.CategoryUncategorized}, nil
}

func (f *fakeClassifier) Local(_ context.Context, tx categorizer.Transaction) (models.Category, bool) {
	f.localCall = append(f.localCall, tx.PartyName)
	c, ok := f.local[tx.PartyName]
	return c, ok
}

func testCategories() Categories {
	return Categories{
		byName: map[string]string{"Alimentation": "C-ALI", "Courses": "C-COU", "Voiture": "C-VOI", "Divers": "C-DIV"},
		byID:   map[string]string{"C-ALI": "Alimentation", "C-COU": "Courses", "C-VOI": "Voiture", "C-DIV": "Divers"},
	}
}

func cand(id, name, cat string) Candidate {
	return Candidate{SplitID: id, Date: "2026-01-02", Name: name, Amount: decimal.RequireFromString("-5"), CategoryName: cat}
}

func TestPreview(t *testing.T) {
	cl := &fakeClassifier{
		full: map[string]models.Category{
			"EMPTY AI":     {Name: "Courses", Source: TierAI},
			"EMPTY NOCAT":  {Name: "Brand New", Source: TierAI}, // not in ICCategory
			"UNKNOWN SEM":  {Name: "Voiture", Source: TierSemantic},
			"ERR":          {},
		},
		fullErr: map[string]error{"ERR": errors.New("provider down")},
		local: map[string]models.Category{
			"REAL KEYWORD": {Name: "Courses", Source: TierKeyword},
			"REAL SAME":    {Name: "Alimentation", Source: TierDirectMapping},
		},
	}
	snap := Snapshot{
		State:      "splits=9",
		Categories: testCategories(),
		Candidates: []Candidate{
			cand("S1", "EMPTY AI", ""),
			cand("S2", "EMPTY NOCAT", ""),
			cand("S3", "UNKNOWN SEM", "Divers"),
			cand("S4", "NOTHING", ""),
			cand("S5", "REAL KEYWORD", "Alimentation"),
			cand("S6", "REAL SAME", "Alimentation"),
			cand("S7", "REAL NOMATCH", "Alimentation"),
			{SplitID: "S8", Name: "BUY", IsInvestment: true},
			{SplitID: "S9", Name: "TRANSFER", IsLinked: true},
			cand("S10", "ERR", ""),
		},
	}

	rep := Preview(context.Background(), snap, cl, logging.NewMockLogger())

	assert.Equal(t, "splits=9", rep.DBState)
	got := map[string]Row{}
	for _, r := range rep.Rows {
		got[r.SplitID] = r
	}

	assert.Equal(t, Row{SplitID: "S1", Date: "2026-01-02", Name: "EMPTY AI", Amount: "-5",
		NewCategory: "Courses", Tier: TierAI, Decision: ActionChange, Apply: true}, got["S1"])
	assert.Equal(t, ActionSkip, got["S2"].Decision)
	assert.Equal(t, "unknown category", got["S2"].Reason)
	assert.False(t, got["S2"].Apply)
	assert.Equal(t, ActionChange, got["S3"].Decision)
	assert.Equal(t, "Divers", got["S3"].OldCategory)
	assert.Equal(t, ActionKeep, got["S4"].Decision)
	assert.Equal(t, ReasonNoSuggestion, got["S4"].Reason)
	assert.Equal(t, ActionChange, got["S5"].Decision)
	assert.Equal(t, TierKeyword, got["S5"].Tier)
	assert.Equal(t, ActionSkip, got["S10"].Decision)
	assert.Contains(t, got["S10"].Reason, "categorizer error")

	for _, id := range []string{"S6", "S7", "S8", "S9"} {
		_, present := got[id]
		assert.False(t, present, "%s must not appear in the report", id)
	}

	// Networked tiers are asked only for empty/unknown splits.
	assert.ElementsMatch(t, []string{"EMPTY AI", "EMPTY NOCAT", "UNKNOWN SEM", "NOTHING", "ERR"}, cl.fullCalls)
	assert.ElementsMatch(t, []string{"REAL KEYWORD", "REAL SAME", "REAL NOMATCH"}, cl.localCall)
}

func TestPreview_PassesContextToTheCategorizer(t *testing.T) {
	cl := &captureClassifier{}
	c := Candidate{SplitID: "S1", Name: "COOP", Comment: "ref 9", Date: "2026-03-04",
		Amount: decimal.RequireFromString("-12.30")}
	Preview(context.Background(), Snapshot{Categories: testCategories(), Candidates: []Candidate{c}}, cl, logging.NewMockLogger())

	require.Len(t, cl.seen, 1)
	assert.Equal(t, "COOP", cl.seen[0].PartyName)
	assert.True(t, cl.seen[0].IsDebtor)
	assert.Equal(t, "-12.3", cl.seen[0].Amount)
	assert.Equal(t, "2026-03-04", cl.seen[0].Date)
	assert.Equal(t, "COOP ref 9", cl.seen[0].Info)
}

type captureClassifier struct{ seen []categorizer.Transaction }

func (c *captureClassifier) Full(_ context.Context, tx categorizer.Transaction) (models.Category, error) {
	c.seen = append(c.seen, tx)
	return models.Category{Name: models.CategoryUncategorized}, nil
}
func (c *captureClassifier) Local(context.Context, categorizer.Transaction) (models.Category, bool) {
	return models.Category{}, false
}

// Review focus 5: Ctrl-C must leave a valid partial report, not nothing.
func TestPreview_CancelledKeepsPartialReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cl := &fakeClassifier{
		full:   map[string]models.Category{"A": {Name: "Courses", Source: TierAI}, "B": {Name: "Courses", Source: TierAI}},
		onFull: cancel, // cancel as soon as the first split is classified
	}
	snap := Snapshot{State: "s", Categories: testCategories(),
		Candidates: []Candidate{cand("S1", "A", ""), cand("S2", "B", "")}}

	rep := Preview(ctx, snap, cl, logging.NewMockLogger())

	require.Len(t, rep.Rows, 1, "the split decided before cancellation is kept")
	assert.Equal(t, "S1", rep.Rows[0].SplitID)
	assert.Equal(t, "s", rep.DBState)
	assert.Equal(t, []string{"A"}, cl.fullCalls, "no further splits are classified after cancellation")
}

func TestPreview_LogsSummary(t *testing.T) {
	log := logging.NewMockLogger()
	cl := &fakeClassifier{full: map[string]models.Category{"A": {Name: "Courses", Source: TierAI}}}
	Preview(context.Background(), Snapshot{Categories: testCategories(),
		Candidates: []Candidate{cand("S1", "A", ""), {SplitID: "S2", IsInvestment: true}}}, cl, log)

	assert.True(t, log.HasEntry("INFO", "Recategorization preview complete"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/icompta/ -run TestPreview -v`
Expected: FAIL, `undefined: Preview`.

- [ ] **Step 3: Write the implementation**

`internal/icompta/preview.go`:

```go
package icompta

import (
	"context"
	"fmt"
	"strings"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// progressEvery is how often preview logs progress. The AI tier is rate
// limited, so a first run can take a long time and the user needs a heartbeat.
const progressEvery = 100

// Classifier is the slice of the categorizer a preview needs.
type Classifier interface {
	// Full runs every tier, including the semantic and AI tiers.
	Full(ctx context.Context, tx categorizer.Transaction) (models.Category, error)
	// Local runs only the tiers that need no network.
	Local(ctx context.Context, tx categorizer.Transaction) (models.Category, bool)
}

type categorizerClassifier struct{ c *categorizer.Categorizer }

// NewCategorizerClassifier adapts the real categorizer. Full uses
// CategorizeTransaction, which never auto-learns.
func NewCategorizerClassifier(c *categorizer.Categorizer) Classifier {
	return categorizerClassifier{c: c}
}

func (k categorizerClassifier) Full(ctx context.Context, tx categorizer.Transaction) (models.Category, error) {
	return k.c.CategorizeTransaction(ctx, tx)
}

func (k categorizerClassifier) Local(ctx context.Context, tx categorizer.Transaction) (models.Category, bool) {
	return k.c.CategorizeLocal(ctx, tx)
}

// Preview classifies every in-scope split and returns the report. It never
// touches the database. Empty or unknown splits go through every tier; splits
// with a real category go through the local tiers only, since those are the
// only ones allowed to override a chosen category. Cancelling the context
// stops the run and returns the rows decided so far.
func Preview(ctx context.Context, snap Snapshot, cl Classifier, log logging.Logger) Report {
	rep := Report{DBState: snap.State}
	excluded := map[string]int{}
	decisions := map[string]int{}
	tiers := map[string]int{}
	processed := 0

	for _, c := range snap.Candidates {
		if ctx.Err() != nil {
			log.WithFields(
				logging.Field{Key: "processed", Value: processed},
				logging.Field{Key: "total", Value: len(snap.Candidates)},
			).Warn("Preview cancelled, keeping the rows decided so far")
			break
		}
		if why := Exclusion(c); why != "" {
			excluded[why]++
			continue
		}
		processed++
		if processed%progressEvery == 0 {
			log.WithFields(
				logging.Field{Key: "processed", Value: processed},
				logging.Field{Key: "total", Value: len(snap.Candidates)},
			).Info("Preview progress")
		}

		row, report := classify(ctx, c, snap.Categories, cl, log)
		if !report {
			continue
		}
		rep.Rows = append(rep.Rows, row)
		decisions[string(row.Decision)]++
		if row.Tier != "" {
			tiers[row.Tier]++
		}
	}

	log.WithFields(
		logging.Field{Key: "candidates", Value: len(snap.Candidates)},
		logging.Field{Key: "reported", Value: len(rep.Rows)},
		logging.Field{Key: "excluded", Value: fmt.Sprint(excluded)},
		logging.Field{Key: "decisions", Value: fmt.Sprint(decisions)},
		logging.Field{Key: "tiers", Value: fmt.Sprint(tiers)},
	).Info("Recategorization preview complete")
	return rep
}

// classify returns the report row for one split, and whether it belongs in the
// report at all (a real category left unchanged does not).
func classify(ctx context.Context, c Candidate, cats Categories, cl Classifier, log logging.Logger) (Row, bool) {
	tx := categorizer.Transaction{
		PartyName: c.PartyName(),
		IsDebtor:  c.IsDebtor(),
		Amount:    c.Amount.String(),
		Date:      c.Date,
		Info:      strings.TrimSpace(c.Name + " " + c.Comment),
	}
	row := Row{
		SplitID:     c.SplitID,
		Date:        c.Date,
		Name:        c.Name,
		Amount:      c.Amount.String(),
		OldCategory: c.CategoryName,
	}

	var proposal Proposal
	if IsUnknownCategory(c.CategoryName) {
		cat, err := cl.Full(ctx, tx)
		if err != nil {
			log.WithError(err).WithFields(logging.Field{Key: "split", Value: c.SplitID}).
				Warn("Categorizer failed for a split")
			row.Decision, row.Reason = ActionSkip, "categorizer error: "+err.Error()
			return row, true
		}
		proposal = Proposal{Category: cat.Name, Tier: cat.Source}
	} else {
		cat, found := cl.Local(ctx, tx)
		if !found {
			return Row{}, false
		}
		proposal = Proposal{Category: cat.Name, Tier: cat.Source}
	}

	d := Decide(c.CategoryName, proposal)
	if d.Reason == ReasonUnchanged {
		return Row{}, false
	}
	row.NewCategory = proposal.Category
	row.Tier = proposal.Tier
	row.Decision = d.Action
	row.Reason = d.Reason
	if d.Action != ActionChange {
		return row, true
	}
	if _, ok := cats.IDByName(proposal.Category); !ok {
		row.Decision, row.Reason = ActionSkip, "unknown category"
		return row, true
	}
	row.Apply = true
	return row, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/icompta/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/icompta/preview.go internal/icompta/preview_test.go
git commit -m "feat(icompta): add preview loop over a database snapshot

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Apply

**Files:**
- Create: `internal/icompta/apply.go`
- Test: `internal/icompta/apply_test.go`

**Interfaces:**
- Consumes: `Report`, `Row` (Task 3); `OpenReadOnly`, `Store.State`, `Categories`, `loadCategories`, `readState`, `dsn`, `newFixtureDB` (Task 4); `SameCategory`, `ActionChange` (Task 2); `logging.Logger`.
- Produces (used by Task 8):
  - `type ApplyOptions struct { DBPath string; Now func() time.Time; IsRunning func() (bool, error) }`
  - `type ApplyResult struct { Applied, Skipped int; BackupPath string }`
  - `func Apply(ctx context.Context, rep Report, opts ApplyOptions, log logging.Logger) (ApplyResult, error)`
  - `func ICComptaRunning() (bool, error)` — default `IsRunning`, runs `pgrep -ix iCompta`.

- [ ] **Step 1: Write the failing test**

`internal/icompta/apply_test.go`:

```go
package icompta

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2026, 10, 4, 21, 30, 5, 0, time.FixedZone("CEST", 2*3600))

func notRunning() (bool, error) { return false, nil }

func categoryOf(t *testing.T, path, splitID string) (cat, modified string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	var c, m sql.NullString
	require.NoError(t, db.QueryRow(
		`SELECT category, lastModificationDate FROM ICTransactionSplit WHERE ID=?`, splitID).Scan(&c, &m))
	return c.String, m.String
}

// previewState returns the state marker the fixture currently has.
func previewState(t *testing.T, path string) string {
	t.Helper()
	s, err := OpenReadOnly(context.Background(), path)
	require.NoError(t, err)
	defer s.Close()
	st, err := s.State(context.Background())
	require.NoError(t, err)
	return st
}

func opts(path string) ApplyOptions {
	return ApplyOptions{DBPath: path, Now: func() time.Time { return fixedNow }, IsRunning: notRunning}
}

func TestApply_WritesApprovedRowsOnly(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{
		DBState: previewState(t, path),
		Rows: []Row{
			{SplitID: "S2", OldCategory: "", NewCategory: "Alimentation", Decision: ActionChange, Apply: true},
			{SplitID: "S6", OldCategory: "Divers", NewCategory: "Alimentation", Decision: ActionChange, Apply: false},
			{SplitID: "S1", OldCategory: "Alimentation", NewCategory: "Divers", Decision: ActionKeep, Apply: true},
		},
	}
	log := logging.NewMockLogger()

	res, err := Apply(context.Background(), rep, opts(path), log)
	require.NoError(t, err)

	assert.Equal(t, 1, res.Applied)
	assert.Equal(t, 2, res.Skipped)

	cat, mod := categoryOf(t, path, "S2")
	assert.Equal(t, "C-ALI", cat)
	assert.Equal(t, "2026-10-04 19:30:05", mod, "lastModificationDate is written in UTC")

	cat, _ = categoryOf(t, path, "S6")
	assert.Equal(t, "C-DIV", cat, "apply=no must not change the split")
	cat, _ = categoryOf(t, path, "S1")
	assert.Equal(t, "C-ALI", cat, "a keep row must never be written, even with apply=yes")

	assert.True(t, log.HasEntry("INFO", "Recategorized split"))
	assert.True(t, log.HasEntry("WARN", "Skipped split"))
}

func TestApply_BacksUpFirst(t *testing.T) {
	_, path := openFixture(t)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	res, err := Apply(context.Background(), Report{DBState: previewState(t, path)}, opts(path), logging.NewMockLogger())
	require.NoError(t, err)

	assert.Equal(t, path+".bak-20261004T193005Z", res.BackupPath)
	backup, err := os.ReadFile(res.BackupPath)
	require.NoError(t, err)
	assert.Equal(t, before, backup)
}

func TestApply_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, path string, rep *Report, o *ApplyOptions)
		wantErr string
	}{
		{"iCompta running", func(_ *testing.T, _ string, _ *Report, o *ApplyOptions) {
			o.IsRunning = func() (bool, error) { return true, nil }
		}, "iCompta is running"},
		{"running check fails", func(_ *testing.T, _ string, _ *Report, o *ApplyOptions) {
			o.IsRunning = func() (bool, error) { return false, errors.New("pgrep exploded") }
		}, "pgrep exploded"},
		{"database changed since preview", func(t *testing.T, path string, _ *Report, _ *ApplyOptions) {
			db, err := sql.Open("sqlite", path)
			require.NoError(t, err)
			defer db.Close()
			_, err = db.Exec(`UPDATE ICTransactionSplit SET lastModificationDate='2026-10-04 20:00:00' WHERE ID='S1'`)
			require.NoError(t, err)
		}, "changed since the preview"},
		{"wal file present", func(t *testing.T, path string, _ *Report, _ *ApplyOptions) {
			require.NoError(t, os.WriteFile(path+"-wal", []byte("x"), 0o600))
		}, "-wal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, path := openFixture(t)
			rep := Report{DBState: previewState(t, path), Rows: []Row{
				{SplitID: "S2", NewCategory: "Alimentation", Decision: ActionChange, Apply: true}}}
			o := opts(path)
			tt.mutate(t, path, &rep, &o)

			_, err := Apply(context.Background(), rep, o, logging.NewMockLogger())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)

			cat, _ := categoryOf(t, path, "S2")
			assert.Empty(t, cat, "a refused apply must write nothing")
			matches, _ := filepath.Glob(path + ".bak-*")
			assert.Empty(t, matches, "a refused apply must not leave a backup behind")
		})
	}
}

func TestApply_SkipsStaleAndUnknownRows(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{DBState: previewState(t, path), Rows: []Row{
		// S1 is Alimentation now, the report believes it was Divers.
		{SplitID: "S1", OldCategory: "Divers", NewCategory: "Alimentation", Decision: ActionChange, Apply: true},
		// Category that does not exist.
		{SplitID: "S2", OldCategory: "", NewCategory: "Brand New", Decision: ActionChange, Apply: true},
		// Split that does not exist.
		{SplitID: "NOPE", OldCategory: "", NewCategory: "Alimentation", Decision: ActionChange, Apply: true},
	}}
	log := logging.NewMockLogger()

	res, err := Apply(context.Background(), rep, opts(path), log)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Applied)
	assert.Equal(t, 3, res.Skipped)
	assert.Len(t, log.GetEntriesByLevel("WARN"), 3)
}

// Review focus 4: running the same report twice is harmless.
func TestApply_SecondRunChangesNothing(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{DBState: previewState(t, path), Rows: []Row{
		{SplitID: "S2", OldCategory: "", NewCategory: "Alimentation", Decision: ActionChange, Apply: true}}}

	first, err := Apply(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, first.Applied)

	// The first run changed the data, so the state marker moved: refuse.
	_, err = Apply(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "changed since the preview")

	// Even with a refreshed state, the stale old_category keeps it a no-op.
	rep.DBState = previewState(t, path)
	log := logging.NewMockLogger()
	second, err := Apply(context.Background(), rep, opts(path), log)
	require.NoError(t, err)
	assert.Equal(t, 0, second.Applied)
	assert.True(t, log.HasEntry("WARN", "Skipped split"))
	cat, _ := categoryOf(t, path, "S2")
	assert.Equal(t, "C-ALI", cat)
}

func TestApply_ResolvesDecomposedCategoryName(t *testing.T) {
	_, path := openFixture(t)
	rep := Report{DBState: previewState(t, path), Rows: []Row{
		// The database stores "Non Classe" + combining accent; the report carries
		// the composed form. Both must resolve to the same category.
		{SplitID: "S6", OldCategory: "Divers", NewCategory: "Non Classé", Decision: ActionChange, Apply: true}}}

	res, err := Apply(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Applied)
	cat, _ := categoryOf(t, path, "S6")
	assert.Equal(t, "C-NC", cat)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/icompta/ -run TestApply -v`
Expected: FAIL, `undefined: Apply`.

- [ ] **Step 3: Write the implementation**

`internal/icompta/apply.go`:

```go
package icompta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"fjacquet/camt-csv/internal/logging"
)

// iComptaTimeFormat is how iCompta writes lastModificationDate (UTC).
const iComptaTimeFormat = "2006-01-02 15:04:05"

// ApplyOptions are the collaborators of Apply, injectable for tests.
type ApplyOptions struct {
	DBPath    string
	Now       func() time.Time
	IsRunning func() (bool, error)
}

// ApplyResult summarises a run.
type ApplyResult struct {
	Applied    int
	Skipped    int
	BackupPath string
}

// ICComptaRunning reports whether the iCompta app is running, by process name.
func ICComptaRunning() (bool, error) {
	err := exec.Command("pgrep", "-ix", "iCompta").Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check whether iCompta is running: %w", err)
}

// Apply writes the approved rows of a report into the database. It aborts
// before writing anything if iCompta is running, if the data changed since the
// preview, or if the database cannot be backed up consistently.
func Apply(ctx context.Context, rep Report, opts ApplyOptions, log logging.Logger) (ApplyResult, error) {
	running, err := opts.IsRunning()
	if err != nil {
		return ApplyResult{}, err
	}
	if running {
		return ApplyResult{}, errors.New("iCompta is running: quit it before applying, it would overwrite or lock the database")
	}
	if fi, err := os.Stat(opts.DBPath + "-wal"); err == nil && fi.Size() > 0 {
		return ApplyResult{}, fmt.Errorf("%s-wal is not empty: the database has uncommitted pages, a file copy would not be a consistent backup", opts.DBPath)
	}

	ro, err := OpenReadOnly(ctx, opts.DBPath)
	if err != nil {
		return ApplyResult{}, err
	}
	state, err := ro.State(ctx)
	_ = ro.Close()
	if err != nil {
		return ApplyResult{}, err
	}
	if state != rep.DBState {
		return ApplyResult{}, fmt.Errorf("the database changed since the preview (preview saw %q, now %q): run preview again", rep.DBState, state)
	}

	backup := fmt.Sprintf("%s.bak-%s", opts.DBPath, opts.Now().UTC().Format("20060102T150405Z"))
	if err := copyVerified(opts.DBPath, backup); err != nil {
		return ApplyResult{}, err
	}
	log.WithFields(logging.Field{Key: "backup", Value: backup}).Info("Database backed up")

	res := ApplyResult{BackupPath: backup}
	// _timeout is the validated shorthand for PRAGMA busy_timeout; _txlock=immediate
	// takes the write lock at BEGIN instead of failing late on the first UPDATE.
	uri, err := dsn(opts.DBPath, "_timeout=5000&_txlock=immediate")
	if err != nil {
		return res, err
	}
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return res, fmt.Errorf("open database for writing: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if err := integrityCheck(ctx, db); err != nil {
		return res, fmt.Errorf("before apply: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("begin write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cats, err := loadCategories(ctx, tx)
	if err != nil {
		return res, err
	}
	stamp := opts.Now().UTC().Format(iComptaTimeFormat)

	for _, row := range rep.Rows {
		skip := func(reason string) {
			res.Skipped++
			log.WithFields(
				logging.Field{Key: "split", Value: row.SplitID},
				logging.Field{Key: "name", Value: row.Name},
				logging.Field{Key: "reason", Value: reason},
			).Warn("Skipped split")
		}
		if row.Decision != ActionChange || !row.Apply {
			skip("not approved: decision " + string(row.Decision) + ", apply " + fmt.Sprint(row.Apply))
			continue
		}
		newID, ok := cats.IDByName(row.NewCategory)
		if !ok {
			skip("unknown category " + row.NewCategory)
			continue
		}
		var currentID sql.NullString
		err := tx.QueryRowContext(ctx,
			`SELECT category FROM ICTransactionSplit WHERE ID = ?`, row.SplitID).Scan(&currentID)
		if errors.Is(err, sql.ErrNoRows) {
			skip("split not found")
			continue
		}
		if err != nil {
			return res, fmt.Errorf("read split %s: %w", row.SplitID, err)
		}
		if !SameCategory(cats.NameByID(currentID.String), row.OldCategory) {
			skip("category changed since preview")
			continue
		}
		r, err := tx.ExecContext(ctx,
			`UPDATE ICTransactionSplit SET category = ?, lastModificationDate = ? WHERE ID = ?`,
			newID, stamp, row.SplitID)
		if err != nil {
			return res, fmt.Errorf("update split %s: %w", row.SplitID, err)
		}
		if n, _ := r.RowsAffected(); n != 1 {
			return res, fmt.Errorf("update split %s: %d rows affected, expected 1", row.SplitID, n)
		}
		res.Applied++
		log.WithFields(
			logging.Field{Key: "split", Value: row.SplitID},
			logging.Field{Key: "name", Value: row.Name},
			logging.Field{Key: "old", Value: row.OldCategory},
			logging.Field{Key: "new", Value: row.NewCategory},
			logging.Field{Key: "tier", Value: row.Tier},
		).Info("Recategorized split")
	}

	if err := integrityCheck(ctx, tx); err != nil {
		return res, fmt.Errorf("after apply, rolled back: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return res, fmt.Errorf("commit: %w", err)
	}
	log.WithFields(
		logging.Field{Key: "applied", Value: res.Applied},
		logging.Field{Key: "skipped", Value: res.Skipped},
		logging.Field{Key: "backup", Value: backup},
	).Info("Recategorization applied")
	return res, nil
}

type integrityQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func integrityCheck(ctx context.Context, q integrityQuerier) error {
	var result string
	if err := q.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result); err != nil {
		return fmt.Errorf("integrity_check: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("integrity_check reported %q", result)
	}
	return nil
}

// copyVerified copies src to dst and checks the copy byte for byte by hash. On
// any failure the partial copy is removed, so a refused run leaves no backup.
func copyVerified(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	defer func() {
		_ = out.Close()
		if err != nil {
			_ = os.Remove(dst)
		}
	}()
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(out, h), in); err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	if err = out.Sync(); err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	copied, err := os.ReadFile(dst)
	if err != nil {
		return fmt.Errorf("verify backup: %w", err)
	}
	sum := sha256.Sum256(copied)
	if !bytes.Equal(sum[:], h.Sum(nil)) {
		err = errors.New("verify backup: copy does not match the source")
		return err
	}
	return nil
}
```

Note for the implementer: `Apply` checks refusals (running, WAL, state) before the backup, so a refused run leaves no backup file, which `TestApply_Refusals` asserts.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./internal/icompta/ -v`
Expected: PASS. If `TestApply_BacksUpFirst` fails on the backup name, the format is `<db>.bak-YYYYMMDDTHHMMSSZ` in UTC (`19:30:05` for `fixedNow`).

- [ ] **Step 5: Commit**

```bash
git add internal/icompta/apply.go internal/icompta/apply_test.go
git commit -m "feat(icompta): apply a reviewed report with backup and guards

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 8: The `recategorize` command

**Files:**
- Create: `cmd/recategorize/recategorize.go`
- Test: `cmd/recategorize/recategorize_test.go`
- Modify: `main.go:40-41` (register the command)

**Interfaces:**
- Consumes: `icompta.OpenReadOnly`, `Store.Snapshot`, `icompta.Preview`, `icompta.NewCategorizerClassifier`, `icompta.Report.Write`, `icompta.ReadReport`, `icompta.Apply`, `icompta.ApplyOptions`, `icompta.ICComptaRunning` (Tasks 3 to 7); `root.GetContainer()`, `root.Log`, `root.GetLogrusAdapter()` (existing, see `cmd/categorize/categorize.go`); `Container.GetCategorizer()`.
- Produces: `var Cmd *cobra.Command` (`recategorize`) with subcommands `preview` and `apply`.

- [ ] **Step 1: Write the failing test**

`cmd/recategorize/recategorize_test.go`:

```go
package recategorize_test

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/cmd/recategorize"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sub(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, c := range recategorize.Cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func TestRecategorizeCommand_Metadata(t *testing.T) {
	assert.Equal(t, "recategorize", recategorize.Cmd.Use)
	assert.Equal(t, "tools", recategorize.Cmd.GroupID)
	assert.NotNil(t, sub(t, "preview"))
	assert.NotNil(t, sub(t, "apply"))
}

func TestPreviewCommand_FlagsAndArgs(t *testing.T) {
	preview := sub(t, "preview")
	require.NotNil(t, preview.Flags().Lookup("db"))
	require.NotNil(t, preview.Flags().Lookup("output"))
	assert.Equal(t, "o", preview.Flags().Lookup("output").Shorthand)
	assert.Error(t, preview.Args(preview, []string{"stray"}), "positional arguments are typos")
}

func TestApplyCommand_FlagsAndArgs(t *testing.T) {
	apply := sub(t, "apply")
	require.NotNil(t, apply.Flags().Lookup("db"))
	require.NotNil(t, apply.Flags().Lookup("report"))
	assert.Error(t, apply.Args(apply, []string{"stray"}))
}

// A missing or malformed report must fail before anything touches the database.
func TestApplyCommand_RejectsBadReport(t *testing.T) {
	dir := t.TempDir()
	notAReport := filepath.Join(dir, "r.csv")
	require.NoError(t, os.WriteFile(notAReport, []byte("a,b\n"), 0o600))

	apply := sub(t, "apply")
	assert.Error(t, apply.RunE(apply, nil), "missing flags")

	require.NoError(t, apply.Flags().Set("db", filepath.Join(dir, "x.cdb")))
	require.NoError(t, apply.Flags().Set("report", notAReport))
	err := apply.RunE(apply, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/recategorize/ -v`
Expected: FAIL, package `recategorize` does not exist.

- [ ] **Step 3: Write the implementation**

`cmd/recategorize/recategorize.go`:

```go
// Package recategorize reclassifies transactions already stored in an iCompta
// database, with a reviewable preview before anything is written.
package recategorize

import (
	"fmt"
	"os"
	"time"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/icompta"

	"github.com/spf13/cobra"
)

// Each subcommand owns its flag variables: sharing one --db variable between
// preview and apply would let one command's value leak into the other.
var (
	previewDB  string
	outputPath string
	applyDB    string
	reportPath string
)

// Cmd is the recategorize command.
var Cmd = &cobra.Command{
	Use:     "recategorize",
	Short:   "Recategorize transactions stored in an iCompta database",
	GroupID: "tools",
	Long: `Run the categorizer over an iCompta database in two steps.

  preview  reads the database (read-only) and writes a CSV report of proposed changes.
  apply    writes the rows you approved in that report into the database.

A category you chose is replaced only by a direct mapping or keyword match. The
semantic and AI tiers only fill empty or "unknown" categories (Divers, Non Classé,
Autre, Uncategorized). Investment splits and linked transfers are left alone.

apply refuses to run while iCompta is open, backs up the database first, and
logs every change. If you use iCloud sync, check another device after the first
apply: whether a change written outside iCompta propagates is not guaranteed.`,
}

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Write a report of proposed category changes (read-only)",
	Args:  cobra.NoArgs,
	RunE:  runPreview,
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the approved rows of a preview report to the database",
	Args:  cobra.NoArgs,
	RunE:  runApply,
}

func init() {
	previewCmd.Flags().StringVar(&previewDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	previewCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Path of the report CSV to write")
	_ = previewCmd.MarkFlagRequired("db")
	_ = previewCmd.MarkFlagRequired("output")

	applyCmd.Flags().StringVar(&applyDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	applyCmd.Flags().StringVar(&reportPath, "report", "", "Path of the reviewed report CSV")
	_ = applyCmd.MarkFlagRequired("db")
	_ = applyCmd.MarkFlagRequired("report")

	Cmd.AddCommand(previewCmd, applyCmd)
}

func runPreview(cmd *cobra.Command, _ []string) error {
	// MarkFlagRequired is enforced by Cobra's execute(), not when RunE is called
	// directly (as the tests do), so the command checks again.
	if previewDB == "" || outputPath == "" {
		return fmt.Errorf("--db and --output are required")
	}
	ctx := cmd.Context()
	log := root.GetLogrusAdapter()

	appContainer := root.GetContainer()
	if appContainer == nil {
		return fmt.Errorf("container not initialized")
	}

	store, err := icompta.OpenReadOnly(ctx, previewDB)
	if err != nil {
		return err
	}
	defer store.Close()
	snap, err := store.Snapshot(ctx)
	if err != nil {
		return err
	}

	rep := icompta.Preview(ctx, snap, icompta.NewCategorizerClassifier(appContainer.GetCategorizer()), log)

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	if err := rep.Write(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("write report: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	log.Infof("Report written to %s: review it, set apply to no on rows you reject, then run apply", outputPath)
	return nil
}

func runApply(cmd *cobra.Command, _ []string) error {
	if applyDB == "" || reportPath == "" {
		return fmt.Errorf("--db and --report are required")
	}
	f, err := os.Open(reportPath)
	if err != nil {
		return fmt.Errorf("open report: %w", err)
	}
	rep, err := icompta.ReadReport(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("report %s: %w", reportPath, err)
	}

	_, err = icompta.Apply(cmd.Context(), rep, icompta.ApplyOptions{
		DBPath:    applyDB,
		Now:       time.Now,
		IsRunning: icompta.ICComptaRunning,
	}, root.GetLogrusAdapter())
	return err
}
```

If `root.GetLogrusAdapter()` returns a type that is not a `logging.Logger` with an `Infof`, use `root.Log` (a `logging.Logger`) and `log.Info("Report written", logging.Field{Key: "path", Value: outputPath})` instead; `cmd/categorize/categorize.go` shows both `root.Log` and `root.GetLogrusAdapter()` in use, check which one satisfies `icompta.Preview(…, log logging.Logger)` and use that for all three calls.

In `main.go`, add the import and register the command after `categorize.Cmd`:

```go
	root.Cmd.AddCommand(convert.Cmd)
	root.Cmd.AddCommand(categorize.Cmd)
	root.Cmd.AddCommand(recategorize.Cmd)
```

with `"fjacquet/camt-csv/cmd/recategorize"` added to the import block.

- [ ] **Step 4: Run tests and build**

Run: `go build ./... && go test -race ./cmd/... ./internal/icompta/ -v`
Expected: build OK, PASS. Then `go run . recategorize --help` prints both subcommands under "Tools:".

- [ ] **Step 5: Commit**

```bash
git add cmd/recategorize main.go
git commit -m "feat: add recategorize preview and apply commands

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Documentation, full verification and a read-only smoke run

**Files:**
- Create: `docs/icompta-recategorize.md`
- Modify: `CHANGELOG.md`, `CLAUDE.md`

**Interfaces:**
- Consumes: the command from Task 8.
- Produces: user guide, changelog entry, project-instruction entry.

- [ ] **Step 1: Write the user guide**

`docs/icompta-recategorize.md`:

````markdown
# Recategorize an iCompta database

`camt-csv recategorize` runs the categorizer over transactions already stored in
iCompta, in two steps so you can review before anything is written.

## 1. Preview (read-only)

```bash
camt-csv recategorize preview --db ~/Desktop/ic25.cdb -o report.csv
```

iCompta may stay open for this step; the database is opened read-only. The AI
tier is limited to about 5 requests a minute, so a first run can take tens of
minutes. Progress is logged every 100 splits. Ctrl-C keeps the rows decided so
far.

The report has one row per proposed change. Columns:
`split_id,date,name,amount,old_category,new_category,tier,decision,reason,apply`.
Set `apply` to `no` on any row you reject, or delete the row. Only rows with
`decision=change` and `apply=yes` are written.

## 2. Apply

Quit iCompta, then:

```bash
camt-csv recategorize apply --db ~/Desktop/ic25.cdb --report report.csv
```

Before writing, apply checks iCompta is not running and that the database has
not changed since the preview, then saves `ic25.cdb.bak-<UTC timestamp>` next to
it. Every change is logged with the old and new category and the tier that
decided it. A row whose current category no longer matches `old_category` is
skipped, so a report cannot overwrite a change you made afterwards.

## Rules

- A category you chose is replaced only by a **direct mapping** or **keyword**
  match. **Semantic** and **AI** results only fill empty or unknown categories
  (`Divers`, `Non Classé`, `Autre`, `Uncategorized`, `Uncategorized (AI)`).
- Investment splits and linked transfers are never touched.
- A proposed category that does not exist in iCompta is reported as
  `skipped: unknown category`, never created.

## iCloud and other backups

Changes are written outside iCompta. If iCloud sync is on, open iCompta on a
second device after the first apply and confirm the changes arrived. Your own
backups (Acronis, iCloud) plus the file written by apply each allow a restore.
````

- [ ] **Step 2: Update CHANGELOG and CLAUDE.md**

In `CHANGELOG.md` under `## [Unreleased]`, add (create `### Added` above the existing `### Fixed` if absent):

```markdown
### Added

- Add `camt-csv recategorize preview|apply` to recategorize transactions stored in an
  iCompta database: a read-only preview writes a reviewable CSV report, and apply
  writes the approved rows with a backup and a log line per change.
- Add `Categorizer.CategorizeLocal`, which runs only the network-free tiers.
- Add the `modernc.org/sqlite` dependency (pure Go, no CGO).
```

In `CLAUDE.md`, in the **Directory Structure** list, add:

```markdown
  - `icompta/` - Recategorize an iCompta database (`ic25.cdb`): read-only `store`, pure `policy`/`report`, `preview` loop, `apply` with backup. Command: `cmd/recategorize`. Real-category splits run only local tiers (`Categorizer.CategorizeLocal`). Tests build fixture databases, never the real file. See `docs/icompta-recategorize.md`.
```

- [ ] **Step 3: Run the whole verification suite**

Run: `make lint && make test && make security`
Expected: all pass, no new lint or gosec findings. gosec may flag `exec.Command("pgrep", …)`; its arguments are constants, so annotate with `// #nosec G204 -- fixed arguments` only if it is actually reported.

- [ ] **Step 4: Smoke-run preview on a copy of the real database (read-only)**

Never point the first run at the real file. Use a copy, and do not run `apply`.

```bash
cp ~/Desktop/ic25.cdb "$TMPDIR/ic25-smoke.cdb"
go run . recategorize preview --db "$TMPDIR/ic25-smoke.cdb" -o "$TMPDIR/report.csv" --log-level info
```

Expected: logs `Preview progress`, then `Recategorization preview complete` with `excluded` showing ~444 investment and ~40 linked splits, `decisions` counts, and a `tiers` histogram. The report opens in a spreadsheet and round-trips (`go run . recategorize apply --db <copy> --report <report>` on the **copy** with iCompta closed should apply, log `Recategorized split` per row, and leave a `.bak-` file beside the copy). Verify on the copy:

```bash
sqlite3 -readonly "$TMPDIR/ic25-smoke.cdb" "select count(*) from ICTransactionSplit where category is null or category='';"
```

Expected: lower than before apply, never zero (investments and transfers stay empty).

- [ ] **Step 5: Commit, push, open the PR**

```bash
git add docs/icompta-recategorize.md CHANGELOG.md CLAUDE.md
git commit -m "docs: document recategorize preview and apply

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
git push -u origin feat/icompta-recategorize
gh pr create --title "feat: recategorize an iCompta database" --body "$(cat <<'EOF'
## Summary
- `camt-csv recategorize preview` reads ic25.cdb read-only and writes a reviewable CSV report.
- `camt-csv recategorize apply` writes the approved rows after a backup, with a log line per change.
- A chosen category is replaced only by direct mapping or keyword; semantic and AI only fill empty/unknown ones.

## Test plan
- [x] Unit tests on fixture SQLite databases (policy, report round trip, store, preview, apply guards)
- [x] `make lint test security`
- [x] Read-only preview smoke run on a copy of the real database

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Self-Review

**Spec coverage**

| Spec requirement | Task |
|---|---|
| `preview` read-only, `apply` from the reviewed report | 4, 6, 7, 8 |
| Selector excludes investment and linked splits | 2 (`Exclusion`), 6 (loop) |
| Policy table | 2 (`Decide`) |
| Candidate to transaction mapping (payee or name, `IsDebtor` by sign, amount/date/info) | 2, 6 |
| Unknown category never created | 6 (`unknown category` skip), 7 |
| Report columns, hand-editable `apply` | 3 (+ `reason`, Task 1 amendment 3) |
| Guards: iCompta running, state mismatch, backup, integrity before/after, one transaction, per-row `old_category`, UTC `lastModificationDate` | 7 |
| Logging per change, per skip, summary | 6, 7 |
| `modernc.org/sqlite`, API checked | 4 (step 1) |
| Schema drift check | 4 (`requireColumns`) |
| AI rate limit: dedupe and progress | 6 (batch cache via `CategorizeTransaction`; `Preview progress`), amendment 1 |
| No auto-learn | 5, 6 (`CategorizeTransaction`, `CategorizeLocal`) |
| Fixture-database tests, real DB never used | 4, 7, 9 |
| iCloud caveat documented | 8 (help text), 9 (guide) |

**Placeholder scan:** no TBD/TODO. Two conditional notes (the logger accessor in Task 8, the gosec annotation in Task 9) name the exact check and the exact fallback.

**Type consistency:** `Candidate`, `Proposal`, `Decision`, `Action`, `Row`, `Report`, `Snapshot`, `Categories`, `Classifier`, `ApplyOptions` and `ApplyResult` are each defined once and used with the same field names downstream. `readState`, `loadCategories`, `dsn` and `newFixtureDB` are defined in Task 4 and reused in Tasks 7 and 8 under the same names. `ReasonUnchanged` and `ReasonNoSuggestion` are defined in Task 2 and used in Tasks 3, 6.

**Review Focus coverage:** items 1 and 2 pinned in Task 4, 2 also in Task 2, 3 in Task 3, 4 in Task 7, 5 in Task 6.
