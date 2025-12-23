package tasks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateTasks_BasicSpec(t *testing.T) {
	specContent := `# Feature Specification: User Authentication

## User Scenarios & Testing

### User Story 1 - Login (Priority: P1)

Users can log in with email and password.

**Acceptance Scenarios**:

1. **Given** a valid user, **When** they enter correct credentials, **Then** they should be logged in

### User Story 2 - Registration (Priority: P2)

Users can create new accounts.

**Acceptance Scenarios**:

1. **Given** a new user, **When** they provide valid info, **Then** account is created

## Requirements

### Functional Requirements

- **FR-001**: System MUST allow users to log in
- **FR-002**: System MUST validate email addresses

### Edge Cases

- What happens when password is empty?
- How does system handle invalid email?
`

	planContent := `# Implementation Plan

## Project Structure

- cmd/auth/
- internal/auth/

## Technologies

Using Go and PostgreSQL for backend.
`

	result, err := generateTasks(specContent, planContent)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Check header
	assert.Contains(t, result, "# Tasks: User Authentication")

	// Check phases
	assert.Contains(t, result, "## Phase 1: Setup")
	assert.Contains(t, result, "## Phase 2: Foundational")
	assert.Contains(t, result, "## Phase 3: User Story 1 - Login")
	assert.Contains(t, result, "## Phase 4: User Story 2 - Registration")
	assert.Contains(t, result, "## Phase N: Polish")

	// Check MVP label
	assert.Contains(t, result, "(Priority: P1) 🎯 MVP")

	// Check task format
	assert.Contains(t, result, "- [ ] T001")
	assert.Contains(t, result, "[US1]")
	assert.Contains(t, result, "[US2]")

	// Check requirements are included
	assert.Contains(t, result, "FR-001")
	assert.Contains(t, result, "FR-002")

	// Check edge cases
	assert.Contains(t, result, "edge case")

	// Check technologies
	assert.Contains(t, result, "Go")
}

func TestGenerateTasks_EmptySpec(t *testing.T) {
	specContent := ``
	planContent := ``

	result, err := generateTasks(specContent, planContent)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Should have fallback feature name
	assert.Contains(t, result, "# Tasks: [FEATURE NAME]")

	// Should have basic phases
	assert.Contains(t, result, "## Phase 1: Setup")
	assert.Contains(t, result, "## Phase 2: Foundational")
	assert.Contains(t, result, "## Phase 3: Core Implementation")
}

func TestParseSpecContent(t *testing.T) {
	content := `# Feature Specification: Test Feature

### User Story 1 - First Story (Priority: P1)

First story description.

### User Story 2 - Second Story (Priority: P2)

Second story description.

## Requirements

### Functional Requirements

- **FR-001**: First requirement
- **FR-002**: Second requirement

### Edge Cases

- Edge case 1
- Edge case 2
`

	spec := parseSpecContent(content)

	assert.Equal(t, "Test Feature", spec.FeatureName)
	assert.Len(t, spec.UserStories, 2)
	assert.Equal(t, "First Story", spec.UserStories[0].Title)
	assert.Equal(t, "P1", spec.UserStories[0].Priority)
	assert.Equal(t, "Second Story", spec.UserStories[1].Title)
	assert.Equal(t, "P2", spec.UserStories[1].Priority)

	assert.Len(t, spec.Requirements, 2)
	assert.Equal(t, "FR-001", spec.Requirements[0].ID)
	assert.Equal(t, "First requirement", spec.Requirements[0].Description)

	assert.Len(t, spec.EdgeCases, 2)
	assert.Equal(t, "Edge case 1", spec.EdgeCases[0])
}

func TestParsePlanContent(t *testing.T) {
	content := `# Implementation Plan

## Project Structure

src/
tests/

## Technologies

This project uses Go and React.

## Phase 1: Setup

Initial setup phase.

## Phase 2: Implementation

Core implementation.
`

	plan := parsePlanContent(content)

	assert.Contains(t, plan.ProjectStructure, "src/")
	assert.Contains(t, plan.Technologies, "Go")
	assert.Contains(t, plan.Technologies, "React")
	assert.Len(t, plan.Phases, 2)
}

func TestExtractSection(t *testing.T) {
	content := `# Title

## Section One

Content of section one.

## Section Two

Content of section two.

## Section Three

Content of section three.
`

	result := extractSection(content, "Section Two")
	assert.Equal(t, "Content of section two.", result)

	result = extractSection(content, "Nonexistent")
	assert.Empty(t, result)
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a longer string", 10, "this is..."},
		{"exactly10!", 10, "exactly10!"},
		{"  trimmed  ", 20, "trimmed"},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGenerateTasks_TaskNumbering(t *testing.T) {
	specContent := `# Feature Specification: Numbering Test

### User Story 1 - Story (Priority: P1)

Test story.
`
	planContent := ``

	result, err := generateTasks(specContent, planContent)
	assert.NoError(t, err)

	// Tasks should be numbered sequentially
	assert.Contains(t, result, "T001")
	assert.Contains(t, result, "T002")

	// Count tasks
	taskCount := strings.Count(result, "- [ ] T")
	assert.Greater(t, taskCount, 5, "Should have multiple tasks")
}
