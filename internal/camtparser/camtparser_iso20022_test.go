package camtparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewISO20022Parser(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	parser := NewISO20022Parser(logger)

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.GetLogger())
}

func TestISO20022Parser_ValidateFormat(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	parser := NewISO20022Parser(logger)

	tests := []struct {
		name        string
		content     string
		createFile  bool
		expectValid bool
		expectError bool
	}{
		{
			name: "valid CAMT.053",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt><Id>TEST</Id></Stmt>
	</BkToCstmrStmt>
</Document>`,
			createFile:  true,
			expectValid: true,
			expectError: false,
		},
		{
			name:        "empty file",
			content:     "",
			createFile:  true,
			expectValid: false,
			expectError: true,
		},
		{
			name: "no statements",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt></BkToCstmrStmt>
</Document>`,
			createFile:  true,
			expectValid: false,
			expectError: true,
		},
		{
			name:        "non-existent file",
			content:     "",
			createFile:  false,
			expectValid: false,
			expectError: true,
		},
		{
			name:        "invalid XML",
			content:     "not xml content",
			createFile:  true,
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.xml")

			if tt.createFile {
				err := os.WriteFile(testFile, []byte(tt.content), 0600)
				require.NoError(t, err)
			}

			isValid, err := parser.ValidateFormat(testFile)

			assert.Equal(t, tt.expectValid, isValid)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
