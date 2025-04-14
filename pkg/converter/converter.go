// Package converter provides functionality to convert CAMT.053 XML files to CSV format.
package converter

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fjacquet/camt-csv/pkg/categorizer"
	"fjacquet/camt-csv/pkg/config"
)

// CAMT053 is a struct that represents the CAMT.053 XML structure
type CAMT053 struct {
	XMLName       xml.Name      `xml:"Document"`
	BkToCstmrStmt BkToCstmrStmt `xml:"BkToCstmrStmt"`
}

// BkToCstmrStmt represents the Bank To Customer Statement
type BkToCstmrStmt struct {
	GrpHdr GrpHdr `xml:"GrpHdr"`
	Stmt   Stmt   `xml:"Stmt"`
}

// GrpHdr represents the Group Header
type GrpHdr struct {
	MsgId    string `xml:"MsgId"`
	CreDtTm  string `xml:"CreDtTm"`
	MsgPgntn struct {
		PgNb       string `xml:"PgNb"`
		LastPgInd  string `xml:"LastPgInd"`
	} `xml:"MsgPgntn"`
}

// Stmt represents the Statement
type Stmt struct {
	Id        string  `xml:"Id"`
	ElctrncSeqNb string `xml:"ElctrncSeqNb"`
	CreDtTm   string  `xml:"CreDtTm"`
	FrToDt    FrToDt  `xml:"FrToDt"`
	Acct      Acct    `xml:"Acct"`
	Bal       []Bal   `xml:"Bal"`
	Ntry      []Ntry  `xml:"Ntry"`
}

// FrToDt represents the From To Date
type FrToDt struct {
	FrDtTm  string `xml:"FrDtTm"`
	ToDtTm  string `xml:"ToDtTm"`
}

// Acct represents the Account
type Acct struct {
	Id   struct {
		IBAN string `xml:"IBAN"`
	} `xml:"Id"`
	Ccy  string `xml:"Ccy"`
	Ownr struct {
		Nm  string `xml:"Nm"`
	} `xml:"Ownr"`
}

// Bal represents the Balance
type Bal struct {
	Tp        Tp     `xml:"Tp"`
	Amt       Amt    `xml:"Amt"`
	CdtDbtInd string `xml:"CdtDbtInd"`
	Dt        struct {
		Dt string `xml:"Dt"`
	} `xml:"Dt"`
}

// Tp represents the Type
type Tp struct {
	CdOrPrtry CdOrPrtry `xml:"CdOrPrtry"`
}

// CdOrPrtry represents the Code or Proprietary
type CdOrPrtry struct {
	Cd string `xml:"Cd"`
}

// Amt represents the Amount
type Amt struct {
	Text string `xml:",chardata"`
	Ccy  string `xml:"Ccy,attr"`
}

// Ntry represents the Entry
type Ntry struct {
	NtryRef      string    `xml:"NtryRef"`
	Amt          Amt       `xml:"Amt"`
	CdtDbtInd    string    `xml:"CdtDbtInd"`
	Sts          string    `xml:"Sts"`
	BookgDt      BookgDt   `xml:"BookgDt"`
	ValDt        ValDt     `xml:"ValDt"`
	AcctSvcrRef  string    `xml:"AcctSvcrRef"`
	BkTxCd       BkTxCd    `xml:"BkTxCd"`
	NtryDtls     NtryDtls  `xml:"NtryDtls"`
	AddtlNtryInf string    `xml:"AddtlNtryInf"`
}

// BookgDt represents the Booking Date
type BookgDt struct {
	Dt string `xml:"Dt"`
}

// ValDt represents the Value Date
type ValDt struct {
	Dt string `xml:"Dt"`
}

// BkTxCd represents the Bank Transaction Code
type BkTxCd struct {
	Domn Domn `xml:"Domn"`
}

// Domn represents the Domain
type Domn struct {
	Cd    string `xml:"Cd"`
	Fmly  Fmly   `xml:"Fmly"`
}

// Fmly represents the Family
type Fmly struct {
	Cd         string `xml:"Cd"`
	SubFmlyCd  string `xml:"SubFmlyCd"`
}

// NtryDtls represents the Entry Details
type NtryDtls struct {
	TxDtls []TxDtls `xml:"TxDtls"`
}

// TxDtls represents the Transaction Details
type TxDtls struct {
	Refs     Refs     `xml:"Refs"`
	Amt      Amt      `xml:"Amt"`
	CdtDbtInd string  `xml:"CdtDbtInd"`
	AmtDtls  AmtDtls  `xml:"AmtDtls"`
	RltdPties RltdPties `xml:"RltdPties"`
	RmtInf    RmtInf   `xml:"RmtInf"`
}

// Refs represents the References
type Refs struct {
	MsgId       string `xml:"MsgId"`
	EndToEndId  string `xml:"EndToEndId"`
	TxId        string `xml:"TxId"`
	InstrId     string `xml:"InstrId"`
}

// AmtDtls represents the Amount Details
type AmtDtls struct {
	InstdAmt struct {
		Amt Amt `xml:"Amt"`
	} `xml:"InstdAmt"`
}

// RltdPties represents the Related Parties
type RltdPties struct {
	Dbtr       Party `xml:"Dbtr"`
	DbtrAcct   Acct  `xml:"DbtrAcct"`
	Cdtr       Party `xml:"Cdtr"`
	CdtrAcct   Acct  `xml:"CdtrAcct"`
}

// Party represents a Party (Debtor or Creditor)
type Party struct {
	Nm string `xml:"Nm"`
}

// RmtInf represents the Remittance Information
type RmtInf struct {
	Ustrd []string `xml:"Ustrd"`
}

// Transaction represents a financial transaction extracted from CAMT.053
type Transaction struct {
	Date            string
	ValueDate       string
	Description     string
	BookkeepingNo   string
	Fund            string
	Amount          string
	Currency        string
	CreditDebit     string
	EntryReference  string
	AccountServicer string
	BankTxCode      string
	Status          string
	Payee           string
	Payer           string
	IBAN            string
	NumberOfShares  string
	StampDuty       string
	Investment      string
	Category        string // Added for automatic categorization
}

// ConvertXMLToCSV converts a CAMT.053 XML file to CSV format
func ConvertXMLToCSV(xmlFile string, csvFile string) error {
	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		return fmt.Errorf("error reading XML file: %w", err)
	}

	var camt053 CAMT053
	err = xml.Unmarshal(xmlData, &camt053)
	if err != nil {
		return fmt.Errorf("error unmarshalling XML: %w", err)
	}

	csvFileHandle, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer csvFileHandle.Close()

	writer := csv.NewWriter(csvFileHandle)
	defer writer.Flush()

	header := []string{"Date", "ValueDate", "Description", "BookkeepingNo", "Fund", "Amount", "Currency", "CreditDebit", "EntryReference", "AccountServicer", "BankTxCode", "Status", "Payee", "Payer", "IBAN", "NumberOfShares", "StampDuty", "Investment", "Category"}
	err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	transactions := extractTransactions(&camt053)
	
	// Load environment variables for API keys
	config.LoadEnv()
	
	// Automatically categorize each transaction
	for i := range transactions {
		// Skip categorization if payee is empty
		if transactions[i].Payee == "" {
			transactions[i].Category = "Uncategorized"
			continue
		}
		
		// Create a categorizer transaction from our converter transaction
		catTx := categorizer.Transaction{
			Payee:   transactions[i].Payee,
			Amount:  transactions[i].Amount,
			Date:    transactions[i].Date,
			Info:    transactions[i].Description,
		}
		
		// Try to categorize the transaction
		category, err := categorizer.CategorizeTransaction(catTx)
		if err != nil {
			// If categorization fails, just use "Uncategorized"
			transactions[i].Category = "Uncategorized"
		} else {
			// Otherwise, use the determined category
			transactions[i].Category = category.Name
		}
	}
	
	for _, transaction := range transactions {
		record := []string{
			transaction.Date,
			transaction.ValueDate,
			transaction.Description,
			transaction.BookkeepingNo,
			transaction.Fund,
			transaction.Amount,
			transaction.Currency,
			transaction.CreditDebit,
			transaction.EntryReference,
			transaction.AccountServicer,
			transaction.BankTxCode,
			transaction.Status,
			transaction.Payee,
			transaction.Payer,
			transaction.IBAN,
			transaction.NumberOfShares,
			transaction.StampDuty,
			transaction.Investment,
			transaction.Category,
		}
		err = writer.Write(record)
		if err != nil {
			return fmt.Errorf("error writing CSV record: %w", err)
		}
	}

	return nil
}

// extractTransactions extracts all transactions from a CAMT.053 document
func extractTransactions(camt053 *CAMT053) []Transaction {
	var transactions []Transaction
	
	// Get account information from statement
	iban := ""
	accountOwner := ""
	if camt053.BkToCstmrStmt.Stmt.Acct.Id.IBAN != "" {
		iban = camt053.BkToCstmrStmt.Stmt.Acct.Id.IBAN
	}
	if camt053.BkToCstmrStmt.Stmt.Acct.Ownr.Nm != "" {
		accountOwner = camt053.BkToCstmrStmt.Stmt.Acct.Ownr.Nm
	}

	for _, entry := range camt053.BkToCstmrStmt.Stmt.Ntry {
		// Default transaction with data available at the entry level
		transaction := Transaction{
			Date:            formatDate(entry.BookgDt.Dt),
			ValueDate:       formatDate(entry.ValDt.Dt),
			Description:     entry.AddtlNtryInf,
			BookkeepingNo:   "",
			Fund:            "",
			Amount:          entry.Amt.Text,
			Currency:        entry.Amt.Ccy,
			CreditDebit:     entry.CdtDbtInd,
			EntryReference:  entry.NtryRef,
			AccountServicer: entry.AcctSvcrRef,
			BankTxCode:      formatBankTxCode(entry.BkTxCd),
			Status:          entry.Sts,
			Payee:           "", // Will be filled in based on transaction details
			Payer:           accountOwner,
			IBAN:            iban,
			NumberOfShares:  "",
			StampDuty:       "",
			Investment:      entry.AddtlNtryInf,
			Category:        "", // Initialize Category field
		}
		
		// Check if there are transaction details
		if len(entry.NtryDtls.TxDtls) > 0 {
			// Process each transaction detail
			for _, txDtl := range entry.NtryDtls.TxDtls {
				// Create a copy of the base transaction for this detail
				txCopy := transaction
				
				// Handle references
				txCopy.BookkeepingNo = extractReference(txDtl.Refs)
				
				// If no description yet, use the reference as description
				if txCopy.Description == "" && txCopy.BookkeepingNo != "" {
					txCopy.Description = txCopy.BookkeepingNo
				}
				
				// Extract payee according to CAMT.053 standard
				// For credit entries, the debtor is the payee (who paid us)
				// For debit entries, the creditor is the payee (whom we paid)
				if entry.CdtDbtInd == "CRDT" && txDtl.RltdPties.Dbtr.Nm != "" {
					txCopy.Payee = txDtl.RltdPties.Dbtr.Nm
				} else if entry.CdtDbtInd == "DBIT" && txDtl.RltdPties.Cdtr.Nm != "" {
					txCopy.Payee = txDtl.RltdPties.Cdtr.Nm
				}
				
				// Extract IBAN information
				if entry.CdtDbtInd == "CRDT" && txDtl.RltdPties.DbtrAcct.Id.IBAN != "" {
					txCopy.IBAN = txDtl.RltdPties.DbtrAcct.Id.IBAN
				} else if entry.CdtDbtInd == "DBIT" && txDtl.RltdPties.CdtrAcct.Id.IBAN != "" {
					txCopy.IBAN = txDtl.RltdPties.CdtrAcct.Id.IBAN
				}
				
				// Check if there is remittance information
				if len(txDtl.RmtInf.Ustrd) > 0 {
					// Process each remittance information
					for _, ustrd := range txDtl.RmtInf.Ustrd {
						// Create a copy of the transaction detail
						rmtCopy := txCopy
						
						// Extract additional information from remittance info
						rmtCopy.Description = cleanDescription(ustrd)
						
						// Try to extract bookkeeping number if not already set
						if rmtCopy.BookkeepingNo == "" {
							rmtCopy.BookkeepingNo = extractBookkeepingNo(ustrd)
						}
						
						rmtCopy.Fund = extractFund(ustrd)
						rmtCopy.Investment = ustrd
						
						// Try to extract payee from remittance info if not set yet
						if rmtCopy.Payee == "" {
							rmtCopy.Payee = extractPayeeFromRemittanceInfo(ustrd, entry.CdtDbtInd)
						}
						
						transactions = append(transactions, rmtCopy)
					}
				} else {
					// No remittance info, just add the transaction with reference info
					
					// If payee is still empty, use a fallback
					if txCopy.Payee == "" {
						txCopy.Payee = extractFallbackPayee(entry, txCopy.Description)
					}
					
					transactions = append(transactions, txCopy)
				}
			}
		} else {
			// No transaction details, just add the base transaction
			// Use the description as fallback payee if no other payee is available
			transaction.Payee = extractFallbackPayee(entry, transaction.Description)
			
			transactions = append(transactions, transaction)
		}
	}

	return transactions
}

// formatBankTxCode formats the bank transaction code properly
func formatBankTxCode(bkTxCd BkTxCd) string {
	if bkTxCd.Domn.Cd == "" {
		return ""
	}
	
	result := bkTxCd.Domn.Cd
	if bkTxCd.Domn.Fmly.Cd != "" {
		result += "/" + bkTxCd.Domn.Fmly.Cd
	}
	if bkTxCd.Domn.Fmly.SubFmlyCd != "" {
		result += "/" + bkTxCd.Domn.Fmly.SubFmlyCd
	}
	
	return result
}

// extractReference gets the best reference from the available options
func extractReference(refs Refs) string {
	if refs.EndToEndId != "" {
		return refs.EndToEndId
	} else if refs.TxId != "" {
		return refs.TxId
	} else if refs.MsgId != "" {
		return refs.MsgId
	} else if refs.InstrId != "" {
		return refs.InstrId
	}
	return ""
}

// extractPayeeFromRemittanceInfo tries to extract a payee from remittance information
func extractPayeeFromRemittanceInfo(ustrd string, cdtDbtInd string) string {
	// Different patterns for different transaction types
	var payeePatterns []*regexp.Regexp
	
	if cdtDbtInd == "CRDT" {
		// For credits, look for patterns indicating who paid us
		payeePatterns = []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:payment from|received from|sender|origin)[:\s]+([A-Za-z0-9\s\.\-&]+)`),
			regexp.MustCompile(`(?i)([A-Za-z0-9\s\.\-&]{3,})\s+(?:AG|SA|GmbH|LLC|Inc\.)`),
		}
	} else {
		// For debits, look for patterns indicating who we paid
		payeePatterns = []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:payment to|paid to|recipient|beneficiary)[:\s]+([A-Za-z0-9\s\.\-&]+)`),
			regexp.MustCompile(`(?i)([A-Za-z0-9\s\.\-&]{3,})\s+(?:AG|SA|GmbH|LLC|Inc\.)`),
		}
	}
	
	// Also try generic name patterns
	genericPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:Mr\.|Mrs\.|Ms\.|Dr\.)\s+([A-Za-z\s\.\-]+)`),
		regexp.MustCompile(`(?i)([A-Za-z\s\.\-]{2,})\s+(?:SA|AG|GmbH|Inc|LLC|Ltd)`),
	}
	
	payeePatterns = append(payeePatterns, genericPatterns...)
	
	for _, pattern := range payeePatterns {
		matches := pattern.FindStringSubmatch(ustrd)
		if len(matches) > 1 {
			// The first captured group should be the name
			payee := strings.TrimSpace(matches[1])
			if payee != "" {
				return payee
			}
		}
	}
	
	return ""
}

// extractFallbackPayee provides a fallback payee when none can be extracted from the transaction details
func extractFallbackPayee(entry Ntry, description string) string {
	// If no description or it's very generic, use transaction type
	if description == "" || description == "TRANSACTION" || description == "PAYMENT" {
		if entry.CdtDbtInd == "DBIT" {
			return "Outgoing Payment"
		} else {
			return "Incoming Payment"
		}
	}
	
	// Check for common payment identifiers
	paymentTerms := []string{"PMT", "CARTE", "TWINT", "PAYMENT", "TRANSFER"}
	for _, term := range paymentTerms {
		if strings.Contains(strings.ToUpper(description), term) {
			return description
		}
	}
	
	// Just use the description directly if it's reasonably short
	if len(description) < 80 {
		return description
	}
	
	// Otherwise take a reasonable substring to avoid overly long payees
	return description[:75] + "..."
}

// extractBookkeepingNo attempts to extract a bookkeeping number from remittance info
func extractBookkeepingNo(info string) string {
	// Look for patterns that might be bookkeeping numbers
	// This is a simple implementation and might need refinement based on actual data
	
	// Check for numeric sequences that might be bookkeeping numbers
	numericPattern := regexp.MustCompile(`\b\d{5,10}\b`)
	matches := numericPattern.FindStringSubmatch(info)
	if len(matches) > 0 {
		return matches[0]
	}
	
	// Check for patterns like "No. 12345" or "Ref: 12345"
	refPattern := regexp.MustCompile(`(?i)(No\.|Ref:?|Reference:?|Booking:?)\s*(\d+)`)
	matches = refPattern.FindStringSubmatch(info)
	if len(matches) > 2 {
		return matches[2]
	}
	
	return ""
}

// extractFund attempts to extract fund information from remittance info
func extractFund(info string) string {
	// Look for patterns that might indicate fund information
	// This is a simple implementation and might need refinement based on actual data
	
	// Check for patterns like "Fund: XYZ" or "Investment in XYZ"
	fundPattern := regexp.MustCompile(`(?i)(Fund:?|Investment in)\s*([A-Za-z0-9\s]+)`)
	matches := fundPattern.FindStringSubmatch(info)
	if len(matches) > 2 {
		return strings.TrimSpace(matches[2])
	}
	
	return ""
}

// cleanDescription cleans up the remittance info to create a better description
func cleanDescription(info string) string {
	// Remove any excessively long numeric sequences
	cleaned := regexp.MustCompile(`\b\d{10,}\b`).ReplaceAllString(info, "")
	
	// Remove any reference prefixes
	cleaned = regexp.MustCompile(`(?i)(Ref:?|Reference:?|No\.|Booking:?)\s*\d+`).ReplaceAllString(cleaned, "")
	
	// Trim spaces and remove duplicate spaces
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	
	return cleaned
}

// formatDate formats a date string from YYYY-MM-DD to a more readable format
func formatDate(date string) string {
	if date == "" {
		return ""
	}

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date // Return original if parsing fails
	}

	return t.Format("Jan 02, 2006")
}

// BatchConvert converts all XML files in a directory to CSV files
func BatchConvert(inputDir, outputDir string) (int, error) {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return 0, fmt.Errorf("error creating output directory: %w", err)
		}
	}

	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			continue
		}

		inputPath := filepath.Join(inputDir, file.Name())
		outputPath := filepath.Join(outputDir, strings.TrimSuffix(file.Name(), ".xml")+".csv")

		err = ConvertXMLToCSV(inputPath, outputPath)
		if err != nil {
			return count, fmt.Errorf("error converting %s: %w", file.Name(), err)
		}

		count++
	}

	return count, nil
}

// ValidateCAMT053 validates if an XML file is a valid CAMT.053 format
func ValidateCAMT053(xmlFile string) (bool, error) {
	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		return false, fmt.Errorf("error reading XML file: %w", err)
	}

	var camt053 CAMT053
	err = xml.Unmarshal(xmlData, &camt053)
	if err != nil {
		return false, nil // Not a valid XML or not in CAMT.053 format
	}

	// Check if it has the required structure for CAMT.053
	if camt053.BkToCstmrStmt.Stmt.Id == "" {
		return false, nil
	}

	return true, nil
}
