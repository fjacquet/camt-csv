package camtparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Swiss banks prefix descriptions with the payment channel. The merchant is
// what matters for categorization, so the marker must come off — unless the
// marker is the whole name, which has its own categorization rules.
func TestCleanPaymentMethodPrefixes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"card payment", "PMT CARTE Coop Pronto", "Coop Pronto"},
		{"twint payment", "PMT TWINT Boulangerie", "Boulangerie"},
		{"e-banking transfer", "BCV-NET Assura", "Assura"},
		{"counter transfer", "VIRT BANC Jean Dupont", "Jean Dupont"},
		{"bare card marker kept", "PMT CARTE", "PMT CARTE"},
		{"bare twint marker kept", "PMT TWINT", "PMT TWINT"},
		{"bare e-banking marker kept", "BCV-NET", "BCV-NET"},
		{"bare counter marker kept", "VIRT BANC", "VIRT BANC"},
		{"marker with only spaces after it", "PMT CARTE   ", "PMT CARTE   "},
		{"no marker passes through", "Migros Lausanne", "Migros Lausanne"},
		{"e-banking transfer label", "BCV-NET Transfer", "Transfer"},
		{"marker not at the start is kept", "Paid via PMT CARTE", "Paid via PMT CARTE"},
		{"empty input", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cleanPaymentMethodPrefixes(tt.input))
		})
	}
}

func TestExtractPartyNameFromDescription(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"card payment", "PMT CARTE Coop Pronto", "Coop Pronto"},
		{"twint payment", "PMT TWINT Boulangerie du Coin", "Boulangerie du Coin"},
		{"counter transfer", "VIRT BANC Jean Dupont", "Jean Dupont"},
		{"e-banking transfer", "BCV-NET Assura SA", "Assura SA"},
		{"marker with nothing after it", "PMT CARTE", ""},
		{"unprefixed description", "Salaire mensuel", ""},
		{"empty input", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractPartyNameFromDescription(tt.input))
		})
	}
}

func TestSetTransactionTypeFromDescription(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"twint", "PMT TWINT Boulangerie", "TWINT"},
		{"card", "PMT CARTE Coop", "CB"},
		{"counter transfer", "VIRT BANC Jean", "Virement"},
		{"e-banking transfer", "BCV-NET Assura", "Virement"},
		{"exact LSV order", "ORDRE LSV +", "Virement"},
		{"LSV order with a suffix is not matched", "ORDRE LSV + Assura", ""},
		{"unknown description", "Salaire", ""},
		{"empty input", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, setTransactionTypeFromDescription(tt.input))
		})
	}
}

func TestIsIBANFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"swiss IBAN", "CH9300762011623852957", true},
		{"german IBAN", "DE89370400440532013000", true},
		{"french IBAN", "FR1420041010050500013M02606", true},
		{"lower-case country code", "ch9300762011623852957", true},
		{"too short", "CH930076201", false},
		{"too long", "CH93007620116238529571234567890123456", false},
		{"digits in the country-code slot", "1230076201162385295", false},
		{"all digits, no country code", "1234567890123456", false},
		{"second character not a letter", "C13007620116238529571", false},
		{"contains punctuation", "CH93-0076-2011-6238-52957", false},
		{"contains a space", "CH93 0076 2011 6238 5295", false},
		{"empty input", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isIBANFormat(tt.input))
		})
	}
}

// Banks populate different reference elements; the first non-empty wins, in
// the order MsgId, EndToEndId, TxId, AcctSvcrRef.
func TestResolveReference(t *testing.T) {
	tests := []struct {
		name string
		refs camtReference
		want string
	}{
		{"message id preferred", camtReference{MsgID: "M1", EndToEndID: "E1", TxID: "T1", AcctSvcrRef: "A1"}, "M1"},
		{"end-to-end id next", camtReference{EndToEndID: "E1", TxID: "T1", AcctSvcrRef: "A1"}, "E1"},
		{"transaction id next", camtReference{TxID: "T1", AcctSvcrRef: "A1"}, "T1"},
		{"servicer ref last", camtReference{AcctSvcrRef: "A1"}, "A1"},
		{"instruction id is not used", camtReference{InstrID: "I1"}, ""},
		{"nothing populated", camtReference{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveReference(tt.refs))
		})
	}
}

func TestResolvePartyIBAN(t *testing.T) {
	const iban = "CH9300762011623852957"

	tests := []struct {
		name    string
		details camtTransactionDetails
		want    string
	}{
		{
			name: "debtor account IBAN preferred",
			details: camtTransactionDetails{RelatedParties: camtRelatedParties{
				Debtor:   camtParty{Account: camtAccount{IBAN: iban}},
				Creditor: camtParty{Account: camtAccount{IBAN: "CH0000000000000000000"}},
			}},
			want: iban,
		},
		{
			name: "creditor account IBAN when debtor has none",
			details: camtTransactionDetails{RelatedParties: camtRelatedParties{
				Creditor: camtParty{Account: camtAccount{IBAN: iban}},
			}},
			want: iban,
		},
		{
			name: "party-level debtor account",
			details: camtTransactionDetails{RelatedParties: camtRelatedParties{
				DebtorAccount: camtAccount{IBAN: iban},
			}},
			want: iban,
		},
		{
			name: "related-accounts fallback",
			details: camtTransactionDetails{RelatedAccounts: camtRelatedAccounts{
				CreditorAccount: camtAccount{IBAN: iban},
			}},
			want: iban,
		},
		{
			name: "IBAN stored in the generic Othr/Id field",
			details: camtTransactionDetails{RelatedParties: camtRelatedParties{
				Debtor: camtParty{Account: camtAccount{ID: iban}},
			}},
			want: iban,
		},
		{
			name: "non-IBAN Othr/Id is ignored",
			details: camtTransactionDetails{RelatedParties: camtRelatedParties{
				Debtor: camtParty{Account: camtAccount{ID: "12345"}},
			}},
			want: "",
		},
		{"nothing populated", camtTransactionDetails{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolvePartyIBAN(tt.details))
		})
	}
}

// "ORDRE LSV +" is a Swiss direct-debit order: the counterparty is the
// creditor, not the debtor, and the type is always a transfer.
func TestResolveParty(t *testing.T) {
	parties := camtRelatedParties{
		Debtor:   camtParty{Name: "Jean Dupont"},
		Creditor: camtParty{Name: "Assura SA"},
	}

	tests := []struct {
		name        string
		description string
		parties     camtRelatedParties
		wantName    string
		wantType    string
	}{
		{"LSV order uses the creditor", "ORDRE LSV + something", parties, "Assura SA", "Virement"},
		{"LSV order without a creditor yields nothing", "ORDRE LSV +", camtRelatedParties{Debtor: camtParty{Name: "Jean"}}, "", ""},
		{"description prefix wins over the debtor", "PMT CARTE Coop Pronto", parties, "Coop Pronto", ""},
		{"debtor name is the fallback", "Salaire mensuel", parties, "Jean Dupont", ""},
		{"empty description falls back to the debtor", "", parties, "Jean Dupont", ""},
		{"nothing available", "Salaire", camtRelatedParties{}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotType := resolveParty(tt.description, tt.parties)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantType, gotType)
		})
	}
}

// A missing or malformed date must not fail the entry: CAMT does not require
// both booking and value dates to be present.
func TestParseCAMTDate(t *testing.T) {
	assert.Equal(t, 2026, parseCAMTDate("2026-03-15").Year())
	assert.True(t, parseCAMTDate("").IsZero())
	assert.True(t, parseCAMTDate("15.03.2026").IsZero())
	assert.True(t, parseCAMTDate("not-a-date").IsZero())
}
