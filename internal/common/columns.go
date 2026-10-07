package common

import "strings"

// NormalizeHeaderName strips a leading UTF-8 byte-order mark and surrounding
// whitespace from a CSV header name. Parsers use it to build their column
// lookups so they match exactly what MissingColumn accepted.
func NormalizeHeaderName(name string) string {
	return strings.TrimSpace(strings.TrimPrefix(name, "\ufeff"))
}

// MissingColumn returns the first required column absent from a CSV header,
// or "" when all are present. Header names are compared trimmed and without
// a UTF-8 byte-order mark, which spreadsheet exports often prepend.
func MissingColumn(header []string, required ...string) string {
	present := make(map[string]bool, len(header))
	for _, name := range header {
		present[NormalizeHeaderName(name)] = true
	}
	for _, name := range required {
		if !present[name] {
			return name
		}
	}
	return ""
}
