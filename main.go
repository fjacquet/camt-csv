package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/cmd/categorize"
	"fjacquet/camt-csv/cmd/convert"
	"fjacquet/camt-csv/cmd/root"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Build-time variables injected via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// 1. Load environment variables silently first (no logging yet)
	loadEnvSilently()

	// 2. Configure global log level directly - this affects ALL new loggers
	configureLogLevelDirectly()

	// 3. Logging level is now handled by the configuration system

	// 4. Now that logging is properly configured, initialize root command
	root.Init()

	// 5. Set version from build-time ldflags
	root.Cmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)

	// 6. Add all subcommands
	root.Cmd.AddCommand(convert.Cmd)
	root.Cmd.AddCommand(categorize.Cmd)
}

// loadEnvSilently loads environment variables without logging anything
func loadEnvSilently() {
	// Try to find .env file in current directory
	envFile := ".env"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// Try to find .env in parent directory (project root)
		envFile = filepath.Join("..", ".env")
		if _, err := os.Stat(envFile); os.IsNotExist(err) {
			return
		}
	}

	// Load .env file silently without logging
	_ = godotenv.Load(envFile)
}

// configureLogLevelDirectly sets the global log level for all logrus instances
// and returns the configured level
func configureLogLevelDirectly() logrus.Level {
	// Get log level from environment variable
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "info" // Default log level
	}

	// Parse the log level
	logLevel, err := logrus.ParseLevel(strings.ToLower(logLevelStr))
	if err != nil {
		// Don't log here, just use default info level if we can't parse
		logLevel = logrus.InfoLevel
	}

	// This is critical: set the global logrus level BEFORE any logging happens
	// This affects ALL existing and future loggers
	logrus.SetLevel(logLevel)

	return logLevel
}

func main() {
	if err := root.Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Execute has returned, so PersistentPostRun has already saved the party
	// mappings; only now is it safe to honour a batch run's failure exit code.
	if code := root.ExitCode(); code != 0 {
		os.Exit(code)
	}
}
