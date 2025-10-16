# Tasks: Codebase Improvements: Parser Factory, Custom Errors, Testable Categorization, Standardized Logging

**Feature Branch**: `002-enahance-the-codebase` | **Date**: mardi 14 octobre 2025
**Input**: Feature specification from `/specs/002-enahance-the-codebase/spec.md`

This document outlines the actionable tasks required to implement the codebase improvements, organized by user story and prioritized for incremental delivery.

## Phase 1: Setup Tasks (Project Initialization)

These tasks establish the foundational structures and interfaces required across the feature.

- [x] **T001**: Create `internal/parser/parser.go` with the `Parser` interface.
  - **File**: `internal/parser/parser.go`
- [x] **T002**: Create `internal/parser/factory.go` for the `ParserFactory` and `ParserType` enumeration.
  - **File**: `internal/parser/factory.go`
- [x] **T003**: Create `internal/parsererror/errors.go` for custom error types (`InvalidFormatError`, `DataExtractionError`).
  - **File**: `internal/parsererror/errors.go`
- [x] **T004**: Create `internal/categorizer/ai_client.go` with the `AIClient` interface.
  - **File**: `internal/categorizer/ai_client.go`
- [x] **T005**: Create `internal/logging/constants.go` for standardized logging field constants.
  - **File**: `internal/logging/constants.go`

## Phase 2: Foundational Tasks (Blocking Prerequisites)

These tasks adapt existing components to the new interfaces and error handling mechanisms, serving as prerequisites for user story implementation.

- [x] **T006**: Update existing parser adapters (`camtparser.Adapter`, `pdfparser.Adapter`, `revolutparser.Adapter`, `revolutinvestmentparser.Adapter`, `selmaparser.Adapter`, `debitparser.Adapter`) to explicitly implement the `internal/parser/Parser` interface.
  - **Files**: `internal/camtparser/adapter.go`, `internal/pdfparser/adapter.go`, `internal/revolutparser/adapter.go`, `internal/revolutinvestmentparser/adapter.go`, `internal/selmaparser/adapter.go`, `internal/debitparser/adapter.go`
- [x] **T007**: Modify existing parser adapters to return `internal/parsererror.InvalidFormatError` or `internal/parsererror.DataExtractionError` where applicable, replacing generic errors.
  - **Files**: `internal/camtparser/adapter.go`, `internal/pdfparser/adapter.go`, `internal/revolutparser/adapter.go`, `internal/revolutinvestmentparser/adapter.go`, `internal/selmaparser/adapter.go`, `internal/debitparser/adapter.go`

## Phase 3: User Story 1 - Centralized Parser Management (P1)

**Goal**: As a developer, I want to easily add new parsers or modify existing parser instantiation logic in a single, centralized location, so that the codebase is more maintainable and extensible.

**Independent Test**: Can be fully tested by adding a new dummy parser type and verifying that it can be instantiated via the factory without changes to existing CLI commands, and delivers a more organized and extensible parser management system.

- [x] **T008 [US1]**: Implement `parser.GetParser` function in `internal/parser/factory.go` to return `Parser` implementations based on `ParserType`.
  - **File**: `internal/parser/factory.go`
- [x] **T009 [US1]**: Update `cmd/*/convert.go` files (e.g., `cmd/camt/convert.go`, `cmd/pdf/convert.go`) to use `parser.GetParser` for parser instantiation.
  - **Files**: `cmd/camt/convert.go`, `cmd/pdf/convert.go`, `cmd/revolut/convert.go`, `cmd/revolut-investment/convert.go`, `cmd/selma/convert.go`, `cmd/debit/convert.go`
- [x] **T010 [US1]**: Write unit tests for `internal/parser/factory.go` to ensure correct parser instantiation and error handling for unknown types.
  - **File**: `internal/parser/factory_test.go` (new file)

## Phase 4: User Story 2 - Clearer Error Handling (P1)

**Goal**: As a developer, I want to receive specific and programmatic error types when parsing or data extraction fails, so that I can implement more robust error handling and provide clearer feedback to users.

**Independent Test**: Can be fully tested by providing malformed input files or files with missing data and asserting that the correct custom error types (`InvalidFormatError`, `DataExtractionError`) are returned with appropriate details, and delivers more precise error reporting.

- [x] **T011 [US2]**: Review and update error handling in `cmd/*/convert.go` to use `errors.As` for `InvalidFormatError` and `DataExtractionError` for more granular error messages.
  - **Files**: `cmd/camt/convert.go`, `cmd/pdf/convert.go`, `cmd/revolut/convert.go`, `cmd/revolut-investment/convert.go`, `cmd/selma/convert.go`, `cmd/debit/convert.go`
- [x] **T012 [US2]**: Write integration tests for `cmd/*/convert.go` commands to verify custom error propagation with invalid input files.
  - **Files**: `cmd/camt/convert_test.go`, `cmd/pdf/convert_test.go`, `cmd/revolut/convert_test.go`, `cmd/revolut-investment/convert_test.go`, `cmd/selma/convert_test.go`, `cmd/debit/convert_test.go` (new/update existing)

## Phase 5: User Story 3 - Testable AI Categorization Logic (P2)

**Goal**: As a developer, I want to easily test the core categorization logic without making actual calls to the Gemini API, so that I can write faster, more reliable unit tests for the categorizer.

**Independent Test**: Can be fully tested by refactoring the `Categorizer` to accept an `AIClient` interface and then providing a mock implementation of `AIClient` in unit tests to verify categorization logic without external API calls, and delivers a more robust and testable categorization module.

- [x] **T013 [US3]**: Implement `internal/categorizer/GeminiClient` struct that implements the `internal/categorizer/AIClient` interface.
  - **File**: `internal/categorizer/gemini_client.go` (new file)
- [x] **T014 [US3]**: Refactor `internal/categorizer/categorizer.go` to accept an `AIClient` interface in its constructor.
  - **File**: `internal/categorizer/categorizer.go`
- [x] **T015 [US3]**: Update `cmd/categorize/categorize.go` to instantiate `GeminiClient` and pass it to the `Categorizer`.
  - **File**: `cmd/categorize/categorize.go`
- [x] **T016 [US3]**: Write unit tests for `internal/categorizer/categorizer.go` using a mock `AIClient` to test categorization logic in isolation.
  - **File**: `internal/categorizer/categorizer_test.go`

## Phase 6: User Story 4 - Standardized Logging (P3)

**Goal**: As a developer, I want to use consistent field names for structured logging across the application, so that logs are easier to parse, filter, and analyze for debugging and monitoring.

**Independent Test**: Can be fully tested by modifying existing log statements to use the new constants and verifying that log outputs consistently use the standardized field names, and delivers more consistent and analyzable logs.

- [x] **T017 [US4]**: Identify key log statements across the codebase (e.g., in parsers, categorizer, CLI commands) that can benefit from standardized fields.
  - **Files**: `internal/**/*.go`, `cmd/**/*.go` (review)
- [x] **T018 [US4]**: Replace hardcoded string literals with `internal/logging/constants.go` for structured log fields in identified locations.
  - **Files**: `internal/**/*.go`, `cmd/**/*.go` (modify)
- [x] **T019 [US4]**: Write unit tests or integration tests to verify that log outputs consistently use the standardized field names.
  - **File**: `internal/logging/con
  - stants_test.go` (new file)

## Phase 7: Polish & Cross-Cutting Concerns

- [x] **T020**: Run `go fmt`, `goimports`, and `golangci-lint` across the entire codebase to ensure code style and quality.
  - **Command**: `go fmt ./... && goimports -w . && golangci-lint run`
- [x] **T021**: Update `README.md` and relevant documentation (e.g., `docs/coding-standards.md`, `docs/api-specifications.md`) to reflect new patterns (Parser Factory, cu
- [ ] stom errors, AIClient, logging constants).
  - **Files**: `README.md`, `docs/coding-standards.md`, `docs/api-specifications.md`
- **T022**: Run the entire test suite (`go test ./...`) to ensure no regressions have been introduced.
  - **Command**: `go test ./...`

## Dependencies

- Phase 1 (Setup) MUST be completed before Phase 2 (Foundational).
- Phase 2 (Foundational) MUST be completed before any User Story Phase (3-6).
- User Story Phases (3-6) can be worked on in parallel where file conflicts are minimal, but generally follow priority order (P1, P2, P3).
- Phase 7 (Polish & Cross-Cutting Concerns) MUST be completed last.

## Parallel Execution Examples

**User Story 1 (P1)**:

- T008 [US1] (Implement `GetParser`) and T009 [US1] (Update CLI commands) can be developed in parallel if developers coordinate on the `parser.GetParser` signature.

**User Story 2 (P1)**:

- T011 [US2] (Update error handling in CLI) and T012 [US2] (Write integration tests) can be done in parallel (TDD approach).

**User Story 3 (P2)**:

- T013 [US3] (Implement `GeminiClient`) and T014 [US3] (Refactor `Categorizer`) can be developed in parallel.

**User Story 4 (P3)**:

- T017 [US4] (Identify log statements) and T018 [US4] (Replace hardcoded strings) can be done in parallel.

## Implementation Strategy

The implementation will follow an incremental delivery approach, prioritizing User Stories by their assigned priority (P1 first, then P2, P3). Each User Story will be treated as a mini-MVP, with its own set of tests to ensure independent testability and functionality before moving to the next story. Foundational tasks will be completed first to unblock all user stories. Polish and cross-cutting concerns will be addressed in the final phase.
