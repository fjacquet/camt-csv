package camtparser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	transactions, err := adapter.Parse(context.Background(), file)
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
	// CSV delimiter is now a constant (models.DefaultCSVDelimiter)

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
	err = adapter.ConvertToCSV(context.Background(), xmlFile, csvFile)
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

// Test error scenarios in CAMT parser
func TestCAMTParser_ErrorScenarios(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	t.Run("invalid XML format", func(t *testing.T) {
		invalidXML := `<invalid>not a camt document</invalid>`
		file := strings.NewReader(invalidXML)

		transactions, err := adapter.Parse(context.Background(), file)
		assert.Error(t, err)
		assert.Nil(t, transactions)
		assert.Contains(t, err.Error(), "error decoding XML")
	})

	t.Run("empty XML document", func(t *testing.T) {
		emptyXML := ``
		file := strings.NewReader(emptyXML)

		transactions, err := adapter.Parse(context.Background(), file)
		assert.Error(t, err)
		assert.Nil(t, transactions)
	})

	t.Run("malformed XML", func(t *testing.T) {
		malformedXML := `<?xml version="1.0"?><Document><unclosed>`
		file := strings.NewReader(malformedXML)

		transactions, err := adapter.Parse(context.Background(), file)
		assert.Error(t, err)
		assert.Nil(t, transactions)
	})

	t.Run("missing required fields", func(t *testing.T) {
		// XML with missing amount
		xmlWithMissingAmount := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`

		file := strings.NewReader(xmlWithMissingAmount)
		transactions, err := adapter.Parse(context.Background(), file)

		// Should not error but may have zero amount
		assert.NoError(t, err)
		if len(transactions) > 0 {
			assert.True(t, transactions[0].Amount.IsZero())
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		xmlWithInvalidDate := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">100.00</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>invalid-date</Dt></BookgDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`

		file := strings.NewReader(xmlWithInvalidDate)
		transactions, err := adapter.Parse(context.Background(), file)

		// Should handle gracefully - may log warning but not fail
		assert.NoError(t, err)
		if len(transactions) > 0 {
			// Date should be zero time for invalid dates
			assert.True(t, transactions[0].Date.IsZero())
		}
	})

	t.Run("invalid amount format", func(t *testing.T) {
		xmlWithInvalidAmount := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">invalid-amount</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`

		file := strings.NewReader(xmlWithInvalidAmount)
		transactions, err := adapter.Parse(context.Background(), file)

		// Should handle gracefully
		assert.NoError(t, err)
		if len(transactions) > 0 {
			assert.True(t, transactions[0].Amount.IsZero())
		}
	})

	t.Run("missing currency", func(t *testing.T) {
		xmlWithoutCurrency := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt>100.00</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`

		file := strings.NewReader(xmlWithoutCurrency)
		transactions, err := adapter.Parse(context.Background(), file)

		// Should handle gracefully
		assert.NoError(t, err)
		if len(transactions) > 0 {
			// Currency should be empty or default
			assert.True(t, transactions[0].Currency == "" || transactions[0].Currency == "CHF")
		}
	})

	t.Run("very large XML document", func(t *testing.T) {
		// Create a large XML with many entries
		var xmlBuilder strings.Builder
		xmlBuilder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>`)

		// Add 1000 entries
		for i := 0; i < 1000; i++ {
			xmlBuilder.WriteString(fmt.Sprintf(`
			<Ntry>
				<Amt Ccy="CHF">%d.00</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
				<NtryDtls>
					<TxDtls>
						<RmtInf><Ustrd>Transaction %d</Ustrd></RmtInf>
					</TxDtls>
				</NtryDtls>
			</Ntry>`, i+1, i+1))
		}

		xmlBuilder.WriteString(`
		</Stmt>
	</BkToCstmrStmt>
</Document>`)

		file := strings.NewReader(xmlBuilder.String())
		transactions, err := adapter.Parse(context.Background(), file)

		assert.NoError(t, err)
		assert.Equal(t, 1000, len(transactions))
	})
}

// Test file validation scenarios
func TestCAMTParser_FileValidation(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	t.Run("non-existent file", func(t *testing.T) {
		isValid, err := adapter.ValidateFormat("/non/existent/file.xml")
		assert.Error(t, err)
		assert.False(t, isValid)
	})

	t.Run("empty file", func(t *testing.T) {
		tempDir := t.TempDir()
		emptyFile := filepath.Join(tempDir, "empty.xml")
		err := os.WriteFile(emptyFile, []byte(""), 0600)
		require.NoError(t, err)

		isValid, err := adapter.ValidateFormat(emptyFile)
		assert.Error(t, err)
		assert.False(t, isValid)
	})

	t.Run("non-XML file", func(t *testing.T) {
		tempDir := t.TempDir()
		textFile := filepath.Join(tempDir, "text.xml")
		err := os.WriteFile(textFile, []byte("This is not XML"), 0600)
		require.NoError(t, err)

		isValid, err := adapter.ValidateFormat(textFile)
		assert.Error(t, err)
		assert.False(t, isValid)
	})

	t.Run("valid CAMT file", func(t *testing.T) {
		tempDir := t.TempDir()
		validFile := filepath.Join(tempDir, "valid.xml")
		err := os.WriteFile(validFile, []byte(testXMLContent), 0600)
		require.NoError(t, err)

		isValid, err := adapter.ValidateFormat(validFile)
		assert.NoError(t, err)
		assert.True(t, isValid)
	})
}

// Test CSV conversion error scenarios
func TestCAMTParser_CSVConversionErrors(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger)

	t.Run("invalid input file", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "output.csv")

		err := adapter.ConvertToCSV(context.Background(), "/non/existent/input.xml", outputFile)
		assert.Error(t, err)
	})

	t.Run("invalid output directory", func(t *testing.T) {
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "input.xml")
		err := os.WriteFile(inputFile, []byte(testXMLContent), 0600)
		require.NoError(t, err)

		// Try to write to non-existent directory
		err = adapter.ConvertToCSV(context.Background(), inputFile, "/non/existent/dir/output.csv")
		assert.Error(t, err)
	})

	t.Run("permission denied on output", func(t *testing.T) {
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "input.xml")
		err := os.WriteFile(inputFile, []byte(testXMLContent), 0600)
		require.NoError(t, err)

		// Create read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err = os.MkdirAll(readOnlyDir, 0400)
		require.NoError(t, err)

		outputFile := filepath.Join(readOnlyDir, "output.csv")
		err = adapter.ConvertToCSV(context.Background(), inputFile, outputFile)
		assert.Error(t, err)
	})
}
