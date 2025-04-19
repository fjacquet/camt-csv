package revolutparser

import "github.com/sirupsen/logrus"

// Adapter implements the parser.Parser interface by wrapping
// the package-level functions of revolutparser.
type Adapter struct{}

// ValidateFormat implements parser.Parser.ValidateFormat
// by delegating to the package-level function.
func (a *Adapter) ValidateFormat(filePath string) (bool, error) {
	return ValidateFormat(filePath)
}

// ConvertToCSV implements parser.Parser.ConvertToCSV
// by delegating to the package-level function.
func (a *Adapter) ConvertToCSV(inputFile, outputFile string) error {
	return ConvertToCSV(inputFile, outputFile)
}

// SetLogger implements parser.Parser.SetLogger
// by delegating to the package-level function.
func (a *Adapter) SetLogger(logger *logrus.Logger) {
	SetLogger(logger)
}

// NewAdapter creates a new adapter for the revolutparser.
func NewAdapter() *Adapter {
	return &Adapter{}
}
