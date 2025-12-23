package tasks

import (
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

		specContent, err := os.ReadFile(specPath) // #nosec G304 -- reading spec files from known project paths
		if err != nil {
			fmt.Printf("Error reading spec.md: %s\n", err)
			return
		}

		planContent, err := os.ReadFile(planPath) // #nosec G304 -- reading plan files from known project paths
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

// UserStory represents a parsed user story from spec.md.
type UserStory struct {
	Number      int
	Title       string
	Priority    string
	Description string
	Scenarios   []string
}

// Requirement represents a functional requirement from spec.md.
type Requirement struct {
	ID          string
	Description string
}

// ParsedSpec holds extracted content from spec.md.
type ParsedSpec struct {
	FeatureName  string
	UserStories  []UserStory
	Requirements []Requirement
	EdgeCases    []string
}

// ParsedPlan holds extracted content from plan.md.
type ParsedPlan struct {
	ProjectStructure string
	Technologies     []string
	Phases           []string
}

func generateTasks(specContent, planContent string) (string, error) {
	spec := parseSpecContent(specContent)
	plan := parsePlanContent(planContent)

	var sb strings.Builder
	taskNum := 1

	// Header
	featureName := spec.FeatureName
	if featureName == "" {
		featureName = "[FEATURE NAME]"
	}
	sb.WriteString(fmt.Sprintf("# Tasks: %s\n\n", featureName))

	// Phase 1: Setup
	sb.WriteString("## Phase 1: Setup (Shared Infrastructure)\n\n")
	sb.WriteString("**Purpose**: Project initialization and basic structure\n\n")
	sb.WriteString(fmt.Sprintf("- [ ] T%03d Create project structure per implementation plan\n", taskNum))
	taskNum++
	if len(plan.Technologies) > 0 {
		sb.WriteString(fmt.Sprintf("- [ ] T%03d Initialize project with dependencies: %s\n", taskNum, strings.Join(plan.Technologies, ", ")))
		taskNum++
	}
	sb.WriteString(fmt.Sprintf("- [ ] T%03d [P] Configure linting and formatting tools\n", taskNum))
	taskNum++
	sb.WriteString("\n---\n\n")

	// Phase 2: Foundational
	sb.WriteString("## Phase 2: Foundational (Blocking Prerequisites)\n\n")
	sb.WriteString("**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented\n\n")
	sb.WriteString("**⚠️ CRITICAL**: No user story work can begin until this phase is complete\n\n")

	// Generate foundational tasks from requirements
	for _, req := range spec.Requirements {
		sb.WriteString(fmt.Sprintf("- [ ] T%03d [P] Implement %s: %s\n", taskNum, req.ID, req.Description))
		taskNum++
	}
	if len(spec.Requirements) == 0 {
		sb.WriteString(fmt.Sprintf("- [ ] T%03d Setup core infrastructure\n", taskNum))
		taskNum++
		sb.WriteString(fmt.Sprintf("- [ ] T%03d [P] Configure error handling and logging\n", taskNum))
		taskNum++
	}
	sb.WriteString("\n**Checkpoint**: Foundation ready - user story implementation can now begin\n\n")
	sb.WriteString("---\n\n")

	// Phases 3+: User Stories
	for i, story := range spec.UserStories {
		phaseNum := i + 3
		mvpLabel := ""
		if story.Priority == "P1" {
			mvpLabel = " 🎯 MVP"
		}
		sb.WriteString(fmt.Sprintf("## Phase %d: User Story %d - %s (Priority: %s)%s\n\n",
			phaseNum, story.Number, story.Title, story.Priority, mvpLabel))

		if story.Description != "" {
			sb.WriteString(fmt.Sprintf("**Goal**: %s\n\n", story.Description))
		}

		sb.WriteString("### Implementation\n\n")

		// Generate tasks for this user story
		usLabel := fmt.Sprintf("US%d", story.Number)
		sb.WriteString(fmt.Sprintf("- [ ] T%03d [P] [%s] Create models/entities for user story %d\n", taskNum, usLabel, story.Number))
		taskNum++
		sb.WriteString(fmt.Sprintf("- [ ] T%03d [%s] Implement core logic for user story %d\n", taskNum, usLabel, story.Number))
		taskNum++
		sb.WriteString(fmt.Sprintf("- [ ] T%03d [%s] Add validation and error handling\n", taskNum, usLabel))
		taskNum++

		// Add tasks from scenarios
		for j, scenario := range story.Scenarios {
			sb.WriteString(fmt.Sprintf("- [ ] T%03d [%s] Implement scenario %d: %s\n", taskNum, usLabel, j+1, truncateString(scenario, 60)))
			taskNum++
		}

		sb.WriteString(fmt.Sprintf("\n**Checkpoint**: User Story %d should be fully functional and testable\n\n", story.Number))
		sb.WriteString("---\n\n")
	}

	// If no user stories found, add placeholder
	if len(spec.UserStories) == 0 {
		sb.WriteString("## Phase 3: Core Implementation\n\n")
		sb.WriteString("**Purpose**: Main feature implementation\n\n")
		sb.WriteString(fmt.Sprintf("- [ ] T%03d Implement core functionality\n", taskNum))
		taskNum++
		sb.WriteString(fmt.Sprintf("- [ ] T%03d [P] Add unit tests\n", taskNum))
		taskNum++
		sb.WriteString("\n---\n\n")
	}

	// Final Phase: Polish
	sb.WriteString("## Phase N: Polish & Cross-Cutting Concerns\n\n")
	sb.WriteString("**Purpose**: Improvements that affect multiple user stories\n\n")
	sb.WriteString(fmt.Sprintf("- [ ] T%03d [P] Documentation updates\n", taskNum))
	taskNum++
	sb.WriteString(fmt.Sprintf("- [ ] T%03d Code cleanup and refactoring\n", taskNum))
	taskNum++

	// Add edge case handling
	for _, edgeCase := range spec.EdgeCases {
		sb.WriteString(fmt.Sprintf("- [ ] T%03d Handle edge case: %s\n", taskNum, truncateString(edgeCase, 60)))
		taskNum++
	}

	sb.WriteString(fmt.Sprintf("- [ ] T%03d Run final validation and testing\n", taskNum))

	sb.WriteString("\n---\n\n")
	sb.WriteString("## Notes\n\n")
	sb.WriteString("- [P] tasks = can run in parallel (different files, no dependencies)\n")
	sb.WriteString("- [USn] = belongs to User Story n\n")
	sb.WriteString("- Commit after each task or logical group\n")

	return sb.String(), nil
}

func parseSpecContent(content string) *ParsedSpec {
	spec := &ParsedSpec{}

	// Extract feature name from title
	titleRe := regexp.MustCompile(`(?m)^# (?:Feature Specification: )?(.+)$`)
	if match := titleRe.FindStringSubmatch(content); len(match) > 1 {
		spec.FeatureName = strings.TrimSpace(match[1])
	}

	// Extract user stories
	// Pattern: ### User Story N - Title (Priority: Pn)
	storyRe := regexp.MustCompile(`(?m)^### User Story (\d+) - ([^\(]+)\(Priority: (P\d+)\)`)
	storyMatches := storyRe.FindAllStringSubmatch(content, -1)

	for _, match := range storyMatches {
		storyNum := 1
		_, _ = fmt.Sscanf(match[1], "%d", &storyNum) // Ignore error, default to 1

		story := UserStory{
			Number:   storyNum,
			Title:    strings.TrimSpace(match[2]),
			Priority: strings.TrimSpace(match[3]),
		}

		// Extract description (text after the header until next section)
		descRe := regexp.MustCompile(`(?ms)### User Story ` + match[1] + `[^\n]*\n\n([^\n]+)`)
		if descMatch := descRe.FindStringSubmatch(content); len(descMatch) > 1 {
			story.Description = strings.TrimSpace(descMatch[1])
		}

		// Extract scenarios (Given/When/Then patterns)
		scenarioRe := regexp.MustCompile(`(?m)\*\*Given\*\* ([^,]+), \*\*When\*\* ([^,]+), \*\*Then\*\* (.+)$`)
		scenarioMatches := scenarioRe.FindAllStringSubmatch(content, -1)
		for _, sm := range scenarioMatches {
			scenario := fmt.Sprintf("Given %s, When %s, Then %s", sm[1], sm[2], sm[3])
			story.Scenarios = append(story.Scenarios, scenario)
		}

		spec.UserStories = append(spec.UserStories, story)
	}

	// Extract functional requirements
	// Pattern: - **FR-001**: Description
	reqRe := regexp.MustCompile(`(?m)^- \*\*(FR-\d+)\*\*: (.+)$`)
	reqMatches := reqRe.FindAllStringSubmatch(content, -1)
	for _, match := range reqMatches {
		req := Requirement{
			ID:          match[1],
			Description: strings.TrimSpace(match[2]),
		}
		spec.Requirements = append(spec.Requirements, req)
	}

	// Extract edge cases
	edgeCaseSection := extractSection(content, "Edge Cases")
	if edgeCaseSection != "" {
		edgeCaseRe := regexp.MustCompile(`(?m)^- (.+)$`)
		ecMatches := edgeCaseRe.FindAllStringSubmatch(edgeCaseSection, -1)
		for _, match := range ecMatches {
			spec.EdgeCases = append(spec.EdgeCases, strings.TrimSpace(match[1]))
		}
	}

	return spec
}

func parsePlanContent(content string) *ParsedPlan {
	plan := &ParsedPlan{}

	// Extract project structure section
	plan.ProjectStructure = extractSection(content, "Project Structure")

	// Extract technologies from content
	techRe := regexp.MustCompile(`(?i)(go|python|javascript|typescript|rust|java|kotlin|swift|ruby|php|c\+\+|c#|react|vue|angular|django|flask|spring|express|nextjs|nuxt)`)
	techMatches := techRe.FindAllString(content, -1)
	seen := make(map[string]bool)
	for _, tech := range techMatches {
		lower := strings.ToLower(tech)
		if !seen[lower] {
			plan.Technologies = append(plan.Technologies, tech)
			seen[lower] = true
		}
	}

	// Extract phases
	phaseRe := regexp.MustCompile(`(?m)^##+ Phase \d+[:\s]+(.+)$`)
	phaseMatches := phaseRe.FindAllStringSubmatch(content, -1)
	for _, match := range phaseMatches {
		plan.Phases = append(plan.Phases, strings.TrimSpace(match[1]))
	}

	return plan
}

func extractSection(content, sectionName string) string {
	re := regexp.MustCompile(`(?m)^##+ ` + regexp.QuoteMeta(sectionName) + `[^\n]*$([\s\S]*?)(^##+ |\z)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	// Here you will define your flags and configuration settings.
}
