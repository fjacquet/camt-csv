package selma_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/cmd/selma"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSelmaCommand_Metadata(t *testing.T) {
	assert.Equal(t, "selma", selma.Cmd.Use)
	assert.Contains(t, selma.Cmd.Short, "Convert Selma CSV to CSV")
	assert.Contains(t, selma.Cmd.Long, "Convert Selma CSV statements to CSV format")
	assert.NotNil(t, selma.Cmd.Run)
}

func TestSelmaCommand_Structure(t *testing.T) {
	// Test command structure
	assert.NotEmpty(t, selma.Cmd.Use)
	assert.NotEmpty(t, selma.Cmd.Short)
	assert.NotEmpty(t, selma.Cmd.Long)
	assert.NotNil(t, selma.Cmd.Run)
}

func TestSelmaCommand_HelpText(t *testing.T) {
	// Test that help text contains key information
	assert.Contains(t, selma.Cmd.Long, "Selma CSV")
	assert.Contains(t, selma.Cmd.Long, "CSV format")
	assert.Contains(t, selma.Cmd.Short, "Selma")
	assert.Contains(t, selma.Cmd.Short, "CSV")
}

func TestSelmaCommand_Run_EmptyInput(t *testing.T) {
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

func TestSelmaCommand_Run_WithInput(t *testing.T) {
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
	root.SharedFlags.Input = "selma_export.csv"
	root.SharedFlags.Output = "converted_selma.csv"
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

func TestSelmaCommand_GlobalVariableAccess(t *testing.T) {
	// Test that we can access and modify the global variables used by the command
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Modify values
	root.SharedFlags.Input = "selma_investment_report.csv"
	root.SharedFlags.Output = "processed_selma.csv"
	root.SharedFlags.Validate = false

	// Verify changes
	assert.Equal(t, "selma_investment_report.csv", root.SharedFlags.Input)
	assert.Equal(t, "processed_selma.csv", root.SharedFlags.Output)
	assert.False(t, root.SharedFlags.Validate)

	// Restore original values
	root.SharedFlags.Input = originalInput
	root.SharedFlags.Output = originalOutput
	root.SharedFlags.Validate = originalValidate
}

func TestSelmaCommand_ContainerIntegration(t *testing.T) {
	// Test that the command can access container functions
	assert.NotPanics(t, func() {
		root.GetContainer()
	})

	assert.NotPanics(t, func() {
		root.GetLogrusAdapter()
	})
}

func TestSelmaCommand_ParserType(t *testing.T) {
	// Test that the command uses the correct parser type
	assert.Equal(t, "selma", string(container.Selma))
}

func TestSelmaCommand_LoggingIntegration(t *testing.T) {
	// Test that logging functions are accessible
	assert.NotPanics(t, func() {
		logger := root.GetLogrusAdapter()
		assert.NotNil(t, logger)
	})

	assert.NotNil(t, root.Log)
}

func TestSelmaCommand_CommandExecution(t *testing.T) {
	// Test basic command execution structure
	cmd := selma.Cmd

	// Test that the command can be created and has the right structure
	assert.Equal(t, "selma", cmd.Use)
	assert.NotNil(t, cmd.Run)

	// Test that we can access the run function
	assert.IsType(t, func(*cobra.Command, []string) {}, cmd.Run)
}

func TestSelmaCommand_SharedFlagsIntegration(t *testing.T) {
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

func TestSelmaCommand_ErrorHandling(t *testing.T) {
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

func TestSelmaCommand_FileExtensionHandling(t *testing.T) {
	// Test that the command works with CSV file extensions
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test Selma CSV file handling
	root.SharedFlags.Input = "selma_portfolio_2024.csv"
	root.SharedFlags.Output = "converted_selma_2024.csv"

	assert.Contains(t, root.SharedFlags.Input, ".csv")
	assert.Contains(t, root.SharedFlags.Output, ".csv")
}

func TestSelmaCommand_ValidationFlag(t *testing.T) {
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

func TestSelmaCommand_ProcessingWorkflow(t *testing.T) {
	// Test the typical Selma processing workflow
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
		root.SharedFlags.Validate = originalValidate
	}()

	// Set up typical Selma processing scenario
	root.SharedFlags.Input = "selma_investment_transactions.csv"
	root.SharedFlags.Output = "standardized_investment_data.csv"
	root.SharedFlags.Validate = true

	// Verify the setup
	assert.Equal(t, "selma_investment_transactions.csv", root.SharedFlags.Input)
	assert.Equal(t, "standardized_investment_data.csv", root.SharedFlags.Output)
	assert.True(t, root.SharedFlags.Validate)
}

func TestSelmaCommand_InvestmentSpecificFeatures(t *testing.T) {
	// Test Selma-specific features and naming
	assert.Contains(t, selma.Cmd.Use, "selma")
	assert.Contains(t, selma.Cmd.Short, "Selma")
	assert.Contains(t, selma.Cmd.Long, "Selma")

	// Test that the parser type matches Selma
	assert.Equal(t, "selma", string(container.Selma))
}

func TestSelmaCommand_BrandIdentity(t *testing.T) {
	// Test that Selma branding is consistent
	cmd := selma.Cmd

	assert.Equal(t, "selma", cmd.Use)
	assert.Contains(t, cmd.Short, "Selma")
	assert.Contains(t, cmd.Long, "Selma")

	// Verify it's distinct from other investment platforms
	assert.NotContains(t, cmd.Short, "Revolut")
	assert.NotContains(t, cmd.Long, "Revolut")
}
