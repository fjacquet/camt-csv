// Package linktransfers links both sides of transfers between the user's own
// accounts in an iCompta database, with a reviewable preview before anything
// is written.
package linktransfers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/icompta"
	"fjacquet/camt-csv/internal/logging"

	"github.com/spf13/cobra"
)

// Each subcommand owns its flag variables, as in recategorize.
var (
	previewDB   string
	outputPath  string
	forceOutput bool
	folders     []string
	fromDate    string
	toDate      string
	applyDB     string
	reportPath  string
)

// Cmd is the link-transfers command.
var Cmd = &cobra.Command{
	Use:     "link-transfers",
	Short:   "Link both sides of transfers between your own iCompta accounts",
	GroupID: "tools",
	Long: `Find transfers between your own accounts that were imported as two unrelated
transactions, and link them the way iCompta does, in two steps.

  preview  reads the database (read-only) and writes a CSV report of candidate pairs.
  apply    links the pairs you approved in that report.

Only accounts under the --folder folders are considered (any depth). A pair is
the same amount in opposite directions, in two accounts of the same currency,
at most 4 days apart. Pairs at most 2 days apart with no competing candidate, and,
when the two sides have different categories, one of them a transfer or
uncategorized, are pre-approved (apply=yes). The others are listed with apply=no
and a reason.

apply refuses to run while iCompta is open, backs up the database first, and
changes only the link between the two splits: categories and amounts stay.`,
}

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Write a report of candidate transfer pairs (read-only)",
	Args:  cobra.NoArgs,
	RunE:  runPreview,
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Link the approved pairs of a preview report",
	Args:  cobra.NoArgs,
	RunE:  runApply,
}

func init() {
	previewCmd.Flags().StringVar(&previewDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	previewCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Path of the report CSV to write")
	previewCmd.Flags().BoolVar(&forceOutput, "force", false, "Overwrite the report if it already exists")
	previewCmd.Flags().StringArrayVar(&folders, "folder", nil, "iCompta folder holding your accounts (repeatable)")
	previewCmd.Flags().StringVar(&fromDate, "from", "", "Only pairs with at least one side on or after this date (YYYY-MM-DD)")
	previewCmd.Flags().StringVar(&toDate, "to", "", "Only pairs with at least one side on or before this date (YYYY-MM-DD)")
	_ = previewCmd.MarkFlagRequired("db")
	_ = previewCmd.MarkFlagRequired("output")
	_ = previewCmd.MarkFlagRequired("folder")

	applyCmd.Flags().StringVar(&applyDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	applyCmd.Flags().StringVar(&reportPath, "report", "", "Path of the reviewed report CSV")
	_ = applyCmd.MarkFlagRequired("db")
	_ = applyCmd.MarkFlagRequired("report")

	Cmd.AddCommand(previewCmd, applyCmd)
}

func runPreview(cmd *cobra.Command, _ []string) error {
	// MarkFlagRequired is enforced by Cobra's execute(), not when RunE is called
	// directly, so the command checks again.
	if previewDB == "" || outputPath == "" || len(folders) == 0 {
		return fmt.Errorf("--db, --output and --folder are required")
	}
	return runPreviewWith(cmd.Context(), previewDB, outputPath, forceOutput, folders, fromDate, toDate, root.GetLogrusAdapter())
}

func checkDate(flag, v string) error {
	if v == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return fmt.Errorf("%s %q is not a YYYY-MM-DD date", flag, v)
	}
	return nil
}

// runPreviewWith validates the dates, then opens the report before reading the
// database, so a bad path fails at once and a reviewed report is never
// overwritten by accident. A failed run leaves no report file.
func runPreviewWith(ctx context.Context, db, out string, force bool, folderNames []string, from, to string, log logging.Logger) (err error) {
	if err := checkDate("--from", from); err != nil {
		return err
	}
	if err := checkDate("--to", to); err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(out, flags, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("%s already exists: use --force to overwrite it", out)
		}
		return fmt.Errorf("create report: %w", err)
	}
	written := false
	defer func() {
		_ = f.Close()
		if !written {
			_ = os.Remove(out)
		}
	}()

	store, err := icompta.OpenReadOnly(ctx, db)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	snap, err := store.TransferSnapshot(ctx)
	if err != nil {
		return err
	}
	scope, err := icompta.ResolveScope(snap.Accounts, folderNames)
	if err != nil {
		return err
	}
	pairs := icompta.PairsInRange(icompta.MatchTransfers(icompta.EligibleLegs(snap.Legs, scope)), from, to)
	rep := icompta.NewTransferReport(snap.State, pairs)
	if err := rep.Write(f); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	written = true

	sure := 0
	for _, p := range pairs {
		if p.Sure {
			sure++
		}
	}
	log.Info("Report written: review it, set apply to yes or no, then run apply",
		logging.Field{Key: "path", Value: out},
		logging.Field{Key: "accounts", Value: len(scope)},
		logging.Field{Key: "sure", Value: sure},
		logging.Field{Key: "doubtful", Value: len(pairs) - sure})
	return nil
}

func runApply(cmd *cobra.Command, _ []string) error {
	if applyDB == "" || reportPath == "" {
		return fmt.Errorf("--db and --report are required")
	}
	f, err := os.Open(reportPath)
	if err != nil {
		return fmt.Errorf("open report: %w", err)
	}
	rep, err := icompta.ReadTransferReport(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("report %s: %w", reportPath, err)
	}
	res, err := icompta.ApplyLinks(cmd.Context(), rep, icompta.ApplyOptions{
		DBPath:    applyDB,
		Now:       time.Now,
		IsRunning: icompta.ICComptaRunning,
	}, root.GetLogrusAdapter())
	if err != nil && res.BackupPath != "" {
		return fmt.Errorf("%w (backup: %s)", err, res.BackupPath)
	}
	return err
}
