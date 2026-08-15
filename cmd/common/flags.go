// Package common contains shared functionality for command handlers
package common

import "github.com/spf13/cobra"

// RegisterFormatFlags adds --format and --date-format flags to a command.
func RegisterFormatFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("format", "f", "",
		"Output format: icompta (iCompta-compatible), standard (29-column comma-delimited CSV), or jumpsoft (7-column Jumpsoft Money CSV). Default: icompta (overridable via CAMT_OUTPUT_FORMAT env var)")
	cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
	// The flag has never been wired to the writers: output dates are always
	// rendered with models.DateFormatCSV. Keep it registered so existing
	// invocations do not break, but tell the user it does nothing.
	if err := cmd.Flags().MarkDeprecated("date-format", "it has no effect; output dates are always DD.MM.YYYY"); err != nil {
		panic(err)
	}
}
