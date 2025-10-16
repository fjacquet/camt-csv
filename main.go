package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/cmd/batch"
	"fjacquet/camt-csv/cmd/camt"
	"fjacquet/camt-csv/cmd/categorize"
	"fjacquet/camt-csv/cmd/debit"
	"fjacquet/camt-csv/cmd/pdf"
	"fjacquet/camt-csv/cmd/revolut"
	revolutinvestment "fjacquet/camt-csv/cmd/revolut-investment"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/cmd/selma"
	"fjacquet/camt-csv/internal/logging"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func init() {
	// 1. Load environment variables silently first (no logging yet)
	loadEnvSilently()

	// 2. Configure global log level directly - this affects ALL new loggers
	logLevel := configureLogLevelDirectly()

	// 3. Force this level on ALL existing and future loggers
	logging.SetAllLogLevels(logLevel)

	// 4. Now that logging is properly configured, initialize root command
	root.Init()

	// 5. Add all subcommands
	root.Cmd.AddCommand(camt.Cmd)
	root.Cmd.AddCommand(batch.Cmd)
	root.Cmd.AddCommand(categorize.Cmd)
	root.Cmd.AddCommand(pdf.Cmd)
	root.Cmd.AddCommand(selma.Cmd)
	root.Cmd.AddCommand(revolut.Cmd)
	root.Cmd.AddCommand(debit.Cmd)
	root.Cmd.AddCommand(revolutinvestment.Cmd)
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
}
