# ADR-013: Universal BatchProcessor via Composition

## Status
Accepted — superseded in v1.4 (batch removed; folder detection built into every command)

## Context

Before v1.2, batch processing was implemented ad-hoc: the `batch` subcommand held its own directory-walking logic and called each parser separately. The PDF parser had its own `--batch` flag with separate consolidation logic. This meant:

- 3 separate batch code paths with inconsistent error reporting
- No manifest output for batch runs
- Exit codes didn't distinguish "all failed" from "some failed"
- Adding batch support to a new parser required duplicating batch logic

## Decision (v1.2)

Extract batch processing into a single `BatchProcessor` struct that any parser can compose:

```go
// internal/batchprocessor/processor.go
type BatchProcessor struct {
    parser    parser.FullParser
    formatter formatter.OutputFormatter
    logger    logging.Logger
}

func (bp *BatchProcessor) Process(inputDir, outputDir string) BatchResult
```

`BatchResult` carries per-file success/failure status and is written as a manifest JSON file. Exit codes are semantic:
- `0` — all files succeeded
- `1` — some files failed (partial success)
- `2` — all files failed

Each parser's adapter implements `BatchConvert(inputDir, outputDir string) (int, error)` by constructing a `BatchProcessor` — one line of delegation.

## Consequences

**Positive:**
- Consistent error reporting and manifest across all parsers
- Semantic exit codes enable reliable scripting (`if camt-csv batch ...; then ...`)
- New parsers get batch support by implementing the adapter interface

**Negative:**
- `BatchProcessor` couples parsers to a specific directory-in/directory-out contract

## Supersession Note (v1.4)

The `batch` subcommand and `--batch` flags are removed in v1.4. Folder detection is now built into every command directly: passing a folder as input triggers multi-file processing automatically. The `BatchProcessor` internal infrastructure may be reused or removed during v1.4 implementation.

## Future Work

~~Recursive directory scanning (not needed for current flat-directory use case)~~
Superseded by v1.4 folder-detection approach.
