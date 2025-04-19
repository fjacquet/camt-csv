package main

import (
	"fmt"
	"os"

	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/pdfparser"
	"fjacquet/camt-csv/internal/revolutparser"
	"fjacquet/camt-csv/internal/selmaparser"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log         = logrus.New()
	xmlFile     string
	csvFile     string
	pdfFile     string
	inputDir    string
	outputDir   string
	validate    bool
	partyName   string
	isDebtor    bool
	amount      string
	date        string
	info        string
	selmaFile   string
	revolutFile string
	debitFile   string
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

var revolutCmd = &cobra.Command{
	Use:   "revolut",
	Short: "Process Revolut CSV files",
	Long:  `Process Revolut CSV files to convert to standard format and categorize transactions.`,
	Run: revolutFunc,
}

var debitCmd = &cobra.Command{
	Use:   "debit",
	Short: "Process Debit CSV files",
	Long:  `Process Visa Debit CSV files to convert to standard format and categorize transactions.`,
	Run: debitFunc,
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
	
	log.Info("Selma CSV conversion completed successfully!")
}

func revolutFunc(cmd *cobra.Command, args []string) {
	log.Info("Revolut CSV process command called")
	log.Infof("Input Revolut CSV file: %s", revolutFile)
	log.Infof("Output CSV file: %s", csvFile)
	
	// Set the logger for the revolut parser
	revolutparser.SetLogger(log)
	
	if validate {
		log.Info("Validating Revolut CSV format...")
		valid, err := revolutparser.ValidateFormat(revolutFile)
		if err != nil {
			log.Fatalf("Error validating Revolut CSV file: %v", err)
		}
		if !valid {
			log.Fatal("The file is not a valid Revolut CSV")
		}
		log.Info("Validation successful. File is a valid Revolut CSV.")
	}

	// Use the standardized ConvertToCSV method
	err := revolutparser.ConvertToCSV(revolutFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting Revolut CSV to standard CSV: %v", err)
	}
	
	log.Info("Revolut CSV conversion completed successfully!")
}

func debitFunc(cmd *cobra.Command, args []string) {
	log.Info("Debit command called")
	log.Infof("Input file: %s", debitFile)
	log.Infof("Output file: %s", csvFile)

	// Set the logger for the debit parser
	debitparser.SetLogger(log)

	if validate {
		log.Info("Validating Debit CSV format...")
		valid, err := debitparser.ValidateFormat(debitFile)
		if err != nil {
			log.Fatalf("Error validating Debit CSV file: %v", err)
		}
		if !valid {
			log.Fatal("The file is not a valid Visa Debit CSV format")
		}
		log.Info("Validation successful. File is in valid Visa Debit CSV format.")
	}

	err := debitparser.ConvertToCSV(debitFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting Debit CSV to standard CSV: %v", err)
	}
	log.Info("Debit CSV to standard CSV conversion completed successfully!")
}

func init() {
	// Initialize and configure logging
	config.LoadEnv()
	log = config.ConfigureLogging()
	
	// Set the configured logger for all parsers
	camtparser.SetLogger(log)
	pdfparser.SetLogger(log)
	selmaparser.SetLogger(log)
	revolutparser.SetLogger(log)
	debitparser.SetLogger(log)
	
	// CAMT command flags
	camtCmd.Flags().StringVarP(&xmlFile, "input", "i", "", "Input CAMT.053 XML file")
	camtCmd.Flags().StringVarP(&csvFile, "output", "o", "", "Output CSV file")
	camtCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate XML file format before conversion")
	camtCmd.MarkFlagRequired("input")
	camtCmd.MarkFlagRequired("output")

	// Batch command flags
	batchCmd.Flags().StringVarP(&inputDir, "input-dir", "i", "", "Input directory containing CAMT.053 XML files")
	batchCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for CSV files")
	batchCmd.MarkFlagRequired("input-dir")
	batchCmd.MarkFlagRequired("output-dir")

	// PDF command flags
	pdfCmd.Flags().StringVarP(&pdfFile, "input", "i", "", "Input PDF file")
	pdfCmd.Flags().StringVarP(&csvFile, "output", "o", "", "Output CSV file")
	pdfCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate PDF file format before conversion")
	pdfCmd.MarkFlagRequired("input")
	pdfCmd.MarkFlagRequired("output")
	
	// Selma command flags
	selmaCmd.Flags().StringVarP(&selmaFile, "input", "i", "", "Input Selma CSV file")
	selmaCmd.Flags().StringVarP(&csvFile, "output", "o", "", "Output CSV file")
	selmaCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate Selma CSV file format before conversion")
	selmaCmd.MarkFlagRequired("input")
	selmaCmd.MarkFlagRequired("output")
	
	// Revolut command flags
	revolutCmd.Flags().StringVarP(&revolutFile, "input", "i", "", "Input Revolut CSV file")
	revolutCmd.Flags().StringVarP(&csvFile, "output", "o", "", "Output CSV file")
	revolutCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate Revolut CSV file format before conversion")
	revolutCmd.MarkFlagRequired("input")
	revolutCmd.MarkFlagRequired("output")

	// Debit command flags
	debitCmd.Flags().StringVarP(&debitFile, "input", "i", "", "Input Visa Debit CSV file")
	debitCmd.Flags().StringVarP(&csvFile, "output", "o", "", "Output CSV file")
	debitCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate Debit CSV file format before conversion")
	debitCmd.MarkFlagRequired("input")
	debitCmd.MarkFlagRequired("output")

	// Add commands to root
	rootCmd.AddCommand(camtCmd)
	rootCmd.AddCommand(batchCmd)
	rootCmd.AddCommand(categorizeCmd)
	rootCmd.AddCommand(pdfCmd)
	rootCmd.AddCommand(selmaCmd)
	rootCmd.AddCommand(revolutCmd)
	rootCmd.AddCommand(debitCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
