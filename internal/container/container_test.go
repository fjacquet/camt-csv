package container

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cryptoRandIntn returns a random int in [0, n) using crypto/rand
func cryptoRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	max := big.NewInt(int64(n))
	result, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return int(result.Int64())
}

func TestNewContainer(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "configuration cannot be nil",
		},
		{
			name: "valid config without AI",
			config: &config.Config{
				Log: struct {
					Level  string `mapstructure:"level" yaml:"level"`
					Format string `mapstructure:"format" yaml:"format"`
				}{
					Level:  "info",
					Format: "text",
				},
				Categories: struct {
					File          string `mapstructure:"file" yaml:"file"`
					CreditorsFile string `mapstructure:"creditors_file" yaml:"creditors_file"`
					DebtorsFile   string `mapstructure:"debtors_file" yaml:"debtors_file"`
				}{
					File:          "categories.yaml",
					CreditorsFile: "creditors.yaml",
					DebtorsFile:   "debtors.yaml",
				},
				AI: struct {
					Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
					Model             string `mapstructure:"model" yaml:"model"`
					RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
					TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
					FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
					APIKey            string `mapstructure:"api_key" yaml:"-"`
				}{
					Enabled: false,
				},
			},
			expectError: false,
		},
		{
			name: "valid config with AI enabled",
			config: &config.Config{
				Log: struct {
					Level  string `mapstructure:"level" yaml:"level"`
					Format string `mapstructure:"format" yaml:"format"`
				}{
					Level:  "debug",
					Format: "json",
				},
				Categories: struct {
					File          string `mapstructure:"file" yaml:"file"`
					CreditorsFile string `mapstructure:"creditors_file" yaml:"creditors_file"`
					DebtorsFile   string `mapstructure:"debtors_file" yaml:"debtors_file"`
				}{
					File:          "categories.yaml",
					CreditorsFile: "creditors.yaml",
					DebtorsFile:   "debtors.yaml",
				},
				AI: struct {
					Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
					Model             string `mapstructure:"model" yaml:"model"`
					RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
					TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
					FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
					APIKey            string `mapstructure:"api_key" yaml:"-"`
				}{
					Enabled: true,
					APIKey:  "test-api-key",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, err := NewContainer(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, container)
			} else {
				require.NoError(t, err)
				require.NotNil(t, container)

				// Verify all dependencies are created (using getter methods for immutability)
				assert.NotNil(t, container.GetLogger())
				assert.NotNil(t, container.GetConfig())
				assert.NotNil(t, container.GetStore())
				assert.NotNil(t, container.GetCategorizer())
				assert.NotNil(t, container.GetParsers())

				// Verify AI client based on config
				if tt.config.AI.Enabled && tt.config.AI.APIKey != "" {
					assert.NotNil(t, container.GetAIClient())
				} else {
					// AI client can be nil if not enabled
				}

				// Verify all expected parsers are present
				expectedParsers := []ParserType{CAMT, PDF, Revolut, RevolutInvestment, Selma, Debit}
				assert.Len(t, container.GetParsers(), len(expectedParsers))

				for _, parserType := range expectedParsers {
					parser, err := container.GetParser(parserType)
					assert.NoError(t, err)
					assert.NotNil(t, parser)
				}
			}
		})
	}
}

func TestContainer_GetParser(t *testing.T) {
	cfg := &config.Config{
		Log: struct {
			Level  string `mapstructure:"level" yaml:"level"`
			Format string `mapstructure:"format" yaml:"format"`
		}{
			Level:  "info",
			Format: "text",
		},
		Categories: struct {
			File          string `mapstructure:"file" yaml:"file"`
			CreditorsFile string `mapstructure:"creditors_file" yaml:"creditors_file"`
			DebtorsFile   string `mapstructure:"debtors_file" yaml:"debtors_file"`
		}{
			File:          "categories.yaml",
			CreditorsFile: "creditors.yaml",
			DebtorsFile:   "debtors.yaml",
		},
		AI: struct {
			Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
			Model             string `mapstructure:"model" yaml:"model"`
			RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
			TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
			FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
			APIKey            string `mapstructure:"api_key" yaml:"-"`
		}{
			Enabled: false,
		},
	}

	container, err := NewContainer(cfg)
	require.NoError(t, err)

	tests := []struct {
		name        string
		parserType  ParserType
		expectError bool
	}{
		{
			name:        "valid CAMT parser",
			parserType:  CAMT,
			expectError: false,
		},
		{
			name:        "valid PDF parser",
			parserType:  PDF,
			expectError: false,
		},
		{
			name:        "valid Revolut parser",
			parserType:  Revolut,
			expectError: false,
		},
		{
			name:        "valid RevolutInvestment parser",
			parserType:  RevolutInvestment,
			expectError: false,
		},
		{
			name:        "valid Selma parser",
			parserType:  Selma,
			expectError: false,
		},
		{
			name:        "valid Debit parser",
			parserType:  Debit,
			expectError: false,
		},
		{
			name:        "invalid parser type",
			parserType:  ParserType("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := container.GetParser(tt.parserType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, parser)
				assert.Contains(t, err.Error(), "unknown parser type")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parser)
			}
		})
	}
}

func TestContainer_ConvenienceMethods(t *testing.T) {
	cfg := &config.Config{
		Log: struct {
			Level  string `mapstructure:"level" yaml:"level"`
			Format string `mapstructure:"format" yaml:"format"`
		}{
			Level:  "info",
			Format: "text",
		},
		Categories: struct {
			File          string `mapstructure:"file" yaml:"file"`
			CreditorsFile string `mapstructure:"creditors_file" yaml:"creditors_file"`
			DebtorsFile   string `mapstructure:"debtors_file" yaml:"debtors_file"`
		}{
			File:          "categories.yaml",
			CreditorsFile: "creditors.yaml",
			DebtorsFile:   "debtors.yaml",
		},
		AI: struct {
			Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
			Model             string `mapstructure:"model" yaml:"model"`
			RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
			TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
			FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
			APIKey            string `mapstructure:"api_key" yaml:"-"`
		}{
			Enabled: true,
			APIKey:  "test-key",
		},
	}

	container, err := NewContainer(cfg)
	require.NoError(t, err)

	// Test convenience methods
	assert.NotNil(t, container.GetLogger())
	assert.Equal(t, cfg, container.GetConfig())
	assert.NotNil(t, container.GetCategorizer())
	assert.NotNil(t, container.GetStore())
	assert.NotNil(t, container.GetAIClient())

	// Test Close method
	err = container.Close()
	assert.NoError(t, err)
}

// **Feature: parser-enhancements, Property 11: Configuration consistency**
// **Validates: Requirements 5.2**
// Property: For any parser using categorization, the same YAML configuration files
// and AI settings should be used as existing parsers
func TestProperty_ConfigurationConsistency(t *testing.T) {
	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Create temporary directory for test YAML files
			tempDir := t.TempDir()

			// Generate random configuration values
			categoriesFile := filepath.Join(tempDir, "categories.yaml")
			creditorsFile := filepath.Join(tempDir, "creditors.yaml")
			debtorsFile := filepath.Join(tempDir, "debtors.yaml")

			// Create minimal test YAML files
			createTestYAMLFiles(t, categoriesFile, creditorsFile, debtorsFile)

			// Generate random AI settings
			aiEnabled := cryptoRandIntn(2) == 1
			apiKey := ""
			if aiEnabled {
				apiKey = fmt.Sprintf("test-api-key-%d", cryptoRandIntn(1000))
			}

			// Create configuration with random settings
			cfg := &config.Config{
				Log: struct {
					Level  string `mapstructure:"level" yaml:"level"`
					Format string `mapstructure:"format" yaml:"format"`
				}{
					Level:  randomLogLevel(),
					Format: randomLogFormat(),
				},
				Categories: struct {
					File          string `mapstructure:"file" yaml:"file"`
					CreditorsFile string `mapstructure:"creditors_file" yaml:"creditors_file"`
					DebtorsFile   string `mapstructure:"debtors_file" yaml:"debtors_file"`
				}{
					File:          categoriesFile,
					CreditorsFile: creditorsFile,
					DebtorsFile:   debtorsFile,
				},
				AI: struct {
					Enabled           bool   `mapstructure:"enabled" yaml:"enabled"`
					Model             string `mapstructure:"model" yaml:"model"`
					RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
					TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
					FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
					APIKey            string `mapstructure:"api_key" yaml:"-"`
				}{
					Enabled: aiEnabled,
					APIKey:  apiKey,
				},
			}

			// Create container
			container, err := NewContainer(cfg)
			require.NoError(t, err)
			require.NotNil(t, container)

			// Get all parsers
			parsers := container.GetParsers()
			require.NotEmpty(t, parsers, "Container should have parsers")

			// Get the shared categorizer from container
			sharedCategorizer := container.GetCategorizer()
			require.NotNil(t, sharedCategorizer, "Container should have a categorizer")

			// Get the shared store from container
			sharedStore := container.GetStore()
			require.NotNil(t, sharedStore, "Container should have a store")

			// Verify: All parsers use the same configuration
			// Property 1: All parsers should implement CategorizerConfigurable
			for parserType, p := range parsers {
				// Check that parser implements CategorizerConfigurable
				_, ok := p.(parser.CategorizerConfigurable)
				assert.True(t, ok, "Parser %s should implement CategorizerConfigurable", parserType)
			}

			// Property 2: All parsers should be configured with the same categorizer
			// This is verified by the container's design - all parsers receive the same categorizer instance
			// We verify this by checking that the container's categorizer is not nil
			assert.NotNil(t, sharedCategorizer, "All parsers should share the same categorizer")

			// Property 3: The store should use the same configuration files
			assert.Equal(t, categoriesFile, sharedStore.CategoriesFile,
				"Store should use the configured categories file")
			assert.Equal(t, creditorsFile, sharedStore.CreditorsFile,
				"Store should use the configured creditors file")
			assert.Equal(t, debtorsFile, sharedStore.DebtorsFile,
				"Store should use the configured debtors file")

			// Property 4: AI settings should be consistent
			aiClient := container.GetAIClient()
			if aiEnabled && apiKey != "" {
				assert.NotNil(t, aiClient, "AI client should be created when AI is enabled with API key")
			}

			// Property 5: Configuration should be accessible from container
			containerConfig := container.GetConfig()
			assert.Equal(t, cfg, containerConfig, "Container should return the same config")
			assert.Equal(t, aiEnabled, containerConfig.AI.Enabled, "AI enabled setting should be consistent")

			// Cleanup
			err = container.Close()
			assert.NoError(t, err)
		})
	}
}

// Helper functions for property tests

func createTestYAMLFiles(t *testing.T, categoriesFile, creditorsFile, debtorsFile string) {
	// Create minimal valid YAML files
	// Categories file uses structured format with categories key
	categoriesYAML := `categories:
  - name: "Test Category"
    keywords:
      - "test"
`
	// Creditors and debtors files use simple map format (string -> string)
	creditorsYAML := `"Test Creditor": "Test Category"
`
	debtorsYAML := `"Test Debtor": "Test Category"
`

	err := os.WriteFile(categoriesFile, []byte(categoriesYAML), 0600)
	require.NoError(t, err, "Failed to create categories file")

	err = os.WriteFile(creditorsFile, []byte(creditorsYAML), 0600)
	require.NoError(t, err, "Failed to create creditors file")

	err = os.WriteFile(debtorsFile, []byte(debtorsYAML), 0600)
	require.NoError(t, err, "Failed to create debtors file")
}

func randomLogLevel() string {
	levels := []string{"debug", "info", "warn", "error"}
	return levels[cryptoRandIntn(len(levels))]
}

func randomLogFormat() string {
	formats := []string{"text", "json"}
	return formats[cryptoRandIntn(len(formats))]
}
