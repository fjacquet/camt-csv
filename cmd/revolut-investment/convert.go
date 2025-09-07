// Package revolutinvestment handles Revolut investment CSV file processing commands
package revolutinvestment

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/revolutinvestmentparser"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut-investment command
var Cmd = &cobra.Command{
	Use:   "revolut-investment",
	Short: "Process Revolut investment CSV files",
	Long:  `Process Revolut investment CSV files to convert to standard format and categorize transactions.`,
	Run:   revolutInvestmentFunc,
}

func revolutInvestmentFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Revolut investment CSV process command called")
	root.Log.Infof("Input Revolut investment CSV file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output CSV file: %s", root.SharedFlags.Output)

	p := revolutinvestmentparser.NewAdapter()
	if err := common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log); err != nil {
		root.Log.Fatalf("Error processing Revolut investment CSV file: %v", err)
	}
	root.Log.Info("Revolut investment CSV conversion completed successfully!")
}
