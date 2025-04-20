// Package camtparser provides functionality to parse and process CAMT.053 XML files.
package camtparser

import (
	"fmt"
	"os"
	"path/filepath"
	
	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/currencyutils"
	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/fileutils"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/textutils"
	"fjacquet/camt-csv/internal/xmlutils"
	
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gopkg.in/xmlpath.v2"
)

// XPathParser implements the Parser interface using XPath to extract data from CAMT.053 XML files
type XPathParser struct {
	log *logrus.Logger
}

// NewXPathParser creates a new parser that uses XPath for extraction
func NewXPathParser(logger *logrus.Logger) *XPathParser {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &XPathParser{
		log: logger,
	}
}

// SetLogger sets a custom logger for the parser
func (p *XPathParser) SetLogger(logger *logrus.Logger) {
	if logger != nil {
		p.log = logger
	}
}

// ParseFile parses a CAMT.053 XML file using XPath and returns a slice of Transaction objects
func (p *XPathParser) ParseFile(xmlFilePath string) ([]models.Transaction, error) {
	p.log.WithField("file", xmlFilePath).Info("Parsing CAMT.053 XML file (XPath mode)")
	
	// Check if file exists
	if !fileutils.FileExists(xmlFilePath) {
		return nil, fmt.Errorf("file does not exist: %s", xmlFilePath)
	}
	
	// Check if file is a valid CAMT.053 XML
	isValid, err := p.ValidateFormat(xmlFilePath)
	if err != nil {
		return nil, err
	}
	
	if !isValid {
		return nil, fmt.Errorf("invalid CAMT.053 XML format: %s", xmlFilePath)
	}
	
	// Extract transactions
	transactions, err := p.extractTransactionsFromXMLPath(xmlFilePath)
	if err != nil {
		return nil, err
	}
	
	// Categorize transactions
	transactions = p.categorizeTransactions(transactions)
	
	p.log.WithFields(logrus.Fields{
		"count": len(transactions),
	}).Info("Successfully extracted transactions from CAMT.053 file (XPath mode)")
	
	return transactions, nil
}

// ValidateFormat checks if a file is a valid CAMT.053 XML file
func (p *XPathParser) ValidateFormat(filePath string) (bool, error) {
	p.log.WithField("file", filePath).Info("Validating CAMT.053 format (XPath mode)")
	
	// Try to open and read the file
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("error opening file: %w", err)
	}
	defer xmlFile.Close()
	
	// Try to parse the XML file
	root, err := xmlpath.Parse(xmlFile)
	if err != nil {
		p.log.WithField("file", filePath).Info("File is not a valid XML")
		return false, nil
	}
	
	// Check for CAMT.053 specific elements
	path := xmlpath.MustCompile("//BkToCstmrStmt/Stmt")
	if iter := path.Iter(root); !iter.Next() {
		p.log.WithField("file", filePath).Info("File is not a valid CAMT.053 XML (no statements)")
		return false, nil
	}
	
	p.log.WithField("file", filePath).Info("File is a valid CAMT.053 XML")
	return true, nil
}

// ConvertToCSV converts a CAMT.053 XML file to CSV format
func (p *XPathParser) ConvertToCSV(inputFile, outputFile string) error {
	// Check if input file exists
	if !fileutils.FileExists(inputFile) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}
	
	// Parse XML file
	transactions, err := p.ParseFile(inputFile)
	if err != nil {
		return err
	}
	
	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := fileutils.EnsureDirectoryExists(outputDir); err != nil {
		return err
	}
	
	// Write transactions to CSV
	return p.WriteToCSV(transactions, outputFile)
}

// extractTransactionsFromXMLPath uses XPath to extract transactions from a CAMT.053 XML file
func (p *XPathParser) extractTransactionsFromXMLPath(xmlFilePath string) ([]models.Transaction, error) {
	// Open and read XML file
	xmlRoot, err := xmlutils.LoadXMLFile(xmlFilePath)
	if err != nil {
		return nil, err
	}
	
	// Initialize XPath constants
	// Note: Using xmlutils defaults directly, no need for local variable
	
	// Extract data using XPath
	amounts, err := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathAmount)
	if err != nil {
		return nil, err
	}
	
	currencies, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathCurrency)
	creditDebitInds, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathCreditDebitInd)
	bookingDates, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathBookingDate)
	valueDates, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathValueDate)
	statuses, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathStatus)
	
	refs, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathEndToEndID)
	altRefs, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathTransactionID)
	acctSvcRefs, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathAccountSvcRef)
	pmtInfIds, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathPaymentInfoID)
	
	remittanceInfos, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathRemittanceInfo)
	addEntryInfos, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathAddEntryInfo)
	addTxInfos, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathAddTxInfo)
	
	debtorNames, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathDebtorName)
	// These agent fields may be used in the future but are currently unused
	//debtorAgentNames, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathDebtorAgentName)
	creditorNames, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathCreditorName)
	//creditorAgentNames, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathCreditorAgentName)
	ultimateDebtors, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathUltimateDebtor)
	ultimateCreditors, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathUltimateCreditor)
	
	ibans, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathIBAN)
	
	// These bank transaction code fields may be used in the future but are currently unused
	//bankTxDomains, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathBankTxDomain)
	//bankTxFamilies, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathBankTxFamily)
	//bankTxSubFamilies, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathBankTxSubFamily)
	//proprietaryCodes, _ := xmlutils.ExtractFromXML(xmlRoot, xmlutils.XPathProprietaryCode)
	
	transactions := make([]models.Transaction, 0, len(amounts))
	
	// Helper function to get a value from a slice with index bounds checking
	getOrEmpty := func(slice []string, index int) string {
		if index < len(slice) {
			return slice[index]
		}
		return ""
	}
	
	// Process each transaction
	for i := 0; i < len(amounts); i++ {
		// Get references
		reference := p.getReference(getOrEmpty(refs, i), getOrEmpty(altRefs, i), getOrEmpty(acctSvcRefs, i), getOrEmpty(pmtInfIds, i))
		
		// Get remittance info
		remittanceInfo := getOrEmpty(remittanceInfos, i)
		additionalEntryInfo := getOrEmpty(addEntryInfos, i)
		additionalTxInfo := getOrEmpty(addTxInfos, i)
		
		// Combine remittance info
		description := remittanceInfo
		if description == "" && additionalEntryInfo != "" {
			description = additionalEntryInfo
		}
		if description == "" && additionalTxInfo != "" {
			description = additionalTxInfo
		}
		
		// Get party details
		debtorName := getOrEmpty(debtorNames, i)
		// Not using these variables currently, but may be needed in future enhancements
		//debtorAgentName := getOrEmpty(debtorAgentNames, i)
		creditorName := getOrEmpty(creditorNames, i)
		//creditorAgentName := getOrEmpty(creditorAgentNames, i)
		ultimateDebtor := getOrEmpty(ultimateDebtors, i)
		ultimateCreditor := getOrEmpty(ultimateCreditors, i)
		
		// Not using these bank transaction codes currently
		//bankTxDomain := getOrEmpty(bankTxDomains, i)
		//bankTxFamily := getOrEmpty(bankTxFamilies, i)
		//bankTxSubFamily := getOrEmpty(bankTxSubFamilies, i)
		//proprietaryCode := getOrEmpty(proprietaryCodes, i)
		
		// Get credit/debit indicator
		creditDebitInd := getOrEmpty(creditDebitInds, i)
		
		// Get amount and currency
		amountStr := getOrEmpty(amounts, i)
		amount, err := currencyutils.ParseAmount(amountStr)
		if err != nil {
			p.log.WithError(err).Warnf("Failed to parse amount: %s", amountStr)
			continue
		}
		
		currency := getOrEmpty(currencies, i)
		
		// Handle credit/debit indicator
		if creditDebitInd == "DBIT" {
			amount = amount.Neg()
		}
		
		// Get booking date
		bookingDateStr := getOrEmpty(bookingDates, i)
		bookingDate, _, err := dateutils.ParseDate(bookingDateStr)
		if err != nil {
			p.log.WithError(err).Warnf("Failed to parse booking date: %s", bookingDateStr)
			continue
		}
		
		// Get value date
		valueDateStr := getOrEmpty(valueDates, i)
		valueDate := bookingDate
		if valueDateStr != "" {
			if parsedDate, _, err := dateutils.ParseDate(valueDateStr); err == nil {
				valueDate = parsedDate
			}
		}
		
		// Get status
		status := getOrEmpty(statuses, i)
		
		// Get IBAN
		var iban string
		if len(ibans) > 0 {
			iban = ibans[0]
		}
		
		// Create Transaction object
		transaction := models.Transaction{
			Date:              dateutils.ToISODate(bookingDate),
			ValueDate:         dateutils.ToISODate(valueDate),
			Description:       description,
			Reference:         reference,
			Amount:            amount,
			Currency:          currency,
			Status:            status,
			Category:          "Uncategorized", // Will be set by categorization
			BookkeepingNumber: textutils.ExtractBookkeepingNumber(description),
			IBAN:              iban,
			RemittanceInfo:    remittanceInfo,
			CreditDebit:       creditDebitInd,
		}
		
		// Set party details based on debit/credit
		if amount.IsNegative() {
			transaction.DebitFlag = true
			transaction.Debit = amount.Abs()
			transaction.Credit = decimal.Zero
			transaction.PartyName = creditorName
			transaction.PartyIBAN = ""
			if ultimateCreditor != "" {
				transaction.PartyName = ultimateCreditor
			}
			if transaction.PartyName == "" {
				transaction.PartyName = additionalEntryInfo
			}
		} else {
			transaction.DebitFlag = false
			transaction.Debit = decimal.Zero
			transaction.Credit = amount
			transaction.PartyName = debtorName
			transaction.PartyIBAN = ""
			if ultimateDebtor != "" {
				transaction.PartyName = ultimateDebtor
			}
		}
		
		// Extract funds information if present
		transaction.Fund = textutils.ExtractFundInfo(description)
		
		// Add transaction to list
		transactions = append(transactions, transaction)
	}
	
	return transactions, nil
}

// categorizeTransactions applies categorization to all transactions
func (p *XPathParser) categorizeTransactions(transactions []models.Transaction) []models.Transaction {
	for i := range transactions {
		// Skip transactions that already have a category (like the special case for DELL salary)
		if transactions[i].Category != "" {
			continue
		}
		
		// Create categorizer transaction from our transaction
		isDebtor := transactions[i].DebitFlag
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

// getReference combines multiple reference fields to get a single reference
func (p *XPathParser) getReference(endToEndId, txId, acctSvcRef, pmtInfId string) string {
	if endToEndId != "" {
		return endToEndId
	}
	if txId != "" {
		return txId
	}
	if acctSvcRef != "" {
		return acctSvcRef
	}
	if pmtInfId != "" {
		return pmtInfId
	}
	return ""
}

// WriteToCSV writes transactions to a CSV file
func (p *XPathParser) WriteToCSV(transactions []models.Transaction, csvFile string) error {
	p.log.WithFields(logrus.Fields{
		"file":  csvFile,
		"count": len(transactions),
	}).Info("Writing transactions to CSV file")
	
	// Create the directory if it doesn't exist
	dir := filepath.Dir(csvFile)
	if err := fileutils.EnsureDirectoryExists(dir); err != nil {
		return err
	}
	
	return common.WriteTransactionsToCSV(transactions, csvFile)
}
