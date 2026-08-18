// Package viseca handles Viseca statement conversion commands
package viseca

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the viseca command
var Cmd = &cobra.Command{
	Use:   "viseca",
	Short: "Convert Viseca CSV to CSV",
	Long: `Convert the CSV transaction export from the Viseca One portal to CSV format.

Prefer this over the pdf command: the export carries the merchant name, the
foreign-currency detail and a stable transaction identifier that the PDF
statements do not.`,
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConvert(cmd, args, container.Viseca, "Viseca")
	},
}

func init() {
	common.RegisterFormatFlags(Cmd)

	// Read back by root.applyFlagOverrides; a viper.BindPFlag would be ignored
	// because InitializeConfig builds its own Viper instance.
	Cmd.Flags().Bool("keep-payments", false,
		"Import the monthly card settlement rows (dropped by default because the bank statement carries the same payments)")
}
