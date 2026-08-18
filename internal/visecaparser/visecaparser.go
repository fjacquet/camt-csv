// Package visecaparser converts the CSV transaction export offered by Viseca's
// One card portal into the shared Transaction model.
//
// It covers the same statements as the PDF parser but reads structured fields
// instead of recovering them from laid-out text, so it carries the merchant
// name, the foreign-currency detail and — unlike the PDF — a stable per
// transaction identifier.
package visecaparser

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parsererror"

	"github.com/shopspring/decimal"
)

// byteOrderMark prefixes the Viseca export. csv.Reader does not strip it, so it
// would otherwise become part of the first column's name.
const byteOrderMark = "\ufeff"

// requiredHeaders identifies a Viseca export. The set is deliberately narrow
// enough that no other CSV this tool reads can satisfy it, which is what keeps
// format detection unambiguous.
func requiredHeaders() []string {
	return []string{
		"TransactionId",
		"CardId",
		"Date",
		"ValutaDate",
		"Amount",
		"Currency",
		"MerchantName",
		"StateType",
		"Details",
		"Type",
	}
}

// visecaCSVRow is one raw row of the export, before type conversion.
type visecaCSVRow struct {
	TransactionID    string
	Date             string
	ValutaDate       string
	Amount           string
	Currency         string
	OriginalAmount   string
	OriginalCurrency string
	MerchantName     string
	StateType        string
	Details          string
	Type             string
	ExchangeRate     string
}

// errSkipRow marks a row that parsed cleanly but is deliberately not imported.
var errSkipRow = errors.New("row skipped")

// Options tunes what ParseWithOptions keeps. The zero value is the default the
// CLI uses.
type Options struct {
	// KeepPayments retains the monthly card settlement rows. They are dropped by
	// default: the same money movement already imports from the bank statement as
	// a debit, and an accounting tool that books that debit as a transfer to the
	// card account credits the card itself. Importing Viseca's side as well
	// credits it twice, so the card balance drifts up by the full payment total.
	KeepPayments bool
}

// paymentDescriptors are the settlement descriptors Viseca writes in Details,
// one per portal language. Matching is on lowercase substrings because the
// descriptor carries a trailing thank-you whose wording differs per language.
var paymentDescriptors = []string{
	"votre paiement",   // fr
	"ihre zahlung",     // de
	"your payment",     // en
	"suo pagamento",    // it
	"vostro pagamento", // it
}

// isCardSettlement reports whether the row is the cardholder's monthly payment
// to the issuer rather than a card transaction.
//
// All three conditions are required. A refund is also negative in issuer signs
// and some merchant rows arrive with an empty MerchantName, so neither test
// alone separates a settlement from a transaction worth importing.
func isCardSettlement(row visecaCSVRow, amount decimal.Decimal) bool {
	if !amount.IsNegative() || strings.TrimSpace(row.MerchantName) != "" {
		return false
	}

	details := strings.ToLower(row.Details)
	for _, descriptor := range paymentDescriptors {
		if strings.Contains(details, descriptor) {
			return true
		}
	}
	return false
}

// ParseWithCategorizer reads a Viseca CSV export and returns categorized
// transactions. A malformed row is skipped with a warning rather than failing
// the file, matching the other CSV parsers.
func ParseWithCategorizer(
	ctx context.Context,
	r io.Reader,
	logger logging.Logger,
	categorizer models.TransactionCategorizer,
) ([]models.Transaction, error) {
	return ParseWithOptions(ctx, r, logger, categorizer, Options{})
}

// ParseWithOptions is ParseWithCategorizer with explicit options.
func ParseWithOptions(
	ctx context.Context,
	r io.Reader,
	logger logging.Logger,
	categorizer models.TransactionCategorizer,
	opts Options,
) ([]models.Transaction, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	// FieldsPerRecord is left at its default so csv.Reader rejects a row whose
	// column count does not match the header. Without that guard a ragged row is
	// mapped positionally and silently imported with values shifted between
	// columns, rather than being skipped as malformed.
	reader := csv.NewReader(r)

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("CSV file is empty")
		}
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	stripBOM(header)

	columns := make(map[int]string, len(header))
	for i, name := range header {
		columns[i] = name
	}

	var transactions []models.Transaction
	settlements := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.WithError(err).Warn("Skipping malformed CSV row")
			continue
		}

		row := mapRecord(record, columns)
		if row.TransactionID == "" || row.Date == "" {
			continue
		}

		tx, err := convertRowToTransaction(row, opts)
		if err != nil {
			if err == errSkipRow {
				settlements++
				continue
			}
			logger.WithError(err).Warn("Failed to convert row to transaction",
				logging.Field{Key: "transactionId", Value: row.TransactionID})
			continue
		}
		transactions = append(transactions, tx)
	}

	if settlements > 0 {
		logger.Info("Dropped Viseca card settlement rows; the bank statement carries the same payments",
			logging.Field{Key: "skipped", Value: settlements})
	}

	return common.ProcessTransactionsWithCategorizationStats(ctx, transactions, logger, categorizer, "Viseca")
}

// stripBOM removes a leading byte order mark from the first header cell.
func stripBOM(header []string) {
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], byteOrderMark)
	}
}

// mapRecord projects a CSV record onto visecaCSVRow by column name, so a change
// in Viseca's column order cannot silently shift values between fields.
func mapRecord(record []string, columns map[int]string) visecaCSVRow {
	row := visecaCSVRow{}
	for i, value := range record {
		switch columns[i] {
		case "TransactionId":
			row.TransactionID = value
		case "Date":
			row.Date = value
		case "ValutaDate":
			row.ValutaDate = value
		case "Amount":
			row.Amount = value
		case "Currency":
			row.Currency = value
		case "OriginalAmount":
			row.OriginalAmount = value
		case "OriginalCurrency":
			row.OriginalCurrency = value
		case "MerchantName":
			row.MerchantName = value
		case "StateType":
			row.StateType = value
		case "Details":
			row.Details = value
		case "Type":
			row.Type = value
		case "Exchange Rate":
			row.ExchangeRate = value
		}
	}
	return row
}

// convertRowToTransaction maps one export row onto a Transaction.
//
// Viseca writes card-issuer signs: a purchase is positive because it is money
// the issuer owes, and a refund is negative. Every parser in this tool emits
// debit-negative amounts, so the sign is inverted here.
func convertRowToTransaction(row visecaCSVRow, opts Options) (models.Transaction, error) {
	amount, err := decimal.NewFromString(row.Amount)
	if err != nil {
		return models.Transaction{}, &parsererror.DataExtractionError{
			FilePath:       "(from reader)",
			FieldName:      "Amount",
			RawDataSnippet: row.Amount,
			Msg:            fmt.Sprintf("failed to parse amount: %v", err),
		}
	}

	if !opts.KeepPayments && isCardSettlement(row, amount) {
		return models.Transaction{}, errSkipRow
	}

	amount = amount.Neg()

	// Viseca leaves MerchantName empty on some rows; Details always carries the
	// payment descriptor, so it is the fallback the categorizer matches on.
	name := strings.TrimSpace(row.MerchantName)
	if name == "" {
		name = strings.TrimSpace(row.Details)
	}

	builder := models.NewTransactionBuilder().
		// The export carries its own identifier; do not mint a UUID.
		WithID("").
		WithBookkeepingNumber(row.TransactionID).
		WithDateFromDatetime(row.Date).
		WithValueDateFromDatetime(row.ValutaDate).
		WithAmount(amount, row.Currency).
		WithStatus(mapStateType(row.StateType)).
		WithDescription(row.Details).
		WithPartyName(name).
		// Viseca's own classification ("merchant"/"fee") does not describe a fee
		// in any accounting sense and matches no iCompta type, so it is kept as
		// Product rather than Type: Build copies a set Type into Investment,
		// which would surface it as an InvestmentType on a card transaction.
		WithProduct(row.Type)

	// Name and Recipient are derived from the party fields by Build, the same
	// way the PDF parser populates them.
	if amount.IsNegative() {
		builder = builder.AsDebit().WithPayee(name, "")
	} else {
		builder = builder.AsCredit().WithPayer(name, "")
	}

	// Foreign-currency detail is only meaningful when the original differs.
	if row.OriginalCurrency != "" && row.OriginalCurrency != row.Currency {
		if original, err := decimal.NewFromString(row.OriginalAmount); err == nil {
			builder = builder.WithOriginalAmount(original.Abs(), row.OriginalCurrency)
		}
		if rate, err := decimal.NewFromString(row.ExchangeRate); err == nil {
			builder = builder.WithExchangeRate(rate)
		}
	}

	return builder.Build()
}

// mapStateType converts Viseca's state to the status vocabulary the formatters
// expect.
//
// Only BOOKED has been observed in exports. Anything else maps to PDNG rather
// than passing through, because the iCompta formatter turns an unrecognized
// status into "cleared" — so a pending authorisation would otherwise import as
// a settled transaction.
func mapStateType(state string) string {
	if strings.EqualFold(state, "BOOKED") {
		return "BOOK"
	}
	return "PDNG"
}

// validateFormat reports whether r looks like a Viseca CSV export.
func validateFormat(r io.Reader, logger logging.Logger) (bool, error) {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}

	reader := csv.NewReader(r)

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return false, fmt.Errorf("CSV file is empty")
		}
		return false, fmt.Errorf("failed to read CSV header: %w", err)
	}
	stripBOM(header)

	present := make(map[string]bool, len(header))
	for _, name := range header {
		present[strings.TrimSpace(name)] = true
	}

	for _, required := range requiredHeaders() {
		if !present[required] {
			logger.Debug("Not a Viseca CSV export",
				logging.Field{Key: "missingColumn", Value: required})
			return false, nil
		}
	}

	return true, nil
}
