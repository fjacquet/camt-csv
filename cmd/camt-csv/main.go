// Package main provides the entry point for the camt-csv CLI application.
package main

import (
	"fmt"

	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/pdfparser"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log       = logrus.New()
	xmlFile   string
	csvFile   string
	pdfFile   string
	debitFile string
	inputDir  string
	outputDir string
	validate  bool
	partyName string
	isDebtor  bool
	amount    string
	date      string
	info      string
)

var rootCmd = &cobra.Command{
	Use:   "camt-csv",
	Short: "A CLI tool to convert CAMT.053 XML files to CSV and categorize transactions.",
	Long: `camt-csv is a CLI tool that converts CAMT.053 XML files to CSV format.
It also provides transaction categorization based on the party's name using Gemini-2.0-fast model.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to camt-csv!")
		fmt.Println("Use --help to see available commands")
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
	Run:   convertFunc,
}

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch convert multiple CAMT.053 XML files to CSV",
	Long:  `Batch convert all CAMT.053 XML files in a directory to CSV format.`,
	Run:   batchFunc,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate if an XML file is in CAMT.053 format",
	Long:  `Validate if an XML file follows the CAMT.053 format structure.`,
	Run:   validateFunc,
}

var categorizeCmd = &cobra.Command{
	Use:   "categorize",
	Short: "Categorize transactions using Gemini-2.0-fast model",
	Long:  `Categorize transactions based on the party's name and typical activity using Gemini-2.0-fast model.`,
	Run:   categorizeFunc,
}

var pdfCmd = &cobra.Command{
	Use:   "pdf",
	Short: "Convert PDF to CSV",
	Long:  `Convert PDF file containing transaction data to CSV format.`,
	Run:   pdfFunc,
}

var debitCmd = &cobra.Command{
	Use:   "debit",
	Short: "Convert Visa Debit CSV to standard CSV",
	Long:  `Convert Visa Debit CSV files to standard CSV format with categorization.`,
	Run:   debitFunc,
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
			log.Fatal("The XML file is not in valid CAMT.053 format")
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

func validateFunc(cmd *cobra.Command, args []string) {
	log.Info("Validate command called")
	log.Infof("Input file: %s", xmlFile)

	valid, err := camtparser.ValidateFormat(xmlFile)
	if err != nil {
		log.Fatalf("Error validating XML file: %v", err)
	}

	if valid {
		log.Info("The XML file is in valid CAMT.053 format")
	} else {
		log.Info("The XML file is NOT in valid CAMT.053 format")
	}
}

func categorizeFunc(cmd *cobra.Command, args []string) {
	log.Info("Categorize command called")

	if partyName == "" {
		log.Fatal("Party name is required")
	}

	// Load environment variables to ensure API keys are available
	config.LoadEnv()

	transaction := categorizer.Transaction{
		PartyName: partyName,
		IsDebtor:  isDebtor,
		Amount:    amount,
		Date:      date,
		Info:      info,
	}

	log.Infof("Categorizing transaction for party: %s (IsDebtor: %t)", partyName, isDebtor)
	category, err := categorizer.CategorizeTransaction(transaction)
	if err != nil {
		log.Warnf("Error categorizing transaction: %v", err)
		log.Info("Using default category due to error")
		category = models.Category{
			Name:        "Uncategorized",
			Description: "Could not categorize due to an error",
		}
	}

	log.Infof("Transaction categorized as: %s", category.Name)
	log.Infof("Description: %s", category.Description)
}

func pdfFunc(cmd *cobra.Command, args []string) {
	log.Info("PDF convert command called")
	log.Infof("Input file: %s", pdfFile)
	log.Infof("Output file: %s", csvFile)

	// Set the logger for the pdf parser
	pdfparser.SetLogger(log)

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

func debitFunc(cmd *cobra.Command, args []string) {
	log.Info("Debit convert command called")
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
	// Configure logging
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Add commands to root command
	rootCmd.AddCommand(camtCmd)
	rootCmd.AddCommand(batchCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(categorizeCmd)
	rootCmd.AddCommand(pdfCmd)
	rootCmd.AddCommand(debitCmd)

	// Define flags for convert command
	camtCmd.Flags().StringVarP(&xmlFile, "xml", "x", "", "Input XML file (required)")
	camtCmd.Flags().StringVarP(&csvFile, "csv", "c", "", "Output CSV file (required)")
	camtCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate XML format before conversion")
	if err := camtCmd.MarkFlagRequired("xml"); err != nil {
		panic(err)
	}
	if err := camtCmd.MarkFlagRequired("csv"); err != nil {
		panic(err)
	}

	// Define flags for batch command
	batchCmd.Flags().StringVarP(&inputDir, "input", "i", "", "Input directory containing XML files (required)")
	batchCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for CSV files (required)")
	if err := batchCmd.MarkFlagRequired("input"); err != nil {
		panic(err)
	}
	if err := batchCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	// Define flags for validate command
	validateCmd.Flags().StringVarP(&xmlFile, "xml", "x", "", "XML file to validate (required)")
	if err := validateCmd.MarkFlagRequired("xml"); err != nil {
		panic(err)
	}

	// Define flags for categorize command
	categorizeCmd.Flags().StringVarP(&partyName, "party", "p", "", "Name of the party (required)")
	categorizeCmd.Flags().BoolVarP(&isDebtor, "debtor", "d", false, "Whether the party is a debtor (sender) or creditor (recipient)")
	categorizeCmd.Flags().StringVarP(&amount, "amount", "a", "", "Transaction amount")
	categorizeCmd.Flags().StringVar(&date, "date", "", "Transaction date")
	categorizeCmd.Flags().StringVar(&info, "info", "", "Additional transaction information")
	if err := categorizeCmd.MarkFlagRequired("party"); err != nil {
		panic(err)
	}

	// Define flags for pdf command
	pdfCmd.Flags().StringVarP(&pdfFile, "pdf", "p", "", "Input PDF file (required)")
	pdfCmd.Flags().StringVarP(&csvFile, "csv", "c", "", "Output CSV file (required)")
	pdfCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate PDF format before conversion")
	if err := pdfCmd.MarkFlagRequired("pdf"); err != nil {
		panic(err)
	}
	if err := pdfCmd.MarkFlagRequired("csv"); err != nil {
		panic(err)
	}

	// Define flags for debit command
	debitCmd.Flags().StringVarP(&debitFile, "in", "i", "", "Input Visa Debit CSV file (required)")
	debitCmd.Flags().StringVarP(&csvFile, "out", "o", "", "Output CSV file (required)")
	debitCmd.Flags().BoolVarP(&validate, "validate", "v", false, "Validate Debit CSV format before conversion")
	if err := debitCmd.MarkFlagRequired("in"); err != nil {
		panic(err)
	}
	if err := debitCmd.MarkFlagRequired("out"); err != nil {
		panic(err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
