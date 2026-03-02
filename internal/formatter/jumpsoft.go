package formatter

import (
	"fjacquet/camt-csv/internal/models"
)

// JumpsoftFormatter produces 7-column comma-delimited output compatible with
// Jumpsoft Money CSV import. Columns: Date,Description,Amount,Currency,Category,Type,Notes
type JumpsoftFormatter struct{}

// NewJumpsoftFormatter creates a new JumpsoftFormatter instance.
func NewJumpsoftFormatter() *JumpsoftFormatter {
	return &JumpsoftFormatter{}
}

// Header returns the 7 Jumpsoft Money column names.
func (f *JumpsoftFormatter) Header() []string {
	return []string{"Date", "Description", "Amount", "Currency", "Category", "Type", "Notes"}
}

// Format converts transactions to Jumpsoft Money-compatible CSV rows.
// Date format: YYYY-MM-DD (ISO 8601, e.g., "2026-02-15")
// Amount: signed decimal — negative for debits, positive for credits
// Category: from tx.Category; falls back to "Uncategorized" if empty
// Notes: from tx.RemittanceInfo if set, otherwise tx.Description
func (f *JumpsoftFormatter) Format(transactions []models.Transaction) ([][]string, error) {
	rows := make([][]string, 0, len(transactions))

	for _, tx := range transactions {
		// Date: YYYY-MM-DD (ISO 8601)
		dateStr := ""
		if !tx.Date.IsZero() {
			dateStr = tx.Date.Format("2006-01-02")
		}

		// Description: prefer tx.Description, fall back to tx.Name
		description := tx.Description
		if description == "" {
			description = tx.Name
		}

		// Amount: signed — negative for debits, positive for credits
		// Transaction.Amount already carries sign from parsers (negative = debit)
		// If amount is zero but DebitFlag is set, negate it
		amount := tx.Amount
		if tx.DebitFlag && amount.IsPositive() {
			amount = amount.Neg()
		}
		amountStr := amount.StringFixed(2)

		// Currency
		currency := tx.Currency

		// Category: fall back to "Uncategorized" if empty
		category := tx.Category
		if category == "" {
			category = "Uncategorized"
		}

		// Type: from tx.Type
		txType := tx.Type

		// Notes: from tx.RemittanceInfo if set, otherwise tx.Description
		notes := tx.RemittanceInfo
		if notes == "" {
			notes = tx.Description
		}

		rows = append(rows, []string{dateStr, description, amountStr, currency, category, txType, notes})
	}

	return rows, nil
}

// Delimiter returns comma as the delimiter for Jumpsoft Money format.
func (f *JumpsoftFormatter) Delimiter() rune {
	return ','
}
