package report

import (
	"encoding/json"
	"encoding/xml"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportGenerator_GenerateReport_JSON(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	generator := NewReportGenerator(logger)

	// Create a sample report
	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\nfunc main() {}\n",
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Error Handling",
		Description:      "Errors must be handled.",
		Category:         "Quality",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "_ = err",
	}
	finding := models.Finding{
		Principle: principle,
		Status:    models.FindingStatusNonCompliant,
		Details:   "Error not handled at line 2",
	}
	report := models.NewComplianceReport([]models.CodebaseSection{section}, []models.ConstitutionPrinciple{principle})
	report.Findings = append(report.Findings, finding)
	report.OverallStatus = models.OverallStatusNonCompliant

	jsonBytes, err := generator.GenerateReport(report, "json")
	assert.NoError(t, err)
	assert.NotNil(t, jsonBytes)

	// Unmarshal to verify content
	var generatedReport models.ComplianceReport
	err = json.Unmarshal(jsonBytes, &generatedReport)
	assert.NoError(t, err)

	assert.Equal(t, report.ReportID, generatedReport.ReportID)
	assert.Equal(t, report.OverallStatus, generatedReport.OverallStatus)
	assert.Len(t, generatedReport.Findings, 1)
	assert.Equal(t, finding.Details, generatedReport.Findings[0].Details)
}

func TestReportGenerator_GenerateReport_XML(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	generator := NewReportGenerator(logger)

	// Create a sample report
	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\nfunc main() {}\n",
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Error Handling",
		Description:      "Errors must be handled.",
		Category:         "Quality",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "_ = err",
	}
	finding := models.Finding{
		Principle: principle,
		Status:    models.FindingStatusNonCompliant,
		Details:   "Error not handled at line 2",
	}
	report := models.NewComplianceReport([]models.CodebaseSection{section}, []models.ConstitutionPrinciple{principle})
	report.Findings = append(report.Findings, finding)
	report.OverallStatus = models.OverallStatusNonCompliant

	xmlBytes, err := generator.GenerateReport(report, "xml")
	assert.NoError(t, err)
	assert.NotNil(t, xmlBytes)

	// Check for XML header
	assert.Contains(t, string(xmlBytes), xml.Header)

	// Unmarshal to verify content (requires XML tags on models)
	var generatedReport models.ComplianceReport
	err = xml.Unmarshal(xmlBytes, &generatedReport)
	assert.NoError(t, err)

	assert.Equal(t, report.ReportID, generatedReport.ReportID)
	assert.Equal(t, report.OverallStatus, generatedReport.OverallStatus)
	assert.Len(t, generatedReport.Findings, 1)
	assert.Equal(t, finding.Details, generatedReport.Findings[0].Details)
}

func TestReportGenerator_GenerateReport_UnsupportedFormat(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	generator := NewReportGenerator(logger)
	report := models.NewComplianceReport([]models.CodebaseSection{}, []models.ConstitutionPrinciple{})

	_, err := generator.GenerateReport(report, "csv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported report format: csv")
}

func TestReportGenerator_GenerateReport_EmptyReport(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	generator := NewReportGenerator(logger)
	report := models.NewComplianceReport([]models.CodebaseSection{}, []models.ConstitutionPrinciple{})

	jsonBytes, err := generator.GenerateReport(report, "json")
	assert.NoError(t, err)
	assert.NotNil(t, jsonBytes)

	var generatedReport models.ComplianceReport
	err = json.Unmarshal(jsonBytes, &generatedReport)
	assert.NoError(t, err)
	assert.Len(t, generatedReport.Findings, 0)
	assert.Equal(t, models.OverallStatusCompliant, generatedReport.OverallStatus)
}
