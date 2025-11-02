package pdfparser

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
	return extractTextFromPDF(pdfPath)
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
