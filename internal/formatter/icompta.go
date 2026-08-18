package formatter

import (
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
)

// iComptaFormatter produces semicolon-delimited output compatible with iCompta's
// CSV import plugins. It projects Transaction fields to match the schema expected
// by iCompta (see .planning/reference/icompta-schema.sql).
type iComptaFormatter struct{}

// NewIComptaFormatter creates a new iComptaFormatter instance.
func NewIComptaFormatter() *iComptaFormatter {
	return &iComptaFormatter{}
}

// Header returns the iCompta column names.
//
// The first ten columns are the original layout and must keep their names and
// positions: existing import plugins resolve columns by name, so renaming or
// reordering them silently breaks every configured import. The trailing columns
// carry the investment and identity fields the plugins reference — see
// docs/icompta-plugin-setup.md and the coverage test in icompta_plugins_test.go.
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
		"ValueDate",
		"CreditDebit",
		"BookkeepingNumber",
		"InvestmentType",
		"Fund",
		"NumberOfShares",
		"Fees",
		"Recipient",
		"Number",
		"TaxRate",
	}
}

// blankIfZeroShares renders a share count, or an empty cell when there is none.
// iCompta treats a literal "0" in an investment column as real data and creates
// phantom zero-share securities, so unset fields must be empty rather than zero.
// Shares keep their natural precision because brokers allocate fractional units.
func blankIfZeroShares(d decimal.Decimal) string {
	if d.IsZero() {
		return ""
	}
	return d.String()
}

// isInvestment reports whether a transaction is actually an investment.
//
// Transaction.Investment is not investment-only: UpdateInvestmentTypeFromLegacyField
// back-fills it from Type, so an ordinary Revolut card payment carries
// Investment="CARD_PAYMENT". The Revolut plugins map investmentTransactionInfo.type
// to InvestmentType, so emitting it unconditionally would import every card payment
// as an investment transaction. A security is what makes a row an investment.
func isInvestment(tx models.Transaction) bool {
	return tx.Fund != "" || !tx.NumberOfShares.IsZero()
}

// blankIfZeroAmount renders a monetary value with the same two-decimal contract
// as the Amount column, or an empty cell when there is nothing to report.
func blankIfZeroAmount(d decimal.Decimal) string {
	if d.IsZero() {
		return ""
	}
	return d.StringFixed(2)
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

		// InvestmentType is only meaningful on a row that names a security.
		investmentType := ""
		if isInvestment(tx) {
			investmentType = tx.Investment
		}

		// ValueDate: same dd.MM.yyyy contract as Date
		valueDateStr := ""
		if !tx.ValueDate.IsZero() {
			valueDateStr = tx.ValueDate.Format("02.01.2006")
		}

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
			valueDateStr,
			tx.CreditDebit,
			tx.BookkeepingNumber,
			investmentType,
			tx.Fund,
			blankIfZeroShares(tx.NumberOfShares),
			blankIfZeroAmount(tx.Fees),
			tx.Recipient,
			tx.Number,
			blankIfZeroAmount(tx.TaxRate),
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
