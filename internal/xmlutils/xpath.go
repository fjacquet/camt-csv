// Package xmlutils provides XML-related utility functions used throughout the application.
package xmlutils

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/xmlpath.v2"
)

var log = logrus.New()

// SetLogger sets a custom logger for this package
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
	}
}

// LoadXMLFile loads an XML file and returns the XML root node
func LoadXMLFile(xmlFilePath string) (*xmlpath.Node, error) {
	file, err := os.Open(xmlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open XML file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Warn("Failed to close file")
		}
	}()

	root, err := xmlpath.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML file: %w", err)
	}

	return root, nil
}

// ExtractFromXML extracts values from an XML node using an XPath expression
func ExtractFromXML(root *xmlpath.Node, xpath string) ([]string, error) {
	path, err := xmlpath.Compile(xpath)
	if err != nil {
		return nil, fmt.Errorf("failed to compile XPath: %w", err)
	}

	var values []string
	iter := path.Iter(root)
	for iter.Next() {
		values = append(values, iter.Node().String())
	}

	return values, nil
}

// ExtractWithXPath extracts values from an XML file using an XPath expression
func ExtractWithXPath(xmlFilePath, xpath string) ([]string, error) {
	root, err := LoadXMLFile(xmlFilePath)
	if err != nil {
		return nil, err
	}

	return ExtractFromXML(root, xpath)
}

// GetOrEmpty returns the value at the specified index in a slice, or an empty string if the index is out of bounds
func GetOrEmpty(slice []string, index int) string {
	if index < len(slice) {
		return slice[index]
	}
	return ""
}

// CleanText removes unnecessary whitespace and newlines from XML text content
func CleanText(text string) string {
	// Use strings.Builder for efficient string operations
	var builder strings.Builder
	builder.Grow(len(text)) // Pre-allocate capacity
	
	// Replace multiple spaces, tabs, and newlines with a single space
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")

	// Remove leading/trailing spaces
	text = strings.TrimSpace(text)

	// Replace multiple consecutive spaces with a single space more efficiently
	words := strings.Fields(text) // This automatically handles multiple spaces
	if len(words) > 0 {
		builder.WriteString(words[0])
		for i := 1; i < len(words); i++ {
			builder.WriteByte(' ')
			builder.WriteString(words[i])
		}
		text = builder.String()
	}

	// Remove excessive whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Remove common prefixes and noise
	prefixes := []string{
		"Remittance Info: ",
		"Remittance Information: ",
		"Additional Entry Info: ",
		"Additional Transaction Info: ",
		"Details: ",
		"End-to-End: ",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(text, prefix) {
			text = text[len(prefix):]
		}
	}

	// Remove IBAN patterns
	text = regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}[A-Z0-9]{4}[0-9]{7}([A-Z0-9]?){0,16}\b`).ReplaceAllString(text, "IBAN")

	// Remove excessive separators
	text = regexp.MustCompile(`[,;.]+\s*`).ReplaceAllString(text, " ")

	// Trim whitespace from result
	return strings.TrimSpace(text)
}
