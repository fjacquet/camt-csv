package analyze

import (
	"fmt"
	"regexp"
	"strings"
)

// Finding represents a single issue found during analysis.
type Finding struct {
	ID             string
	Category       string
	Severity       string
	Location       string
	Summary        string
	Recommendation string
}

// Report represents the full analysis report.
type Report struct {
	Findings           []Finding
	CoverageSummary    map[string][]string
	ConstitutionIssues []Finding
	UnmappedTasks      []Task
	Metrics            map[string]int
}

// Analyze performs the analysis of the parsed artifacts.
func Analyze(spec *Spec, plan *Plan, tasks []Task, constitution *Constitution) *Report {
	report := &Report{
		Metrics: make(map[string]int),
	}

	findings := []Finding{}

	ambiguityFindings := detectAmbiguity(spec, plan)
	findings = append(findings, ambiguityFindings...)
	report.Metrics["Ambiguity Count"] = len(ambiguityFindings)

	duplicationFindings := detectDuplication(spec)
	findings = append(findings, duplicationFindings...)
	report.Metrics["Duplication Count"] = len(duplicationFindings)

	underspecificationFindings := detectUnderspecification(spec, tasks)
	findings = append(findings, underspecificationFindings...)

	constitutionFindings := checkConstitutionAlignment(spec, plan, constitution)
	findings = append(findings, constitutionFindings...)
	report.ConstitutionIssues = constitutionFindings

	coverageFindings, unmappedTasks, coverageSummary := detectCoverageGaps(spec, tasks)
	findings = append(findings, coverageFindings...)
	report.UnmappedTasks = unmappedTasks
	report.CoverageSummary = coverageSummary

	inconsistencyFindings := detectInconsistency(spec, plan)
	findings = append(findings, inconsistencyFindings...)

	report.Findings = findings

	requirements := getRequirements(spec)
	report.Metrics["Total Requirements"] = len(requirements)
	report.Metrics["Total Tasks"] = len(tasks)

	coveredRequirements := 0
	for _, taskIDs := range coverageSummary {
		if len(taskIDs) > 0 {
			coveredRequirements++
		}
	}
	if len(requirements) > 0 {
		report.Metrics["Coverage %"] = (coveredRequirements * 100) / len(requirements)
	} else {
		report.Metrics["Coverage %"] = 100
	}

	criticalIssues := 0
	for _, finding := range findings {
		if finding.Severity == "CRITICAL" {
			criticalIssues++
		}
	}
	report.Metrics["Critical Issues Count"] = criticalIssues

	return report
}

func detectAmbiguity(spec *Spec, plan *Plan) []Finding {
	findings := []Finding{}
	vagueWords := []string{"fast", "scalable", "secure", "intuitive", "robust"}
	placeholders := []string{"TODO", "TKTK", "???", "<placeholder>"}

	checkContent := func(content, location string) {
		for _, word := range vagueWords {
			if strings.Contains(strings.ToLower(content), word) {
				findings = append(findings, Finding{
					ID:             fmt.Sprintf("A%d", len(findings)+1),
					Category:       "Ambiguity",
					Severity:       "HIGH",
					Location:       location,
					Summary:        fmt.Sprintf("Vague term '%s' used without measurable criteria.", word),
					Recommendation: "Define specific, measurable criteria for this requirement.",
				})
			}
		}
		for _, placeholder := range placeholders {
			if strings.Contains(content, placeholder) {
				findings = append(findings, Finding{
					ID:             fmt.Sprintf("A%d", len(findings)+1),
					Category:       "Ambiguity",
					Severity:       "HIGH",
					Location:       location,
					Summary:        fmt.Sprintf("Unresolved placeholder '%s' found.", placeholder),
					Recommendation: "Replace the placeholder with a concrete implementation detail.",
				})
			}
		}
	}

	checkContent(spec.FunctionalRequirements, "spec.md:Functional Requirements")
	checkContent(spec.NonFunctionalRequirements, "spec.md:Non-Functional Requirements")
	checkContent(plan.Architecture, "plan.md:Architecture")
	checkContent(plan.TechnicalConstraints, "plan.md:Technical constraints")

	return findings
}

func detectDuplication(spec *Spec) []Finding {
	findings := []Finding{}

	requirements := getRequirements(spec)

	for i := 0; i < len(requirements); i++ {
		for j := i + 1; j < len(requirements); j++ {
			if areSimilar(requirements[i], requirements[j]) {
				findings = append(findings, Finding{
					ID:             fmt.Sprintf("D%d", len(findings)+1),
					Category:       "Duplication",
					Severity:       "HIGH",
					Location:       "spec.md:Functional Requirements",
					Summary:        fmt.Sprintf("Found similar requirements: '%s' and '%s'", requirements[i], requirements[j]),
					Recommendation: "Merge the two requirements into a single, clearer one.",
				})
			}
		}
	}

	return findings
}

func detectUnderspecification(spec *Spec, tasks []Task) []Finding {
	findings := []Finding{}

	// Check for short requirements
	for _, req := range getRequirements(spec) {
		if len(strings.Fields(req)) < 5 {
			findings = append(findings, Finding{
				ID:             fmt.Sprintf("U%d", len(findings)+1),
				Category:       "Underspecification",
				Severity:       "MEDIUM",
				Location:       "spec.md",
				Summary:        fmt.Sprintf("Requirement is very short and may lack detail: '%s'", req),
				Recommendation: "Expand the requirement to include a clear object and measurable outcome.",
			})
		}
	}

	// Check for user stories without acceptance criteria
	if spec.UserStories != "" && !strings.Contains(strings.ToLower(spec.UserStories), "acceptance criteria") {
		findings = append(findings, Finding{
			ID:             fmt.Sprintf("U%d", len(findings)+1),
			Category:       "Underspecification",
			Severity:       "HIGH",
			Location:       "spec.md:User Stories",
			Summary:        "User stories section is present but does not mention 'acceptance criteria'.",
			Recommendation: "Ensure each user story has clear and testable acceptance criteria.",
		})
	}

	return findings
}

func checkConstitutionAlignment(spec *Spec, plan *Plan, constitution *Constitution) []Finding {
	findings := []Finding{}

	for name, principle := range constitution.Principles {
		if strings.Contains(principle, "MUST NOT") {
			re := regexp.MustCompile(`MUST NOT (.*)`)
			matches := re.FindStringSubmatch(principle)
			if len(matches) > 1 {
				forbidden := matches[1]
				checkContent := func(content, location string) {
					if strings.Contains(content, forbidden) {
						findings = append(findings, Finding{
							ID:             fmt.Sprintf("C%d", len(findings)+1),
							Category:       "Constitution",
							Severity:       "CRITICAL",
							Location:       location,
							Summary:        fmt.Sprintf("Violation of principle '%s': found forbidden '%s'", name, forbidden),
							Recommendation: "Remove the forbidden element or revise the spec/plan to align with the constitution.",
						})
					}
				}
				checkContent(spec.FunctionalRequirements, "spec.md:Functional Requirements")
				checkContent(spec.NonFunctionalRequirements, "spec.md:Non-Functional Requirements")
				checkContent(plan.Architecture, "plan.md:Architecture")
			}
		}
	}

	return findings
}

func detectCoverageGaps(spec *Spec, tasks []Task) ([]Finding, []Task, map[string][]string) {
	findings := []Finding{}
	unmappedTasks := []Task{}
	coverageSummary := make(map[string][]string)

	requirements := getRequirements(spec)
	reqTaskMap := make(map[string][]string)

	for _, req := range requirements {
		reqTaskMap[req] = []string{}
	}

	taskReqMap := make(map[string]string)

	for _, task := range tasks {
		mapped := false
		for _, req := range requirements {
			if strings.Contains(task.Description, req) {
				reqTaskMap[req] = append(reqTaskMap[req], task.ID)
				taskReqMap[task.ID] = req
				mapped = true
			}
		}
		if !mapped {
			unmappedTasks = append(unmappedTasks, task)
		}
	}

	for req, taskIDs := range reqTaskMap {
		if len(taskIDs) == 0 {
			findings = append(findings, Finding{
				ID:             fmt.Sprintf("G%d", len(findings)+1),
				Category:       "Coverage Gap",
				Severity:       "CRITICAL",
				Location:       "spec.md",
				Summary:        fmt.Sprintf("Requirement has no associated tasks: '%s'", req),
				Recommendation: "Create one or more tasks to implement this requirement.",
			})
		}
		coverageSummary[req] = taskIDs
	}

	return findings, unmappedTasks, coverageSummary
}

func detectInconsistency(spec *Spec, plan *Plan) []Finding {
	findings := []Finding{}

	specKeywords := getKeywords(spec.FunctionalRequirements + " " + spec.NonFunctionalRequirements)
	planKeywords := getKeywords(plan.Architecture + " " + plan.DataModel)

	for sk := range specKeywords {
		found := false
		for pk := range planKeywords {
			if strings.ToLower(sk) == strings.ToLower(pk) {
				found = true
				break
			}
		}
		if !found {
			for pk := range planKeywords {
				if areSimilar(sk, pk) {
					findings = append(findings, Finding{
						ID:             fmt.Sprintf("I%d", len(findings)+1),
						Category:       "Inconsistency",
						Severity:       "MEDIUM",
						Location:       "spec.md, plan.md",
						Summary:        fmt.Sprintf("Potential terminology drift: '%s' in spec.md might be the same as '%s' in plan.md.", sk, pk),
						Recommendation: "Use consistent terminology across all artifacts.",
					})
				}
			}
		}
	}

	return findings
}

func getKeywords(content string) map[string]bool {
	keywords := make(map[string]bool)
	re := regexp.MustCompile(`\b[A-Z][a-zA-Z]*\b`)
	matches := re.FindAllString(content, -1)
	for _, match := range matches {
		keywords[match] = true
	}
	return keywords
}

func getRequirements(spec *Spec) []string {
	reqs := []string{}
	reqs = append(reqs, strings.Split(spec.FunctionalRequirements, "\n")...)
	reqs = append(reqs, strings.Split(spec.NonFunctionalRequirements, "\n")...)

	filteredReqs := []string{}
	for _, req := range reqs {
		if strings.TrimSpace(req) != "" {
			filteredReqs = append(filteredReqs, strings.TrimSpace(req))
		}
	}

	return filteredReqs
}

func areSimilar(s1, s2 string) bool {
	words1 := strings.Fields(strings.ToLower(s1))
	words2 := strings.Fields(strings.ToLower(s2))

	if len(words1) == 0 || len(words2) == 0 {
		return false
	}

	commonWords := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				commonWords++
			}
		}
	}

	similarity := float64(commonWords) / float64(len(words1)+len(words2)-commonWords)
	return similarity > 0.7
}
