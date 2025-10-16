// Package debit handles debit statement conversion commands
package debit

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/parser"

	"github.com/spf13/cobra"
)

// Cmd represents the debit command
var Cmd = &cobra.Command{
	Use:   "debit",
	Short: "Convert Debit CSV to CSV",
	Long:  `Convert Debit CSV statements to CSV format.`,
	Run:   debitFunc,
}

func debitFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Debit convert command called")
	root.Log.Infof("Input file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output file: %s", root.SharedFlags.Output)

	p, err := parser.GetParser(parser.Debit)
	if err != nil {
		root.Log.Fatalf("Error getting Debit parser: %v", err)
	}
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("Debit to CSV conversion completed successfully!")
}
