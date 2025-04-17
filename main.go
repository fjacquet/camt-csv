package main

import (
	"fmt"
	"os"

	"fjacquet/camt-csv/pkg/categorizer"
	"fjacquet/camt-csv/pkg/converter"
	"fjacquet/camt-csv/pkg/config"
	"fjacquet/camt-csv/pkg/pdfparser"
	"fjacquet/camt-csv/pkg/paprika"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log = logrus.New()
	xmlFile string
	csvFile string
	inputDir string
	outputDir string
	pdfFile string
	paprikaFile string
)

var rootCmd = &cobra.Command{
	Use:   "camt-csv",
	Short: "A CLI tool to convert CAMT.053 XML files to CSV and categorize transactions.",
	Long: `camt-csv is a CLI tool that converts CAMT.053 XML files to CSV format.
It also provides transaction categorization based on the party's name.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
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

var convertCmd = &cobra.Command{
	Use:   "convert",
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

func convertFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Convert command called")
	// Call the convertXMLToCSV function from converter package
	err := converter.ConvertXMLToCSV(xmlFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting XML to CSV: %v", err)
	}
	fmt.Println("XML to CSV conversion completed successfully!")
}

func batchFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Batch convert command called")
	fmt.Printf("Input directory: %s\n", inputDir)
	fmt.Printf("Output directory: %s\n", outputDir)
	
	count, err := converter.BatchConvert(inputDir, outputDir)
	if err != nil {
		log.Fatalf("Error during batch conversion: %v", err)
	}
	fmt.Printf("Batch conversion completed successfully! Converted %d files.\n", count)
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

var exportPaprikaCmd = &cobra.Command{
	Use:   "export-paprika",
	Short: "Export CSV to Paprika format",
	Long:  `Export transaction data from CSV file to Paprika-compatible format for import into financial applications.`,
	Run: exportPaprikaFunc,
}

var importPaprikaCmd = &cobra.Command{
	Use:   "import-paprika",
	Short: "Import from Paprika format to CSV",
	Long:  `Import transaction data from Paprika-compatible format into CSV for use with this application.`,
	Run: importPaprikaFunc,
}

func pdfFunc(cmd *cobra.Command, args []string) {
	fmt.Println("PDF convert command called")
	fmt.Printf("Input PDF file: %s\n", pdfFile)
	fmt.Printf("Output CSV file: %s\n", csvFile)
	
	// Validate PDF file
	isValid, err := pdfparser.ValidatePDF(pdfFile)
	if err != nil {
		log.Fatalf("Error validating PDF file: %v", err)
	}
	if !isValid {
		log.Fatalf("Invalid PDF file: %s", pdfFile)
	}
	
	// Call the ConvertPDFToCSV function from pdfparser package
	err = pdfparser.ConvertPDFToCSV(pdfFile, csvFile)
	if err != nil {
		log.Fatalf("Error converting PDF to CSV: %v", err)
	}
	fmt.Println("PDF to CSV conversion completed successfully!")
}

func categorizeFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Categorize command called")
	
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
	
	fmt.Printf("Transaction categorized as: %s\n", category.Name)
}

func exportPaprikaFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Export Paprika command called")
	fmt.Printf("Input CSV file: %s\n", csvFile)
	fmt.Printf("Output Paprika file: %s\n", paprikaFile)
	
	// Call the ExportToPaprika function from paprika package
	err := paprika.ExportToPaprika(csvFile, paprikaFile)
	if err != nil {
		log.Fatalf("Error exporting to Paprika: %v", err)
	}
	fmt.Println("Export to Paprika completed successfully!")
}

func importPaprikaFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Import Paprika command called")
	fmt.Printf("Input Paprika file: %s\n", paprikaFile)
	fmt.Printf("Output CSV file: %s\n", csvFile)
	
	// Call the ImportFromPaprika function from paprika package
	err := paprika.ImportFromPaprika(paprikaFile, csvFile)
	if err != nil {
		log.Fatalf("Error importing from Paprika: %v", err)
	}
	fmt.Println("Import from Paprika completed successfully!")
}

func init() {
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(batchCmd)
	rootCmd.AddCommand(categorizeCmd)
	rootCmd.AddCommand(pdfCmd)
	rootCmd.AddCommand(exportPaprikaCmd)
	rootCmd.AddCommand(importPaprikaCmd)

	convertCmd.Flags().StringVarP(&xmlFile, "xml", "i", "", "Input XML file")
	convertCmd.Flags().StringVarP(&csvFile, "csv", "o", "", "Output CSV file")
	convertCmd.MarkFlagRequired("xml")
	convertCmd.MarkFlagRequired("csv")
	
	batchCmd.Flags().StringVarP(&inputDir, "input", "i", "", "Input directory containing XML files")
	batchCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for CSV files")
	batchCmd.MarkFlagRequired("input")
	batchCmd.MarkFlagRequired("output")
	
	pdfCmd.Flags().StringVarP(&pdfFile, "pdf", "i", "", "Input PDF file")
	pdfCmd.Flags().StringVarP(&csvFile, "csv", "o", "", "Output CSV file")
	pdfCmd.MarkFlagRequired("pdf")
	pdfCmd.MarkFlagRequired("csv")
	
	exportPaprikaCmd.Flags().StringVarP(&csvFile, "csv", "i", "", "Input CSV file")
	exportPaprikaCmd.Flags().StringVarP(&paprikaFile, "paprika", "o", "", "Output Paprika file")
	exportPaprikaCmd.MarkFlagRequired("csv")
	exportPaprikaCmd.MarkFlagRequired("paprika")
	
	importPaprikaCmd.Flags().StringVarP(&paprikaFile, "paprika", "i", "", "Input Paprika file")
	importPaprikaCmd.Flags().StringVarP(&csvFile, "csv", "o", "", "Output CSV file")
	importPaprikaCmd.MarkFlagRequired("paprika")
	importPaprikaCmd.MarkFlagRequired("csv")
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
