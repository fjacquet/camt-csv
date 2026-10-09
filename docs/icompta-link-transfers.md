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
- `--from` / `--to` (YYYY-MM-DD) keep the pairs with at least one side in that
  period, for example the month you just imported. Matching still looks at every
  transaction, so a transfer that leaves on the 30th and arrives on the 1st is
  found from either month.
- `--force` overwrites an existing report (without it, preview refuses to replace
  a report you may have reviewed).

A pair is the same amount in opposite directions, in two accounts of the same
currency, at most 4 days apart, with each side the only split of its
transaction, not planned and not already linked.

- **sure** (`apply=yes`): at most 2 days apart and no competing candidate, and,
  when the two sides have different categories, one of them is Virements or
  uncategorized.
- **doubtful** (`apply=no`): 3-4 days apart, or several candidates. The `reason`
  column says which. Every combination is listed: set `apply=yes` on the right
  one.
- `note` says `categories differ` when the two sides are filed differently.
  Nothing is changed about categories.

Columns: `debit_split_id, credit_split_id, debit_date, credit_date, gap_days,
debit_account, credit_account, debit_name, credit_name, amount, debit_category,
credit_category, confidence, reason, note, apply`.

Transfers between a CHF and a EUR account are never proposed: the two accounts
have different currencies. Link them in iCompta by hand.

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
