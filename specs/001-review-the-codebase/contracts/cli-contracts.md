# CLI Contracts: Codebase Compliance Review

## Command: `camt-csv review`

### Description
Performs an automated compliance review of specified codebase sections against defined constitution principles. Generates a structured report detailing compliance status and identified non-compliant areas.

### Usage
`camt-csv review [path...] [flags]`

### Arguments
- `path` (required, variadic): One or more absolute paths to files or directories to be reviewed. Supports glob patterns.

### Flags
- `--constitution-files` (string array, optional): Paths to one or more constitution definition files (e.g., YAML, TOML). If not provided, the tool will attempt to discover constitution files in a default location (e.g., `.camt-csv/constitution.yaml`).
- `--principles` (string array, optional): A comma-separated list of specific constitution principle IDs to apply during the review. If not provided, all principles from the loaded constitution files will be applied.
- `--output-format` (string, optional): The desired output format for the compliance report. Supported values: `json`, `xml`. Default: `json`.
- `--git-ref` (string, optional): A Git reference (e.g., commit hash, branch name) to compare the current codebase against for a diff-based review. If provided, the report will highlight changes relative to this reference.
- `--output-file` (string, optional): The absolute path to a file where the compliance report should be written. If not provided, the report will be printed to `stdout`.

### Example
```bash
camt-csv review /path/to/src/file.go /path/to/src/module/ --constitution-files .camt-csv/constitution.yaml --principles "GO-001,GO-002" --output-format json --output-file compliance_report.json
```

### Output (JSON Example)
```json
{
  "reportId": "uuid-v4",
  "timestamp": "2025-10-12T10:00:00Z",
  "codebaseSection": [
    {
      "path": "/path/to/src/file.go",
      "type": "File"
    }
  ],
  "overallStatus": "NonCompliant",
  "findings": [
    {
      "principleId": "GO-001",
      "status": "NonCompliant",
      "details": "Error not handled at line 25",
      "proposedCorrectiveAction": {
        "actionId": "action-123",
        "description": "Add error check for file.Close()",
        "severity": "High",
        "status": "Proposed"
      }
    },
    {
      "principleId": "GO-006",
      "status": "ManualReviewRequired",
      "details": "Subjective principle: 'Clarity of function name' requires human judgment."
    }
  ]
}
```