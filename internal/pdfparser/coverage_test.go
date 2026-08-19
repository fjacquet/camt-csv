package pdfparser

import (
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
	tempDir := t.TempDir()

	t.Run("valid PDF", func(t *testing.T) {
		pdfFile := filepath.Join(tempDir, "test.pdf")
		require.NoError(t, os.WriteFile(pdfFile, []byte("%PDF-1.7\nsome text content"), 0600))

		extractor := NewMockPDFExtractor("some text content", nil)
		adapter := NewAdapter(logger, extractor)

		valid, err := adapter.ValidateFormat(pdfFile)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("invalid PDF - extraction error", func(t *testing.T) {
		pdfFile := filepath.Join(tempDir, "bad.pdf")
		require.NoError(t, os.WriteFile(pdfFile, []byte("%PDF-1.7\ncorrupted"), 0600))

		extractor := NewMockPDFExtractor("", errors.New("not a PDF"))
		adapter := NewAdapter(logger, extractor)

		valid, err := adapter.ValidateFormat(pdfFile)
		assert.NoError(t, err) // ValidateFormat returns false, nil on extraction error
		assert.False(t, valid)
	})
}
