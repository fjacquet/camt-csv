// Package revolut handles Revolut statement conversion commands
package revolut

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut command
var Cmd = &cobra.Command{
	Use:   "revolut",
	Short: "Convert Revolut CSV to CSV",
	Long:  `Convert Revolut CSV statements to CSV format.`,
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConvert(cmd, args, container.Revolut, "Revolut")
	},
}

func init() { common.RegisterConvertFlags(Cmd) }
