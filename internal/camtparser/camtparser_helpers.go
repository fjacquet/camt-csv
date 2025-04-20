// Package camtparser provides functionality to parse CAMT.053 XML files and convert them to CSV format.
package camtparser

import (
	"os"
	"gopkg.in/xmlpath.v2"
	"regexp"
	"strings"

	"fjacquet/camt-csv/internal/models"
)

// extractTransactionsFromXMLPath uses XPath to extract transactions from a CAMT.053 XML file.
func extractTransactionsFromXMLPath(xmlFilePath string) ([]models.Transaction, error) {
	amounts, _ := extractWithXPath(xmlFilePath, "//Ntry/Amt")
	currencies, _ := extractWithXPath(xmlFilePath, "//Ntry/Amt/@Ccy")
	cdtDbtInds, _ := extractWithXPath(xmlFilePath, "//Ntry/CdtDbtInd")
	bookingDates, _ := extractWithXPath(xmlFilePath, "//Ntry/BookgDt/Dt")
	valueDates, _ := extractWithXPath(xmlFilePath, "//Ntry/ValDt/Dt")
	statuses, _ := extractWithXPath(xmlFilePath, "//Ntry/Sts")
	refs, _ := extractWithXPath(xmlFilePath, "//Ntry/NtryDtls/TxDtls/Refs/EndToEndId")
	altRefs, _ := extractWithXPath(xmlFilePath, "//Ntry/NtryDtls/TxDtls/Refs/TxId")
	accSvcRefs, _ := extractWithXPath(xmlFilePath, "//Ntry/AcctSvcrRef")
	ustrdTexts, _ := extractWithXPath(xmlFilePath, "//Ntry/NtryDtls/TxDtls/RmtInf/Ustrd")
	payers, _ := extractWithXPath(xmlFilePath, "//Ntry/NtryDtls/TxDtls/RltdPties/Dbtr/Nm")
	payees, _ := extractWithXPath(xmlFilePath, "//Ntry/NtryDtls/TxDtls/RltdPties/Cdtr/Nm")
	ibans, _ := extractWithXPath(xmlFilePath, "//BkToCstmrStmt/Stmt/Acct/Id/IBAN")
	bankTxDomains, _ := extractWithXPath(xmlFilePath, "//Ntry/BkTxCd/Domn/Cd")
	bankTxFamilies, _ := extractWithXPath(xmlFilePath, "//Ntry/BkTxCd/Domn/Fmly/Cd")
	bankTxSubFamilies, _ := extractWithXPath(xmlFilePath, "//Ntry/BkTxCd/Domn/Fmly/SubFmlyCd")
	
	// Figure out how many transactions to create
	n := len(amounts)
	if n == 0 {
		return []models.Transaction{}, nil
	}
	
	// For simplicity, get the first IBAN if available
	iban := ""
	if len(ibans) > 0 {
		iban = ibans[0]
	}
	
	// Build transactions
	transactions := make([]models.Transaction, 0, n)
	for i := 0; i < n; i++ {
		// Generate a better description from remittance info
		remittanceInfo := getOrEmpty(ustrdTexts, i)
		description := cleanDescription(remittanceInfo)
		
		// Extract additional data from remittance info
		bookkeepingNo := extractBookkeepingNo(remittanceInfo)
		fund := extractFund(remittanceInfo)
		
		// Determine payee
		payee := getOrEmpty(payees, i)
		payer := getOrEmpty(payers, i)
		
		// If no explicit payee, try to extract from remittance info
		if payee == "" {
			// For DBIT transactions, extract payee; for CRDT transactions, extract payer
			payee = extractPayeeFromRemittanceInfo(remittanceInfo, getOrEmpty(cdtDbtInds, i))
			
			// If still no payee, try to extract from description
			if payee == "" {
				payee = extractMerchantFromDescription(description)
			}
		}
		
		// Format bank transaction code if available
		bankTxCode := formatBankTxCodeFromParts(
			getOrEmpty(bankTxDomains, i),
			getOrEmpty(bankTxFamilies, i),
			getOrEmpty(bankTxSubFamilies, i),
		)
		
		// Get the best reference
		reference := getOrEmpty(refs, i)
		if reference == "" || reference == "NOTPROVIDED" {
			reference = getOrEmpty(altRefs, i)
		}
		
		tx := models.Transaction{
			Amount:           models.ParseAmount(getOrEmpty(amounts, i)),
			Currency:         getOrEmpty(currencies, i),
			CreditDebit:      getOrEmpty(cdtDbtInds, i),
			Date:             getOrEmpty(bookingDates, i),
			ValueDate:        getOrEmpty(valueDates, i),
			Status:           getOrEmpty(statuses, i),
			Description:      description,
			Payee:            payee,
			Payer:            payer,
			EntryReference:   reference,
			AccountServicer:  getOrEmpty(accSvcRefs, i),
			BankTxCode:       bankTxCode,
			BookkeepingNo:    bookkeepingNo,
			Fund:             fund,
			IBAN:             iban,
			Category:         "", // Will be determined later during categorization
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

// getOrEmpty returns the value at index i if present, otherwise "".
func getOrEmpty(slice []string, i int) string {
	if i < len(slice) {
		return slice[i]
	}
	return ""
}

// formatBankTxCodeFromParts formats the bank transaction code from its component parts.
func formatBankTxCodeFromParts(domain, family, subfamily string) string {
	if domain == "" {
		return ""
	}
	
	if family == "" {
		return domain
	}
	
	return domain + "/" + family + "/" + subfamily
}

// extractPayeeFromRemittanceInfo tries to extract a payee from remittance information.
func extractPayeeFromRemittanceInfo(ustrd string, cdtDbtInd string) string {
	// Common patterns for payee information in remittance texts
	patterns := []string{
		`(?i)Payee:\s*([^\n]+)`,
		`(?i)Beneficiary:\s*([^\n]+)`,
		`(?i)Payment to:\s*([^\n]+)`,
		`(?i)In favor of:\s*([^\n]+)`,
		`(?i)To:\s*([^\n]+)`,
		`(?i)Recipient:\s*([^\n]+)`,
	}
	
	// For credit indicators (money coming in), adjust patterns
	if cdtDbtInd == "CRDT" {
		patterns = []string{
			`(?i)Payer:\s*([^\n]+)`,
			`(?i)Originator:\s*([^\n]+)`,
			`(?i)Payment from:\s*([^\n]+)`,
			`(?i)From:\s*([^\n]+)`,
			`(?i)Sender:\s*([^\n]+)`,
		}
	}
	
	// Try each pattern
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(ustrd)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

// extractBookkeepingNo attempts to extract a bookkeeping number from remittance info.
func extractBookkeepingNo(info string) string {
	patterns := []string{
		`Booking No\.?\s*:\s*([A-Z0-9-]+)`,
		`Ref\.?\s*:\s*([A-Z0-9-]+)`,
		`Reference\.?\s*:\s*([A-Z0-9-]+)`,
		`Booking Number\.?\s*:\s*([A-Z0-9-]+)`,
		`Booking Ref\.?\s*:\s*([A-Z0-9-]+)`,
		`Booking\.?\s*:\s*([A-Z0-9-]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(info)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

// extractFund attempts to extract fund information from remittance info.
func extractFund(info string) string {
	patterns := []string{
		`Fund:\s*([^,;]+)`,
		`Investment Fund:\s*([^,;]+)`,
		`Portfolio:\s*([^,;]+)`,
		`(?i)fund[:\s]+([^,;]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(info)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

// cleanDescription cleans up the remittance info to create a better description.
func cleanDescription(info string) string {
	// Remove excessive whitespace
	info = regexp.MustCompile(`\s+`).ReplaceAllString(info, " ")
	
	// Remove common prefixes
	prefixes := []string{
		`(?i)^remittance info:`,
		`(?i)^payment info:`,
		`(?i)^additional info:`,
		`(?i)^transaction details:`,
		`(?i)^details:`,
		`(?i)^PMT CARTE `,  // Remove PMT CARTE prefix
		`(?i)^PMT TWINT `,  // Remove PMT TWINT prefix
		`(?i)^VIRT BANC `,  // Remove VIRT BANC prefix
		`(?i)^VIR TWINT `,  // Remove VIR TWINT prefix
	}
	
	for _, prefix := range prefixes {
		re := regexp.MustCompile(prefix)
		info = re.ReplaceAllString(info, "")
	}
	
	// Remove technical keys often found in CAMT descriptions
	info = regexp.MustCompile(`(?i)end-to-end-id:[^\s]+`).ReplaceAllString(info, "")
	info = regexp.MustCompile(`(?i)instruction-id:[^\s]+`).ReplaceAllString(info, "")
	
	return strings.TrimSpace(info)
}

// extractMerchantFromDescription attempts to extract a merchant name from description
// especially for card and TWINT payments where the description contains the merchant name
func extractMerchantFromDescription(description string) string {
	// Check for card payment patterns
	if strings.HasPrefix(strings.ToUpper(description), "PMT CARTE ") {
		return strings.TrimSpace(strings.TrimPrefix(description, "PMT CARTE "))
	}
	
	// Check for TWINT payment patterns
	if strings.HasPrefix(strings.ToUpper(description), "PMT TWINT ") {
		return strings.TrimSpace(strings.TrimPrefix(description, "PMT TWINT "))
	}
	
	return ""
}

// extractWithXPath extracts all string values matching an XPath expression from an XML file.
func extractWithXPath(xmlFilePath, xpath string) ([]string, error) {
	f, err := os.Open(xmlFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	root, err := xmlpath.Parse(f)
	if err != nil {
		return nil, err
	}
	path := xmlpath.MustCompile(xpath)
	iter := path.Iter(root)
	var results []string
	for iter.Next() {
		results = append(results, iter.Node().String())
	}
	return results, nil
}

// Example usage: extract all <Ntry><Amt> values from a CAMT.053 XML file
// amounts, err := extractWithXPath("/path/to/file.xml", "//Ntry/Amt")
// if err == nil {
//     for _, amt := range amounts {
//         fmt.Println("Amount:", amt)
//     }
// }
