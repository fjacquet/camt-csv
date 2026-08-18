// Package root contains the root command for the application
package root

import (
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/logging"
	"log"
	"sync"

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

	// registerFinalizeOnce guards the logrus exit-handler registration, which
	// must happen exactly once even if a test initializes the container twice.
	registerFinalizeOnce sync.Once

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

			// Command flags shadow the file/env configuration, so they must be
			// applied before the container reads it.
			applyFlagOverrides(cmd)

			// Initialize container with dependency injection
			initializeContainer()

			// Note: Logger is now injected through dependency injection container
			// Individual parsers receive loggers through their constructors

			// CSV delimiter is now a constant (models.DefaultCSVDelimiter = ',')
			// Configuration value is logged for reference but no longer used to set the delimiter
			Log.WithField(logging.FieldDelimiter, string(common.Delimiter)).Debug("Using CSV delimiter")
		},
		// Add a PersistentPostRun hook to save party mappings when ANY command finishes
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			finalize()
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

// applyFlagOverrides copies command flags that shadow configuration onto the
// loaded config.
//
// viper.BindPFlag cannot do this: InitializeConfig builds its own Viper
// instance, so a binding registered on the global one is never read.
func applyFlagOverrides(cmd *cobra.Command) {
	if f := cmd.Flags().Lookup("keep-payments"); f != nil && f.Changed {
		keep, err := cmd.Flags().GetBool("keep-payments")
		if err != nil {
			Log.WithError(err).Warn("Ignoring unreadable --keep-payments flag")
		} else {
			AppConfig.Parsers.Viseca.KeepPayments = keep
		}
	}
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

	// Commands end early by logging at fatal level, which exits the process
	// without unwinding to PersistentPostRun. logrus runs its exit handlers
	// before that exit, so registering finalize here is what makes the
	// mapping save survive every Fatal/Fatalf call site under cmd/.
	registerFinalizeOnce.Do(func() { logrus.RegisterExitHandler(finalize) })
}

// finalize saves the learned party mappings and releases background work.
// It runs after a command completes normally (PersistentPostRun) and before a
// fatal-level log exits the process. Both paths may fire in the same run, so
// it is safe to call more than once: the save is a no-op unless the mappings
// are dirty, and Shutdown tolerates repeat calls.
func finalize() {
	if AppContainer == nil {
		Log.Warn("Container not initialized, skipping category mapping save")
		return
	}

	categorizerInstance := AppContainer.GetCategorizer()
	if err := categorizerInstance.SaveCreditorsToYAML(); err != nil {
		Log.WithError(err).Warn("Failed to save creditor mappings")
	}

	if err := categorizerInstance.SaveDebitorsToYAML(); err != nil {
		Log.WithError(err).Warn("Failed to save debitor mappings")
	}

	// Stop any embedding warm-up still running so it does not keep
	// issuing API calls while the command is shutting down.
	AppContainer.Close()
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
	Cmd.PersistentFlags().Bool("auto-learn", false, "Enable AI auto-learning of categorizations (default: false)")

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
	if err := viper.BindPFlag("categorization.auto_learn", Cmd.PersistentFlags().Lookup("auto-learn")); err != nil {
		log.Printf("Warning: failed to bind auto-learn flag: %v", err)
	}
}
