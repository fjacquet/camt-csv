// Package pdf handles PDF statement conversion commands
package pdf

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/pdfparser"

	"github.com/spf13/cobra"
)

// Cmd represents the pdf command
var Cmd = &cobra.Command{
	Use:   "pdf",
	Short: "Convert PDF to CSV",
	Long:  `Convert PDF statements to CSV format.`,
	Run:   pdfFunc,
}

func pdfFunc(cmd *cobra.Command, args []string) {
	root.Log.Info("PDF convert command called")
	root.Log.Infof("Input file: %s", root.SharedFlags.Input)
	root.Log.Infof("Output file: %s", root.SharedFlags.Output)

	p := pdfparser.NewAdapter()
	if err := common.ProcessFile(p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log); err != nil {
		root.Log.Fatalf("Error processing PDF file: %v", err)
	}
	root.Log.Info("PDF to CSV conversion completed successfully!")
}
