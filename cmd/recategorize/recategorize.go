// Package recategorize reclassifies transactions already stored in an iCompta
// database, with a reviewable preview before anything is written.
package recategorize

import (
	"fmt"
	"os"
	"time"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/icompta"

	"github.com/spf13/cobra"
)

// Each subcommand owns its flag variables: sharing one --db variable between
// preview and apply would let one command's value leak into the other.
var (
	previewDB  string
	outputPath string
	applyDB    string
	reportPath string
)

// Cmd is the recategorize command.
var Cmd = &cobra.Command{
	Use:     "recategorize",
	Short:   "Recategorize transactions stored in an iCompta database",
	GroupID: "tools",
	Long: `Run the categorizer over an iCompta database in two steps.

  preview  reads the database (read-only) and writes a CSV report of proposed changes.
  apply    writes the rows you approved in that report into the database.

A category you chose is replaced only by a direct mapping or keyword match. The
semantic and AI tiers only fill empty or "unknown" categories (Divers, Non Classé,
Autre, Uncategorized). Investment splits and linked transfers are left alone.

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
	ctx := cmd.Context()
	log := root.GetLogrusAdapter()

	appContainer := root.GetContainer()
	if appContainer == nil {
		return fmt.Errorf("container not initialized")
	}

	store, err := icompta.OpenReadOnly(ctx, previewDB)
	if err != nil {
		return err
	}
	defer store.Close()
	snap, err := store.Snapshot(ctx)
	if err != nil {
		return err
	}

	rep := icompta.Preview(ctx, snap, icompta.NewCategorizerClassifier(appContainer.GetCategorizer()), log)

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	if err := rep.Write(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("write report: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	log.Infof("Report written to %s: review it, set apply to no on rows you reject, then run apply", outputPath)
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
