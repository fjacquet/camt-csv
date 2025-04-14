package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCAMT053(t *testing.T) {
	// Create a temporary valid CAMT.053 XML file for testing
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document>
  <BkToCstmrStmt>
    <Stmt>
      <Id>12345</Id>
      <CreDtTm>2023-01-01T12:00:00Z</CreDtTm>
      <Bal>
        <Tp>
          <CdOrPrtry>
            <Cd>OPBD</Cd>
          </CdOrPrtry>
        </Tp>
        <Amt Ccy="EUR">1000.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
      </Bal>
      <Ntry>
        <Amt Ccy="EUR">100.00</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt>
          <Dt>2023-01-01</Dt>
        </BookgDt>
        <NtryDtls>
          <TxDtls>
            <RmtInf>
              <Ustrd>Coffee Shop</Ustrd>
            </RmtInf>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	// Create a temporary invalid XML file for testing
	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document>
  <SomeOtherTag>
    <NotCAMT053>This is not a CAMT.053 file</NotCAMT053>
  </SomeOtherTag>
</Document>`

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.xml")
	invalidFile := filepath.Join(tempDir, "invalid.xml")

	// Write test files
	if err := os.WriteFile(validFile, []byte(validXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(invalidFile, []byte(invalidXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test valid file
	valid, err := ValidateCAMT053(validFile)
	if err != nil {
		t.Errorf("Error validating valid XML: %v", err)
	}
	if !valid {
		t.Errorf("Expected valid XML to be valid, but got invalid")
	}

	// Test invalid file
	valid, err = ValidateCAMT053(invalidFile)
	if err != nil {
		t.Errorf("Error validating invalid XML: %v", err)
	}
	if valid {
		t.Errorf("Expected invalid XML to be invalid, but got valid")
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2023-01-01", "Jan 01, 2023"},
		{"2023-12-31", "Dec 31, 2023"},
		{"", ""},
		{"invalid-date", "invalid-date"}, // Should return original if parsing fails
	}

	for _, test := range tests {
		result := formatDate(test.input)
		if result != test.expected {
			t.Errorf("formatDate(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestExtractTransactions(t *testing.T) {
	// Create a test CAMT053 structure
	camt053 := &CAMT053{
		BkToCstmrStmt: BkToCstmrStmt{
			Stmt: Stmt{
				Id:      "12345",
				CreDtTm: "2023-01-01T12:00:00Z",
				Ntry: []Ntry{
					{
						Amt: Amt{
							Text: "100.00",
							Ccy:  "EUR",
						},
						CdtDbtInd: "DBIT",
						BookgDt: BookgDt{
							Dt: "2023-01-01",
						},
						NtryDtls: NtryDtls{
							TxDtls: []TxDtls{
								{
									RmtInf: RmtInf{
										Ustrd: []string{"Coffee Shop"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	transactions := extractTransactions(camt053)

	if len(transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(transactions))
	}

	if transactions[0].Amount != "100.00" {
		t.Errorf("Expected Amount '100.00', got '%s'", transactions[0].Amount)
	}

	if transactions[0].Currency != "EUR" {
		t.Errorf("Expected Currency 'EUR', got '%s'", transactions[0].Currency)
	}

	if transactions[0].CreditDebit != "DBIT" {
		t.Errorf("Expected CreditDebit 'DBIT', got '%s'", transactions[0].CreditDebit)
	}

	if transactions[0].Investment != "Coffee Shop" {
		t.Errorf("Expected Investment 'Coffee Shop', got '%s'", transactions[0].Investment)
	}
}

func TestConvertXMLToCSV(t *testing.T) {
	// Create a temporary valid CAMT.053 XML file for testing
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document>
  <BkToCstmrStmt>
    <Stmt>
      <Id>12345</Id>
      <CreDtTm>2023-01-01T12:00:00Z</CreDtTm>
      <Ntry>
        <Amt Ccy="EUR">100.00</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt>
          <Dt>2023-01-01</Dt>
        </BookgDt>
        <NtryDtls>
          <TxDtls>
            <RmtInf>
              <Ustrd>Coffee Shop</Ustrd>
            </RmtInf>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	tempDir := t.TempDir()
	xmlFile := filepath.Join(tempDir, "test.xml")
	csvFile := filepath.Join(tempDir, "test.csv")

	// Write test file
	if err := os.WriteFile(xmlFile, []byte(validXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test conversion
	err := ConvertXMLToCSV(xmlFile, csvFile)
	if err != nil {
		t.Errorf("Error converting XML to CSV: %v", err)
	}

	// Check if CSV file was created
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		t.Errorf("CSV file was not created")
	}

	// Read CSV file content
	content, err := os.ReadFile(csvFile)
	if err != nil {
		t.Errorf("Error reading CSV file: %v", err)
	}

	// Check if CSV content contains expected data
	csvContent := string(content)
	expectedHeaders := "Date,Description,BookkeepingNo,Fund,Amount,Currency,CreditDebit,NumberOfShares,StampDuty,Investment"
	if !contains(csvContent, expectedHeaders) {
		t.Errorf("CSV headers not found in content: %s", csvContent)
	}

	// Check for expected data fields
	if !contains(csvContent, "100.00") || !contains(csvContent, "EUR") || 
	   !contains(csvContent, "DBIT") || !contains(csvContent, "Coffee Shop") {
		t.Errorf("Expected data not found in CSV content: %s", csvContent)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
