// Package batch handles batch processing of files
package batch

import (
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/factory"
	"fmt"
	"os"

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

	// Get CAMT parser from factory with logger
	parser, err := factory.GetParserWithLogger(factory.CAMT, logger)
	if err != nil {
		logger.Fatalf("Failed to get CAMT parser: %v", err)
	}
	
	count, err := parser.BatchConvert(inputDir, outputDir)
	if err != nil {
		logger.Fatalf("Error during batch conversion: %v", err)
	}

	root.Log.Info(fmt.Sprintf("Batch processing completed. %d files converted.", count))
}
