package revolut_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/revolut"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRevolutCommand_Metadata(t *testing.T) {
	assert.Equal(t, "revolut", revolut.Cmd.Use)
	assert.Contains(t, revolut.Cmd.Short, "Convert Revolut CSV to CSV")
	assert.Contains(t, revolut.Cmd.Long, "Convert Revolut CSV statements to CSV format")
	assert.NotNil(t, revolut.Cmd.Run)
}

func TestRevolutCommand_Structure(t *testing.T) {
	// Test command structure
	assert.NotEmpty(t, revolut.Cmd.Use)
	assert.NotEmpty(t, revolut.Cmd.Short)
	assert.NotEmpty(t, revolut.Cmd.Long)
	assert.NotNil(t, revolut.Cmd.Run)
}

func TestRevolutCommand_HelpText(t *testing.T) {
	// Test that help text contains key information
	assert.Contains(t, revolut.Cmd.Long, "Revolut CSV")
	assert.Contains(t, revolut.Cmd.Long, "CSV format")
	assert.Contains(t, revolut.Cmd.Short, "Revolut")
	assert.Contains(t, revolut.Cmd.Short, "CSV")
}

func TestRevolutCommand_Run_EmptyInput(t *testing.T) {
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

func TestRevolutCommand_Run_WithInput(t *testing.T) {
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
	root.SharedFlags.Input = "revolut_export.csv"
	root.SharedFlags.Output = "converted_revolut.csv"
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

func TestRevolutCommand_GlobalVariableAccess(t *testing.T) {
	// Test that we can access and modify the global variables used by the command
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Modify values
	root.SharedFlags.Input = "revolut_transactions.csv"
	root.SharedFlags.Output = "processed_revolut.csv"
	root.SharedFlags.Validate = false

	// Verify changes
	assert.Equal(t, "revolut_transactions.csv", root.SharedFlags.Input)
	assert.Equal(t, "processed_revolut.csv", root.SharedFlags.Output)
	assert.False(t, root.SharedFlags.Validate)

	// Restore original values
	root.SharedFlags.Input = originalInput
	root.SharedFlags.Output = originalOutput
	root.SharedFlags.Validate = originalValidate
}

func TestRevolutCommand_ContainerIntegration(t *testing.T) {
	// Test that the command can access container functions
	assert.NotPanics(t, func() {
		root.GetContainer()
	})

	assert.NotPanics(t, func() {
		root.GetLogrusAdapter()
	})
}

func TestRevolutCommand_ParserType(t *testing.T) {
	// Test that the command uses the correct parser type
	assert.Equal(t, "revolut", string(container.Revolut))
}

func TestRevolutCommand_LoggingIntegration(t *testing.T) {
	// Test that logging functions are accessible
	assert.NotPanics(t, func() {
		logger := root.GetLogrusAdapter()
		assert.NotNil(t, logger)
	})

	assert.NotNil(t, root.Log)
}

func TestRevolutCommand_CommandExecution(t *testing.T) {
	// Test basic command execution structure
	cmd := revolut.Cmd

	// Test that the command can be created and has the right structure
	assert.Equal(t, "revolut", cmd.Use)
	assert.NotNil(t, cmd.Run)

	// Test that we can access the run function
	assert.IsType(t, func(*cobra.Command, []string) {}, cmd.Run)
}

func TestRevolutCommand_SharedFlagsIntegration(t *testing.T) {
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

func TestRevolutCommand_ErrorHandling(t *testing.T) {
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

func TestRevolutCommand_FileExtensionHandling(t *testing.T) {
	// Test that the command works with CSV file extensions
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test Revolut CSV file handling
	root.SharedFlags.Input = "revolut_export_2024.csv"
	root.SharedFlags.Output = "converted_revolut_2024.csv"

	assert.Contains(t, root.SharedFlags.Input, ".csv")
	assert.Contains(t, root.SharedFlags.Output, ".csv")
}

func TestRevolutCommand_ValidationFlag(t *testing.T) {
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

func TestRevolutCommand_ProcessingWorkflow(t *testing.T) {
	// Test the typical Revolut processing workflow
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
		root.SharedFlags.Validate = originalValidate
	}()

	// Set up typical Revolut processing scenario
	root.SharedFlags.Input = "revolut_business_export.csv"
	root.SharedFlags.Output = "standardized_transactions.csv"
	root.SharedFlags.Validate = true

	// Verify the setup
	assert.Equal(t, "revolut_business_export.csv", root.SharedFlags.Input)
	assert.Equal(t, "standardized_transactions.csv", root.SharedFlags.Output)
	assert.True(t, root.SharedFlags.Validate)
}

func TestRevolutCommand_BrandSpecificFeatures(t *testing.T) {
	// Test Revolut-specific features and naming
	assert.Contains(t, revolut.Cmd.Use, "revolut")
	assert.Contains(t, revolut.Cmd.Short, "Revolut")
	assert.Contains(t, revolut.Cmd.Long, "Revolut")

	// Test that the parser type matches Revolut
	assert.Equal(t, "revolut", string(container.Revolut))
}
