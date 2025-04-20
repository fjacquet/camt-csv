// Package root contains the root command for the application
package root

import (
	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/pdfparser"
	"fjacquet/camt-csv/internal/revolutparser"
	"fjacquet/camt-csv/internal/selmaparser"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// CommonFlags represents the flags that are common to multiple commands
type CommonFlags struct {
	Input    string
	Output   string
	Validate bool
}

var (
	// Log is the shared logger instance for commands
	Log = logrus.New()
	
	// Cmd is the root command
	Cmd = &cobra.Command{
		Use:   "camt-csv",
		Short: "A CLI tool to convert CAMT.053 XML files to CSV and categorize transactions.",
		Long: `camt-csv is a CLI tool that converts CAMT.053 XML files to CSV format.
It also provides transaction categorization based on the party's name.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			Log.Info("Welcome to camt-csv!")
			Log.Info("Use --help to see available commands")
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize and configure logging
			config.LoadEnv()
			Log = config.ConfigureLogging()
			
			// Set the configured logger for all parsers
			camtparser.SetLogger(Log)
			pdfparser.SetLogger(Log)
			selmaparser.SetLogger(Log)
			revolutparser.SetLogger(Log)
			debitparser.SetLogger(Log)
			
			// Ensure CSV delimiter is updated after env variables are loaded
			if delim := os.Getenv("CSV_DELIMITER"); delim != "" {
				Log.WithField("delimiter", delim).Debug("Setting CSV delimiter from environment")
				commonDelim := []rune(delim)[0]
				common.SetDelimiter(commonDelim)
			}
		},
		// Add a PersistentPostRun hook to save party mappings when ANY command finishes
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Save the creditor and debitor mappings back to disk after any command runs
			err := categorizer.SaveCreditorsToYAML()
			if err != nil {
				Log.Warnf("Failed to save creditor mappings: %v", err)
			}
			
			err = categorizer.SaveDebitorsToYAML()
			if err != nil {
				Log.Warnf("Failed to save debitor mappings: %v", err)
			}
		},
	}

	// Common flags accessible to all commands
	SharedFlags = CommonFlags{}
	
	// Specific batch command flags
	InputDir  string
	OutputDir string
	
	// Specific categorize command flags
	PartyName string
	IsDebtor  bool
	Amount    string
	Date      string
	Info      string
)

// Init initializes the root command and all flags
func Init() {
	// Add persistent flags to root command for common options
	Cmd.PersistentFlags().StringVarP(&SharedFlags.Input, "input", "i", "", "Input file")
	Cmd.PersistentFlags().StringVarP(&SharedFlags.Output, "output", "o", "", "Output file")
	Cmd.PersistentFlags().BoolVarP(&SharedFlags.Validate, "validate", "v", false, "Validate file format before conversion")
}
