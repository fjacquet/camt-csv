package container

import (
	"fmt"

	"fjacquet/camt-csv/internal/parser"
)

// detectionOrder is the order in which parsers are offered a file during format
// detection. Structurally distinctive formats come first — CAMT.053 is XML and
// PDF has a magic header, so neither can be confused with a CSV — followed by
// the CSV formats, each of which identifies itself by its exact header row.
//
// The order matters only as a tie-break; every validator in the list is
// specific enough that at most one should accept a given file.
func detectionOrder() []ParserType {
	return []ParserType{
		CAMT,
		PDF,
		Revolut,
		RevolutCrypto,
		RevolutInvestment,
		Selma,
		Debit,
		Viseca,
	}
}

// ErrFormatNotRecognized is returned when no registered parser accepts a file.
var ErrFormatNotRecognized = fmt.Errorf("no parser recognizes this file format")

// DetectParser identifies which parser can handle filePath by asking each one
// to validate it, and returns that parser along with its type.
//
// A validator that errors is treated as a rejection, not a failure: a parser
// refusing to open a file it does not understand must not stop the others from
// trying. If nothing accepts the file, ErrFormatNotRecognized is returned.
func (c *Container) DetectParser(filePath string) (parser.FullParser, ParserType, error) {
	for _, parserType := range detectionOrder() {
		p, err := c.GetParser(parserType)
		if err != nil {
			continue
		}

		valid, err := p.ValidateFormat(filePath)
		if err != nil || !valid {
			continue
		}

		return p, parserType, nil
	}

	return nil, "", ErrFormatNotRecognized
}

// DetectionOrder returns the parser types tried during detection, in order.
func DetectionOrder() []ParserType {
	return detectionOrder()
}
