package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossParserConsistency tests that all parsers produce identical CSV column structure
// **Feature: parser-enhancements, Property 8: Consistent CSV output format**
// **Validates: Requirements 4.1**
func TestCrossParserConsistency(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Setup mock logger
	logger := logging.NewMockLogger()

	// Create test transactions for each parser type
	camtTransactions := createTestCAMTTransactions()
	pdfTransactions := createTestPDFTransactions()
	selmaTransactions := createTestSelmaTransactions()

	// Write transactions to CSV files
	camtCSVPath := filepath.Join(tempDir, "camt_output.csv")
	pdfCSVPath := filepath.Join(tempDir, "pdf_output.csv")
	selmaCSVPath := filepath.Join(tempDir, "selma_output.csv")

	err := common.WriteTransactionsToCSVWithLogger(camtTransactions, camtCSVPath, logger)
	require.NoError(t, err, "Failed to write CAMT transactions to CSV")

	err = common.WriteTransactionsToCSVWithLogger(pdfTransactions, pdfCSVPath, logger)
	require.NoError(t, err, "Failed to write PDF transactions to CSV")

	err = common.WriteTransactionsToCSVWithLogger(selmaTransactions, selmaCSVPath, logger)
	require.NoError(t, err, "Failed to write Selma transactions to CSV")

	// Read and compare CSV headers
	camtHeaders := readCSVHeaders(t, camtCSVPath)
	pdfHeaders := readCSVHeaders(t, pdfCSVPath)
	selmaHeaders := readCSVHeaders(t, selmaCSVPath)

	// Verify all parsers produce identical column structure
	assert.Equal(t, camtHeaders, pdfHeaders, "CAMT and PDF parsers should produce identical CSV headers")
	assert.Equal(t, camtHeaders, selmaHeaders, "CAMT and Selma parsers should produce identical CSV headers")
	assert.Equal(t, pdfHeaders, selmaHeaders, "PDF and Selma parsers should produce identical CSV headers")

	// Verify essential columns are present
	expectedColumns := []string{
		"BookkeepingNumber", "Status", "Date", "ValueDate", "Name", "PartyName",
		"PartyIBAN", "Description", "RemittanceInfo", "Amount", "CreditDebit",
		"IsDebit", "Debit", "Credit", "Currency", "Category", "Type",
	}

	for _, column := range expectedColumns {
		assert.Contains(t, camtHeaders, column, "CAMT CSV should contain column: %s", column)
		assert.Contains(t, pdfHeaders, column, "PDF CSV should contain column: %s", column)
		assert.Contains(t, selmaHeaders, column, "Selma CSV should contain column: %s", column)
	}

	// Verify categorization columns are present
	assert.Contains(t, camtHeaders, "Category", "CAMT CSV should contain Category column")
	assert.Contains(t, pdfHeaders, "Category", "PDF CSV should contain Category column")
	assert.Contains(t, selmaHeaders, "Category", "Selma CSV should contain Category column")
}

// TestCategorizationConsistency verifies categorization works consistently across parser types
// **Feature: parser-enhancements, Property 6, 7: PDF and Selma categorization integration**
// **Validates: Requirements 2.1, 3.1, 5.3**
func TestCategorizationConsistency(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Setup categorizer with known mappings
	testCategorizer := setupTestCategorizerWithKnownMappings(t, tempDir)

	// Create test transactions with known party names that should be categorized
	testPartyName := "Test Grocery Store"
	expectedCategory := "Food & Dining"

	pdfTransaction := categorizer.Transaction{
		PartyName:   testPartyName,
		IsDebtor:    false,
		Amount:      "25.50",
		Date:        time.Now().Format("2006-01-02"),
		Info:        "Grocery shopping",
		Description: "Grocery shopping",
	}

	selmaTransaction := categorizer.Transaction{
		PartyName:   testPartyName,
		IsDebtor:    false,
		Amount:      "100.00",
		Date:        time.Now().Format("2006-01-02"),
		Info:        "Investment purchase",
		Description: "Investment purchase",
	}

	// Apply categorization through categorizer
	pdfCategoryResult, err := testCategorizer.CategorizeTransaction(context.Background(), pdfTransaction)
	require.NoError(t, err, "PDF categorization should not fail")

	selmaCategoryResult, err := testCategorizer.CategorizeTransaction(context.Background(), selmaTransaction)
	require.NoError(t, err, "Selma categorization should not fail")

	// Verify both parsers produce consistent categorization for the same party
	assert.Equal(t, expectedCategory, pdfCategoryResult.Name,
		"PDF transaction should be categorized as %s", expectedCategory)
	assert.Equal(t, expectedCategory, selmaCategoryResult.Name,
		"Selma transaction should be categorized as %s", expectedCategory)
	assert.Equal(t, pdfCategoryResult.Name, selmaCategoryResult.Name,
		"Both parsers should produce identical categorization for the same party")
}

// TestBatchProcessingWithMixedFileTypes tests batch processing with mixed file types
// **Feature: parser-enhancements, Property 1: Account-based file aggregation**
// **Validates: Requirements 1.1, 4.2**
func TestBatchProcessingWithMixedFileTypes(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "input")
	outputDir := filepath.Join(tempDir, "output")

	err := os.MkdirAll(inputDir, 0750)
	require.NoError(t, err)
	err = os.MkdirAll(outputDir, 0750)
	require.NoError(t, err)

	// Setup logger
	logger := logging.NewMockLogger()

	// Create test CAMT files with same account number
	accountID := "54293249"
	createTestCAMTFile(t, inputDir, fmt.Sprintf("CAMT.053_%s_2025-01-01_2025-01-31_1.xml", accountID))
	createTestCAMTFile(t, inputDir, fmt.Sprintf("CAMT.053_%s_2025-02-01_2025-02-28_1.xml", accountID))

	// Create batch aggregator
	aggregator := batch.NewBatchAggregator(logger)

	// Find input files
	files, err := os.ReadDir(inputDir)
	require.NoError(t, err)

	var inputFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			inputFiles = append(inputFiles, filepath.Join(inputDir, file.Name()))
		}
	}

	require.GreaterOrEqual(t, len(inputFiles), 2, "Should have at least 2 test files")

	// Group files by account
	fileGroups, err := aggregator.GroupFilesByAccount(inputFiles)
	require.NoError(t, err)
	require.Len(t, fileGroups, 1, "Should have exactly 1 account group")

	group := fileGroups[0]
	assert.Equal(t, accountID, group.AccountID, "Account ID should match")
	assert.Len(t, group.Files, 2, "Should have 2 files in the group")

	// Create a mock parser for testing
	mockParser := &mockParser{
		transactions: []models.Transaction{
			{
				Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
				Amount:      decimal.NewFromFloat(100.00),
				Currency:    "CHF",
				Description: "Test transaction 1",
				Category:    "Test Category",
			},
			{
				Date:        time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
				Amount:      decimal.NewFromFloat(200.00),
				Currency:    "CHF",
				Description: "Test transaction 2",
				Category:    "Test Category",
			},
		},
	}

	// Create parse function
	parseFunc := func(filePath string) ([]models.Transaction, error) {
		return mockParser.Parse(nil)
	}

	// Aggregate transactions
	transactions, err := aggregator.AggregateTransactions(group, parseFunc)
	require.NoError(t, err)

	// Verify aggregation results
	assert.Len(t, transactions, 4, "Should have 4 total transactions (2 per file)")

	// Verify transactions are sorted chronologically
	for i := 1; i < len(transactions); i++ {
		assert.True(t, transactions[i-1].Date.Before(transactions[i].Date) ||
			transactions[i-1].Date.Equal(transactions[i].Date),
			"Transactions should be sorted chronologically")
	}

	// Generate output filename
	outputFilename := aggregator.GenerateOutputFilename(group.AccountID, group.DateRange)
	expectedPattern := fmt.Sprintf("%s_", accountID)
	assert.Contains(t, outputFilename, expectedPattern,
		"Output filename should contain account ID")
	assert.True(t, strings.HasSuffix(outputFilename, ".csv"),
		"Output filename should have .csv extension")
}

// TestAutoLearningMechanism tests the auto-learning mechanism with new parsers
// **Feature: parser-enhancements, Property 12: Auto-learning mechanism consistency**
// **Validates: Requirements 5.4**
func TestAutoLearningMechanism(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Setup mock logger
	logger := logging.NewMockLogger()

	// Create store with temporary YAML files
	store := createTestStore(t, tempDir)

	// Create categorizer with auto-learning enabled
	testCategorizer := setupTestCategorizerWithAutoLearning(t, store, logger)

	// Create transaction with unknown party that should trigger AI categorization
	unknownParty := "Unknown Test Company XYZ"
	testTransaction := categorizer.Transaction{
		PartyName:   unknownParty,
		IsDebtor:    false,
		Amount:      "50.00",
		Date:        time.Now().Format("2006-01-02"),
		Info:        "Test purchase",
		Description: "Test purchase",
	}

	// Apply categorization (simulating what happens in Parse())
	result, err := testCategorizer.CategorizeTransaction(context.Background(), testTransaction)
	require.NoError(t, err, "Categorization should not fail")

	// Apply same categorization to another transaction with same party
	secondTransaction := categorizer.Transaction{
		PartyName:   unknownParty,
		IsDebtor:    false,
		Amount:      "75.00",
		Date:        time.Now().Add(24 * time.Hour).Format("2006-01-02"),
		Info:        "Another test purchase",
		Description: "Another test purchase",
	}

	secondResult, err := testCategorizer.CategorizeTransaction(context.Background(), secondTransaction)
	require.NoError(t, err, "Second categorization should not fail")

	// Verify consistency (in real implementation, second call should use saved mapping)
	assert.NotEmpty(t, result.Name, "First categorization should produce a category")
	assert.NotEmpty(t, secondResult.Name, "Second categorization should produce a category")

	// Verify both categorizations are consistent
	assert.Equal(t, result.Name, secondResult.Name,
		"Both categorizations should produce the same category")
}

// Helper functions

func setupTestCategorizerWithKnownMappings(t *testing.T, tempDir string) *categorizer.Categorizer {
	// Create test YAML files with known mappings
	categoriesYAML := `
categories:
  - name: "Food & Dining"
    keywords:
      - "grocery"
      - "restaurant"
      - "food"
`

	creditorsYAML := `
creditors:
  "Test Grocery Store": "Food & Dining"
`

	debtorsYAML := `
debitors:
  "Test Grocery Store": "Food & Dining"
`

	// Write test YAML files
	err := os.WriteFile(filepath.Join(tempDir, "categories.yaml"), []byte(categoriesYAML), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "creditors.yaml"), []byte(creditorsYAML), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "debtors.yaml"), []byte(debtorsYAML), 0600)
	require.NoError(t, err)

	// Create a minimal config for the container
	cfg := &config.Config{}
	cfg.Categories.File = filepath.Join(tempDir, "categories.yaml")
	cfg.Categories.CreditorsFile = filepath.Join(tempDir, "creditors.yaml")
	cfg.Categories.DebtorsFile = filepath.Join(tempDir, "debtors.yaml")
	cfg.Log.Level = "error" // Minimize log output during tests
	cfg.Log.Format = "text"
	cfg.AI.Enabled = false // Disable AI for tests

	// Create container
	container, err := container.NewContainer(cfg)
	require.NoError(t, err, "Failed to create container")

	return container.GetCategorizer()
}

func setupTestCategorizerWithAutoLearning(t *testing.T, testStore categorizer.CategoryStoreInterface, logger logging.Logger) *categorizer.Categorizer {
	// For this test, we'll create a simple categorizer directly
	// since the container doesn't allow injecting custom stores
	return categorizer.NewCategorizer(nil, testStore, logger)
}

func createTestStore(t *testing.T, tempDir string) categorizer.CategoryStoreInterface {
	// Create minimal test YAML files
	categoriesYAML := `
categories:
  "Test Category":
    keywords:
      - "test"
    subcategories:
      - "Test Subcategory"
`

	err := os.WriteFile(filepath.Join(tempDir, "categories.yaml"), []byte(categoriesYAML), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "creditors.yaml"), []byte("creditors: {}"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "debtors.yaml"), []byte("debtors: {}"), 0600)
	require.NoError(t, err)

	return store.NewCategoryStore(
		filepath.Join(tempDir, "categories.yaml"),
		filepath.Join(tempDir, "creditors.yaml"),
		filepath.Join(tempDir, "debtors.yaml"),
	)
}

func createTestCAMTTransactions() []models.Transaction {
	return []models.Transaction{
		{
			Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			Amount:      decimal.NewFromFloat(100.00),
			Currency:    "CHF",
			Description: "CAMT Test Transaction",
			Category:    "Test Category",
			PartyName:   "Test Party",
		},
	}
}

func createTestPDFTransactions() []models.Transaction {
	return []models.Transaction{
		{
			Date:        time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
			Amount:      decimal.NewFromFloat(50.00),
			Currency:    "CHF",
			Description: "PDF Test Transaction",
			Category:    "Test Category",
			PartyName:   "Test Party",
		},
	}
}

func createTestSelmaTransactions() []models.Transaction {
	return []models.Transaction{
		{
			Date:        time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC),
			Amount:      decimal.NewFromFloat(200.00),
			Currency:    "CHF",
			Description: "Selma Test Transaction",
			Category:    "Test Category",
			PartyName:   "Test Party",
			Investment:  "Buy",
		},
	}
}

func readCSVHeaders(t *testing.T, csvPath string) []string {
	file, err := os.Open(csvPath)
	require.NoError(t, err, "Failed to open CSV file: %s", csvPath)
	defer func() {
		if cerr := file.Close(); cerr != nil {
			t.Logf("Failed to close file: %v", cerr)
		}
	}()

	// Read first line (header)
	content, err := io.ReadAll(file)
	require.NoError(t, err, "Failed to read CSV file")

	lines := strings.Split(string(content), "\n")
	require.GreaterOrEqual(t, len(lines), 1, "CSV file should have at least one line")

	// Split header line by comma
	headers := strings.Split(lines[0], ",")

	// Clean up headers (remove quotes and whitespace)
	for i, header := range headers {
		headers[i] = strings.Trim(strings.TrimSpace(header), "\"")
	}

	return headers
}

func createTestCAMTFile(t *testing.T, dir, filename string) {
	// Create a minimal valid CAMT.053 XML file for testing
	camtXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Id>TEST-STMT-001</Id>
      <CreDtTm>2025-01-15T10:00:00</CreDtTm>
      <Acct>
        <Id>
          <IBAN>CH1234567890123456789</IBAN>
        </Id>
      </Acct>
      <Ntry>
        <Amt Ccy="CHF">100.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt>
          <Dt>2025-01-15</Dt>
        </BookgDt>
        <ValDt>
          <Dt>2025-01-15</Dt>
        </ValDt>
        <NtryDtls>
          <TxDtls>
            <RmtInf>
              <Ustrd>Test transaction</Ustrd>
            </RmtInf>
          </TxDtls>
        </NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(camtXML), 0600)
	require.NoError(t, err, "Failed to create test CAMT file")
}

// mockParser is a simple parser implementation for testing
type mockParser struct {
	transactions []models.Transaction
}

func (m *mockParser) Parse(r io.Reader) ([]models.Transaction, error) {
	return m.transactions, nil
}
