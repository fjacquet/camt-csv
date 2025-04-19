package main

import (
	"fmt"
	"os"

	"fjacquet/camt-csv/cmd/batch"
	"fjacquet/camt-csv/cmd/camt"
	"fjacquet/camt-csv/cmd/categorize"
	"fjacquet/camt-csv/cmd/debit"
	"fjacquet/camt-csv/cmd/pdf"
	"fjacquet/camt-csv/cmd/revolut"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/cmd/selma"
)

func init() {
	// Initialize root command
	root.Init()
	
	// Add all subcommands to the root command
	root.Cmd.AddCommand(camt.Cmd)
	root.Cmd.AddCommand(batch.Cmd)
	root.Cmd.AddCommand(categorize.Cmd)
	root.Cmd.AddCommand(pdf.Cmd)
	root.Cmd.AddCommand(selma.Cmd)
	root.Cmd.AddCommand(revolut.Cmd)
	root.Cmd.AddCommand(debit.Cmd)
}

func main() {
	if err := root.Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
