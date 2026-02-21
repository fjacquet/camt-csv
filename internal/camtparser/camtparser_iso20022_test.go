package camtparser

import (
	"context"
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

func TestAdapter_BatchConvert(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")

	err := os.MkdirAll(inputDir, 0750)
	require.NoError(t, err)

	// Create valid CAMT XML file
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">100.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<Sts>BOOK</Sts>
				<BookgDt><Dt>2023-05-15</Dt></BookgDt>
				<ValDt><Dt>2023-05-15</Dt></ValDt>
				<AcctSvcrRef>REF123</AcctSvcrRef>
				<NtryDtls><TxDtls><Refs><EndToEndId>E2E123</EndToEndId></Refs></TxDtls></NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`

	validFile := filepath.Join(inputDir, "statement.xml")
	err = os.WriteFile(validFile, []byte(validXML), 0600)
	require.NoError(t, err)

	// Create invalid file (should be skipped)
	invalidFile := filepath.Join(inputDir, "invalid.xml")
	err = os.WriteFile(invalidFile, []byte("not xml"), 0600)
	require.NoError(t, err)

	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	count, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestIsIBANFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid CH IBAN", "CH9300762011623852957", true},
		{"valid DE IBAN", "DE89370400440532013000", true},
		{"valid FR IBAN", "FR1420041010050500013M02606", true},
		{"lowercase valid", "ch9300762011623852957", true},
		{"too short", "CH93", false},
		{"too long", "CH93007620116238529571234567890123456", false},
		{"no country code", "1234567890123456", false},
		{"invalid country code", "1293007620116238529", false},
		{"special chars", "CH93-0076-2011-6238-5295-7", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIBANFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanPaymentMethodPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PMT CARTE only", "PMT CARTE", "PMT CARTE"},
		{"PMT CARTE with merchant", "PMT CARTE Starbucks", "Starbucks"},
		{"PMT TWINT only", "PMT TWINT", "PMT TWINT"},
		{"PMT TWINT with merchant", "PMT TWINT Coffee Shop", "Coffee Shop"},
		{"BCV-NET only", "BCV-NET", "BCV-NET"},
		{"BCV-NET with description", "BCV-NET Transfer", "Transfer"},
		{"VIRT BANC only", "VIRT BANC", "VIRT BANC"},
		{"VIRT BANC with description", "VIRT BANC Payment", "Payment"},
		{"no prefix", "Regular Payment", "Regular Payment"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanPaymentMethodPrefixes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPartyNameFromDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PMT TWINT with merchant", "PMT TWINT Coffee Shop", "Coffee Shop"},
		{"PMT CARTE with merchant", "PMT CARTE Grocery Store", "Grocery Store"},
		{"VIRT BANC with description", "VIRT BANC Transfer", "Transfer"},
		{"BCV-NET with description", "BCV-NET Payment", "Payment"},
		{"no prefix", "Regular Payment", ""},
		{"prefix only", "PMT TWINT", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPartyNameFromDescription(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetTransactionTypeFromDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PMT TWINT", "PMT TWINT Coffee", "TWINT"},
		{"PMT CARTE", "PMT CARTE Store", "CB"},
		{"VIRT BANC", "VIRT BANC Transfer", "Virement"},
		{"BCV-NET", "BCV-NET Payment", "Virement"},
		{"ORDRE LSV +", "ORDRE LSV +", "Virement"},
		{"no match", "Regular Payment", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setTransactionTypeFromDescription(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
