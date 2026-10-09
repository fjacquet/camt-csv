package icompta

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"fjacquet/camt-csv/internal/csvsafe"
)

const transferReportMagic = "# camt-csv link-transfers report v1"

const (
	confidenceSure     = "sure"
	confidenceDoubtful = "doubtful"
)

var transferReportHeader = []string{
	"debit_split_id", "credit_split_id", "debit_date", "credit_date", "gap_days",
	"debit_account", "credit_account", "debit_name", "credit_name", "amount",
	"debit_category", "credit_category", "confidence", "reason", "note", "apply",
}

// TransferRow is one reviewed candidate pair. Only rows with Apply are linked.
type TransferRow struct {
	DebitSplitID, CreditSplitID string
	DebitDate, CreditDate       string
	GapDays                     int
	DebitAccount, CreditAccount string
	DebitName, CreditName       string
	Amount                      string // absolute, 2 decimals
	DebitCategory               string
	CreditCategory              string
	Confidence                  string // "sure" or "doubtful"
	Reason, Note                string
	Apply                       bool
}

// TransferReport is what link-transfers preview writes and apply reads.
type TransferReport struct {
	DBState string
	Rows    []TransferRow
}

// NewTransferReport turns candidate pairs into report rows; sure pairs are
// pre-approved, doubtful ones are not.
func NewTransferReport(state string, pairs []Pair) TransferReport {
	rep := TransferReport{DBState: state, Rows: make([]TransferRow, 0, len(pairs))}
	for _, p := range pairs {
		conf := confidenceDoubtful
		if p.Sure {
			conf = confidenceSure
		}
		rep.Rows = append(rep.Rows, TransferRow{
			DebitSplitID: p.Debit.SplitID, CreditSplitID: p.Credit.SplitID,
			DebitDate: p.Debit.Date, CreditDate: p.Credit.Date, GapDays: p.GapDays,
			DebitAccount: p.Debit.AccountName, CreditAccount: p.Credit.AccountName,
			DebitName: p.Debit.Name, CreditName: p.Credit.Name,
			Amount:        p.Debit.Amount.Abs().StringFixed(2),
			DebitCategory: p.Debit.Category, CreditCategory: p.Credit.Category,
			Confidence: conf, Reason: p.Reason, Note: p.Note, Apply: p.Sure,
		})
	}
	return rep
}

// Write encodes the report: two comment lines, then a CSV with a header.
func (r TransferReport) Write(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s\n%s%s\n", transferReportMagic, stateKey, r.DBState); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(transferReportHeader); err != nil {
		return err
	}
	e := csvsafe.Escape
	for _, row := range r.Rows {
		apply := "no"
		if row.Apply {
			apply = "yes"
		}
		if err := cw.Write([]string{
			e(row.DebitSplitID), e(row.CreditSplitID), e(row.DebitDate), e(row.CreditDate), strconv.Itoa(row.GapDays),
			e(row.DebitAccount), e(row.CreditAccount), e(row.DebitName), e(row.CreditName), row.Amount,
			e(row.DebitCategory), e(row.CreditCategory), row.Confidence, e(row.Reason), e(row.Note), apply,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ReadTransferReport decodes a report, tolerating what a spreadsheet does to a
// CSV (BOM, CRLF, trailing commas on comment lines, any case of yes/no).
func ReadTransferReport(rd io.Reader) (TransferReport, error) {
	br := bufio.NewReader(rd)
	state, err := readReportPreamble(br, transferReportMagic, "link-transfers")
	if err != nil {
		return TransferReport{}, err
	}
	rep := TransferReport{DBState: state}

	cr := csv.NewReader(br)
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return TransferReport{}, fmt.Errorf("read report rows: %w", err)
	}
	if len(records) == 0 || strings.Join(records[0], ",") != strings.Join(transferReportHeader, ",") {
		return TransferReport{}, fmt.Errorf("report header is not %q", strings.Join(transferReportHeader, ","))
	}
	u := csvsafe.Unescape
	for i, rec := range records[1:] {
		lineNo := i + 4 // 2 comment lines, 1 header, 1-based
		if len(rec) != len(transferReportHeader) {
			return TransferReport{}, fmt.Errorf("line %d: %d fields, expected %d", lineNo, len(rec), len(transferReportHeader))
		}
		gap, err := strconv.Atoi(strings.TrimSpace(rec[4]))
		if err != nil {
			return TransferReport{}, fmt.Errorf("line %d: gap_days %q is not a number", lineNo, rec[4])
		}
		conf := strings.ToLower(strings.TrimSpace(rec[12]))
		if conf != confidenceSure && conf != confidenceDoubtful {
			return TransferReport{}, fmt.Errorf("line %d: confidence %q is not sure or doubtful", lineNo, rec[12])
		}
		var apply bool
		switch strings.ToLower(strings.TrimSpace(rec[15])) {
		case "yes":
			apply = true
		case "no":
		default:
			return TransferReport{}, fmt.Errorf("line %d: apply %q is not yes or no", lineNo, rec[15])
		}
		rep.Rows = append(rep.Rows, TransferRow{
			DebitSplitID: u(rec[0]), CreditSplitID: u(rec[1]), DebitDate: u(rec[2]), CreditDate: u(rec[3]),
			GapDays: gap, DebitAccount: u(rec[5]), CreditAccount: u(rec[6]), DebitName: u(rec[7]),
			CreditName: u(rec[8]), Amount: rec[9], DebitCategory: u(rec[10]), CreditCategory: u(rec[11]),
			Confidence: conf, Reason: u(rec[13]), Note: u(rec[14]), Apply: apply,
		})
	}
	return rep, nil
}
