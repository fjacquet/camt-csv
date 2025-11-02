package report

import (
	"encoding/json"
	"encoding/xml"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fmt"
)

// ReportGenerator provides functionality to generate compliance reports in various formats.
type ReportGenerator struct {
	logger logging.Logger
}

// NewReportGenerator creates a new instance of ReportGenerator.
func NewReportGenerator(logger logging.Logger) *ReportGenerator {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	return &ReportGenerator{
		logger: logger.WithField("component", "ReportGenerator"),
	}
}

// GenerateReport generates a compliance report in the specified format (json or xml).
// It returns the report as a byte slice and an error if generation fails or the format is unsupported.
func (g *ReportGenerator) GenerateReport(report *models.ComplianceReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return g.generateJSONReport(report)
	case "xml":
		return g.generateXMLReport(report)
	default:
		return nil, fmt.Errorf("unsupported report format: %s", format)
	}
}

// generateJSONReport generates a compliance report in JSON format.
func (g *ReportGenerator) generateJSONReport(report *models.ComplianceReport) ([]byte, error) {
	jsonReport, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		g.logger.WithError(err).Error("Failed to marshal JSON report")
		return nil, fmt.Errorf("failed to marshal JSON report: %w", err)
	}
	return jsonReport, nil
}

// generateXMLReport generates a compliance report in XML format.
func (g *ReportGenerator) generateXMLReport(report *models.ComplianceReport) ([]byte, error) {
	xmlReport, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		g.logger.WithError(err).Error("Failed to marshal XML report")
		return nil, fmt.Errorf("failed to marshal XML report: %w", err)
	}
	return []byte(xml.Header + string(xmlReport)), nil
}
