package reviewer

import (
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
)

// PrincipleEvaluator defines the interface for evaluating constitution principles.
type PrincipleEvaluator interface {
	// Evaluate checks a codebase section against a constitution principle.
	// It returns a Finding detailing the compliance status and an error if evaluation fails.
	Evaluate(section models.CodebaseSection, principle models.ConstitutionPrinciple) (models.Finding, error)
}

// AutomatedPrincipleEvaluator implements PrincipleEvaluator for automated checks.
type AutomatedPrincipleEvaluator struct {
	logger *logrus.Logger
}

// NewAutomatedPrincipleEvaluator creates a new instance of AutomatedPrincipleEvaluator.
func NewAutomatedPrincipleEvaluator() *AutomatedPrincipleEvaluator {
	return &AutomatedPrincipleEvaluator{
		logger: logging.GetLogger().WithField("component", "AutomatedPrincipleEvaluator").Logger,
	}
}

// Evaluate checks a codebase section against an automated principle using regex patterns.
// It returns a Finding with Compliant or NonCompliant status based on pattern matching.
// It returns an error if the principle is not automated, the section is not a file, or the pattern is invalid.
func (e *AutomatedPrincipleEvaluator) Evaluate(section models.CodebaseSection, principle models.ConstitutionPrinciple) (models.Finding, error) {
	if principle.EvaluationMethod != models.EvaluationMethodAutomated {
		return models.Finding{}, fmt.Errorf("principle %s is not an automated evaluation method", principle.ID)
	}

	if section.Type != models.CodebaseSectionTypeFile {
		return models.Finding{}, fmt.Errorf("automated evaluation only supports file sections, got %s for %s", section.Type, section.Path)
	}

	if principle.Pattern == "" {
		return models.Finding{}, fmt.Errorf("automated principle %s has no pattern defined", principle.ID)
	}

	regex, err := regexp.Compile(principle.Pattern)
	if err != nil {
		e.logger.Errorf("Invalid regex pattern for principle %s: %v", principle.ID, err)
		return models.Finding{}, fmt.Errorf("invalid regex pattern for principle %s: %w", principle.ID, err)
	}

	// Check if the pattern is found in the file content
	if regex.MatchString(section.Content) {
		return models.Finding{
			Principle: principle,
			Status:    models.FindingStatusCompliant,
			Details:   fmt.Sprintf("Pattern '%s' found in %s", principle.Pattern, section.Path),
		}, nil
	} else {
		return models.Finding{
			Principle: principle,
			Status:    models.FindingStatusNonCompliant,
			Details:   fmt.Sprintf("Pattern '%s' not found in %s", principle.Pattern, section.Path),
		}, nil
	}
}
