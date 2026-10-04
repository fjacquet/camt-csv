# iCompta recategorization — design

Date: 2026-10-04
Status: draft, awaiting review

## Purpose

Run camt-csv's four-tier categorizer over transactions already stored in an
iCompta database (`ic25.cdb`) to fill in missing categories and replace
low-quality ones, with a human-reviewable preview before anything is written.

Success means: after `apply`, splits that were uncategorised or filed under a
"don't know" category carry a sensible category; categories the user chose
deliberately are untouched unless a deterministic tier disagrees; every change
is traceable in a log and reversible from a backup.

### What was said

- Scope: the whole database, not only the uncategorised splits.
- Overwrite rule: a category is replaced only when the decision comes from a
  deterministic tier (direct mapping, keyword). AI and semantic results may only
  fill an empty or "unknown" category.
- A preview report is required before any write.
- Logging of every change is required.

### Assumptions (correct me)

- Investment splits (`ICTransaction.investmentTransactionInfo` set) and linked
  splits (`ICTransactionSplit.linkedSplit` set, i.e. transfers between accounts)
  stay as they are. They are legitimately uncategorised: 444 of the 723 empty
  splits in the current database are investment operations, ~40 are transfers.
- "Unknown" categories are `Divers`, `Non Classé`, `Autre`, `Uncategorized`,
  `Uncategorized (AI)`.
- Writing directly into the iCompta SQLite file is acceptable with the safeguards
  below. iCloud sync is active on this database; whether iCloud propagates an
  externally written change cannot be verified from here (see Risks).

## Commands

```text
camt-csv recategorize preview --db ic25.cdb -o report.csv
camt-csv recategorize apply   --db ic25.cdb --report report.csv
```

`preview` never modifies the database. `apply` writes only what the report says,
so the reviewed file is exactly what is applied; there is no second AI pass and
no run-to-run drift.

## Components

New package `internal/icompta/`, command in `cmd/recategorize/`.

| Unit | Responsibility | Depends on |
|---|---|---|
| `reader` | Open the DB read-only, return `Candidate` rows (split ID, transaction name, payee, comment, split amount, current category ID and name). Load the `ICCategory` name↔ID map. | SQLite driver |
| `selector` | Keep a split if it is empty or "unknown", or may be overridden by a deterministic tier. Drop investment and linked splits. | `reader` types |
| `recategorizer` | Build a `models.Transaction` per candidate and call the existing `Categorizer`; return category plus `Source`. Auto-learn writes are disabled; AI suggestions still go to the staging files. | `categorizer` |
| `policy` | Pure function: `(current, proposed, source) -> Decision` implementing the overwrite rule. | none |
| `report` | Write and read the report CSV. | none |
| `applier` | Guards, backup, transaction, logging. | SQLite driver, `report` |

`policy`, `selector` and `report` are pure and tested without a database.

### Candidate to transaction mapping

- `PartyName` is `ICTransaction.payee` when non-empty, else `ICTransaction.name`.
  In the current data `payee` is mostly empty and `name` carries the bank label.
- `IsDebtor` is `split amount < 0`, the same convention the Revolut investment
  parser uses.
- Amount, date and `comment` are passed as the categorizer's extra context.
- The category name returned by the categorizer is resolved to an `ICCategory`
  ID. A name that does not exist in `ICCategory` is never created: the row is
  reported as `skipped: unknown category`.

## Policy

| Current category | Proposed from | Decision |
|---|---|---|
| empty or "unknown" | any tier, not uncategorised | change |
| empty or "unknown" | no result | keep, report `no suggestion` |
| a real category | `direct_mapping` or `keyword`, different | change |
| a real category | `semantic` or `ai` | keep (never overwrites) |
| a real category | any tier, same | keep, not reported |

## Report format

CSV, one row per candidate that is not "keep, same":

`split_id, date, name, amount, old_category, new_category, tier, decision, apply`

`decision` is `change`, `keep` or `skipped: <reason>`. `apply` is `yes` for
`change` rows and `no` otherwise. The user edits `apply` (or deletes rows) before
running `apply`.

## Apply safeguards

In order, aborting on the first failure with nothing written:

1. Refuse if iCompta is running (process check).
2. Refuse if the report was produced for a different database (the report header
   carries the DB file size and the max `lastModificationDate` seen at preview;
   a mismatch means iCompta changed the file since).
3. Copy `ic25.cdb` to `ic25.cdb.bak-<timestamp>` and verify the copy.
4. `PRAGMA integrity_check` before.
5. One SQL transaction. For each `apply=yes` row, update
   `ICTransactionSplit.category` and `lastModificationDate` (UTC,
   `YYYY-MM-DD HH:MM:SS`, the format iCompta writes) only if the split's current
   category still equals the report's `old_category`. Otherwise skip and log.
6. `PRAGMA integrity_check` after; roll back on failure.

## Logging

- `INFO` per change: split ID, name, `old -> new`, tier.
- `WARN` per skipped row with the reason (`category changed since preview`,
  `unknown category`, `apply=no`).
- `INFO` summary: counts by decision and by tier, backup path, duration.
- `preview` logs candidates read, selector exclusions by reason, and a tier
  histogram.

## Dependency

Add `modernc.org/sqlite` (pure Go, no CGO) so the GoReleaser build stays
CGO-free. Verified against its documentation before use.

## Out of scope

- Creating or merging categories in iCompta, and editing import mappings.
- Touching iCompta rules (`LARule`), scheduled transactions, or investment data.
- Auto-learning into `database/*.yaml` from this command.

## Risks

- **iCloud sync.** Direct writes bypass iCompta's own change tracking. Setting
  `lastModificationDate` mirrors what iCompta does, but propagation to other
  devices is unverified. Mitigation: documented in the command help; after the
  first apply, check a second device. The backup allows rollback.
- **AI rate limit.** The default is 5 requests/minute. The categorizer already
  deduplicates by `party|isDebtor` within a run, but bank labels often embed
  dates and references, so many names will be unique. A first preview can take
  tens of minutes. The preview logs progress and honours cancellation.
- **Schema drift.** The reader and applier check the columns they use exist and
  fail with a clear message otherwise.

## Testing

- Fixture SQLite database built inside each test with the minimal schema
  (`ICCategory`, `ICTransaction`, `ICTransactionSplit`). The real `ic25.cdb` is
  never used in tests.
- `policy`: table-driven over every row of the policy table.
- `selector`: investment split, linked split, unknown category, empty category.
- `report`: round trip, hand-edited `apply`, malformed row.
- `applier`: success on a temp copy; stale `old_category` skipped; unknown
  category skipped; failed integrity check rolls back; refused when the DB-state
  header mismatches.
- `recategorizer`: fake `TransactionCategorizer` returning each `Source`.
