// Package selma handles Selma CSV file processing commands
package selma

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/selmaparser"

	"github.com/spf13/cobra"
)

// Cmd represents the selma command
var Cmd = &cobra.Command{
	Use:   "selma",
	Short: "Process Selma CSV files",
	Long:  `Process Selma CSV files to categorize and organize investment transactions.`,
	Run:   selmaFunc,
}

func selmaFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Selma CSV process command called")
	root.Log.Infof("Input Selma CSV file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output CSV file: %s", root.SharedFlags.Output)

	p := selmaparser.NewAdapter()
	if err := common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log); err != nil {
		root.Log.Fatalf("Error processing Selma CSV file: %v", err)
	}
	root.Log.Info("Selma CSV conversion completed successfully!")
}
