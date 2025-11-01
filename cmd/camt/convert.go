// Package camt handles CAMT file processing commands
package camt

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the camt command
var Cmd = &cobra.Command{
	Use:   "camt",
	Short: "Process CAMT.053 files",
	Long:  `Process CAMT.053 files to convert to CSV and categorize transactions.`,
	Run:   camtFunc,
}

func camtFunc(cmd *cobra.Command, args []string) {
	logger := root.GetLogrusAdapter()
	root.Log.Info("CAMT.053 process command called")
	logger.Infof("Input CAMT.053 file: %s", root.SharedFlags.Input)
	logger.Infof("Output CSV file: %s", root.SharedFlags.Output)

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container
	p, err := appContainer.GetParser(container.CAMT)
	if err != nil {
		logger.Fatalf("Error getting CAMT.053 parser: %v", err)
	}
	
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("CAMT.053 to CSV conversion completed successfully!")
}
