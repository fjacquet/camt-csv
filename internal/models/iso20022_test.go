package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntry_GetFirstTxDetails(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected *TransactionDetails
	}{
		{
			name: "entry with transaction details",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							Refs: References{
								MsgID: "MSG123",
							},
						},
					},
				},
			},
			expected: &TransactionDetails{
				Refs: References{
					MsgID: "MSG123",
				},
			},
		},
		{
			name: "entry without transaction details",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetFirstTxDetails()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Refs.MsgID, result.Refs.MsgID)
			}
		})
	}
}

func TestEntry_GetCreditDebit(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "debit entry",
			entry: Entry{
				CdtDbtInd: TransactionTypeDebit,
			},
			expected: TransactionTypeDebit,
		},
		{
			name: "credit entry",
			entry: Entry{
				CdtDbtInd: TransactionTypeCredit,
			},
			expected: TransactionTypeCredit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetCreditDebit()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_IsCredit(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected bool
	}{
		{
			name: "credit entry",
			entry: Entry{
				CdtDbtInd: TransactionTypeCredit,
			},
			expected: true,
		},
		{
			name: "debit entry",
			entry: Entry{
				CdtDbtInd: TransactionTypeDebit,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.IsCredit()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_GetBankTxCode(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "domain code with family and subfamily",
			entry: Entry{
				BkTxCd: BankTxCode{
					Domn: struct {
						Cd   string `xml:"Cd"`
						Fmly struct {
							Cd        string `xml:"Cd"`
							SubFmlyCd string `xml:"SubFmlyCd"`
						} `xml:"Fmly"`
					}{
						Cd: "PMNT",
						Fmly: struct {
							Cd        string `xml:"Cd"`
							SubFmlyCd string `xml:"SubFmlyCd"`
						}{
							Cd:        "RCDT",
							SubFmlyCd: "ESCT",
						},
					},
				},
			},
			expected: "PMNT/RCDT/ESCT",
		},
		{
			name: "domain code only",
			entry: Entry{
				BkTxCd: BankTxCode{
					Domn: struct {
						Cd   string `xml:"Cd"`
						Fmly struct {
							Cd        string `xml:"Cd"`
							SubFmlyCd string `xml:"SubFmlyCd"`
						} `xml:"Fmly"`
					}{
						Cd: "PMNT",
					},
				},
			},
			expected: "PMNT",
		},
		{
			name: "proprietary code",
			entry: Entry{
				BkTxCd: BankTxCode{
					Prtry: struct {
						Cd   string `xml:"Cd"`
						Issr string `xml:"Issr"`
					}{
						Cd: "CUSTOM",
					},
				},
			},
			expected: "CUSTOM",
		},
		{
			name:     "no code",
			entry:    Entry{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetBankTxCode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_GetRemittanceInfo(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "single remittance info",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RmtInf: struct {
								Ustrd []string `xml:"Ustrd"`
								Strd  []struct {
									CdtrRefInf struct {
										Ref string `xml:"Ref"`
									} `xml:"CdtrRefInf"`
								} `xml:"Strd"`
							}{
								Ustrd: []string{"Payment for invoice 123"},
							},
						},
					},
				},
			},
			expected: "Payment for invoice 123",
		},
		{
			name: "multiple remittance info",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RmtInf: struct {
								Ustrd []string `xml:"Ustrd"`
								Strd  []struct {
									CdtrRefInf struct {
										Ref string `xml:"Ref"`
									} `xml:"CdtrRefInf"`
								} `xml:"Strd"`
							}{
								Ustrd: []string{"Payment 1", "Payment 2"},
							},
						},
					},
				},
			},
			expected: "Payment 1, Payment 2",
		},
		{
			name:     "no remittance info",
			entry:    Entry{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetRemittanceInfo()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_GetPayer(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "credit transaction with ultimate debtor",
			entry: Entry{
				CdtDbtInd: TransactionTypeCredit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdPties: RelatedParties{
								UltmtDbtr: struct {
									Nm string `xml:"Nm"`
								}{
									Nm: "Ultimate Payer",
								},
							},
						},
					},
				},
			},
			expected: "Ultimate Payer",
		},
		{
			name: "credit transaction with debtor",
			entry: Entry{
				CdtDbtInd: TransactionTypeCredit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdPties: RelatedParties{
								Dbtr: struct {
									Nm      string `xml:"Nm"`
									PstlAdr struct {
										AdrLine []string `xml:"AdrLine"`
										StrtNm  string   `xml:"StrtNm"`
										BldgNb  string   `xml:"BldgNb"`
										PstCd   string   `xml:"PstCd"`
										TwnNm   string   `xml:"TwnNm"`
										Ctry    string   `xml:"Ctry"`
									} `xml:"PstlAdr"`
								}{
									Nm: "Regular Payer",
								},
							},
						},
					},
				},
			},
			expected: "Regular Payer",
		},
		{
			name: "credit transaction with debtor agent",
			entry: Entry{
				CdtDbtInd: TransactionTypeCredit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdAgts: RelatedAgents{
								DbtrAgt: struct {
									FinInstnID struct {
										BIC     string `xml:"BIC"`
										Nm      string `xml:"Nm"`
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
								}{
									FinInstnID: struct {
										BIC     string `xml:"BIC"`
										Nm      string `xml:"Nm"`
										PstlAdr struct {
											AdrLine []string `xml:"AdrLine"`
										} `xml:"PstlAdr"`
										ClrSysMmbID struct {
											ClrSysID struct {
												Cd string `xml:"Cd"`
											} `xml:"ClrSysId"`
											MmbID string `xml:"MmbId"`
										} `xml:"ClrSysMmbId"`
									}{
										Nm: "Payer Bank",
									},
								},
							},
						},
					},
				},
			},
			expected: "Payer Bank",
		},
		{
			name: "debit transaction",
			entry: Entry{
				CdtDbtInd: TransactionTypeDebit,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetPayer()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_GetPayee(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "debit transaction with ultimate creditor",
			entry: Entry{
				CdtDbtInd: TransactionTypeDebit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdPties: RelatedParties{
								UltmtCdtr: struct {
									Nm string `xml:"Nm"`
								}{
									Nm: "Ultimate Payee",
								},
							},
						},
					},
				},
			},
			expected: "Ultimate Payee",
		},
		{
			name: "debit transaction with creditor",
			entry: Entry{
				CdtDbtInd: TransactionTypeDebit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdPties: RelatedParties{
								Cdtr: struct {
									Nm      string `xml:"Nm"`
									PstlAdr struct {
										AdrLine []string `xml:"AdrLine"`
										StrtNm  string   `xml:"StrtNm"`
										BldgNb  string   `xml:"BldgNb"`
										PstCd   string   `xml:"PstCd"`
										TwnNm   string   `xml:"TwnNm"`
										Ctry    string   `xml:"Ctry"`
									} `xml:"PstlAdr"`
								}{
									Nm: "Regular Payee",
								},
							},
						},
					},
				},
			},
			expected: "Regular Payee",
		},
		{
			name: "debit transaction with creditor agent",
			entry: Entry{
				CdtDbtInd: TransactionTypeDebit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdAgts: RelatedAgents{
								CdtrAgt: struct {
									FinInstnID struct {
										BIC     string `xml:"BIC"`
										Nm      string `xml:"Nm"`
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
								}{
									FinInstnID: struct {
										BIC     string `xml:"BIC"`
										Nm      string `xml:"Nm"`
										PstlAdr struct {
											AdrLine []string `xml:"AdrLine"`
										} `xml:"PstlAdr"`
										ClrSysMmbID struct {
											ClrSysID struct {
												Cd string `xml:"Cd"`
											} `xml:"ClrSysId"`
											MmbID string `xml:"MmbId"`
										} `xml:"ClrSysMmbId"`
									}{
										Nm: "Payee Bank",
									},
								},
							},
						},
					},
				},
			},
			expected: "Payee Bank",
		},
		{
			name: "credit transaction",
			entry: Entry{
				CdtDbtInd: TransactionTypeCredit,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetPayee()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_GetReference(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "entry reference",
			entry: Entry{
				NtryRef: "ENTRY123",
			},
			expected: "ENTRY123",
		},
		{
			name: "end to end ID",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							Refs: References{
								EndToEndID: "E2E123",
							},
						},
					},
				},
			},
			expected: "E2E123",
		},
		{
			name: "transaction ID",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							Refs: References{
								TxID: "TX123",
							},
						},
					},
				},
			},
			expected: "TX123",
		},
		{
			name: "payment info ID",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							Refs: References{
								PmtInfID: "PMT123",
							},
						},
					},
				},
			},
			expected: "PMT123",
		},
		{
			name: "message ID",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							Refs: References{
								MsgID: "MSG123",
							},
						},
					},
				},
			},
			expected: "MSG123",
		},
		{
			name: "instruction ID",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							Refs: References{
								InstrID: "INSTR123",
							},
						},
					},
				},
			},
			expected: "INSTR123",
		},
		{
			name: "structured reference",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RmtInf: struct {
								Ustrd []string `xml:"Ustrd"`
								Strd  []struct {
									CdtrRefInf struct {
										Ref string `xml:"Ref"`
									} `xml:"CdtrRefInf"`
								} `xml:"Strd"`
							}{
								Strd: []struct {
									CdtrRefInf struct {
										Ref string `xml:"Ref"`
									} `xml:"CdtrRefInf"`
								}{
									{
										CdtrRefInf: struct {
											Ref string `xml:"Ref"`
										}{
											Ref: "STRD123",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "STRD123",
		},
		{
			name:     "no reference",
			entry:    Entry{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetReference()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_GetIBAN(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "debit transaction with creditor IBAN",
			entry: Entry{
				CdtDbtInd: TransactionTypeDebit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdPties: RelatedParties{
								CdtrAcct: struct {
									ID struct {
										IBAN string `xml:"IBAN"`
										Othr struct {
											ID string `xml:"Id"`
										} `xml:"Othr"`
									} `xml:"Id"`
								}{
									ID: struct {
										IBAN string `xml:"IBAN"`
										Othr struct {
											ID string `xml:"Id"`
										} `xml:"Othr"`
									}{
										IBAN: "CH9300762011623852957",
									},
								},
							},
						},
					},
				},
			},
			expected: "CH9300762011623852957",
		},
		{
			name: "credit transaction with debtor IBAN",
			entry: Entry{
				CdtDbtInd: TransactionTypeCredit,
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RltdPties: RelatedParties{
								DbtrAcct: struct {
									ID struct {
										IBAN string `xml:"IBAN"`
										Othr struct {
											ID string `xml:"Id"`
										} `xml:"Othr"`
									} `xml:"Id"`
								}{
									ID: struct {
										IBAN string `xml:"IBAN"`
										Othr struct {
											ID string `xml:"Id"`
										} `xml:"Othr"`
									}{
										IBAN: "CH9300762011623852957",
									},
								},
							},
						},
					},
				},
			},
			expected: "CH9300762011623852957",
		},
		{
			name:     "no IBAN",
			entry:    Entry{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetIBAN()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntry_BuildDescription(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected string
	}{
		{
			name: "with additional entry info",
			entry: Entry{
				AddtlNtryInf: "Payment received",
			},
			expected: "Payment received",
		},
		{
			name: "with transaction additional info",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							AddtlTxInf: "Additional transaction info",
						},
					},
				},
			},
			expected: "Additional transaction info",
		},
		{
			name: "with remittance info",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							RmtInf: struct {
								Ustrd []string `xml:"Ustrd"`
								Strd  []struct {
									CdtrRefInf struct {
										Ref string `xml:"Ref"`
									} `xml:"CdtrRefInf"`
								} `xml:"Strd"`
							}{
								Ustrd: []string{"Invoice 123", "Payment for services"},
							},
						},
					},
				},
			},
			expected: "Invoice 123 - Payment for services",
		},
		{
			name: "with purpose code",
			entry: Entry{
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							Purp: struct {
								Cd string `xml:"Cd"`
							}{
								Cd: "SALA",
							},
						},
					},
				},
			},
			expected: "Purpose: SALA",
		},
		{
			name: "combined description",
			entry: Entry{
				AddtlNtryInf: "Payment received",
				NtryDtls: EntryDetails{
					TxDtls: []TransactionDetails{
						{
							AddtlTxInf: "Additional info",
							RmtInf: struct {
								Ustrd []string `xml:"Ustrd"`
								Strd  []struct {
									CdtrRefInf struct {
										Ref string `xml:"Ref"`
									} `xml:"CdtrRefInf"`
								} `xml:"Strd"`
							}{
								Ustrd: []string{"Invoice 123"},
							},
							Purp: struct {
								Cd string `xml:"Cd"`
							}{
								Cd: "SALA",
							},
						},
					},
				},
			},
			expected: "Payment received - Additional info - Invoice 123 - Purpose: SALA",
		},
		{
			name: "empty description with reference",
			entry: Entry{
				NtryRef: "REF123",
			},
			expected: "Transaction REF123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.BuildDescription()
			assert.Equal(t, tt.expected, result)
		})
	}
}
