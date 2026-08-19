# ADR-021: CLI Command Restructure

## Status

Accepted — planned for v3.0.0 (breaking)

Extends ADR-017 (Input Auto-Detection), which moved file-vs-folder routing off
the CLI surface. This ADR applies the same reasoning to format routing.
Amends ADR-013 (Universal Batch Processor): a directory now yields one
consolidated CSV instead of one CSV per input file.

## Context

The root command namespace holds ten commands (`main.go:52-61`) that mix three
unrelated things:

| Command | What it actually is |
|---|---|
| `convert` | The primary function: auto-detecting conversion, covers every format |
| `camt`, `pdf`, `selma`, `viseca`, `revolut`, `revolut-crypto`, `revolut-investment`, `debit` | Eight variants of `convert` with the format pinned |
| `categorize` | An accessory: looks up the category for a single party name, converts nothing |

Three concrete defects follow:

1. **Mixed grammar.** `convert` and `categorize` are verbs. `camt`, `pdf` and
   `debit` are format nouns. Same level, different semantics.
2. **Eight tenths of the namespace is one concept.** Their only value over
   `convert` is rejecting a file instead of guessing at it
   (`cmd/convert/convert.go:31-32`). That is a flag, not eight root commands.
3. **`categorize` is not in the pipeline.** It is a diagnostic that inspects the
   YAML category database.

`cmd/root/root.go:41` also still describes the tool as converting CAMT.053 files
only, three formats out of date.

Two of the eight format commands are not pure redundancy and must be absorbed
rather than dropped:

- **`pdf`** consolidates a directory of PDFs into a *single* chronologically
  sorted CSV (`cmd/pdf/convert.go:88-104`). Every other command writes one CSV
  per input file via `BatchProcessor`.
- **`viseca`** owns `--keep-payments` (`cmd/viseca/convert.go:25-27`), already
  read back by `root.applyFlagOverrides`.

### Constraints

- The tool is used interactively only. No scripts, cron jobs or Makefiles call
  it, so a clean break needs no deprecation aliases.
- The purpose is mass processing: many statements in, one import file out.

## Decision

### Target CLI surface

```
camt-csv convert    -i <path> -o <file.csv> [flags]
camt-csv categorize --party "Migros" [flags]
```

Help output is grouped with `cobra.Group` (available in cobra v1.10.2):

```
Conversion:
  convert      Convert a statement to CSV, detecting its format

Tools:
  categorize   Look up the category a party name resolves to

Additional Commands:
  completion, help
```

Two commands instead of ten. The root namespace contains verbs only. Primary and
accessory stay distinct by group, not by nesting.

### Flags on `convert`

Root persistent flags are unchanged: `-i`, `-o`, `-v`, `--config`,
`--log-level`, `--log-format`, `--csv-delimiter`, `--ai-enabled`,
`--auto-learn`.

| Flag | Role | Origin |
|---|---|---|
| `--from <format>` | Pin the parser, bypassing detection | New — absorbs the eight commands |
| `--format`, `-f` | **Output** format (`icompta` / `standard` / `jumpsoft`) | Unchanged |
| `--recursive` | Descend into subdirectories | Unchanged |
| `--keep-payments` | Keep Viseca card settlement rows | Unchanged, promoted to `convert` |
| `--date-format` | Deprecated, no effect | Unchanged |

`--from` accepts `camt`, `pdf`, `revolut`, `revolut-crypto`,
`revolut-investment`, `selma`, `debit`, `viseca`. The list is derived from
`container.DetectionOrder()` so it cannot drift from the registered parsers.

When the input is a directory, `--from` pins the parser for **every** file in
the batch; files the pinned parser cannot read fail individually and are
recorded in the manifest. It is an escape hatch for a misdetecting batch, not a
filter that selects matching files.

`--format` keeps its current meaning (output format). Renaming it into a
symmetric `--from`/`--to` pair was considered and rejected: symmetry is not worth
breaking the flag the user types most.

### Directory semantics: always one output file

A directory input always produces a single CSV. There is no per-file output mode
and no flag to request one.

This removes the `-o` ambiguity: `-o` is always a file. If the given path is an
**existing directory**, the output is written to
`<dir>/<source-dir-basename>.csv` — preserving the convenience of the current
PDF code path rather than failing.

`--recursive` remains as the only remaining axis: which files are read. It is
orthogonal to output shape, not a duplicate of it.

Files of different formats in one directory are detected individually and merged
into the same output, sorted by date. This is the intended workflow: drop every
statement into a folder, get one iCompta import.

### Data flow

```
convert -i <path> -o <output.csv>
  |
  +- path is a file
  |     DetectParser (or --from) -> Parse -> Formatter -> output.csv
  |
  +- path is a directory
        discoverFiles(--recursive)
        for each file: DetectParser (or --from) -> Parse -> []Transaction
                       failure -> recorded in manifest, processing continues
        AggregateTransactions -> chronological sort + duplicates reported
        Formatter              -> output.csv
                               -> output.manifest.json
```

The pivot is that `processFile` no longer **writes**; it **returns**
`[]models.Transaction`. Writing becomes a single call at the end. That is what
makes all the output-naming code fall away.

### Reuse: `BatchAggregator` is already written

`internal/batch/aggregator.go` already implements exactly this consolidation:
`AggregateTransactions` sorts chronologically and detects duplicates. It is dead
code in production — its only caller is
`internal/integration/cross_parser_test.go:166`. This design wires it up rather
than writing a new consolidation path.

### Partial failure: write anyway

One unreadable file out of forty must not discard the other thirty-nine. The CSV
is written with what succeeded, the exit code reports partial success, and the
manifest names the failures so the user knows what to re-run.

Exit codes: `0` all succeeded, `1` partial, `2` all failed or no files found —
`manifest.ExitCode()`. This was planned as unchanged, but two branches were
added during implementation that this design did not anticipate: the
pre-restructure `convert` command's directory mode never called an exit-code
function at all (always 0, regardless of failures), and a batch that read
files without error but produced zero transactions now also exits 2. Both are
deliberate changes made while implementing this ADR, not something this
document originally specified.

The manifest replaces the output's `.csv` extension rather than appending to it:
`-o releves/2024.csv` writes `releves/2024.csv` and `releves/2024.manifest.json`.
This replaces the current fixed `<outputDir>/.manifest.json`
(`internal/batch/processor.go:117`), which has no directory to live in once the
output is a file. A single-file conversion writes no manifest: its outcome is
the exit code, and a one-entry report adds nothing.

### Duplicates: report, never remove

`detectAndLogDuplicates` logs and does not delete. That stays. Merging mixed
formats is exactly where overlaps appear — a Viseca PDF alongside the Viseca CSV
export of the same month. Removing on a similarity heuristic would eventually
erase two genuine identical purchases made on the same day. iCompta runs its own
duplicate detection at import.

### New guard: output must not live under the input directory

`sameDirectory` (`cmd/convert/convert.go:213-226`) currently prevents writing
into the input directory. It disappears with the per-file mode, but the hazard
returns in another form: with `-o releves/2024.csv` pointing inside the folder
being read, a second `--recursive` run would read its own output back as input.
`2024.csv` carries no header any validator accepts, so today it would merely be
logged as unrecognized — noise now, and a corruption path the day a formatter
emits a CSV some parser does accept. Refuse it explicitly.

### Error handling

| Situation | Behavior |
|---|---|
| `--from <unknown>` | Immediate cobra error listing valid values from `container.DetectionOrder()` |
| Single file not recognized | Fatal, listing supported formats (current behavior) |
| File not recognized **within a batch** | Warn, `success:false` manifest entry, continue |
| `-o` missing for a directory input | Fatal (current behavior) |
| A parser fails on one file of a batch | Warn, manifest entry, continue |
| Output path falls under the input directory | Refused |

## Consequences

### Code impact

| File | Action |
|---|---|
| `main.go` | Ten `AddCommand` calls become two |
| `cmd/camt/`, `cmd/pdf/`, `cmd/selma/`, `cmd/viseca/`, `cmd/revolut/`, `cmd/revolut-crypto/`, `cmd/revolut-investment/`, `cmd/debit/` | Deleted |
| `cmd/convert/convert.go` | Sole handler; `outputPathFor`, `sameDirectory` and `discoverInputs` deleted as duplicates of `BatchProcessor` |
| `cmd/common/convert.go` | `RunConvert` deleted; `FolderConvert` rewritten to aggregate |
| `cmd/common/flags.go` | Adds `--from` and `--keep-payments`; `--format`, `--recursive`, `--date-format` unchanged |
| `internal/batch/processor.go` | `ProcessDirectory` writes one CSV; `outputPathFor` and `claimed` deleted; `processFile` returns transactions |
| `internal/batch/aggregator.go` | Wired into production |
| `cmd/root/root.go` | Cobra groups; `Short` and `Long` corrected |

### Testing

Deleted along with their packages (~12 files): `cmd/camt/*_test.go`,
`cmd/debit/`, `cmd/pdf/`, `cmd/revolut/`, `cmd/revolut-crypto/`,
`cmd/revolut-investment/`, `cmd/selma/`. They mostly asserted that a command
exists and registers its flags; that coverage moves into `cmd/convert`,
parameterized over `--from`.

To write:

1. `--from` accepts every value of `DetectionOrder()` and rejects an unknown one
2. `--from` genuinely bypasses detection — a Revolut file forced to `--from selma`
   must fail parsing, not silently fall back to the right parser
3. Mixed-format directory produces one CSV holding every transaction, date-sorted
4. Partial failure writes the CSV of the successes, exits `1`, and names the
   failure in the manifest
5. Empty directory exits `2`
6. `-o` pointing at an existing directory generates a name inside it
7. `--recursive` puts the whole tree into the single CSV
8. **PDF consolidation regression** — the behavior of `cmd/pdf` disappears as a
   command and must be proven equivalent through `convert`
9. Output under the input directory is refused
10. Help output shows the groups and none of the eight removed commands

Unchanged and more load-bearing than before, now that detection is the only path:
`TestDetectParser_ValidatorsDoNotOverlap`. Also unchanged:
`TestIComptaHeaderCoversPluginMappings`.

### Migration

Breaking, so **v3.0.0**.

- `CHANGELOG.md`: `Removed` (eight commands), `Changed` (directory input now
  produces a single file)
- `README.md` and `docs/`: every `camt-csv camt -i …` example becomes
  `camt-csv convert -i …`
- `CLAUDE.md`: the "Adding a New Parser" section loses step 5 ("Add CLI command
  in `cmd/{name}/convert.go`"), which no longer exists

### Rejected alternatives

- **Noun-verb namespaces** (`camt-csv category test --party …`). Extensible if
  YAML database management ever grows a CLI, but `category` would have exactly
  one member today, and it costs two words for a rarely-used diagnostic. YAGNI.
- **Root command as convert** (`camt-csv statement.xml -o out.csv`). Shortest to
  type, but positional args on the root alongside subcommands are ambiguous — a
  file named `categorize` resolves as a command. Fragile for one saved word.
- **Backward-compatible aliases** (hidden commands plus deprecation warnings).
  Unnecessary: usage is interactive only, so nothing breaks silently.
- **Keeping per-file output behind `--split`.** A flag that is never used is a
  flag that must still be maintained.
