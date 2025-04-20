// Package camtparser provides functionality to parse CAMT.053 XML files and convert them to CSV format.
package camtparser

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/textutils"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// ISO20022Parser is a parser implementation for CAMT.053 files using ISO20022 standard definitions
type ISO20022Parser struct {
	log *logrus.Logger
}

// NewISO20022Parser creates a new ISO20022 parser for CAMT.053 files
func NewISO20022Parser(logger *logrus.Logger) *ISO20022Parser {
	return &ISO20022Parser{
		log: logger,
	}
}

// ParseFile parses a CAMT.053 XML file using ISO20022 standard format
func (p *ISO20022Parser) ParseFile(xmlFilePath string) ([]models.Transaction, error) {
	p.log.WithField("file", xmlFilePath).Info("Parsing CAMT.053 XML file (ISO20022 mode)")

	// Read XML file
	xmlBytes, err := os.ReadFile(xmlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML file: %w", err)
	}

	// Parse XML into ISO20022 document structure
	var document models.ISO20022Document
	if err := xml.Unmarshal(xmlBytes, &document); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	// Extract transactions from document
	transactions, err := p.extractTransactions(document)
	if err != nil {
		return nil, fmt.Errorf("failed to extract transactions: %w", err)
	}

	// Categorize transactions
	transactions = p.categorizeTransactions(transactions)

	p.log.WithField("count", len(transactions)).Info("Successfully extracted transactions from CAMT.053 file (ISO20022 mode)")
	return transactions, nil
}

// extractTransactions extracts transactions from an ISO20022 document
func (p *ISO20022Parser) extractTransactions(document models.ISO20022Document) ([]models.Transaction, error) {
	var transactions []models.Transaction

	// Process each statement in the document
	for _, stmt := range document.BkToCstmrStmt.Stmt {
		// Get IBAN from statement
		iban := ""
		if stmt.Acct.ID.IBAN != "" {
			iban = stmt.Acct.ID.IBAN
		}

		// Process each entry (transaction) in the statement
		for _, entry := range stmt.Ntry {
			// Convert entry to transaction model
			transaction := p.entryToTransaction(&entry)
			
			// Set IBAN if not already set in the entry
			if transaction.IBAN == "" {
				transaction.IBAN = iban
			}
			
			transactions = append(transactions, transaction)
		}
	}

	return transactions, nil
}

// entryToTransaction converts an ISO20022 Entry to the Transaction model
func (p *ISO20022Parser) entryToTransaction(entry *models.Entry) models.Transaction {
	amount, err := decimal.NewFromString(entry.Amt.Value)
	if err != nil {
		p.log.WithError(err).WithField("value", entry.Amt.Value).Warn("Failed to parse amount, using zero")
		amount = decimal.Zero
	}
	
	// Create transaction from entry
	tx := models.Transaction{
		BookkeepingNumber: textutils.ExtractBookkeepingNumber(entry.GetRemittanceInfo()),
		Status:           entry.Sts,
		Date:             entry.BookgDt.Dt,
		ValueDate:        entry.ValDt.Dt,
		Name:             "", // Will be set based on credit/debit direction
		Description:      entry.BuildDescription(),
		Amount:           amount,
		CreditDebit:      entry.GetCreditDebit(),
		Debit:            decimal.Zero, // Will be set below
		Credit:           decimal.Zero, // Will be set below
		Currency:         entry.Amt.Ccy,
		AmountExclTax:    amount, // Default to full amount
		AmountTax:        decimal.Zero,
		TaxRate:          decimal.Zero,
		Recipient:        entry.GetPayee(),
		Investment:       "", // Will be determined later if applicable
		Number:           "", // Not typically provided in CAMT.053
		Category:         "", // Will be determined later
		Type:             "", // Will be determined later
		Fund:             textutils.ExtractFund(entry.GetRemittanceInfo()),
		NumberOfShares:   0,
		Fees:             decimal.Zero,
		IBAN:             entry.GetIBAN(),
		EntryReference:   entry.GetReference(),
		Reference:        entry.GetReference(), // Using EntryReference as Reference for now
		AccountServicer:  entry.AcctSvcrRef,
		BankTxCode:       entry.GetBankTxCode(),
		OriginalCurrency: "",
		OriginalAmount:   decimal.Zero,
		ExchangeRate:     decimal.Zero,
        
		// Keep these for backward compatibility
		Payer:            entry.GetPayer(),
		Payee:            entry.GetPayee(),
	}
	
	// Set Debit/Credit amounts based on CreditDebit indicator
	if tx.CreditDebit == "DBIT" {
		tx.Debit = amount
		tx.Name = tx.Payee // For debits, the name is the payee/recipient
	} else {
		tx.Credit = amount
		tx.Name = tx.Payer // For credits, the name is the payer/sender
	}
	
	// Special case for DELL salary
	if strings.Contains(tx.Description, "VIRT BANC") && strings.Contains(tx.Description, "DELL SA") {
		tx.Category = "Salaire"
		p.log.WithField("description", tx.Description).Debug("Categorized as salary payment from DELL SA")
	}
	
	return tx
}

// categorizeTransactions applies categorization to all transactions
func (p *ISO20022Parser) categorizeTransactions(transactions []models.Transaction) []models.Transaction {
	for i := range transactions {
		// Skip transactions that already have a category (like the special case for DELL salary)
		if transactions[i].Category != "" {
			continue
		}

		// Create categorizer transaction from our transaction
		isDebtor := transactions[i].CreditDebit == "DBIT"
		catTx := categorizer.Transaction{
			PartyName:   transactions[i].GetPartyName(),
			IsDebtor:    isDebtor,
			Amount:      fmt.Sprintf("%s %s", transactions[i].Amount.String(), transactions[i].Currency),
			Date:        transactions[i].Date,
			Info:        transactions[i].Description,
			Description: transactions[i].Description,
		}

		// Try to categorize using the categorizer
		if category, err := categorizer.CategorizeTransaction(catTx); err == nil {
			transactions[i].Category = category.Name
			p.log.WithFields(logrus.Fields{
				"category": category.Name,
				"amount":   transactions[i].Amount.String(),
				"party":    catTx.PartyName,
			}).Debug("Transaction categorized")
		} else {
			p.log.WithFields(logrus.Fields{
				"amount": transactions[i].Amount.String(),
				"party":  catTx.PartyName,
			}).WithError(err).Debug("Failed to categorize transaction")
		}
	}

	return transactions
}

// ValidateFormat checks if the file is a valid CAMT.053 XML file
func (p *ISO20022Parser) ValidateFormat(filePath string) (bool, error) {
	p.log.WithField("file", filePath).Info("Validating CAMT.053 format")

	// Try to open and read the file
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("error opening file: %w", err)
	}
	defer xmlFile.Close()

	// Read the file content
	xmlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("error reading file: %w", err)
	}

	// Try to unmarshal the XML data into our ISO20022 document structure
	var document models.ISO20022Document
	if err := xml.Unmarshal(xmlBytes, &document); err != nil {
		p.log.WithField("file", filePath).Info("File is not a valid CAMT.053 XML")
		return false, nil
	}

	// Check if we have at least one statement
	if len(document.BkToCstmrStmt.Stmt) == 0 {
		p.log.WithField("file", filePath).Info("File is not a valid CAMT.053 XML (no statements)")
		return false, nil
	}

	p.log.WithField("file", filePath).Info("File is a valid CAMT.053 XML")
	return true, nil
}

// ConvertToCSV converts a CAMT.053 XML file to CSV format
func (p *ISO20022Parser) ConvertToCSV(inputFile, outputFile string) error {
	// Validate input file format
	isValid, err := p.ValidateFormat(inputFile)
	if err != nil {
		return fmt.Errorf("error validating file format: %w", err)
	}
	if !isValid {
		return fmt.Errorf("input file is not a valid CAMT.053 XML")
	}

	// Parse the XML file into transactions
	transactions, err := p.ParseFile(inputFile)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Write transactions to CSV file
	return common.WriteTransactionsToCSV(transactions, outputFile)
}

// WriteToCSV writes the transactions to a CSV file.
func (c *ISO20022Parser) WriteToCSV(transactions []models.Transaction, outputFile string) error {
	if transactions == nil || len(transactions) == 0 {
		// Create an empty CSV file with headers
		c.log.WithFields(logrus.Fields{
			"file": outputFile,
		}).Info("No transactions found, creating empty CSV file with headers")
		
		// Create a CSV file with headers only
		emptyFile, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer emptyFile.Close()
		
		// Create a new CSV writer and write headers directly
		csvWriter := csv.NewWriter(emptyFile)
		defer csvWriter.Flush()
		
		// Get header names from the Transaction struct using reflection
		headers := []string{
			"BookkeepingNumber", "Status", "Date", "ValueDate", "Name", "PartyName", "PartyIBAN",
			"Description", "RemittanceInfo", "Amount", "CreditDebit", "IsDebit", "Debit", "Credit",
			"Currency", "AmountExclTax", "AmountTax", "TaxRate", "Recipient", "InvestmentType",
			"Number", "Category", "Type", "Fund", "NumberOfShares", "Fees", "IBAN",
			"EntryReference", "Reference", "AccountServicer", "BankTxCode", "OriginalCurrency",
			"OriginalAmount", "ExchangeRate",
		}
		
		if err := csvWriter.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
		
		return nil
	}

	c.log.WithFields(logrus.Fields{
		"count": len(transactions),
		"file":  outputFile,
	}).Info("Writing transactions to CSV file")

	return common.ExportTransactionsToCSV(transactions, outputFile)
}

// SetLogger sets a custom logger for the parser
func (p *ISO20022Parser) SetLogger(logger *logrus.Logger) {
	if logger != nil {
		p.log = logger
	}
}
