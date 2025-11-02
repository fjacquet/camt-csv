package factory_test

import (
	"testing"

	"fjacquet/camt-csv/internal/factory"
	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
)

func TestGetParser(t *testing.T) {
	tests := []struct {
		name        string
		parserType  factory.ParserType
		expectError bool
	}{
		{
			name:        "CAMT Parser",
			parserType:  factory.CAMT,
			expectError: false,
		},
		{
			name:        "PDF Parser",
			parserType:  factory.PDF,
			expectError: false,
		},
		{
			name:        "Revolut Parser",
			parserType:  factory.Revolut,
			expectError: false,
		},
		{
			name:        "Revolut Investment Parser",
			parserType:  factory.RevolutInvestment,
			expectError: false,
		},
		{
			name:        "Selma Parser",
			parserType:  factory.Selma,
			expectError: false,
		},
		{
			name:        "Debit Parser",
			parserType:  factory.Debit,
			expectError: false,
		},
		{
			name:        "Unknown Parser Type",
			parserType:  "unknown",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewLogrusAdapter("info", "text")
			p, err := factory.GetParserWithLogger(tt.parserType, logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, p)
				assert.Contains(t, err.Error(), "unknown parser type")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p)
			}
		})
	}
}

func TestGetParserWithLogger(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	
	p, err := factory.GetParserWithLogger(factory.CAMT, logger)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	
	// Test that we can set a logger on the parser
	p.SetLogger(logger)
}