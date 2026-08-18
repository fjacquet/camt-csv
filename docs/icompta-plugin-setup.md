# iCompta Import Plugin Setup

camt-csv's `icompta` output format is consumed by import plugins configured
inside the iCompta document. This document records how the two fit together and
what has to stay in sync.

## How iCompta reads our CSV

Plugin configuration lives in the `ICImportPlugin` table of the iCompta document
(`.cdb`, a plain SQLite database). Three columns matter:

| Column | Meaning |
| --- | --- |
| `CSV_separator` | Field delimiter the plugin expects |
| `CSV_hasHeader` | `1` on every plugin, so columns resolve **by name** |
| `transactionsMapping` | JSON of iCompta field -> CSV column **name** |

Because matching is by name and not by position, **appending columns to the
`icompta` output is safe**: a plugin that does not reference a new column simply
ignores it. The inverse is not safe — if the formatter stops emitting a column a
plugin maps, the mapping resolves to nothing and iCompta drops that field on
import **without reporting an error**.

That silent-drop behaviour is why `TestIComptaHeaderCoversPluginMappings` exists.

## Keeping the reference in sync

`.planning/reference/icompta-import-plugins.txt` is generated from the live
document and is what the tests assert against. Regenerate it whenever plugins
change:

```bash
sqlite3 ~/Desktop/ic25.cdb \
  "SELECT name,CSV_separator,dateFormat,encoding,transactionsMapping
   FROM ICImportPlugin WHERE name LIKE 'CSV-%' ORDER BY name;"
```

Preserve the comment header when rewriting the file; the parser skips `#` lines.

## Editing the document safely

**Quit iCompta before touching the `.cdb`.** The app holds the database open and
will overwrite external edits on its next save, or corrupt the file. Always take
a backup first:

```bash
cp ~/Desktop/ic25.cdb ~/Desktop/ic25.cdb.bak-$(date +%F)
```

Editing plugins through iCompta's own UI avoids the issue entirely and is the
better route when only a few fields change.

## Required plugin configuration

Every CSV plugin consuming camt-csv output must use:

- **Separator**: semicolon — matches `iComptaFormatter.Delimiter()`
- **Date format**: `dd.MM.yyyy` — output dates are always `models.DateFormatCSV`
- **Encoding**: UTF-8, with a header row

## Column semantics

Some columns carry constraints that are not obvious from the name:

- **`NumberOfShares`** is blank, never `0`, when a transaction is not an
  investment. iCompta treats a literal `0` as real data and creates phantom
  zero-share securities. `Fees` follows the same rule.
- **`NumberOfShares`** keeps its natural precision (`0.4523`, not `0`), because
  brokers allocate fractional units. It is a `decimal.Decimal`, like every other
  numeric field on `Transaction`.
- **`BookkeepingNumber`** is the source system's stable per-transaction
  identifier and is what plugins should map to `externalID` and `number`. It lets
  iCompta deduplicate re-imported statements.
- **`Number`** is *not* a stable identifier. `TransactionBuilder` mints a random
  UUID into it, so it changes on every export. Never map it to `externalID` or
  `number`.

## externalID coverage by parser

| Parser | `BookkeepingNumber` source | Stable across re-exports |
| --- | --- | --- |
| CAMT | account servicer reference (`657697337026085/1`) | yes |
| Selma | `Bookkeeping No.` column (`55026832483`) | yes |
| Revolut | none — source has no identifier | n/a |
| Viseca (CSV) | `TransactionId` (`TRX2025030300002685972`) | yes |
| Viseca (PDF) | none — source has no identifier | n/a |

Revolut imports, and Viseca imports taken from PDF rather than CSV, therefore rely on iCompta's own reconciliation
(`reconcile`, `reconcileUsingDate`, `numberOfDays`) rather than `externalID`.
Consider enabling `reconcileUsingName` for those two, since date-only matching
within a three-day window is a loose guard against duplicates.
