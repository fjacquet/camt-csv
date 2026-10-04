# Recategorize an iCompta database

`camt-csv recategorize` runs the categorizer over transactions already stored in
iCompta, in two steps so you can review before anything is written.

## 1. Preview (read-only)

```bash
camt-csv recategorize preview --db ~/Desktop/ic25.cdb -o report.csv
```

iCompta may stay open for this step; the database is opened read-only. The AI
tier is limited to about 5 requests a minute, so a first run can take tens of
minutes. Progress is logged every 100 splits. Ctrl-C stops cleanly and keeps the
rows decided so far. The report file is created first, so a bad path fails at
once; an existing report is never overwritten unless you pass `--force`.

The report has one row per proposed change. Columns:
`split_id,date,name,amount,old_category,new_category,tier,decision,reason,apply`.
Set `apply` to `no` on any row you reject, or delete the row. Only rows with
`decision=change` and `apply=yes` are written. Text that would be read as a
spreadsheet formula (a bank label starting with `=`, `+`, `-` or `@`) is stored
with a leading apostrophe; it is removed again when the report is read.

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
