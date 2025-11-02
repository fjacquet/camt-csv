package container

import (
	"testing"

	"fjacquet/camt-csv/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

				// Verify all dependencies are created
				assert.NotNil(t, container.Logger)
				assert.NotNil(t, container.Config)
				assert.NotNil(t, container.Store)
				assert.NotNil(t, container.Categorizer)
				assert.NotNil(t, container.Parsers)

				// Verify AI client based on config
				if tt.config.AI.Enabled && tt.config.AI.APIKey != "" {
					assert.NotNil(t, container.AIClient)
				} else {
					// AI client can be nil if not enabled
				}

				// Verify all expected parsers are present
				expectedParsers := []ParserType{CAMT, PDF, Revolut, RevolutInvestment, Selma, Debit}
				assert.Len(t, container.Parsers, len(expectedParsers))

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