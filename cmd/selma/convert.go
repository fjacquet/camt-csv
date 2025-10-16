// Package selma handles Selma statement conversion commands
package selma

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/parser"

	"github.com/spf13/cobra"
)

// Cmd represents the selma command
var Cmd = &cobra.Command{
	Use:   "selma",
	Short: "Convert Selma CSV to CSV",
	Long:  `Convert Selma CSV statements to CSV format.`,
	Run:   selmaFunc,
}

func selmaFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Selma convert command called")
	root.Log.Infof("Input file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output file: %s", root.SharedFlags.Output)

	p, err := parser.GetParser(parser.Selma)
	if err != nil {
		root.Log.Fatalf("Error getting Selma parser: %v", err)
	}
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("Selma to CSV conversion completed successfully!")
}
