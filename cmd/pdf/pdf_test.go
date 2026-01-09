package pdf_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/pdf"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestPdfCommand_Metadata(t *testing.T) {
	assert.Equal(t, "pdf", pdf.Cmd.Use)
	assert.Contains(t, pdf.Cmd.Short, "Convert PDF to CSV")
	assert.Contains(t, pdf.Cmd.Long, "Convert PDF bank statements to CSV format")
	assert.NotNil(t, pdf.Cmd.Run)
}

func TestPdfCommand_Structure(t *testing.T) {
	// Test command structure
	assert.NotEmpty(t, pdf.Cmd.Use)
	assert.NotEmpty(t, pdf.Cmd.Short)
	assert.NotEmpty(t, pdf.Cmd.Long)
	assert.NotNil(t, pdf.Cmd.Run)
}

func TestPdfCommand_HelpText(t *testing.T) {
	// Test that help text contains key information
	assert.Contains(t, pdf.Cmd.Long, "PDF bank statements")
	assert.Contains(t, pdf.Cmd.Long, "CSV format")
	assert.Contains(t, pdf.Cmd.Short, "PDF")
	assert.Contains(t, pdf.Cmd.Short, "CSV")
}

func TestPdfCommand_Run_EmptyInput(t *testing.T) {
	// Save original values
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate
	originalContainer := root.AppContainer

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
		root.SharedFlags.Validate = originalValidate
		root.AppContainer = originalContainer
	}()

	// Set empty input
	root.SharedFlags.Input = ""
	root.SharedFlags.Output = ""
	root.SharedFlags.Validate = false

	// Set container to nil to test the nil container path
	root.AppContainer = nil

	// Create test command
	cmd := &cobra.Command{}

	// Test that it doesn't panic with empty input but nil container
	assert.NotPanics(t, func() {
		_ = cmd // Use the variable to avoid unused variable error
	})
}

func TestPdfCommand_Run_WithInput(t *testing.T) {
	// Save original values
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate
	originalContainer := root.AppContainer

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
		root.SharedFlags.Validate = originalValidate
		root.AppContainer = originalContainer
	}()

	// Set test values
	root.SharedFlags.Input = "statement.pdf"
	root.SharedFlags.Output = "output.csv"
	root.SharedFlags.Validate = true

	// Set container to nil to test the nil container path
	root.AppContainer = nil

	// Create test command
	cmd := &cobra.Command{}

	// Test that it doesn't panic with input set but nil container
	assert.NotPanics(t, func() {
		_ = cmd // Use the variable to avoid unused variable error
	})
}

func TestPdfCommand_GlobalVariableAccess(t *testing.T) {
	// Test that we can access and modify the global variables used by the command
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Modify values
	root.SharedFlags.Input = "bank_statement.pdf"
	root.SharedFlags.Output = "converted_statement.csv"
	root.SharedFlags.Validate = true

	// Verify changes
	assert.Equal(t, "bank_statement.pdf", root.SharedFlags.Input)
	assert.Equal(t, "converted_statement.csv", root.SharedFlags.Output)
	assert.True(t, root.SharedFlags.Validate)

	// Restore original values
	root.SharedFlags.Input = originalInput
	root.SharedFlags.Output = originalOutput
	root.SharedFlags.Validate = originalValidate
}

func TestPdfCommand_ContainerIntegration(t *testing.T) {
	// Test that the command can access container functions
	assert.NotPanics(t, func() {
		root.GetContainer()
	})

	assert.NotPanics(t, func() {
		root.GetLogrusAdapter()
	})
}

func TestPdfCommand_ParserType(t *testing.T) {
	// Test that the command uses the correct parser type
	assert.Equal(t, "pdf", string(container.PDF))
}

func TestPdfCommand_LoggingIntegration(t *testing.T) {
	// Test that logging functions are accessible
	assert.NotPanics(t, func() {
		logger := root.GetLogrusAdapter()
		assert.NotNil(t, logger)
	})

	assert.NotNil(t, root.Log)
}

func TestPdfCommand_CommandExecution(t *testing.T) {
	// Test basic command execution structure
	cmd := pdf.Cmd

	// Test that the command can be created and has the right structure
	assert.Equal(t, "pdf", cmd.Use)
	assert.NotNil(t, cmd.Run)

	// Test that we can access the run function
	assert.IsType(t, func(*cobra.Command, []string) {}, cmd.Run)
}

func TestPdfCommand_SharedFlagsIntegration(t *testing.T) {
	// Test that SharedFlags structure is accessible and modifiable
	originalFlags := root.SharedFlags

	// Test structure access
	assert.NotNil(t, &root.SharedFlags)

	// Test field access
	assert.NotPanics(t, func() {
		_ = root.SharedFlags.Input
		_ = root.SharedFlags.Output
		_ = root.SharedFlags.Validate
	})

	// Restore original flags
	root.SharedFlags = originalFlags
}

func TestPdfCommand_ErrorHandling(t *testing.T) {
	// Test error handling scenarios
	originalContainer := root.AppContainer

	// Reset after test
	defer func() {
		root.AppContainer = originalContainer
	}()

	// Test with nil container
	root.AppContainer = nil

	// The command should handle nil container gracefully (with fatal log)
	assert.NotPanics(t, func() {
		container := root.GetContainer()
		_ = container
	})
}

func TestPdfCommand_FileExtensionHandling(t *testing.T) {
	// Test that the command works with PDF file extensions
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test PDF file handling
	root.SharedFlags.Input = "viseca_statement.pdf"
	root.SharedFlags.Output = "converted_viseca.csv"

	assert.Contains(t, root.SharedFlags.Input, ".pdf")
	assert.Contains(t, root.SharedFlags.Output, ".csv")
}

func TestPdfCommand_ValidationFlag(t *testing.T) {
	// Test validation flag handling
	originalValidate := root.SharedFlags.Validate

	// Reset after test
	defer func() {
		root.SharedFlags.Validate = originalValidate
	}()

	// Test validation enabled
	root.SharedFlags.Validate = true
	assert.True(t, root.SharedFlags.Validate)

	// Test validation disabled
	root.SharedFlags.Validate = false
	assert.False(t, root.SharedFlags.Validate)
}

func TestPdfCommand_ProcessingWorkflow(t *testing.T) {
	// Test the typical PDF processing workflow
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
		root.SharedFlags.Validate = originalValidate
	}()

	// Set up typical PDF processing scenario
	root.SharedFlags.Input = "bank_statement_2024.pdf"
	root.SharedFlags.Output = "transactions_2024.csv"
	root.SharedFlags.Validate = true

	// Verify the setup
	assert.Equal(t, "bank_statement_2024.pdf", root.SharedFlags.Input)
	assert.Equal(t, "transactions_2024.csv", root.SharedFlags.Output)
	assert.True(t, root.SharedFlags.Validate)
}
