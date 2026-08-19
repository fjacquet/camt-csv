package convert

import (
	"strings"

	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
)

// registerConvertFlags adds the conversion flags to a command.
func registerConvertFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("format", "f", "",
		"Output format: icompta (iCompta-compatible), standard (29-column comma-delimited CSV), or jumpsoft (7-column Jumpsoft Money CSV). Default: icompta (overridable via CAMT_OUTPUT_FORMAT env var)")
	cmd.Flags().String("from", "",
		"Input format, bypassing auto-detection: "+strings.Join(parserTypeNames(), ", ")+
			". On a directory this pins every file to that parser; files it cannot read fail individually")
	cmd.Flags().Bool("recursive", false,
		"When the input is a directory, also process files in its subdirectories")
	// Read back by root.applyFlagOverrides; a viper.BindPFlag would be ignored
	// because InitializeConfig builds its own Viper instance.
	cmd.Flags().Bool("keep-payments", false,
		"Import the monthly Viseca card settlement rows (dropped by default because the bank statement carries the same payments)")
	cmd.Flags().String("date-format", "DD.MM.YYYY",
		"Date format in output: DD.MM.YYYY, YYYY-MM-DD, MM/DD/YYYY, etc. (Go layout: 02.01.2006, 2006-01-02, 01/02/2006)")
	// The flag has never been wired to the writers: output dates are always
	// rendered with models.DateFormatCSV. Keep it registered so existing
	// invocations do not break, but tell the user it does nothing.
	if err := cmd.Flags().MarkDeprecated("date-format", "it has no effect; output dates are always DD.MM.YYYY"); err != nil {
		panic(err)
	}
}

// parserTypeNames lists the detectable formats for flag help and error
// messages. Shared so every place that needs the list — flag help text,
// the unknown-`--from` error, the "could not determine format" error, and the
// convert command's Long description — draws from the single source that
// backs DetectionOrder, rather than each keeping its own copy that can drift.
func parserTypeNames() []string {
	types := container.DetectionOrder()
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	return names
}
