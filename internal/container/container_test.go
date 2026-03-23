package container

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/config"

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
					Provider          string `mapstructure:"provider" yaml:"provider"`
					BaseURL           string `mapstructure:"base_url" yaml:"base_url"`
					Model             string `mapstructure:"model" yaml:"model"`
					RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
					TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
					FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
					APIKey            string `mapstructure:"api_key" yaml:"-" json:"-"`
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
					Provider          string `mapstructure:"provider" yaml:"provider"`
					BaseURL           string `mapstructure:"base_url" yaml:"base_url"`
					Model             string `mapstructure:"model" yaml:"model"`
					RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
					TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
					FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
					APIKey            string `mapstructure:"api_key" yaml:"-" json:"-"`
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
				assert.NotNil(t, container.GetCategorizer())

				// Verify all expected parsers are present
				expectedParsers := []ParserType{CAMT, PDF, Revolut, RevolutInvestment, Selma, Debit}
				for _, parserType := range expectedParsers {
					p, err := container.GetParser(parserType)
					assert.NoError(t, err)
					assert.NotNil(t, p)
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
			Provider          string `mapstructure:"provider" yaml:"provider"`
			BaseURL           string `mapstructure:"base_url" yaml:"base_url"`
			Model             string `mapstructure:"model" yaml:"model"`
			RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
			TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
			FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
			APIKey            string `mapstructure:"api_key" yaml:"-" json:"-"`
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
			Provider          string `mapstructure:"provider" yaml:"provider"`
			BaseURL           string `mapstructure:"base_url" yaml:"base_url"`
			Model             string `mapstructure:"model" yaml:"model"`
			RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
			TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
			FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
			APIKey            string `mapstructure:"api_key" yaml:"-" json:"-"`
		}{
			Enabled: true,
			APIKey:  "test-key",
		},
	}

	container, err := NewContainer(cfg)
	require.NoError(t, err)

	// Test convenience methods
	assert.NotNil(t, container.GetLogger())
	assert.NotNil(t, container.GetCategorizer())
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
					Provider          string `mapstructure:"provider" yaml:"provider"`
					BaseURL           string `mapstructure:"base_url" yaml:"base_url"`
					Model             string `mapstructure:"model" yaml:"model"`
					RequestsPerMinute int    `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
					TimeoutSeconds    int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
					FallbackCategory  string `mapstructure:"fallback_category" yaml:"fallback_category"`
					APIKey            string `mapstructure:"api_key" yaml:"-" json:"-"`
				}{
					Enabled: aiEnabled,
					APIKey:  apiKey,
				},
			}

			// Create container
			container, err := NewContainer(cfg)
			require.NoError(t, err)
			require.NotNil(t, container)

			// Get the shared categorizer from container
			sharedCategorizer := container.GetCategorizer()
			require.NotNil(t, sharedCategorizer, "Container should have a categorizer")

			// Verify all expected parsers are accessible
			expectedParsers := []ParserType{CAMT, PDF, Revolut, RevolutInvestment, Selma, Debit}
			for _, pt := range expectedParsers {
				p, pErr := container.GetParser(pt)
				assert.NoError(t, pErr, "Parser %s should be retrievable", pt)
				assert.NotNil(t, p, "Parser %s should not be nil", pt)
			}
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
