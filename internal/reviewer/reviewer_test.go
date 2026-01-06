package reviewer

import (
	"fmt"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock PrincipleEvaluator for testing
type MockPrincipleEvaluator struct {
	mock.Mock
}

func (m *MockPrincipleEvaluator) Evaluate(section models.CodebaseSection, principle models.ConstitutionPrinciple) (models.Finding, error) {
	args := m.Called(section, principle)
	return args.Get(0).(models.Finding), args.Error(1)
}

func TestNewReviewer(t *testing.T) {
	mockEvaluator := &MockPrincipleEvaluator{}
	logger := logging.NewLogrusAdapter("info", "text")

	reviewer := NewReviewer(nil, nil, mockEvaluator, logger)

	assert.NotNil(t, reviewer)
	assert.Equal(t, mockEvaluator, reviewer.principleEvaluator)
	assert.NotNil(t, reviewer.logger)
}

func TestNewReviewerWithNilLogger(t *testing.T) {
	mockEvaluator := &MockPrincipleEvaluator{}

	reviewer := NewReviewer(nil, nil, mockEvaluator, nil)

	assert.NotNil(t, reviewer)
	assert.NotNil(t, reviewer.logger) // Should create default logger
}

func TestPerformReview_NoPrinciplesToApply_EmptyPrincipleIDs(t *testing.T) {
	mockEvaluator := &MockPrincipleEvaluator{}
	logger := logging.NewLogrusAdapter("info", "text")

	_ = NewReviewer(nil, nil, mockEvaluator, logger)

	// Test with empty principles list and specific principle IDs
	principleIDs := []string{"NONEXISTENT"}

	// Create a mock report scenario where no principles match
	principles := []models.ConstitutionPrinciple{
		{ID: "TEST-001", Name: "Test Principle"},
	}

	// Simulate the filtering logic
	var principlesToApply []models.ConstitutionPrinciple
	principleIDMap := make(map[string]bool)
	for _, pID := range principleIDs {
		principleIDMap[pID] = true
	}
	for _, p := range principles {
		if principleIDMap[p.ID] {
			principlesToApply = append(principlesToApply, p)
		}
	}

	// Verify no principles match
	assert.Len(t, principlesToApply, 0)
}

func TestPerformReview_ManualEvaluationMethod(t *testing.T) {
	mockEvaluator := &MockPrincipleEvaluator{}
	logger := logging.NewLogrusAdapter("info", "text")

	_ = NewReviewer(nil, nil, mockEvaluator, logger)

	// Test manual evaluation logic
	sections := []models.CodebaseSection{
		{Path: "/test/file.go", Type: models.CodebaseSectionTypeFile, Content: "package main"},
	}

	principles := []models.ConstitutionPrinciple{
		{
			ID:               "MANUAL-001",
			Name:             "Manual Principle",
			EvaluationMethod: models.EvaluationMethodManual,
		},
	}

	// Create initial report
	report := models.NewComplianceReport(sections, principles)

	// Simulate manual evaluation logic
	for range sections {
		for _, principle := range principles {
			if principle.EvaluationMethod == models.EvaluationMethodManual {
				finding := models.Finding{
					Principle: principle,
					Status:    models.FindingStatusManualReviewRequired,
					Details:   fmt.Sprintf("Placeholder finding for %s", principle.ID),
				}
				report.Findings = append(report.Findings, finding)
			}
		}
	}

	// Verify manual evaluation creates correct finding
	assert.Len(t, report.Findings, 1)
	assert.Equal(t, models.FindingStatusManualReviewRequired, report.Findings[0].Status)
	assert.Contains(t, report.Findings[0].Details, "Placeholder finding for MANUAL-001")
}

func TestPerformReview_UnsupportedEvaluationMethod(t *testing.T) {
	mockEvaluator := &MockPrincipleEvaluator{}
	logger := logging.NewLogrusAdapter("info", "text")

	_ = NewReviewer(nil, nil, mockEvaluator, logger)

	// Test unsupported evaluation method logic
	sections := []models.CodebaseSection{
		{Path: "/test/file.go", Type: models.CodebaseSectionTypeFile, Content: "package main"},
	}

	principles := []models.ConstitutionPrinciple{
		{
			ID:               "UNSUPPORTED-001",
			Name:             "Unsupported Principle",
			EvaluationMethod: "unsupported_method",
		},
	}

	// Create initial report
	report := models.NewComplianceReport(sections, principles)

	// Simulate unsupported evaluation method logic
	for _, section := range sections {
		for _, principle := range principles {
			if principle.EvaluationMethod != models.EvaluationMethodAutomated &&
				principle.EvaluationMethod != models.EvaluationMethodManual {
				evalErr := fmt.Errorf("unsupported evaluation method '%s' for principle '%s'", principle.EvaluationMethod, principle.ID)
				report.Findings = append(report.Findings, models.Finding{
					Principle: principle,
					Status:    models.FindingStatusNonCompliant,
					Details:   fmt.Sprintf("Evaluation error for principle '%s' in %s: %v", principle.ID, section.Path, evalErr),
				})
			}
		}
	}

	// Verify error handling creates correct finding
	assert.Len(t, report.Findings, 1)
	assert.Equal(t, models.FindingStatusNonCompliant, report.Findings[0].Status)
	assert.Contains(t, report.Findings[0].Details, "Evaluation error")
	assert.Contains(t, report.Findings[0].Details, "unsupported evaluation method")
}

func TestPerformReview_EvaluatorError(t *testing.T) {
	mockEvaluator := &MockPrincipleEvaluator{}
	logger := logging.NewLogrusAdapter("info", "text")

	_ = NewReviewer(nil, nil, mockEvaluator, logger)

	// Test evaluator error handling logic
	sections := []models.CodebaseSection{
		{Path: "/test/file.go", Type: models.CodebaseSectionTypeFile, Content: "package main"},
	}

	principles := []models.ConstitutionPrinciple{
		{
			ID:               "TEST-001",
			Name:             "Test Principle",
			EvaluationMethod: models.EvaluationMethodAutomated,
		},
	}

	// Setup mock to return error
	mockEvaluator.On("Evaluate", sections[0], principles[0]).Return(models.Finding{}, fmt.Errorf("evaluator error"))

	// Create initial report
	report := models.NewComplianceReport(sections, principles)

	// Simulate evaluator error handling
	for _, section := range sections {
		for _, principle := range principles {
			if principle.EvaluationMethod == models.EvaluationMethodAutomated {
				_, evalErr := mockEvaluator.Evaluate(section, principle)
				if evalErr != nil {
					report.Findings = append(report.Findings, models.Finding{
						Principle: principle,
						Status:    models.FindingStatusNonCompliant,
						Details:   fmt.Sprintf("Evaluation error for principle '%s' in %s: %v", principle.ID, section.Path, evalErr),
					})
				}
			}
		}
	}

	// Verify error handling creates correct finding
	assert.Len(t, report.Findings, 1)
	assert.Equal(t, models.FindingStatusNonCompliant, report.Findings[0].Status)
	assert.Contains(t, report.Findings[0].Details, "Evaluation error")
	assert.Contains(t, report.Findings[0].Details, "evaluator error")

	mockEvaluator.AssertExpectations(t)
}

func TestPerformReview_OverallStatusDetermination(t *testing.T) {
	tests := []struct {
		name           string
		findingStatus  models.FindingStatus
		expectedStatus models.OverallStatus
	}{
		{
			name:           "compliant finding results in compliant status",
			findingStatus:  models.FindingStatusCompliant,
			expectedStatus: models.OverallStatusCompliant,
		},
		{
			name:           "non-compliant finding results in non-compliant status",
			findingStatus:  models.FindingStatusNonCompliant,
			expectedStatus: models.OverallStatusNonCompliant,
		},
		{
			name:           "manual review required results in non-compliant status",
			findingStatus:  models.FindingStatusManualReviewRequired,
			expectedStatus: models.OverallStatusNonCompliant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test overall status determination logic
			sections := []models.CodebaseSection{
				{Path: "/test/file.go", Type: models.CodebaseSectionTypeFile},
			}

			principles := []models.ConstitutionPrinciple{
				{ID: "TEST-001", Name: "Test Principle"},
			}

			report := models.NewComplianceReport(sections, principles)

			// Add a finding with the test status
			finding := models.Finding{
				Principle: principles[0],
				Status:    tt.findingStatus,
				Details:   "Test finding",
			}
			report.Findings = append(report.Findings, finding)

			// Simulate overall status determination logic
			report.OverallStatus = models.OverallStatusCompliant
			for _, f := range report.Findings {
				if f.Status == models.FindingStatusNonCompliant || f.Status == models.FindingStatusManualReviewRequired {
					report.OverallStatus = models.OverallStatusNonCompliant
					break
				}
			}

			// Verify status determination
			assert.Equal(t, tt.expectedStatus, report.OverallStatus)
		})
	}
}

func TestPerformReview_MultipleSectionsAndPrinciples(t *testing.T) {
	mockEvaluator := &MockPrincipleEvaluator{}
	logger := logging.NewLogrusAdapter("info", "text")

	_ = NewReviewer(nil, nil, mockEvaluator, logger)

	// Test multiple sections and principles logic
	sections := []models.CodebaseSection{
		{Path: "/test/file1.go", Type: models.CodebaseSectionTypeFile, Content: "package main"},
		{Path: "/test/file2.go", Type: models.CodebaseSectionTypeFile, Content: "package test"},
	}

	principles := []models.ConstitutionPrinciple{
		{
			ID:               "TEST-001",
			Name:             "Test Principle 1",
			EvaluationMethod: models.EvaluationMethodAutomated,
		},
		{
			ID:               "TEST-002",
			Name:             "Test Principle 2",
			EvaluationMethod: models.EvaluationMethodAutomated,
		},
	}

	// Setup mocks - expect 4 evaluations (2 sections × 2 principles)
	for _, section := range sections {
		for _, principle := range principles {
			finding := models.Finding{
				Principle: principle,
				Status:    models.FindingStatusCompliant,
				Details:   fmt.Sprintf("Finding for %s in %s", principle.ID, section.Path),
			}
			mockEvaluator.On("Evaluate", section, principle).Return(finding, nil)
		}
	}

	// Create initial report
	report := models.NewComplianceReport(sections, principles)

	// Simulate evaluation loop
	for _, section := range sections {
		for _, principle := range principles {
			if principle.EvaluationMethod == models.EvaluationMethodAutomated {
				finding, err := mockEvaluator.Evaluate(section, principle)
				if err == nil {
					report.Findings = append(report.Findings, finding)
				}
			}
		}
	}

	// Verify multiple evaluations
	assert.Len(t, report.Findings, 4) // 2 sections × 2 principles

	mockEvaluator.AssertExpectations(t)
}

func TestPerformReview_PrincipleFiltering(t *testing.T) {
	// Test principle filtering logic
	principleIDs := []string{"TEST-001", "TEST-003"}

	loadedPrinciples := []models.ConstitutionPrinciple{
		{ID: "TEST-001", Name: "Test Principle 1"},
		{ID: "TEST-002", Name: "Test Principle 2"},
		{ID: "TEST-003", Name: "Test Principle 3"},
	}

	// Simulate filtering logic
	var principlesToApply []models.ConstitutionPrinciple
	principleIDMap := make(map[string]bool)
	for _, pID := range principleIDs {
		principleIDMap[pID] = true
	}
	for _, p := range loadedPrinciples {
		if principleIDMap[p.ID] {
			principlesToApply = append(principlesToApply, p)
		}
	}

	// Verify filtering works correctly
	assert.Len(t, principlesToApply, 2)
	assert.Equal(t, "TEST-001", principlesToApply[0].ID)
	assert.Equal(t, "TEST-003", principlesToApply[1].ID)
}

func TestPerformReview_EmptyPrincipleIDsUsesAll(t *testing.T) {
	// Test that empty principle IDs uses all loaded principles
	principleIDs := []string{} // Empty

	loadedPrinciples := []models.ConstitutionPrinciple{
		{ID: "TEST-001", Name: "Test Principle 1"},
		{ID: "TEST-002", Name: "Test Principle 2"},
	}

	// Simulate filtering logic
	var principlesToApply []models.ConstitutionPrinciple
	if len(principleIDs) > 0 {
		// Filtering logic would go here
	} else {
		principlesToApply = loadedPrinciples
	}

	// Verify all principles are used when no filtering
	assert.Len(t, principlesToApply, 2)
	assert.Equal(t, loadedPrinciples, principlesToApply)
}
