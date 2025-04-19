// Package camt handles CAMT.053 XML file conversion commands
package camt

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/camtparser"

	"github.com/spf13/cobra"
)

// Cmd represents the camt command
var Cmd = &cobra.Command{
	Use:   "camt",
	Short: "Convert CAMT.053 XML to CSV",
	Long:  `Convert CAMT.053 XML files to CSV format.`,
	Run:   convertFunc,
}

func convertFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Convert command called")
	root.Log.Infof("Input file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output file: %s", root.SharedFlags.Output)

	p := camtparser.NewAdapter()
	if err := common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log); err != nil {
		root.Log.Fatalf("Error processing CAMT file: %v", err)
	}
	root.Log.Info("XML to CSV conversion completed successfully!")
}
