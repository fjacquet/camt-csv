# ADR-017: Input Auto-Detection (File vs Folder)

## Status
Accepted — planned for v1.4

## Context

After v1.2 introduced the `BatchProcessor` (ADR-013), the CLI had two separate surfaces for single-file and multi-file processing:

```bash
camt-csv camt input.xml --output result.csv        # single file
camt-csv batch --input ./statements/ --output ./out/  # folder
```

This creates unnecessary cognitive load: users must remember which subcommand to use based on whether their input is a file or a folder. The `batch` subcommand also has different flag names (`--input` vs positional arg) creating inconsistency.

The root cause: routing logic (file vs folder) was placed at the CLI surface instead of inside the command.

## Decision

Remove the `batch` subcommand and `--batch` flag entirely. Each parser command detects input type at runtime:

```bash
camt-csv camt input.xml --output result.csv          # single file
camt-csv camt ./statements/ --output ./out/          # folder → processes all .xml files
```

**Rules:**
1. If input path is a file → single-file mode (unchanged behavior)
2. If input path is a folder → folder mode: process all matching files (non-recursive, extension-filtered)
3. Folder mode **requires** `--output` to be a directory path; error if omitted
4. PDF folder mode consolidates all PDFs into one CSV (matching prior `--consolidate` behavior)

**Implementation location:** `cmd/common/convert.go` — the shared `RunConvert` function detects input type and routes accordingly. Each parser command calls `RunConvert` unchanged.

**`--format` default change:** `icompta` becomes the default (no flag needed for typical use). `--format standard` is the explicit override.

## Consequences

**Positive:**
- One command surface regardless of input shape — zero CLI learning curve
- `batch` subcommand removal reduces command surface area by one
- Consistent flag names across all parsers (no more `--input` vs positional)
- iCompta users never need to pass `--format` for typical use

**Negative:**
- **Breaking change:** existing scripts using `camt-csv batch` or `--batch` will fail
- PDF consolidation is now the only folder behavior for `pdf` — independent per-file PDF output requires processing files one by one

## Migration

```bash
# Before (v1.3)
camt-csv batch --input ./statements/ --output ./out/ --format icompta

# After (v1.4)
camt-csv camt ./statements/ --output ./out/
```

## Future Work

- Recursive folder scanning if use case emerges
- Glob pattern support for selective folder processing (`./statements/*.xml`)
