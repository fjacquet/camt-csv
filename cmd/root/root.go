// Package root contains the root command for the application
package root

import (
	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/pdfparser"
	"fjacquet/camt-csv/internal/revolutinvestmentparser"
	"fjacquet/camt-csv/internal/revolutparser"
	"fjacquet/camt-csv/internal/selmaparser"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CommonFlags represents the flags that are common to multiple commands
type CommonFlags struct {
	Input    string
	Output   string
	Validate bool
}

var (
	// Log is the shared logger instance for commands - will be updated with config
	Log = logging.GetLogger()

	// Global configuration instance
	AppConfig *config.Config

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
			// Initialize configuration first
			initializeConfiguration()
			
			// Set the configured logger for all parsers
			// This will propagate our centralized logging configuration to each package
			camtparser.SetLogger(Log)
			pdfparser.SetLogger(Log)
			selmaparser.SetLogger(Log)
			revolutparser.SetLogger(Log)
			revolutinvestmentparser.SetLogger(Log)
			debitparser.SetLogger(Log)
			categorizer.SetLogger(Log)
			common.SetLogger(Log)

			// Set CSV delimiter from configuration
			commonDelim := []rune(AppConfig.CSV.Delimiter)[0]
			common.SetDelimiter(commonDelim)
			Log.WithField("delimiter", AppConfig.CSV.Delimiter).Debug("Setting CSV delimiter from configuration")
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

// initializeConfiguration loads the configuration using Viper and sets up logging
func initializeConfiguration() {
	var err error
	AppConfig, err = config.InitializeConfig()
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Configure logging based on the loaded configuration
	Log = config.ConfigureLoggingFromConfig(AppConfig)
	
	// Set the configured logger as the global logger
	logging.SetLogger(Log)
}

// Init initializes the root command and all flags
func Init() {
	// Add persistent flags to root command for common options
	Cmd.PersistentFlags().StringVarP(&SharedFlags.Input, "input", "i", "", "Input file")
	Cmd.PersistentFlags().StringVarP(&SharedFlags.Output, "output", "o", "", "Output file")
	Cmd.PersistentFlags().BoolVarP(&SharedFlags.Validate, "validate", "v", false, "Validate file format before conversion")

	// Add configuration-related flags
	Cmd.PersistentFlags().String("config", "", "Config file (default is $HOME/.camt-csv/config.yaml)")
	Cmd.PersistentFlags().String("log-level", "", "Log level (debug, info, warn, error)")
	Cmd.PersistentFlags().String("log-format", "", "Log format (text, json)")
	Cmd.PersistentFlags().String("csv-delimiter", "", "CSV delimiter character")
	Cmd.PersistentFlags().Bool("ai-enabled", false, "Enable AI categorization")

	// Bind flags to viper
	if err := viper.BindPFlag("log.level", Cmd.PersistentFlags().Lookup("log-level")); err != nil {
		log.Printf("Warning: failed to bind log-level flag: %v", err)
	}
	if err := viper.BindPFlag("log.format", Cmd.PersistentFlags().Lookup("log-format")); err != nil {
		log.Printf("Warning: failed to bind log-format flag: %v", err)
	}
	if err := viper.BindPFlag("csv.delimiter", Cmd.PersistentFlags().Lookup("csv-delimiter")); err != nil {
		log.Printf("Warning: failed to bind csv-delimiter flag: %v", err)
	}
	if err := viper.BindPFlag("ai.enabled", Cmd.PersistentFlags().Lookup("ai-enabled")); err != nil {
		log.Printf("Warning: failed to bind ai-enabled flag: %v", err)
	}
}
