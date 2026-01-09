// Package pdf handles PDF statement conversion commands
package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	internalcommon "fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"github.com/spf13/cobra"
)

// Cmd represents the pdf command
var Cmd = &cobra.Command{
	Use:   "pdf",
	Short: "Convert PDF to CSV",
	Long: `Convert PDF bank statements to CSV format.

Supports two modes:
  1. Single file: Convert one PDF to CSV
  2. Directory: Consolidate all PDFs in directory to single CSV

Examples:
  # Single file
  camt-csv pdf -i statement.pdf -o output.csv

  # Directory consolidation
  camt-csv pdf -i pdf_dir/ -o consolidated.csv

Directory mode parses all PDF files and consolidates their transactions
into a single CSV file, sorted chronologically by date.`,
	Run: pdfFunc,
}

func pdfFunc(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()
	root.Log.Info("PDF convert command called")

	inputPath := root.SharedFlags.Input
	logger.Infof("Input: %s", inputPath)
	logger.Infof("Output: %s", root.SharedFlags.Output)

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container
	p, err := appContainer.GetParser(container.PDF)
	if err != nil {
		logger.Fatalf("Error getting PDF parser: %v", err)
	}

	// Check if input is directory or file
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		logger.Fatalf("Error accessing input path: %v", err)
	}

	if fileInfo.IsDir() {
		// Directory mode - consolidate all PDFs
		count, err := consolidatePDFDirectory(ctx, p, inputPath,
			root.SharedFlags.Output, root.SharedFlags.Validate, logger)
		if err != nil {
			logger.Fatalf("Error consolidating PDFs: %v", err)
		}
		logger.Infof("Consolidated %d PDF files successfully!", count)
	} else {
		// File mode - existing flow
		common.ProcessFile(ctx, p, inputPath, root.SharedFlags.Output,
			root.SharedFlags.Validate, root.Log)
		root.Log.Info("PDF to CSV conversion completed successfully!")
	}
}

// consolidatePDFDirectory consolidates all PDF files in a directory into a single CSV
func consolidatePDFDirectory(ctx context.Context, p parser.FullParser,
	inputDir, outputFile string, validate bool, logger logging.Logger) (int, error) {

	logger.Info("Consolidating PDF files from directory",
		logging.Field{Key: "inputDir", Value: inputDir},
		logging.Field{Key: "outputFile", Value: outputFile})

	// Read directory and filter PDF files
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read input directory: %w", err)
	}

	var pdfFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		if strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
			pdfFiles = append(pdfFiles, filepath.Join(inputDir, fileName))
		}
	}

	if len(pdfFiles) == 0 {
		logger.Warn("No PDF files found in directory")
		return 0, nil
	}

	logger.Info("Found PDF files", logging.Field{Key: "count", Value: len(pdfFiles)})

	// Parse all PDF files and collect transactions
	var allTransactions []models.Transaction
	processedCount := 0

	for _, pdfFile := range pdfFiles {
		// Check for cancellation
		select {
		case <-ctx.Done():
			logger.Warn("Consolidation cancelled",
				logging.Field{Key: "processed", Value: processedCount},
				logging.Field{Key: "total", Value: len(pdfFiles)})
			return processedCount, ctx.Err()
		default:
		}

		logger.Debug("Processing PDF", logging.Field{Key: "file", Value: filepath.Base(pdfFile)})

		// Validate if requested
		if validate {
			isValid, err := p.ValidateFormat(pdfFile)
			if err != nil {
				logger.WithError(err).Warn("Error validating PDF",
					logging.Field{Key: "file", Value: filepath.Base(pdfFile)})
				continue // Skip this file
			}
			if !isValid {
				logger.Warn("Skipping invalid PDF",
					logging.Field{Key: "file", Value: filepath.Base(pdfFile)})
				continue
			}
		}

		// Open and parse the file
		file, err := os.Open(pdfFile) // #nosec G304 -- CLI tool requires user-provided paths
		if err != nil {
			logger.WithError(err).Warn("Failed to open PDF",
				logging.Field{Key: "file", Value: filepath.Base(pdfFile)})
			continue
		}

		transactions, err := p.Parse(ctx, file)
		if closeErr := file.Close(); closeErr != nil {
			logger.WithError(closeErr).Warn("Failed to close PDF file",
				logging.Field{Key: "file", Value: filepath.Base(pdfFile)})
		}

		if err != nil {
			logger.WithError(err).Warn("Failed to parse PDF",
				logging.Field{Key: "file", Value: filepath.Base(pdfFile)})
			continue
		}

		logger.Debug("Parsed transactions",
			logging.Field{Key: "file", Value: filepath.Base(pdfFile)},
			logging.Field{Key: "count", Value: len(transactions)})

		allTransactions = append(allTransactions, transactions...)
		processedCount++
	}

	if len(allTransactions) == 0 {
		logger.Warn("No transactions found in any PDF files")
		return processedCount, fmt.Errorf("no transactions extracted from PDF files")
	}

	// Sort transactions chronologically
	sortTransactionsChronologically(allTransactions)

	logger.Info("Writing consolidated transactions",
		logging.Field{Key: "total_transactions", Value: len(allTransactions)},
		logging.Field{Key: "output", Value: outputFile})

	// Write consolidated CSV
	if err := internalcommon.WriteTransactionsToCSVWithLogger(allTransactions, outputFile, logger); err != nil {
		return processedCount, fmt.Errorf("failed to write consolidated CSV: %w", err)
	}

	logger.Info("Successfully wrote consolidated CSV",
		logging.Field{Key: "files_processed", Value: processedCount},
		logging.Field{Key: "total_transactions", Value: len(allTransactions)},
		logging.Field{Key: "output", Value: outputFile})

	return processedCount, nil
}

// sortTransactionsChronologically sorts transactions by date, then value date, then amount
func sortTransactionsChronologically(transactions []models.Transaction) {
	sort.Slice(transactions, func(i, j int) bool {
		// Primary sort: by transaction date
		if !transactions[i].Date.Equal(transactions[j].Date) {
			return transactions[i].Date.Before(transactions[j].Date)
		}

		// Secondary sort: by value date
		if !transactions[i].ValueDate.Equal(transactions[j].ValueDate) {
			return transactions[i].ValueDate.Before(transactions[j].ValueDate)
		}

		// Tertiary sort: by amount (for consistency)
		return transactions[i].Amount.LessThan(transactions[j].Amount)
	})
}
