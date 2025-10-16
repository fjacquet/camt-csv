package parser_test

import (
	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/parser"
	"fjacquet/camt-csv/internal/pdfparser"
	"fjacquet/camt-csv/internal/revolutinvestmentparser"
	"fjacquet/camt-csv/internal/revolutparser"
	"fjacquet/camt-csv/internal/selmaparser"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetParser(t *testing.T) {
	tests := []struct {
		name         string
		parserType   parser.ParserType
		expectedType interface{}
		expectError  bool
	}{
		{
			name:         "CAMT Parser",
			parserType:   parser.CAMT,
			expectedType: &camtparser.Adapter{},
			expectError:  false,
		},
		{
			name:         "PDF Parser",
			parserType:   parser.PDF,
			expectedType: &pdfparser.Adapter{},
			expectError:  false,
		},
		{
			name:         "Revolut Parser",
			parserType:   parser.Revolut,
			expectedType: &revolutparser.Adapter{},
			expectError:  false,
		},
		{
			name:         "Revolut Investment Parser",
			parserType:   parser.RevolutInvestment,
			expectedType: &revolutinvestmentparser.Adapter{},
			expectError:  false,
		},
		{
			name:         "Selma Parser",
			parserType:   parser.Selma,
			expectedType: &selmaparser.Adapter{},
			expectError:  false,
		},
		{
			name:         "Debit Parser",
			parserType:   parser.Debit,
			expectedType: &debitparser.Adapter{},
			expectError:  false,
		},
		{
			name:         "Unknown Parser Type",
			parserType:   "unknown",
			expectedType: nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parser.GetParser(tt.parserType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, p)
				assert.Contains(t, err.Error(), "unknown parser type")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p)
				assert.IsType(t, tt.expectedType, p)
			}
		})
	}
}
