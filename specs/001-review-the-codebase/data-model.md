# Data Model: Codebase Compliance Review

## Entities

### Codebase Section
- **Description**: Represents an entire file or directory within the project that is subject to compliance review.
- **Attributes**:
  - `Path`: String (absolute path to the file or directory)
  - `Type`: Enum (File, Directory)
  - `Content`: String (content of the file, if `Type` is File)

### Constitution Principle
- **Description**: A specific rule or guideline from the project constitution against which the codebase is reviewed.
- **Attributes**:
  - `ID`: String (unique identifier for the principle)
  - `Name`: String (short, descriptive name)
  - `Description`: String (detailed explanation of the principle)
  - `Category`: String (e.g., "Error Handling", "Testing", "Security")
  - `EvaluationMethod`: Enum (Automated, Manual) - indicates if the principle can be checked by the tool or requires human judgment.
  - `Pattern`: String (regex or other pattern for automated checks, if `EvaluationMethod` is Automated)

### Compliance Report
- **Description**: A document detailing the findings of a compliance review for a selected codebase section against a set of constitution principles.
- **Attributes**:
  - `ReportID`: String (unique identifier for the report)
  - `Timestamp`: DateTime (when the report was generated)
  - `CodebaseSection`: Reference to `Codebase Section` (the section reviewed)
  - `PrinciplesReviewed`: List of References to `Constitution Principle`
  - `OverallStatus`: Enum (Compliant, NonCompliant, PartialCompliance)
  - `Findings`: List of `Finding` objects

### Finding
- **Description**: Details a specific compliance status for a principle within a codebase section.
- **Attributes**:
  - `Principle`: Reference to `Constitution Principle`
  - `Status`: Enum (Compliant, NonCompliant, ManualReviewRequired)
  - `Details`: String (explanation of the finding, e.g., line numbers, specific violations)
  - `ProposedCorrectiveAction`: Reference to `Corrective Action` (optional)

### Corrective Action
- **Description**: A proposed change or action to address a non-compliant area identified in a `Finding`.
- **Attributes**:
  - `ActionID`: String (unique identifier for the action)
  - `Description`: String (detailed description of the proposed action)
  - `Severity`: Enum (High, Medium, Low)
  - `Status`: Enum (Proposed, Implemented, Rejected)
  - `RelatedFinding`: Reference to `Finding`

## Relationships

- `Compliance Report` contains multiple `Finding`s.
- Each `Finding` is related to one `Constitution Principle` and optionally one `Corrective Action`.
- `Corrective Action` is related to one `Finding`.
- `Compliance Report` is generated for one `Codebase Section`.

## Validation Rules

- `Constitution Principle.ID` must be unique.
- `Codebase Section.Path` must be a valid absolute path.
- `Compliance Report.OverallStatus` must accurately reflect the `Findings`.

## State Transitions

### Corrective Action Status
- `Proposed` -> `Implemented`
- `Proposed` -> `Rejected`