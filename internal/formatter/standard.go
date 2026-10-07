package formatter

import (
	"fjacquet/camt-csv/internal/csvsafe"
	"fjacquet/camt-csv/internal/models"
)

// StandardFormatter produces the standard 29-column CSV format.
// This formatter maintains compatibility with existing camt-csv output,
// using comma delimiters and delegating to Transaction.MarshalCSV().
type StandardFormatter struct{}

// NewStandardFormatter creates a new StandardFormatter instance.
func NewStandardFormatter() *StandardFormatter {
	return &StandardFormatter{}
}

// Header returns the 29 standard column names.
func (f *StandardFormatter) Header() []string {
	return []string{
		"Status", "Date", "ValueDate", "Name", "PartyName", "PartyIBAN",
		"Description", "RemittanceInfo", "Amount", "CreditDebit", "Currency",
		"Product", "AmountExclTax", "TaxRate", "InvestmentType", "Number", "Category",
		"Type", "Fund", "NumberOfShares", "Fees", "IBAN", "EntryReference", "Reference",
		"AccountServicer", "BankTxCode", "OriginalCurrency", "OriginalAmount", "ExchangeRate",
	}
}

// standardRawColumns are the columns that can never hold text: they are
// formatted from time or decimal values, and escaping would corrupt them
// (-180.00 must stay -180.00). Every other column (Status, Currency, Product,
// InvestmentType, references, IBANs, servicer, names, descriptions...) is built
// from statement content a spreadsheet could evaluate as a formula, so it is
// escaped; columns added later are escaped by default.
var standardRawColumns = map[int]bool{
	1: true, 2: true, // Date, ValueDate
	8: true, 12: true, 13: true, // Amount, AmountExclTax, TaxRate
	19: true, 20: true, // NumberOfShares, Fees
	27: true, 28: true, // OriginalAmount, ExchangeRate
}

// Format converts transactions to CSV rows using the existing MarshalCSV method.
// This preserves backward compatibility with the current output format.
func (f *StandardFormatter) Format(transactions []models.Transaction) ([][]string, error) {
	rows := make([][]string, 0, len(transactions))

	for _, tx := range transactions {
		row, err := tx.MarshalCSV()
		if err != nil {
			return nil, err
		}
		for col := range row {
			if !standardRawColumns[col] {
				row[col] = csvsafe.Escape(row[col])
			}
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// Delimiter returns comma as the delimiter for standard CSV format.
func (f *StandardFormatter) Delimiter() rune {
	return ','
}
