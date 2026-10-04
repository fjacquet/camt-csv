package icompta

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

const (
	reportMagic = "# camt-csv recategorize report v1"
	stateKey    = "# db_state="
)

var reportHeader = []string{
	"split_id", "date", "name", "amount", "old_category", "new_category",
	"tier", "decision", "reason", "apply",
}

// Row is one reviewed decision. Only rows with Decision == ActionChange and
// Apply == true are ever written to the database.
type Row struct {
	SplitID     string
	Date        string
	Name        string
	Amount      string
	OldCategory string
	NewCategory string
	Tier        string
	Decision    Action
	Reason      string
	Apply       bool
}

// Report is what preview writes and apply reads.
type Report struct {
	DBState string
	Rows    []Row
}

// formulaTriggers are the leading characters a spreadsheet treats as the start
// of a formula (OWASP, CSV injection).
const formulaTriggers = "=+-@\t\r"

// needsEscape reports whether a spreadsheet could read s as a formula, or s is
// already shaped like an escaped value and so would not round-trip unescaped.
func needsEscape(s string) bool {
	if s == "" {
		return false
	}
	if strings.ContainsRune(formulaTriggers, rune(s[0])) {
		return true
	}
	return s[0] == '\'' && len(s) > 1 && strings.ContainsRune(formulaTriggers, rune(s[1]))
}

// escapeCell neutralises text that comes from the database, which a spreadsheet
// would otherwise evaluate. A leading apostrophe is the spreadsheet convention
// for "this is text".
func escapeCell(s string) string {
	if needsEscape(s) {
		return "'" + s
	}
	return s
}

// unescapeCell reverses escapeCell exactly: it strips one leading apostrophe
// only when what remains is something escapeCell would have escaped.
func unescapeCell(s string) string {
	if len(s) > 1 && s[0] == '\'' && needsEscape(s[1:]) {
		return s[1:]
	}
	return s
}

// Write encodes the report: two comment lines, then a CSV with a header.
func (r Report) Write(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s\n%s%s\n", reportMagic, stateKey, r.DBState); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(reportHeader); err != nil {
		return err
	}
	for _, row := range r.Rows {
		apply := "no"
		if row.Apply {
			apply = "yes"
		}
		rec := []string{
			row.SplitID, row.Date, escapeCell(row.Name), row.Amount,
			escapeCell(row.OldCategory), escapeCell(row.NewCategory),
			row.Tier, string(row.Decision), escapeCell(row.Reason), apply,
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ReadReport decodes a report, tolerating what a spreadsheet does to a CSV:
// a UTF-8 BOM, CRLF line endings, trailing commas on the comment lines and any
// case of yes/no.
func ReadReport(rd io.Reader) (Report, error) {
	br := bufio.NewReader(rd)
	if b, err := br.Peek(3); err == nil && bytes.Equal(b, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}

	var rep Report
	magicSeen := false
	for {
		first, err := br.Peek(1)
		if err != nil || first[0] != '#' {
			break
		}
		line, err := br.ReadString('\n')
		line = strings.TrimRight(line, ", \r\n")
		switch {
		case line == reportMagic:
			magicSeen = true
		case strings.HasPrefix(line, stateKey):
			rep.DBState = strings.TrimPrefix(line, stateKey)
		}
		if err != nil {
			break
		}
	}
	if !magicSeen {
		return Report{}, fmt.Errorf("not a camt-csv recategorize report (missing %q line)", reportMagic)
	}
	if rep.DBState == "" {
		return Report{}, fmt.Errorf("report has no db_state line")
	}

	cr := csv.NewReader(br)
	cr.FieldsPerRecord = -1 // checked below so a wrong header reads as a header error
	records, err := cr.ReadAll()
	if err != nil {
		return Report{}, fmt.Errorf("read report rows: %w", err)
	}
	if len(records) == 0 || strings.Join(records[0], ",") != strings.Join(reportHeader, ",") {
		return Report{}, fmt.Errorf("report header is not %q", strings.Join(reportHeader, ","))
	}

	for i, rec := range records[1:] {
		lineNo := i + 4 // 2 comment lines, 1 header, 1-based
		if len(rec) != len(reportHeader) {
			return Report{}, fmt.Errorf("line %d: %d fields, expected %d", lineNo, len(rec), len(reportHeader))
		}
		decision := Action(rec[7])
		switch decision {
		case ActionChange, ActionKeep, ActionSkip:
		default:
			return Report{}, fmt.Errorf("line %d: decision %q is not change, keep or skipped", lineNo, rec[7])
		}
		var apply bool
		switch strings.ToLower(strings.TrimSpace(rec[9])) {
		case "yes":
			apply = true
		case "no":
		default:
			return Report{}, fmt.Errorf("line %d: apply %q is not yes or no", lineNo, rec[9])
		}
		rep.Rows = append(rep.Rows, Row{
			SplitID: rec[0], Date: rec[1], Name: unescapeCell(rec[2]), Amount: rec[3],
			OldCategory: unescapeCell(rec[4]), NewCategory: unescapeCell(rec[5]), Tier: rec[6],
			Decision: decision, Reason: unescapeCell(rec[8]), Apply: apply,
		})
	}
	return rep, nil
}
