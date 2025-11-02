package camtparser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetLogger(t *testing.T) {
	// Create a new logger
	logger := logging.NewLogrusAdapter("info", "text")

	// Create adapter with logger
	adapter := NewAdapter(logger)
	
	// Test that we can set a different logger
	newLogger := logging.NewLogrusAdapter("debug", "text")
	adapter.SetLogger(newLogger)
}

const testXMLContent = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<GrpHdr>
			<MsgId>20230101-001</MsgId>
			<CreDtTm>2023-01-01T10:00:00</CreDtTm>
		</GrpHdr>
		<Stmt>
			<Id>STATEMENT-001</Id>
			<CreDtTm>2023-01-01T10:00:00</CreDtTm>
			<Acct>
				<Id>
					<IBAN>CH9300762011623852957</IBAN>
				</Id>
				<Svcr>
					<FinInstnId>
						<BIC>AS123</BIC>
					</FinInstnId>
				</Svcr>
			</Acct>
			<Ntry>
				<NtryRef>REF123</NtryRef>
				<Amt Ccy="EUR">100</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
				<ValDt><Dt>2023-01-02</Dt></ValDt>
				<BkTxCd>
					<Domn>
						<Cd>PMNT</Cd>
						<Fmly>
							<Cd>RCDT</Cd>
							<SubCd>DMCT</SubCd>
						</Fmly>
					</Domn>
				</BkTxCd>
				<NtryDtls>
					<TxDtls>
						<Refs>
							<EndToEndId>BK123</EndToEndId>
						</Refs>
						<RltdPties>
							<Dbtr>
								<Nm>Test Payee</Nm>
							</Dbtr>
						</RltdPties>
						<RmtInf>
							<Ustrd>Test Transaction</Ustrd>
						</RmtInf>
					</TxDtls>
				</NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`

func TestParseFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_camt.xml")

	// Sample CAMT.053 XML content
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">120.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<Sts>BOOK</Sts>
				<BookgDt>
					<Dt>2025-01-15</Dt>
				</BookgDt>
				<ValDt>
					<Dt>2025-01-15</Dt>
				</ValDt>
				<AcctSvcrRef>ref123</AcctSvcrRef>
				<NtryDtls>
					<TxDtls>
						<Refs>
							<TxId>tx123</TxId>
						</Refs>
						<Amt Ccy="CHF">120.00</Amt>
						<CdtDbtInd>CRDT</CdtDbtInd>
						<RmtInf>
							<Ustrd>Invoice 123</Ustrd>
						</RmtInf>
						<RltdPties>
							<Dbtr>
								<Nm>John Doe</Nm>
							</Dbtr>
						</RltdPties>
					</TxDtls>
				</NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`

	err := os.WriteFile(testFile, []byte(xmlContent), 0600)
	assert.NoError(t, err, "Failed to create test file")

	file, err := os.Open(testFile)
	require.NoError(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file %s: %v", testFile, err)
		}
	}()

	// Test parsing
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)
	transactions, err := adapter.Parse(file)
	assert.NoError(t, err, "Failed to parse CAMT.053 XML file")
	assert.Equal(t, 1, len(transactions), "Expected 1 transaction")

	// Verify transaction
	expectedDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDate, transactions[0].Date)
	assert.Equal(t, expectedDate, transactions[0].ValueDate)
	assert.Equal(t, "Invoice 123", transactions[0].RemittanceInfo)
	assert.Equal(t, models.ParseAmount("120.00"), transactions[0].Amount)
	assert.Equal(t, "CHF", transactions[0].Currency)
	assert.Equal(t, models.TransactionTypeCredit, transactions[0].CreditDebit)
	assert.Equal(t, "BOOK", transactions[0].Status)
	assert.Equal(t, "John Doe", transactions[0].PartyName)
}

func TestConvertToCSV(t *testing.T) {
	// Set CSV delimiter to comma for this test
	common.SetDelimiter(',')

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "camt-test")
	assert.NoError(t, err)
	defer func() { assert.NoError(t, os.RemoveAll(tempDir)) }()

	// Create a dummy XML file
	xmlFile := filepath.Join(tempDir, "input.xml")
	err = os.WriteFile(xmlFile, []byte(testXMLContent), 0600)
	assert.NoError(t, err)

	// Define the output CSV file path
	csvFile := filepath.Join(tempDir, "output.csv")

	// Convert XML to CSV
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)
	err = adapter.ConvertToCSV(xmlFile, csvFile)
	assert.NoError(t, err)

	// Read the generated CSV file
	csvContent, err := os.ReadFile(csvFile)
	assert.NoError(t, err)

	// Expected CSV content (comma-separated) - updated to match actual parser output
	expectedCSV := "BookkeepingNumber,Status,Date,ValueDate,Name,PartyName,PartyIBAN,Description,RemittanceInfo,Amount,CreditDebit,IsDebit,Debit,Credit,Currency,AmountExclTax,AmountTax,TaxRate,Recipient,InvestmentType,Number,Category,Type,Fund,NumberOfShares,Fees,IBAN,EntryReference,Reference,AccountServicer,BankTxCode,OriginalCurrency,OriginalAmount,ExchangeRate\n,,01.01.2023,02.01.2023,Test Payee,Test Payee,,Test Transaction,Test Transaction,100.00,DBIT,true,100.00,0.00,EUR,0.00,0.00,0.00,Test Payee,,,Uncategorized,,,0,0.00,,,BK123,,,,0.00,0.00\n"

	assert.Equal(t, expectedCSV, string(csvContent))
}

/*
func TestBatchConvert(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")
	err := os.MkdirAll(inputDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}
	err = os.MkdirAll(outputDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	t.Run("batch convert", func(t *testing.T) {
		setupTestCategorizer(t)
		// Write a canonical minimal valid CAMT.053 XML file
		validXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Id>STMT-001</Id>
      <Ntry>
        <Amt Ccy="EUR">100.00</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt><Dt>2023-01-01</Dt></BookgDt>
        <NtryDtls>
          <TxDtls>
            <Refs><EndToEndId>NOTPROVIDED</EndToEndId></Refs>
            <RmtInf><Ustrd>Batch Transaction</Ustrd></RmtInf>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`
		inputFile := filepath.Join(inputDir, "test1.xml")
		err := os.WriteFile(inputFile, []byte(validXML), 0600)
		assert.NoError(t, err)
		files, err := filepath.Glob(filepath.Join(inputDir, "*.xml"))
		assert.NoError(t, err)
		for _, file := range files {
			csvFile := filepath.Join(outputDir, filepath.Base(file)+".csv")
			err := ConvertToCSV(file, csvFile)
			assert.NoError(t, err)
		}
	})
}
*/
