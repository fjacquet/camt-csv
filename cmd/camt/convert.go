// Package camt handles CAMT file processing commands
package camt

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the camt command
var Cmd = &cobra.Command{
	Use:   "camt",
	Short: "Process CAMT.053 files",
	Long:  `Process CAMT.053 files to convert to CSV and categorize transactions.`,
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConvert(cmd, args, container.CAMT, "CAMT.053")
	},
}

func init() { common.RegisterFormatFlags(Cmd) }
