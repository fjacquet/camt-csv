// Package selma handles Selma statement conversion commands
package selma

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the selma command
var Cmd = &cobra.Command{
	Use:   "selma",
	Short: "Convert Selma CSV to CSV",
	Long:  `Convert Selma CSV statements to CSV format.`,
	Run:   selmaFunc,
}

func init() {
	Cmd.Flags().StringP("format", "f", "standard",
		"Output format: standard (35-column CSV) or icompta (iCompta-compatible)")
	Cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
}

func selmaFunc(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()
	root.Log.Info("Selma convert command called")
	logger.Infof("Input file: %s", root.SharedFlags.Input)
	logger.Infof("Output file: %s", root.SharedFlags.Output)

	// Get format flags
	format, _ := cmd.Flags().GetString("format")
	dateFormat, _ := cmd.Flags().GetString("date-format")

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container
	p, err := appContainer.GetParser(container.Selma)
	if err != nil {
		logger.Fatalf("Error getting Selma parser: %v", err)
	}

	common.ProcessFile(ctx, p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log, appContainer, format, dateFormat)
	root.Log.Info("Selma to CSV conversion completed successfully!")
}
