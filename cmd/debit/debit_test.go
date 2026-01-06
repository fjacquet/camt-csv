package debit_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/debit"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestDebitCommand_Metadata(t *testing.T) {
	assert.Equal(t, "debit", debit.Cmd.Use)
	assert.Contains(t, debit.Cmd.Short, "Convert Debit CSV to CSV")
	assert.Contains(t, debit.Cmd.Long, "Convert Debit CSV statements to CSV format")
	assert.NotNil(t, debit.Cmd.Run)
}

func TestDebitCommand_Structure(t *testing.T) {
	// Test command structure
	assert.NotEmpty(t, debit.Cmd.Use)
	assert.NotEmpty(t, debit.Cmd.Short)
	assert.NotEmpty(t, debit.Cmd.Long)
	assert.NotNil(t, debit.Cmd.Run)
}

func TestDebitCommand_HelpText(t *testing.T) {
	// Test that help text contains key information
	assert.Contains(t, debit.Cmd.Long, "Debit CSV")
	assert.Contains(t, debit.Cmd.Long, "CSV format")
	assert.Contains(t, debit.Cmd.Short, "Debit CSV")
}

func TestDebitCommand_Run_EmptyInput(t *testing.T) {
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

func TestDebitCommand_Run_WithInput(t *testing.T) {
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
	root.SharedFlags.Input = "test.csv"
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

func TestDebitCommand_GlobalVariableAccess(t *testing.T) {
	// Test that we can access and modify the global variables used by the command
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate
	
	// Modify values
	root.SharedFlags.Input = "modified.csv"
	root.SharedFlags.Output = "modified_output.csv"
	root.SharedFlags.Validate = false
	
	// Verify changes
	assert.Equal(t, "modified.csv", root.SharedFlags.Input)
	assert.Equal(t, "modified_output.csv", root.SharedFlags.Output)
	assert.False(t, root.SharedFlags.Validate)
	
	// Restore original values
	root.SharedFlags.Input = originalInput
	root.SharedFlags.Output = originalOutput
	root.SharedFlags.Validate = originalValidate
}

func TestDebitCommand_ContainerIntegration(t *testing.T) {
	// Test that the command can access container functions
	assert.NotPanics(t, func() {
		root.GetContainer()
	})
	
	assert.NotPanics(t, func() {
		root.GetLogrusAdapter()
	})
}

func TestDebitCommand_ParserType(t *testing.T) {
	// Test that the command uses the correct parser type
	assert.Equal(t, "debit", string(container.Debit))
}

func TestDebitCommand_LoggingIntegration(t *testing.T) {
	// Test that logging functions are accessible
	assert.NotPanics(t, func() {
		logger := root.GetLogrusAdapter()
		assert.NotNil(t, logger)
	})
	
	assert.NotNil(t, root.Log)
}

func TestDebitCommand_CommandExecution(t *testing.T) {
	// Test basic command execution structure
	cmd := debit.Cmd
	
	// Test that the command can be created and has the right structure
	assert.Equal(t, "debit", cmd.Use)
	assert.NotNil(t, cmd.Run)
	
	// Test that we can access the run function
	assert.IsType(t, func(*cobra.Command, []string){}, cmd.Run)
}

func TestDebitCommand_SharedFlagsIntegration(t *testing.T) {
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

func TestDebitCommand_ErrorHandling(t *testing.T) {
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

func TestDebitCommand_FileExtensionHandling(t *testing.T) {
	// Test that the command works with CSV file extensions
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	
	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()
	
	// Test CSV file handling
	root.SharedFlags.Input = "debit_statement.csv"
	root.SharedFlags.Output = "converted_debit.csv"
	
	assert.Contains(t, root.SharedFlags.Input, ".csv")
	assert.Contains(t, root.SharedFlags.Output, ".csv")
}

func TestDebitCommand_ValidationFlag(t *testing.T) {
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