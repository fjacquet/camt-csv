package analyze

import (
	"regexp"
	"strings"
)

// Spec represents the parsed content of a spec.md file.
type Spec struct {
	Overview                  string
	FunctionalRequirements    string
	NonFunctionalRequirements string
	UserStories               string
	EdgeCases                 string
}

// Plan represents the parsed content of a plan.md file.
type Plan struct {
	Architecture         string
	DataModel            string
	Phases               string
	TechnicalConstraints string
}

// Task represents a single task from a tasks.md file.
type Task struct {
	ID          string
	Description string
	Phase       string
	IsParallel  bool
	Files       []string
}

// Constitution represents the parsed content of a constitution.md file.
type Constitution struct {
	Principles map[string]string
}

func parseSpec(content string) *Spec {
	spec := &Spec{}

	spec.Overview = extractSection(content, "Overview/Context")
	spec.FunctionalRequirements = extractSection(content, "Functional Requirements")
	spec.NonFunctionalRequirements = extractSection(content, "Non-Functional Requirements")
	spec.UserStories = extractSection(content, "User Stories")
	spec.EdgeCases = extractSection(content, "Edge Cases")

	return spec
}

func parsePlan(content string) *Plan {
	plan := &Plan{}

	plan.Architecture = extractSection(content, "Architecture/stack choices")
	plan.DataModel = extractSection(content, "Data Model references")
	plan.Phases = extractSection(content, "Phases")
	plan.TechnicalConstraints = extractSection(content, "Technical constraints")

	return plan
}

func parseTasks(content string) []Task {
	var tasks []Task
	// This is a very basic implementation. It assumes tasks are list items.
	// A more robust implementation would handle different task formats.
	re := regexp.MustCompile(`(?m)^- \[( |x)\] \*\*(\S+)\*\*: (.*)$`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		task := Task{
			ID:          match[2],
			Description: match[3],
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func parseConstitution(content string) *Constitution {
	constitution := &Constitution{
		Principles: make(map[string]string),
	}
	// This is a very basic implementation. It assumes principles are H3 headings.
	re := regexp.MustCompile(`(?m)^### (.*)$([\s\S]*?)(^### |\z)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		principleName := strings.TrimSpace(match[1])
		principleBody := strings.TrimSpace(match[2])
		constitution.Principles[principleName] = principleBody
	}

	return constitution
}

func extractSection(content, sectionName string) string {
	// This is a very basic way to extract sections. It assumes sections are H2 headings.
	// A more robust implementation would use a proper markdown parser.
	re := regexp.MustCompile(`(?m)^## ` + regexp.QuoteMeta(sectionName) + `\s*$([\s\S]*?)(^## |\z)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}
