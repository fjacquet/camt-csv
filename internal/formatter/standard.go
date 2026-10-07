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

// standardTextColumns are the free-text columns of the standard layout, which
// carry bank or merchant text a spreadsheet could evaluate as a formula.
var standardTextColumns = []int{3, 4, 6, 7, 16, 17, 18} // Name, PartyName, Description, RemittanceInfo, Category, Type, Fund

// Format converts transactions to CSV rows using the existing MarshalCSV method.
// This preserves backward compatibility with the current output format.
func (f *StandardFormatter) Format(transactions []models.Transaction) ([][]string, error) {
	rows := make([][]string, 0, len(transactions))

	for _, tx := range transactions {
		row, err := tx.MarshalCSV()
		if err != nil {
			return nil, err
		}
		for _, col := range standardTextColumns {
			row[col] = csvsafe.Escape(row[col])
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// Delimiter returns comma as the delimiter for standard CSV format.
func (f *StandardFormatter) Delimiter() rune {
	return ','
}
