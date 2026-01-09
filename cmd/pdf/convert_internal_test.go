package pdf

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockParserForConsolidation implements parser.FullParser for testing consolidation
type mockParserForConsolidation struct {
	validateErr    error
	validateResult bool
	parseErr       error
	transactions   []models.Transaction
	parseCalls     int
	validateCalls  int

	// Function fields for custom behavior
	ParseFunc          func(ctx context.Context, r io.Reader) ([]models.Transaction, error)
	ValidateFormatFunc func(filePath string) (bool, error)
}

func (m *mockParserForConsolidation) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	m.parseCalls++
	if m.ParseFunc != nil {
		return m.ParseFunc(ctx, r)
	}
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.transactions, nil
}

func (m *mockParserForConsolidation) ValidateFormat(filePath string) (bool, error) {
	m.validateCalls++
	if m.ValidateFormatFunc != nil {
		return m.ValidateFormatFunc(filePath)
	}
	return m.validateResult, m.validateErr
}

func (m *mockParserForConsolidation) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
	return nil
}

func (m *mockParserForConsolidation) SetLogger(logger logging.Logger) {}

func (m *mockParserForConsolidation) SetCategorizer(categorizer models.TransactionCategorizer) {}

func (m *mockParserForConsolidation) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
	return 0, nil
}

var _ parser.FullParser = (*mockParserForConsolidation)(nil)

func TestSortTransactionsChronologically(t *testing.T) {
	tests := []struct {
		name     string
		input    []models.Transaction
		expected []models.Transaction
	}{
		{
			name: "sort by date - different dates",
			input: []models.Transaction{
				{Date: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(200)},
				{Date: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(300)},
			},
			expected: []models.Transaction{
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(200)},
				{Date: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(300)},
				{Date: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
			},
		},
		{
			name: "sort by value date - same dates",
			input: []models.Transaction{
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(200)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(300)},
			},
			expected: []models.Transaction{
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(200)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(300)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
			},
		},
		{
			name: "sort by amount - same date and value date",
			input: []models.Transaction{
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(300)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(200)},
			},
			expected: []models.Transaction{
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(200)},
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(300)},
			},
		},
		{
			name:     "empty slice",
			input:    []models.Transaction{},
			expected: []models.Transaction{},
		},
		{
			name: "single transaction",
			input: []models.Transaction{
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
			},
			expected: []models.Transaction{
				{Date: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Amount: decimal.NewFromInt(100)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortTransactionsChronologically(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestConsolidatePDFDirectory_Success(t *testing.T) {
	// Create temp directory with PDF files
	tempDir := t.TempDir()

	// Create test PDF files
	pdf1 := filepath.Join(tempDir, "statement1.pdf")
	pdf2 := filepath.Join(tempDir, "statement2.pdf")
	pdf3 := filepath.Join(tempDir, "statement3.pdf")

	require.NoError(t, os.WriteFile(pdf1, []byte("pdf content 1"), 0600))
	require.NoError(t, os.WriteFile(pdf2, []byte("pdf content 2"), 0600))
	require.NoError(t, os.WriteFile(pdf3, []byte("pdf content 3"), 0600))

	// Create output file path
	outputFile := filepath.Join(tempDir, "consolidated.csv")

	// Create mock parser with transactions
	mockParser := &mockParserForConsolidation{
		validateResult: true,
		transactions: []models.Transaction{
			{
				Date:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				ValueDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Amount:    decimal.NewFromInt(100),
				Currency:  "CHF",
			},
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")

	// Execute
	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, false, logger)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 3, count, "Should process all 3 PDF files")
	assert.Equal(t, 3, mockParser.parseCalls, "Parser should be called 3 times")

	// Verify output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Output file should exist")
}

func TestConsolidatePDFDirectory_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	mockParser := &mockParserForConsolidation{
		validateResult: true,
	}

	logger := logging.NewLogrusAdapter("info", "text")

	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, false, logger)

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestConsolidatePDFDirectory_MixedValidInvalid(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid PDF files
	validPDF1 := filepath.Join(tempDir, "valid1.pdf")
	validPDF2 := filepath.Join(tempDir, "valid2.pdf")
	require.NoError(t, os.WriteFile(validPDF1, []byte("valid pdf 1"), 0600))
	require.NoError(t, os.WriteFile(validPDF2, []byte("valid pdf 2"), 0600))

	// Create non-PDF file
	txtFile := filepath.Join(tempDir, "readme.txt")
	require.NoError(t, os.WriteFile(txtFile, []byte("not a pdf"), 0600))

	// Create subdirectory (should be skipped)
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0700))

	outputFile := filepath.Join(tempDir, "output.csv")

	mockParser := &mockParserForConsolidation{
		validateResult: true,
		transactions: []models.Transaction{
			{
				Date:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Amount:   decimal.NewFromInt(100),
				Currency: "CHF",
			},
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")

	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, false, logger)

	require.NoError(t, err)
	assert.Equal(t, 2, count, "Should only process 2 valid PDF files")
	assert.Equal(t, 2, mockParser.parseCalls)
}

func TestConsolidatePDFDirectory_ValidationEnabled(t *testing.T) {
	tempDir := t.TempDir()

	// Create PDF files
	validPDF := filepath.Join(tempDir, "valid.pdf")
	invalidPDF := filepath.Join(tempDir, "invalid.pdf")
	require.NoError(t, os.WriteFile(validPDF, []byte("valid"), 0600))
	require.NoError(t, os.WriteFile(invalidPDF, []byte("invalid"), 0600))

	outputFile := filepath.Join(tempDir, "output.csv")

	// Mock parser that validates only the first file
	callCount := 0
	mockParser := &mockParserForConsolidation{
		transactions: []models.Transaction{
			{
				Date:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Amount:   decimal.NewFromInt(100),
				Currency: "CHF",
			},
		},
		ValidateFormatFunc: func(filePath string) (bool, error) {
			callCount++
			if callCount == 1 {
				return true, nil
			}
			return false, nil
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")

	// Execute with validation enabled
	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, true, logger)

	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should only process valid PDF")
}

func TestConsolidatePDFDirectory_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple PDF files
	for i := 1; i <= 5; i++ {
		pdfFile := filepath.Join(tempDir, "statement"+string(rune('0'+i))+".pdf")
		require.NoError(t, os.WriteFile(pdfFile, []byte("content"), 0600))
	}

	outputFile := filepath.Join(tempDir, "output.csv")

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Create a counter to track parse calls
	parseCallCounter := 0
	transactions := []models.Transaction{
		{
			Date:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount:   decimal.NewFromInt(100),
			Currency: "CHF",
		},
	}

	// Mock parser that will trigger cancellation after first parse
	mockParser := &mockParserForConsolidation{
		validateResult: true,
		transactions:   transactions,
		ParseFunc: func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
			parseCallCounter++
			if parseCallCounter == 1 {
				cancel() // Cancel after first file
			}
			return transactions, nil
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")

	count, err := consolidatePDFDirectory(ctx, mockParser, tempDir, outputFile, false, logger)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.LessOrEqual(t, count, 2, "Should process at most 2 files before cancellation")
}

func TestConsolidatePDFDirectory_ParseErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Create PDF files
	pdf1 := filepath.Join(tempDir, "good.pdf")
	pdf2 := filepath.Join(tempDir, "bad.pdf")
	pdf3 := filepath.Join(tempDir, "good2.pdf")

	require.NoError(t, os.WriteFile(pdf1, []byte("good"), 0600))
	require.NoError(t, os.WriteFile(pdf2, []byte("bad"), 0600))
	require.NoError(t, os.WriteFile(pdf3, []byte("good"), 0600))

	outputFile := filepath.Join(tempDir, "output.csv")

	// Create a counter to track parse calls
	parseCallCounter := 0
	transactions := []models.Transaction{
		{
			Date:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount:   decimal.NewFromInt(100),
			Currency: "CHF",
		},
	}

	// Mock parser that fails on second file
	mockParser := &mockParserForConsolidation{
		validateResult: true,
		transactions:   transactions,
		ParseFunc: func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
			parseCallCounter++
			if parseCallCounter == 2 {
				return nil, errors.New("parse error")
			}
			return transactions, nil
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")

	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, false, logger)

	// Should succeed but skip the bad file
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Should process 2 good files, skip 1 bad file")
}

func TestConsolidatePDFDirectory_NoTransactions(t *testing.T) {
	tempDir := t.TempDir()

	pdfFile := filepath.Join(tempDir, "empty.pdf")
	require.NoError(t, os.WriteFile(pdfFile, []byte("empty"), 0600))

	outputFile := filepath.Join(tempDir, "output.csv")

	// Mock parser that returns empty transactions
	mockParser := &mockParserForConsolidation{
		validateResult: true,
		transactions:   []models.Transaction{}, // Empty
	}

	logger := logging.NewLogrusAdapter("info", "text")

	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, false, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transactions extracted")
	assert.Equal(t, 1, count, "File was processed even though no transactions")
}

func TestConsolidatePDFDirectory_CaseInsensitivePDFExtension(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with different case extensions
	pdf1 := filepath.Join(tempDir, "file1.pdf")
	pdf2 := filepath.Join(tempDir, "file2.PDF")
	pdf3 := filepath.Join(tempDir, "file3.Pdf")

	require.NoError(t, os.WriteFile(pdf1, []byte("content"), 0600))
	require.NoError(t, os.WriteFile(pdf2, []byte("content"), 0600))
	require.NoError(t, os.WriteFile(pdf3, []byte("content"), 0600))

	outputFile := filepath.Join(tempDir, "output.csv")

	mockParser := &mockParserForConsolidation{
		validateResult: true,
		transactions: []models.Transaction{
			{
				Date:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Amount:   decimal.NewFromInt(100),
				Currency: "CHF",
			},
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")

	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, false, logger)

	require.NoError(t, err)
	assert.Equal(t, 3, count, "Should process all PDF files regardless of case")
}

func TestConsolidatePDFDirectory_TransactionsSorted(t *testing.T) {
	tempDir := t.TempDir()

	// Create PDF files
	pdf1 := filepath.Join(tempDir, "file1.pdf")
	pdf2 := filepath.Join(tempDir, "file2.pdf")

	require.NoError(t, os.WriteFile(pdf1, []byte("content"), 0600))
	require.NoError(t, os.WriteFile(pdf2, []byte("content"), 0600))

	outputFile := filepath.Join(tempDir, "output.csv")

	// Create a counter to track parse calls
	parseCallCounter := 0

	// Create parser that returns transactions in reverse chronological order
	mockParser := &mockParserForConsolidation{
		validateResult: true,
		ParseFunc: func(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
			parseCallCounter++
			if parseCallCounter == 1 {
				// First file: March transaction
				return []models.Transaction{
					{
						Date:     time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
						Amount:   decimal.NewFromInt(300),
						Currency: "CHF",
					},
				}, nil
			}
			// Second file: January transaction
			return []models.Transaction{
				{
					Date:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					Amount:   decimal.NewFromInt(100),
					Currency: "CHF",
				},
			}, nil
		},
	}

	logger := logging.NewLogrusAdapter("info", "text")

	count, err := consolidatePDFDirectory(context.Background(), mockParser, tempDir, outputFile, false, logger)

	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Read output file and verify sorting
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// The CSV should have the January transaction before March transaction
	csvContent := string(content)
	assert.Contains(t, csvContent, "01.01.2023", "Should contain January date")
	assert.Contains(t, csvContent, "01.03.2023", "Should contain March date")

	// Verify January comes before March in the file
	jan := []byte("01.01.2023")
	mar := []byte("01.03.2023")
	janPos := -1
	marPos := -1

	for i := range content {
		if i+len(jan) <= len(content) && string(content[i:i+len(jan)]) == string(jan) {
			janPos = i
		}
		if i+len(mar) <= len(content) && string(content[i:i+len(mar)]) == string(mar) {
			marPos = i
		}
	}

	if janPos != -1 && marPos != -1 {
		assert.Less(t, janPos, marPos, "January transaction should appear before March transaction")
	}
}
