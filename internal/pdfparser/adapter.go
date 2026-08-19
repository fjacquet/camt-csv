package pdfparser

import (
	"context"
	"io"
	"os"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the parser.FullParser interface for PDF bank statements.
type Adapter struct {
	parser.BaseParser
	extractor PDFExtractor
}

// NewAdapter creates a new adapter for the pdfparser with dependency injection.
func NewAdapter(logger logging.Logger, extractor PDFExtractor) *Adapter {
	if extractor == nil {
		extractor = NewRealPDFExtractor()
	}
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
		extractor:  extractor,
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithExtractorAndCategorizer(ctx, r, a.extractor, a.GetLogger(), a.GetCategorizer())
}

// pdfMagicHeader is the byte signature every PDF file starts with.
const pdfMagicHeader = "%PDF-"

// ValidateFormat checks if a file is a valid PDF file.
//
// The magic header is checked before the file is handed to the extractor:
// auto-detection now offers every file in a batch to every parser in turn, so
// without this a directory of N non-PDF statements would spawn N pdftotext
// subprocesses for files that are never going to validate.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	f, err := os.Open(file) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return false, nil
	}
	defer func() {
		_ = f.Close()
	}()

	header := make([]byte, len(pdfMagicHeader))
	if n, _ := io.ReadFull(f, header); n < len(pdfMagicHeader) || string(header) != pdfMagicHeader {
		return false, nil
	}

	a.GetLogger().Debug("Validating PDF format",
		logging.Field{Key: "file", Value: file})

	// Try to extract text as a validation check using the injected extractor.
	// Logged at debug, not error: a detection probe rejecting a corrupted or
	// unreadable PDF is a routine outcome of offering every file in a batch
	// to every parser, not a system error worth an ERROR line per file.
	if _, err := a.extractor.ExtractText(file); err != nil {
		a.GetLogger().WithError(err).Debug("PDF text extraction failed during validation")
		return false, nil
	}

	return true, nil
}
