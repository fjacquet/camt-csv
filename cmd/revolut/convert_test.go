package revolut_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"github.com/stretchr/testify/assert"
)

// mockFullParser implements parser.FullParser for testing
type mockFullParser struct {
	validateErr    error
	validateResult bool
	convertErr     error
}

func (m *mockFullParser) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return nil, nil
}

func (m *mockFullParser) ValidateFormat(filePath string) (bool, error) {
	return m.validateResult, m.validateErr
}

func (m *mockFullParser) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
	return m.convertErr
}

func (m *mockFullParser) SetLogger(logger logging.Logger) {}

func (m *mockFullParser) SetCategorizer(categorizer models.TransactionCategorizer) {}

func (m *mockFullParser) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
	return 0, nil
}

var _ parser.FullParser = (*mockFullParser)(nil)

func TestRevolutConvert_InvalidFormat(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	mockParser := &mockFullParser{
		validateResult: false,
		validateErr:    nil,
	}

	err := common.ProcessFileWithError(context.Background(), mockParser, "input.csv", "output.csv", true, logger)

	assert.Error(t, err)
	assert.ErrorIs(t, err, common.ErrInvalidFormat)
}

func TestRevolutConvert_ConversionError(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	mockParser := &mockFullParser{
		validateResult: true,
		convertErr:     errors.New("Revolut parsing failed"),
	}

	err := common.ProcessFileWithError(context.Background(), mockParser, "input.csv", "output.csv", true, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error converting to CSV")
}
