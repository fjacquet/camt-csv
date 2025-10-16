// Package revolutinvestment handles Revolut Investment statement conversion commands
package revolutinvestment

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/parser"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut-investment command
var Cmd = &cobra.Command{
	Use:   "revolut-investment",
	Short: "Convert Revolut Investment CSV to CSV",
	Long:  `Convert Revolut Investment CSV statements to CSV format.`,
	Run:   revolutInvestmentFunc,
}

func revolutInvestmentFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Revolut Investment convert command called")
	root.Log.Infof("Input file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output file: %s", root.SharedFlags.Output)

	p, err := parser.GetParser(parser.RevolutInvestment)
	if err != nil {
		root.Log.Fatalf("Error getting Revolut Investment parser: %v", err)
	}
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("Revolut Investment to CSV conversion completed successfully!")
}
