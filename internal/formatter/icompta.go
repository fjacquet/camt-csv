package formatter

import (
	"fjacquet/camt-csv/internal/models"
)

// iComptaFormatter produces 10-column semicolon-delimited output compatible with
// iCompta's CSV import plugins. It projects Transaction fields to match the schema
// expected by iCompta (see .planning/reference/icompta-schema.sql).
type iComptaFormatter struct{}

// NewIComptaFormatter creates a new iComptaFormatter instance.
func NewIComptaFormatter() *iComptaFormatter {
	return &iComptaFormatter{}
}

// Header returns the 10 iCompta column names.
func (f *iComptaFormatter) Header() []string {
	return []string{
		"Date",
		"Name",
		"Amount",
		"Description",
		"Status",
		"Category",
		"SplitAmount",
		"SplitAmountExclTax",
		"SplitTaxRate",
		"Type",
	}
}

// Format converts transactions to iCompta-compatible CSV rows.
// Date format: dd.MM.yyyy (e.g., "15.02.2026")
// Status mapping: BOOK/RCVD→"cleared", PDNG→"pending", REVD/CANC→"reverted", default→"cleared"
// Category: warns if empty, uses "Uncategorized" as fallback
func (f *iComptaFormatter) Format(transactions []models.Transaction) ([][]string, error) {
	rows := make([][]string, 0, len(transactions))

	for _, tx := range transactions {
		// Date: dd.MM.yyyy format
		dateStr := ""
		if !tx.Date.IsZero() {
			dateStr = tx.Date.Format("02.01.2006")
		}

		// Name: prefer tx.Name, fall back to PartyName
		name := tx.Name
		if name == "" {
			name = tx.PartyName
		}

		// Amount: always 2 decimal places
		amount := tx.Amount.StringFixed(2)

		// Description
		description := tx.Description

		// Status: map CAMT statuses to iCompta equivalents
		status := mapStatusToICompta(tx.Status)

		// Category: warn if empty, use "Uncategorized" as fallback
		category := tx.Category
		if category == "" {
			category = "Uncategorized"
			// TODO: Add logging for missing category warning
		}

		// SplitAmount: same as Amount for v1 (no split support yet)
		splitAmount := tx.Amount.StringFixed(2)

		// SplitAmountExclTax
		splitAmountExclTax := tx.AmountExclTax.StringFixed(2)

		// SplitTaxRate
		splitTaxRate := tx.TaxRate.StringFixed(2)

		// Type
		txType := tx.Type

		row := []string{
			dateStr,
			name,
			amount,
			description,
			status,
			category,
			splitAmount,
			splitAmountExclTax,
			splitTaxRate,
			txType,
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// Delimiter returns semicolon as the delimiter for iCompta format.
func (f *iComptaFormatter) Delimiter() rune {
	return ';'
}

// mapStatusToICompta converts CAMT status codes to iCompta status values.
// Mapping:
// - BOOK (Booked), RCVD (Received) → "cleared"
// - PDNG (Pending) → "pending"
// - REVD (Reverted), CANC (Cancelled) → "reverted"
// - Default → "cleared"
func mapStatusToICompta(camtStatus string) string {
	switch camtStatus {
	case "BOOK", "RCVD":
		return "cleared"
	case "PDNG":
		return "pending"
	case "REVD", "CANC":
		return "reverted"
	default:
		// Default to cleared for unknown statuses
		return "cleared"
	}
}
