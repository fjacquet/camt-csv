// Package common contains shared functionality for command handlers
package common

import "github.com/spf13/cobra"

// RegisterFormatFlags adds --format and --date-format flags to a command.
func RegisterFormatFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("format", "f", "",
		"Output format: icompta (iCompta-compatible), standard (29-column comma-delimited CSV), or jumpsoft (7-column Jumpsoft Money CSV). Default: icompta (overridable via CAMT_OUTPUT_FORMAT env var)")
	cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
}
