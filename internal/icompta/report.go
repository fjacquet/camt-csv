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
			row.SplitID, row.Date, row.Name, row.Amount, row.OldCategory, row.NewCategory,
			row.Tier, string(row.Decision), row.Reason, apply,
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
			SplitID: rec[0], Date: rec[1], Name: rec[2], Amount: rec[3],
			OldCategory: rec[4], NewCategory: rec[5], Tier: rec[6],
			Decision: decision, Reason: rec[8], Apply: apply,
		})
	}
	return rep, nil
}
