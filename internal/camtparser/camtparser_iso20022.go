// Package camtparser provides functionality to parse CAMT.053 XML files and convert them to CSV format.
package camtparser

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/parsererror"
	"fjacquet/camt-csv/internal/textutils"

	"github.com/shopspring/decimal"
)

// ISO20022Parser is a parser implementation for CAMT.053 files using ISO20022 standard definitions
type ISO20022Parser struct {
	parser.BaseParser
}

// NewISO20022Parser creates a new ISO20022 parser for CAMT.053 files
func NewISO20022Parser(logger logging.Logger) *ISO20022Parser {
	return &ISO20022Parser{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// ParseFile parses a CAMT.053 XML file using ISO20022 standard format
func (p *ISO20022Parser) ParseFile(xmlFilePath string) ([]models.Transaction, error) {
	p.GetLogger().Info("Parsing CAMT.053 XML file (ISO20022 mode)",
		logging.Field{Key: "file", Value: xmlFilePath})

	// Read XML file
	xmlBytes, err := os.ReadFile(xmlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML file: %w", err)
	}

	// Parse XML into ISO20022 document structure
	var document models.ISO20022Document
	if err := xml.Unmarshal(xmlBytes, &document); err != nil {
		return nil, &parsererror.ParseError{
			Parser: "CAMT",
			Field:  "XML document",
			Value:  xmlFilePath,
			Err:    fmt.Errorf("failed to unmarshal XML: %w", err),
		}
	}

	// Extract transactions from document
	transactions, err := p.extractTransactions(document)
	if err != nil {
		return nil, fmt.Errorf("failed to extract transactions: %w", err)
	}

	// Categorize transactions
	transactions = p.categorizeTransactions(transactions)

	p.GetLogger().Info("Successfully extracted transactions from CAMT.053 file (ISO20022 mode)",
		logging.Field{Key: "count", Value: len(transactions)})
	return transactions, nil
}

// extractTransactions extracts transactions from an ISO20022 document
func (p *ISO20022Parser) extractTransactions(document models.ISO20022Document) ([]models.Transaction, error) {
	// Pre-allocate slice with estimated capacity based on total entries
	totalEntries := 0
	for _, stmt := range document.BkToCstmrStmt.Stmt {
		totalEntries += len(stmt.Ntry)
	}
	transactions := make([]models.Transaction, 0, totalEntries)

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

// entryToTransaction converts an ISO20022 Entry to the Transaction model using TransactionBuilder
func (p *ISO20022Parser) entryToTransaction(entry *models.Entry) models.Transaction {
	amount, err := decimal.NewFromString(entry.Amt.Value)
	if err != nil {
		p.GetLogger().WithError(err).Warn("Failed to parse amount, using zero",
			logging.Field{Key: "value", Value: entry.Amt.Value})
		amount = decimal.Zero
	}

	// Use TransactionBuilder for consistent transaction construction
	builder := models.NewTransactionBuilder().
		WithBookkeepingNumber(textutils.ExtractBookkeepingNumber(entry.GetRemittanceInfo())).
		WithStatus(entry.Sts).
		WithDate(entry.BookgDt.Dt).
		WithValueDate(entry.ValDt.Dt).
		WithDescription(entry.BuildDescription()).
		WithAmount(amount, entry.Amt.Ccy).
		WithPayer(entry.GetPayer(), entry.GetIBAN()).
		WithPayee(entry.GetPayee(), entry.GetIBAN()).
		WithReference(entry.GetReference()).
		WithEntryReference(entry.GetReference()).
		WithAccountServicer(entry.AcctSvcrRef).
		WithBankTxCode(entry.GetBankTxCode()).
		WithFund(textutils.ExtractFund(entry.GetRemittanceInfo()))

	// Set transaction direction based on credit/debit indicator
	if entry.GetCreditDebit() == models.TransactionTypeDebit {
		builder = builder.AsDebit()
	} else {
		builder = builder.AsCredit()
	}

	// Build the transaction
	tx, err := builder.Build()
	if err != nil {
		p.GetLogger().WithError(err).Warn("Failed to build transaction, using fallback",
			logging.Field{Key: "entry_reference", Value: entry.GetReference()})
		// Return a minimal transaction as fallback
		fallback, _ := models.NewTransactionBuilder().
			WithDate(entry.BookgDt.Dt).
			WithAmount(amount, entry.Amt.Ccy).
			WithDescription("Failed to parse transaction").
			Build()
		return fallback
	}

	// Special case for DELL salary
	if strings.Contains(tx.Description, "VIRT BANC") && strings.Contains(tx.Description, "DELL SA") {
		tx.Category = models.CategorySalary
		p.GetLogger().Debug("Categorized as salary payment from DELL SA",
			logging.Field{Key: "description", Value: tx.Description})
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
		isDebtor := transactions[i].CreditDebit == models.TransactionTypeDebit
		catTx := categorizer.Transaction{
			PartyName:   transactions[i].GetPartyName(),
			IsDebtor:    isDebtor,
			Amount:      fmt.Sprintf("%s %s", transactions[i].Amount.String(), transactions[i].Currency),
			Date:        transactions[i].Date.Format("02.01.2006"),
			Info:        transactions[i].Description,
			Description: transactions[i].Description,
		}

		p.GetLogger().Debug("About to categorize transaction",
			logging.Field{Key: "party", Value: catTx.PartyName},
			logging.Field{Key: "isDebtor", Value: catTx.IsDebtor},
			logging.Field{Key: "description", Value: catTx.Description})

		// Note: Categorization is now handled by the categorizer component
		// through dependency injection, not directly in the parser
		transactions[i].Category = models.CategoryUncategorized

		p.GetLogger().Debug("Transaction parsed without categorization",
			logging.Field{Key: "amount", Value: transactions[i].Amount.String()},
			logging.Field{Key: "party", Value: catTx.PartyName})
	}

	return transactions
}

// ValidateFormat checks if the file is a valid CAMT.053 XML file
func (p *ISO20022Parser) ValidateFormat(filePath string) (bool, error) {
	p.GetLogger().Info("Validating CAMT.053 format",
		logging.Field{Key: "file", Value: filePath})

	// Try to open and read the file
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		if err := xmlFile.Close(); err != nil {
			p.GetLogger().Warn("Failed to close XML file",
				logging.Field{Key: "error", Value: err})
		}
	}()

	// Read the file content
	xmlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("error reading file: %w", err)
	}

	// Check if file is empty
	if len(xmlBytes) == 0 {
		p.GetLogger().Info("File is empty",
			logging.Field{Key: "file", Value: filePath})
		return false, fmt.Errorf("file is empty")
	}

	// Try to unmarshal the XML data into our ISO20022 document structure
	var document models.ISO20022Document
	if err := xml.Unmarshal(xmlBytes, &document); err != nil {
		p.GetLogger().Info("File is not a valid CAMT.053 XML",
			logging.Field{Key: "file", Value: filePath})
		return false, fmt.Errorf("invalid XML format: %w", err)
	}

	// Check if we have at least one statement
	if len(document.BkToCstmrStmt.Stmt) == 0 {
		p.GetLogger().Info("File is not a valid CAMT.053 XML (no statements)",
			logging.Field{Key: "file", Value: filePath})
		return false, fmt.Errorf("no statements found in CAMT.053 file")
	}

	p.GetLogger().Info("File is a valid CAMT.053 XML",
		logging.Field{Key: "file", Value: filePath})
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
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Write transactions to CSV file using inherited method
	return p.WriteToCSV(transactions, outputFile)
}

// CreateEmptyCSVFile creates an empty CSV file with transaction headers
// using the currently set CSV delimiter
func (c *ISO20022Parser) CreateEmptyCSVFile(outputFile string) error {
	c.GetLogger().Info("No transactions found, creating empty CSV file with headers",
		logging.Field{Key: "file", Value: outputFile},
		logging.Field{Key: "delimiter", Value: string(common.Delimiter)})

	// Create an empty transaction slice and use common package for consistency
	emptyTransactions := []models.Transaction{}
	return common.WriteTransactionsToCSV(emptyTransactions, outputFile)
}
