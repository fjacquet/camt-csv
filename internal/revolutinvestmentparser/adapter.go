package revolutinvestmentparser

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

// Adapter implements the parser.FullParser interface for Revolut investment CSV files.
type Adapter struct {
	parser.BaseParser
}

// NewAdapter creates a new adapter for the revolutinvestmentparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	return ParseWithCategorizer(r, a.GetLogger(), a.GetCategorizer())
}

// ConvertToCSV implements parser.FullParser.ConvertToCSV
func (a *Adapter) ConvertToCSV(ctx context.Context, inputFile, outputFile string) error {
	file, err := os.Open(inputFile) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			a.GetLogger().WithError(err).Warn("Failed to close input file",
				logging.Field{Key: "file", Value: inputFile})
		}
	}()

	transactions, err := a.Parse(ctx, file)
	if err != nil {
		return err
	}

	return a.WriteToCSV(transactions, outputFile)
}

// ValidateFormat checks if a file is a valid Revolut Investment CSV file.
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

	// For now, we'll just check if it's a valid CSV file
	// A more robust implementation would check for specific headers
	_, err = csv.NewReader(f).Read()
	return err == nil, nil
}

// BatchConvert converts all Revolut investment CSV files in inputDir to standard CSV format in outputDir.
// Returns the count of successfully converted files.
func (a *Adapter) BatchConvert(ctx context.Context, inputDir, outputDir string) (int, error) {
	logger := a.GetLogger()
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	logger.Info("Starting batch conversion",
		logging.Field{Key: "inputDir", Value: inputDir},
		logging.Field{Key: "outputDir", Value: outputDir})

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Read input directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read input directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only process CSV files
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
			logger.Debug("Skipping non-CSV file", logging.Field{Key: "file", Value: file.Name()})
			continue
		}

		inputPath := filepath.Join(inputDir, file.Name())
		outputPath := filepath.Join(outputDir, file.Name())

		logger.Info("Converting file",
			logging.Field{Key: "input", Value: inputPath},
			logging.Field{Key: "output", Value: outputPath})

		// Validate format before conversion
		valid, err := a.ValidateFormat(inputPath)
		if err != nil || !valid {
			logger.WithError(err).Warn("Skipping invalid file", logging.Field{Key: "file", Value: file.Name()})
			continue
		}

		// Convert file
		if err := a.ConvertToCSV(ctx, inputPath, outputPath); err != nil {
			logger.WithError(err).Warn("Failed to convert file", logging.Field{Key: "file", Value: file.Name()})
			continue
		}

		count++
	}

	logger.Info("Batch conversion complete",
		logging.Field{Key: "filesConverted", Value: count})

	return count, nil
}
