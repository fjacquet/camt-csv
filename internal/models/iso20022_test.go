package models

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISO20022Document exists to answer one question — "is this a CAMT.053 file?" —
// by unmarshalling and counting statements. These tests pin that contract.
func TestISO20022Document_CountsStatements(t *testing.T) {
	tests := []struct {
		name      string
		xml       string
		wantErr   bool
		wantStmts int
	}{
		{
			name: "single statement",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt><Stmt><Id>STMT-1</Id></Stmt></BkToCstmrStmt>
</Document>`,
			wantStmts: 1,
		},
		{
			name: "several statements",
			xml: `<Document>
  <BkToCstmrStmt>
    <Stmt><Id>A</Id></Stmt>
    <Stmt><Id>B</Id></Stmt>
    <Stmt><Id>C</Id></Stmt>
  </BkToCstmrStmt>
</Document>`,
			wantStmts: 3,
		},
		{
			name:      "well-formed XML with no statements",
			xml:       `<Document><BkToCstmrStmt></BkToCstmrStmt></Document>`,
			wantStmts: 0,
		},
		{
			// XMLName pins the root element, so a well-formed XML document that
			// is not a CAMT.053 Document is rejected outright rather than
			// silently yielding zero statements.
			name:    "well-formed XML with a different root",
			xml:     `<Something><Else/></Something>`,
			wantErr: true,
		},
		{
			name:    "malformed XML",
			xml:     `<Document><BkToCstmrStmt>`,
			wantErr: true,
		},
		{
			name:    "not XML at all",
			xml:     `Bénéficiaire;Date;Montant;Monnaie`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc ISO20022Document
			err := xml.Unmarshal([]byte(tt.xml), &doc)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, doc.BkToCstmrStmt.Stmt, tt.wantStmts)
		})
	}
}

// Elements the type does not model must be ignored rather than rejected: real
// statements carry balances, entries and bank-specific extensions that
// validation has no interest in.
func TestISO20022Document_IgnoresUnmodelledElements(t *testing.T) {
	richXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <GrpHdr><MsgId>MSG-1</MsgId><CreDtTm>2026-03-15T10:00:00</CreDtTm></GrpHdr>
    <Stmt>
      <Id>STMT-1</Id>
      <CreDtTm>2026-03-15T10:00:00</CreDtTm>
      <Acct><Id><IBAN>CH9300762011623852957</IBAN></Id></Acct>
      <Bal><Amt Ccy="CHF">1000.00</Amt></Bal>
      <Ntry>
        <Amt Ccy="CHF">24.50</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <NtryDtls><TxDtls><RmtInf><Ustrd>line one</Ustrd><Ustrd>line two</Ustrd></RmtInf></TxDtls></NtryDtls>
      </Ntry>
      <ProprietaryExtension><Whatever>x</Whatever></ProprietaryExtension>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

	var doc ISO20022Document
	require.NoError(t, xml.Unmarshal([]byte(richXML), &doc))

	require.Len(t, doc.BkToCstmrStmt.Stmt, 1)
	assert.Equal(t, "STMT-1", doc.BkToCstmrStmt.Stmt[0].ID)
}
