// Package camtparser provides functionality to parse CAMT.053 XML files and convert them to CSV format.
package camtparser

import (
	"encoding/xml"
	"fmt"
	"os"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// ISO20022Parser is a parser implementation for CAMT.053 files using ISO20022 standard definitions
type ISO20022Parser struct {
	parser.BaseParser
}

// NewISO20022Parser creates a new ISO20022 parser for CAMT.053 files
func NewISO20022Parser(logger logging.Logger) *ISO20022Parser {
	return &ISO20022Parser{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// ValidateFormat checks if the file is a valid CAMT.053 XML file
func (p *ISO20022Parser) ValidateFormat(filePath string) (bool, error) {
	p.GetLogger().Info("Validating CAMT.053 format",
		logging.Field{Key: "file", Value: filePath})

	// Try to open and read the file
	xmlFile, err := os.Open(filePath) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return false, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		if err := xmlFile.Close(); err != nil {
			p.GetLogger().Warn("Failed to close XML file",
				logging.Field{Key: "error", Value: err})
		}
	}()

	// Read the file content
	xmlBytes, err := os.ReadFile(filePath) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return false, fmt.Errorf("error reading file: %w", err)
	}

	// Check if file is empty
	if len(xmlBytes) == 0 {
		p.GetLogger().Info("File is empty",
			logging.Field{Key: "file", Value: filePath})
		return false, fmt.Errorf("file is empty")
	}

	// Try to unmarshal the XML data into our ISO20022 document structure
	var document models.ISO20022Document
	if err := xml.Unmarshal(xmlBytes, &document); err != nil {
		p.GetLogger().Info("File is not a valid CAMT.053 XML",
			logging.Field{Key: "file", Value: filePath})
		return false, fmt.Errorf("invalid XML format: %w", err)
	}

	// Check if we have at least one statement
	if len(document.BkToCstmrStmt.Stmt) == 0 {
		p.GetLogger().Info("File is not a valid CAMT.053 XML (no statements)",
			logging.Field{Key: "file", Value: filePath})
		return false, fmt.Errorf("no statements found in CAMT.053 file")
	}

	p.GetLogger().Info("File is a valid CAMT.053 XML",
		logging.Field{Key: "file", Value: filePath})
	return true, nil
}
