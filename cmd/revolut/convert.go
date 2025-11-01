// Package revolut handles Revolut statement conversion commands
package revolut

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut command
var Cmd = &cobra.Command{
	Use:   "revolut",
	Short: "Convert Revolut CSV to CSV",
	Long:  `Convert Revolut CSV statements to CSV format.`,
	Run:   revolutFunc,
}

func revolutFunc(cmd *cobra.Command, args []string) {
	logger := root.GetLogrusAdapter()
	root.Log.Info("Revolut convert command called")
	logger.Infof("Input file: %s", root.SharedFlags.Input)
	logger.Infof("Output file: %s", root.SharedFlags.Output)

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container
	p, err := appContainer.GetParser(container.Revolut)
	if err != nil {
		logger.Fatalf("Error getting Revolut parser: %v", err)
	}
	
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("Revolut to CSV conversion completed successfully!")
}
