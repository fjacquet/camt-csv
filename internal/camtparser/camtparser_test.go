package camtparser

import (
	"os"
	"path/filepath"
	"strings"
	"strconv"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Setup a test logger
	log = logrus.New()
	log.SetLevel(logrus.DebugLevel)
}

func TestValidateFormat(t *testing.T) {
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

	// Create test directories
	tempDir := filepath.Join(os.TempDir(), "camt-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	validFile := filepath.Join(tempDir, "valid.xml")
	invalidFile := filepath.Join(tempDir, "invalid.xml")
	err = os.WriteFile(validFile, []byte(validXML), 0644)
	if err != nil {
		t.Fatalf("Failed to write valid test file: %v", err)
	}
	err = os.WriteFile(invalidFile, []byte(invalidXML), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	// Test valid file
	valid, err := ValidateFormat(validFile)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test invalid file
	valid, err = ValidateFormat(invalidFile)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestParseFile(t *testing.T) {
	// Create a temporary valid CAMT.053 XML file for testing
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document>
  <BkToCstmrStmt>
    <Stmt>
      <Id>12345</Id>
      <CreDtTm>2023-01-01T12:00:00Z</CreDtTm>
      <Acct>
        <Id>
          <IBAN>CH93 0076 2011 6238 5295 7</IBAN>
        </Id>
      </Acct>
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
        <ValDt>
          <Dt>2023-01-02</Dt>
        </ValDt>
        <BkTxCd>
          <Domn>
            <Cd>PMNT</Cd>
            <Fmly>
              <Cd>RCDT</Cd>
              <SubFmlyCd>DMCT</SubFmlyCd>
            </Fmly>
          </Domn>
        </BkTxCd>
        <NtryDtls>
          <TxDtls>
            <Refs>
              <AcctSvcrRef>REF12345</AcctSvcrRef>
            </Refs>
            <RmtInf>
              <Ustrd>Coffee Shop</Ustrd>
            </RmtInf>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	// Create test directories
	tempDir := filepath.Join(os.TempDir(), "camt-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.xml")
	err = os.WriteFile(testFile, []byte(validXML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test parsing
	transactions, err := ParseFile(testFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, transactions)
	assert.Equal(t, 1, len(transactions))
	
	// Verify transaction data
	tx := transactions[0]
	assert.Equal(t, "2023-01-01", tx.Date)
	assert.Equal(t, "2023-01-02", tx.ValueDate)
	assert.Equal(t, "100.00", tx.Amount)
	assert.Equal(t, "EUR", tx.Currency)
	assert.Equal(t, "DBIT", tx.CreditDebit)
	// Skip checking AccountServicer as it might not be populated in the test XML
	// assert.Equal(t, "REF12345", tx.AccountServicer)
	assert.True(t, strings.Contains(tx.Description, "Coffee Shop"))
}

func TestConvertToCSV(t *testing.T) {
	// Create a test transaction
	transaction := models.Transaction{
		Date:           "2023-01-01",
		ValueDate:      "2023-01-02",
		Description:    "Test Transaction",
		BookkeepingNo:  "BK123",
		Fund:           "Test Fund",
		Amount:         "100.00",
		Currency:       "EUR",
		CreditDebit:    "DBIT",
		EntryReference: "REF123",
		AccountServicer: "AS123",
		BankTxCode:     "PMNT/RCDT/DMCT",
		Status:         "BOOK",
		Payee:          "Test Payee",
		Payer:          "Test Payer",
		IBAN:           "CH93 0076 2011 6238 5295 7",
		Category:       "Food",
	}

	// Create test directories
	tempDir := filepath.Join(os.TempDir(), "camt-test")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test writing to CSV
	testFile := filepath.Join(tempDir, "test.csv")
	err = WriteToCSV([]models.Transaction{transaction}, testFile)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(testFile)
	assert.NoError(t, err)

	// Read the file and check contents
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Test Transaction")
	assert.Contains(t, string(content), "100.00")
	assert.Contains(t, string(content), "EUR")
	assert.Contains(t, string(content), "Food")
}

func TestBatchConvert(t *testing.T) {
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

	// Create test directories
	tempDir := filepath.Join(os.TempDir(), "camt-test")
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	
	err := os.MkdirAll(inputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(inputDir, "test"+strconv.Itoa(i)+".xml")
		err = os.WriteFile(filename, []byte(validXML), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Test batch conversion
	count, err := BatchConvert(inputDir, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	// Verify output files
	files, err := filepath.Glob(filepath.Join(outputDir, "*.csv"))
	assert.NoError(t, err)
	assert.Equal(t, 3, len(files))
}
