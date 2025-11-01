package factory

import (
	"fmt"

	"fjacquet/camt-csv/internal/camtparser"
	"fjacquet/camt-csv/internal/debitparser"
	"fjacquet/camt-csv/internal/logging"
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
// Deprecated: Use GetParserWithLogger instead for dependency injection.
func GetParser(parserType ParserType) (models.Parser, error) {
	logger := logging.GetLogger()
	return GetParserWithLogger(parserType, logger)
}

// GetParserWithLogger returns a new instance of the appropriate parser for the given type
// with the provided logger for dependency injection.
func GetParserWithLogger(parserType ParserType, logger logging.Logger) (models.Parser, error) {
	switch parserType {
	case CAMT:
		return camtparser.NewAdapter(logger), nil
	case PDF:
		return pdfparser.NewAdapter(logger, nil), nil
	case Revolut:
		return revolutparser.NewAdapter(logger), nil
	case RevolutInvestment:
		return revolutinvestmentparser.NewAdapter(logger), nil
	case Selma:
		return selmaparser.NewAdapter(logger), nil
	case Debit:
		return debitparser.NewAdapter(logger), nil
	default:
		return nil, fmt.Errorf("unknown parser type: %s", parserType)
	}
}