package common_test

import (
	"errors"
	"io"
	"testing"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFullParser implements parser.FullParser for testing
type MockFullParser struct {
	mock.Mock
	ValidateResult bool
	ValidateError  error
	ConvertError   error
	ParseResult    []models.Transaction
	ParseError     error
	logger         logging.Logger
}

func (m *MockFullParser) Parse(r io.Reader) ([]models.Transaction, error) {
	args := m.Called(r)
	return args.Get(0).([]models.Transaction), args.Error(1)
}

func (m *MockFullParser) ConvertToCSV(inputFile, outputFile string) error {
	args := m.Called(inputFile, outputFile)
	return args.Error(0)
}

func (m *MockFullParser) SetLogger(logger logging.Logger) {
	m.Called(logger)
	m.logger = logger
}

func (m *MockFullParser) SetCategorizer(categorizer models.TransactionCategorizer) {
	m.Called(categorizer)
}

func (m *MockFullParser) ValidateFormat(file string) (bool, error) {
	args := m.Called(file)
	return args.Bool(0), args.Error(1)
}

func (m *MockFullParser) BatchConvert(inputDir, outputDir string) (int, error) {
	args := m.Called(inputDir, outputDir)
	return args.Int(0), args.Error(1)
}

// MockLegacyParser implements models.Parser for testing legacy functions
type MockLegacyParser struct {
	mock.Mock
	logger logging.Logger
}

func (m *MockLegacyParser) ConvertToCSV(inputFile, outputFile string) error {
	args := m.Called(inputFile, outputFile)
	return args.Error(0)
}

func (m *MockLegacyParser) SetLogger(logger logging.Logger) {
	m.Called(logger)
	m.logger = logger
}

func (m *MockLegacyParser) ValidateFormat(file string) (bool, error) {
	args := m.Called(file)
	return args.Bool(0), args.Error(1)
}

func (m *MockLegacyParser) BatchConvert(inputDir, outputDir string) (int, error) {
	args := m.Called(inputDir, outputDir)
	return args.Int(0), args.Error(1)
}

func (m *MockLegacyParser) Parse(r io.Reader) ([]models.Transaction, error) {
	args := m.Called(r)
	return args.Get(0).([]models.Transaction), args.Error(1)
}

func (m *MockLegacyParser) WriteToCSV(transactions []models.Transaction, csvFile string) error {
	args := m.Called(transactions, csvFile)
	return args.Error(0)
}

// Test ProcessFileWithError function
func TestProcessFileWithError_Success(t *testing.T) {
	mockParser := &MockFullParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ValidateFormat", "input.xml").Return(true, nil)
	mockParser.On("ConvertToCSV", "input.xml", "output.csv").Return(nil)
	
	// Test with validation
	err := common.ProcessFileWithError(mockParser, "input.xml", "output.csv", true, mockLogger)
	
	assert.NoError(t, err)
	mockParser.AssertExpectations(t)
}

func TestProcessFileWithError_ValidationError(t *testing.T) {
	mockParser := &MockFullParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ValidateFormat", "input.xml").Return(false, errors.New("validation failed"))
	
	// Test with validation error
	err := common.ProcessFileWithError(mockParser, "input.xml", "output.csv", true, mockLogger)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error validating file")
	mockParser.AssertExpectations(t)
}

func TestProcessFileWithError_InvalidFormat(t *testing.T) {
	mockParser := &MockFullParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ValidateFormat", "input.xml").Return(false, nil)
	
	// Test with invalid format
	err := common.ProcessFileWithError(mockParser, "input.xml", "output.csv", true, mockLogger)
	
	assert.Error(t, err)
	assert.Equal(t, common.ErrInvalidFormat, err)
	mockParser.AssertExpectations(t)
}

func TestProcessFileWithError_ConversionError(t *testing.T) {
	mockParser := &MockFullParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ConvertToCSV", "input.xml", "output.csv").Return(errors.New("conversion failed"))
	
	// Test without validation (skip validation)
	err := common.ProcessFileWithError(mockParser, "input.xml", "output.csv", false, mockLogger)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error converting to CSV")
	mockParser.AssertExpectations(t)
}

func TestProcessFileWithError_NoValidation(t *testing.T) {
	mockParser := &MockFullParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ConvertToCSV", "input.xml", "output.csv").Return(nil)
	
	// Test without validation
	err := common.ProcessFileWithError(mockParser, "input.xml", "output.csv", false, mockLogger)
	
	assert.NoError(t, err)
	mockParser.AssertExpectations(t)
}

// Test ProcessFile function (deprecated)
func TestProcessFile_Success(t *testing.T) {
	mockParser := &MockFullParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ConvertToCSV", "input.xml", "output.csv").Return(nil)
	
	// Test without validation
	assert.NotPanics(t, func() {
		common.ProcessFile(mockParser, "input.xml", "output.csv", false, mockLogger)
	})
	
	mockParser.AssertExpectations(t)
}

// Test ProcessFileLegacyWithError function
func TestProcessFileLegacyWithError_Success(t *testing.T) {
	mockParser := &MockLegacyParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ValidateFormat", "input.xml").Return(true, nil)
	mockParser.On("ConvertToCSV", "input.xml", "output.csv").Return(nil)
	
	// Test with validation
	err := common.ProcessFileLegacyWithError(mockParser, "input.xml", "output.csv", true, mockLogger)
	
	assert.NoError(t, err)
	mockParser.AssertExpectations(t)
}

func TestProcessFileLegacyWithError_ValidationError(t *testing.T) {
	mockParser := &MockLegacyParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ValidateFormat", "input.xml").Return(false, errors.New("validation failed"))
	
	// Test with validation error
	err := common.ProcessFileLegacyWithError(mockParser, "input.xml", "output.csv", true, mockLogger)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error validating file")
	mockParser.AssertExpectations(t)
}

func TestProcessFileLegacyWithError_InvalidFormat(t *testing.T) {
	mockParser := &MockLegacyParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ValidateFormat", "input.xml").Return(false, nil)
	
	// Test with invalid format
	err := common.ProcessFileLegacyWithError(mockParser, "input.xml", "output.csv", true, mockLogger)
	
	assert.Error(t, err)
	assert.Equal(t, common.ErrInvalidFormat, err)
	mockParser.AssertExpectations(t)
}

func TestProcessFileLegacyWithError_ConversionError(t *testing.T) {
	mockParser := &MockLegacyParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ConvertToCSV", "input.xml", "output.csv").Return(errors.New("conversion failed"))
	
	// Test without validation
	err := common.ProcessFileLegacyWithError(mockParser, "input.xml", "output.csv", false, mockLogger)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error converting to CSV")
	mockParser.AssertExpectations(t)
}

// Test ProcessFileLegacy function (deprecated)
func TestProcessFileLegacy_Success(t *testing.T) {
	mockParser := &MockLegacyParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ConvertToCSV", "input.xml", "output.csv").Return(nil)
	
	// Test without validation
	assert.NotPanics(t, func() {
		common.ProcessFileLegacy(mockParser, "input.xml", "output.csv", false, mockLogger)
	})
	
	mockParser.AssertExpectations(t)
}

// Test SaveMappings function (deprecated)
func TestSaveMappings(t *testing.T) {
	logger := logrus.New()
	
	// Test that SaveMappings doesn't panic
	assert.NotPanics(t, func() {
		common.SaveMappings(logger)
	})
}

// Test error constants
func TestErrInvalidFormat(t *testing.T) {
	assert.Equal(t, "file is not in a valid format", common.ErrInvalidFormat.Error())
	assert.True(t, errors.Is(common.ErrInvalidFormat, common.ErrInvalidFormat))
}

// Test edge cases - removed nil logger test as it's not a realistic scenario
func TestProcessFileWithError_EdgeCases(t *testing.T) {
	mockParser := &MockFullParser{}
	mockLogger := logging.NewLogrusAdapter("info", "text")
	
	// Setup expectations for empty file paths
	mockParser.On("SetLogger", mockLogger).Return()
	mockParser.On("ConvertToCSV", "", "").Return(nil)
	
	// Test with empty file paths
	err := common.ProcessFileWithError(mockParser, "", "", false, mockLogger)
	
	assert.NoError(t, err)
	mockParser.AssertExpectations(t)
}

// Test that the original mock implementations still work
func TestMockFullParser_ImplementsInterface(t *testing.T) {
	parser := &MockFullParser{}

	// Setup expectations for all method calls
	mockLogger := &logging.MockLogger{}
	parser.On("SetLogger", mockLogger).Return()
	parser.On("SetCategorizer", mock.Anything).Return()
	parser.On("ValidateFormat", "test.xml").Return(false, nil)
	parser.On("ConvertToCSV", "input.xml", "output.csv").Return(nil)
	parser.On("Parse", mock.Anything).Return([]models.Transaction{}, nil)

	// Test SetLogger
	parser.SetLogger(mockLogger)
	assert.NotNil(t, parser.logger)

	// Test SetCategorizer
	parser.SetCategorizer(nil)

	// Test ValidateFormat
	valid, err := parser.ValidateFormat("test.xml")
	assert.NoError(t, err)
	assert.False(t, valid)

	// Test ConvertToCSV
	err = parser.ConvertToCSV("input.xml", "output.csv")
	assert.NoError(t, err)

	// Test Parse
	txns, err := parser.Parse(nil)
	assert.NoError(t, err)
	assert.Empty(t, txns)

	parser.AssertExpectations(t)
}

// TestMockFullParser_WithErrors tests error scenarios
func TestMockFullParser_WithErrors(t *testing.T) {
	parser := &MockFullParser{
		ValidateError: assert.AnError,
		ConvertError:  assert.AnError,
		ParseError:    assert.AnError,
	}

	// Setup expectations for error scenarios
	parser.On("ValidateFormat", "test.xml").Return(false, assert.AnError)
	parser.On("ConvertToCSV", "input.xml", "output.csv").Return(assert.AnError)
	parser.On("Parse", mock.Anything).Return([]models.Transaction{}, assert.AnError)

	_, err := parser.ValidateFormat("test.xml")
	assert.Error(t, err)

	err = parser.ConvertToCSV("input.xml", "output.csv")
	assert.Error(t, err)

	_, err = parser.Parse(nil)
	assert.Error(t, err)

	parser.AssertExpectations(t)
}

// TestMockLogger_CapturesEntries tests that the mock logger captures entries
func TestMockLogger_CapturesEntries(t *testing.T) {
	logger := &logging.MockLogger{}

	logger.Info("test message")
	logger.Warn("warning message")
	logger.Error("error message")
	logger.Fatalf("fatal: %s", "critical error")

	entries := logger.GetEntries()
	assert.Len(t, entries, 4)
	assert.True(t, logger.HasEntry("INFO", "test message"))
	assert.True(t, logger.HasEntry("WARN", "warning message"))
	assert.True(t, logger.HasEntry("ERROR", "error message"))
	assert.True(t, logger.HasEntry("FATAL", "fatal: critical error"))
}
