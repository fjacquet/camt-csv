// Package debit handles Visa Debit CSV file processing commands
package debit

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/debitparser"

	"github.com/spf13/cobra"
)

// Cmd represents the debit command
var Cmd = &cobra.Command{
	Use:   "debit",
	Short: "Process Visa Debit CSV files",
	Long:  `Process Visa Debit CSV files and convert to standard format.`,
	Run:   debitFunc,
}

func debitFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Debit command called")
	root.Log.Infof("Input file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output file: %s", root.SharedFlags.Output)

	p := debitparser.NewAdapter()
	if err := common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log); err != nil {
		root.Log.Fatalf("Error processing Debit CSV file: %v", err)
	}
	root.Log.Info("Debit CSV to standard CSV conversion completed successfully!")
}
