// Package debit handles Debit statement conversion commands
package debit

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the debit command
var Cmd = &cobra.Command{
	Use:   "debit",
	Short: "Convert Debit CSV to CSV",
	Long:  `Convert Debit CSV statements to CSV format.`,
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConvert(cmd, args, container.Debit, "Debit")
	},
}

func init() { common.RegisterFormatFlags(Cmd) }
