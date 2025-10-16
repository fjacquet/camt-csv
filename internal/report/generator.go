package report

import (
	"encoding/json"
	"encoding/xml"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fmt"

	"github.com/sirupsen/logrus"
)

// ReportGenerator provides functionality to generate compliance reports in various formats.
type ReportGenerator struct {
	logger *logrus.Logger
}

// NewReportGenerator creates a new instance of ReportGenerator.
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		logger: logging.GetLogger().WithField("component", "ReportGenerator").Logger,
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
		g.logger.Errorf("Failed to marshal JSON report: %v", err)
		return nil, fmt.Errorf("failed to marshal JSON report: %w", err)
	}
	return jsonReport, nil
}

// generateXMLReport generates a compliance report in XML format.
func (g *ReportGenerator) generateXMLReport(report *models.ComplianceReport) ([]byte, error) {
	xmlReport, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		g.logger.Errorf("Failed to marshal XML report: %v", err)
		return nil, fmt.Errorf("failed to marshal XML report: %w", err)
	}
	return []byte(xml.Header + string(xmlReport)), nil
}
