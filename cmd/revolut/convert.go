// Package revolut handles Revolut CSV file processing commands
package revolut

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/revolutparser"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut command
var Cmd = &cobra.Command{
	Use:   "revolut",
	Short: "Process Revolut CSV files",
	Long:  `Process Revolut CSV files to convert to standard format and categorize transactions.`,
	Run:   revolutFunc,
}

func revolutFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Revolut CSV process command called")
	root.Log.Infof("Input Revolut CSV file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output CSV file: %s", root.SharedFlags.Output)

	p := revolutparser.NewAdapter()
	if err := common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log); err != nil {
		root.Log.Fatalf("Error processing Revolut CSV file: %v", err)
	}
	root.Log.Info("Revolut CSV conversion completed successfully!")
}
