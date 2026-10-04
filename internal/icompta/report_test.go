package icompta

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleReport() Report {
	return Report{
		DBState: "splits=3;split_modified=2026-10-04 19:32:27;tx_modified=2026-10-04 19:49:30",
		Rows: []Row{
			{SplitID: "S1", Date: "2026-01-02", Name: "Café \"Chez Paul\", Genève\nterrasse", Amount: "-12.50",
				OldCategory: "Divers", NewCategory: "Restaurants", Tier: TierKeyword, Decision: ActionChange, Apply: true},
			{SplitID: "S2", Date: "2026-01-03", Name: "# looks like a comment", Amount: "8.7",
				OldCategory: "", NewCategory: "", Decision: ActionKeep, Reason: ReasonNoSuggestion},
			{SplitID: "S3", Date: "2026-01-04", Name: "X", Amount: "1",
				OldCategory: "Divers", NewCategory: "Nope", Tier: TierAI, Decision: ActionSkip, Reason: "unknown category"},
		},
	}
}

func TestReport_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, sampleReport().Write(&buf))

	got, err := ReadReport(&buf)
	require.NoError(t, err)
	assert.Equal(t, sampleReport(), got)
}

// A report edited in a spreadsheet comes back with a BOM, CRLF line endings,
// trailing commas on the comment lines, and a hand-typed "YES".
func TestReadReport_SpreadsheetEdited(t *testing.T) {
	in := "\xef\xbb\xbf# camt-csv recategorize report v1,,,,,,,,,\r\n" +
		"# db_state=splits=1;split_modified=a;tx_modified=b,,,,,,,,,\r\n" +
		"split_id,date,name,amount,old_category,new_category,tier,decision,reason,apply\r\n" +
		"S1,2026-01-02,Coop,-5,Divers,Alimentation,keyword,change,,YES\r\n"

	got, err := ReadReport(strings.NewReader(in))
	require.NoError(t, err)
	assert.Equal(t, "splits=1;split_modified=a;tx_modified=b", got.DBState)
	require.Len(t, got.Rows, 1)
	assert.True(t, got.Rows[0].Apply)
	assert.Equal(t, ActionChange, got.Rows[0].Decision)
}

func TestReadReport_Errors(t *testing.T) {
	const header = "split_id,date,name,amount,old_category,new_category,tier,decision,reason,apply\n"
	tests := []struct {
		name, in, want string
	}{
		{"not a report", "a,b,c\n", "not a camt-csv recategorize report"},
		{"missing state", "# camt-csv recategorize report v1\n" + header, "db_state"},
		{"wrong header", "# camt-csv recategorize report v1\n# db_state=x\nfoo,bar\n", "header"},
		{"bad apply", "# camt-csv recategorize report v1\n# db_state=x\n" + header +
			"S1,d,n,1,a,b,keyword,change,,maybe\n", "line 4"},
		{"short row", "# camt-csv recategorize report v1\n# db_state=x\n" + header + "S1,d,n\n", "line 4"},
		{"bad decision", "# camt-csv recategorize report v1\n# db_state=x\n" + header +
			"S1,d,n,1,a,b,keyword,explode,,yes\n", "line 4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadReport(strings.NewReader(tt.in))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}
