// Package batch handles batch conversion of multiple CAMT.053 XML files
package batch

import (
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/camtparser"

	"github.com/spf13/cobra"
)

// Cmd represents the batch command
var Cmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch convert multiple CAMT.053 XML files to CSV",
	Long:  `Batch convert all CAMT.053 XML files in a directory to CSV format.`,
	Run:   batchFunc,
}

func init() {
	// Batch command specific flags
	Cmd.Flags().StringVarP(&root.InputDir, "input-dir", "d", "", "Input directory containing files")
	Cmd.Flags().StringVarP(&root.OutputDir, "output-dir", "r", "", "Output directory for CSV files")
	Cmd.MarkFlagRequired("input-dir")
	Cmd.MarkFlagRequired("output-dir")
}

func batchFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Batch convert command called")
	root.Log.Infof("Input directory: %s", root.InputDir)
	root.Log.Infof("Output directory: %s", root.OutputDir)

	count, err := camtparser.BatchConvert(root.InputDir, root.OutputDir)
	if err != nil {
		root.Log.Fatalf("Error during batch conversion: %v", err)
	}
	root.Log.Infof("Successfully processed %d files", count)
}
