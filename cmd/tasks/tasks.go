package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Prerequisites struct {
	FeatureDir    string   `json:"FEATURE_DIR"`
	AvailableDocs []string `json:"AVAILABLE_DOCS"`
}

var TasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Generates an actionable, dependency-ordered tasks.md for the feature",
	Long: `Generates an actionable, dependency-ordered tasks.md for the feature based on available design artifacts.
This command creates a detailed, executable plan organized by user story.`,
	Run: func(cmd *cobra.Command, args []string) {
		scriptPath := ".specify/scripts/bash/check-prerequisites.sh"
		out, err := exec.Command("bash", scriptPath, "--json").Output()
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

		specPath := filepath.Join(prereqs.FeatureDir, "spec.md")
		planPath := filepath.Join(prereqs.FeatureDir, "plan.md")

		specContent, err := os.ReadFile(specPath)
		if err != nil {
			fmt.Printf("Error reading spec.md: %s\n", err)
			return
		}

		planContent, err := os.ReadFile(planPath)
		if err != nil {
			fmt.Printf("Error reading plan.md: %s\n", err)
			return
		}

		tasksContent, err := generateTasks(string(specContent), string(planContent))
		if err != nil {
			fmt.Printf("Error generating tasks: %s\n", err)
			return
		}

		tasksPath := filepath.Join(prereqs.FeatureDir, "tasks.md")
		err = os.WriteFile(tasksPath, []byte(tasksContent), 0600)
		if err != nil {
			fmt.Printf("Error writing tasks.md: %s\n", err)
			return
		}

		fmt.Printf("Successfully generated %s\n", tasksPath)
	},
}

func generateTasks(specContent, planContent string) (string, error) {
	// TODO: Implement the actual task generation logic.
	return "## tasks.md placeholder", nil
}

func init() {
	// Here you will define your flags and configuration settings.
}
