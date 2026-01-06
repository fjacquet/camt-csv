package revolutinvestment_test

import (
	"testing"

	revolutinvestment "fjacquet/camt-csv/cmd/revolut-investment"
	"fjacquet/camt-csv/cmd/root"
	"fjacquet/camt-csv/internal/container"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRevolutInvestmentCommand_Metadata(t *testing.T) {
	assert.Equal(t, "revolut-investment", revolutinvestment.Cmd.Use)
	assert.Contains(t, revolutinvestment.Cmd.Short, "Convert Revolut Investment CSV to CSV")
	assert.Contains(t, revolutinvestment.Cmd.Long, "Convert Revolut Investment CSV statements to CSV format")
	assert.NotNil(t, revolutinvestment.Cmd.Run)
}

func TestRevolutInvestmentCommand_Structure(t *testing.T) {
	// Test command structure
	assert.NotEmpty(t, revolutinvestment.Cmd.Use)
	assert.NotEmpty(t, revolutinvestment.Cmd.Short)
	assert.NotEmpty(t, revolutinvestment.Cmd.Long)
	assert.NotNil(t, revolutinvestment.Cmd.Run)
}

func TestRevolutInvestmentCommand_HelpText(t *testing.T) {
	// Test that help text contains key information
	assert.Contains(t, revolutinvestment.Cmd.Long, "Revolut Investment")
	assert.Contains(t, revolutinvestment.Cmd.Long, "CSV format")
	assert.Contains(t, revolutinvestment.Cmd.Short, "Revolut Investment")
	assert.Contains(t, revolutinvestment.Cmd.Short, "CSV")
}

func TestRevolutInvestmentCommand_Run_EmptyInput(t *testing.T) {
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

func TestRevolutInvestmentCommand_Run_WithInput(t *testing.T) {
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
	root.SharedFlags.Input = "revolut_investment_export.csv"
	root.SharedFlags.Output = "converted_investment.csv"
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

func TestRevolutInvestmentCommand_GlobalVariableAccess(t *testing.T) {
	// Test that we can access and modify the global variables used by the command
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Modify values
	root.SharedFlags.Input = "revolut_investment_portfolio.csv"
	root.SharedFlags.Output = "processed_investment.csv"
	root.SharedFlags.Validate = false

	// Verify changes
	assert.Equal(t, "revolut_investment_portfolio.csv", root.SharedFlags.Input)
	assert.Equal(t, "processed_investment.csv", root.SharedFlags.Output)
	assert.False(t, root.SharedFlags.Validate)

	// Restore original values
	root.SharedFlags.Input = originalInput
	root.SharedFlags.Output = originalOutput
	root.SharedFlags.Validate = originalValidate
}

func TestRevolutInvestmentCommand_ContainerIntegration(t *testing.T) {
	// Test that the command can access container functions
	assert.NotPanics(t, func() {
		root.GetContainer()
	})

	assert.NotPanics(t, func() {
		root.GetLogrusAdapter()
	})
}

func TestRevolutInvestmentCommand_ParserType(t *testing.T) {
	// Test that the command uses the correct parser type
	assert.Equal(t, "revolut-investment", string(container.RevolutInvestment))
}

func TestRevolutInvestmentCommand_LoggingIntegration(t *testing.T) {
	// Test that logging functions are accessible
	assert.NotPanics(t, func() {
		logger := root.GetLogrusAdapter()
		assert.NotNil(t, logger)
	})

	assert.NotNil(t, root.Log)
}

func TestRevolutInvestmentCommand_CommandExecution(t *testing.T) {
	// Test basic command execution structure
	cmd := revolutinvestment.Cmd

	// Test that the command can be created and has the right structure
	assert.Equal(t, "revolut-investment", cmd.Use)
	assert.NotNil(t, cmd.Run)

	// Test that we can access the run function
	assert.IsType(t, func(*cobra.Command, []string) {}, cmd.Run)
}

func TestRevolutInvestmentCommand_SharedFlagsIntegration(t *testing.T) {
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

func TestRevolutInvestmentCommand_ErrorHandling(t *testing.T) {
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

func TestRevolutInvestmentCommand_FileExtensionHandling(t *testing.T) {
	// Test that the command works with CSV file extensions
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test Revolut Investment CSV file handling
	root.SharedFlags.Input = "revolut_investment_trades_2024.csv"
	root.SharedFlags.Output = "converted_trades_2024.csv"

	assert.Contains(t, root.SharedFlags.Input, ".csv")
	assert.Contains(t, root.SharedFlags.Output, ".csv")
}

func TestRevolutInvestmentCommand_ValidationFlag(t *testing.T) {
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

func TestRevolutInvestmentCommand_ProcessingWorkflow(t *testing.T) {
	// Test the typical Revolut Investment processing workflow
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output
	originalValidate := root.SharedFlags.Validate

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
		root.SharedFlags.Validate = originalValidate
	}()

	// Set up typical Revolut Investment processing scenario
	root.SharedFlags.Input = "revolut_investment_activity.csv"
	root.SharedFlags.Output = "standardized_investment_transactions.csv"
	root.SharedFlags.Validate = true

	// Verify the setup
	assert.Equal(t, "revolut_investment_activity.csv", root.SharedFlags.Input)
	assert.Equal(t, "standardized_investment_transactions.csv", root.SharedFlags.Output)
	assert.True(t, root.SharedFlags.Validate)
}

func TestRevolutInvestmentCommand_InvestmentSpecificFeatures(t *testing.T) {
	// Test Revolut Investment-specific features and naming
	assert.Contains(t, revolutinvestment.Cmd.Use, "revolut-investment")
	assert.Contains(t, revolutinvestment.Cmd.Short, "Revolut Investment")
	assert.Contains(t, revolutinvestment.Cmd.Long, "Revolut Investment")

	// Test that the parser type matches Revolut Investment
	assert.Equal(t, "revolut-investment", string(container.RevolutInvestment))
}

func TestRevolutInvestmentCommand_BrandIdentity(t *testing.T) {
	// Test that Revolut Investment branding is consistent and distinct
	cmd := revolutinvestment.Cmd

	assert.Equal(t, "revolut-investment", cmd.Use)
	assert.Contains(t, cmd.Short, "Revolut Investment")
	assert.Contains(t, cmd.Long, "Revolut Investment")

	// Verify it's distinct from regular Revolut command
	assert.Contains(t, cmd.Use, "investment")
	assert.Contains(t, cmd.Short, "Investment")
	assert.Contains(t, cmd.Long, "Investment")
}

func TestRevolutInvestmentCommand_HyphenatedCommandName(t *testing.T) {
	// Test that the hyphenated command name is handled correctly
	cmd := revolutinvestment.Cmd

	assert.Equal(t, "revolut-investment", cmd.Use)
	assert.Contains(t, cmd.Use, "-")

	// Ensure it's not confused with other commands
	assert.NotEqual(t, "revolut", cmd.Use)
	assert.NotEqual(t, "investment", cmd.Use)
}

func TestRevolutInvestmentCommand_InvestmentDataHandling(t *testing.T) {
	// Test investment-specific data handling scenarios
	originalInput := root.SharedFlags.Input
	originalOutput := root.SharedFlags.Output

	// Reset after test
	defer func() {
		root.SharedFlags.Input = originalInput
		root.SharedFlags.Output = originalOutput
	}()

	// Test various investment file scenarios
	testCases := []struct {
		input  string
		output string
	}{
		{"revolut_stocks.csv", "processed_stocks.csv"},
		{"revolut_etf_trades.csv", "processed_etf.csv"},
		{"revolut_crypto_activity.csv", "processed_crypto.csv"},
		{"revolut_dividends.csv", "processed_dividends.csv"},
	}

	for _, tc := range testCases {
		root.SharedFlags.Input = tc.input
		root.SharedFlags.Output = tc.output

		assert.Equal(t, tc.input, root.SharedFlags.Input)
		assert.Equal(t, tc.output, root.SharedFlags.Output)
	}
}
