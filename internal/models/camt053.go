// Package models provides the data structures used throughout the application.
package models

import "encoding/xml"

// CAMT053 is a struct that represents the CAMT.053 XML structure
type CAMT053 struct {
	XMLName       xml.Name      `xml:"Document"`
	BkToCstmrStmt BkToCstmrStmt `xml:"BkToCstmrStmt"`
}

// BkToCstmrStmt represents the Bank To Customer Statement
type BkToCstmrStmt struct {
	GrpHdr GrpHdr `xml:"GrpHdr"`
	Stmt   Stmt   `xml:"Stmt"`
}

// GrpHdr represents the Group Header
type GrpHdr struct {
	MsgId    string `xml:"MsgId"`
	CreDtTm  string `xml:"CreDtTm"`
	MsgPgntn struct {
		PgNb       string `xml:"PgNb"`
		LastPgInd  string `xml:"LastPgInd"`
	} `xml:"MsgPgntn"`
}

// Stmt represents the Statement
type Stmt struct {
	Id          string  `xml:"Id"`
	ElctrncSeqNb string `xml:"ElctrncSeqNb"`
	CreDtTm     string  `xml:"CreDtTm"`
	FrToDt      FrToDt  `xml:"FrToDt"`
	Acct        Acct    `xml:"Acct"`
	Bal         []Bal   `xml:"Bal"`
	Ntry        []Ntry  `xml:"Ntry"`
}

// FrToDt represents the From To Date
type FrToDt struct {
	FrDtTm  string `xml:"FrDtTm"`
	ToDtTm  string `xml:"ToDtTm"`
}

// Acct represents the Account
type Acct struct {
	Id   struct {
		IBAN string `xml:"IBAN"`
	} `xml:"Id"`
	Ccy  string `xml:"Ccy"`
	Ownr struct {
		Nm  string `xml:"Nm"`
	} `xml:"Ownr"`
}

// Bal represents the Balance
type Bal struct {
	Tp        Tp     `xml:"Tp"`
	Amt       Amt    `xml:"Amt"`
	CdtDbtInd string `xml:"CdtDbtInd"`
	Dt        struct {
		Dt string `xml:"Dt"`
	} `xml:"Dt"`
}

// Tp represents the Type
type Tp struct {
	CdOrPrtry CdOrPrtry `xml:"CdOrPrtry"`
}

// CdOrPrtry represents the Code or Proprietary
type CdOrPrtry struct {
	Cd string `xml:"Cd"`
}

// Amt represents the Amount
type Amt struct {
	Text string `xml:",chardata"`
	Ccy  string `xml:"Ccy,attr"`
}

// Ntry represents the Entry
type Ntry struct {
	NtryRef      string    `xml:"NtryRef"`
	Amt          Amt       `xml:"Amt"`
	CdtDbtInd    string    `xml:"CdtDbtInd"`
	Sts          string    `xml:"Sts"`
	BookgDt      BookgDt   `xml:"BookgDt"`
	ValDt        ValDt     `xml:"ValDt"`
	AcctSvcrRef  string    `xml:"AcctSvcrRef"`
	BkTxCd       BkTxCd    `xml:"BkTxCd"`
	NtryDtls     NtryDtls  `xml:"NtryDtls"`
	AddtlNtryInf string    `xml:"AddtlNtryInf"`
}

// BookgDt represents the Booking Date
type BookgDt struct {
	Dt string `xml:"Dt"`
}

// ValDt represents the Value Date
type ValDt struct {
	Dt string `xml:"Dt"`
}

// BkTxCd represents the Bank Transaction Code
type BkTxCd struct {
	Domn Domn `xml:"Domn"`
}

// Domn represents the Domain
type Domn struct {
	Cd    string `xml:"Cd"`
	Fmly  Fmly   `xml:"Fmly"`
}

// Fmly represents the Family
type Fmly struct {
	Cd         string `xml:"Cd"`
	SubFmlyCd  string `xml:"SubFmlyCd"`
}

// NtryDtls represents the Entry Details
type NtryDtls struct {
	TxDtls []TxDtls `xml:"TxDtls"`
}

// TxDtls represents the Transaction Details
type TxDtls struct {
	Refs     Refs     `xml:"Refs"`
	Amt      Amt      `xml:"Amt"`
	CdtDbtInd string  `xml:"CdtDbtInd"`
	AmtDtls  AmtDtls  `xml:"AmtDtls"`
	RltdPties RltdPties `xml:"RltdPties"`
	RmtInf    RmtInf   `xml:"RmtInf"`
}

// Refs represents the References
type Refs struct {
	MsgId       string `xml:"MsgId"`
	EndToEndId  string `xml:"EndToEndId"`
	TxId        string `xml:"TxId"`
	InstrId     string `xml:"InstrId"`
}

// AmtDtls represents the Amount Details
type AmtDtls struct {
	InstdAmt struct {
		Amt Amt `xml:"Amt"`
	} `xml:"InstdAmt"`
}

// RltdPties represents the Related Parties
type RltdPties struct {
	Dbtr       Party `xml:"Dbtr"`
	DbtrAcct   Acct  `xml:"DbtrAcct"`
	Cdtr       Party `xml:"Cdtr"`
	CdtrAcct   Acct  `xml:"CdtrAcct"`
}

// Party represents a Party (Debtor or Creditor)
type Party struct {
	Nm string `xml:"Nm"`
}

// RmtInf represents the Remittance Information
type RmtInf struct {
	Ustrd []string `xml:"Ustrd"`
}
