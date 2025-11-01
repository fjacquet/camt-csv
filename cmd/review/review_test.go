package review

import (
	"bytes"
	"encoding/json"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/models"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T) (*cobra.Command, string, *bytes.Buffer, func()) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a new root command for testing
	testRootCmd := &cobra.Command{
		Use:   "camt-csv-test",
		Short: "A test CLI tool.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize configuration for the test command
			var err error
			testAppConfig, err := config.InitializeConfig()
			if err != nil {
				logrus.Fatalf("Failed to initialize test configuration: %v", err)
			}
			// Configure logging for test
			config.ConfigureLoggingFromConfig(testAppConfig)
		},
	}

	// Initialize the review command with the test root command
	testRootCmd.AddCommand(GetReviewCommand())

	// Backup original os.Args and restore after test
	oldArgs := os.Args
	os.Args = []string{"camt-csv-test"}

	// Clear constitutionFiles for each test
	constitutionFiles = []string{}

	// Re-initialize config and logger to ensure clean state for each test
	// This is crucial because config and logger are global singletons
	// For tests, we want to ensure they are isolated
	testAppConfig, _ := config.InitializeConfig() // Re-initialize with defaults
	config.ConfigureLoggingFromConfig(testAppConfig)

	// Capture stdout and stderr
	outBuf := new(bytes.Buffer)
	testRootCmd.SetOut(outBuf)
	testRootCmd.SetErr(outBuf) // Redirect stderr to the same buffer for simplicity in tests

	// Note: Output redirection removed for logging interface compatibility

	return testRootCmd, tempDir, outBuf, func() {
		os.Args = oldArgs
		// Restore global logger and config to a default clean state
		defaultAppConfig, _ := config.InitializeConfig()
		config.ConfigureLoggingFromConfig(defaultAppConfig)
		// Clear constitutionFiles for next test
		constitutionFiles = []string{}

		// Note: Output restoration removed for logging interface compatibility
	}
}

func TestReviewCommand_BasicExecution(t *testing.T) {
	testRootCmd, tempDir, outBuf, teardown := setupTest(t)
	defer teardown()

	// Create a dummy constitution file
	constitutionPath := filepath.Join(tempDir, "constitution.yaml")
	constitutionContent := []byte(`
principles:
  - id: TEST-001
    name: "Test Principle"
    description: "A test principle."
    category: "Test"
    evaluation_method: "Manual"
`)
	err := os.WriteFile(constitutionPath, constitutionContent, 0600)
	assert.NoError(t, err)

	// Create a dummy Go file to review
	filePath := filepath.Join(tempDir, "test.go")
	fileContent := []byte(`package main\nfunc main() {}\n`)
	err = os.WriteFile(filePath, fileContent, 0600)
	assert.NoError(t, err)

	// Set command arguments
	testRootCmd.SetArgs([]string{
		"review",
		filePath,
		"--constitution-files", constitutionPath,
		"--output-format", "json",
	})

	// Execute the command
	err = testRootCmd.Execute()
	assert.NoError(t, err)

	// Verify output contains a JSON report
	output := outBuf.String()

	var report models.ComplianceReport
	err = json.Unmarshal([]byte(output), &report)
	assert.NoError(t, err)

	assert.NotEmpty(t, report.ReportID)
	assert.Len(t, report.CodebaseSection, 1)
	assert.Equal(t, filePath, report.CodebaseSection[0].Path)
	assert.Len(t, report.PrinciplesReviewed, 1)
	assert.Equal(t, "TEST-001", report.PrinciplesReviewed[0].ID)
	assert.Equal(t, models.OverallStatusNonCompliant, report.OverallStatus)
	assert.Len(t, report.Findings, 1)
	assert.Equal(t, models.FindingStatusManualReviewRequired, report.Findings[0].Status)
	assert.Contains(t, report.Findings[0].Details, "Placeholder finding for TEST-001")
}

func TestReviewCommand_MissingPath(t *testing.T) {
	testRootCmd, tempDir, _, teardown := setupTest(t)
	defer teardown()

	// Create a dummy constitution file
	constitutionPath := filepath.Join(tempDir, "constitution.yaml")
	constitutionContent := []byte(`
principles:
  - id: TEST-001
    name: "Test Principle"
    description: "A test principle."
    category: "Test"
    evaluation_method: "Manual"
`)
	err := os.WriteFile(constitutionPath, constitutionContent, 0600)
	assert.NoError(t, err)

	// Capture error output
	errBuf := new(bytes.Buffer)
	testRootCmd.SetErr(errBuf)

	// Set command arguments without a path
	testRootCmd.SetArgs([]string{
		"review",
		"--constitution-files", constitutionPath,
		"--output-format", "json",
	})

	// Execute the command
	err = testRootCmd.Execute()
	assert.Error(t, err)

	// Verify error message
	output := errBuf.String()
	assert.Contains(t, output, "Error: requires at least 1 arg(s), only received 0")
}

func TestReviewCommand_InvalidConstitutionFile(t *testing.T) {
	testRootCmd, tempDir, _, teardown := setupTest(t)
	defer teardown()

	// Create an invalid constitution file
	constitutionPath := filepath.Join(tempDir, "invalid_constitution.yaml")
	constitutionContent := []byte(`
invalid yaml content
`)
	err := os.WriteFile(constitutionPath, constitutionContent, 0600)
	assert.NoError(t, err)

	// Create a dummy Go file to review
	filePath := filepath.Join(tempDir, "test.go")
	fileContent := []byte(`package main\nfunc main() {}\n`)
	err = os.WriteFile(filePath, fileContent, 0600)
	assert.NoError(t, err)

	// Capture error output
	errBuf := new(bytes.Buffer)
	testRootCmd.SetErr(errBuf)

	// Set command arguments
	testRootCmd.SetArgs([]string{
		"review",
		filePath,
		"--constitution-files", constitutionPath,
		"--output-format", "json",
	})

	// Execute the command
	err = testRootCmd.Execute()
	assert.Error(t, err)

	// Verify error message
	output := errBuf.String()
	assert.Contains(t, output, "failed to load constitution file")
}

func TestReviewCommand_OutputToFile(t *testing.T) {
	testRootCmd, tempDir, _, teardown := setupTest(t)
	defer teardown()

	// Create a dummy constitution file
	constitutionPath := filepath.Join(tempDir, "constitution.yaml")
	constitutionContent := []byte(`
principles:
  - id: TEST-001
    name: "Test Principle"
    description: "A test principle."
    category: "Test"
    evaluation_method: "Manual"
`)
	err := os.WriteFile(constitutionPath, constitutionContent, 0600)
	assert.NoError(t, err)

	// Create a dummy Go file to review
	filePath := filepath.Join(tempDir, "test.go")
	fileContent := []byte(`package main\nfunc main() {}\n`)
	err = os.WriteFile(filePath, fileContent, 0600)
	assert.NoError(t, err)

	outputPath := filepath.Join(tempDir, "report.json")

	// Set command arguments
	testRootCmd.SetArgs([]string{
		"review",
		filePath,
		"--constitution-files", constitutionPath,
		"--output-format", "json",
		"--output-file", outputPath,
	})

	// Execute the command
	err = testRootCmd.Execute()
	assert.NoError(t, err)

	// Verify report file was created and contains expected content
	reportBytes, err := os.ReadFile(outputPath)
	assert.NoError(t, err)

	var report models.ComplianceReport
	err = json.Unmarshal(reportBytes, &report)
	assert.NoError(t, err)

	assert.NotEmpty(t, report.ReportID)
	assert.Len(t, report.CodebaseSection, 1)
	assert.Equal(t, filePath, report.CodebaseSection[0].Path)
	assert.Len(t, report.PrinciplesReviewed, 1)
	assert.Equal(t, "TEST-001", report.PrinciplesReviewed[0].ID)
	assert.Equal(t, models.OverallStatusNonCompliant, report.OverallStatus)
	assert.Len(t, report.Findings, 1)
	assert.Equal(t, models.FindingStatusManualReviewRequired, report.Findings[0].Status)
	assert.Contains(t, report.Findings[0].Details, "Placeholder finding for TEST-001")
}
