// Package common contains shared functionality for command handlers
package common

import "github.com/spf13/cobra"

// RegisterFormatFlags adds --format and --date-format flags to a command.
func RegisterFormatFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("format", "f", "standard",
		"Output format: standard (35-column CSV) or icompta (iCompta-compatible)")
	cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
}
