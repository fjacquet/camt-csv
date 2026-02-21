// Package revolutinvestment handles RevolutInvestment statement conversion commands
package revolutinvestment

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut-investment command
var Cmd = &cobra.Command{
	Use:   "revolut-investment",
	Short: "Convert Revolut Investment CSV to CSV",
	Long:  `Convert Revolut Investment CSV statements to CSV format.`,
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConvert(cmd, args, container.RevolutInvestment, "Revolut Investment")
	},
}

func init() { common.RegisterFormatFlags(Cmd) }
