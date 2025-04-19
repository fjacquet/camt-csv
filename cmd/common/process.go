// Package common contains shared functionality for command handlers
package common

import (
	"fmt"

	"fjacquet/camt-csv/internal/parser"

	"github.com/sirupsen/logrus"
)

// ProcessFile is a helper function that handles the common validation and conversion logic
func ProcessFile(p parser.Parser, input, output string, validate bool, log *logrus.Logger) error {
	if validate {
		log.Info("Validating file format...")
		valid, err := p.ValidateFormat(input)
		if err != nil {
			return fmt.Errorf("error validating file: %w", err)
		}
		if !valid {
			return fmt.Errorf("the file is not in a valid format")
		}
		log.Info("Validation successful.")
	}

	if err := p.ConvertToCSV(input, output); err != nil {
		return fmt.Errorf("error converting file: %w", err)
	}
	
	return nil
}
