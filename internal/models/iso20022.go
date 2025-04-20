// Package models provides the data structures used throughout the application.
package models

import (
	"encoding/xml"
	"strings"
)

// ISO20022Document represents the root structure of a CAMT.053 XML document
type ISO20022Document struct {
	XMLName     xml.Name `xml:"Document"`
	BkToCstmrStmt struct {
		Stmt []Statement `xml:"Stmt"`
	} `xml:"BkToCstmrStmt"`
}

// Statement represents a bank statement in the CAMT.053 format
type Statement struct {
	ID        string    `xml:"Id"`
	CreDtTm   string    `xml:"CreDtTm"`
	FrToDt    *Period   `xml:"FrToDt"`
	Acct      Account   `xml:"Acct"`
	Bal       []Balance `xml:"Bal"`
	Ntry      []Entry   `xml:"Ntry"`
}

// Period represents a time period in the CAMT.053 format
type Period struct {
	FrDtTm string `xml:"FrDtTm"`
	ToDtTm string `xml:"ToDtTm"`
}

// Account represents a bank account in the CAMT.053 format
type Account struct {
	ID struct {
		IBAN string `xml:"IBAN"`
		Othr struct {
			ID string `xml:"Id"`
		} `xml:"Othr"`
	} `xml:"Id"`
	Ccy  string `xml:"Ccy"`
	Nm   string `xml:"Nm"`
	Ownr struct {
		Nm string `xml:"Nm"`
	} `xml:"Ownr"`
	Svcr struct {
		FinInstnId struct {
			BIC string `xml:"BIC"`
			Nm  string `xml:"Nm"`
		} `xml:"FinInstnId"`
	} `xml:"Svcr"`
}

// Balance represents a balance in the CAMT.053 format
type Balance struct {
	Tp struct {
		CdOrPrtry struct {
			Cd string `xml:"Cd"`
		} `xml:"CdOrPrtry"`
	} `xml:"Tp"`
	Amt      Amount `xml:"Amt"`
	CdtDbtInd string `xml:"CdtDbtInd"`
	Dt        struct {
		Dt string `xml:"Dt"`
	} `xml:"Dt"`
}

// Amount represents a monetary amount with currency
type Amount struct {
	Value   string `xml:",chardata"`
	Ccy     string `xml:"Ccy,attr"`
}

// Entry represents a transaction entry in the CAMT.053 format
type Entry struct {
	NtryRef    string    `xml:"NtryRef"`
	Amt        Amount    `xml:"Amt"`
	CdtDbtInd  string    `xml:"CdtDbtInd"` // CRDT or DBIT
	Sts        string    `xml:"Sts"`       // Status (BOOK, etc.)
	BookgDt    EntryDate `xml:"BookgDt"`   // Booking date
	ValDt      EntryDate `xml:"ValDt"`     // Value date
	AcctSvcrRef string   `xml:"AcctSvcrRef"` // Bank reference
	BkTxCd     BankTxCode `xml:"BkTxCd"`   // Bank transaction code
	NtryDtls   EntryDetails `xml:"NtryDtls"` // Transaction details
	AddtlNtryInf string    `xml:"AddtlNtryInf"` // Additional entry information
}

// EntryDate represents a date in ISO20022 format
type EntryDate struct {
	Dt string `xml:"Dt"`
}

// BankTxCode represents a bank transaction code in the CAMT.053 format
type BankTxCode struct {
	Domn struct {
		Cd   string `xml:"Cd"`
		Fmly struct {
			Cd        string `xml:"Cd"`
			SubFmlyCd string `xml:"SubFmlyCd"`
		} `xml:"Fmly"`
	} `xml:"Domn"`
	Prtry struct {
		Cd   string `xml:"Cd"`
		Issr string `xml:"Issr"`
	} `xml:"Prtry"`
}

// EntryDetails represents transaction details in the CAMT.053 format
type EntryDetails struct {
	TxDtls []TransactionDetails `xml:"TxDtls"`
}

// TransactionDetails represents detailed transaction information in the CAMT.053 format
type TransactionDetails struct {
	Refs       References `xml:"Refs"`
	Amt        Amount     `xml:"Amt"`
	CdtDbtInd  string     `xml:"CdtDbtInd"`
	AmtDtls    struct {
		InstdAmt struct {
			Amt Amount `xml:"Amt"`
		} `xml:"InstdAmt"`
	} `xml:"AmtDtls"`
	RmtInf     struct {
		Ustrd []string `xml:"Ustrd"`
		Strd  []struct {
			CdtrRefInf struct {
				Ref string `xml:"Ref"`
			} `xml:"CdtrRefInf"`
		} `xml:"Strd"`
	} `xml:"RmtInf"`
	RltdPties  RelatedParties `xml:"RltdPties"`
	RltdAgts   RelatedAgents  `xml:"RltdAgts"`
	Purp       struct {
		Cd string `xml:"Cd"`
	} `xml:"Purp"`
	AddtlTxInf string `xml:"AddtlTxInf"`
}

// References represents transaction references in the CAMT.053 format
type References struct {
	MsgID      string `xml:"MsgId"`
	EndToEndID string `xml:"EndToEndId"`
	TxID       string `xml:"TxId"`
	PmtInfID   string `xml:"PmtInfId"`
	InstrID    string `xml:"InstrId"`
}

// RelatedParties represents parties involved in the transaction in the CAMT.053 format
type RelatedParties struct {
	Dbtr struct {
		Nm     string `xml:"Nm"`
		PstlAdr struct {
			AdrLine []string `xml:"AdrLine"`
			StrtNm  string   `xml:"StrtNm"`
			BldgNb  string   `xml:"BldgNb"`
			PstCd   string   `xml:"PstCd"`
			TwnNm   string   `xml:"TwnNm"`
			Ctry    string   `xml:"Ctry"`
		} `xml:"PstlAdr"`
	} `xml:"Dbtr"`
	DbtrAcct struct {
		ID struct {
			IBAN string `xml:"IBAN"`
			Othr struct {
				ID string `xml:"Id"`
			} `xml:"Othr"`
		} `xml:"Id"`
	} `xml:"DbtrAcct"`
	UltmtDbtr struct {
		Nm string `xml:"Nm"`
	} `xml:"UltmtDbtr"`
	Cdtr struct {
		Nm     string `xml:"Nm"`
		PstlAdr struct {
			AdrLine []string `xml:"AdrLine"`
			StrtNm  string   `xml:"StrtNm"`
			BldgNb  string   `xml:"BldgNb"`
			PstCd   string   `xml:"PstCd"`
			TwnNm   string   `xml:"TwnNm"`
			Ctry    string   `xml:"Ctry"`
		} `xml:"PstlAdr"`
	} `xml:"Cdtr"`
	CdtrAcct struct {
		ID struct {
			IBAN string `xml:"IBAN"`
			Othr struct {
				ID string `xml:"Id"`
			} `xml:"Othr"`
		} `xml:"Id"`
	} `xml:"CdtrAcct"`
	UltmtCdtr struct {
		Nm string `xml:"Nm"`
	} `xml:"UltmtCdtr"`
}

// RelatedAgents represents financial institutions involved in the transaction
type RelatedAgents struct {
	DbtrAgt struct {
		FinInstnID struct {
			BIC  string `xml:"BIC"`
			Nm   string `xml:"Nm"`
			PstlAdr struct {
				AdrLine []string `xml:"AdrLine"`
			} `xml:"PstlAdr"`
			ClrSysMmbID struct {
				ClrSysID struct {
					Cd string `xml:"Cd"`
				} `xml:"ClrSysId"`
				MmbID string `xml:"MmbId"`
			} `xml:"ClrSysMmbId"`
		} `xml:"FinInstnId"`
	} `xml:"DbtrAgt"`
	CdtrAgt struct {
		FinInstnID struct {
			BIC  string `xml:"BIC"`
			Nm   string `xml:"Nm"`
			PstlAdr struct {
				AdrLine []string `xml:"AdrLine"`
			} `xml:"PstlAdr"`
			ClrSysMmbID struct {
				ClrSysID struct {
					Cd string `xml:"Cd"`
				} `xml:"ClrSysId"`
				MmbID string `xml:"MmbId"`
			} `xml:"ClrSysMmbId"`
		} `xml:"FinInstnId"`
	} `xml:"CdtrAgt"`
}

// GetFirstTxDetails returns the first transaction details if available
func (e *Entry) GetFirstTxDetails() *TransactionDetails {
	if len(e.NtryDtls.TxDtls) > 0 {
		return &e.NtryDtls.TxDtls[0]
	}
	return nil
}

// GetCreditDebit returns the credit/debit indicator
func (e *Entry) GetCreditDebit() string {
	return e.CdtDbtInd
}

// IsCredit returns true if the entry is a credit transaction
func (e *Entry) IsCredit() bool {
	return e.CdtDbtInd == "CRDT"
}

// GetBankTxCode returns the bank transaction code
func (e *Entry) GetBankTxCode() string {
	if e.BkTxCd.Domn.Cd != "" {
		family := e.BkTxCd.Domn.Fmly.Cd
		subFamily := e.BkTxCd.Domn.Fmly.SubFmlyCd
		if family != "" && subFamily != "" {
			return e.BkTxCd.Domn.Cd + "/" + family + "/" + subFamily
		}
		return e.BkTxCd.Domn.Cd
	}
	if e.BkTxCd.Prtry.Cd != "" {
		return e.BkTxCd.Prtry.Cd
	}
	return ""
}

// GetRemittanceInfo returns all remittance information
func (e *Entry) GetRemittanceInfo() string {
	var result []string
	txDetails := e.GetFirstTxDetails()
	if txDetails != nil {
		result = txDetails.RmtInf.Ustrd
	}
	return strings.Join(result, ", ")
}

// GetPayer returns the payer name
func (e *Entry) GetPayer() string {
	if e.CdtDbtInd == "CRDT" {
		// For credit transactions (incoming money), debtor is the payer
		txDetails := e.GetFirstTxDetails()
		if txDetails != nil {
			// Try ultimate debtor first
			if txDetails.RltdPties.UltmtDbtr.Nm != "" {
				return txDetails.RltdPties.UltmtDbtr.Nm
			}
			// Then regular debtor
			if txDetails.RltdPties.Dbtr.Nm != "" {
				return txDetails.RltdPties.Dbtr.Nm
			}
			// Try debtor agent (bank)
			if txDetails.RltdAgts.DbtrAgt.FinInstnID.Nm != "" {
				return txDetails.RltdAgts.DbtrAgt.FinInstnID.Nm
			}
		}
	}
	// Default or unknown payer
	return ""
}

// GetPayee returns the payee name
func (e *Entry) GetPayee() string {
	if e.CdtDbtInd == "DBIT" {
		// For debit transactions (outgoing money), creditor is the payee
		txDetails := e.GetFirstTxDetails()
		if txDetails != nil {
			// Try ultimate creditor first
			if txDetails.RltdPties.UltmtCdtr.Nm != "" {
				return txDetails.RltdPties.UltmtCdtr.Nm
			}
			// Then regular creditor
			if txDetails.RltdPties.Cdtr.Nm != "" {
				return txDetails.RltdPties.Cdtr.Nm
			}
			// Try creditor agent (bank)
			if txDetails.RltdAgts.CdtrAgt.FinInstnID.Nm != "" {
				return txDetails.RltdAgts.CdtrAgt.FinInstnID.Nm
			}
		}
	}
	// Default or unknown payee
	return ""
}

// GetReference returns the transaction reference
func (e *Entry) GetReference() string {
	// First try the entry reference
	if e.NtryRef != "" {
		return e.NtryRef
	}
	
	// Then try references from transaction details
	txDetails := e.GetFirstTxDetails()
	if txDetails != nil {
		refs := txDetails.Refs
		
		// Try different reference types in order of preference
		if refs.EndToEndID != "" {
			return refs.EndToEndID
		}
		if refs.TxID != "" {
			return refs.TxID
		}
		if refs.PmtInfID != "" {
			return refs.PmtInfID
		}
		if refs.MsgID != "" {
			return refs.MsgID
		}
		if refs.InstrID != "" {
			return refs.InstrID
		}
		
		// Check for structured reference in remittance info
		for _, strd := range txDetails.RmtInf.Strd {
			if strd.CdtrRefInf.Ref != "" {
				return strd.CdtrRefInf.Ref
			}
		}
	}
	
	return ""
}

// GetIBAN returns the IBAN for the transaction
func (e *Entry) GetIBAN() string {
	txDetails := e.GetFirstTxDetails()
	if txDetails == nil {
		return ""
	}
	
	if e.CdtDbtInd == "DBIT" {
		// For debit transactions, get creditor IBAN
		if txDetails.RltdPties.CdtrAcct.ID.IBAN != "" {
			return txDetails.RltdPties.CdtrAcct.ID.IBAN
		}
	} else {
		// For credit transactions, get debtor IBAN
		if txDetails.RltdPties.DbtrAcct.ID.IBAN != "" {
			return txDetails.RltdPties.DbtrAcct.ID.IBAN
		}
	}
	
	return ""
}

// BuildDescription builds a detailed description from the transaction data
func (e *Entry) BuildDescription() string {
	var parts []string
	
	// Add entry information if available
	if e.AddtlNtryInf != "" {
		parts = append(parts, e.AddtlNtryInf)
	}
	
	txDetails := e.GetFirstTxDetails()
	if txDetails != nil {
		// Add transaction additional info
		if txDetails.AddtlTxInf != "" {
			parts = append(parts, txDetails.AddtlTxInf)
		}
		
		// Add remittance information
		for _, ustrd := range txDetails.RmtInf.Ustrd {
			if ustrd != "" {
				parts = append(parts, ustrd)
			}
		}
		
		// Add purpose code if available
		if txDetails.Purp.Cd != "" {
			parts = append(parts, "Purpose: "+txDetails.Purp.Cd)
		}
	}
	
	// Join all parts together
	description := strings.Join(parts, " - ")
	
	// Clean up the description
	description = strings.ReplaceAll(description, "\n", " ")
	description = strings.ReplaceAll(description, "\r", "")
	description = strings.TrimSpace(description)
	
	// Handle empty description
	if description == "" {
		description = "Transaction " + e.GetReference()
	}
	
	return description
}
