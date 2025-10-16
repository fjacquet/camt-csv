# Quickstart Guide: Codebase Compliance Review Tool

This guide provides a quick overview of how to use the `camt-csv review` command to perform automated compliance checks against your project's constitution.

## 1. Build the Tool

Navigate to the project root directory and build the `camt-csv` executable:

```bash
go build -o camt-csv main.go
```

This will create an executable named `camt-csv` in your current directory.

## 2. Define Your Constitution

Ensure you have one or more constitution definition files (e.g., `constitution.yaml`) that outline your project's principles. These files should be in a format that the tool can parse (e.g., YAML).

Example `constitution.yaml` (simplified):

```yaml
principles:
  - id: GO-001
    name: "Robust Error Handling"
    description: "Errors MUST always be checked and propagated."
    evaluationMethod: "Automated"
    pattern: "_ = (err|e)"
  - id: GO-006
    name: "Clarity of Function Name"
    description: "Function names should clearly indicate their purpose."
    evaluationMethod: "Manual"
```

## 3. Run a Compliance Review

Execute the `review` command, specifying the codebase sections to analyze and your constitution files. You can also specify output format and an output file.

```bash
./camt-csv review /path/to/your/project/src/ --constitution-files ./constitution.yaml --output-format json --output-file compliance_report.json
```

- Replace `/path/to/your/project/src/` with the actual path to your source code.
- Adjust `./constitution.yaml` to the path of your constitution file(s).

## 4. Interpret the Report

The generated `compliance_report.json` (or XML) will contain a structured overview of the compliance status. Key sections include:

- `overallStatus`: Indicates the general compliance level (e.g., `Compliant`, `NonCompliant`, `PartialCompliance`).
- `findings`: A list of specific issues found, including the `principleId`, `status` (e.g., `NonCompliant`, `ManualReviewRequired`), `details` (e.g., line numbers), and optionally a `proposedCorrectiveAction`.

**Example JSON Output Snippet:**

```json
{
  "reportId": "uuid-v4",
  "timestamp": "2025-10-12T10:00:00Z",
  "codebaseSection": [
    {
      "path": "/path/to/your/project/src/main.go",
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
    }
  ]
}
```

For `ManualReviewRequired` findings, human intervention is needed to assess compliance.