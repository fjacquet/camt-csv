package camtparser

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"fjacquet/camt-csv/internal/common"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"

	"golang.org/x/net/html/charset"
)

// Adapter implements the parser.FullParser interface for CAMT.053 XML files.
type Adapter struct {
	parser.BaseParser
}

// NewAdapter creates a new adapter for the camtparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// Parse decodes a CAMT.053 XML document and returns its entries as Transactions,
// categorizing each one along the way.
//
// The document shape lives in camt053_schema.go and the per-entry mapping in
// entry_mapping.go; this method only drives the decode-map-categorize loop.
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	xmlData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading from reader: %w", err)
	}

	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	// Swiss banks emit CAMT files in several encodings; resolve whatever the
	// XML declaration asks for.
	decoder.CharsetReader = charset.NewReaderLabel

	var doc camtDocument
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("error decoding XML: %w", err)
	}

	var transactions []models.Transaction
	for _, stmt := range doc.BkToCstmrStmt.Stmt {
		// Banks put the account in either element of <Acct><Id>, so both are
		// read; whichever is present identifies the statement's account.
		statementAccount := stmt.Account.IBAN
		if statementAccount == "" {
			statementAccount = stmt.Account.ID
		}

		for _, entry := range stmt.Entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			transaction, ok := a.entryToTransaction(entry, statementAccount)
			if !ok {
				continue
			}
			transactions = append(transactions, transaction)
		}
	}

	return common.ProcessTransactionsWithPartyName(ctx, transactions, a.GetLogger(), a.GetCategorizer(), "CAMT", camtPartyName)
}

// camtPartyName is who a CAMT entry is with: the resolved party, else the
// entry text, without the card/transfer prefix the bank puts in front.
func camtPartyName(tx models.Transaction) string {
	party := tx.PartyName
	if party == "" {
		party = tx.Description
	}
	if party == "" {
		party = tx.RemittanceInfo
	}
	return cleanPaymentMethodPrefixes(party)
}

// ValidateFormat checks if a file is a valid CAMT.053 XML file.
func (a *Adapter) ValidateFormat(xmlFile string) (bool, error) {
	return NewISO20022Parser(a.GetLogger()).ValidateFormat(xmlFile)
}
