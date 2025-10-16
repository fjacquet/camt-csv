---
description: "Generate an actionable, dependency-ordered tasks.md for the feature based on available design artifacts."
---

# Tasks: Review Codebase for Constitution Compliance

**Input**: Design documents from `/specs/001-review-the-codebase/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), data-model.md, contracts/cli-contracts.md, quickstart.md

**Tests**: The feature specification implies a need for testing through "Independent Test" and "Acceptance Scenarios" sections. Therefore, test tasks are included.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `src/`, `tests/` at repository root
- Paths shown below assume single project - adjust based on plan.md structure

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create the basic CLI command structure using `cobra` for `camt-csv review` in `cmd/camt/review.go`.
- [x] T002 Initialize Go module and add primary dependencies (`spf13/cobra`, `spf13/viper`, `github.com/sirupsen/logrus`, `gopkg.in/yaml.v3`).
- [x] T003 [P] Configure `viper` for configuration management, including loading multiple constitution files in `internal/config/`.
- [x] T004 [P] Configure `logrus` for structured logging in `internal/logging/logger.go`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Create core data models (`CodebaseSection`, `ConstitutionPrinciple`, `ComplianceReport`, `Finding`, `CorrectiveAction`) in `internal/models/`.
- [x] T006 Implement a `CodebaseScanner` service in `internal/services/scanner.go` to read files and directories.
- [x] T007 Implement a `ConstitutionLoader` service in `internal/services/parser/constitution.go` to parse constitution definition files (YAML).
- [x] T008 Implement a `ReportGenerator` service in `internal/services/report/generator.go` to create structured reports (JSON/XML).

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Systematically Review Codebase (Priority: P1) 🎯 MVP

**Goal**: As a developer, I want to systematically review the codebase against the project constitution, so that I can identify areas of non-compliance and propose corrective actions.

**Independent Test**: The tool can generate a compliance report for a given codebase section and constitution. The report accurately identifies non-compliant areas based on defined principles. The report can flag subjective principles for manual review. The report can be generated in JSON and XML formats.

### Tests for User Story 1 ⚠️d


NOTE: Write these tests FIRST, ensure they FAIL before implementation

- [x] T009 [P][us1] Write unit tests for `CodebaseScanner` (e.g., scanning files, directories, handling non-existent paths) in `internal/services/scanner_test.go`.
- [x] T010 [P][us1] Write unit tests for `ConstitutionLoader` (e.g., parsing valid/invalid YAML, loading multiple files) in `internal/services/parser/constitution_test.go`.
- [x] T011 [P][us1] Write unit tests for `ReportGenerator` (e.g., generating JSON/XML, handling different finding statuses) in `internal/services/report/generator_test.go`.
- [X] T012 [US1] Write integration tests for the `camt-csv review` command, verifying argument parsing and basic report generation in `cmd/camt/review_test.go`.

### Implementation for User Story 1

- [X] T013 [US1] Implement argument parsing for `camt-csv review` command (`path`, `constitution-files`, `output-format`, `output-file`) in `cmd/camt/review.go`.
- [X] T014 [US1] Implement the `PrincipleEvaluator` interface and a basic `AutomatedPrincipleEvaluator` in `internal/services/reviewer/evaluator.go` to check for patterns.
- [X] T014.1 [P] [US1] Write unit tests for `AutomatedPrincipleEvaluator` in `internal/reviewer/evaluator_test.go`.
- [X] T015 [US1] Implement the main `Reviewer` service in `internal/services/reviewer/reviewer.go` that orchestrates scanning, loading, and evaluating.
- [X] T016 [US1] Integrate `Reviewer` service with the `camt-csv review` CLI command in `cmd/camt/review.go`.
- [X] T017 [US1] Implement logic to flag subjective principles (`EvaluationMethod: Manual`) for manual review in the report within `internal/services/report/generator.go`.
- [X] T018 [US1] Implement the `--principles` flag to allow selecting specific constitution principles in `cmd/camt/review.go`.
- [X] T019 [US1] Implement the `--git-ref` flag for Git integration (diff-based review) in `cmd/camt/review.go` and `internal/git/git.go`.
- [X] T020 [US1] Implement the mechanism to record proposed corrective actions (FR-005) within the `ReportGenerator` in `internal/services/report/generator.go`.

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T021 Add comprehensive Godoc comments to all public functions, types, and interfaces in `internal/models/`, `internal/services/`, `cmd/camt/review.go`.
- [X] T022 Run `golangci-lint` and fix all reported issues across the codebase.
- [X] T023 Update `README.md` with usage instructions for the `camt-csv review` command.
- [X] T024 Ensure all user input is validated (paths, formats, etc.) in `cmd/camt/review.go`.
- [X] T025 Review and refine logging statements for clarity and appropriate levels in `internal/logging/logger.go` and other relevant files.
- [X] T026 Run `uv run ruff check --fix` and `uv run yamlfix src` to ensure adherence to style and quality standards.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "T009 [P] [US1] Write unit tests for `CodebaseScanner` (e.g., scanning files, directories, handling non-existent paths) in `internal/services/scanner_test.go`."
Task: "T010 [P] [US1] Write unit tests for `ConstitutionLoader` (e.g., parsing valid/invalid YAML, loading multiple files) in `internal/services/parser/constitution_test.go`."
Task: "T011 [P] [US1] Write unit tests for `ReportGenerator` (e.g., generating JSON/XML, handling different finding statuses) in `internal/services/report/generator_test.go`."

# Launch all models for User Story 1 together (if applicable, in this case, models are foundational):
# (No specific parallel model tasks for US1 as models are foundational)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
