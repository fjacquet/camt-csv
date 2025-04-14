package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log = logrus.New()
	xmlFile string
	csvFile string
)

var rootCmd = &cobra.Command{
	Use:   "camt-csv",
	Short: "A CLI tool to convert CAMT.053 XML files to CSV and categorize transactions.",
	Long: `camt-csv is a CLI tool that converts CAMT.053 XML files to CSV format.
It also provides transaction categorization based on the seller's name.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		fmt.Println("Welcome to camt-csv!")
	},
}

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert CAMT.053 XML to CSV",
	Long:  `Convert CAMT.053 XML files to CSV format.`,
	Run: convertFunc,
}

func convertFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Convert command called")
	// Call the convertXMLToCSV function from converter.go
	err := convertXMLToCSV(xmlFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting XML to CSV: %v", err)
	}
	fmt.Println("XML to CSV conversion completed successfully!")
}

var categorizeCmd = &cobra.Command{
	Use:   "categorize",
	Short: "Categorize transactions using Gemini-2.0-fast model",
	Long:  `Categorize transactions based on the seller's name and typical activity using Gemini-2.0-fast model.`,
	Run: categorizeFunc,
}

func categorizeFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Categorize command called")
	// TODO: Implement Gemini-2.0-fast model integration here
}

func init() {
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(categorizeCmd)

	convertCmd.Flags().StringVarP(&xmlFile, "xml", "i", "", "Input XML file")
	convertCmd.Flags().StringVarP(&csvFile, "csv", "o", "", "Output CSV file")
	convertCmd.MarkFlagRequired("xml")
	convertCmd.MarkFlagRequired("csv")
}

func main() {
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
