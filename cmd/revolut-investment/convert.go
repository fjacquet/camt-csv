// Package revolutinvestment handles Revolut Investment statement conversion commands
package revolutinvestment

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

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
	ctx := cmd.Context()
	logger := root.GetLogrusAdapter()
	root.Log.Info("Revolut Investment convert command called")
	logger.Infof("Input file: %s", root.SharedFlags.Input)
	logger.Infof("Output file: %s", root.SharedFlags.Output)

	// Get container from root command context
	appContainer := root.GetContainer()
	if appContainer == nil {
		logger.Fatal("Container not initialized")
	}

	// Get parser from container
	p, err := appContainer.GetParser(container.RevolutInvestment)
	if err != nil {
		logger.Fatalf("Error getting Revolut Investment parser: %v", err)
	}

	common.ProcessFile(ctx, p, root.SharedFlags.Input, root.SharedFlags.Output, root.SharedFlags.Validate, root.Log)
	root.Log.Info("Revolut Investment to CSV conversion completed successfully!")
}
