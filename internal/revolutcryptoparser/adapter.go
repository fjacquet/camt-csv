package revolutcryptoparser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the parser.FullParser interface for Revolut Crypto CSV files.
type Adapter struct {
	parser.BaseParser
}

// NewAdapter creates a new Adapter for the revolutcryptoparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithCategorizer(r, a.GetLogger(), a.GetCategorizer())
}

// ConvertToCSV implements parser.FullParser.ConvertToCSV.
func (a *Adapter) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
	return a.ConvertToCSVDefault(ctx, inputFile, outputFile, a.Parse)
}

// ValidateFormat checks if a file is a valid Revolut Crypto CSV file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	f, err := os.Open(file) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			a.GetLogger().WithError(err).Warn("Failed to close file during format validation",
				logging.Field{Key: "file", Value: file})
		}
	}()

	header, err := csv.NewReader(f).Read()
	if err != nil || len(header) < 7 {
		return false, nil
	}
	// Check for the three distinctive headers that differ from standard Revolut CSV
	required := map[string]bool{"Symbol": false, "Type": false, "Date": false}
	for _, h := range header {
		if _, ok := required[strings.TrimSpace(h)]; ok {
			required[strings.TrimSpace(h)] = true
		}
	}
	for _, found := range required {
		if !found {
			return false, nil
		}
	}
	return true, nil
}

// BatchConvert converts all Revolut Crypto CSV files in inputDir to outputDir.
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
	logger := a.GetLogger()
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read input directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
			continue
		}

		inputPath := filepath.Join(inputDir, file.Name())
		outputPath := filepath.Join(outputDir, file.Name())

		valid, err := a.ValidateFormat(inputPath)
		if err != nil || !valid {
			logger.WithError(err).Warn("Skipping invalid file", logging.Field{Key: "file", Value: file.Name()})
			continue
		}

		if err := a.ConvertToCSV(ctx, inputPath, outputPath); err != nil {
			logger.WithError(err).Warn("Failed to convert file", logging.Field{Key: "file", Value: file.Name()})
			continue
		}
		count++
	}

	logger.Info("Batch conversion complete", logging.Field{Key: "filesConverted", Value: count})
	return count, nil
}
