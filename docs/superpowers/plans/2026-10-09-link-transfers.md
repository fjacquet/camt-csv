# link-transfers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `camt-csv link-transfers preview|apply`. It finds unlinked transfers between the user's own iCompta accounts, writes a reviewable CSV, and links the approved pairs the way iCompta does.

**Architecture:** New transfer-specific files in `internal/icompta`, next to the existing recategorize code:
- scope resolution (pure);
- matching (pure);
- a store read;
- a report;
- an apply step.

A new `cmd/linktransfers` command is registered in `main.go`. The pre-write safety sequence of `Apply` is extracted into one helper that both applies share.

**Tech Stack:** Go 1.27, cobra, `modernc.org/sqlite`, `shopspring/decimal`, testify.

**Spec:** `docs/superpowers/specs/2026-10-09-link-transfers-design.md`

## Global Constraints

- Amounts are `shopspring/decimal`, never float or int. Compare amounts rounded to cents (`Round(2)`).
- Tests build fixture databases in `t.TempDir()`. They never open `~/Desktop/ic25.cdb`.
- `recategorize` behaviour must not change. Its existing tests must pass untouched after every task.
- Report text columns pass through `csvsafe.Escape` on write and `csvsafe.Unescape` on read.
- The report file mode is `0o600`. Never overwrite an existing report without `--force`.
- The magic line is `# camt-csv link-transfers report v1`. The state key is `# db_state=`, shared with recategorize.
- `max_days` = 4 and `sure_days` = 2 are constants, not settings.
- Scope comes only from `--folder` (required, repeatable): every `ICAccount` row with `class='ICAccount'` under those folders, at any depth. There is no config key and no exclude list.
- Apply writes only:
  - `ICTransactionSplit.linkedSplit` (both directions);
  - `lastModificationDate` on both splits and both parent transactions (UTC, `2006-01-02 15:04:05`).

  Categories and amounts are never written.
- Never use `math/rand`. Run `make lint` (golangci-lint) clean.
- Update `CHANGELOG.md` under `## [Unreleased]`, in imperative mood.
- Root builds the categorizer for every command (container). `link-transfers` must never call it, so no embedding warm-up or AI request happens.

## Review Focus

1. **A `--folder` name typed in a different case** (`fred` instead of `Fred`). Expected: an error that lists the available folder names, not an empty scope. Test in Task 1.
2. **A report applied a second time.** Expected: refused with "run preview again", since the first apply changed the data state. Nothing is written twice. Test in Task 5.
3. **A transaction date stored with a time** (`2026-01-10 00:00:00`). Expected: it matches like a plain date. Test in Task 3.
4. **The same debit imported twice** (two identical debits, one credit). Expected: both pairs listed as doubtful ("2 candidates for the credit"), neither pre-approved. Test in Task 2.
5. **A report saved by Numbers/Excel** (BOM, CRLF, `Yes`). Expected: read without error. Test in Task 4.

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/icompta/transfers_scope.go` (new) | `Account`, `ResolveScope`: folder names to account IDs |
| `internal/icompta/transfers.go` (new) | `Leg`, `Pair`, `EligibleLegs`, `MatchTransfers`: pure matching |
| `internal/icompta/transfers_store.go` (new) | `TransferSnapshot`: read accounts and legs; transfer column check |
| `internal/icompta/store.go` (modify) | `requireColumns` takes the column map as a parameter |
| `internal/icompta/transfers_report.go` (new) | `TransferReport`, write and read |
| `internal/icompta/report.go` (modify) | extract the comment-header reader, shared by both reports |
| `internal/icompta/apply.go` (modify) | extract `openForWrite` (pre-checks, backup, write connection) |
| `internal/icompta/transfers_apply.go` (new) | `ApplyLinks` |
| `cmd/linktransfers/linktransfers.go` (new) | cobra command `link-transfers preview/apply` |
| `main.go` (modify) | register the command |
| `docs/icompta-link-transfers.md` (new), `CLAUDE.md`, `CHANGELOG.md` | documentation |

Every new `internal/icompta` file has a `_test.go` sibling in `package icompta`.

---

### Task 1: Scope resolution

**Files:**
- Create: `internal/icompta/transfers_scope.go`
- Test: `internal/icompta/transfers_scope_test.go`

**Interfaces:**
- Consumes: `normalize(string) string` from `internal/icompta/policy.go` (NFC + trim).
- Produces:
  - `type Account struct { ID, Name, Class, ParentID, CurrencyID string }`
  - constants `classAccount = "ICAccount"` and `classFolder = "ICAccountsGroup"`
  - `func ResolveScope(accounts []Account, folders []string) (map[string]bool, error)`: the set of account IDs in scope.

- [ ] **Step 1: Write the failing tests**

```go
package icompta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopeAccounts() []Account {
	return []Account{
		{ID: "F", Name: "Fred", Class: classFolder},
		{ID: "FR", Name: "France", Class: classFolder, ParentID: "F"},
		{ID: "CA", Name: "CA-Fred", Class: classFolder, ParentID: "FR"},
		{ID: "A1", Name: "Compte cheque", Class: classAccount, ParentID: "CA"}, // two levels under Fred
		{ID: "A2", Name: "Revolut CHF", Class: classAccount, ParentID: "F"},
		{ID: "L", Name: "Lydie", Class: classFolder},
		{ID: "A3", Name: "Livret A", Class: classAccount, ParentID: "L"},
		{ID: "P", Name: "Florence", Class: "ICPerson", ParentID: "F"},
		{ID: "A4", Name: "Orphan", Class: classAccount},
	}
}

func TestResolveScope_NestedAccountsOfOneFolder(t *testing.T) {
	scope, err := ResolveScope(scopeAccounts(), []string{"Fred"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"A1": true, "A2": true}, scope,
		"accounts at any depth under Fred; never folders, persons or other folders' accounts")
}

func TestResolveScope_SeveralFolders(t *testing.T) {
	scope, err := ResolveScope(scopeAccounts(), []string{"Fred", "Lydie"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"A1": true, "A2": true, "A3": true}, scope)
}

// Review focus 1: a wrong-case name must fail loudly and say what exists.
func TestResolveScope_UnknownFolderListsAvailable(t *testing.T) {
	_, err := ResolveScope(scopeAccounts(), []string{"fred"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `folder "fred" not found`)
	assert.Contains(t, err.Error(), "CA-Fred, France, Fred, Lydie")
}

func TestResolveScope_DuplicateFolderNameNamesBothPaths(t *testing.T) {
	accts := append(scopeAccounts(), Account{ID: "F2", Name: "Fred", Class: classFolder, ParentID: "L"})
	_, err := ResolveScope(accts, []string{"Fred"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Fred, Lydie / Fred")
}

func TestResolveScope_NoFolderIsAnError(t *testing.T) {
	_, err := ResolveScope(scopeAccounts(), nil)
	require.Error(t, err)
}

func TestResolveScope_ParentCycleTerminates(t *testing.T) {
	accts := []Account{
		{ID: "X", Name: "X", Class: classFolder, ParentID: "Y"},
		{ID: "Y", Name: "Y", Class: classFolder, ParentID: "X"},
		{ID: "A", Name: "A", Class: classAccount, ParentID: "X"},
	}
	scope, err := ResolveScope(accts, []string{"Y"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"A": true}, scope)
}
```

- [ ] **Step 2: Run the tests and check they fail**

Run: `go test ./internal/icompta/ -run TestResolveScope -v`
Expected: FAIL to compile, with "undefined: Account".

- [ ] **Step 3: Implement**

```go
package icompta

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ICAccount.class values: a real account, a folder ("accounts group"). Persons
// (ICPerson) and anything else are never in scope.
const (
	classAccount = "ICAccount"
	classFolder  = "ICAccountsGroup"
)

// Account is one row of ICAccount.
type Account struct {
	ID, Name, Class, ParentID, CurrencyID string
}

// ResolveScope returns the IDs of the real accounts under the named folders, at
// any depth. A name that matches no folder, or several, is an error: a silent
// wrong scope would link transfers to accounts the user does not manage.
func ResolveScope(accounts []Account, folders []string) (map[string]bool, error) {
	if len(folders) == 0 {
		return nil, errors.New("at least one --folder is required")
	}
	byID := make(map[string]Account, len(accounts))
	for _, a := range accounts {
		byID[a.ID] = a
	}
	chosen := map[string]bool{}
	for _, f := range folders {
		var hits []Account
		for _, a := range accounts {
			if a.Class == classFolder && normalize(a.Name) == normalize(f) {
				hits = append(hits, a)
			}
		}
		switch len(hits) {
		case 0:
			return nil, fmt.Errorf("folder %q not found; folders are: %s", f, strings.Join(folderNames(accounts), ", "))
		case 1:
			chosen[hits[0].ID] = true
		default:
			paths := make([]string, len(hits))
			for i, h := range hits {
				paths[i] = folderPath(byID, h)
			}
			sort.Strings(paths)
			return nil, fmt.Errorf("folder %q is ambiguous: %s", f, strings.Join(paths, ", "))
		}
	}
	scope := map[string]bool{}
	for _, a := range accounts {
		if a.Class == classAccount && underAny(byID, a, chosen) {
			scope[a.ID] = true
		}
	}
	return scope, nil
}

// underAny walks a's parent chain; seen guards against a corrupt cycle.
func underAny(byID map[string]Account, a Account, chosen map[string]bool) bool {
	seen := map[string]bool{}
	for p := a.ParentID; p != "" && !seen[p]; p = byID[p].ParentID {
		if chosen[p] {
			return true
		}
		seen[p] = true
	}
	return false
}

func folderPath(byID map[string]Account, a Account) string {
	parts := []string{a.Name}
	seen := map[string]bool{a.ID: true}
	for p := a.ParentID; p != "" && !seen[p]; p = byID[p].ParentID {
		seen[p] = true
		parts = append([]string{byID[p].Name}, parts...)
	}
	return strings.Join(parts, " / ")
}

func folderNames(accounts []Account) []string {
	var names []string
	for _, a := range accounts {
		if a.Class == classFolder {
			names = append(names, a.Name)
		}
	}
	sort.Strings(names)
	return names
}
```

- [ ] **Step 4: Run the tests and check they pass**

Run: `go test ./internal/icompta/ -run TestResolveScope -v`
Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/icompta/transfers_scope.go internal/icompta/transfers_scope_test.go
git commit -m "feat(icompta): resolve link-transfers scope from folder names"
```

---

### Task 2: Pair matching

**Files:**
- Create: `internal/icompta/transfers.go`
- Test: `internal/icompta/transfers_test.go`

**Interfaces:**
- Consumes: `normalize` (policy.go).
- Produces:
  - `type Leg struct { SplitID, TxID, AccountID, AccountName, CurrencyID, Date, Name string; Amount decimal.Decimal; Category, Status string; SplitCount int; Linked bool }`. `Date` is `YYYY-MM-DD`.
  - `type Pair struct { Debit, Credit Leg; GapDays int; Sure bool; Reason, Note string }`
  - constants `maxGapDays = 4` and `sureGapDays = 2`
  - `func EligibleLegs(legs []Leg, scope map[string]bool, from, to string) []Leg`
  - `func MatchTransfers(legs []Leg) []Pair`

- [ ] **Step 1: Write the failing tests**

```go
package icompta

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func leg(id, acct, date, amount string) Leg {
	return Leg{SplitID: id, TxID: "T" + id, AccountID: acct, AccountName: "acct " + acct, CurrencyID: "CHF",
		Date: date, Name: "tx " + id, Amount: decimal.RequireFromString(amount), Category: "Virements", SplitCount: 1}
}

func TestMatchTransfers_SurePair(t *testing.T) {
	pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-750"), leg("C", "B", "2026-01-11", "750")})
	require.Len(t, pairs, 1)
	p := pairs[0]
	assert.Equal(t, "D", p.Debit.SplitID)
	assert.Equal(t, "C", p.Credit.SplitID)
	assert.Equal(t, 1, p.GapDays)
	assert.True(t, p.Sure)
	assert.Empty(t, p.Reason)
	assert.Empty(t, p.Note)
}

func TestMatchTransfers_Gaps(t *testing.T) {
	for _, tc := range []struct {
		credit string
		want   int // number of pairs
		sure   bool
		reason string
	}{
		{"2026-01-12", 1, true, ""},
		{"2026-01-13", 1, false, "gap 3 days"},
		{"2026-01-14", 1, false, "gap 4 days"},
		{"2026-01-15", 0, false, ""},
		{"2026-01-06", 1, false, "gap 4 days"}, // credit before debit counts too
	} {
		pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-100"), leg("C", "B", tc.credit, "100")})
		require.Len(t, pairs, tc.want, tc.credit)
		if tc.want == 1 {
			assert.Equal(t, tc.sure, pairs[0].Sure, tc.credit)
			assert.Equal(t, tc.reason, pairs[0].Reason, tc.credit)
		}
	}
}

func TestMatchTransfers_TwoCreditsForOneDebit(t *testing.T) {
	pairs := MatchTransfers([]Leg{
		leg("D", "A", "2026-01-10", "-100"),
		leg("C1", "B", "2026-01-10", "100"),
		leg("C2", "C", "2026-01-11", "100"),
	})
	require.Len(t, pairs, 2, "every combination is listed")
	for _, p := range pairs {
		assert.False(t, p.Sure)
		assert.Equal(t, "2 candidates for the debit", p.Reason)
	}
}

// Review focus 4: one credit, the same debit imported twice.
func TestMatchTransfers_DuplicateDebit(t *testing.T) {
	pairs := MatchTransfers([]Leg{
		leg("D1", "A", "2026-01-10", "-100"),
		leg("D2", "A", "2026-01-10", "-100"),
		leg("C", "B", "2026-01-10", "100"),
	})
	require.Len(t, pairs, 2)
	for _, p := range pairs {
		assert.False(t, p.Sure)
		assert.Equal(t, "2 candidates for the credit", p.Reason)
	}
}

func TestMatchTransfers_NoPair(t *testing.T) {
	eur := leg("C", "B", "2026-01-10", "100")
	eur.CurrencyID = "EUR"
	for name, legs := range map[string][]Leg{
		"different currency": {leg("D", "A", "2026-01-10", "-100"), eur},
		"same account":       {leg("D", "A", "2026-01-10", "-100"), leg("C", "A", "2026-01-10", "100")},
		"amount differs":     {leg("D", "A", "2026-01-10", "-100"), leg("C", "B", "2026-01-10", "100.05")},
		"same sign":          {leg("D", "A", "2026-01-10", "100"), leg("C", "B", "2026-01-10", "100")},
		"bad date":           {leg("D", "A", "10.01.2026", "-100"), leg("C", "B", "2026-01-10", "100")},
	} {
		assert.Empty(t, MatchTransfers(legs), name)
	}
}

func TestMatchTransfers_CentsRounding(t *testing.T) {
	pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-100.004"), leg("C", "B", "2026-01-10", "100.00")})
	require.Len(t, pairs, 1)
}

func TestMatchTransfers_CategoriesDifferIsANoteOnly(t *testing.T) {
	c := leg("C", "B", "2026-01-10", "100")
	c.Category = "Revenus Financiers"
	pairs := MatchTransfers([]Leg{leg("D", "A", "2026-01-10", "-100"), c})
	require.Len(t, pairs, 1)
	assert.True(t, pairs[0].Sure)
	assert.Equal(t, "categories differ", pairs[0].Note)
}

func TestMatchTransfers_DeterministicOrder(t *testing.T) {
	legs := []Leg{
		leg("D2", "A", "2026-02-01", "-50"), leg("C2", "B", "2026-02-01", "50"),
		leg("D1", "A", "2026-01-01", "-20"), leg("C1", "B", "2026-01-01", "20"),
		leg("D3", "A", "2026-01-01", "-10"), leg("C3", "B", "2026-01-01", "10"),
	}
	first := MatchTransfers(legs)
	reversed := make([]Leg, len(legs))
	for i, l := range legs {
		reversed[len(legs)-1-i] = l
	}
	second := MatchTransfers(reversed)
	require.Len(t, first, 3)
	assert.Equal(t, first, second)
	assert.Equal(t, []string{"D3", "D1", "D2"}, []string{first[0].Debit.SplitID, first[1].Debit.SplitID, first[2].Debit.SplitID},
		"by date, then smaller amount first")
}

func TestEligibleLegs(t *testing.T) {
	scope := map[string]bool{"A": true, "B": true}
	ok := leg("OK", "A", "2026-01-10", "-1")
	out := leg("OUT", "Z", "2026-01-10", "-1")
	linked := leg("LNK", "A", "2026-01-10", "-1")
	linked.Linked = true
	multi := leg("MUL", "A", "2026-01-10", "-1")
	multi.SplitCount = 2
	zero := leg("ZER", "A", "2026-01-10", "0.001")
	planned := leg("PLN", "A", "2026-01-10", "-1")
	planned.Status = "ICTransactionStatus.PlannedStatus"
	early := leg("EAR", "A", "2025-12-31", "-1")
	late := leg("LAT", "A", "2026-02-01", "-1")

	got := EligibleLegs([]Leg{ok, out, linked, multi, zero, planned, early, late}, scope, "2026-01-01", "2026-01-31")
	require.Len(t, got, 1)
	assert.Equal(t, "OK", got[0].SplitID)

	all := EligibleLegs([]Leg{ok, early, late}, scope, "", "")
	assert.Len(t, all, 3, "no date bounds without --from/--to")
}
```

- [ ] **Step 2: Run the tests and check they fail**

Run: `go test ./internal/icompta/ -run 'TestMatchTransfers|TestEligibleLegs' -v`
Expected: FAIL to compile, with "undefined: Leg".

- [ ] **Step 3: Implement**

```go
package icompta

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// A candidate pair's dates may be at most maxGapDays apart; only pairs at most
// sureGapDays apart, with no competing candidate, are pre-approved.
const (
	maxGapDays  = 4
	sureGapDays = 2
)

// Leg is one split that could be one side of a transfer.
type Leg struct {
	SplitID, TxID, AccountID, AccountName, CurrencyID string
	Date                                              string // YYYY-MM-DD
	Name                                              string
	Amount                                            decimal.Decimal
	Category                                          string
	Status                                            string
	SplitCount                                        int
	Linked                                            bool
}

// Pair is a candidate transfer: money leaving Debit's account and arriving in
// Credit's.
type Pair struct {
	Debit, Credit Leg
	GapDays       int
	Sure          bool
	Reason        string // why it is not sure; "" when Sure
	Note          string // informational, e.g. "categories differ"
}

// EligibleLegs keeps the legs that may be one side of a transfer: in scope, not
// already linked, the only split of their transaction, non-zero, not planned,
// and within [from, to] when those are set (inclusive, YYYY-MM-DD).
func EligibleLegs(legs []Leg, scope map[string]bool, from, to string) []Leg {
	var out []Leg
	for _, l := range legs {
		switch {
		case !scope[l.AccountID], l.Linked, l.SplitCount != 1, l.Amount.Round(2).IsZero(),
			strings.Contains(l.Status, "Planned"),
			from != "" && l.Date < from, to != "" && l.Date > to:
			continue
		}
		out = append(out, l)
	}
	return out
}

// MatchTransfers pairs debits with credits of the same amount, in another
// account of the same currency, at most maxGapDays apart. Every combination is
// returned; a pair is Sure only when it is close and unambiguous on both sides.
func MatchTransfers(legs []Leg) []Pair {
	var debits, credits []Leg
	for _, l := range legs {
		if l.Amount.IsNegative() {
			debits = append(debits, l)
		} else {
			credits = append(credits, l)
		}
	}
	type candidate struct{ d, c, gap int }
	var cands []candidate
	perDebit := map[int]int{}
	perCredit := map[int]int{}
	for i, d := range debits {
		for j, c := range credits {
			if d.AccountID == c.AccountID || d.CurrencyID != c.CurrencyID {
				continue
			}
			if !d.Amount.Round(2).Neg().Equal(c.Amount.Round(2)) {
				continue
			}
			gap, ok := dayGap(d.Date, c.Date)
			if !ok || gap > maxGapDays {
				continue
			}
			cands = append(cands, candidate{i, j, gap})
			perDebit[i]++
			perCredit[j]++
		}
	}

	pairs := make([]Pair, 0, len(cands))
	for _, k := range cands {
		p := Pair{Debit: debits[k.d], Credit: credits[k.c], GapDays: k.gap}
		var reasons []string
		if k.gap > sureGapDays {
			reasons = append(reasons, fmt.Sprintf("gap %d days", k.gap))
		}
		if n := perDebit[k.d]; n > 1 {
			reasons = append(reasons, fmt.Sprintf("%d candidates for the debit", n))
		}
		if n := perCredit[k.c]; n > 1 {
			reasons = append(reasons, fmt.Sprintf("%d candidates for the credit", n))
		}
		p.Sure = len(reasons) == 0
		p.Reason = strings.Join(reasons, "; ")
		if normalize(p.Debit.Category) != normalize(p.Credit.Category) {
			p.Note = "categories differ"
		}
		pairs = append(pairs, p)
	}

	sort.SliceStable(pairs, func(a, b int) bool {
		x, y := pairs[a], pairs[b]
		switch {
		case x.Debit.Date != y.Debit.Date:
			return x.Debit.Date < y.Debit.Date
		case x.Debit.AccountName != y.Debit.AccountName:
			return x.Debit.AccountName < y.Debit.AccountName
		case !x.Debit.Amount.Equal(y.Debit.Amount):
			return x.Debit.Amount.GreaterThan(y.Debit.Amount) // -10 before -20: smaller amount first
		case x.Debit.SplitID != y.Debit.SplitID:
			return x.Debit.SplitID < y.Debit.SplitID
		case x.Credit.Date != y.Credit.Date:
			return x.Credit.Date < y.Credit.Date
		}
		return x.Credit.SplitID < y.Credit.SplitID
	})
	return pairs
}

// dayGap is the absolute number of days between two YYYY-MM-DD dates.
func dayGap(a, b string) (int, bool) {
	ta, err := time.Parse("2006-01-02", a)
	if err != nil {
		return 0, false
	}
	tb, err := time.Parse("2006-01-02", b)
	if err != nil {
		return 0, false
	}
	d := int(ta.Sub(tb).Hours() / 24)
	if d < 0 {
		d = -d
	}
	return d, true
}
```

- [ ] **Step 4: Run the tests and check they pass**

Run: `go test ./internal/icompta/ -run 'TestMatchTransfers|TestEligibleLegs' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/icompta/transfers.go internal/icompta/transfers_test.go
git commit -m "feat(icompta): match transfer pairs between own accounts"
```

---

### Task 3: Read accounts and legs from the database

**Files:**
- Modify: `internal/icompta/store.go` (`requireColumns` signature and its call in `OpenReadOnly`)
- Create: `internal/icompta/transfers_store.go`
- Test: `internal/icompta/transfers_store_test.go`

**Interfaces:**
- Consumes:
  - `Account`, `classAccount` and `classFolder` (Task 1);
  - `Leg`, `EligibleLegs` and `MatchTransfers` (Task 2);
  - `loadCategories`, `readState` and `rowsQuerier` (store.go).
- Produces:
  - `type TransferSnapshot struct { Accounts []Account; Legs []Leg; State string }`
  - `func (s *Store) TransferSnapshot(ctx context.Context) (TransferSnapshot, error)`
  - `requireColumns(ctx context.Context, q rowsQuerier, required map[string][]string) error`, whose signature changes
  - test helper `newTransferFixtureDB(t *testing.T) string` (in `transfers_store_test.go`), used by Tasks 5 and 6's internal tests

- [ ] **Step 1: Write the failing tests**

```go
package icompta

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const transferFixtureSchema = `
CREATE TABLE ICCategory (ID TEXT UNIQUE NOT NULL, name TEXT NOT NULL);
CREATE TABLE ICAccount (ID TEXT UNIQUE NOT NULL, name TEXT, class TEXT, parent TEXT, currency TEXT);
CREATE TABLE ICTransaction (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, account TEXT, date TEXT NOT NULL, name TEXT NOT NULL,
	comment TEXT, amount TEXT, payee TEXT, investmentTransactionInfo TEXT, status TEXT);
CREATE TABLE ICTransactionSplit (
	ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, "transaction" TEXT NOT NULL,
	amount TEXT, category TEXT, linkedSplit TEXT);
`

// newTransferFixtureDB: Fred owns BCV (A1) and Selma (A2), CHF, plus a EUR
// account (A3); Lydie's account (A4) is outside. S1/S2 are a sure transfer.
func newTransferFixtureDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ic link.cdb")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	for _, s := range []string{
		transferFixtureSchema,
		`INSERT INTO ICCategory VALUES ('C-VIR','Virements'),('C-ALI','Alimentation')`,
		`INSERT INTO ICAccount VALUES ('F','Fred','ICAccountsGroup',NULL,NULL),
			('A1','Compte privé','ICAccount','F','CHF'),('A2','Selma','ICAccount','F','CHF'),
			('A3','Epargne EUR','ICAccount','F','EUR'),
			('L','Lydie','ICAccountsGroup',NULL,NULL),('A4','Livret A','ICAccount','L','CHF'),
			('P','Florence','ICPerson','F',NULL)`,
		// Review focus 3: a date stored with a time must match like a plain date.
		`INSERT INTO ICTransaction VALUES ('T1',NULL,'A1','2026-01-10 00:00:00','VERS SELMA','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S1',NULL,'T1','-750','C-VIR',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T2',NULL,'A2','2026-01-11','DEPOT','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S2',NULL,'T2','750','C-VIR','')`,
		`INSERT INTO ICTransaction VALUES ('T3',NULL,'A1','2026-01-15','MIGROS','','-42.10','',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S3',NULL,'T3','','C-ALI',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T4',NULL,'A4','2026-01-11','LYDIE','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S4',NULL,'T4','750','C-VIR',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T5',NULL,'A1','2026-01-20','SPLIT','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S5a',NULL,'T5','-30','C-ALI',NULL),('S5b',NULL,'T5','-20','C-ALI',NULL)`,
		`INSERT INTO ICTransaction VALUES ('T6',NULL,'A2','2026-01-20','PLANNED','',NULL,'',NULL,'ICTransactionStatus.PlannedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S6',NULL,'T6','50','C-VIR',NULL)`,
	} {
		_, err := db.Exec(s)
		require.NoError(t, err, s)
	}
	return path
}

func TestTransferSnapshot_ReadsAccountsAndLegs(t *testing.T) {
	s, err := OpenReadOnly(context.Background(), newTransferFixtureDB(t))
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	snap, err := s.TransferSnapshot(context.Background())
	require.NoError(t, err)
	assert.Len(t, snap.Accounts, 7)
	assert.Contains(t, snap.State, "splits=7")

	byID := map[string]Leg{}
	for _, l := range snap.Legs {
		byID[l.SplitID] = l
	}
	require.Len(t, byID, 7)
	s1 := byID["S1"]
	assert.Equal(t, "2026-01-10", s1.Date, "time part dropped")
	assert.Equal(t, "T1", s1.TxID)
	assert.Equal(t, "A1", s1.AccountID)
	assert.Equal(t, "Compte privé", s1.AccountName)
	assert.Equal(t, "CHF", s1.CurrencyID)
	assert.Equal(t, "Virements", s1.Category)
	assert.Equal(t, "-750", s1.Amount.String())
	assert.Equal(t, 1, s1.SplitCount)
	assert.False(t, s1.Linked)
	assert.Equal(t, "-42.1", byID["S3"].Amount.String(), "empty split amount falls back to the transaction")
	assert.Equal(t, 2, byID["S5a"].SplitCount)
	assert.Contains(t, byID["S6"].Status, "Planned")
}

func TestTransferSnapshot_EndToEndFindsTheOneSurePair(t *testing.T) {
	s, err := OpenReadOnly(context.Background(), newTransferFixtureDB(t))
	require.NoError(t, err)
	defer func() { _ = s.Close() }()
	snap, err := s.TransferSnapshot(context.Background())
	require.NoError(t, err)

	scope, err := ResolveScope(snap.Accounts, []string{"Fred"})
	require.NoError(t, err)
	pairs := MatchTransfers(EligibleLegs(snap.Legs, scope, "", ""))
	require.Len(t, pairs, 1, "Lydie's S4 is out of scope, so S1/S2 stay unambiguous")
	assert.Equal(t, "S1", pairs[0].Debit.SplitID)
	assert.Equal(t, "S2", pairs[0].Credit.SplitID)
	assert.True(t, pairs[0].Sure)
}

func TestTransferSnapshot_MissingAccountTableIsClear(t *testing.T) {
	s, _ := openFixture(t) // the recategorize fixture has no ICAccount table
	_, err := s.TransferSnapshot(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ICAccount.ID is missing")
}
```

- [ ] **Step 2: Run the tests and check they fail**

Run: `go test ./internal/icompta/ -run TestTransferSnapshot -v`
Expected: FAIL to compile, with "s.TransferSnapshot undefined".

- [ ] **Step 3: Implement**

In `internal/icompta/store.go`, change `requireColumns` so it takes the map and a querier, and update its one caller:

```go
	if err := requireColumns(ctx, db, requiredColumns); err != nil {
```

```go
func requireColumns(ctx context.Context, q rowsQuerier, required map[string][]string) error {
	// Sorted so the first reported problem is deterministic.
	tables := make([]string, 0, len(required))
	for table := range required {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		cols := required[table]
		rows, err := q.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
		// ... the rest of the body is unchanged ...
```

Create `internal/icompta/transfers_store.go`:

```go
package icompta

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// transferColumns are what link-transfers reads beyond requiredColumns. They are
// checked only by TransferSnapshot, so recategorize keeps working on databases
// (and fixtures) without them.
var transferColumns = map[string][]string{
	"ICAccount":     {"ID", "name", "class", "parent", "currency"},
	"ICTransaction": {"account", "status"},
}

// TransferSnapshot is a consistent read of what link-transfers needs.
type TransferSnapshot struct {
	Accounts []Account
	Legs     []Leg
	State    string
}

// TransferSnapshot reads accounts, every split as a Leg, and the state marker in
// one read transaction.
func (s *Store) TransferSnapshot(ctx context.Context) (TransferSnapshot, error) {
	if err := requireColumns(ctx, s.db, transferColumns); err != nil {
		return TransferSnapshot{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TransferSnapshot{}, fmt.Errorf("begin read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cats, err := loadCategories(ctx, tx)
	if err != nil {
		return TransferSnapshot{}, err
	}
	state, err := readState(ctx, tx)
	if err != nil {
		return TransferSnapshot{}, err
	}

	arows, err := tx.QueryContext(ctx, `
SELECT ID, COALESCE(name,''), COALESCE(class,''), COALESCE(parent,''), COALESCE(currency,'')
FROM ICAccount`)
	if err != nil {
		return TransferSnapshot{}, fmt.Errorf("read accounts: %w", err)
	}
	var accounts []Account
	byID := map[string]Account{}
	for arows.Next() {
		var a Account
		if err := arows.Scan(&a.ID, &a.Name, &a.Class, &a.ParentID, &a.CurrencyID); err != nil {
			_ = arows.Close()
			return TransferSnapshot{}, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
		byID[a.ID] = a
	}
	if err := arows.Err(); err != nil {
		_ = arows.Close()
		return TransferSnapshot{}, fmt.Errorf("read accounts: %w", err)
	}
	_ = arows.Close()

	rows, err := tx.QueryContext(ctx, `
SELECT s.ID, t.ID, COALESCE(t.account,''), substr(t.date,1,10), t.name,
       COALESCE(NULLIF(s.amount,''), t.amount, ''), COALESCE(s.category,''), COALESCE(t.status,''),
       (SELECT COUNT(*) FROM ICTransactionSplit s2 WHERE s2."transaction" = t.ID),
       CASE WHEN COALESCE(s.linkedSplit,'') <> '' THEN 1 ELSE 0 END
FROM ICTransactionSplit s
JOIN ICTransaction t ON t.ID = s."transaction"
ORDER BY t.date, s.ID`)
	if err != nil {
		return TransferSnapshot{}, fmt.Errorf("read splits: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var legs []Leg
	for rows.Next() {
		var l Leg
		var amount, categoryID string
		var linked int
		if err := rows.Scan(&l.SplitID, &l.TxID, &l.AccountID, &l.Date, &l.Name,
			&amount, &categoryID, &l.Status, &l.SplitCount, &linked); err != nil {
			return TransferSnapshot{}, fmt.Errorf("scan split: %w", err)
		}
		if strings.TrimSpace(amount) != "" {
			l.Amount, err = decimal.NewFromString(strings.TrimSpace(amount))
			if err != nil {
				return TransferSnapshot{}, fmt.Errorf("split %s: amount %q: %w", l.SplitID, amount, err)
			}
		}
		l.Category = cats.NameByID(categoryID)
		l.AccountName = byID[l.AccountID].Name
		l.CurrencyID = byID[l.AccountID].CurrencyID
		l.Linked = linked == 1
		legs = append(legs, l)
	}
	if err := rows.Err(); err != nil {
		return TransferSnapshot{}, fmt.Errorf("read splits: %w", err)
	}
	return TransferSnapshot{Accounts: accounts, Legs: legs, State: state}, nil
}
```

- [ ] **Step 4: Run the tests and check they pass, recategorize included**

Run: `go test ./internal/icompta/ -v`
Expected: PASS, all existing recategorize tests included.

- [ ] **Step 5: Commit**

```bash
git add internal/icompta/store.go internal/icompta/transfers_store.go internal/icompta/transfers_store_test.go
git commit -m "feat(icompta): read accounts and transfer legs"
```

---

### Task 4: Transfer report

**Files:**
- Modify: `internal/icompta/report.go`. Extract the BOM and comment-line reader into `readReportPreamble`; `ReadReport` uses it, with unchanged behaviour and messages.
- Create: `internal/icompta/transfers_report.go`
- Test: `internal/icompta/transfers_report_test.go`

**Interfaces:**
- Consumes: `Pair` (Task 2); `csvsafe.Escape` and `csvsafe.Unescape`; `stateKey`.
- Produces:
  - `const transferReportMagic = "# camt-csv link-transfers report v1"`
  - `type TransferRow struct { DebitSplitID, CreditSplitID, DebitDate, CreditDate string; GapDays int; DebitAccount, CreditAccount, DebitName, CreditName, Amount, DebitCategory, CreditCategory, Confidence, Reason, Note string; Apply bool }`. `Confidence` is `"sure"` or `"doubtful"`; `Amount` is the absolute value, 2 decimals.
  - `type TransferReport struct { DBState string; Rows []TransferRow }`
  - `func NewTransferReport(state string, pairs []Pair) TransferReport`
  - `func (r TransferReport) Write(w io.Writer) error`
  - `func ReadTransferReport(rd io.Reader) (TransferReport, error)`
  - `func readReportPreamble(br *bufio.Reader, magic, kind string) (state string, err error)`

- [ ] **Step 1: Write the failing tests**

```go
package icompta

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samplePairs() []Pair {
	d := Leg{SplitID: "S1", Date: "2026-01-10", AccountName: "Compte privé", Name: "=HYPERLINK(1)",
		Amount: decimal.RequireFromString("-750"), Category: "Virements"}
	c := Leg{SplitID: "S2", Date: "2026-01-11", AccountName: "Selma", Name: "DEPOT",
		Amount: decimal.RequireFromString("750"), Category: "Virements"}
	return []Pair{
		{Debit: d, Credit: c, GapDays: 1, Sure: true},
		{Debit: d, Credit: c, GapDays: 3, Reason: "gap 3 days", Note: "categories differ"},
	}
}

func TestTransferReport_RoundTrip(t *testing.T) {
	rep := NewTransferReport("splits=8;x", samplePairs())
	require.Len(t, rep.Rows, 2)
	assert.Equal(t, "750.00", rep.Rows[0].Amount)
	assert.Equal(t, "sure", rep.Rows[0].Confidence)
	assert.True(t, rep.Rows[0].Apply)
	assert.Equal(t, "doubtful", rep.Rows[1].Confidence)
	assert.False(t, rep.Rows[1].Apply)

	var buf bytes.Buffer
	require.NoError(t, rep.Write(&buf))
	assert.True(t, strings.HasPrefix(buf.String(), "# camt-csv link-transfers report v1\n# db_state=splits=8;x\n"))
	assert.Contains(t, buf.String(), "'=HYPERLINK(1)", "formula escaped in the file")

	back, err := ReadTransferReport(&buf)
	require.NoError(t, err)
	assert.Equal(t, rep, back)
}

// Review focus 5: what Numbers/Excel do to a CSV.
func TestReadTransferReport_SpreadsheetEdited(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewTransferReport("st", samplePairs()).Write(&buf))
	edited := "\xEF\xBB\xBF" + strings.ReplaceAll(buf.String(), "\n", "\r\n")
	edited = strings.Replace(edited, "# db_state=st\r\n", "# db_state=st,,,\r\n", 1)
	edited = strings.Replace(edited, ",yes\r\n", ",Yes\r\n", 1)

	rep, err := ReadTransferReport(strings.NewReader(edited))
	require.NoError(t, err)
	assert.Equal(t, "st", rep.DBState)
	assert.True(t, rep.Rows[0].Apply)
}

func TestReadTransferReport_Errors(t *testing.T) {
	var good bytes.Buffer
	require.NoError(t, NewTransferReport("st", samplePairs()).Write(&good))
	for name, tc := range map[string]struct{ in, want string }{
		"recategorize report": {"# camt-csv recategorize report v1\n# db_state=st\n", "not a camt-csv link-transfers report"},
		"no state":            {"# camt-csv link-transfers report v1\n", "no db_state line"},
		"bad header":          {"# camt-csv link-transfers report v1\n# db_state=st\na,b\n", "report header is not"},
		"bad apply":           {strings.Replace(good.String(), ",yes\n", ",maybe\n", 1), `line 4: apply "maybe"`},
		"bad confidence":      {strings.Replace(good.String(), ",sure,", ",certain,", 1), `line 4: confidence "certain"`},
		"bad gap":             {strings.Replace(good.String(), ",2026-01-11,1,", ",2026-01-11,x,", 1), `line 4: gap_days "x"`},
	} {
		_, err := ReadTransferReport(strings.NewReader(tc.in))
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), tc.want, name)
	}
}

func TestReadReport_RejectsATransferReport(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewTransferReport("st", samplePairs()).Write(&buf))
	_, err := ReadReport(&buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a camt-csv recategorize report")
}
```

- [ ] **Step 2: Run the tests and check they fail**

Run: `go test ./internal/icompta/ -run 'TransferReport|RejectsATransferReport' -v`
Expected: FAIL to compile, with "undefined: NewTransferReport".

- [ ] **Step 3: Implement**

In `internal/icompta/report.go`, add the helper. Then replace the BOM and comment loop at the top of `ReadReport` with a call to it, keeping the same messages:

```go
// readReportPreamble skips a UTF-8 BOM and reads the leading "#" lines,
// tolerating a spreadsheet's trailing commas. It returns the db_state value and
// fails unless the magic line identified a report of this kind.
func readReportPreamble(br *bufio.Reader, magic, kind string) (string, error) {
	if b, err := br.Peek(3); err == nil && bytes.Equal(b, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}
	state := ""
	magicSeen := false
	for {
		first, err := br.Peek(1)
		if err != nil || first[0] != '#' {
			break
		}
		line, err := br.ReadString('\n')
		line = strings.TrimRight(line, ", \r\n")
		switch {
		case line == magic:
			magicSeen = true
		case strings.HasPrefix(line, stateKey):
			state = strings.TrimPrefix(line, stateKey)
		}
		if err != nil {
			break
		}
	}
	if !magicSeen {
		return "", fmt.Errorf("not a camt-csv %s report (missing %q line)", kind, magic)
	}
	if state == "" {
		return "", fmt.Errorf("report has no db_state line")
	}
	return state, nil
}
```

`ReadReport` now starts like this:

```go
func ReadReport(rd io.Reader) (Report, error) {
	br := bufio.NewReader(rd)
	state, err := readReportPreamble(br, reportMagic, "recategorize")
	if err != nil {
		return Report{}, err
	}
	rep := Report{DBState: state}

	cr := csv.NewReader(br)
	// ... unchanged from here ...
```

Create `internal/icompta/transfers_report.go`:

```go
package icompta

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"fjacquet/camt-csv/internal/csvsafe"
)

const transferReportMagic = "# camt-csv link-transfers report v1"

const (
	confidenceSure     = "sure"
	confidenceDoubtful = "doubtful"
)

var transferReportHeader = []string{
	"debit_split_id", "credit_split_id", "debit_date", "credit_date", "gap_days",
	"debit_account", "credit_account", "debit_name", "credit_name", "amount",
	"debit_category", "credit_category", "confidence", "reason", "note", "apply",
}

// TransferRow is one reviewed candidate pair. Only rows with Apply are linked.
type TransferRow struct {
	DebitSplitID, CreditSplitID string
	DebitDate, CreditDate       string
	GapDays                     int
	DebitAccount, CreditAccount string
	DebitName, CreditName       string
	Amount                      string // absolute, 2 decimals
	DebitCategory               string
	CreditCategory              string
	Confidence                  string // "sure" or "doubtful"
	Reason, Note                string
	Apply                       bool
}

// TransferReport is what link-transfers preview writes and apply reads.
type TransferReport struct {
	DBState string
	Rows    []TransferRow
}

// NewTransferReport turns candidate pairs into report rows; sure pairs are
// pre-approved, doubtful ones are not.
func NewTransferReport(state string, pairs []Pair) TransferReport {
	rep := TransferReport{DBState: state, Rows: make([]TransferRow, 0, len(pairs))}
	for _, p := range pairs {
		conf := confidenceDoubtful
		if p.Sure {
			conf = confidenceSure
		}
		rep.Rows = append(rep.Rows, TransferRow{
			DebitSplitID: p.Debit.SplitID, CreditSplitID: p.Credit.SplitID,
			DebitDate: p.Debit.Date, CreditDate: p.Credit.Date, GapDays: p.GapDays,
			DebitAccount: p.Debit.AccountName, CreditAccount: p.Credit.AccountName,
			DebitName: p.Debit.Name, CreditName: p.Credit.Name,
			Amount:        p.Debit.Amount.Abs().StringFixed(2),
			DebitCategory: p.Debit.Category, CreditCategory: p.Credit.Category,
			Confidence: conf, Reason: p.Reason, Note: p.Note, Apply: p.Sure,
		})
	}
	return rep
}

// Write encodes the report: two comment lines, then a CSV with a header.
func (r TransferReport) Write(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s\n%s%s\n", transferReportMagic, stateKey, r.DBState); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(transferReportHeader); err != nil {
		return err
	}
	e := csvsafe.Escape
	for _, row := range r.Rows {
		apply := "no"
		if row.Apply {
			apply = "yes"
		}
		if err := cw.Write([]string{
			e(row.DebitSplitID), e(row.CreditSplitID), e(row.DebitDate), e(row.CreditDate), strconv.Itoa(row.GapDays),
			e(row.DebitAccount), e(row.CreditAccount), e(row.DebitName), e(row.CreditName), row.Amount,
			e(row.DebitCategory), e(row.CreditCategory), row.Confidence, e(row.Reason), e(row.Note), apply,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ReadTransferReport decodes a report, tolerating what a spreadsheet does to a
// CSV (BOM, CRLF, trailing commas on comment lines, any case of yes/no).
func ReadTransferReport(rd io.Reader) (TransferReport, error) {
	br := bufio.NewReader(rd)
	state, err := readReportPreamble(br, transferReportMagic, "link-transfers")
	if err != nil {
		return TransferReport{}, err
	}
	rep := TransferReport{DBState: state}

	cr := csv.NewReader(br)
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return TransferReport{}, fmt.Errorf("read report rows: %w", err)
	}
	if len(records) == 0 || strings.Join(records[0], ",") != strings.Join(transferReportHeader, ",") {
		return TransferReport{}, fmt.Errorf("report header is not %q", strings.Join(transferReportHeader, ","))
	}
	u := csvsafe.Unescape
	for i, rec := range records[1:] {
		lineNo := i + 4 // 2 comment lines, 1 header, 1-based
		if len(rec) != len(transferReportHeader) {
			return TransferReport{}, fmt.Errorf("line %d: %d fields, expected %d", lineNo, len(rec), len(transferReportHeader))
		}
		gap, err := strconv.Atoi(strings.TrimSpace(rec[4]))
		if err != nil {
			return TransferReport{}, fmt.Errorf("line %d: gap_days %q is not a number", lineNo, rec[4])
		}
		conf := strings.ToLower(strings.TrimSpace(rec[12]))
		if conf != confidenceSure && conf != confidenceDoubtful {
			return TransferReport{}, fmt.Errorf("line %d: confidence %q is not sure or doubtful", lineNo, rec[12])
		}
		var apply bool
		switch strings.ToLower(strings.TrimSpace(rec[15])) {
		case "yes":
			apply = true
		case "no":
		default:
			return TransferReport{}, fmt.Errorf("line %d: apply %q is not yes or no", lineNo, rec[15])
		}
		rep.Rows = append(rep.Rows, TransferRow{
			DebitSplitID: u(rec[0]), CreditSplitID: u(rec[1]), DebitDate: u(rec[2]), CreditDate: u(rec[3]),
			GapDays: gap, DebitAccount: u(rec[5]), CreditAccount: u(rec[6]), DebitName: u(rec[7]),
			CreditName: u(rec[8]), Amount: rec[9], DebitCategory: u(rec[10]), CreditCategory: u(rec[11]),
			Confidence: conf, Reason: u(rec[13]), Note: u(rec[14]), Apply: apply,
		})
	}
	return rep, nil
}
```

- [ ] **Step 4: Run all icompta tests**

Run: `go test ./internal/icompta/ -v`
Expected: PASS, the existing `TestReadReport_*` included.

- [ ] **Step 5: Commit**

```bash
git add internal/icompta/report.go internal/icompta/transfers_report.go internal/icompta/transfers_report_test.go
git commit -m "feat(icompta): link-transfers review report"
```

---

### Task 5: Apply links

**Files:**
- Modify: `internal/icompta/apply.go`. Extract the shared pre-write sequence into `openForWrite`; `Apply` calls it, with its behaviour and error messages unchanged.
- Create: `internal/icompta/transfers_apply.go`
- Test: `internal/icompta/transfers_apply_test.go`

**Interfaces:**
- Consumes:
  - `TransferReport` and `TransferRow` (Task 4);
  - `newTransferFixtureDB` (Task 3 test helper);
  - `ApplyOptions` and `ApplyResult` (apply.go);
  - `fixedNow`, `notRunning`, `opts` and `previewState` (apply_test.go helpers, same package).
- Produces:
  - `func openForWrite(ctx context.Context, opts ApplyOptions, reportState string, log logging.Logger) (*sql.DB, string, error)`. The returned string is the backup path, set as soon as the backup exists, even when a later step fails.
  - `func ApplyLinks(ctx context.Context, rep TransferReport, opts ApplyOptions, log logging.Logger) (ApplyResult, error)`. `ApplyResult.Skipped` counts approved rows that were skipped.

- [ ] **Step 1: Write the failing tests**

```go
package icompta

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func linkOf(t *testing.T, path, splitID string) (linked, splitMod, txMod string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	var l, sm, tm sql.NullString
	require.NoError(t, db.QueryRow(`
SELECT s.linkedSplit, s.lastModificationDate, t.lastModificationDate
FROM ICTransactionSplit s JOIN ICTransaction t ON t.ID = s."transaction" WHERE s.ID = ?`, splitID).Scan(&l, &sm, &tm))
	return l.String, sm.String, tm.String
}

func balances(t *testing.T, path string) map[string]string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	rows, err := db.Query(`
SELECT t.account, COALESCE(NULLIF(s.amount,''), t.amount, '0')
FROM ICTransactionSplit s JOIN ICTransaction t ON t.ID = s."transaction"`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	sum := map[string]decimal.Decimal{}
	for rows.Next() {
		var acct, amt string
		require.NoError(t, rows.Scan(&acct, &amt))
		sum[acct] = sum[acct].Add(decimal.RequireFromString(amt))
	}
	out := map[string]string{}
	for k, v := range sum {
		out[k] = v.StringFixed(2)
	}
	return out
}

func backups(t *testing.T, path string) []string {
	t.Helper()
	m, err := filepath.Glob(path + ".bak-*")
	require.NoError(t, err)
	return m
}

func row(debit, credit string, apply bool) TransferRow {
	return TransferRow{DebitSplitID: debit, CreditSplitID: credit, Confidence: "sure", Apply: apply}
}

func TestApplyLinks_LinksBothWaysAndKeepsBalances(t *testing.T) {
	path := newTransferFixtureDB(t)
	before := balances(t, path)
	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{
		row("S1", "S2", true),
		row("S3", "S4", false), // not approved: untouched
	}}

	res, err := ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, res.Applied)
	assert.Equal(t, 0, res.Skipped)
	assert.NotEmpty(t, res.BackupPath)
	assert.FileExists(t, res.BackupPath)

	stamp := fixedNow.UTC().Format(iComptaTimeFormat)
	l, sm, tm := linkOf(t, path, "S1")
	assert.Equal(t, []string{"S2", stamp, stamp}, []string{l, sm, tm})
	l, sm, tm = linkOf(t, path, "S2")
	assert.Equal(t, []string{"S1", stamp, stamp}, []string{l, sm, tm})
	l, _, _ = linkOf(t, path, "S3")
	assert.Empty(t, l)
	assert.Equal(t, before, balances(t, path), "linking never changes an amount")
}

// Review focus 2: the first apply moves the state marker.
func TestApplyLinks_SecondApplyIsRefused(t *testing.T) {
	path := newTransferFixtureDB(t)
	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{row("S1", "S2", true)}}
	_, err := ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.NoError(t, err)

	_, err = ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run preview again")
}

func TestApplyLinks_SkipsWhatNoLongerFits(t *testing.T) {
	path := newTransferFixtureDB(t)
	// S2 already linked elsewhere before the preview was taken.
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE ICTransactionSplit SET linkedSplit='OTHER' WHERE ID='S2'`)
	require.NoError(t, err)
	_ = db.Close()

	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{
		row("S1", "S2", true),   // S2 already linked
		row("S3", "S6", true),   // -42.10 vs 50: amounts do not match
		row("GONE", "S4", true), // debit split missing
	}}

	log := logging.NewMockLogger()
	res, err := ApplyLinks(context.Background(), rep, opts(path), log)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Applied)
	assert.Equal(t, 3, res.Skipped)
	assert.True(t, log.HasEntry("WARN", "Skipped pair"))
	l, _, _ := linkOf(t, path, "S1")
	assert.Empty(t, l)
}

func TestApplyLinks_SplitInTwoApprovedRowsRejectsEverything(t *testing.T) {
	path := newTransferFixtureDB(t)
	rep := TransferReport{DBState: previewState(t, path), Rows: []TransferRow{
		row("S1", "S2", true),
		row("S1", "S4", true),
	}}
	_, err := ApplyLinks(context.Background(), rep, opts(path), logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "S1")
	assert.Empty(t, backups(t, path), "rejected before any backup or write")
	l, _, _ := linkOf(t, path, "S2")
	assert.Empty(t, l)
}

func TestApplyLinks_RefusesWhileICComptaRuns(t *testing.T) {
	path := newTransferFixtureDB(t)
	o := opts(path)
	o.IsRunning = func() (bool, error) { return true, nil }
	_, err := ApplyLinks(context.Background(), TransferReport{DBState: previewState(t, path)}, o, logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "iCompta is running")
	assert.Empty(t, backups(t, path))
}
```

- [ ] **Step 2: Run the tests and check they fail**

Run: `go test ./internal/icompta/ -run TestApplyLinks -v`
Expected: FAIL to compile, with "undefined: ApplyLinks".

- [ ] **Step 3: Extract `openForWrite` from `Apply`**

In `internal/icompta/apply.go`, move the code from the `IsRunning` check through `integrityCheck(ctx, db)` "before apply" into:

```go
// openForWrite runs the checks every apply needs before touching the database:
// iCompta closed, no pending WAL, data unchanged since the preview. It then
// backs the database up, opens a write connection and checks integrity. The
// backup path is returned as soon as the backup exists, even on a later error.
func openForWrite(ctx context.Context, opts ApplyOptions, reportState string, log logging.Logger) (*sql.DB, string, error) {
	running, err := opts.IsRunning()
	if err != nil {
		return nil, "", err
	}
	if running {
		return nil, "", errors.New("iCompta is running: quit it before applying, it would overwrite or lock the database")
	}
	if fi, err := os.Stat(opts.DBPath + "-wal"); err == nil && fi.Size() > 0 {
		return nil, "", fmt.Errorf("%s-wal is not empty: the database has uncommitted pages, a file copy would not be a consistent backup", opts.DBPath)
	}

	ro, err := OpenReadOnly(ctx, opts.DBPath)
	if err != nil {
		return nil, "", err
	}
	state, err := ro.State(ctx)
	_ = ro.Close()
	if err != nil {
		return nil, "", err
	}
	if state != reportState {
		return nil, "", fmt.Errorf("the database changed since the preview (preview saw %q, now %q): run preview again", reportState, state)
	}

	backup := fmt.Sprintf("%s.bak-%s", opts.DBPath, opts.Now().UTC().Format("20060102T150405Z"))
	if err := copyVerified(opts.DBPath, backup); err != nil {
		return nil, "", err
	}
	log.WithFields(logging.Field{Key: "backup", Value: backup}).Info("Database backed up")

	// _timeout is the validated shorthand for PRAGMA busy_timeout; _txlock=immediate
	// takes the write lock at BEGIN instead of failing late on the first UPDATE.
	uri, err := dsn(opts.DBPath, "_timeout=5000&_txlock=immediate")
	if err != nil {
		return nil, backup, err
	}
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, backup, fmt.Errorf("open database for writing: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := integrityCheck(ctx, db); err != nil {
		_ = db.Close()
		return nil, backup, fmt.Errorf("before apply: %w", err)
	}
	return db, backup, nil
}
```

`Apply` then begins:

```go
func Apply(ctx context.Context, rep Report, opts ApplyOptions, log logging.Logger) (ApplyResult, error) {
	db, backup, err := openForWrite(ctx, opts, rep.DBState, log)
	res := ApplyResult{BackupPath: backup}
	if err != nil {
		return res, err
	}
	defer func() { _ = db.Close() }()

	tx, err := db.BeginTx(ctx, nil)
	// ... unchanged from here ...
```

Run: `go test ./internal/icompta/ -run TestApply_ -v`
Expected: PASS. The recategorize apply tests are unchanged and must stay green.

- [ ] **Step 4: Implement `ApplyLinks`**

Create `internal/icompta/transfers_apply.go`:

```go
package icompta

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"fjacquet/camt-csv/internal/logging"

	"github.com/shopspring/decimal"
)

// ApplyLinks links the approved pairs of a report, both ways, as iCompta does:
// each split's linkedSplit names the other. Only linkedSplit and modification
// dates are written. A split used by two approved rows rejects the whole report
// before any backup or write.
func ApplyLinks(ctx context.Context, rep TransferReport, opts ApplyOptions, log logging.Logger) (ApplyResult, error) {
	var approved []TransferRow
	used := map[string]int{}
	for _, r := range rep.Rows {
		if !r.Apply {
			continue
		}
		approved = append(approved, r)
		used[r.DebitSplitID]++
		used[r.CreditSplitID]++
	}
	var twice []string
	for id, n := range used {
		if n > 1 {
			twice = append(twice, id)
		}
	}
	if len(twice) > 0 {
		return ApplyResult{}, fmt.Errorf("a split can be linked only once, but these appear in several approved rows: %s", strings.Join(twice, ", "))
	}

	db, backup, err := openForWrite(ctx, opts, rep.DBState, log)
	res := ApplyResult{BackupPath: backup}
	if err != nil {
		return res, err
	}
	defer func() { _ = db.Close() }()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("begin write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stamp := opts.Now().UTC().Format(iComptaTimeFormat)

	for _, r := range approved {
		reason, err := linkPair(ctx, tx, r, stamp)
		if err != nil {
			return res, err
		}
		if reason != "" {
			res.Skipped++
			log.WithFields(
				logging.Field{Key: "debit", Value: r.DebitSplitID},
				logging.Field{Key: "credit", Value: r.CreditSplitID},
				logging.Field{Key: "reason", Value: reason},
			).Warn("Skipped pair")
			continue
		}
		res.Applied++
		log.WithFields(
			logging.Field{Key: "debit_date", Value: r.DebitDate},
			logging.Field{Key: "credit_date", Value: r.CreditDate},
			logging.Field{Key: "from", Value: r.DebitAccount},
			logging.Field{Key: "to", Value: r.CreditAccount},
			logging.Field{Key: "amount", Value: r.Amount},
		).Info("Linked transfer")
	}

	if err := integrityCheck(ctx, tx); err != nil {
		return res, fmt.Errorf("after apply, rolled back: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return res, fmt.Errorf("commit: %w", err)
	}
	log.WithFields(
		logging.Field{Key: "linked", Value: res.Applied},
		logging.Field{Key: "skipped", Value: res.Skipped},
		logging.Field{Key: "backup", Value: backup},
	).Info("Transfers linked")
	return res, nil
}

type linkSide struct {
	txID   string
	amount decimal.Decimal
	linked bool
}

func readLinkSide(ctx context.Context, tx *sql.Tx, splitID string) (linkSide, error) {
	var side linkSide
	var amount, linked string
	err := tx.QueryRowContext(ctx, `
SELECT s."transaction", COALESCE(NULLIF(s.amount,''), t.amount, ''), COALESCE(s.linkedSplit,'')
FROM ICTransactionSplit s JOIN ICTransaction t ON t.ID = s."transaction"
WHERE s.ID = ?`, splitID).Scan(&side.txID, &amount, &linked)
	if err != nil {
		return side, err
	}
	side.linked = linked != ""
	if strings.TrimSpace(amount) != "" {
		side.amount, err = decimal.NewFromString(strings.TrimSpace(amount))
		if err != nil {
			return side, fmt.Errorf("split %s: amount %q: %w", splitID, amount, err)
		}
	}
	return side, nil
}

// linkPair links one approved row. It returns a skip reason when the pair no
// longer fits, or an error when the database misbehaves.
func linkPair(ctx context.Context, tx *sql.Tx, r TransferRow, stamp string) (string, error) {
	d, err := readLinkSide(ctx, tx, r.DebitSplitID)
	if errors.Is(err, sql.ErrNoRows) {
		return "debit split not found", nil
	}
	if err != nil {
		return "", fmt.Errorf("read split %s: %w", r.DebitSplitID, err)
	}
	c, err := readLinkSide(ctx, tx, r.CreditSplitID)
	if errors.Is(err, sql.ErrNoRows) {
		return "credit split not found", nil
	}
	if err != nil {
		return "", fmt.Errorf("read split %s: %w", r.CreditSplitID, err)
	}
	if d.linked || c.linked {
		return "already linked", nil
	}
	if !d.amount.IsNegative() || !d.amount.Round(2).Neg().Equal(c.amount.Round(2)) {
		return "amounts no longer match", nil
	}

	for _, u := range []struct{ split, other, txID string }{
		{r.DebitSplitID, r.CreditSplitID, d.txID},
		{r.CreditSplitID, r.DebitSplitID, c.txID},
	} {
		res, err := tx.ExecContext(ctx,
			`UPDATE ICTransactionSplit SET linkedSplit = ?, lastModificationDate = ? WHERE ID = ? AND COALESCE(linkedSplit,'') = ''`,
			u.other, stamp, u.split)
		if err != nil {
			return "", fmt.Errorf("link split %s: %w", u.split, err)
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return "", fmt.Errorf("link split %s: %d rows affected, expected 1", u.split, n)
		}
		res, err = tx.ExecContext(ctx,
			`UPDATE ICTransaction SET lastModificationDate = ? WHERE ID = ?`, stamp, u.txID)
		if err != nil {
			return "", fmt.Errorf("touch transaction %s: %w", u.txID, err)
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return "", fmt.Errorf("touch transaction %s: %d rows affected, expected 1", u.txID, n)
		}
	}
	return "", nil
}
```

- [ ] **Step 5: Run all icompta tests**

Run: `go test ./internal/icompta/ -v`
Expected: PASS: every new test, plus every `TestApply_*` recategorize test.

- [ ] **Step 6: Commit**

```bash
git add internal/icompta/apply.go internal/icompta/transfers_apply.go internal/icompta/transfers_apply_test.go
git commit -m "feat(icompta): apply reviewed transfer links with backup"
```

---

### Task 6: CLI command

**Files:**
- Create: `cmd/linktransfers/linktransfers.go`
- Create: `cmd/linktransfers/linktransfers_test.go` (package `linktransfers_test`, metadata)
- Create: `cmd/linktransfers/preview_internal_test.go` (package `linktransfers`, calls `runPreviewWith`)
- Modify: `main.go` (import and `root.Cmd.AddCommand(linktransfers.Cmd)` next to recategorize)

**Interfaces:**
- Consumes:
  - `icompta.OpenReadOnly`, `(*Store).TransferSnapshot`, `icompta.ResolveScope`, `icompta.EligibleLegs`, `icompta.MatchTransfers`, `icompta.NewTransferReport`, `icompta.ReadTransferReport`, `icompta.ApplyLinks`, `icompta.ApplyOptions` and `icompta.ICComptaRunning`;
  - `root.GetLogrusAdapter()`.
- Produces: `linktransfers.Cmd` (`Use: "link-transfers"`, `GroupID: "tools"`), with subcommands `preview` and `apply`.

- [ ] **Step 1: Write the failing tests**

`cmd/linktransfers/linktransfers_test.go`:

```go
package linktransfers_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/linktransfers"

	"github.com/stretchr/testify/assert"
)

func TestLinkTransfersCommand_Metadata(t *testing.T) {
	assert.Equal(t, "link-transfers", linktransfers.Cmd.Use)
	assert.Equal(t, "tools", linktransfers.Cmd.GroupID)
	names := map[string]bool{}
	for _, c := range linktransfers.Cmd.Commands() {
		names[c.Name()] = true
	}
	assert.True(t, names["preview"])
	assert.True(t, names["apply"])
}
```

`cmd/linktransfers/preview_internal_test.go`:

```go
package linktransfers

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// fixtureDB is a minimal iCompta database: Fred owns A1 and A2; S1/S2 are a
// sure transfer.
func fixtureDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ic.cdb")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	for _, s := range []string{
		`CREATE TABLE ICCategory (ID TEXT UNIQUE NOT NULL, name TEXT NOT NULL)`,
		`CREATE TABLE ICAccount (ID TEXT UNIQUE NOT NULL, name TEXT, class TEXT, parent TEXT, currency TEXT)`,
		`CREATE TABLE ICTransaction (ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, account TEXT, date TEXT NOT NULL,
			name TEXT NOT NULL, comment TEXT, amount TEXT, payee TEXT, investmentTransactionInfo TEXT, status TEXT)`,
		`CREATE TABLE ICTransactionSplit (ID TEXT UNIQUE NOT NULL, lastModificationDate TEXT, "transaction" TEXT NOT NULL,
			amount TEXT, category TEXT, linkedSplit TEXT)`,
		`INSERT INTO ICCategory VALUES ('C-VIR','Virements')`,
		`INSERT INTO ICAccount VALUES ('F','Fred','ICAccountsGroup',NULL,NULL),
			('A1','Compte privé','ICAccount','F','CHF'),('A2','Selma','ICAccount','F','CHF')`,
		`INSERT INTO ICTransaction VALUES ('T1',NULL,'A1','2026-01-10','VERS SELMA','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus'),
			('T2',NULL,'A2','2026-01-11','DEPOT','',NULL,'',NULL,'ICTransactionStatus.ClearedStatus')`,
		`INSERT INTO ICTransactionSplit VALUES ('S1',NULL,'T1','-750','C-VIR',NULL),('S2',NULL,'T2','750','C-VIR',NULL)`,
	} {
		_, err := db.Exec(s)
		require.NoError(t, err, s)
	}
	return path
}

func TestRunPreview_WritesOneSurePair(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	require.NoError(t, runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "", "", logging.NewMockLogger()))

	b, err := os.ReadFile(out)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	require.Len(t, lines, 4, "magic, state, header, one pair")
	assert.Equal(t, "# camt-csv link-transfers report v1", lines[0])
	assert.True(t, strings.HasPrefix(lines[3], "S1,S2,2026-01-10,2026-01-11,1,"))
	assert.True(t, strings.HasSuffix(lines[3], ",sure,,,yes"))
	fi, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
}

func TestRunPreview_RefusesToOverwriteWithoutForce(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	require.NoError(t, os.WriteFile(out, []byte("reviewed"), 0o600))
	err := runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "", "", logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
	b, _ := os.ReadFile(out)
	assert.Equal(t, "reviewed", string(b))

	require.NoError(t, runPreviewWith(context.Background(), db, out, true, []string{"Fred"}, "", "", logging.NewMockLogger()))
}

func TestRunPreview_UnknownFolderLeavesNoFile(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	err := runPreviewWith(context.Background(), db, out, false, []string{"Nobody"}, "", "", logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `folder "Nobody" not found`)
	assert.NoFileExists(t, out)
}

func TestRunPreview_DateBounds(t *testing.T) {
	db := fixtureDB(t)
	out := filepath.Join(t.TempDir(), "pairs.csv")
	err := runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "10.01.2026", "", logging.NewMockLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from")
	assert.NoFileExists(t, out)

	require.NoError(t, runPreviewWith(context.Background(), db, out, false, []string{"Fred"}, "2026-01-11", "", logging.NewMockLogger()))
	b, _ := os.ReadFile(out)
	assert.Len(t, strings.Split(strings.TrimSpace(string(b)), "\n"), 3, "S1 is before --from: no pair left")
}
```

- [ ] **Step 2: Run the tests and check they fail**

Run: `go test ./cmd/linktransfers/ -v`
Expected: FAIL to compile (package does not exist).

- [ ] **Step 3: Implement the command**

`cmd/linktransfers/linktransfers.go`:

```go
// Package linktransfers links both sides of transfers between the user's own
// accounts in an iCompta database, with a reviewable preview before anything
// is written.
package linktransfers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/icompta"
	"fjacquet/camt-csv/internal/logging"

	"github.com/spf13/cobra"
)

// Each subcommand owns its flag variables, as in recategorize.
var (
	previewDB   string
	outputPath  string
	forceOutput bool
	folders     []string
	fromDate    string
	toDate      string
	applyDB     string
	reportPath  string
)

// Cmd is the link-transfers command.
var Cmd = &cobra.Command{
	Use:     "link-transfers",
	Short:   "Link both sides of transfers between your own iCompta accounts",
	GroupID: "tools",
	Long: `Find transfers between your own accounts that were imported as two unrelated
transactions, and link them the way iCompta does, in two steps.

  preview  reads the database (read-only) and writes a CSV report of candidate pairs.
  apply    links the pairs you approved in that report.

Only accounts under the --folder folders are considered (any depth). A pair is
the same amount in opposite directions, in two accounts of the same currency,
at most 4 days apart. Pairs at most 2 days apart with no competing candidate are
pre-approved (apply=yes); the others are listed with apply=no and a reason.

apply refuses to run while iCompta is open, backs up the database first, and
changes only the link between the two splits: categories and amounts stay.`,
}

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Write a report of candidate transfer pairs (read-only)",
	Args:  cobra.NoArgs,
	RunE:  runPreview,
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Link the approved pairs of a preview report",
	Args:  cobra.NoArgs,
	RunE:  runApply,
}

func init() {
	previewCmd.Flags().StringVar(&previewDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	previewCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Path of the report CSV to write")
	previewCmd.Flags().BoolVar(&forceOutput, "force", false, "Overwrite the report if it already exists")
	previewCmd.Flags().StringArrayVar(&folders, "folder", nil, "iCompta folder holding your accounts (repeatable)")
	previewCmd.Flags().StringVar(&fromDate, "from", "", "Only transactions on or after this date (YYYY-MM-DD)")
	previewCmd.Flags().StringVar(&toDate, "to", "", "Only transactions on or before this date (YYYY-MM-DD)")
	_ = previewCmd.MarkFlagRequired("db")
	_ = previewCmd.MarkFlagRequired("output")
	_ = previewCmd.MarkFlagRequired("folder")

	applyCmd.Flags().StringVar(&applyDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	applyCmd.Flags().StringVar(&reportPath, "report", "", "Path of the reviewed report CSV")
	_ = applyCmd.MarkFlagRequired("db")
	_ = applyCmd.MarkFlagRequired("report")

	Cmd.AddCommand(previewCmd, applyCmd)
}

func runPreview(cmd *cobra.Command, _ []string) error {
	// MarkFlagRequired is enforced by Cobra's execute(), not when RunE is called
	// directly, so the command checks again.
	if previewDB == "" || outputPath == "" || len(folders) == 0 {
		return fmt.Errorf("--db, --output and --folder are required")
	}
	return runPreviewWith(cmd.Context(), previewDB, outputPath, forceOutput, folders, fromDate, toDate, root.GetLogrusAdapter())
}

func checkDate(flag, v string) error {
	if v == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return fmt.Errorf("%s %q is not a YYYY-MM-DD date", flag, v)
	}
	return nil
}

// runPreviewWith validates the dates, then opens the report before reading the
// database, so a bad path fails at once and a reviewed report is never
// overwritten by accident. A failed run leaves no report file.
func runPreviewWith(ctx context.Context, db, out string, force bool, folderNames []string, from, to string, log logging.Logger) (err error) {
	if err := checkDate("--from", from); err != nil {
		return err
	}
	if err := checkDate("--to", to); err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(out, flags, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("%s already exists: use --force to overwrite it", out)
		}
		return fmt.Errorf("create report: %w", err)
	}
	written := false
	defer func() {
		_ = f.Close()
		if !written {
			_ = os.Remove(out)
		}
	}()

	store, err := icompta.OpenReadOnly(ctx, db)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	snap, err := store.TransferSnapshot(ctx)
	if err != nil {
		return err
	}
	scope, err := icompta.ResolveScope(snap.Accounts, folderNames)
	if err != nil {
		return err
	}
	pairs := icompta.MatchTransfers(icompta.EligibleLegs(snap.Legs, scope, from, to))
	rep := icompta.NewTransferReport(snap.State, pairs)
	if err := rep.Write(f); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	written = true
	if err := f.Close(); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	sure := 0
	for _, p := range pairs {
		if p.Sure {
			sure++
		}
	}
	log.Info("Report written: review it, set apply to yes or no, then run apply",
		logging.Field{Key: "path", Value: out},
		logging.Field{Key: "accounts", Value: len(scope)},
		logging.Field{Key: "sure", Value: sure},
		logging.Field{Key: "doubtful", Value: len(pairs) - sure})
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
	rep, err := icompta.ReadTransferReport(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("report %s: %w", reportPath, err)
	}
	_, err = icompta.ApplyLinks(cmd.Context(), rep, icompta.ApplyOptions{
		DBPath:    applyDB,
		Now:       time.Now,
		IsRunning: icompta.ICComptaRunning,
	}, root.GetLogrusAdapter())
	return err
}
```

In `main.go`, add `"fjacquet/camt-csv/cmd/linktransfers"` to the imports next to recategorize, and register the command right after `root.Cmd.AddCommand(recategorize.Cmd)`:

```go
	root.Cmd.AddCommand(linktransfers.Cmd)
```

- [ ] **Step 4: Run the tests, the full suite and lint**

Run: `go test ./cmd/linktransfers/ -v && go test ./... && make lint`
Expected: PASS, and lint reports `0 issues.`. If `samples/pdf/viseca.pdf` is missing (a gitignored sample), only `TestConvert_PDFDirectoryConsolidates` fails, and it fails on `main` too. Report it; do not fix it.

- [ ] **Step 5: Check the CLI help**

Run: `go run . link-transfers preview --help`
Expected: lists `--db`, `--output`, `--force`, `--folder`, `--from`, `--to`.

- [ ] **Step 6: Commit**

```bash
git add cmd/linktransfers main.go
git commit -m "feat: link-transfers command (preview, apply)"
```

---

### Task 7: Documentation and changelog

**Files:**
- Create: `docs/icompta-link-transfers.md`
- Modify: `CLAUDE.md`, in the `internal/icompta` bullet of "Directory Structure"
- Modify: `CHANGELOG.md`, under `## [Unreleased]`

- [ ] **Step 1: Write `docs/icompta-link-transfers.md`**

````markdown
# Link transfers between your own iCompta accounts

When money moves between two of your accounts (BCV to Selma, BCV to Revolut…),
each bank statement brings in one side. iCompta then holds two unrelated
transactions: one counts as spending, the other as income, and budgets and
reports are inflated. `camt-csv link-transfers` finds those pairs and links them
the way iCompta does, in two steps so you review before anything is written.

## 1. Preview (read-only)

```bash
camt-csv link-transfers preview --db ~/Desktop/ic25.cdb --folder Fred -o pairs.csv
```

- `--folder` (required, repeatable) names the iCompta folder(s) holding your
  accounts. Every account under them, at any depth, is considered. Nothing else
  is. A folder name that does not exist, or exists twice, stops the command.
- `--from` / `--to` (YYYY-MM-DD) limit the run to a period, for example the month
  you just imported.

A pair is the same amount in opposite directions, in two accounts of the same
currency, at most 4 days apart, with each side the only split of its
transaction, not planned and not already linked.

- **sure** (`apply=yes`): at most 2 days apart and no competing candidate.
- **doubtful** (`apply=no`): 3-4 days apart, or several candidates. The `reason`
  column says which. Every combination is listed: set `apply=yes` on the right
  one.
- `note` says `categories differ` when the two sides are filed differently.
  Nothing is changed about categories.

Columns: `debit_split_id, credit_split_id, debit_date, credit_date, gap_days,
debit_account, credit_account, debit_name, credit_name, amount, debit_category,
credit_category, confidence, reason, note, apply`.

Transfers between CHF and EUR accounts never match (different amounts). Link
them in iCompta by hand.

## 2. Apply

Quit iCompta, then:

```bash
camt-csv link-transfers apply --db ~/Desktop/ic25.cdb --report pairs.csv
```

Apply checks that iCompta is closed and that the database has not changed since
the preview. It then saves `ic25.cdb.bak-<UTC timestamp>` next to it. Each
approved pair has its two splits pointed at each other, and their modification
dates are updated. Amounts and categories are not touched, so balances do not
change.

- A split that appears in two approved rows rejects the whole report before
  anything is written.
- A pair whose split vanished, is already linked, or no longer has matching
  amounts is skipped and logged.
- Running apply twice is refused: the first run changed the database, so preview
  again.

After the first apply, open iCompta and check one pair shows as a transfer. If
iCloud sync is on, check another device received the change.
````

- [ ] **Step 2: Update `CLAUDE.md`**

In "Directory Structure", the `icompta/` bullet ends with `See \`docs/icompta-recategorize.md\`.`. Append this sentence after it:

```
`link-transfers` (`cmd/linktransfers`) reuses the same store/backup/apply rails to link both sides of own-account transfers: scope is `--folder` only, pairs are matched by `MatchTransfers` (pure), and apply writes only `linkedSplit` and modification dates. See `docs/icompta-link-transfers.md`.
```

- [ ] **Step 3: Update `CHANGELOG.md`**

Under `## [Unreleased]`, add:

```markdown
### Added

- Add `link-transfers preview|apply` to link both sides of transfers between your own iCompta accounts, so budgets and reports stop counting them as spending and income. Scope is the iCompta folder(s) given with `--folder`; pairs at most 2 days apart with no competing candidate are pre-approved, others are listed for review; apply backs the database up and writes only the links.
```

- [ ] **Step 4: Commit**

```bash
git add docs/icompta-link-transfers.md CLAUDE.md CHANGELOG.md
git commit -m "docs: link-transfers usage, CLAUDE.md, changelog"
```

---

### Task 8: Verification on a copy of the real database (controller, not a subagent)

This task never touches `~/Desktop/ic25.cdb` itself. It needs iCompta closed only for the copy.

- [ ] **Step 1: Copy the database and record balances**

```bash
S=/private/tmp/claude-501/linkcheck && rm -rf $S && mkdir -p $S && cp ~/Desktop/ic25.cdb $S/ic25.cdb
sqlite3 -readonly $S/ic25.cdb "select t.account, round(sum(s.amount),2) from ICTransaction t join ICTransactionSplit s on s.\"transaction\"=t.ID group by 1 order by 1" > $S/before.txt
```

- [ ] **Step 2: Preview with `--folder Fred`, review the counts**

```bash
go run . link-transfers preview --db $S/ic25.cdb --folder Fred -o $S/pairs.csv
grep -c ',sure,' $S/pairs.csv; grep -c ',doubtful,' $S/pairs.csv
grep -E 'Lydie|RAUCAZ|Tobias' $S/pairs.csv | head
```

Expected:
- No row names a Lydie or Tobias account.
- The sure pairs include 3249 ↔ Selma and 3249 ↔ Revolut CHF.

Show the user the counts and 10 sample rows before going on.

- [ ] **Step 3: Apply on the copy, then check balances and BCV reconciliation**

```bash
go run . link-transfers apply --db $S/ic25.cdb --report $S/pairs.csv
sqlite3 -readonly $S/ic25.cdb "select t.account, round(sum(s.amount),2) from ICTransaction t join ICTransactionSplit s on s.\"transaction\"=t.ID group by 1 order by 1" > $S/after.txt
diff $S/before.txt $S/after.txt && echo "balances unchanged"
python3 -I /private/tmp/claude-501/stress/icrecon.py $S/ic25.cdb
sqlite3 -readonly $S/ic25.cdb "pragma integrity_check"
```

Expected:
- `balances unchanged`;
- every BCV account `months OK` 69/69 (62/62 for x3547);
- `ok`.

- [ ] **Step 4: Re-run preview: applied pairs are gone**

```bash
go run . link-transfers preview --db $S/ic25.cdb --folder Fred -o $S/pairs2.csv
grep -c ',sure,' $S/pairs2.csv
```

Expected: `0` sure pairs left. Only the doubtful pairs remain, minus any that shared a split with an applied pair.

- [ ] **Step 5: Hand over to the user**

The real run is done by the user, with iCompta closed. The user then opens iCompta and checks that one linked pair shows as a transfer, and that iCloud sync followed.
