// Package textutils provides text extraction and manipulation utilities.
package textutils

import (
	"regexp"
	"strings"
)

// ExtractBookkeepingNumber attempts to extract a bookkeeping number from remittance info.
func ExtractBookkeepingNumber(info string) string {
	patterns := []string{
		`BookKeeping Number:[\s]*([\d-]+)`,
		`Booking no:[\s]*([\d-]+)`,
		`No booking:[\s]*([\d-]+)`,
		`Reference:[\s]*([\d-]+)`,
	}
	
	for _, pattern := range patterns {
		matches := regexp.MustCompile(pattern).FindStringSubmatch(info)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// ExtractFund attempts to extract fund information from remittance info.
func ExtractFund(info string) string {
	patterns := []string{
		`Fund:[\s]*([^,;]+)`,
		`Investment Fund:[\s]*([^,;]+)`,
	}
	
	for _, pattern := range patterns {
		matches := regexp.MustCompile(pattern).FindStringSubmatch(info)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// ExtractPayee tries to extract a payee from remittance information.
func ExtractPayee(ustrd string) string {
	// Common patterns for payees in remittance info
	patterns := []string{
		`Payee:\s*([^,;]+)`,
		`Bénéficiaire:\s*([^,;]+)`,
		`Recipient:\s*([^,;]+)`,
		`To:\s*([^,;]+)`,
		`Payment to:\s*([^,;]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(ustrd)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

// ExtractMerchant attempts to extract a merchant name from description
// especially for card and TWINT payments where the description contains the merchant name
func ExtractMerchant(description string) string {
	description = strings.ToLower(description)
	
	// Common patterns in card and TWINT payments
	if strings.Contains(description, "card purchase") || 
	   strings.Contains(description, "twint") {
		
		// Extract merchant name after "at" or similar terms
		patterns := []string{
			`(?i)at\s+(.+?)(?:\s+on|$)`,
			`(?i)chez\s+(.+?)(?:\s+le|$)`,
			`(?i)auprès de\s+(.+?)(?:\s+le|$)`,
		}
		
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(description)
			if len(matches) > 1 {
				return strings.TrimSpace(matches[1])
			}
		}
	}
	
	return ""
}

// ExtractFundInfo extracts fund information from text
// For example, "UBS FUND GLOBAL EQUITY" or "VANGUARD ETF S&P 500"
func ExtractFundInfo(text string) string {
	if text == "" {
		return ""
	}

	fundPattern := regexp.MustCompile(`(?i)((?:UBS|PICTET|CS|CREDIT SUISSE|VONTOBEL|SWISSCANTO|BLACKROCK|FIDELITY|VANGUARD|AMUNDI|ISHARES|JPM|JP MORGAN)\s+(?:FUND|ETF|INDEX|STRATEGIE|MONEY MARKET|BOND|EQUITY|BALANCED|ALLOCATION|ESG|MSCI|S&P|NASDAQ|DOW|GOLD|OIL|GLOBAL|EUROPE|EMERGING|FRONTIER)s?\s*[-A-Z0-9.\s]{1,30})`)
	matches := fundPattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		// Clean up and normalize the fund name
		fund := strings.TrimSpace(matches[1])
		fund = strings.ToUpper(fund)
		return fund
	}

	return ""
}

// FormatBankTxCode formats the bank transaction code from its component parts.
func FormatBankTxCode(domain, family, subfamily string) string {
	if domain == "" {
		return ""
	}
	
	code := domain
	if family != "" {
		code += "." + family
		if subfamily != "" {
			code += "." + subfamily
		}
	}
	
	return code
}
