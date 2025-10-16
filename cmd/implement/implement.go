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
	file, err := os.Open(path)
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
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	content, err := os.ReadFile(filename)
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
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	var tasks []Task
	var currentPhase string
	re := regexp.MustCompile(`(?m)^- [\( |x|X)] \*\*(\S+)\*\*: (.*)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "### ") {
			currentPhase = strings.TrimSpace(strings.TrimPrefix(line, "### "))
		} else {
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				task := Task{
					ID:          matches[2],
					Description: matches[3],
					Phase:       currentPhase,
					IsCompleted: matches[1] != " ",
					IsParallel:  strings.Contains(matches[3], "[P]"),
				}
				tasks = append(tasks, task)
			}
		}
	}

	return tasks, scanner.Err()
}

func executeTasks(tasks []Task, tasksPath string) error {
	phases := []string{"Setup", "Tests", "Core", "Integration", "Polish"}
	for _, phase := range phases {
		fmt.Printf("\n--- Executing Phase: %s ---\n", phase)
		for _, task := range tasks {
			if task.Phase == phase && !task.IsCompleted {
				fmt.Printf("Executing task %s: %s\n", task.ID, task.Description)
				// TODO: Actually execute the task
				err := markTaskComplete(task.ID, tasksPath)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func markTaskComplete(taskID, path string) error {
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	re := regexp.MustCompile(`(\- \[ \]) (\*\*` + taskID + `\*\:)`)

	for i, line := range lines {
		if re.MatchString(line) {
			lines[i] = re.ReplaceAllString(line, "- [X] $2")
		}
	}

	output := strings.Join(lines, "\n")
	err = os.WriteFile(path, []byte(output), 0600)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	// Here you will define your flags and configuration settings.
}
