package camtparser

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"fjacquet/camt-csv/internal/dateutils"
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
		for _, entry := range stmt.Entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			transaction := a.categorizeTransaction(ctx, a.entryToTransaction(entry))

			// categorizeTransaction swallows categorizer errors by design, so a
			// cancellation surfaces here rather than as a failed transaction.
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			transactions = append(transactions, transaction)
		}
	}

	return transactions, nil
}

// categorizeTransaction assigns a category to a transaction using the injected
// categorizer, falling back to Uncategorized when none is configured or the
// lookup fails. Categorization failure is never fatal to a run.
func (a *Adapter) categorizeTransaction(ctx context.Context, transaction models.Transaction) models.Transaction {
	cat := a.GetCategorizer()
	if cat == nil {
		transaction.Category = models.CategoryUncategorized
		return transaction
	}

	// Fall back through the fields most likely to name the counterparty.
	partyName := transaction.PartyName
	if partyName == "" {
		if transaction.Description != "" {
			partyName = transaction.Description
		} else {
			partyName = transaction.RemittanceInfo
		}
	}
	// Strip the payment channel so the categorizer sees the merchant.
	partyName = cleanPaymentMethodPrefixes(partyName)

	category, err := cat.Categorize(
		ctx,
		partyName,
		transaction.CreditDebit == models.TransactionTypeDebit,
		transaction.Amount.String(),
		transaction.Date.Format(dateutils.DateLayoutEuropean),
		transaction.RemittanceInfo,
	)
	if err != nil {
		a.GetLogger().WithError(err).WithFields(
			logging.Field{Key: "party", Value: partyName},
		).Warn("Failed to categorize transaction")
		transaction.Category = models.CategoryUncategorized
		return transaction
	}

	transaction.Category = category.Name
	a.GetLogger().WithFields(
		logging.Field{Key: "party", Value: partyName},
		logging.Field{Key: "category", Value: category.Name},
	).Debug("Transaction categorized successfully")

	return transaction
}

// ValidateFormat checks if a file is a valid CAMT.053 XML file.
func (a *Adapter) ValidateFormat(xmlFile string) (bool, error) {
	return NewISO20022Parser(a.GetLogger()).ValidateFormat(xmlFile)
}
