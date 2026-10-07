package pdfparser

import (
	"crypto/sha256"
	"os"
	"sync"
)

// PDFExtractor defines the interface for extracting text from PDF files.
// This interface allows for dependency injection and makes the PDF parser testable
// by providing different implementations for production and testing.
type PDFExtractor interface {
	// ExtractText extracts text content from a PDF file at the given path.
	// Returns the extracted text as a string or an error if extraction fails.
	ExtractText(pdfPath string) (string, error)
}

// RealPDFExtractor implements PDFExtractor using the actual pdftotext command.
// This is the production implementation that requires pdftotext to be installed.
type RealPDFExtractor struct{}

// NewRealPDFExtractor creates a new RealPDFExtractor instance.
func NewRealPDFExtractor() *RealPDFExtractor {
	return &RealPDFExtractor{}
}

// ExtractText extracts text from a PDF file using the pdftotext command.
func (e *RealPDFExtractor) ExtractText(pdfPath string) (string, error) {
	return extractTextFromPDFImpl(pdfPath)
}

// MockPDFExtractor implements PDFExtractor for testing purposes.
// It returns predefined mock data instead of actually extracting from PDF files.
type MockPDFExtractor struct {
	MockText string
	MockErr  error
}

// NewMockPDFExtractor creates a new MockPDFExtractor with the given mock data.
func NewMockPDFExtractor(mockText string, mockErr error) *MockPDFExtractor {
	return &MockPDFExtractor{
		MockText: mockText,
		MockErr:  mockErr,
	}
}

// ExtractText returns the predefined mock text or error.
func (e *MockPDFExtractor) ExtractText(pdfPath string) (string, error) {
	if e.MockErr != nil {
		return "", e.MockErr
	}
	return e.MockText, nil
}

// cachingExtractor runs the inner extractor once per distinct PDF content.
// Format detection extracts the original file, then Parse extracts a temp
// copy of the same bytes; keying by content hash makes the second a hit.
type cachingExtractor struct {
	inner PDFExtractor
	mu    sync.Mutex
	texts map[[sha256.Size]byte]string
}

func newCachingExtractor(inner PDFExtractor) PDFExtractor {
	return &cachingExtractor{inner: inner, texts: map[[sha256.Size]byte]string{}}
}

func (c *cachingExtractor) ExtractText(pdfPath string) (string, error) {
	data, err := os.ReadFile(pdfPath) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return c.inner.ExtractText(pdfPath)
	}
	sum := sha256.Sum256(data)
	c.mu.Lock()
	text, ok := c.texts[sum]
	c.mu.Unlock()
	if ok {
		return text, nil
	}
	text, err = c.inner.ExtractText(pdfPath)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.texts[sum] = text
	c.mu.Unlock()
	return text, nil
}
