// Package models provides the data structures used throughout the application.
package models

import "encoding/xml"

// ISO20022Document is the root of a CAMT.053 (ISO 20022,
// urn:iso:std:iso:20022:tech:xsd:camt.053.001.02) bank statement document.
//
// This type exists for format *validation* only: camtparser.ISO20022Parser
// unmarshals a candidate file into it and checks that at least one statement
// came back. It deliberately models only the elements that question needs.
//
// Data extraction uses a separate, fuller shape in
// internal/camtparser/camt053_schema.go. The two are kept apart on purpose:
// this one answers "is this a CAMT.053 file?", that one answers "what is in
// it?", and conflating them would make a validation tweak able to change
// parsing output.
type ISO20022Document struct {
	XMLName       xml.Name `xml:"Document"`
	BkToCstmrStmt struct {
		Stmt []Statement `xml:"Stmt"`
	} `xml:"BkToCstmrStmt"`
}

// Statement is one <Stmt> block: a statement for a single account and period.
//
// Only the identifier is modelled. Validation counts statements rather than
// inspecting them, and encoding/xml appends one element per <Stmt> whatever
// fields are declared here.
type Statement struct {
	ID string `xml:"Id"`
}
