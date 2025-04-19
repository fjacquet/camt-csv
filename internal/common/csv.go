// Package common provides shared functionality across different parsers.
package common

import (
	"fmt"
	"os"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/models"

	"github.com/gocarina/gocsv"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// SetLogger allows setting a configured logger
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
	}
}

// WriteTransactionsToCSV is a generalized function to write transactions to CSV 
// with categorization. All parsers can use this function.
func WriteTransactionsToCSV(transactions []models.Transaction, csvFile string) error {
	log.WithField("file", csvFile).Info("Writing transactions to CSV file")

	// Process and categorize transactions
	for i := range transactions {
		// Skip categorization if already categorized
		if transactions[i].Category != "" {
			continue
		}

		// Determine if transaction is for debtor or creditor based on CreditDebit
		isDebtor := transactions[i].CreditDebit == "DBIT"
		
		// Use Payee if available, otherwise use Description
		partyName := transactions[i].Payee
		if partyName == "" {
			partyName = transactions[i].Description
		}
		
		// Create categorizer transaction
		catTx := categorizer.Transaction{
			PartyName: partyName,
			IsDebtor:  isDebtor,
			Amount:    transactions[i].Amount,
			Date:      transactions[i].Date,
			Info:      transactions[i].Description,
		}
		
		// Get category
		category, err := categorizer.CategorizeTransaction(catTx)
		if err != nil {
			log.WithError(err).WithField("party", partyName).Warning("Failed to categorize transaction")
		} else {
			// Store the category in the transaction
			transactions[i].Category = category.Name
			log.WithFields(logrus.Fields{
				"party":    partyName,
				"category": category.Name,
			}).Debug("Transaction categorized")
		}
	}

	// Create output file
	file, err := os.Create(csvFile)
	if err != nil {
		log.WithError(err).Error("Failed to create output CSV file")
		return fmt.Errorf("error creating output CSV file: %w", err)
	}
	defer file.Close()

	// Use gocsv to marshal and write in one step
	if err := gocsv.MarshalFile(&transactions, file); err != nil {
		log.WithError(err).Error("Failed to write CSV")
		return fmt.Errorf("error writing CSV: %w", err)
	}

	log.WithField("count", len(transactions)).Info("Successfully wrote transactions to CSV")
	return nil
}

// GeneralizedConvertToCSV is a utility function that combines parsing and writing to CSV
// This can be used by any parser that follows the standard interface
func GeneralizedConvertToCSV(
	inputFile string, 
	outputFile string,
	parseFunc func(string) ([]models.Transaction, error),
	validateFunc func(string) (bool, error),
) error {
	// Validate format if validateFunc is provided
	if validateFunc != nil {
		valid, err := validateFunc(inputFile)
		if err != nil {
			return fmt.Errorf("error validating file format: %w", err)
		}
		if !valid {
			return fmt.Errorf("input file is not in a valid format")
		}
	}

	// Parse the file
	transactions, err := parseFunc(inputFile)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// Write to CSV
	err = WriteTransactionsToCSV(transactions, outputFile)
	if err != nil {
		return fmt.Errorf("error writing to CSV: %w", err)
	}

	log.Info("Conversion completed successfully")
	return nil
}
