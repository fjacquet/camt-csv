package analyze

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type Prerequisites struct {
	FeatureDir    string   `json:"FEATURE_DIR"`
	AvailableDocs []string `json:"AVAILABLE_DOCS"`
}

var AnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Performs a non-destructive cross-artifact consistency and quality analysis",
	Long: `Performs a non-destructive cross-artifact consistency and quality analysis
across spec.md, plan.md, and tasks.md after task generation.`,
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

		specPath := filepath.Join(prereqs.FeatureDir, "spec.md")
		planPath := filepath.Join(prereqs.FeatureDir, "plan.md")
		tasksPath := filepath.Join(prereqs.FeatureDir, "tasks.md")
		constitutionPath := ".specify/memory/constitution.md"

		specContent, err := os.ReadFile(specPath) // #nosec G304 -- reading spec files from known project paths
		if err != nil {
			fmt.Printf("Error reading spec.md: %s\n", err)
			return
		}

		planContent, err := os.ReadFile(planPath) // #nosec G304 -- reading plan files from known project paths
		if err != nil {
			fmt.Printf("Warning: could not read plan.md: %s\n", err)
		}

		tasksContent, err := os.ReadFile(tasksPath) // #nosec G304 -- reading task files from known project paths
		if err != nil {
			fmt.Printf("Error reading tasks.md: %s\n", err)
			return
		}

		constitutionContent, err := os.ReadFile(constitutionPath)
		if err != nil {
			fmt.Printf("Error reading constitution.md: %s\n", err)
			return
		}

		spec := parseSpec(string(specContent))
		plan := parsePlan(string(planContent))
		tasks := parseTasks(string(tasksContent))
		constitution := parseConstitution(string(constitutionContent))

		report := Analyze(spec, plan, tasks, constitution)

		fmt.Println("## Specification Analysis Report")
		fmt.Println("| ID | Category | Severity | Location(s) | Summary | Recommendation |")
		fmt.Println("|----|----------|----------|-------------|---------|----------------|")
		for _, finding := range report.Findings {
			fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
				finding.ID, finding.Category, finding.Severity,
				finding.Location, finding.Summary, finding.Recommendation)
		}

		fmt.Println("\n**Coverage Summary Table:**")
		fmt.Println("| Requirement Key | Has Task? | Task IDs |")
		fmt.Println("|-----------------|-----------|----------|")
		for req, taskIDs := range report.CoverageSummary {
			hasTask := "No"
			if len(taskIDs) > 0 {
				hasTask = "Yes"
			}
			fmt.Printf("| %s | %s | %s |\n", req, hasTask, strings.Join(taskIDs, ", "))
		}

		if len(report.ConstitutionIssues) > 0 {
			fmt.Println("\n**Constitution Alignment Issues:**")
			for _, issue := range report.ConstitutionIssues {
				fmt.Printf("- **%s**: %s (%s)\n", issue.Severity, issue.Summary, issue.Location)
			}
		}

		if len(report.UnmappedTasks) > 0 {
			fmt.Println("\n**Unmapped Tasks:**")
			for _, task := range report.UnmappedTasks {
				fmt.Printf("- **%s**: %s\n", task.ID, task.Description)
			}
		}

		fmt.Println("\n**Metrics:**")
		for metric, value := range report.Metrics {
			fmt.Printf("- %s: %d\n", metric, value)
		}

		fmt.Println("\n### Next Actions")
		if report.Metrics["Critical Issues Count"] > 0 {
			fmt.Println("- **CRITICAL**: Resolve critical issues before running `/implement`.")
		} else {
			fmt.Println("- No critical issues found. You can proceed with `/implement`.")
		}
		fmt.Println("- Consider running `/specify` or `/plan` to refine the artifacts based on the findings.")

		fmt.Println("\nWould you like me to suggest concrete remediation edits for the top 3 issues?")
	},
}

func init() {
	// Here you will define your flags and configuration settings.
}
