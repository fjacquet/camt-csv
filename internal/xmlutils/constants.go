// Package xmlutils provides XML-related utility functions used throughout the application.
package xmlutils

// CAMT053 contains all XPath expressions used for CAMT.053 XML parsing
type CAMT053 struct {
	// Entry contains XPath expressions for basic entry data
	Entry struct {
		Amount          string
		Currency        string
		CreditDebitInd  string
		BookingDate     string
		ValueDate       string
		Status          string
		AccountSvcRef   string
		BankTxDomain    string
		BankTxFamily    string
		BankTxSubFamily string
		ProprietaryCode string
		AddEntryInfo    string
	}

	// References contains XPath expressions for transaction references
	References struct {
		EndToEndID    string
		TransactionID string
		PaymentInfoID string
	}

	// Remittance contains XPath expressions for remittance information
	Remittance struct {
		UnstructuredInfo string
		AdditionalTxInfo string
	}

	// Party contains XPath expressions for party information
	Party struct {
		DebtorName        string
		DebtorAgentName   string
		CreditorName      string
		CreditorAgentName string
		UltimateDebtor    string
		UltimateCreditor  string
	}

	// Account contains XPath expressions for account information
	Account struct {
		IBAN string
	}
}

// DefaultCamt053XPaths returns a CAMT053 struct with the default XPath expressions
func DefaultCamt053XPaths() CAMT053 {
	camt := CAMT053{}

	// Basic entry data
	camt.Entry.Amount = "//Ntry/Amt"
	camt.Entry.Currency = "//Ntry/Amt/@Ccy"
	camt.Entry.CreditDebitInd = "//Ntry/CdtDbtInd"
	camt.Entry.BookingDate = "//Ntry/BookgDt/Dt"
	camt.Entry.ValueDate = "//Ntry/ValDt/Dt"
	camt.Entry.Status = "//Ntry/Sts"
	camt.Entry.AccountSvcRef = "//Ntry/AcctSvcrRef"
	camt.Entry.BankTxDomain = "//Ntry/BkTxCd/Domn/Cd"
	camt.Entry.BankTxFamily = "//Ntry/BkTxCd/Domn/Fmly/Cd"
	camt.Entry.BankTxSubFamily = "//Ntry/BkTxCd/Domn/Fmly/SubFmlyCd"
	camt.Entry.ProprietaryCode = "//Ntry/BkTxCd/Prtry/Cd"
	camt.Entry.AddEntryInfo = "//Ntry/AddtlNtryInf"

	// Transaction references
	camt.References.EndToEndID = "//Ntry/NtryDtls/TxDtls/Refs/EndToEndId"
	camt.References.TransactionID = "//Ntry/NtryDtls/TxDtls/Refs/TxId"
	camt.References.PaymentInfoID = "//Ntry/NtryDtls/TxDtls/Refs/PmtInfId"

	// Remittance information
	camt.Remittance.UnstructuredInfo = "//Ntry/NtryDtls/TxDtls/RmtInf/Ustrd"
	camt.Remittance.AdditionalTxInfo = "//Ntry/NtryDtls/TxDtls/AddtlTxInf"

	// Party information
	camt.Party.DebtorName = "//Ntry/NtryDtls/TxDtls/RltdPties/Dbtr/Nm"
	camt.Party.DebtorAgentName = "//Ntry/NtryDtls/TxDtls/RltdAgts/DbtrAgt/FinInstnId/Nm"
	camt.Party.CreditorName = "//Ntry/NtryDtls/TxDtls/RltdPties/Cdtr/Nm"
	camt.Party.CreditorAgentName = "//Ntry/NtryDtls/TxDtls/RltdAgts/CdtrAgt/FinInstnId/Nm"
	camt.Party.UltimateDebtor = "//Ntry/NtryDtls/TxDtls/RltdPties/UltmtDbtr/Nm"
	camt.Party.UltimateCreditor = "//Ntry/NtryDtls/TxDtls/RltdPties/UltmtCdtr/Nm"

	// Account information
	camt.Account.IBAN = "//BkToCstmrStmt/Stmt/Acct/Id/IBAN"

	return camt
}

// XPath expressions used for CAMT.053 XML parsing
// Deprecated: Use DefaultCamt053XPaths() instead
const (
	// Basic entry data
	XPathAmount         = "//Ntry/Amt"
	XPathCurrency       = "//Ntry/Amt/@Ccy"
	XPathCreditDebitInd = "//Ntry/CdtDbtInd" // #nosec G101 -- XPath expression, not credentials
	XPathBookingDate    = "//Ntry/BookgDt/Dt"
	XPathValueDate      = "//Ntry/ValDt/Dt"
	XPathStatus         = "//Ntry/Sts"
	XPathAccountSvcRef  = "//Ntry/AcctSvcrRef"

	// Transaction references
	XPathEndToEndID    = "//Ntry/NtryDtls/TxDtls/Refs/EndToEndId"
	XPathTransactionID = "//Ntry/NtryDtls/TxDtls/Refs/TxId"
	XPathPaymentInfoID = "//Ntry/NtryDtls/TxDtls/Refs/PmtInfId"

	// Remittance information
	XPathRemittanceInfo = "//Ntry/NtryDtls/TxDtls/RmtInf/Ustrd"
	XPathAddEntryInfo   = "//Ntry/AddtlNtryInf"
	XPathAddTxInfo      = "//Ntry/NtryDtls/TxDtls/AddtlTxInf"

	// Party information
	XPathDebtorName        = "//Ntry/NtryDtls/TxDtls/RltdPties/Dbtr/Nm"              // #nosec G101 -- XPath expression, not credentials
	XPathDebtorAgentName   = "//Ntry/NtryDtls/TxDtls/RltdAgts/DbtrAgt/FinInstnId/Nm" // #nosec G101 -- XPath expression, not credentials
	XPathCreditorName      = "//Ntry/NtryDtls/TxDtls/RltdPties/Cdtr/Nm"              // #nosec G101 -- XPath expression, not credentials
	XPathCreditorAgentName = "//Ntry/NtryDtls/TxDtls/RltdAgts/CdtrAgt/FinInstnId/Nm" // #nosec G101 -- XPath expression, not credentials
	XPathUltimateDebtor    = "//Ntry/NtryDtls/TxDtls/RltdPties/UltmtDbtr/Nm"
	XPathUltimateCreditor  = "//Ntry/NtryDtls/TxDtls/RltdPties/UltmtCdtr/Nm" // #nosec G101 -- XPath expression, not credentials

	// Account information
	XPathIBAN = "//BkToCstmrStmt/Stmt/Acct/Id/IBAN"

	// Bank transaction code
	XPathBankTxDomain    = "//Ntry/BkTxCd/Domn/Cd"
	XPathBankTxFamily    = "//Ntry/BkTxCd/Domn/Fmly/Cd"
	XPathBankTxSubFamily = "//Ntry/BkTxCd/Domn/Fmly/SubFmlyCd"
	XPathProprietaryCode = "//Ntry/BkTxCd/Prtry/Cd"
)
