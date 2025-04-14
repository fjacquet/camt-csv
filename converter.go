package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"os"
)

// CAMT053 is a struct that represents the CAMT.053 XML structure
type CAMT053 struct {
	XMLName             xml.Name `xml:"Document"`
	BkToCstmrStmt      BkToCstmrStmt `xml:"BkToCstmrStmt"`
}

// BkToCstmrStmt represents the Bank To Customer Statement
type BkToCstmrStmt struct {
	Stmt Stmt `xml:"Stmt"`
}

// Stmt represents the Statement
type Stmt struct {
	Id       string `xml:"Id"`
	CreDtTm  string `xml:"CreDtTm"`
	Bal      []Bal  `xml:"Bal"`
	Ntry     []Ntry `xml:"Ntry"`
}

// Bal represents the Balance
type Bal struct {
	Tp     Tp     `xml:"Tp"`
	Amt    Amt    `xml:"Amt"`
	CdtDbtInd string `xml:"CdtDbtInd"`
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
	Text    string `xml:"Text"`
	Ccy string `xml:"Ccy,attr"`
}

// Ntry represents the Entry
type Ntry struct {
	Amt          Amt          `xml:"Amt"`
	CdtDbtInd    string       `xml:"CdtDbtInd"`
	BookgDt      BookgDt      `xml:"BookgDt"`
	NtryDtls     NtryDtls     `xml:"NtryDtls"`
}

// BookgDt represents the Booking Date
type BookgDt struct {
	Dt string `xml:"Dt"`
}

// NtryDtls represents the Entry Details
type NtryDtls struct {
	TxDtls []TxDtls `xml:"TxDtls"`
}

// TxDtls represents the Transaction Details
type TxDtls struct {
	RmtInf RmtInf `xml:"RmtInf"`
}

// RmtInf represents the Remittance Information
type RmtInf struct {
	Ustrd []string `xml:"Ustrd"`
}


func convertXMLToCSV(xmlFile string, csvFile string) error {
	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		return fmt.Errorf("error reading XML file: %w", err)
	}

	var camt053 CAMT053
	err = xml.Unmarshal(xmlData, &camt053)
	if err != nil {
		return fmt.Errorf("error unmarshalling XML: %w", err)
	}

	csvFileHandle, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %w", err)
	}
	defer csvFileHandle.Close()

	writer := csv.NewWriter(csvFileHandle)
	defer writer.Flush()

	header := []string{"Transaction Amount", "Credit/Debit", "Booking Date", "Remittance Info"}
	err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	for _, entry := range camt053.BkToCstmrStmt.Stmt.Ntry {
		for _, txDtl := range entry.NtryDtls.TxDtls {
			for _, ustrd := range txDtl.RmtInf.Ustrd {
				record := []string{
					entry.Amt.Text + " " + entry.Amt.Ccy,
					entry.CdtDbtInd,
					entry.BookgDt.Dt,
					ustrd,
				}
				err = writer.Write(record)
				if err != nil {
					return fmt.Errorf("error writing CSV record: %w", err)
				}
			}
		}
	}

	return nil
}
