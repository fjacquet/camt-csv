// Package csvsafe neutralises CSV cells a spreadsheet would evaluate as formulas.
package csvsafe

import "strings"

// formulaTriggers are the leading characters a spreadsheet treats as the start
// of a formula (OWASP, CSV injection).
const formulaTriggers = "=+-@\t\r"

// NeedsEscape reports whether a spreadsheet could read s as a formula, or s is
// already shaped like an escaped value and so would not round-trip unescaped.
func NeedsEscape(s string) bool {
	if s == "" {
		return false
	}
	if strings.ContainsRune(formulaTriggers, rune(s[0])) {
		return true
	}
	return s[0] == '\'' && len(s) > 1 && strings.ContainsRune(formulaTriggers, rune(s[1]))
}

// Escape neutralises text that comes from untrusted input, which a spreadsheet
// would otherwise evaluate. A leading apostrophe is the spreadsheet convention
// for "this is text".
func Escape(s string) string {
	if NeedsEscape(s) {
		return "'" + s
	}
	return s
}

// Unescape reverses Escape exactly: it strips one leading apostrophe
// only when what remains is something Escape would have escaped.
func Unescape(s string) string {
	if len(s) > 1 && s[0] == '\'' && NeedsEscape(s[1:]) {
		return s[1:]
	}
	return s
}
