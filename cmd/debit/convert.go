// Package debit handles debit statement conversion commands
package debit

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/factory"

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
	logger := root.GetLogrusAdapter()
	root.Log.Info("Debit convert command called")
	logger.Infof("Input file: %s", root.SharedFlags.Input)
	logger.Infof("Output file: %s", root.SharedFlags.Output)

	p, err := factory.GetParser(factory.Debit)
	if err != nil {
		logger.Fatalf("Error getting Debit parser: %v", err)
	}
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("Debit to CSV conversion completed successfully!")
}
