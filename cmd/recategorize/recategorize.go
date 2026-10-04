// Package recategorize reclassifies transactions already stored in an iCompta
// database, with a reviewable preview before anything is written.
package recategorize

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

// Each subcommand owns its flag variables: sharing one --db variable between
// preview and apply would let one command's value leak into the other.
var (
	previewDB   string
	outputPath  string
	forceOutput bool
	applyDB     string
	reportPath  string
)

// Cmd is the recategorize command.
var Cmd = &cobra.Command{
	Use:     "recategorize",
	Short:   "Recategorize transactions stored in an iCompta database",
	GroupID: "tools",
	Long: `Run the categorizer over an iCompta database in two steps.

  preview  reads the database (read-only) and writes a CSV report of proposed changes.
  apply    writes the rows you approved in that report into the database.

A category you chose is replaced only by a direct mapping (an exact match on the
party name). Keyword, semantic and AI results only fill empty or "unknown"
categories (Divers, Non Classé, Autre, Uncategorized). Investment splits and
linked transfers are left alone. The report is sorted by category move, so
identical changes sit together.

apply refuses to run while iCompta is open, backs up the database first, and
logs every change. If you use iCloud sync, check another device after the first
apply: whether a change written outside iCompta propagates is not guaranteed.`,
}

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Write a report of proposed category changes (read-only)",
	Args:  cobra.NoArgs,
	RunE:  runPreview,
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the approved rows of a preview report to the database",
	Args:  cobra.NoArgs,
	RunE:  runApply,
}

func init() {
	previewCmd.Flags().StringVar(&previewDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	previewCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Path of the report CSV to write")
	previewCmd.Flags().BoolVar(&forceOutput, "force", false, "Overwrite the report if it already exists")
	_ = previewCmd.MarkFlagRequired("db")
	_ = previewCmd.MarkFlagRequired("output")

	applyCmd.Flags().StringVar(&applyDB, "db", "", "Path to the iCompta database (ic25.cdb)")
	applyCmd.Flags().StringVar(&reportPath, "report", "", "Path of the reviewed report CSV")
	_ = applyCmd.MarkFlagRequired("db")
	_ = applyCmd.MarkFlagRequired("report")

	Cmd.AddCommand(previewCmd, applyCmd)
}

func runPreview(cmd *cobra.Command, _ []string) error {
	// MarkFlagRequired is enforced by Cobra's execute(), not when RunE is called
	// directly (as the tests do), so the command checks again.
	if previewDB == "" || outputPath == "" {
		return fmt.Errorf("--db and --output are required")
	}
	appContainer := root.GetContainer()
	if appContainer == nil {
		return fmt.Errorf("container not initialized")
	}
	cl := icompta.NewCategorizerClassifier(appContainer.GetCategorizer())
	return runPreviewWith(cmd.Context(), previewDB, outputPath, forceOutput, cl, root.GetLogrusAdapter())
}

// runPreviewWith opens the report first, so a bad path fails before any slow
// classification and a reviewed report is never overwritten by accident. If ctx
// is cancelled mid-run the rows decided so far are still written.
func runPreviewWith(ctx context.Context, db, out string, force bool, cl icompta.Classifier, log logging.Logger) (err error) {
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
	// A run that fails before the report is written must not leave an empty
	// file that the next run would refuse to overwrite.
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
	snap, err := store.Snapshot(ctx)
	if err != nil {
		return err
	}

	rep := icompta.Preview(ctx, snap, cl, log)
	if err := rep.Write(f); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	written = true
	if err := f.Close(); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	log.Info("Report written: review it, set apply to no on rows you reject, then run apply",
		logging.Field{Key: "path", Value: out})
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
	rep, err := icompta.ReadReport(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("report %s: %w", reportPath, err)
	}

	_, err = icompta.Apply(cmd.Context(), rep, icompta.ApplyOptions{
		DBPath:    applyDB,
		Now:       time.Now,
		IsRunning: icompta.ICComptaRunning,
	}, root.GetLogrusAdapter())
	return err
}
