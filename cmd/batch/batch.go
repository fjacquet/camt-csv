// Package batch handles batch processing of files
package batch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"github.com/spf13/cobra"
)

// Cmd represents the batch command
var Cmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch process files from a directory",
	Long: `Batch process files from an input directory and output them to another directory.

The batch command processes all XML files in the input directory and converts them to CSV format
with AI-powered categorization. Each file is validated and converted independently.

Example:
  camt-csv batch -i input_dir/ -o output_dir/`,
	Run: batchFunc,
}

func init() {
	// Override the usage text for the input/output flags in batch context
	Cmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags (for batch, -i/-o refer to directories):
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
}

func batchFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Batch command called")

	// Use the shared flags from root command
	inputDir := root.SharedFlags.Input
	outputDir := root.SharedFlags.Output

	logger := root.GetLogrusAdapter()
	logger.Infof("Input directory: %s", inputDir)
	logger.Infof("Output directory: %s", outputDir)

	if inputDir == "" || outputDir == "" {
		logger.Fatal("Input and output directories must be specified")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		logger.Fatalf("Failed to create output directory: %v", err)
	}

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container (includes categorizer via DI)
	parser, err := appContainer.GetParser(container.CAMT)
	if err != nil {
		logger.Fatalf("Failed to get CAMT parser: %v", err)
	}

	// Use new aggregation-based batch processing
	count, err := batchConvertWithAggregation(inputDir, outputDir, parser, logger)
	if err != nil {
		logger.Fatalf("Error during batch conversion: %v", err)
	}

	root.Log.Info(fmt.Sprintf("Batch processing completed. %d consolidated files created.", count))
}

// batchConvertWithAggregation performs batch conversion using the new aggregation engine
func batchConvertWithAggregation(inputDir, outputDir string, p parser.FullParser, logger logging.Logger) (int, error) {
	// Create batch aggregator
	aggregator := batch.NewBatchAggregator(logger)

	// Read input directory to find all files
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read input directory: %w", err)
	}

	// Filter for supported file types and build full paths
	var inputFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		// Support XML files (CAMT) for now - can be extended for other formats
		if strings.HasSuffix(strings.ToLower(fileName), ".xml") {
			inputFiles = append(inputFiles, filepath.Join(inputDir, fileName))
		}
	}

	if len(inputFiles) == 0 {
		logger.Warn("No supported files found in input directory")
		return 0, nil
	}

	logger.Info("Found files for processing",
		logging.Field{Key: "count", Value: len(inputFiles)})

	// Group files by account
	fileGroups, err := aggregator.GroupFilesByAccount(inputFiles)
	if err != nil {
		return 0, fmt.Errorf("failed to group files by account: %w", err)
	}

	logger.Info("Grouped files into account groups",
		logging.Field{Key: "groups", Value: len(fileGroups)})

	// Process each account group
	consolidatedCount := 0
	for _, group := range fileGroups {
		logger.Info("Processing account group",
			logging.Field{Key: "account", Value: group.AccountID},
			logging.Field{Key: "files", Value: len(group.Files)})

		// Create parse function for this parser
		parseFunc := func(filePath string) ([]models.Transaction, error) {
			// Validate file format first
			if validator, ok := p.(parser.Validator); ok {
				isValid, err := validator.ValidateFormat(filePath)
				if err != nil {
					logger.WithError(err).Warn("Error validating file format",
						logging.Field{Key: "file", Value: filepath.Base(filePath)})
					return nil, fmt.Errorf("validation error: %w", err)
				}
				if !isValid {
					logger.Debug("Skipping invalid file format",
						logging.Field{Key: "file", Value: filepath.Base(filePath)})
					return []models.Transaction{}, nil // Return empty slice for invalid files
				}
			}

			// Open and parse the file
			file, err := os.Open(filePath) // #nosec G304 -- CLI tool requires user-provided file paths
			if err != nil {
				return nil, fmt.Errorf("failed to open file: %w", err)
			}
			defer func() {
				if cerr := file.Close(); cerr != nil {
					logger.WithError(cerr).Warn("Failed to close file")
				}
			}()

			return p.Parse(file)
		}

		// Aggregate transactions from all files in the group
		transactions, err := aggregator.AggregateTransactions(group, parseFunc)
		if err != nil {
			logger.WithError(err).Error("Failed to aggregate transactions for account",
				logging.Field{Key: "account", Value: group.AccountID})
			continue // Skip this group but continue with others
		}

		if len(transactions) == 0 {
			logger.Warn("No transactions found for account group",
				logging.Field{Key: "account", Value: group.AccountID})
			continue
		}

		// Calculate actual date range from transactions if not available from filenames
		actualDateRange := group.DateRange
		if actualDateRange.Start.IsZero() || actualDateRange.End.IsZero() {
			actualDateRange = aggregator.CalculateDateRangeFromTransactions(transactions)
		}

		// Generate output filename
		outputFilename := aggregator.GenerateOutputFilename(group.AccountID, actualDateRange)
		outputPath := filepath.Join(outputDir, outputFilename)

		// Generate source file header
		var sourceFileNames []string
		for _, filePath := range group.Files {
			sourceFileNames = append(sourceFileNames, filepath.Base(filePath))
		}
		headerComment := aggregator.GenerateSourceFileHeader(sourceFileNames)

		// Write consolidated CSV file with header
		if err := writeConsolidatedCSV(transactions, outputPath, headerComment, logger); err != nil {
			logger.WithError(err).Error("Failed to write consolidated CSV",
				logging.Field{Key: "account", Value: group.AccountID},
				logging.Field{Key: "output", Value: outputPath})
			continue
		}

		logger.Info("Created consolidated file",
			logging.Field{Key: "account", Value: group.AccountID},
			logging.Field{Key: "transactions", Value: len(transactions)},
			logging.Field{Key: "output", Value: outputFilename})

		consolidatedCount++
	}

	return consolidatedCount, nil
}

// writeConsolidatedCSV writes transactions to CSV with a custom header comment
func writeConsolidatedCSV(transactions []models.Transaction, outputPath, headerComment string, logger logging.Logger) error {
	// Create the output file
	file, err := os.Create(outputPath) // #nosec G304 -- CLI tool requires user-provided output paths
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			logger.WithError(cerr).Warn("Failed to close output file")
		}
	}()

	// Write header comment if provided
	if headerComment != "" {
		if _, err := file.WriteString(headerComment); err != nil {
			return fmt.Errorf("failed to write header comment: %w", err)
		}
	}

	// Use the common CSV writing function to write transactions
	// We need to write to a temporary file first, then copy the content after the header
	tempFile, err := os.CreateTemp("", "consolidated_*.csv")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFileName := tempFile.Name()
	defer func() {
		if rerr := os.Remove(tempFileName); rerr != nil {
			logger.WithError(rerr).Warn("Failed to remove temporary file")
		}
	}()
	defer func() {
		if cerr := tempFile.Close(); cerr != nil {
			logger.WithError(cerr).Warn("Failed to close temporary file")
		}
	}()

	// Write transactions to temp file using common function
	if err := common.WriteTransactionsToCSVWithLogger(transactions, tempFileName, logger); err != nil {
		return fmt.Errorf("failed to write transactions to temp file: %w", err)
	}

	// Read the temp file and append to our output file (after the header)
	if cerr := tempFile.Close(); cerr != nil {
		logger.WithError(cerr).Warn("Failed to close temporary file before reading")
	}
	tempContent, err := os.ReadFile(tempFileName) // #nosec G304 -- Reading from temp file we just created
	if err != nil {
		return fmt.Errorf("failed to read temporary file: %w", err)
	}

	if _, err := file.Write(tempContent); err != nil {
		return fmt.Errorf("failed to write CSV content: %w", err)
	}

	return nil
}
