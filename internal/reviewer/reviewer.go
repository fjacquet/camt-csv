package reviewer

import (
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/scanner"
	"fmt"
)

// Reviewer orchestrates the codebase review process.
type Reviewer struct {
	logger             logging.Logger
	codebaseScanner    *scanner.CodebaseScanner
	constitutionLoader *parser.ConstitutionLoader
	principleEvaluator PrincipleEvaluator
}

// NewReviewer creates a new instance of Reviewer.
// It takes a CodebaseScanner, ConstitutionLoader, and PrincipleEvaluator as dependencies.
func NewReviewer(codebaseScanner *scanner.CodebaseScanner, constitutionLoader *parser.ConstitutionLoader, principleEvaluator PrincipleEvaluator, logger logging.Logger) *Reviewer {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	return &Reviewer{
		logger:             logger.WithField("component", "Reviewer"),
		codebaseScanner:    codebaseScanner,
		constitutionLoader: constitutionLoader,
		principleEvaluator: principleEvaluator,
	}
}

// PerformReview executes the entire codebase review process.
// It scans the provided paths, loads constitution principles, evaluates them,
// and returns a ComplianceReport. It returns an error if any step fails.
func (r *Reviewer) PerformReview(paths, constitutionFilePaths, principleIDs []string, outputFormat string) (*models.ComplianceReport, error) {
	// 1. Scan codebase paths
	sections, err := r.codebaseScanner.ScanPaths(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to scan codebase paths: %w", err)
	}

	// 2. Load constitution files
	loadedPrinciples, err := r.constitutionLoader.LoadConstitutionFiles(constitutionFilePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to load constitution files: %w", err)
	}

	// 3. Filter principles (if principleIDs are provided)
	var principlesToApply []models.ConstitutionPrinciple
	if len(principleIDs) > 0 {
		principleIDMap := make(map[string]bool)
		for _, pID := range principleIDs {
			principleIDMap[pID] = true
		}
		for _, p := range loadedPrinciples {
			if principleIDMap[p.ID] {
				principlesToApply = append(principlesToApply, p)
			}
		}
	} else {
		principlesToApply = loadedPrinciples
	}

	if len(principlesToApply) == 0 {
		return nil, fmt.Errorf("no constitution principles to apply. Check provided constitution files and principle IDs.")
	}

	// 4. Create initial report
	report := models.NewComplianceReport(sections, principlesToApply)

	// 5. Evaluate principles for each codebase section
	for _, section := range sections {
		for _, principle := range principlesToApply {
			var finding models.Finding
			var evalErr error

			switch principle.EvaluationMethod {
			case models.EvaluationMethodAutomated:
				finding, evalErr = r.principleEvaluator.Evaluate(section, principle)
			case models.EvaluationMethodManual:
				finding = models.Finding{
					Principle: principle,
					Status:    models.FindingStatusManualReviewRequired,
					Details:   fmt.Sprintf("Placeholder finding for %s", principle.ID),
				}
			default:
				evalErr = fmt.Errorf("unsupported evaluation method '%s' for principle '%s'", principle.EvaluationMethod, principle.ID)
			}

			if evalErr != nil {
				r.logger.WithError(evalErr).WithFields(
					logging.Field{Key: "principle", Value: principle.ID},
					logging.Field{Key: "section", Value: section.Path},
				).Error("Error evaluating principle for section")
				// Add a finding for the evaluation error itself
				report.Findings = append(report.Findings, models.Finding{
					Principle: principle,
					Status:    models.FindingStatusNonCompliant,
					Details:   fmt.Sprintf("Evaluation error for principle '%s' in %s: %v", principle.ID, section.Path, evalErr),
				})
			} else {
				report.Findings = append(report.Findings, finding)
			}
		}
	}

	// 6. Determine overall status
	report.OverallStatus = models.OverallStatusCompliant
	for _, f := range report.Findings {
		if f.Status == models.FindingStatusNonCompliant || f.Status == models.FindingStatusManualReviewRequired {
			report.OverallStatus = models.OverallStatusNonCompliant
			break
		}
	}

	return report, nil
}
