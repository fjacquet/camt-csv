package camtparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewISO20022Parser(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	parser := NewISO20022Parser(logger)

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.GetLogger())
}

func TestISO20022Parser_ParseFile(t *testing.T) {
	tests := []struct {
		name          string
		xmlContent    string
		expectError   bool
		expectCount   int
		validateFirst func(t *testing.T, tx models.Transaction)
	}{
		{
			name: "valid single transaction",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Acct><Id><IBAN>CH9300762011623852957</IBAN></Id></Acct>
			<Ntry>
				<Amt Ccy="CHF">100.50</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<Sts>BOOK</Sts>
				<BookgDt><Dt>2023-05-15</Dt></BookgDt>
				<ValDt><Dt>2023-05-16</Dt></ValDt>
				<AcctSvcrRef>SVCR123</AcctSvcrRef>
				<NtryDtls>
					<TxDtls>
						<Refs><EndToEndId>E2E123</EndToEndId></Refs>
						<RmtInf><Ustrd>Payment for services</Ustrd></RmtInf>
						<RltdPties><Cdtr><Nm>Test Creditor</Nm></Cdtr></RltdPties>
					</TxDtls>
				</NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`,
			expectError: false,
			expectCount: 1,
			validateFirst: func(t *testing.T, tx models.Transaction) {
				assert.Equal(t, "CH9300762011623852957", tx.IBAN)
				expected, _ := decimal.NewFromString("100.50")
				assert.True(t, tx.Amount.Equal(expected))
				assert.Equal(t, "CHF", tx.Currency)
				assert.Equal(t, models.TransactionTypeCredit, tx.CreditDebit)
				assert.Equal(t, "BOOK", tx.Status)
			},
		},
		{
			name: "multiple statements",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Acct><Id><IBAN>CH1111111111111111111</IBAN></Id></Acct>
			<Ntry>
				<Amt Ccy="CHF">50.00</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
				<ValDt><Dt>2023-01-01</Dt></ValDt>
			</Ntry>
		</Stmt>
		<Stmt>
			<Acct><Id><IBAN>CH2222222222222222222</IBAN></Id></Acct>
			<Ntry>
				<Amt Ccy="EUR">75.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<BookgDt><Dt>2023-01-02</Dt></BookgDt>
				<ValDt><Dt>2023-01-02</Dt></ValDt>
			</Ntry>
			<Ntry>
				<Amt Ccy="EUR">25.00</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-03</Dt></BookgDt>
				<ValDt><Dt>2023-01-03</Dt></ValDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`,
			expectError: false,
			expectCount: 3,
			validateFirst: func(t *testing.T, tx models.Transaction) {
				assert.Equal(t, "CH1111111111111111111", tx.IBAN)
				expected, _ := decimal.NewFromString("50.00")
				assert.True(t, tx.Amount.Equal(expected))
			},
		},
		{
			name: "debit transaction",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">200.00</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-06-01</Dt></BookgDt>
				<ValDt><Dt>2023-06-01</Dt></ValDt>
				<NtryDtls>
					<TxDtls>
						<RltdPties><Dbtr><Nm>Test Debtor</Nm></Dbtr></RltdPties>
					</TxDtls>
				</NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`,
			expectError: false,
			expectCount: 1,
			validateFirst: func(t *testing.T, tx models.Transaction) {
				assert.Equal(t, models.TransactionTypeDebit, tx.CreditDebit)
				assert.True(t, tx.IsDebit())
			},
		},
		{
			name:        "non-existent file",
			xmlContent:  "", // will not be written
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "invalid XML",
			xmlContent:  "<invalid>not a camt document",
			expectError: true,
			expectCount: 0,
		},
		{
			name: "empty statements",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt></Stmt>
	</BkToCstmrStmt>
</Document>`,
			expectError: false,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogrusAdapter("debug", "text")
			parser := NewISO20022Parser(logger)

			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.xml")

			// Handle non-existent file case
			if tt.name == "non-existent file" {
				testFile = filepath.Join(tempDir, "nonexistent.xml")
			} else {
				err := os.WriteFile(testFile, []byte(tt.xmlContent), 0600)
				require.NoError(t, err)
			}

			transactions, err := parser.ParseFile(testFile)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, transactions, tt.expectCount)

			if tt.validateFirst != nil && len(transactions) > 0 {
				tt.validateFirst(t, transactions[0])
			}
		})
	}
}

func TestISO20022Parser_EntryToTransaction_EdgeCases(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	parser := NewISO20022Parser(logger)

	t.Run("invalid amount falls back to zero", func(t *testing.T) {
		entry := &models.Entry{
			Amt: models.Amount{
				Value: "not-a-number",
				Ccy:   "CHF",
			},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
			ValDt:   models.EntryDate{Dt: "2023-01-01"},
		}

		tx := parser.entryToTransaction(entry)
		assert.True(t, tx.Amount.IsZero())
	})

	t.Run("empty amount string falls back to zero", func(t *testing.T) {
		entry := &models.Entry{
			Amt: models.Amount{
				Value: "",
				Ccy:   "EUR",
			},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
			ValDt:   models.EntryDate{Dt: "2023-01-01"},
		}

		tx := parser.entryToTransaction(entry)
		assert.True(t, tx.Amount.IsZero())
		// Note: Currency is NOT preserved in fallback transaction (when amount is required but missing)
	})

	t.Run("negative amount parsed correctly", func(t *testing.T) {
		entry := &models.Entry{
			Amt: models.Amount{
				Value: "-500.75",
				Ccy:   "CHF",
			},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
			ValDt:   models.EntryDate{Dt: "2023-01-01"},
		}

		tx := parser.entryToTransaction(entry)
		expected, _ := decimal.NewFromString("-500.75")
		assert.True(t, tx.Amount.Equal(expected))
	})

	t.Run("very large amount", func(t *testing.T) {
		entry := &models.Entry{
			Amt: models.Amount{
				Value: "99999999999.99",
				Ccy:   "CHF",
			},
			BookgDt: models.EntryDate{Dt: "2023-01-01"},
			ValDt:   models.EntryDate{Dt: "2023-01-01"},
		}

		tx := parser.entryToTransaction(entry)
		expected, _ := decimal.NewFromString("99999999999.99")
		assert.True(t, tx.Amount.Equal(expected))
	})
}

func TestISO20022Parser_DELLSalarySpecialCase(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	parser := NewISO20022Parser(logger)

	tests := []struct {
		name           string
		xmlContent     string
		expectCategory string
	}{
		{
			name: "DELL SA salary payment",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">5000.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<BookgDt><Dt>2023-01-25</Dt></BookgDt>
				<ValDt><Dt>2023-01-25</Dt></ValDt>
				<NtryDtls>
					<TxDtls>
						<RmtInf><Ustrd>VIRT BANC DELL SA Monthly Salary</Ustrd></RmtInf>
					</TxDtls>
				</NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`,
			expectCategory: models.CategorySalary,
		},
		{
			name: "non-DELL payment stays uncategorized",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">100.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<BookgDt><Dt>2023-01-25</Dt></BookgDt>
				<ValDt><Dt>2023-01-25</Dt></ValDt>
				<NtryDtls>
					<TxDtls>
						<RmtInf><Ustrd>Regular payment</Ustrd></RmtInf>
					</TxDtls>
				</NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`,
			expectCategory: models.CategoryUncategorized,
		},
		{
			name: "partial VIRT BANC without DELL SA",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">1000.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<BookgDt><Dt>2023-01-25</Dt></BookgDt>
				<ValDt><Dt>2023-01-25</Dt></ValDt>
				<NtryDtls>
					<TxDtls>
						<RmtInf><Ustrd>VIRT BANC Other Company</Ustrd></RmtInf>
					</TxDtls>
				</NtryDtls>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`,
			expectCategory: models.CategoryUncategorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.xml")
			err := os.WriteFile(testFile, []byte(tt.xmlContent), 0600)
			require.NoError(t, err)

			transactions, err := parser.ParseFile(testFile)
			require.NoError(t, err)
			require.Len(t, transactions, 1)

			assert.Equal(t, tt.expectCategory, transactions[0].Category)
		})
	}
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

func TestISO20022Parser_ConvertToCSV(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	parser := NewISO20022Parser(logger)

	t.Run("successful conversion", func(t *testing.T) {
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "input.xml")
		outputFile := filepath.Join(tempDir, "output", "result.csv")

		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">123.45</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-07-01</Dt></BookgDt>
				<ValDt><Dt>2023-07-01</Dt></ValDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`
		err := os.WriteFile(inputFile, []byte(xmlContent), 0600)
		require.NoError(t, err)

		err = parser.ConvertToCSV(inputFile, outputFile)
		assert.NoError(t, err)

		// Verify output file exists
		_, err = os.Stat(outputFile)
		assert.NoError(t, err)

		// Verify content
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "123.45")
		assert.Contains(t, string(content), "DBIT")
	})

	t.Run("invalid input file", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "output.csv")

		err := parser.ConvertToCSV("/nonexistent/file.xml", outputFile)
		assert.Error(t, err)
	})

	t.Run("creates output directory", func(t *testing.T) {
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "input.xml")
		outputFile := filepath.Join(tempDir, "nested", "deep", "output.csv")

		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt><Ntry>
			<Amt Ccy="CHF">1.00</Amt>
			<CdtDbtInd>CRDT</CdtDbtInd>
			<BookgDt><Dt>2023-01-01</Dt></BookgDt>
			<ValDt><Dt>2023-01-01</Dt></ValDt>
		</Ntry></Stmt>
	</BkToCstmrStmt>
</Document>`
		err := os.WriteFile(inputFile, []byte(xmlContent), 0600)
		require.NoError(t, err)

		err = parser.ConvertToCSV(inputFile, outputFile)
		assert.NoError(t, err)

		// Verify nested directories were created
		_, err = os.Stat(filepath.Dir(outputFile))
		assert.NoError(t, err)
	})
}

func TestISO20022Parser_CreateEmptyCSVFile(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	parser := NewISO20022Parser(logger)

	t.Run("creates file with headers only", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "empty.csv")

		err := parser.CreateEmptyCSVFile(outputFile)
		assert.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(outputFile)
		assert.NoError(t, err)

		// Verify content has headers
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "BookkeepingNumber")
		assert.Contains(t, string(content), "Amount")
		assert.Contains(t, string(content), "Category")
	})

	t.Run("handles invalid path", func(t *testing.T) {
		err := parser.CreateEmptyCSVFile("/nonexistent/dir/file.csv")
		assert.Error(t, err)
	})
}

func TestISO20022Parser_CategorizeTransactions(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	parser := NewISO20022Parser(logger)

	t.Run("skips already categorized transactions", func(t *testing.T) {
		transactions := []models.Transaction{
			{
				Category:    "Existing Category",
				Description: "Test transaction",
			},
		}

		result := parser.categorizeTransactions(transactions)
		assert.Equal(t, "Existing Category", result[0].Category)
	})

	t.Run("sets uncategorized for empty category", func(t *testing.T) {
		transactions := []models.Transaction{
			{
				Category:    "",
				Description: "Test transaction",
			},
		}

		result := parser.categorizeTransactions(transactions)
		assert.Equal(t, models.CategoryUncategorized, result[0].Category)
	})

	t.Run("handles mixed transactions", func(t *testing.T) {
		transactions := []models.Transaction{
			{Category: "Food", Description: "Grocery shopping"},
			{Category: "", Description: "Unknown payment"},
			{Category: "Transport", Description: "Bus ticket"},
		}

		result := parser.categorizeTransactions(transactions)
		assert.Equal(t, "Food", result[0].Category)
		assert.Equal(t, models.CategoryUncategorized, result[1].Category)
		assert.Equal(t, "Transport", result[2].Category)
	})
}

func TestISO20022Parser_ExtractTransactions(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	parser := NewISO20022Parser(logger)

	t.Run("handles empty document", func(t *testing.T) {
		doc := models.ISO20022Document{}
		transactions, err := parser.extractTransactions(doc)
		assert.NoError(t, err)
		assert.Empty(t, transactions)
	})

	t.Run("inherits IBAN from statement via XML parsing", func(t *testing.T) {
		// Test IBAN inheritance using XML parsing (since BkToCstmrStmt uses inline struct)
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Acct><Id><IBAN>CH1234567890123456789</IBAN></Id></Acct>
			<Ntry>
				<Amt Ccy="CHF">100.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
				<ValDt><Dt>2023-01-01</Dt></ValDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.xml")
		err := os.WriteFile(testFile, []byte(xmlContent), 0600)
		require.NoError(t, err)

		transactions, err := parser.ParseFile(testFile)
		assert.NoError(t, err)
		require.Len(t, transactions, 1)
		assert.Equal(t, "CH1234567890123456789", transactions[0].IBAN)
	})

	t.Run("handles multiple entries across statements", func(t *testing.T) {
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
	<BkToCstmrStmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">10.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<BookgDt><Dt>2023-01-01</Dt></BookgDt>
				<ValDt><Dt>2023-01-01</Dt></ValDt>
			</Ntry>
			<Ntry>
				<Amt Ccy="CHF">20.00</Amt>
				<CdtDbtInd>DBIT</CdtDbtInd>
				<BookgDt><Dt>2023-01-02</Dt></BookgDt>
				<ValDt><Dt>2023-01-02</Dt></ValDt>
			</Ntry>
		</Stmt>
		<Stmt>
			<Ntry>
				<Amt Ccy="CHF">30.00</Amt>
				<CdtDbtInd>CRDT</CdtDbtInd>
				<BookgDt><Dt>2023-01-03</Dt></BookgDt>
				<ValDt><Dt>2023-01-03</Dt></ValDt>
			</Ntry>
		</Stmt>
	</BkToCstmrStmt>
</Document>`
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.xml")
		err := os.WriteFile(testFile, []byte(xmlContent), 0600)
		require.NoError(t, err)

		transactions, err := parser.ParseFile(testFile)
		assert.NoError(t, err)
		assert.Len(t, transactions, 3)

		// Verify amounts are parsed correctly using decimal comparison
		expected1, _ := decimal.NewFromString("10.00")
		expected2, _ := decimal.NewFromString("20.00")
		expected3, _ := decimal.NewFromString("30.00")
		assert.True(t, transactions[0].Amount.Equal(expected1))
		assert.True(t, transactions[1].Amount.Equal(expected2))
		assert.True(t, transactions[2].Amount.Equal(expected3))
	})
}
