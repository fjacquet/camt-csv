package models

import (
	"time"

	"github.com/google/uuid"
)

// CodebaseSectionType defines the type of codebase section (file or directory).
type CodebaseSectionType string

const (
	// CodebaseSectionTypeFile indicates that the section is a single file.
	CodebaseSectionTypeFile CodebaseSectionType = "File"
	// CodebaseSectionTypeDirectory indicates that the section is a directory.
	CodebaseSectionTypeDirectory CodebaseSectionType = "Directory"
)

// CodebaseSection represents an entire file or directory within the project that is subject to compliance review.
type CodebaseSection struct {
	Path    string              `json:"path" yaml:"path"`                           // Absolute path to the file or directory
	Type    CodebaseSectionType `json:"type" yaml:"type"`                           // File or Directory
	Content string              `json:"content,omitempty" yaml:"content,omitempty"` // Content of the file, if Type is File
}

// EvaluationMethod defines how a constitution principle is evaluated.
type EvaluationMethod string

const (
	// EvaluationMethodAutomated indicates that the principle can be checked by an automated tool.
	EvaluationMethodAutomated EvaluationMethod = "Automated"
	// EvaluationMethodManual indicates that the principle requires human judgment for evaluation.
	EvaluationMethodManual EvaluationMethod = "Manual"
)

// ConstitutionPrinciple represents a specific rule or guideline from the project constitution.
type ConstitutionPrinciple struct {
	ID               string           `json:"id" yaml:"id"`                               // Unique identifier for the principle
	Name             string           `json:"name" yaml:"name"`                           // Short, descriptive name
	Description      string           `json:"description" yaml:"description"`             // Detailed explanation of the principle
	Category         string           `json:"category" yaml:"category"`                   // e.g., "Error Handling", "Testing", "Security"
	EvaluationMethod EvaluationMethod `json:"evaluation_method" yaml:"evaluation_method"` // Automated or Manual
	Pattern          string           `json:"pattern,omitempty" yaml:"pattern,omitempty"` // Regex or other pattern for automated checks
}

// OverallStatus defines the overall compliance status of a report.
type OverallStatus string

const (
	// OverallStatusCompliant indicates that all reviewed sections are compliant.
	OverallStatusCompliant OverallStatus = "Compliant"
	// OverallStatusNonCompliant indicates that at least one reviewed section is non-compliant or requires manual review.
	OverallStatusNonCompliant OverallStatus = "NonCompliant"
	// OverallStatusPartialCompliance indicates that some sections are compliant, but others are not or require manual review.
	OverallStatusPartialCompliance OverallStatus = "PartialCompliance"
)

// FindingStatus defines the status of a specific finding.
type FindingStatus string

const (
	// FindingStatusCompliant indicates that the codebase section is compliant with the principle.
	FindingStatusCompliant FindingStatus = "Compliant"
	// FindingStatusNonCompliant indicates that the codebase section is not compliant with the principle.
	FindingStatusNonCompliant FindingStatus = "NonCompliant"
	// FindingStatusManualReviewRequired indicates that the principle requires manual review.
	FindingStatusManualReviewRequired FindingStatus = "ManualReviewRequired"
)

// ComplianceReport represents a document detailing the findings of a compliance review.
type ComplianceReport struct {
	ReportID           string                  `json:"reportId" yaml:"reportId"`
	Timestamp          time.Time               `json:"timestamp" yaml:"timestamp"`
	CodebaseSection    []CodebaseSection       `json:"codebaseSection" yaml:"codebaseSection"`
	PrinciplesReviewed []ConstitutionPrinciple `json:"principlesReviewed" yaml:"principlesReviewed"`
	OverallStatus      OverallStatus           `json:"overallStatus" yaml:"overallStatus"`
	Findings           []Finding               `json:"findings" yaml:"findings"`
}

// Finding details a specific compliance status for a principle within a codebase section.
type Finding struct {
	Principle                ConstitutionPrinciple `json:"principle" yaml:"principle"`
	Status                   FindingStatus         `json:"status" yaml:"status"`
	Details                  string                `json:"details" yaml:"details"`
	ProposedCorrectiveAction *CorrectiveAction     `json:"proposedCorrectiveAction,omitempty" yaml:"proposedCorrectiveAction,omitempty"`
}

// CorrectiveActionStatus defines the status of a proposed corrective action.
type CorrectiveActionStatus string

const (
	// CorrectiveActionStatusProposed indicates that the action has been proposed.
	CorrectiveActionStatusProposed CorrectiveActionStatus = "Proposed"
	// CorrectiveActionStatusImplemented indicates that the action has been implemented.
	CorrectiveActionStatusImplemented CorrectiveActionStatus = "Implemented"
	// CorrectiveActionStatusRejected indicates that the action has been rejected.
	CorrectiveActionStatusRejected CorrectiveActionStatus = "Rejected"
)

// Severity defines the severity of a corrective action.
type Severity string

const (
	// SeverityHigh indicates a high-priority corrective action.
	SeverityHigh Severity = "High"
	// SeverityMedium indicates a medium-priority corrective action.
	SeverityMedium Severity = "Medium"
	// SeverityLow indicates a low-priority corrective action.
	SeverityLow Severity = "Low"
)

// CorrectiveAction represents a proposed change or action to address a non-compliant area.
type CorrectiveAction struct {
	ActionID       string                 `json:"actionId" yaml:"actionId"`
	Description    string                 `json:"description" yaml:"description"`
	Severity       Severity               `json:"severity" yaml:"severity"`
	Status         CorrectiveActionStatus `json:"status" yaml:"status"`
	RelatedFinding *Finding               `json:"-" yaml:"-"` // Avoid circular dependency in JSON/YAML
}

// NewComplianceReport creates a new ComplianceReport with a generated ID and timestamp.
func NewComplianceReport(sections []CodebaseSection, principles []ConstitutionPrinciple) *ComplianceReport {
	return &ComplianceReport{
		ReportID:           uuid.New().String(),
		Timestamp:          time.Now(),
		CodebaseSection:    sections,
		PrinciplesReviewed: principles,
		Findings:           []Finding{},
		OverallStatus:      OverallStatusCompliant, // Default to compliant, update as findings are added
	}
}
