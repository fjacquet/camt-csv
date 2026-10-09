# link-transfers: link both sides of own-account transfers in iCompta

Status: approved design (2026-10-09), spec awaiting review.

## Problem

When money moves between two of the user's own accounts (e.g. BCV x3249 → Saxobank Selma,
x3249 → Revolut CHF), the two statements are imported separately. iCompta then holds two
unrelated transactions, usually both categorized "Virements": one an expense, one an income.
Balances are right, but budgets and reports count the move as spending on one side and
revenue on the other. On the 2025-10..2026-09 data this inflated "Virements" to about
-3,970 CHF/month in the budget view.

iCompta models a transfer as two splits that point at each other through
`ICTransactionSplit.linkedSplit`. Linked transfers are left out of income and spending.
The user links them by hand today; there are about 186 unlinked candidate pairs since 2021.

## Goal

A `camt-csv link-transfers` command that finds candidate pairs, lets the user review them in a
CSV report, and writes the links the way iCompta does, safely, after each monthly import.

Success:
- every applied pair is linked both ways exactly like iCompta's own links;
- no pair is linked without being marked `apply=yes` in the reviewed report;
- account balances are unchanged (only `linkedSplit` and modification dates change).

## Non-goals

- Cross-currency transfers (CHF ↔ EUR): amounts never match exactly; they stay manual.
- Changing categories. Existing iCompta links carry the same category on both sides; when a
  candidate pair's categories differ, the report shows both and nothing is changed.
- Splitting or merging transactions; transactions with more than one split are out of scope.
- Linking to accounts outside the folders given at run time (see Scope).

## Pattern

Same two-step shape and safety rails as `camt-csv recategorize` (`internal/icompta`,
`docs/icompta-recategorize.md`):

1. `link-transfers preview` opens the database read-only (iCompta may stay open) and writes a
   CSV report.
2. The user edits the `apply` column.
3. `link-transfers apply` (iCompta closed) writes the reviewed links.

Reused as is: `OpenReadOnly`, `State` (data-state marker), `ICComptaRunning`, the WAL check,
`copyVerified` backup, `integrityCheck`, `iComptaTimeFormat`, `csvsafe.Escape`, and the
report's magic/state header lines. New code lives next to them in `internal/icompta`
(transfer-specific files) and a new `cmd/linktransfers` command.

## Scope: which accounts

- `--folder NAME` (required, repeatable) names the iCompta folders (`ICAccount` rows with
  `class = 'ICAccountsGroup'`) to work in. The scope is every account (`class = 'ICAccount'`)
  under those folders, at any depth, through the `ICAccount.parent` chain. There is no config
  setting and no exclude list: the user keeps the accounts they manage under one folder
  ("Fred") and passes it at run time.
- `ICPerson` rows ("Personnes") are never candidates.
- A `--folder` name that matches no folder is an error. Two folders with the same name are an
  error that names both paths: a silent wrong scope is worse than a stop.
- `--from` / `--to` (ISO dates, inclusive) keep the pairs with at least one leg dated in the
  range. Matching runs on every eligible leg first, so boundary-crossing transfers are found
  and ambiguity counts are complete. Default: the whole database.
- `max_days` (4) and `sure_days` (2) are constants in v1, not settings.

## Matching rules (pure function, unit-tested)

Input: the candidate splits. A split qualifies when:
- its transaction is in an included account;
- the transaction's status is not Planned;
- the split's `linkedSplit` is empty;
- its amount is not zero;
- its transaction has exactly one split.

Two qualifying splits A (amount < 0) and B (amount > 0) form a candidate pair when:
- they are in different accounts with the same currency (`ICAccount.currency`);
- `|A.amount| = |B.amount|`, compared as decimals rounded to cents;
- their transaction dates are at most 4 days (`max_days`) apart.

Confidence:
- **sure**: gap ≤ 2 days (`sure_days`), and A has exactly one candidate B, and B exactly one candidate A.
  Report `apply=yes`.
- **doubtful**: any other candidate pair. Report `apply=no` and a reason. Reasons:
  - `gap N days`;
  - `N candidates for the debit`;
  - `N candidates for the credit`;
  - `categories differ`.

  Several reasons are joined with `; `. Every combination of an ambiguous group is listed so
  the user can pick one.

"categories differ" is shown in the `note` column. It makes a pair doubtful (reason
`categories differ, neither is a transfer`) only when neither side is "Virements" nor an
unknown/empty category: on real data this caught card purchases that coincided with Revolut
pocket moves.

Ordering: by debit date, then debit account name, then amount. This is deterministic, so two
previews of the same database give the same report.

## Report format

Lines:
1. The same magic first line as the recategorize report, with a different kind tag:
   `# camt-csv link-transfers report v1`.
2. `# db_state=<State()>` (same key as recategorize).
3. A CSV header, then one row per candidate pair.

Columns:

`debit_split_id, credit_split_id, debit_date, credit_date, gap_days, debit_account,
credit_account, debit_name, credit_name, amount, debit_category, credit_category, confidence,
reason, note, apply`

Text columns pass through `csvsafe.Escape`; reading strips the escape. The reader tolerates
the same spreadsheet damage as the recategorize reader: BOM, CRLF, trailing commas on comment
lines, any case of yes/no. A recategorize report given to `link-transfers apply` (or the
reverse) is rejected by the magic line.

The report file is created before reading the database, so a bad path fails at once. An
existing file is not overwritten without `--force`, as in recategorize.

## Apply

Pre-checks, in order, all as in recategorize:
1. iCompta not running.
2. The `-wal` file is empty or absent.
3. Data state equals the report's `db_state`; otherwise stop with "run preview again".
4. Verified backup `ic25.cdb.bak-<UTC>`.
5. `integrity_check` = ok.

Then, in one write transaction, for each row with `apply=yes`:
- **Skip** (log a warning, count it) when:
  - either split no longer exists;
  - either split already has a `linkedSplit`;
  - the two amounts are no longer opposite and equal.
- **Reject the whole report before writing anything** when a split id appears in more than one
  `apply=yes` row: one split can only be linked once.
- Otherwise:
  - set `A.linkedSplit = B.ID` and `B.linkedSplit = A.ID`;
  - set `lastModificationDate` to the apply timestamp (`iComptaTimeFormat`, UTC) on both splits
    and both parent transactions, so iCloud sync picks the change up.

Commit, then run `integrity_check` again; on failure, report the backup path to restore.

Output: linked, skipped (with reasons), backup path. Every link is logged with both dates,
accounts and the amount.

## CLI

```
camt-csv link-transfers preview --db ~/Desktop/ic25.cdb --folder Fred -o pairs.csv [--from 2026-09-01] [--to 2026-09-30] [--force]
camt-csv link-transfers apply   --db ~/Desktop/ic25.cdb --report pairs.csv
```

`--db` is required, as in recategorize. The command needs no categorizer, AI or network, so it must
not build the categorizer or warm up embeddings. Documentation:
`docs/icompta-link-transfers.md`, plus a CLAUDE.md line under `internal/icompta`.

## Testing

- **Matching, table-driven pure tests:**
  - sure pair;
  - 3- and 4-day gaps (doubtful), 5-day gap (no pair);
  - two credits for one debit (both listed, doubtful);
  - different currencies (no pair);
  - account outside the given folders, account nested two levels under one, `ICPerson` (no pair);
  - already-linked split;
  - Planned transaction;
  - multi-split transaction;
  - zero amount;
  - a cents-rounding case;
  - deterministic ordering.
- **Scope resolution:** an account two levels under the folder is included; an account in
  another folder is not; an unknown folder name is an error; a duplicate folder name is an
  error naming both paths.
- **Report:** write/read round trip, formula escaping, BOM/CRLF tolerance, wrong magic rejected.
- **Apply, on fixture databases built in the test (never the real file):**
  - links are set both ways and modification dates updated;
  - state mismatch refuses;
  - a split linked since the preview is skipped;
  - a duplicate split id across `apply=yes` rows rejects the report;
  - backup written and verified;
  - balances (sum of splits per account) unchanged.
- **Before the first real use:**
  1. Run preview and apply on a copy of `~/Desktop/ic25.cdb`.
  2. Every account balance must equal its pre-apply value.
  3. The BCV month-end reconciliation against CAMT CLBD must still be 69/69 (62/62 for x3547).
  4. Re-run preview: the applied pairs must be gone.

## Risks

- **False pairs.** Two unrelated same-amount moves can fall within a few days of each other.
  This is why only `sure` pairs are pre-marked and every row is reviewable.
- **iCompta-side behaviour of a hand-written link.** Mitigated by mirroring existing links
  exactly (both directions, same columns) and the copy-first verification above. The user
  should also open iCompta after the first apply and check one linked pair displays as a
  transfer.
- **iCloud sync.** As in recategorize: after the first apply, check another device received
  the change.
