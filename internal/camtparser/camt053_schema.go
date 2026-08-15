package camtparser

import "encoding/xml"

// The types below describe the subset of CAMT.053 (ISO 20022,
// urn:iso:std:iso:20022:tech:xsd:camt.053.001.02) that this parser consumes.
// They intentionally cover only the elements mapped onto models.Transaction;
// everything else in the schema is ignored by the decoder.
//
// A fuller set of CAMT.053 types, with helper methods, lives in
// internal/models/iso20022.go and is used by ISO20022Parser for format
// validation. The two describe the same standard from different angles and are
// deliberately kept separate for now: the models types expose a slice of entry
// details where the shape below assumes a single one, so they are not
// interchangeable without behaviour changes.

// camtAmount is a monetary value with its ISO 4217 currency attribute.
type camtAmount struct {
	Value    string `xml:",chardata"`
	Currency string `xml:"Ccy,attr"`
}

// camtDate wraps the <Dt> child used by booking and value dates.
type camtDate struct {
	Date string `xml:"Dt"`
}

// camtAdditionalInfo is the free-text <AddtlNtryInf> entry annotation.
type camtAdditionalInfo struct {
	Info string `xml:",chardata"`
}

// camtCreditDebitIndicator carries DBIT or CRDT.
type camtCreditDebitIndicator struct {
	Indicator string `xml:",chardata"`
}

// camtStatus is the entry booking status (BOOK, PDNG, ...).
type camtStatus struct {
	Status string `xml:",chardata"`
}

// camtAccountServicerRef is the servicing institution's own entry reference.
type camtAccountServicerRef struct {
	Ref string `xml:",chardata"`
}

// camtReference holds the several identifiers a transaction may be keyed by.
// Banks populate different ones, so all are read and the first non-empty is used.
type camtReference struct {
	MsgID       string `xml:"MsgId,omitempty"`
	AcctSvcrRef string `xml:"AcctSvcrRef,omitempty"`
	InstrID     string `xml:"InstrId,omitempty"`
	EndToEndID  string `xml:"EndToEndId,omitempty"`
	TxID        string `xml:"TxId,omitempty"`
}

// camtRemittanceInfo is the unstructured remittance text.
type camtRemittanceInfo struct {
	Ustrd string `xml:"Ustrd"`
}

// camtAccount is an account identifier. Some banks put the IBAN in the generic
// Othr/Id element instead of the IBAN element, so both are read.
type camtAccount struct {
	IBAN string `xml:"Id>IBAN,omitempty"`
	ID   string `xml:"Id>Othr>Id,omitempty"`
}

// camtParty is a named counterparty with its account.
type camtParty struct {
	Name    string      `xml:"Nm"`
	Account camtAccount `xml:"Acct,omitempty"`
}

// camtRelatedParties describes both sides of a transaction.
type camtRelatedParties struct {
	Debtor          camtParty   `xml:"Dbtr"`
	Creditor        camtParty   `xml:"Cdtr"`
	DebtorAccount   camtAccount `xml:"DbtrAcct,omitempty"`
	CreditorAccount camtAccount `xml:"CdtrAcct,omitempty"`
}

// camtRelatedAccounts is an alternative placement of the counterparty accounts.
type camtRelatedAccounts struct {
	DebtorAccount   camtAccount `xml:"DbtrAcct,omitempty"`
	CreditorAccount camtAccount `xml:"CdtrAcct,omitempty"`
}

// camtTransactionDetails is the <TxDtls> block inside an entry.
type camtTransactionDetails struct {
	References      camtReference            `xml:"Refs"`
	Amount          camtAmount               `xml:"Amt"`
	CreditDebit     camtCreditDebitIndicator `xml:"CdtDbtInd"`
	RemittanceInfo  camtRemittanceInfo       `xml:"RmtInf"`
	RelatedParties  camtRelatedParties       `xml:"RltdPties"`
	RelatedAccounts camtRelatedAccounts      `xml:"RltdAccts,omitempty"`
}

// camtEntryDetails is the <NtryDtls> wrapper.
type camtEntryDetails struct {
	TransactionDetails camtTransactionDetails `xml:"TxDtls"`
}

// camtEntry is a single statement entry: one booked movement on the account.
type camtEntry struct {
	Amount          camtAmount               `xml:"Amt"`
	CreditDebit     camtCreditDebitIndicator `xml:"CdtDbtInd"`
	Status          camtStatus               `xml:"Sts"`
	BookingDate     camtDate                 `xml:"BookgDt"`
	ValueDate       camtDate                 `xml:"ValDt"`
	AccountServicer camtAccountServicerRef   `xml:"AcctSvcrRef"`
	EntryDetails    camtEntryDetails         `xml:"NtryDtls"`
	AdditionalInfo  camtAdditionalInfo       `xml:"AddtlNtryInf"`
}

// camtStatement is one <Stmt> block: a statement for one account and period.
type camtStatement struct {
	Entries []camtEntry `xml:"Ntry"`
}

// camtDocument is the CAMT.053 document root.
type camtDocument struct {
	XMLName       xml.Name `xml:"Document"`
	BkToCstmrStmt struct {
		Stmt []camtStatement `xml:"Stmt"`
	} `xml:"BkToCstmrStmt"`
}
