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

func init() {
	Cmd.Flags().StringP("format", "f", "standard",
		"Output format: standard (35-column CSV) or icompta (iCompta-compatible)")
	Cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
}

func camtFunc(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()
	root.Log.Info("CAMT.053 process command called")
	logger.Infof("Input CAMT.053 file: %s", root.SharedFlags.Input)
	logger.Infof("Output CSV file: %s", root.SharedFlags.Output)

	// Get format flags
	format, _ := cmd.Flags().GetString("format")
	dateFormat, _ := cmd.Flags().GetString("date-format")

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

	common.ProcessFile(ctx, p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log, appContainer, format, dateFormat)
	root.Log.Info("CAMT.053 to CSV conversion completed successfully!")
}
