// Package root contains the root command for the application
package root

import (
	"fjacquet/camt-csv/cmd/analyze"
	"fjacquet/camt-csv/cmd/implement"
	"fjacquet/camt-csv/cmd/review"
	"fjacquet/camt-csv/cmd/tasks"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/store"
	"log"

	"github.com/sirupsen/logrus"
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
	Log = logging.NewLogrusAdapter("info", "text")

	// Global configuration instance
	AppConfig *config.Config

	// Global container instance for dependency injection
	AppContainer *container.Container

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

			// Initialize container with dependency injection
			initializeContainer()

			// Note: Logger is now injected through dependency injection container
			// Individual parsers receive loggers through their constructors

			// Set CSV delimiter from configuration
			commonDelim := []rune(AppConfig.CSV.Delimiter)[0]
			common.SetDelimiter(commonDelim)
			Log.WithField(logging.FieldDelimiter, AppConfig.CSV.Delimiter).Debug("Setting CSV delimiter from configuration")
		},
		// Add a PersistentPostRun hook to save party mappings when ANY command finishes
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Save the creditor and debitor mappings back to disk after any command runs
			if AppContainer != nil {
				// Use the container's categorizer (preferred method)
				categorizerInstance := AppContainer.GetCategorizer()
				err := categorizerInstance.SaveCreditorsToYAML()
				if err != nil {
					Log.WithError(err).Warn("Failed to save creditor mappings")
				}

				err = categorizerInstance.SaveDebitorsToYAML()
				if err != nil {
					Log.WithError(err).Warn("Failed to save debitor mappings")
				}
			} else {
				// Fallback to old method for backward compatibility
				// Deprecated: This will be removed in v2.0.0
				categorizerInstance := categorizer.NewCategorizer(nil, store.NewCategoryStore("categories.yaml", "creditors.yaml", "debitors.yaml"), Log)
				err := categorizerInstance.SaveCreditorsToYAML()
				if err != nil {
					Log.WithError(err).Warn("Failed to save creditor mappings")
				}

				err = categorizerInstance.SaveDebitorsToYAML()
				if err != nil {
					Log.WithError(err).Warn("Failed to save debitor mappings")
				}
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
	logrusLogger := config.ConfigureLoggingFromConfig(AppConfig)
	Log = logging.NewLogrusAdapterFromLogger(logrusLogger)
}

// initializeContainer creates the dependency injection container
func initializeContainer() {
	var err error
	AppContainer, err = container.NewContainer(AppConfig)
	if err != nil {
		Log.Fatalf("Failed to initialize container: %v", err)
	}

	// Update the global logger to use the container's logger
	Log = AppContainer.GetLogger()
}

// GetLogrusAdapter returns the logger as a LogrusAdapter for backward compatibility
func GetLogrusAdapter() *logging.LogrusAdapter {
	if adapter, ok := Log.(*logging.LogrusAdapter); ok {
		return adapter
	}
	return logging.NewLogrusAdapterFromLogger(logrus.New()).(*logging.LogrusAdapter)
}

// GetContainer returns the global container instance for dependency injection
func GetContainer() *container.Container {
	return AppContainer
}

// GetConfig returns the global configuration instance
//
// Deprecated: Use GetContainer().GetConfig() instead for dependency injection.
// This function will be removed in v3.0.0.
//
// Migration example:
//
//	// Old code:
//	config := GetConfig()
//
//	// New code:
//	container, err := container.NewContainer(config.Load())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	config := container.GetConfig()
func GetConfig() *config.Config {
	return AppConfig
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

	Cmd.AddCommand(review.GetReviewCommand())
	Cmd.AddCommand(analyze.AnalyzeCmd)
	Cmd.AddCommand(implement.ImplementCmd)
	Cmd.AddCommand(tasks.TasksCmd)

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
