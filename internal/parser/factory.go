package parser

import (
	"fmt"

	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/pdfparser"
	"fjacquet/camt-csv/internal/revolutinvestmentparser"
	"fjacquet/camt-csv/internal/revolutparser"
	"fjacquet/camt-csv/internal/selmaparser"
)

// ParserType defines the types of parsers available.
type ParserType string

const (
	CAMT              ParserType = "camt"
	PDF               ParserType = "pdf"
	Revolut           ParserType = "revolut"
	RevolutInvestment ParserType = "revolut-investment"
	Selma             ParserType = "selma"
	Debit             ParserType = "debit"
)

// GetParser returns a new instance of the appropriate parser for the given type.
// It acts as a factory for creating Parser implementations.
func GetParser(parserType ParserType) (models.Parser, error) {
	switch parserType {
	case CAMT:
		return camtparser.NewAdapter(), nil
	case PDF:
		return pdfparser.NewAdapter(), nil
	case Revolut:
		return revolutparser.NewAdapter(), nil
	case RevolutInvestment:
		return revolutinvestmentparser.NewAdapter(), nil
	case Selma:
		return selmaparser.NewAdapter(), nil
	case Debit:
		return debitparser.NewAdapter(), nil
	default:
		return nil, fmt.Errorf("unknown parser type: %s", parserType)
	}
}
