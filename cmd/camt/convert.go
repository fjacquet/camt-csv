// Package camt handles CAMT file processing commands
package camt

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/parser"

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
	root.Log.Info("CAMT.053 process command called")
	root.Log.Infof("Input CAMT.053 file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output CSV file: %s", root.SharedFlags.Output)

	p, err := parser.GetParser(parser.CAMT)
	if err != nil {
		root.Log.Fatalf("Error getting CAMT.053 parser: %v", err)
	}
	common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("CAMT.053 to CSV conversion completed successfully!")
}
