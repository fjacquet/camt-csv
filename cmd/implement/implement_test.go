package implement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTasks(t *testing.T) {
	// Create a temporary tasks.md file
	tempDir := t.TempDir()
	tasksPath := filepath.Join(tempDir, "tasks.md")

	tasksContent := `# Tasks: Test Feature

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization

- [ ] T001 Create project structure
- [ ] T002 [P] Configure linting tools
- [x] T003 Already completed task

---

## Phase 2: Foundational

**Purpose**: Core infrastructure

- [ ] T004 Setup database
` + "```bash\n" + `go mod init test
go mod tidy
` + "```" + `

- [ ] T005 [P] Setup logging

---

## Phase 3: User Story 1

- [ ] T006 [US1] Implement core feature
`

	err := os.WriteFile(tasksPath, []byte(tasksContent), 0600)
	assert.NoError(t, err)

	tasks, err := parseTasks(tasksPath)
	assert.NoError(t, err)
	assert.Len(t, tasks, 6)

	// Check first task
	assert.Equal(t, "T001", tasks[0].ID)
	assert.Equal(t, "Create project structure", tasks[0].Description)
	assert.Equal(t, "Setup (Shared Infrastructure)", tasks[0].Phase)
	assert.False(t, tasks[0].IsCompleted)
	assert.False(t, tasks[0].IsParallel)

	// Check parallel task
	assert.Equal(t, "T002", tasks[1].ID)
	assert.True(t, tasks[1].IsParallel)

	// Check completed task
	assert.Equal(t, "T003", tasks[2].ID)
	assert.True(t, tasks[2].IsCompleted)

	// Check task with command
	assert.Equal(t, "T004", tasks[3].ID)
	assert.Contains(t, tasks[3].Command, "go mod init")
	assert.Equal(t, "Foundational", tasks[3].Phase)
}

func TestExtractPhaseOrder(t *testing.T) {
	tasks := []Task{
		{ID: "T001", Phase: "Setup"},
		{ID: "T002", Phase: "Setup"},
		{ID: "T003", Phase: "Core"},
		{ID: "T004", Phase: "Tests"},
		{ID: "T005", Phase: "Core"},
	}

	phases := extractPhaseOrder(tasks)
	assert.Equal(t, []string{"Setup", "Core", "Tests"}, phases)
}

func TestIsShellCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"go build ./...", true},
		{"npm install", true},
		{"make test", true},
		{"git status", true},
		{"./run.sh", true},
		{"some text", false},
		{"T001", false},
		{"Description of task", false},
		{"docker run -it ubuntu", true},
		{"mkdir -p dir/subdir", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isShellCommand(tt.input)
			assert.Equal(t, tt.expected, result, "input: %s", tt.input)
		})
	}
}

func TestMarkTaskComplete(t *testing.T) {
	tempDir := t.TempDir()
	tasksPath := filepath.Join(tempDir, "tasks.md")

	tasksContent := `# Tasks

- [ ] T001 First task
- [ ] T002 Second task
- [x] T003 Already complete
`

	err := os.WriteFile(tasksPath, []byte(tasksContent), 0600)
	assert.NoError(t, err)

	// Mark T001 as complete
	err = markTaskComplete("T001", tasksPath)
	assert.NoError(t, err)

	// Read back the file
	content, err := os.ReadFile(tasksPath)
	assert.NoError(t, err)

	assert.Contains(t, string(content), "- [x] T001 First task")
	assert.Contains(t, string(content), "- [ ] T002 Second task")
	assert.Contains(t, string(content), "- [x] T003 Already complete")
}

func TestMarkTaskComplete_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	tasksPath := filepath.Join(tempDir, "tasks.md")

	tasksContent := `# Tasks

- [ ] T001 First task
`

	err := os.WriteFile(tasksPath, []byte(tasksContent), 0600)
	assert.NoError(t, err)

	// Try to mark non-existent task
	err = markTaskComplete("T999", tasksPath)
	assert.NoError(t, err) // Should not error, just do nothing

	// Verify file unchanged
	content, err := os.ReadFile(tasksPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "- [ ] T001 First task")
}

func TestParseChecklist(t *testing.T) {
	tempDir := t.TempDir()
	checklistPath := filepath.Join(tempDir, "checklist.md")

	checklistContent := `# Checklist

- [x] Item 1
- [X] Item 2
- [ ] Item 3
- [ ] Item 4
`

	err := os.WriteFile(checklistPath, []byte(checklistContent), 0600)
	assert.NoError(t, err)

	result, err := parseChecklist(checklistPath)
	assert.NoError(t, err)

	assert.Equal(t, "checklist.md", result.Name)
	assert.Equal(t, 4, result.Total)
	assert.Equal(t, 2, result.Completed)
	assert.Equal(t, 2, result.Incomplete)
	assert.False(t, result.Passed)
}

func TestParseChecklist_AllComplete(t *testing.T) {
	tempDir := t.TempDir()
	checklistPath := filepath.Join(tempDir, "complete.md")

	checklistContent := `# Checklist

- [x] Item 1
- [X] Item 2
`

	err := os.WriteFile(checklistPath, []byte(checklistContent), 0600)
	assert.NoError(t, err)

	result, err := parseChecklist(checklistPath)
	assert.NoError(t, err)

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, result.Completed)
	assert.Equal(t, 0, result.Incomplete)
	assert.True(t, result.Passed)
}
