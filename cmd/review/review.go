package review

import (
	"fmt"
	"os"
	"strings"

	"fjacquet/camt-csv/internal/git"
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/report"
	"fjacquet/camt-csv/internal/reviewer"
	"fjacquet/camt-csv/internal/scanner"

	"github.com/spf13/cobra"
)

var (
	// constitutionFiles holds the paths to one or more constitution definition files.
	constitutionFiles []string
	// principles holds a comma-separated list of specific constitution principle IDs to apply.
	principles []string
	// outputFormat holds the desired output format for the compliance report (json, xml).
	outputFormat string
	// outputFile holds the absolute path to a file where the compliance report should be written.
	outputFile string
	// gitRef holds a Git reference to compare the current codebase against for a diff-based review.
	gitRef string
)

// reviewCmd represents the review command
var reviewCmd = &cobra.Command{
	Use:   "review [path...]",
	Short: "Review the codebase against the project constitution",
	Long: `This command systematically reviews the codebase against the project constitution,
identifying areas of non-compliance and proposing corrective actions.`, // Note: The backticks here correctly handle multi-line strings in Go.
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.NewLogrusAdapter("info", "text").WithField("command", "review")

		// 1. Instantiate services
		codebaseScanner := scanner.NewCodebaseScanner(logger)
		constitutionLoader := parser.NewConstitutionLoader(logger)
		automatedEvaluator := reviewer.NewAutomatedPrincipleEvaluator(logger)
		reviewerService := reviewer.NewReviewer(codebaseScanner, constitutionLoader, automatedEvaluator, logger)
		reportGenerator := report.NewReportGenerator(logger)

		// Validate output format
		if outputFormat != "json" && outputFormat != "xml" {
			return fmt.Errorf("unsupported output format: %s. Must be 'json' or 'xml'.", outputFormat)
		}

		// Handle --git-ref flag
		pathsToScan := args
		if gitRef != "" {
			if !git.IsGitRepo() {
				return fmt.Errorf("current directory is not a Git repository, cannot use --git-ref")
			}

			changedFiles, err := git.GetChangedFiles(gitRef)
			if err != nil {
				return fmt.Errorf("failed to get changed files: %w", err)
			}

			if len(changedFiles) == 0 {
				logger.Info("No files changed since reference",
					logging.Field{Key: "git_ref", Value: gitRef})
				fmt.Println("No files changed since", gitRef)
				return nil
			}

			// Filter pathsToScan to only include changed files
			pathsToScan = filterPathsToChangedFiles(args, changedFiles)

			logger.WithFields(
				logging.Field{Key: "git_ref", Value: gitRef},
				logging.Field{Key: "changed_files", Value: len(changedFiles)},
				logging.Field{Key: "paths_to_scan", Value: len(pathsToScan)},
			).Info("Filtered to changed files only")
		}

		// 2. Perform review
		report, err := reviewerService.PerformReview(pathsToScan, constitutionFiles, principles, outputFormat)
		if err != nil {
			return fmt.Errorf("failed to perform review: %w", err)
		}

		// 3. Generate report
		reportBytes, err := reportGenerator.GenerateReport(report, outputFormat)
		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		// 4. Output report
		if outputFile != "" {
			err = os.WriteFile(outputFile, reportBytes, 0600)
			if err != nil {
				return fmt.Errorf("failed to write report to file %s: %w", outputFile, err)
			}
			logger.WithField("file", outputFile).Info("Compliance report written to file")
		} else {
			_, err = cmd.OutOrStdout().Write(reportBytes)
			if err != nil {
				return fmt.Errorf("failed to write report to stdout: %w", err)
			}
		}

		return nil
	},
}

func init() {
	reviewCmd.Flags().StringArrayVar(&constitutionFiles, "constitution-files", []string{}, "Paths to one or more constitution definition files (e.g., YAML, TOML).")
	reviewCmd.Flags().StringArrayVar(&principles, "principles", []string{}, "A comma-separated list of specific constitution principle IDs to apply during the review.")
	reviewCmd.Flags().StringVar(&outputFormat, "output-format", "json", "The desired output format for the compliance report. Supported values: json, xml. (default \"json\")")
	reviewCmd.Flags().StringVar(&outputFile, "output-file", "", "The absolute path to a file where the compliance report should be written.")
	reviewCmd.Flags().StringVar(&gitRef, "git-ref", "", "A Git reference (e.g., commit hash, branch name) to compare the current codebase against for a diff-based review.")
}

// GetReviewCommand returns the Cobra command for the review functionality.
func GetReviewCommand() *cobra.Command {
	return reviewCmd
}

// filterPathsToChangedFiles returns only paths from original that match or contain changed files.
// If original paths are directories, it includes changed files within those directories.
// If original is empty (meaning scan current directory), it returns all changed files.
func filterPathsToChangedFiles(original, changedFiles []string) []string {
	// If no paths specified, use all changed files
	if len(original) == 0 {
		return changedFiles
	}

	changedSet := make(map[string]bool)
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	var filtered []string
	seen := make(map[string]bool)

	for _, path := range original {
		// Check if path is directly in changed files
		if changedSet[path] {
			if !seen[path] {
				filtered = append(filtered, path)
				seen[path] = true
			}
			continue
		}

		// Check if path is a directory containing changed files
		for _, changed := range changedFiles {
			if strings.HasPrefix(changed, path+"/") || path == "." {
				if !seen[changed] {
					filtered = append(filtered, changed)
					seen[changed] = true
				}
			}
		}
	}

	return filtered
}
