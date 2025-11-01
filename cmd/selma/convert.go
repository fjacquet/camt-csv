// Package selma handles Selma statement conversion commands
package selma

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the selma command
var Cmd = &cobra.Command{
	Use:   "selma",
	Short: "Convert Selma CSV to CSV",
	Long:  `Convert Selma CSV statements to CSV format.`,
	Run:   selmaFunc,
}

func selmaFunc(cmd *cobra.Command, args []string) {
	logger := root.GetLogrusAdapter()
	root.Log.Info("Selma convert command called")
	logger.Infof("Input file: %s", root.SharedFlags.Input)
	logger.Infof("Output file: %s", root.SharedFlags.Output)

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container
	p, err := appContainer.GetParser(container.Selma)
	if err != nil {
		logger.Fatalf("Error getting Selma parser: %v", err)
	}
	
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("Selma to CSV conversion completed successfully!")
}
