// Package camtparser provides functionality to parse CAMT.053 XML files and convert them to CSV format.
package camtparser

import (
	"regexp"
	"strings"

	"fjacquet/camt-csv/internal/models"
)

// extractTransactions extracts all transactions from a CAMT.053 document.
// It processes each statement and entry to create Transaction structs.
func extractTransactions(camt053 *models.CAMT053) []models.Transaction {
	var transactions []models.Transaction

	// Process the statement in the CAMT.053 document
	stmt := camt053.BkToCstmrStmt.Stmt
	
	// Extract account information if available
	iban := ""
	// Check if IBAN is present in the account information
	if stmt.Acct.Id.IBAN != "" {
		iban = stmt.Acct.Id.IBAN
	}

	// Process each entry (transaction) in the statement
	for _, entry := range stmt.Ntry {
		// Extract basic information
		amount := entry.Amt.Text // Use Text field for the amount value
		currency := entry.Amt.Ccy
		cdtDbtInd := entry.CdtDbtInd
		
		bookingDate := entry.BookgDt.Dt
		valueDate := entry.ValDt.Dt
		status := entry.Sts
		
		// Format bank transaction code
		bankTxCode := formatBankTxCode(entry.BkTxCd)
		
		// Extract reference
		references := ""
		for _, txDetail := range entry.NtryDtls.TxDtls {
			if ref := extractReference(txDetail.Refs); ref != "" {
				references = ref
				break
			}
		}
		
		// Initialize variables for payer/payee information
		payee := ""
		payer := ""
		description := ""
		
		// Process transaction details for better descriptions and party information
		if len(entry.NtryDtls.TxDtls) > 0 {
			// Use the first transaction detail for remittance information
			txDetail := entry.NtryDtls.TxDtls[0]
			
			// Extract remittance information for description
			if len(txDetail.RmtInf.Ustrd) > 0 {
				// Join all unstructured remittance info lines
				originalDescription := strings.Join(txDetail.RmtInf.Ustrd, " ")
				
				// Check if this is a card or TWINT payment to extract merchant name as payee
				merchantName := ""
				if strings.HasPrefix(originalDescription, "PMT CARTE ") {
					merchantName = strings.TrimSpace(strings.TrimPrefix(originalDescription, "PMT CARTE "))
					if merchantName != "" {
						payee = merchantName
					}
				} else if strings.HasPrefix(originalDescription, "PMT TWINT ") {
					merchantName = strings.TrimSpace(strings.TrimPrefix(originalDescription, "PMT TWINT "))
					if merchantName != "" {
						payee = merchantName
					}
				}
				
				// Clean the description after extracting the merchant name if applicable
				description = cleanDescription(originalDescription)
				
				// Only try to extract payee from remittance info if we haven't already found a merchant name
				if payee == "" {
					extractedPayee := extractPayeeFromRemittanceInfo(description, cdtDbtInd)
					if extractedPayee != "" {
						payee = extractedPayee
					}
				}
			}
			
			// Extract party information from RltdPties if available
			// Check debtor name
			debtorName := txDetail.RltdPties.Dbtr.Nm
			if debtorName != "" {
				payer = debtorName
			}
			
			// Check creditor name - but don't override merchant name if we found one
			creditorName := txDetail.RltdPties.Cdtr.Nm
			if creditorName != "" && payee == "" {
				payee = creditorName
			}
		}
		
		// If no description was found, use additional entry info
		if description == "" && entry.AddtlNtryInf != "" {
			description = cleanDescription(entry.AddtlNtryInf)
		}
		
		// If still no payee, use fallback method
		if payee == "" {
			payee = extractFallbackPayee(entry, description)
		}
		
		// Extract bookkeeping number and fund from description if available
		bookkeepingNo := extractBookkeepingNo(description)
		fund := extractFund(description)
		
		// Create the transaction object
		tx := models.Transaction{
			Date:             bookingDate,
			ValueDate:        valueDate,
			Description:      description,
			BookkeepingNo:    bookkeepingNo,
			Fund:             fund,
			Amount:           amount,
			Currency:         currency,
			CreditDebit:      cdtDbtInd,
			EntryReference:   references,
			AccountServicer:  entry.AcctSvcrRef,
			BankTxCode:       bankTxCode,
			Status:           status,
			Payee:            payee,
			Payer:            payer,
			IBAN:             iban,
			Category:         "", // Will be determined later during categorization
		}
		
		transactions = append(transactions, tx)
	}

	return transactions
}

// formatBankTxCode formats the bank transaction code in a human-readable format.
func formatBankTxCode(bkTxCd models.BkTxCd) string {
	// Check if domain code is available
	domainCode := bkTxCd.Domn.Cd
	if domainCode == "" {
		return ""
	}
	
	// Check if family code is available
	familyCode := bkTxCd.Domn.Fmly.Cd
	if familyCode == "" {
		return domainCode
	}
	
	// Format as DOMAIN/FAMILY/SUBFAMILY
	subFamilyCode := bkTxCd.Domn.Fmly.SubFmlyCd
	return domainCode + "/" + familyCode + "/" + subFamilyCode
}

// extractReference gets the best reference from the available options.
func extractReference(refs models.Refs) string {
	// Use end-to-end ID as primary reference
	if refs.EndToEndId != "" {
		return refs.EndToEndId
	}
	
	// Fall back to transaction ID
	if refs.TxId != "" {
		return refs.TxId
	}
	
	// Fall back to instruction ID
	if refs.InstrId != "" {
		return refs.InstrId
	}
	
	// Fall back to message ID
	return refs.MsgId
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

// extractFallbackPayee provides a fallback payee when none can be extracted from the transaction details.
func extractFallbackPayee(entry models.Ntry, description string) string {
	// Extract payee from wire transfers (VIRT BANC, VIR TWINT)
	if strings.HasPrefix(description, "VIRT BANC ") {
		return strings.TrimSpace(strings.TrimPrefix(description, "VIRT BANC "))
	}
	
	if strings.HasPrefix(description, "VIR TWINT ") {
		return strings.TrimSpace(strings.TrimPrefix(description, "VIR TWINT "))
	}
	
	// If this is a card payment or TWINT payment, extract merchant from description
	if strings.Contains(description, "CARTE") || strings.Contains(description, "TWINT") {
		return description
	}
	
	// Check for common patterns
	patterns := []string{
		"COOP-", 
		"MIG ", 
		"MIGROS-", 
		"MIGROLINO", 
		"DENNER", 
		"ALDI", 
		"LIDL",
		"MANOR",
		"MAMMUT",
		"OCHSNER",
		"SUSHI",
		"PIZZERIA",
		"CAFE",
		"KIOSK",
		"KEBAB",
		"BOUCHERIE",
		"BOULANGERIE",
		"PISCINE",
		"SPA",
		"PRESSING",
	}
	
	for _, pattern := range patterns {
		if idx := strings.Index(strings.ToUpper(description), strings.ToUpper(pattern)); idx >= 0 {
			// Extract a reasonable length substring as the merchant name
			startIdx := idx
			endIdx := len(description)
			
			// Try to find a natural endpoint (space, comma, etc.)
			for i := idx; i < len(description) && i < idx+30; i++ {
				if description[i] == ' ' && i > idx+10 {
					endIdx = i
					break
				}
			}
			
			return strings.TrimSpace(description[startIdx:endIdx])
		}
	}
	
	// If nothing matches, use the entry type to determine party type
	bankTxCode := formatBankTxCode(entry.BkTxCd)
	
	if strings.Contains(bankTxCode, "CCRD") || strings.Contains(bankTxCode, "POSD") {
		return description // For card payments, use the description as merchant
	}
	
	if strings.Contains(bankTxCode, "CWDL") {
		return "ATM Withdrawal" // For ATM withdrawals
	}
	
	if strings.Contains(bankTxCode, "ICDT") || strings.Contains(bankTxCode, "DMCT") || 
	   strings.Contains(bankTxCode, "AUTT") || strings.Contains(bankTxCode, "RCDT") {
		// For transfers (BCV-NET, etc.)
		if strings.Contains(strings.ToUpper(description), "BCV") {
			parts := strings.Split(description, " ")
			if len(parts) > 2 {
				return strings.Join(parts[1:], " ") // For BCV-NET transfers, use the recipient
			}
		}
		
		// Look for recipient name in transfer description
		transferPatternsRe := regexp.MustCompile(`(?i)VIRT|VIR|TRANSFERT|ORDRE|CR`)
		if transferPatternsRe.MatchString(description) {
			descParts := strings.Split(description, " ")
			if len(descParts) >= 3 {
				// Skip the first part (VIRT, VIR, etc.) and use the rest as the payee
				return strings.Join(descParts[1:], " ")
			}
		}
	}
	
	// Check for transaction details with party information
	for _, txDetail := range entry.NtryDtls.TxDtls {
		// For debit transactions, the creditor is the payee
		creditorName := txDetail.RltdPties.Cdtr.Nm
		if entry.CdtDbtInd == "DBIT" && creditorName != "" {
			return creditorName
		}
		
		// For credit transactions, the debtor is the payer (which we use as payee in display)
		debtorName := txDetail.RltdPties.Dbtr.Nm
		if entry.CdtDbtInd == "CRDT" && debtorName != "" {
			return debtorName
		}
	}
	
	return "Unknown Payee"
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
