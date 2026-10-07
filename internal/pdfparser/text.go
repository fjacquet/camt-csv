package pdfparser

import (
	"fjacquet/camt-csv/internal/models"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
)

// extractAmount extracts the amount from a transaction line
func extractAmount(text string) (string, decimal.Decimal, bool) {
	// Match patterns like "123.45 USD" or "USD 123.45"
	amountMatch := amountGenericPattern.FindString(text)

	if amountMatch != "" {
		amountStr := amountMatch
		// Default to debit unless explicitly marked as credit or has a plus sign
		isCredit := strings.Contains(strings.ToLower(text), "credit") ||
			strings.Contains(strings.ToLower(text), "incoming") ||
			strings.Contains(text, "+")
		decimalAmount := models.ParseAmount(amountStr)
		return amountStr, decimalAmount, isCredit
	}

	return "", decimal.Zero, false
}

// cleanDescription removes unwanted elements from the description
func cleanDescription(description string) string {
	// Replace multiple spaces with a single space
	description = multipleSpacePattern.ReplaceAllString(description, " ")

	// Remove leading/trailing whitespace
	description = strings.TrimSpace(description)

	// Remove common noise phrases using pre-compiled patterns
	for _, pattern := range noisePhrasePatterns {
		description = pattern.ReplaceAllString(description, "")
	}

	return description
}

// extractPayee extracts the payee/merchant from a description
func extractPayee(description string) string {
	// Try pre-compiled payee patterns
	for _, re := range payeePatterns {
		matches := re.FindStringSubmatch(description)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// If no pattern matches, try to extract the most likely merchant name
	words := strings.Fields(description)

	// Skip transaction-related terms
	skipWords := map[string]bool{
		"transaction": true, "date": true, "amount": true, "credit": true,
		"debit": true, "card": true, "payment": true, "transfer": true,
		"fee": true, "charge": true, "balance": true, "available": true,
		"withdrawal": true, "deposit": true, "reference": true,
	}

	// Find first sequence of capitalized words
	var merchantWords []string
	inMerchant := false

	for _, word := range words {
		word = strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})

		if word == "" || skipWords[strings.ToLower(word)] {
			continue
		}

		// Check if word is mostly uppercase or a proper noun
		isSignificant := strings.ToUpper(word) == word || (unicode.IsUpper(rune(word[0])) && strings.ToLower(word[1:]) == word[1:])

		if isSignificant {
			merchantWords = append(merchantWords, word)
			inMerchant = true
		} else if inMerchant {
			// Stop collecting words once we hit non-significant words after starting collection
			break
		}
	}

	if len(merchantWords) > 0 {
		return strings.Join(merchantWords, " ")
	}

	// If all else fails, return a shortened version of the description
	if len(description) > 30 {
		return description[:30] + "..."
	}
	return description
}

// extractMerchant extracts the merchant from a description
func extractMerchant(description string) string {
	// Try pre-compiled merchant patterns
	for _, re := range merchantPatterns {
		matches := re.FindStringSubmatch(description)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Check for common transaction patterns and extract merchant name
	words := strings.Fields(description)
	for i, word := range words {
		if strings.HasPrefix(strings.ToLower(word), "card") && i+1 < len(words) {
			// "Card purchase at MERCHANT"
			if strings.ToLower(word) == "card" && i+2 < len(words) &&
				(strings.ToLower(words[i+1]) == "purchase" || strings.ToLower(words[i+1]) == "payment") &&
				strings.ToLower(words[i+2]) == "at" && i+3 < len(words) {
				return strings.Join(words[i+3:min(i+6, len(words))], " ")
			}
		}
	}

	return ""
}

// preProcessText performs initial cleanup and restructuring of PDF text
func preProcessText(text string) string {
	// Replace non-standard line breaks and ensure proper line splitting
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Replace multiple consecutive spaces with a single space
	// But preserve alignment for amount values which are typically right-aligned
	text = tripleSpacePattern.ReplaceAllString(text, "   ")

	// Remove empty lines
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	// Join the lines back together
	return strings.Join(lines, "\n")
}

// containsMerchantIdentifier checks if a line contains common merchant identifiers
func containsMerchantIdentifier(line string) bool {
	identifiers := []string{
		"merchant", "vendor", "shop", "store", "payee name",
		"business", "company", "paid to", "payment to",
	}

	lowerLine := strings.ToLower(line)
	for _, id := range identifiers {
		if strings.Contains(lowerLine, id) {
			return true
		}
	}

	// Look for "Card purchase at" pattern
	if cardPurchasePattern.MatchString(line) {
		return true
	}

	return false
}

// containsAmount checks if a line contains a monetary amount
func containsAmount(line string) bool {
	return amountCurrencyPattern.MatchString(line)
}
