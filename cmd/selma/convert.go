// Package selma handles Selma statement conversion commands
package selma

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the selma command
var Cmd = &cobra.Command{
	Use:   "selma",
	Short: "Convert Selma CSV to CSV",
	Long:  `Convert Selma CSV statements to CSV format.`,
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConvert(cmd, args, container.Selma, "Selma")
	},
}

func init() { common.RegisterFormatFlags(Cmd) }
