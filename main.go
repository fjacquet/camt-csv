package main

import (
	"fmt"
	"os"

	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/pdfparser"
	"fjacquet/camt-csv/internal/selmaparser"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log       = logrus.New()
	xmlFile   string
	csvFile   string
	pdfFile   string
	inputDir  string
	outputDir string
	validate  bool
	partyName string
	isDebtor  bool
	amount    string
	date      string
	info      string
	selmaFile string
)

var rootCmd = &cobra.Command{
	Use:   "camt-csv",
	Short: "A CLI tool to convert CAMT.053 XML files to CSV and categorize transactions.",
	Long: `camt-csv is a CLI tool that converts CAMT.053 XML files to CSV format.
It also provides transaction categorization based on the party's name.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		log.Info("Welcome to camt-csv!")
		log.Info("Use --help to see available commands")
	},
	// Add a PersistentPostRun hook to save party mappings when ANY command finishes
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Save the creditor and debitor mappings back to disk after any command runs
		err := categorizer.SaveCreditorsToYAML()
		if err != nil {
			log.Warnf("Failed to save creditor mappings: %v", err)
		}
		
		err = categorizer.SaveDebitorsToYAML()
		if err != nil {
			log.Warnf("Failed to save debitor mappings: %v", err)
		}
	},
}

var camtCmd = &cobra.Command{
	Use:   "camt",
	Short: "Convert CAMT.053 XML to CSV",
	Long:  `Convert CAMT.053 XML files to CSV format.`,
	Run: convertFunc,
}

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch convert multiple CAMT.053 XML files to CSV",
	Long:  `Batch convert all CAMT.053 XML files in a directory to CSV format.`,
	Run: batchFunc,
}

var categorizeCmd = &cobra.Command{
	Use:   "categorize",
	Short: "Categorize transactions using Gemini-2.0-fast model",
	Long:  `Categorize transactions based on the party's name and typical activity using Gemini-2.0-fast model.`,
	Run: categorizeFunc,
}

var pdfCmd = &cobra.Command{
	Use:   "pdf",
	Short: "Convert PDF to CSV",
	Long:  `Convert PDF statements to CSV format.`,
	Run: pdfFunc,
}

var selmaCmd = &cobra.Command{
	Use:   "selma",
	Short: "Process Selma CSV files",
	Long:  `Process Selma CSV files to categorize and organize investment transactions.`,
	Run: selmaFunc,
}

func convertFunc(cmd *cobra.Command, args []string) {
	log.Info("Convert command called")
	log.Infof("Input file: %s", xmlFile)
	log.Infof("Output file: %s", csvFile)

	if validate {
		log.Info("Validating CAMT.053 format...")
		valid, err := camtparser.ValidateFormat(xmlFile)
		if err != nil {
			log.Fatalf("Error validating XML file: %v", err)
		}
		if !valid {
			log.Fatal("The file is not in valid CAMT.053 format")
		}
		log.Info("Validation successful. File is in valid CAMT.053 format.")
	}

	err := camtparser.ConvertToCSV(xmlFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting XML to CSV: %v", err)
	}
	log.Info("XML to CSV conversion completed successfully!")
}

func batchFunc(cmd *cobra.Command, args []string) {
	log.Info("Batch convert command called")
	log.Infof("Input directory: %s", inputDir)
	log.Infof("Output directory: %s", outputDir)

	count, err := camtparser.BatchConvert(inputDir, outputDir)
	if err != nil {
		log.Fatalf("Error during batch conversion: %v", err)
	}
	log.Infof("Batch conversion completed successfully! Converted %d files.", count)
}

func categorizeFunc(cmd *cobra.Command, args []string) {
	log.Info("Categorize command called")
	
	// Ensure the environment variables are loaded
	config.LoadEnv()
	
	// Example transaction to categorize (as a creditor/recipient)
	transaction := categorizer.Transaction{
		PartyName: "Coffee Shop",
		IsDebtor:  false,  // This is a creditor (recipient of payment)
		Amount:    "10.50 EUR",
		Date:      "2023-05-01",
		Info:      "Morning coffee",
	}
	
	// Categorize the transaction
	category, err := categorizer.CategorizeTransaction(transaction)
	if err != nil {
		log.Fatalf("Error categorizing transaction: %v", err)
	}
	
	log.Infof("Transaction categorized as: %s", category.Name)
}

func pdfFunc(cmd *cobra.Command, args []string) {
	log.Info("PDF convert command called")
	log.Infof("Input file: %s", pdfFile)
	log.Infof("Output file: %s", csvFile)

	if validate {
		log.Info("Validating PDF format...")
		valid, err := pdfparser.ValidateFormat(pdfFile)
		if err != nil {
			log.Fatalf("Error validating PDF file: %v", err)
		}
		if !valid {
			log.Fatal("The file is not a valid PDF")
		}
		log.Info("Validation successful. File is a valid PDF.")
	}

	err := pdfparser.ConvertToCSV(pdfFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting PDF to CSV: %v", err)
	}
	log.Info("PDF to CSV conversion completed successfully!")
}

func selmaFunc(cmd *cobra.Command, args []string) {
	log.Info("Selma CSV process command called")
	log.Infof("Input Selma CSV file: %s", selmaFile)
	log.Infof("Output CSV file: %s", csvFile)
	
	// Set the logger for the selma parser
	selmaparser.SetLogger(log)
	
	if validate {
		log.Info("Validating Selma CSV format...")
		valid, err := selmaparser.ValidateFormat(selmaFile)
		if err != nil {
			log.Fatalf("Error validating Selma CSV file: %v", err)
		}
		if !valid {
			log.Fatal("The file is not a valid Selma CSV")
		}
		log.Info("Validation successful. File is a valid Selma CSV.")
	}

	// Use the standardized ConvertToCSV method
	err := selmaparser.ConvertToCSV(selmaFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting Selma CSV to standard CSV: %v", err)
	}
	
	log.Info("Selma CSV processing completed successfully!")
}

func init() {
	// Initialize and configure logging
	config.LoadEnv()
	log = config.ConfigureLogging()
	
	// Set the configured logger for all parsers
	camtparser.SetLogger(log)
	pdfparser.SetLogger(log)
	selmaparser.SetLogger(log)
	
	rootCmd.AddCommand(camtCmd)
	rootCmd.AddCommand(batchCmd)
	rootCmd.AddCommand(categorizeCmd)
	rootCmd.AddCommand(pdfCmd)
	rootCmd.AddCommand(selmaCmd)

	camtCmd.Flags().StringVarP(&xmlFile, "xml", "i", "", "Input XML file")
	camtCmd.Flags().StringVarP(&csvFile, "csv", "o", "", "Output CSV file")
	camtCmd.MarkFlagRequired("xml")
	camtCmd.MarkFlagRequired("csv")
	
	batchCmd.Flags().StringVarP(&inputDir, "input", "i", "", "Input directory containing XML files")
	batchCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for CSV files")
	batchCmd.MarkFlagRequired("input")
	batchCmd.MarkFlagRequired("output")
	
	pdfCmd.Flags().StringVarP(&pdfFile, "pdf", "i", "", "Input PDF file")
	pdfCmd.Flags().StringVarP(&csvFile, "csv", "o", "", "Output CSV file")
	pdfCmd.MarkFlagRequired("pdf")
	pdfCmd.MarkFlagRequired("csv")
	
	selmaCmd.Flags().StringVarP(&selmaFile, "selma", "i", "", "Input Selma CSV file")
	selmaCmd.Flags().StringVarP(&csvFile, "csv", "o", "", "Output processed CSV file")
	selmaCmd.MarkFlagRequired("selma")
	selmaCmd.MarkFlagRequired("csv")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
