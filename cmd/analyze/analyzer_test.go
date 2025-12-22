package analyze

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAreSimilar(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected bool
	}{
		{
			name:     "identical strings are similar",
			s1:       "hello world",
			s2:       "hello world",
			expected: true,
		},
		{
			name:     "strings with high overlap are similar",
			s1:       "the user should be able to login",
			s2:       "the user should be able to authenticate",
			expected: true,
		},
		{
			name:     "completely different strings are not similar",
			s1:       "apple orange banana",
			s2:       "car truck motorcycle",
			expected: false,
		},
		{
			name:     "empty strings are not similar",
			s1:       "",
			s2:       "",
			expected: false,
		},
		{
			name:     "one empty string is not similar",
			s1:       "hello",
			s2:       "",
			expected: false,
		},
		{
			name:     "case insensitive comparison",
			s1:       "HELLO WORLD",
			s2:       "hello world",
			expected: true,
		},
		{
			name:     "single common word below threshold",
			s1:       "the quick brown fox",
			s2:       "the lazy dog jumps",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := areSimilar(tt.s1, tt.s2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetKeywords(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "extracts capitalized words",
			content:  "The User should Login to the System",
			expected: []string{"The", "User", "Login", "System"},
		},
		{
			name:     "handles empty content",
			content:  "",
			expected: []string{},
		},
		{
			name:     "no capitalized words",
			content:  "all lowercase text here",
			expected: []string{},
		},
		{
			name:     "camelCase words",
			content:  "The userService and dataManager",
			expected: []string{"The"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getKeywords(tt.content)
			for _, expected := range tt.expected {
				assert.True(t, result[expected], "expected keyword %s to be present", expected)
			}
		})
	}
}

func TestGetRequirements(t *testing.T) {
	tests := []struct {
		name     string
		spec     *Spec
		expected int
	}{
		{
			name: "extracts functional and non-functional requirements",
			spec: &Spec{
				FunctionalRequirements:    "REQ1: Login\nREQ2: Logout",
				NonFunctionalRequirements: "NFR1: Performance",
			},
			expected: 3,
		},
		{
			name: "filters empty lines",
			spec: &Spec{
				FunctionalRequirements:    "REQ1: Login\n\n\nREQ2: Logout\n",
				NonFunctionalRequirements: "",
			},
			expected: 2,
		},
		{
			name: "handles empty spec",
			spec: &Spec{
				FunctionalRequirements:    "",
				NonFunctionalRequirements: "",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRequirements(tt.spec)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestDetectAmbiguity(t *testing.T) {
	tests := []struct {
		name          string
		spec          *Spec
		plan          *Plan
		expectedCount int
		checkSeverity string
		checkCategory string
	}{
		{
			name: "detects vague terms",
			spec: &Spec{
				FunctionalRequirements: "The system must be fast and scalable",
			},
			plan:          &Plan{},
			expectedCount: 2, // "fast" and "scalable"
			checkSeverity: "HIGH",
			checkCategory: "Ambiguity",
		},
		{
			name: "detects placeholders",
			spec: &Spec{
				FunctionalRequirements: "TODO: Define this requirement",
			},
			plan:          &Plan{},
			expectedCount: 1,
			checkSeverity: "HIGH",
			checkCategory: "Ambiguity",
		},
		{
			name: "no findings for clean spec",
			spec: &Spec{
				FunctionalRequirements: "The system shall store user data in PostgreSQL",
			},
			plan:          &Plan{},
			expectedCount: 0,
		},
		{
			name: "detects in plan architecture",
			spec: &Spec{},
			plan: &Plan{
				Architecture: "Use a robust microservices architecture",
			},
			expectedCount: 1, // "robust"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := detectAmbiguity(tt.spec, tt.plan)
			assert.Len(t, findings, tt.expectedCount)
			if tt.expectedCount > 0 && tt.checkSeverity != "" {
				assert.Equal(t, tt.checkSeverity, findings[0].Severity)
			}
			if tt.expectedCount > 0 && tt.checkCategory != "" {
				assert.Equal(t, tt.checkCategory, findings[0].Category)
			}
		})
	}
}

func TestDetectDuplication(t *testing.T) {
	tests := []struct {
		name          string
		spec          *Spec
		expectedCount int
	}{
		{
			name: "detects similar requirements",
			spec: &Spec{
				FunctionalRequirements: "The user should be able to login\nThe user should be able to authenticate",
			},
			expectedCount: 1,
		},
		{
			name: "no duplication for distinct requirements",
			spec: &Spec{
				FunctionalRequirements: "The user can create accounts\nThe admin can delete users",
			},
			expectedCount: 0,
		},
		{
			name: "empty spec has no duplication",
			spec: &Spec{
				FunctionalRequirements: "",
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := detectDuplication(tt.spec)
			assert.Len(t, findings, tt.expectedCount)
		})
	}
}

func TestDetectUnderspecification(t *testing.T) {
	tests := []struct {
		name          string
		spec          *Spec
		tasks         []Task
		expectedCount int
	}{
		{
			name: "detects short requirements",
			spec: &Spec{
				FunctionalRequirements: "Login\nUser signup with email and password validation",
			},
			tasks:         []Task{},
			expectedCount: 1, // "Login" is too short
		},
		{
			name: "detects missing acceptance criteria",
			spec: &Spec{
				UserStories: "As a user, I want to login so that I can access my account",
			},
			tasks:         []Task{},
			expectedCount: 1,
		},
		{
			name: "no issues with well-specified content",
			spec: &Spec{
				FunctionalRequirements: "The system shall validate user credentials against the database",
				UserStories:            "As a user, I want to login. Acceptance criteria: valid credentials succeed",
			},
			tasks:         []Task{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := detectUnderspecification(tt.spec, tt.tasks)
			assert.Len(t, findings, tt.expectedCount)
		})
	}
}

func TestDetectCoverageGaps(t *testing.T) {
	tests := []struct {
		name             string
		spec             *Spec
		tasks            []Task
		expectedGaps     int
		expectedUnmapped int
	}{
		{
			name: "detects uncovered requirements",
			spec: &Spec{
				FunctionalRequirements: "Implement login\nImplement logout",
			},
			tasks: []Task{
				{ID: "T1", Description: "Implement login functionality"},
			},
			expectedGaps:     1, // logout not covered
			expectedUnmapped: 0,
		},
		{
			name: "detects unmapped tasks",
			spec: &Spec{
				FunctionalRequirements: "Implement login",
			},
			tasks: []Task{
				{ID: "T1", Description: "Setup CI/CD pipeline"},
			},
			expectedGaps:     1, // login not covered
			expectedUnmapped: 1, // CI/CD task unmapped
		},
		{
			name: "full coverage no gaps",
			spec: &Spec{
				FunctionalRequirements: "Implement login",
			},
			tasks: []Task{
				{ID: "T1", Description: "Implement login feature"},
			},
			expectedGaps:     0,
			expectedUnmapped: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, unmapped, _ := detectCoverageGaps(tt.spec, tt.tasks)
			assert.Len(t, findings, tt.expectedGaps)
			assert.Len(t, unmapped, tt.expectedUnmapped)
		})
	}
}

func TestAnalyze(t *testing.T) {
	t.Run("generates complete report", func(t *testing.T) {
		spec := &Spec{
			FunctionalRequirements:    "The system must be fast\nImplement user login",
			NonFunctionalRequirements: "High availability",
			UserStories:               "As a user, I want to login",
		}
		plan := &Plan{
			Architecture: "Microservices with secure API gateway",
		}
		tasks := []Task{
			{ID: "T1", Description: "Implement user login"},
		}
		constitution := &Constitution{
			Principles: map[string]string{
				"Security": "All APIs MUST NOT expose internal IDs",
			},
		}

		report := Analyze(spec, plan, tasks, constitution)

		assert.NotNil(t, report)
		assert.NotNil(t, report.Findings)
		assert.NotNil(t, report.Metrics)
		assert.NotNil(t, report.CoverageSummary)

		// Check metrics are populated
		assert.Contains(t, report.Metrics, "Total Requirements")
		assert.Contains(t, report.Metrics, "Total Tasks")
		assert.Contains(t, report.Metrics, "Coverage %")
		assert.Contains(t, report.Metrics, "Critical Issues Count")
	})

	t.Run("handles empty inputs", func(t *testing.T) {
		spec := &Spec{}
		plan := &Plan{}
		tasks := []Task{}
		constitution := &Constitution{}

		report := Analyze(spec, plan, tasks, constitution)

		assert.NotNil(t, report)
		assert.Equal(t, 0, report.Metrics["Total Tasks"])
		assert.Equal(t, 100, report.Metrics["Coverage %"]) // 0/0 = 100%
	})
}
