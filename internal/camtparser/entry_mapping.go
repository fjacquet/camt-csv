package camtparser

import (
	"strings"
	"time"

	"fjacquet/camt-csv/internal/dateutils"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// entryToTransaction maps one CAMT.053 statement entry onto a Transaction.
//
// If the builder rejects the assembled values, a minimal fallback transaction
// is returned instead so that one malformed entry does not abort a statement.
func (a *Adapter) entryToTransaction(entry camtEntry) models.Transaction {
	bookingDate := parseCAMTDate(entry.BookingDate.Date)
	valueDate := parseCAMTDate(entry.ValueDate.Date)

	builder := models.NewTransactionBuilder().
		WithID(""). // CAMT entries carry their own references; do not mint a UUID
		WithDatetime(bookingDate).
		WithValueDatetime(valueDate).
		WithAmount(models.ParseAmount(entry.Amount.Value), entry.Amount.Currency).
		WithAccountServicer(entry.AccountServicer.Ref).
		WithStatus(entry.Status.Status)

	if entry.CreditDebit.Indicator == models.TransactionTypeDebit {
		builder = builder.AsDebit()
	} else {
		builder = builder.AsCredit()
	}

	txDetails := entry.EntryDetails.TransactionDetails

	// AddtlNtryInf is the bank's own annotation and is preferred; remittance
	// text is the fallback description.
	description := entry.AdditionalInfo.Info
	if description == "" {
		description = txDetails.RemittanceInfo.Ustrd
	}
	if description != "" {
		builder = builder.WithDescription(description)
	}

	// RemittanceInfo is recorded separately whenever present, even when it was
	// also used as the description.
	if txDetails.RemittanceInfo.Ustrd != "" {
		builder = builder.WithRemittanceInfo(txDetails.RemittanceInfo.Ustrd)
	}

	partyName, transactionType := resolveParty(description, txDetails.RelatedParties)
	if partyName != "" {
		builder = builder.WithPartyName(partyName)
	}
	if transactionType == "" {
		transactionType = setTransactionTypeFromDescription(description)
	}
	if transactionType != "" {
		builder = builder.WithType(transactionType)
	}

	if iban := resolvePartyIBAN(txDetails); iban != "" {
		builder = builder.WithPartyIBAN(iban)
	}

	reference := resolveReference(txDetails.References)
	if reference != "" {
		builder = builder.WithReference(reference)
	}

	transaction, err := builder.Build()
	if err != nil {
		a.GetLogger().WithError(err).Warn("Failed to build transaction, using fallback",
			logging.Field{Key: "entry_reference", Value: reference})

		transaction, _ = models.NewTransactionBuilder().
			WithDatetime(bookingDate).
			WithAmount(models.ParseAmount(entry.Amount.Value), entry.Amount.Currency).
			WithDescription("Failed to parse transaction").
			Build()
	}

	// Name and Payee/Payer are set here so that UpdateNameFromParties does not
	// overwrite Name during export.
	if transaction.Name == "" {
		transaction.Name = transaction.PartyName
	}
	if transaction.IsDebit() {
		transaction.Payee = transaction.PartyName
	} else {
		transaction.Payer = transaction.PartyName
	}

	transaction.UpdateDebitCreditAmounts()

	return transaction
}

// parseCAMTDate parses an ISO date, returning the zero time when the value is
// absent or malformed — CAMT entries are not required to carry both dates.
func parseCAMTDate(value string) time.Time {
	parsed, err := time.Parse(dateutils.DateLayoutISO, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// resolveParty determines the counterparty name and, where the description
// implies it, the transaction type.
//
// "ORDRE LSV +" marks a Swiss direct-debit order whose counterparty is the
// creditor rather than the debtor, so it is handled separately.
func resolveParty(description string, parties camtRelatedParties) (name, transactionType string) {
	if strings.Contains(description, "ORDRE LSV +") {
		if parties.Creditor.Name != "" {
			return parties.Creditor.Name, "Virement"
		}
		return "", ""
	}

	if description != "" {
		if extracted := extractPartyNameFromDescription(description); extracted != "" {
			return extracted, ""
		}
	}

	return parties.Debtor.Name, ""
}

// resolvePartyIBAN finds the counterparty IBAN. Banks place it in several
// different elements, and some use the generic Othr/Id field, so the candidates
// are tried in order of specificity.
func resolvePartyIBAN(txDetails camtTransactionDetails) string {
	parties := txDetails.RelatedParties
	accounts := txDetails.RelatedAccounts

	candidates := []string{
		parties.Debtor.Account.IBAN,
		parties.Creditor.Account.IBAN,
		parties.DebtorAccount.IBAN,
		parties.CreditorAccount.IBAN,
		accounts.DebtorAccount.IBAN,
		accounts.CreditorAccount.IBAN,
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}

	// Fall back to generic account identifiers that look like an IBAN.
	for _, candidate := range []string{parties.Debtor.Account.ID, parties.Creditor.Account.ID} {
		if candidate != "" && isIBANFormat(candidate) {
			return candidate
		}
	}

	return ""
}

// resolveReference picks the first identifier the bank populated.
func resolveReference(refs camtReference) string {
	for _, candidate := range []string{refs.MsgID, refs.EndToEndID, refs.TxID, refs.AcctSvcrRef} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

// isIBANFormat reports whether s has the shape of an IBAN: 15-34 characters,
// two leading letters for the country code, alphanumeric thereafter.
func isIBANFormat(s string) bool {
	if len(s) < 15 || len(s) > 34 {
		return false
	}

	if !isASCIILetter(s[0]) || !isASCIILetter(s[1]) {
		return false
	}

	for i := 2; i < len(s); i++ {
		c := s[i]
		if !isASCIILetter(c) && !('0' <= c && c <= '9') {
			return false
		}
	}

	return true
}

func isASCIILetter(c byte) bool {
	return ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z')
}

// paymentMethodPrefixes are the terminal/channel markers Swiss banks prepend to
// a description: card payment, TWINT, e-banking transfer, counter transfer.
var paymentMethodPrefixes = []string{"PMT CARTE", "PMT TWINT", "BCV-NET", "VIRT BANC"}

// cleanPaymentMethodPrefixes strips a leading payment-method marker from a
// party name so that categorization sees the merchant rather than the channel.
//
// A name consisting of nothing but the marker is left untouched: those values
// have their own categorization rules.
func cleanPaymentMethodPrefixes(partyName string) string {
	for _, prefix := range paymentMethodPrefixes {
		if partyName == prefix {
			return partyName
		}
	}

	for _, prefix := range paymentMethodPrefixes {
		if strings.HasPrefix(partyName, prefix) {
			if cleaned := strings.TrimSpace(partyName[len(prefix):]); cleaned != "" {
				return cleaned
			}
			return partyName
		}
	}

	return partyName
}

// extractPartyNameFromDescription pulls the merchant out of a description that
// begins with a payment-method marker, e.g. "PMT CARTE Coop Pronto".
func extractPartyNameFromDescription(description string) string {
	for _, prefix := range []string{"PMT TWINT", "PMT CARTE", "VIRT BANC", "BCV-NET"} {
		if strings.HasPrefix(description, prefix) {
			if remaining := strings.TrimSpace(description[len(prefix):]); remaining != "" {
				return remaining
			}
		}
	}
	return ""
}

// setTransactionTypeFromDescription derives the payment channel from the
// description's leading marker.
func setTransactionTypeFromDescription(description string) string {
	switch {
	case strings.HasPrefix(description, "PMT TWINT"):
		return "TWINT"
	case strings.HasPrefix(description, "PMT CARTE"):
		return "CB"
	case strings.HasPrefix(description, "VIRT BANC"):
		return "Virement"
	case strings.HasPrefix(description, "BCV-NET"):
		return "Virement"
	case description == "ORDRE LSV +":
		return "Virement"
	}
	return ""
}
