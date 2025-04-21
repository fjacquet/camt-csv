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
	root.Log.Info("Converting CAMT file to CSV")
	root.Log.Debugf("Input file: %s", root.SharedFlags.Input)
	root.Log.Debugf("Output file: %s", root.SharedFlags.Output)

	p := camtparser.NewAdapter()
	if err := common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log); err != nil {
		root.Log.Fatalf("Error processing CAMT file: %v", err)
	}
	root.Log.Info("Conversion completed successfully")
	
	// Save any updated mappings to the database
	common.SaveCategoryMappings(root.Log)
}
