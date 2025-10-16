// Package batch handles batch processing of files
package batch

import (
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/camtparser"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Cmd represents the batch command
var Cmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch process files from a directory",
	Long:  `Batch process files from an input directory and output them to another directory.`,
	Run:   batchFunc,
}

func batchFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("Batch command called")
	root.Log.Infof("Input directory: %s", root.InputDir)
	root.Log.Infof("Output directory: %s", root.OutputDir)

	if root.InputDir == "" || root.OutputDir == "" {
		root.Log.Fatal("Input and output directories must be specified")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(root.OutputDir, 0750); err != nil {
		root.Log.Fatalf("Failed to create output directory: %v", err)
	}

	adapter := camtparser.NewAdapter()
	count, err := adapter.BatchConvert(root.InputDir, root.OutputDir)
	if err != nil {
		root.Log.Fatalf("Error during batch conversion: %v", err)
	}

	root.Log.Info(fmt.Sprintf("Batch processing completed. %d files converted.", count))
}
