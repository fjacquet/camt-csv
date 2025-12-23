package implement

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

type Prerequisites struct {
	FeatureDir    string   `json:"FEATURE_DIR"`
	AvailableDocs []string `json:"AVAILABLE_DOCS"`
}

type ChecklistResult struct {
	Name       string
	Total      int
	Completed  int
	Incomplete int
	Passed     bool
}

type Task struct {
	ID          string
	Description string
	Phase       string
	IsParallel  bool
	IsCompleted bool
	Command     string // Optional shell command to execute
}

var ImplementCmd = &cobra.Command{
	Use:   "implement",
	Short: "Executes the implementation plan by processing and executing all tasks defined in tasks.md",
	Long: `Executes the implementation plan by processing and executing all tasks defined in tasks.md.
This command follows a strict, phase-by-phase execution model, respecting dependencies and TDD principles.`,
	Run: func(cmd *cobra.Command, args []string) {
		scriptPath := ".specify/scripts/bash/check-prerequisites.sh"
		out, err := exec.Command("bash", scriptPath, "--json", "--require-tasks", "--include-tasks").Output()
		if err != nil {
			fmt.Printf("Error executing script: %s\n", err)
			return
		}

		var prereqs Prerequisites
		err = json.Unmarshal(out, &prereqs)
		if err != nil {
			fmt.Printf("Error unmarshalling json: %s\n", err)
			return
		}

		checklistsDir := filepath.Join(prereqs.FeatureDir, "checklists")
		if _, err := os.Stat(checklistsDir); !os.IsNotExist(err) {
			results, allPassed, err := checkChecklists(checklistsDir)
			if err != nil {
				fmt.Printf("Error checking checklists: %s\n", err)
				return
			}

			printChecklistTable(results)

			if !allPassed {
				fmt.Print("\nSome checklists are incomplete. Do you want to proceed with implementation anyway? (yes/no): ")
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					fmt.Printf("Error reading response: %s\n", err)
					return
				}
				if strings.ToLower(response) != "yes" {
					fmt.Println("Implementation halted.")
					return
				}
			}
		}

		err = verifyProjectSetup()
		if err != nil {
			fmt.Printf("Error verifying project setup: %s\n", err)
			return
		}

		tasksPath := filepath.Join(prereqs.FeatureDir, "tasks.md")
		tasks, err := parseTasks(tasksPath)
		if err != nil {
			fmt.Printf("Error parsing tasks: %s\n", err)
			return
		}

		err = executeTasks(tasks, tasksPath)
		if err != nil {
			fmt.Printf("Error executing tasks: %s\n", err)
			return
		}

		fmt.Println("\nImplementation completed successfully.")
	},
}

func checkChecklists(dir string) ([]ChecklistResult, bool, error) {
	var results []ChecklistResult
	allPassed := true

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, false, err
	}

	for _, file := range files {
		if !file.IsDir() {
			result, err := parseChecklist(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, false, err
			}
			results = append(results, result)
			if !result.Passed {
				allPassed = false
			}
		}
	}

	return results, allPassed, nil
}

func parseChecklist(path string) (ChecklistResult, error) {
	file, err := os.Open(path) // #nosec G304 -- reading checklist files from known project paths
	if err != nil {
		return ChecklistResult{}, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	result := ChecklistResult{Name: filepath.Base(path)}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- [ ]") {
			result.Total++
			result.Incomplete++
		} else if strings.HasPrefix(line, "- [X]") || strings.HasPrefix(line, "- [x]") {
			result.Total++
			result.Completed++
		}
	}

	result.Passed = result.Incomplete == 0
	return result, scanner.Err()
}

func printChecklistTable(results []ChecklistResult) {
	fmt.Println("| Checklist | Total | Completed | Incomplete | Status |")
	fmt.Println("|-----------|-------|-----------|------------|--------|")
	for _, r := range results {
		status := "✓ PASS"
		if !r.Passed {
			status = "✗ FAIL"
		}
		fmt.Printf("| %s | %d | %d | %d | %s |\n", r.Name, r.Total, r.Completed, r.Incomplete, status)
	}
}

func verifyProjectSetup() error {
	if _, err := os.Stat(".git"); !os.IsNotExist(err) {
		err := createOrVerifyIgnoreFile(".gitignore", []string{"node_modules/", "dist/", "build/", "*.log", ".env*"})
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("package.json"); !os.IsNotExist(err) {
		// .gitignore is already handled above
	}

	return nil
}

func createOrVerifyIgnoreFile(filename string, patterns []string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600) // #nosec G304 -- writing to known config files
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	content, err := os.ReadFile(filename) // #nosec G304 -- reading known config files
	if err != nil {
		return err
	}

	existingContent := string(content)
	for _, pattern := range patterns {
		if !strings.Contains(existingContent, pattern) {
			if _, err := file.WriteString("\n" + pattern); err != nil {
				return err
			}
			fmt.Printf("Appended '%s' to %s\n", pattern, filename)
		}
	}

	return nil
}

func parseTasks(path string) ([]Task, error) {
	content, err := os.ReadFile(path) // #nosec G304 -- reading task files from known project paths
	if err != nil {
		return nil, err
	}

	var tasks []Task
	var currentPhase string

	// Regex patterns
	phaseRe := regexp.MustCompile(`(?m)^##+ (?:Phase \d+[:\s]+)?(.+)$`)
	// Match task lines: - [ ] T001 Description or - [x] T001 Description
	taskRe := regexp.MustCompile(`(?m)^- \[([ xX])\] (T\d{3})\s*(.*)$`)

	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		// Check for phase headers
		if phaseMatch := phaseRe.FindStringSubmatch(line); len(phaseMatch) > 1 {
			currentPhase = strings.TrimSpace(phaseMatch[1])
			continue
		}

		// Check for task lines
		if taskMatch := taskRe.FindStringSubmatch(line); len(taskMatch) > 3 {
			task := Task{
				ID:          taskMatch[2],
				Description: strings.TrimSpace(taskMatch[3]),
				Phase:       currentPhase,
				IsCompleted: taskMatch[1] != " ",
				IsParallel:  strings.Contains(taskMatch[3], "[P]"),
			}

			// Look for a code block immediately following this task (within next 5 lines)
			for j := i + 1; j < len(lines) && j <= i+5; j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
					// Found start of code block, extract it
					blockStart := j
					for k := j + 1; k < len(lines); k++ {
						if strings.HasPrefix(strings.TrimSpace(lines[k]), "```") {
							// Extract code between markers
							codeLines := lines[blockStart+1 : k]
							task.Command = strings.TrimSpace(strings.Join(codeLines, "\n"))
							break
						}
					}
					break
				}
				// Stop if we hit another task or header
				if strings.HasPrefix(lines[j], "- [") || strings.HasPrefix(lines[j], "##") {
					break
				}
			}

			tasks = append(tasks, task)
		}
	}

	// Alternative: use regex on full content for code block extraction
	// This is a fallback for inline commands in task descriptions
	for i := range tasks {
		if tasks[i].Command == "" {
			// Check for inline command pattern: `command`
			inlineRe := regexp.MustCompile("`([^`]+)`")
			if match := inlineRe.FindStringSubmatch(tasks[i].Description); len(match) > 1 {
				cmd := match[1]
				// Only treat as command if it looks like a shell command
				if isShellCommand(cmd) {
					tasks[i].Command = cmd
				}
			}
		}
	}

	return tasks, nil
}

// isShellCommand returns true if the string looks like a shell command.
func isShellCommand(s string) bool {
	// Common command prefixes
	prefixes := []string{
		"go ", "npm ", "yarn ", "make ", "git ", "docker ",
		"cd ", "mkdir ", "cp ", "mv ", "rm ", "touch ",
		"python ", "pip ", "cargo ", "rustc ",
		"./", "bash ", "sh ", "curl ", "wget ",
	}
	s = strings.TrimSpace(s)
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func executeTasks(tasks []Task, tasksPath string) error {
	// Group tasks by phase (extract unique phases from tasks)
	phaseOrder := extractPhaseOrder(tasks)

	for _, phase := range phaseOrder {
		fmt.Printf("\n--- Executing Phase: %s ---\n", phase)

		// Collect tasks for this phase
		var phaseTasks []Task
		for _, task := range tasks {
			if task.Phase == phase && !task.IsCompleted {
				phaseTasks = append(phaseTasks, task)
			}
		}

		if len(phaseTasks) == 0 {
			fmt.Println("  (no pending tasks)")
			continue
		}

		// Execute tasks
		for _, task := range phaseTasks {
			fmt.Printf("\n[%s] %s\n", task.ID, task.Description)

			if task.Command != "" {
				fmt.Printf("  Command: %s\n", task.Command)
				err := executeCommand(task.Command)
				if err != nil {
					fmt.Printf("  ✗ Failed: %v\n", err)
					// Ask user if they want to continue
					fmt.Print("  Continue with remaining tasks? (yes/no): ")
					var response string
					if _, scanErr := fmt.Scanln(&response); scanErr != nil {
						return fmt.Errorf("failed to read response: %w", scanErr)
					}
					if strings.ToLower(response) != "yes" {
						return fmt.Errorf("task %s failed: %w", task.ID, err)
					}
				} else {
					fmt.Println("  ✓ Completed")
				}
			} else {
				fmt.Println("  (manual task - no command)")
			}

			// Mark task as complete
			err := markTaskComplete(task.ID, tasksPath)
			if err != nil {
				return fmt.Errorf("failed to mark task %s as complete: %w", task.ID, err)
			}
		}
	}
	return nil
}

// extractPhaseOrder returns unique phases in the order they appear in tasks.
func extractPhaseOrder(tasks []Task) []string {
	seen := make(map[string]bool)
	var phases []string
	for _, task := range tasks {
		if task.Phase != "" && !seen[task.Phase] {
			seen[task.Phase] = true
			phases = append(phases, task.Phase)
		}
	}
	return phases
}

// executeCommand runs a shell command and returns any error.
func executeCommand(command string) error {
	// Split multi-line commands
	lines := strings.Split(command, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fmt.Printf("  $ %s\n", line)

		// #nosec G204 -- executing user-defined commands from tasks.md is intentional
		cmd := exec.Command("bash", "-c", line)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command failed: %w", err)
		}
	}
	return nil
}

func markTaskComplete(taskID, path string) error {
	input, err := os.ReadFile(path) // #nosec G304 -- reading task files from known project paths
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	// Match: - [ ] T001 ... and replace with - [x] T001 ...
	re := regexp.MustCompile(`^(- \[) \] (` + regexp.QuoteMeta(taskID) + `\s)`)

	modified := false
	for i, line := range lines {
		if re.MatchString(line) {
			lines[i] = re.ReplaceAllString(line, "${1}x] $2")
			modified = true
		}
	}

	if !modified {
		// Task might already be complete or not found
		return nil
	}

	output := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(output), 0600)
}

func init() {
	// Here you will define your flags and configuration settings.
}
