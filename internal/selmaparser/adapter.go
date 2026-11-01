package selmaparser

import (
	"context"
	"io"
	"os"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/parser"
)

// Adapter implements the parser.FullParser interface for Selma CSV files.
type Adapter struct {
	parser.BaseParser
	securityNames map[string]string // ISIN -> existing iCompta security name
}

// SetSecurityNames makes Parse name rows after the given securities, keyed by ISIN.
func (a *Adapter) SetSecurityNames(names map[string]string) { a.securityNames = names }

// NewAdapter creates a new adapter for the selmaparser.
func NewAdapter(logger logging.Logger) *Adapter {
	return &Adapter{
		BaseParser: parser.NewBaseParser(logger),
	}
}

// Parse reads data from the provided io.Reader and returns a slice of Transaction models.
func (a *Adapter) Parse(ctx context.Context, r io.Reader) ([]models.Transaction, error) {
	txs, err := ParseWithCategorizer(ctx, r, a.GetLogger(), a.GetCategorizer())
	missing := map[string]bool{}
	for i := range txs {
		if a.securityNames == nil || txs[i].Fund == "" {
			continue
		}
		if name, ok := a.securityNames[txs[i].Fund]; ok {
			// Fund keeps the full "ISIN-name" (the security); the payee is the readable name.
			txs[i].Fund = name
			txs[i].PartyName = strings.TrimPrefix(name, txs[i].PartyName+"-")
			txs[i].Name = txs[i].PartyName
		} else {
			missing[txs[i].Fund] = true
		}
	}
	// Never created here: the user adds the security and its ticker in iCompta.
	for isin := range missing {
		a.GetLogger().Warn("NEW SECURITY NEEDED in iCompta (not created): create it as \"ISIN-name\" with its ticker, then re-import",
			logging.Field{Key: "isin", Value: isin})
	}
	return txs, err
}

// ValidateFormat checks if a file is a valid Selma CSV file.
func (a *Adapter) ValidateFormat(file string) (bool, error) {
	f, err := os.Open(file) // #nosec G304 -- CLI tool requires user-provided file paths
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			a.GetLogger().WithError(err).Warn("Failed to close file during format validation",
				logging.Field{Key: "file", Value: file})
		}
	}()

	return validateFormat(f, a.GetLogger())
}
