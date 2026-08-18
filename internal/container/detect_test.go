package container

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDetectionContainer builds a container with AI disabled so detection tests
// never reach a network.
func newDetectionContainer(t *testing.T) *Container {
	t.Helper()

	// The zero value leaves AI disabled, so detection never reaches a network.
	cfg := &config.Config{}
	cfg.Log.Level = "error"
	cfg.Log.Format = "text"

	c, err := NewContainer(cfg)
	require.NoError(t, err)
	return c
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

const (
	camtXML = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Id>STMT-001</Id>
      <Ntry>
        <Amt Ccy="CHF">100.00</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt><Dt>2026-03-15</Dt></BookgDt>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	revolutCSV = `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2026-03-15 10:00:00,2026-03-15 10:00:00,Coop Pronto,-24.50,0.00,CHF,COMPLETED,975.50`

	revolutInvestmentCSV = `Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2026-03-15T10:00:00.000Z,AAPL,BUY - MARKET,1,$180.00,$180.00,USD,0.88`

	selmaCSV = `Date,Description,Bookkeeping No.,Fund,Amount,Currency,Number of Shares
2026-03-15,trade,1,IE00B4L5Y983,100.00,CHF,1.5`

	debitCSV = `Bénéficiaire;Date;Montant;Monnaie
COOP PRONTO;15.03.2026;-24.50;CHF`

	revolutCryptoCSV = `Symbol,Type,Quantity,Price,Value,Fees,Date
BTC,Achat,0.001,50000.00,50.00,0.50,15 mars 2026 10:00:00`

	visecaCSV = `TransactionId,CardId,Date,ValutaDate,Amount,Currency,OriginalAmount,OriginalCurrency,MerchantName,MerchantPlace,MerchantCountry,StateType,Details,Type,Exchange Rate
TRX2026031500001,000000XXXXXXXXXX,2026-03-15 15:12:22,2026-03-16 00:00:00,29.000,CHF,29.000,CHF,Coop,Montreux,CHE,BOOKED,Coop Montreux,merchant,1.000000`
)

// Every supported format must be recognized as itself. This is the contract the
// `convert` command rests on.
func TestDetectParser_RecognizesEveryFormat(t *testing.T) {
	c := newDetectionContainer(t)
	dir := t.TempDir()

	tests := []struct {
		name     string
		fileName string
		content  string
		want     ParserType
	}{
		{"CAMT.053 XML", "statement.xml", camtXML, CAMT},
		{"Revolut CSV", "revolut.csv", revolutCSV, Revolut},
		{"Revolut Investment CSV", "investment.csv", revolutInvestmentCSV, RevolutInvestment},
		{"Revolut Crypto CSV", "crypto.csv", revolutCryptoCSV, RevolutCrypto},
		{"Selma CSV", "selma.csv", selmaCSV, Selma},
		{"Visa Debit CSV", "debit.csv", debitCSV, Debit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, dir, tt.fileName, tt.content)

			p, got, err := c.DetectParser(path)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.NotNil(t, p)
		})
	}
}

// The detectors must not overlap: a file of one format must be claimed by that
// format alone. Before the Revolut Investment validator checked its headers it
// accepted any readable CSV, which would have made detection meaningless.
func TestDetectParser_ValidatorsDoNotOverlap(t *testing.T) {
	c := newDetectionContainer(t)
	dir := t.TempDir()

	samples := map[ParserType]string{
		Revolut:           revolutCSV,
		RevolutInvestment: revolutInvestmentCSV,
		RevolutCrypto:     revolutCryptoCSV,
		Selma:             selmaCSV,
		Debit:             debitCSV,
		Viseca:            visecaCSV,
	}

	for owner, content := range samples {
		path := writeFile(t, dir, string(owner)+"-sample.csv", content)

		for _, candidate := range DetectionOrder() {
			p, err := c.GetParser(candidate)
			require.NoError(t, err)

			valid, err := p.ValidateFormat(path)
			accepted := err == nil && valid

			if candidate == owner {
				assert.True(t, accepted, "%s must accept its own sample", candidate)
			} else {
				assert.False(t, accepted, "%s must not claim a %s file", candidate, owner)
			}
		}
	}
}

// An unrecognized file must be reported, never guessed at. camt-csv's own
// 29-column output is the realistic case: it is a valid CSV that no input
// parser should accept.
func TestDetectParser_RejectsUnknownFormats(t *testing.T) {
	c := newDetectionContainer(t)
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{"camt-csv standard output", "BookkeepingNumber,Status,Date,ValueDate,Name,PartyName\n1,BOOK,15.03.2026,15.03.2026,X,Y"},
		{"unrelated CSV", "foo,bar,baz\n1,2,3"},
		{"empty file", ""},
		{"plain text", "this is not a statement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, dir, "unknown.csv", tt.content)

			_, _, err := c.DetectParser(path)

			require.ErrorIs(t, err, ErrFormatNotRecognized)
		})
	}
}

func TestDetectParser_MissingFile(t *testing.T) {
	c := newDetectionContainer(t)

	_, _, err := c.DetectParser(filepath.Join(t.TempDir(), "does-not-exist.csv"))

	require.ErrorIs(t, err, ErrFormatNotRecognized)
}

// DetectionOrder hands out a copy: a caller mutating the result must not
// reorder detection for everyone else.
func TestDetectionOrder_IsACopy(t *testing.T) {
	first := DetectionOrder()
	require.NotEmpty(t, first)

	original := first[0]
	first[0] = "mutated"

	assert.Equal(t, original, DetectionOrder()[0])
}
