package common

import "strings"

// MissingColumn returns the first required column absent from a CSV header,
// or "" when all are present. Header names are compared trimmed and without
// a UTF-8 byte-order mark, which spreadsheet exports often prepend.
func MissingColumn(header []string, required ...string) string {
	present := make(map[string]bool, len(header))
	for _, name := range header {
		present[strings.TrimSpace(strings.TrimPrefix(name, "\ufeff"))] = true
	}
	for _, name := range required {
		if !present[name] {
			return name
		}
	}
	return ""
}
