# Implementation Plan: Codebase Improvements: Parser Factory, Custom Errors, Testable Categorization, Standardized Logging

**Branch**: `002-enahance-the-codebase` | **Date**: mardi 14 octobre 2025 | **Spec**: /specs/002-enahance-the-codebase/spec.md
**Input**: Feature specification from `/specs/002-enahance-the-codebase/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This feature aims to enhance the codebase by centralizing parser creation logic (Parser Factory), improving error handling with custom error types, making AI categorization logic testable via an `AIClient` interface, and standardizing structured logging field names. The technical approach involves refactoring existing parser instantiation, defining new error types, introducing an interface for AI clients, and creating constants for logging fields.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: `spf13/cobra`, `spf13/viper`, `github.com/sirupsen/logrus`, `gopkg.in/yaml.v3`, `go/ast`, `go/parser`, `go/token`, `github.com/stretchr/testify`, `github.com/shopspring/decimal`, `github.com/gocarina/gocsv`, `github.com/joho/godotenv`
**Storage**: Files (constitution definitions, generated reports), YAML files (categories, creditors, debitors)
**Testing**: `go test` with `github.com/stretchr/testify`
**Target Platform**: Linux server
**Project Type**: Single project (Go CLI application)
**Performance Goals**: NEEDS CLARIFICATION
**Constraints**: NEEDS CLARIFICATION
**Scale/Scope**: Small to medium CLI application

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The feature specification aligns well with the project constitution. The proposed changes actively improve adherence to several key principles:

- **II. Robust Error Handling**: Explicitly addresses custom error types.
- **III. Comprehensive Testing (NON-NEGOTIABLE)**: Emphasizes testability for AI categorization.
- **VI. Interface-Driven Design**: Introduces `Parser` and `AIClient` interfaces.
- **VII. Single Responsibility Principle**: Promotes better separation of concerns with the Parser Factory and interfaces.
- **Design Patterns**: Leverages Factory and Adapter patterns.
- **Monitoring & Observability**: Standardized logging contributes to better observability.

The only area needing clarification is "Performance Goals" and "Constraints" in the Technical Context, which will be noted for potential future research if they become critical. No immediate constitution violations are identified.

## Project Structure

### Documentation (this feature)

```
specs/002-enahance-the-codebase/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```
src/
tests/
cmd/
internal/
├── camtparser/
├── categorizer/
├── common/
├── config/
├── currencyutils/
├── dateutils/
├── debitparser/
├── fileutils/
├── git/
├── logging/          # Will be enhanced with standardized field constants
├── models/
├── parser/           # Will contain the Parser interface and the new Parser Factory
├── parsererror/      # Will contain custom error types
├── pdfparser/
├── report/
├── reviewer/
├── revolutinvestmentparser/
├── revolutparser/
├── scanner/
├── selmaparser/
├── store/
├── textutils/
├── validation/
└── xmlutils/
```

**Structure Decision**: The existing single project structure is suitable. New components (Parser Factory, custom error types, `AIClient` interface, `GeminiClient` implementation, logging constants) will be integrated into existing or new `internal` packages, specifically `internal/parser`, `internal/parsererror`, `internal/categorizer`, and `internal/logging`.

## Complexity Tracking

*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |