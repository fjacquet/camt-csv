// Package revolutcrypto handles Revolut Crypto statement conversion commands.
package revolutcrypto

import (
	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// Cmd represents the revolut-crypto command.
var Cmd = &cobra.Command{
	Use:   "revolut-crypto",
	Short: "Convert Revolut Crypto CSV to CSV",
	Long:  `Convert Revolut Crypto account statement CSV files to CSV format.`,
	Run: func(cmd *cobra.Command, args []string) {
		common.RunConvert(cmd, args, container.RevolutCrypto, "Revolut Crypto")
	},
}

func init() { common.RegisterFormatFlags(Cmd) }
