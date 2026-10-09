package icompta

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samplePairs() []Pair {
	d := Leg{SplitID: "S1", Date: "2026-01-10", AccountName: "Compte privé", Name: "=HYPERLINK(1)",
		Amount: decimal.RequireFromString("-750"), Category: "Virements"}
	c := Leg{SplitID: "S2", Date: "2026-01-11", AccountName: "Selma", Name: "DEPOT",
		Amount: decimal.RequireFromString("750"), Category: "Virements"}
	return []Pair{
		{Debit: d, Credit: c, GapDays: 1, Sure: true},
		{Debit: d, Credit: c, GapDays: 3, Reason: "gap 3 days", Note: "categories differ"},
	}
}

func TestTransferReport_RoundTrip(t *testing.T) {
	rep := NewTransferReport("splits=8;x", samplePairs())
	require.Len(t, rep.Rows, 2)
	assert.Equal(t, "750.00", rep.Rows[0].Amount)
	assert.Equal(t, "sure", rep.Rows[0].Confidence)
	assert.True(t, rep.Rows[0].Apply)
	assert.Equal(t, "doubtful", rep.Rows[1].Confidence)
	assert.False(t, rep.Rows[1].Apply)

	var buf bytes.Buffer
	require.NoError(t, rep.Write(&buf))
	assert.True(t, strings.HasPrefix(buf.String(), "# camt-csv link-transfers report v1\n# db_state=splits=8;x\n"))
	assert.Contains(t, buf.String(), "'=HYPERLINK(1)", "formula escaped in the file")

	back, err := ReadTransferReport(&buf)
	require.NoError(t, err)
	assert.Equal(t, rep, back)
}

// Review focus 5: what Numbers/Excel do to a CSV.
func TestReadTransferReport_SpreadsheetEdited(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewTransferReport("st", samplePairs()).Write(&buf))
	edited := "\xEF\xBB\xBF" + strings.ReplaceAll(buf.String(), "\n", "\r\n")
	edited = strings.Replace(edited, "# db_state=st\r\n", "# db_state=st,,,\r\n", 1)
	edited = strings.Replace(edited, ",yes\r\n", ",Yes\r\n", 1)

	rep, err := ReadTransferReport(strings.NewReader(edited))
	require.NoError(t, err)
	assert.Equal(t, "st", rep.DBState)
	assert.True(t, rep.Rows[0].Apply)
}

func TestReadTransferReport_Errors(t *testing.T) {
	var good bytes.Buffer
	require.NoError(t, NewTransferReport("st", samplePairs()).Write(&good))
	for name, tc := range map[string]struct{ in, want string }{
		"recategorize report": {"# camt-csv recategorize report v1\n# db_state=st\n", "not a camt-csv link-transfers report"},
		"no state":            {"# camt-csv link-transfers report v1\n", "no db_state line"},
		"bad header":          {"# camt-csv link-transfers report v1\n# db_state=st\na,b\n", "report header is not"},
		"bad apply":           {strings.Replace(good.String(), ",yes\n", ",maybe\n", 1), `line 4: apply "maybe"`},
		"bad confidence":      {strings.Replace(good.String(), ",sure,", ",certain,", 1), `line 4: confidence "certain"`},
		"bad gap":             {strings.Replace(good.String(), ",2026-01-11,1,", ",2026-01-11,x,", 1), `line 4: gap_days "x"`},
	} {
		_, err := ReadTransferReport(strings.NewReader(tc.in))
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), tc.want, name)
	}
}

func TestReadReport_RejectsATransferReport(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, NewTransferReport("st", samplePairs()).Write(&buf))
	_, err := ReadReport(&buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a camt-csv recategorize report")
}
