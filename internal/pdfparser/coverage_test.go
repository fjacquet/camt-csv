package pdfparser

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter_NilExtractor(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	adapter := NewAdapter(logger, nil)
	require.NotNil(t, adapter)
	// Should have created a RealPDFExtractor
	_, ok := adapter.extractor.(*RealPDFExtractor)
	assert.True(t, ok)
}

func TestNewRealPDFExtractor(t *testing.T) {
	extractor := NewRealPDFExtractor()
	require.NotNil(t, extractor)
}

func TestAdapter_ValidateFormat_WithMockExtractor(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")

	t.Run("valid PDF", func(t *testing.T) {
		extractor := NewMockPDFExtractor("some text content", nil)
		adapter := NewAdapter(logger, extractor)

		valid, err := adapter.ValidateFormat("test.pdf")
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("invalid PDF - extraction error", func(t *testing.T) {
		extractor := NewMockPDFExtractor("", errors.New("not a PDF"))
		adapter := NewAdapter(logger, extractor)

		valid, err := adapter.ValidateFormat("bad.pdf")
		assert.NoError(t, err) // ValidateFormat returns false, nil on extraction error
		assert.False(t, valid)
	})
}

func TestAdapter_ConvertToCSV_NonexistentFile(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	extractor := NewMockPDFExtractor("", nil)
	adapter := NewAdapter(logger, extractor)

	err := adapter.ConvertToCSV(context.Background(), "/nonexistent/file.pdf", "/tmp/out.csv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error opening input file")
}

func TestAdapter_BatchConvert_NonexistentDir(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	extractor := NewMockPDFExtractor("", nil)
	adapter := NewAdapter(logger, extractor)

	n, err := adapter.BatchConvert(context.Background(), "/nonexistent/dir", "/tmp/out")
	assert.Error(t, err)
	assert.Equal(t, 0, n)
}

func TestAdapter_BatchConvert_EmptyDir(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	extractor := NewMockPDFExtractor("some text", nil)
	adapter := NewAdapter(logger, extractor)

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	n, err := adapter.BatchConvert(context.Background(), inputDir, outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	// Check manifest was written
	manifestPath := filepath.Join(outputDir, ".manifest.json")
	_, statErr := os.Stat(manifestPath)
	assert.NoError(t, statErr)
}
