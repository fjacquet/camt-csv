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

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/pkg/categorizer"
)

//------------------------------------------------------------------------------
// MAIN CONVERSION FUNCTIONS
//------------------------------------------------------------------------------

// ConvertXMLToCSV converts a CAMT.053 XML file to CSV format
func ConvertXMLToCSV(xmlFile string, csvFile string) error {
	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		return fmt.Errorf("error reading XML file: %w", err)
	}

	var camt053 models.CAMT053
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

	// Write CSV header
	header := []string{
		"Date", "Value Date", "Description", "Bookkeeping No.", "Fund", 
		"Amount", "Currency", "Credit/Debit", "Entry Reference", 
		"Account Servicer Ref", "Bank Transaction Code", "Status", 
		"Payee", "Payer", "IBAN", "Category",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Extract transactions and write to CSV
	transactions := extractTransactions(&camt053)
	for _, tx := range transactions {
		// Determine if the party is a debtor (payer) or creditor (payee) based on credit/debit indicator
		var partyName string
		var isDebtor bool
		
		if tx.CreditDebit == "CRDT" {
			// For credit entries (money coming IN), the party is a creditor (paying you)
			partyName = tx.Payer
			isDebtor = false
		} else {
			// For debit entries (money going OUT), the party is a debtor (you're paying them)
			partyName = tx.Payee
			isDebtor = true
		}
		
		// Try to categorize the transaction
		catTx := categorizer.Transaction{
			PartyName: partyName,
			IsDebtor:  isDebtor,
			Amount:    tx.Amount,
			Date:      tx.Date,
			Info:      tx.Description,
		}
		
		cat, err := categorizer.CategorizeTransaction(catTx)
		if err == nil {
			tx.Category = cat.Name
		} else {
			tx.Category = "Uncategorized"
		}

		record := []string{
			tx.Date,
			tx.ValueDate,
			tx.Description,
			tx.BookkeepingNo,
			tx.Fund,
			tx.Amount,
			tx.Currency,
			tx.CreditDebit,
			tx.EntryReference,
			tx.AccountServicer,
			tx.BankTxCode,
			tx.Status,
			tx.Payee,
			tx.Payer,
			tx.IBAN,
			tx.Category,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error writing CSV record: %w", err)
		}
	}

	return nil
}

// BatchConvert converts all XML files in a directory to CSV files
func BatchConvert(inputDir, outputDir string) (int, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, fmt.Errorf("error creating output directory: %w", err)
	}

	// Get all XML files in input directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return 0, fmt.Errorf("error reading input directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			continue
		}

		inputFile := filepath.Join(inputDir, file.Name())
		outputFile := filepath.Join(outputDir, strings.TrimSuffix(file.Name(), ".xml")+".csv")

		// Validate if the file is CAMT.053
		isValid, err := ValidateCAMT053(inputFile)
		if err != nil {
			fmt.Printf("Error validating %s: %v\n", file.Name(), err)
			continue
		}

		if !isValid {
			fmt.Printf("Skipping %s: Not a valid CAMT.053 file\n", file.Name())
			continue
		}

		// Convert file
		fmt.Printf("Converting %s to %s\n", file.Name(), filepath.Base(outputFile))
		if err := ConvertXMLToCSV(inputFile, outputFile); err != nil {
			fmt.Printf("Error converting %s: %v\n", file.Name(), err)
			continue
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

	var camt053 models.CAMT053
	err = xml.Unmarshal(xmlData, &camt053)
	if err != nil {
		return false, nil // Not a valid XML or not CAMT.053
	}

	// Check if it has the necessary CAMT.053 structure
	if camt053.BkToCstmrStmt.Stmt.Id == "" {
		return false, nil
	}

	return true, nil
}

//------------------------------------------------------------------------------
// TRANSACTION EXTRACTION AND PROCESSING
//------------------------------------------------------------------------------

// extractTransactions extracts all transactions from a CAMT.053 document
func extractTransactions(camt053 *models.CAMT053) []models.Transaction {
	var transactions []models.Transaction

	for _, entry := range camt053.BkToCstmrStmt.Stmt.Ntry {
		// Skip entries that are not booked
		if entry.Sts != "BOOK" {
			continue
		}

		// Process each transaction detail
		for _, txDtl := range entry.NtryDtls.TxDtls {
			// Extract remittance information
			var remittanceInfo string
			if len(txDtl.RmtInf.Ustrd) > 0 {
				remittanceInfo = strings.Join(txDtl.RmtInf.Ustrd, " ")
			}

			// Clean up description
			description := cleanDescription(remittanceInfo)
			
			// Extract payee from remittance info or related parties
			var payee string
			if entry.CdtDbtInd == "DBIT" && txDtl.RltdPties.Cdtr.Nm != "" {
				// For debit transactions, the creditor is the payee
				payee = txDtl.RltdPties.Cdtr.Nm
			} else if entry.CdtDbtInd == "CRDT" && txDtl.RltdPties.Dbtr.Nm != "" {
				// For credit transactions, the debtor is the payee
				payee = txDtl.RltdPties.Dbtr.Nm
			} else {
				// Try to extract payee from remittance information
				payee = extractPayeeFromRemittanceInfo(remittanceInfo, entry.CdtDbtInd)
			}

			// If no payee could be extracted, use a fallback
			if payee == "" {
				payee = extractFallbackPayee(entry, description)
			}

			// Normalize payee name: trim spaces, convert to lowercase for better matching
			if payee != "" {
				payee = strings.TrimSpace(payee)
			}

			// Extract payer information
			var payer string
			if entry.CdtDbtInd == "DBIT" {
				// For debit transactions, account owner is the payer
				payer = camt053.BkToCstmrStmt.Stmt.Acct.Ownr.Nm
			} else if txDtl.RltdPties.Dbtr.Nm != "" {
				payer = txDtl.RltdPties.Dbtr.Nm
			}

			// Ensure payer is not empty
			if payer == "" {
				payer = "JACQUET" // Default payer based on account owner
			}

			// Normalize payer name: trim spaces for better matching
			if payer != "" {
				payer = strings.TrimSpace(payer)
			}

			// Extract IBAN
			var iban string
			if entry.CdtDbtInd == "DBIT" && txDtl.RltdPties.CdtrAcct.Id.IBAN != "" {
				iban = txDtl.RltdPties.CdtrAcct.Id.IBAN
			} else if entry.CdtDbtInd == "CRDT" && txDtl.RltdPties.DbtrAcct.Id.IBAN != "" {
				iban = txDtl.RltdPties.DbtrAcct.Id.IBAN
			}

			// Format the bank transaction code
			bankTxCode := formatBankTxCode(entry.BkTxCd)

			// Format dates
			bookingDate := formatDate(entry.BookgDt.Dt)
			valueDate := formatDate(entry.ValDt.Dt)

			// Extract additional info from remittance info
			bookkeepingNo := extractBookkeepingNo(remittanceInfo)
			fund := extractFund(remittanceInfo)

			// Create transaction object
			tx := models.Transaction{
				Date:            bookingDate,
				ValueDate:       valueDate,
				Description:     description,
				BookkeepingNo:   bookkeepingNo,
				Fund:            fund,
				Amount:          entry.Amt.Text,
				Currency:        entry.Amt.Ccy,
				CreditDebit:     entry.CdtDbtInd,
				EntryReference:  entry.NtryRef,
				AccountServicer: entry.AcctSvcrRef,
				BankTxCode:      bankTxCode,
				Status:          entry.Sts,
				Payee:           payee,
				Payer:           payer,
				IBAN:            iban,
				Category:        "", // Will be set during categorization
			}

			transactions = append(transactions, tx)
		}
	}

	return transactions
}

//------------------------------------------------------------------------------
// TEXT EXTRACTION AND FORMATTING UTILITIES
//------------------------------------------------------------------------------

// formatBankTxCode formats the bank transaction code properly
func formatBankTxCode(bkTxCd models.BkTxCd) string {
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
func extractReference(refs models.Refs) string {
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
func extractFallbackPayee(entry models.Ntry, description string) string {
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
