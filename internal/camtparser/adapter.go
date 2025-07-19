// Package camtparser provides functionality to parse and process CAMT.053 XML files.
package camtparser

import (
	"bytes"
	"encoding/xml"
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/parsererror"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html/charset"
)

// Adapter implements the parser.Parser interface for CAMT.053 XML files.
// It supports multiple parser implementations through a strategy pattern.
type Adapter struct {
	defaultParser parser.DefaultParser
}

// NewAdapter creates a new adapter for the camtparser.
func NewAdapter() parser.Parser {
	// Default to using the active parser strategy
	adapter := &Adapter{}

	// Set up the default parser with this adapter as implementation
	adapter.defaultParser = parser.DefaultParser{
		Logger: logrus.New(),
		Impl:   adapter,
	}

	return adapter
}

// ParseFile implements parser.Parser.ParseFile
// Direct implementation to avoid recursive calls to package-level function
func (a *Adapter) ParseFile(filePath string) ([]models.Transaction, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, parsererror.FileNotFoundError(filePath)
	}

	// Default to ISO20022 parser (only implementation now)
	return parseFileISO20022(filePath)
}

// ValidateFormat implements parser.Parser.ValidateFormat
// Direct implementation to avoid recursive calls to package-level function
func (a *Adapter) ValidateFormat(filePath string) (bool, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, parsererror.FileNotFoundError(filePath)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logrus.Warnf("Failed to close file: %v", err)
		}
	}()

	// Read enough bytes to check for XML header and CAMT053 identifiers
	buffer := make([]byte, 4096)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}
	xmlHeader := string(buffer[:n])

	// Basic checks for XML format
	if !strings.Contains(xmlHeader, "<?xml") {
		return false, nil
	}

	// Check for CAMT.053 specific elements (simplified for quick validation)
	isCamt := strings.Contains(xmlHeader, "Document") &&
		(strings.Contains(xmlHeader, "BkToCstmrStmt") ||
			strings.Contains(xmlHeader, "camt.053"))

	return isCamt, nil
}

// ConvertToCSV converts an XML file to a CSV file based on the chosen parser type
func (a *Adapter) ConvertToCSV(xmlFile, csvFile string) error {
	// Parse the XML file
	transactions, err := a.ParseFile(xmlFile)
	if err != nil {
		return err
	}

	// Handle empty transactions list
	if len(transactions) == 0 {
		// Create an empty transaction slice and use common package for consistency
		// This ensures the delimiter setting is properly respected
		logrus.WithFields(logrus.Fields{
			"file":      csvFile,
			"delimiter": string(common.Delimiter),
		}).Info("No transactions found, created empty CSV file with headers")

		emptyTransactions := []models.Transaction{}
		return common.WriteTransactionsToCSV(emptyTransactions, csvFile)
	}

	// Write the transactions to the CSV file
	logrus.WithFields(logrus.Fields{
		"count": len(transactions),
		"file":  csvFile,
	}).Info("Writing transactions to CSV file")

	// Create the directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := common.ExportTransactionsToCSV(transactions, csvFile); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"count": len(transactions),
		"file":  csvFile,
	}).Info("Successfully wrote transactions to CSV file")

	return nil
}

// WriteToCSV implements parser.Parser.WriteToCSV
// Uses the standardized implementation from DefaultParser.
func (a *Adapter) WriteToCSV(transactions []models.Transaction, csvFile string) error {
	return a.defaultParser.WriteToCSV(transactions, csvFile)
}

// SetLogger implements parser.Parser.SetLogger
// Sets the logger for both the adapter and the active parser strategy.
func (a *Adapter) SetLogger(logger *logrus.Logger) {
	a.defaultParser.Logger = logger
	a.defaultParser.SetLogger(logger)
	// Remove the circular dependency - don't call back to the package-level function
	// SetLogger(logger)
}

// NewISO20022Adapter creates a new adapter that specifically uses the ISO20022 parser.
func NewISO20022Adapter(logger *logrus.Logger) parser.Parser {
	// Create the adapter without calling SetParserType to avoid recursion
	adapter := &Adapter{}

	// Set up the default parser with this adapter as implementation
	adapter.defaultParser = parser.DefaultParser{
		Logger: logger,
		Impl:   adapter,
	}

	// Manually set the parser type flag (implementation detail)
	currentParserType = "iso20022"

	return adapter
}

// Helper functions for the actual parsing implementations
func parseFileISO20022(filePath string) ([]models.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logrus.Warnf("Failed to close file: %v", err)
		}
	}()

	// Read the XML content
	xmlData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Create decoder with proper namespace handling
	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	decoder.CharsetReader = charset.NewReaderLabel

	// Define the Document structure for CAMT.053
	type Amount struct {
		Value    string `xml:",chardata"`
		Currency string `xml:"Ccy,attr"`
	}

	type Date struct {
		Date string `xml:"Dt"`
	}

	type AdditionalInfo struct {
		Info string `xml:",chardata"`
	}

	type CreditDebitIndicator struct {
		Indicator string `xml:",chardata"`
	}

	type Status struct {
		Status string `xml:",chardata"`
	}

	type AccountServicerRef struct {
		Ref string `xml:",chardata"`
	}

	type Reference struct {
		MsgId       string `xml:"MsgId,omitempty"`
		AcctSvcrRef string `xml:"AcctSvcrRef,omitempty"`
		InstrId     string `xml:"InstrId,omitempty"`
		EndToEndId  string `xml:"EndToEndId,omitempty"`
		TxId        string `xml:"TxId,omitempty"`
	}

	type RemittanceInfo struct {
		Ustrd string `xml:"Ustrd"`
	}

	type Account struct {
		IBAN string `xml:"Id>IBAN,omitempty"`
		ID   string `xml:"Id>Othr>Id,omitempty"`
	}

	type RelatedParties struct {
		Debtor struct {
			Name    string  `xml:"Nm"`
			Account Account `xml:"Acct,omitempty"`
		} `xml:"Dbtr"`
		Creditor struct {
			Name    string  `xml:"Nm"`
			Account Account `xml:"Acct,omitempty"`
		} `xml:"Cdtr"`
		DebtorAccount   Account `xml:"DbtrAcct,omitempty"`
		CreditorAccount Account `xml:"CdtrAcct,omitempty"`
	}

	type RelatedAccounts struct {
		DebtorAccount   Account `xml:"DbtrAcct,omitempty"`
		CreditorAccount Account `xml:"CdtrAcct,omitempty"`
	}

	type TransactionDetails struct {
		References      Reference            `xml:"Refs"`
		Amount          Amount               `xml:"Amt"`
		CreditDebit     CreditDebitIndicator `xml:"CdtDbtInd"`
		RemittanceInfo  RemittanceInfo       `xml:"RmtInf"`
		RelatedParties  RelatedParties       `xml:"RltdPties"`
		RelatedAccounts RelatedAccounts      `xml:"RltdAccts,omitempty"`
	}

	type EntryDetails struct {
		TransactionDetails TransactionDetails `xml:"TxDtls"`
	}

	type Entry struct {
		Amount          Amount               `xml:"Amt"`
		CreditDebit     CreditDebitIndicator `xml:"CdtDbtInd"`
		Status          Status               `xml:"Sts"`
		BookingDate     Date                 `xml:"BookgDt"`
		ValueDate       Date                 `xml:"ValDt"`
		AccountServicer AccountServicerRef   `xml:"AcctSvcrRef"`
		EntryDetails    EntryDetails         `xml:"NtryDtls"`
		AdditionalInfo  AdditionalInfo       `xml:"AddtlNtryInf"`
	}

	type Statement struct {
		Entries []Entry `xml:"Ntry"`
	}

	type Document struct {
		XMLName       xml.Name `xml:"Document"`
		BkToCstmrStmt struct {
			Stmt []Statement `xml:"Stmt"`
		} `xml:"BkToCstmrStmt"`
	}

	// Unmarshal the XML
	var doc Document
	err = decoder.Decode(&doc)
	if err != nil {
		return nil, fmt.Errorf("error decoding XML: %w", err)
	}

	var transactions []models.Transaction

	// Process all statements and entries
	for _, stmt := range doc.BkToCstmrStmt.Stmt {
		for _, entry := range stmt.Entries {
			// Convert dates to standard format
			bookingDate := entry.BookingDate.Date
			valueDate := entry.ValueDate.Date

			// Format the dates as DD.MM.YYYY
			if bookingDateParsed, err := time.Parse("2006-01-02", bookingDate); err == nil {
				bookingDate = bookingDateParsed.Format("02.01.2006")
			}

			if valueDateParsed, err := time.Parse("2006-01-02", valueDate); err == nil {
				valueDate = valueDateParsed.Format("02.01.2006")
			}

			// Create transaction
			transaction := models.Transaction{
				Date:            bookingDate,
				ValueDate:       valueDate,
				Amount:          models.ParseAmount(entry.Amount.Value),
				Currency:        entry.Amount.Currency,
				CreditDebit:     entry.CreditDebit.Indicator,
				AccountServicer: entry.AccountServicer.Ref,
				Status:          entry.Status.Status,
			}

			// Add details from transaction details if available
			txDetails := entry.EntryDetails.TransactionDetails

			// Set description from AddtlNtryInf or RemittanceInfo
			if entry.AdditionalInfo.Info != "" {
				transaction.Description = entry.AdditionalInfo.Info
			} else if transaction.RemittanceInfo != "" {
				// Use RemittanceInfo as Description if there's no AddtlNtryInf
				transaction.Description = transaction.RemittanceInfo
			}

			// Handle special case for ORDRE LSV + transactions
			if strings.Contains(transaction.Description, "ORDRE LSV +") {
				// For LSV+ transactions, get the creditor name from related parties if available
				cdtrName := ""
				if len(txDetails.RelatedParties.Creditor.Name) > 0 {
					cdtrName = txDetails.RelatedParties.Creditor.Name
					// Set PartyName and Name to the creditor name
					transaction.PartyName = cdtrName
					transaction.Name = cdtrName
					transaction.Type = "Virement"
				}
			} else {
				// Extract PartyName from Description if it starts with specific prefixes
				if transaction.Description != "" {
					extractedName := extractPartyNameFromDescription(transaction.Description)
					if extractedName != "" {
						transaction.PartyName = extractedName
					}
				}

				// If PartyName is still empty, try to get debtor name from related parties if available
				if transaction.PartyName == "" && len(txDetails.RelatedParties.Debtor.Name) > 0 {
					transaction.PartyName = txDetails.RelatedParties.Debtor.Name
				}

				// Check for IBAN in related parties and accounts
				// Try multiple possible paths for finding the IBAN
				if txDetails.RelatedParties.Debtor.Account.IBAN != "" {
					transaction.PartyIBAN = txDetails.RelatedParties.Debtor.Account.IBAN
				} else if txDetails.RelatedParties.Creditor.Account.IBAN != "" {
					transaction.PartyIBAN = txDetails.RelatedParties.Creditor.Account.IBAN
				} else if txDetails.RelatedParties.DebtorAccount.IBAN != "" {
					transaction.PartyIBAN = txDetails.RelatedParties.DebtorAccount.IBAN
				} else if txDetails.RelatedParties.CreditorAccount.IBAN != "" {
					transaction.PartyIBAN = txDetails.RelatedParties.CreditorAccount.IBAN
				} else if txDetails.RelatedAccounts.DebtorAccount.IBAN != "" {
					transaction.PartyIBAN = txDetails.RelatedAccounts.DebtorAccount.IBAN
				} else if txDetails.RelatedAccounts.CreditorAccount.IBAN != "" {
					transaction.PartyIBAN = txDetails.RelatedAccounts.CreditorAccount.IBAN
				} else if txDetails.RelatedParties.Debtor.Account.ID != "" && isIBANFormat(txDetails.RelatedParties.Debtor.Account.ID) {
					// Some CAMT files store IBAN in the ID field
					transaction.PartyIBAN = txDetails.RelatedParties.Debtor.Account.ID
				} else if txDetails.RelatedParties.Creditor.Account.ID != "" && isIBANFormat(txDetails.RelatedParties.Creditor.Account.ID) {
					transaction.PartyIBAN = txDetails.RelatedParties.Creditor.Account.ID
				}

				// Set transaction Type based on description prefix
				transactionType := setTransactionTypeFromDescription(transaction.Description)
				if transactionType != "" {
					transaction.Type = transactionType
				}
			}

			// Get remittance info
			if txDetails.RemittanceInfo.Ustrd != "" {
				transaction.RemittanceInfo = txDetails.RemittanceInfo.Ustrd
			}

			// Get reference information
			if txDetails.References.MsgId != "" {
				transaction.Reference = txDetails.References.MsgId
			} else if txDetails.References.EndToEndId != "" {
				transaction.Reference = txDetails.References.EndToEndId
			} else if txDetails.References.TxId != "" {
				transaction.Reference = txDetails.References.TxId
			} else if txDetails.References.AcctSvcrRef != "" {
				transaction.Reference = txDetails.References.AcctSvcrRef
			}

			// Set Name from PartyName and also update Payee/Payer fields to ensure
			// that UpdateNameFromParties won't override our Name during export
			if transaction.Name == "" {
				transaction.Name = transaction.PartyName
			}
			if transaction.IsDebit() {
				transaction.Payee = transaction.PartyName
			} else {
				transaction.Payer = transaction.PartyName
			}

			// Update derived fields
			transaction.UpdateDebitCreditAmounts()

			// Categorize the transaction
			catTransaction := categorizer.Transaction{
				PartyName:   transaction.PartyName,
				IsDebtor:    transaction.CreditDebit == "DBIT", // Use the CreditDebit field to determine if it's a debit transaction
				Amount:      transaction.Amount.String(),
				Date:        transaction.Date,
				Info:        transaction.RemittanceInfo,
				Description: transaction.Description,
			}

			// If PartyName is empty, use Description or RemittanceInfo to help with categorization
			if catTransaction.PartyName == "" {
				// Try to use Description as PartyName if available
				if transaction.Description != "" {
					catTransaction.PartyName = transaction.Description
				} else if transaction.RemittanceInfo != "" {
					// Otherwise use RemittanceInfo
					catTransaction.PartyName = transaction.RemittanceInfo
				}
			}

			// Clean PartyName by removing payment method prefixes before categorization
			catTransaction.PartyName = cleanPaymentMethodPrefixes(catTransaction.PartyName)

			// Apply categorization
			category, err := categorizer.CategorizeTransaction(catTransaction)
			if err == nil && category.Name != "" {
				transaction.Category = category.Name
			}

			transactions = append(transactions, transaction)
		}
	}

	return transactions, nil
}

// isIBANFormat checks if a string appears to be in IBAN format
func isIBANFormat(s string) bool {
	// Basic check: IBANs typically start with a country code (2 letters) followed by
	// checksum digits and the account number
	if len(s) < 15 || len(s) > 34 {
		return false
	}

	// Check if the first two characters are letters (country code)
	if len(s) >= 2 && !('A' <= s[0] && s[0] <= 'Z') && !('a' <= s[0] && s[0] <= 'z') {
		return false
	}
	if len(s) >= 2 && !('A' <= s[1] && s[1] <= 'Z') && !('a' <= s[1] && s[1] <= 'z') {
		return false
	}

	// Check if the rest is alphanumeric
	for i := 2; i < len(s); i++ {
		c := s[i]
		if !('0' <= c && c <= '9') && !('A' <= c && c <= 'Z') && !('a' <= c && c <= 'z') {
			return false
		}
	}

	return true
}

// Helper function to clean payment method prefixes from party names
func cleanPaymentMethodPrefixes(partyName string) string {
	// If the party name consists only of one of these terms, leave it as is
	// This ensures "PMT CARTE", "PMT TWINT", and "BCV-NET" are still categorized correctly
	if partyName == "PMT CARTE" || partyName == "PMT TWINT" || partyName == "BCV-NET" || partyName == "VIRT BANC" {
		return partyName
	}

	// Remove these prefixes if they're part of a longer string
	prefixes := []string{"PMT CARTE", "PMT TWINT", "BCV-NET", "VIRT BANC"}
	cleanedName := partyName

	for _, prefix := range prefixes {
		// Check if the party name starts with the prefix
		if strings.HasPrefix(cleanedName, prefix) {
			// Extract the remaining part after the prefix and trim spaces
			cleanedName = strings.TrimSpace(cleanedName[len(prefix):])
			break // Only need to remove one prefix
		}
	}

	// If we removed everything, return the original name
	if cleanedName == "" {
		return partyName
	}

	return cleanedName
}

// extractPartyNameFromDescription extracts party name from description based on prefixes
func extractPartyNameFromDescription(description string) string {
	prefixes := []string{"PMT TWINT", "PMT CARTE", "VIRT BANC", "BCV-NET"}

	for _, prefix := range prefixes {
		if strings.HasPrefix(description, prefix) {
			// Extract the remaining part after the prefix and trim spaces
			remaining := strings.TrimSpace(description[len(prefix):])
			if remaining != "" {
				return remaining
			}
		}
	}

	return ""
}

// setTransactionTypeFromDescription sets the transaction type based on description prefix
func setTransactionTypeFromDescription(description string) string {
	if strings.HasPrefix(description, "PMT TWINT") {
		return "TWINT"
	} else if strings.HasPrefix(description, "PMT CARTE") {
		return "CB"
	} else if strings.HasPrefix(description, "VIRT BANC") {
		return "Virement"
	} else if strings.HasPrefix(description, "BCV-NET") {
		return "Virement"
	} else if description == "ORDRE LSV +" {
		return "Virement"
	}
	return ""
}
