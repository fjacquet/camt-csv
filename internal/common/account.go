// Package common provides shared functionality across different parsers.
package common

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// AccountIdentifier represents an extracted account identifier with its source
type AccountIdentifier struct {
	ID     string // The account identifier (e.g., "54293249")
	Source string // Source of identification: "filename", "content", "default"
}

// CAMT filename pattern: CAMT.053_{account}_{start_date}_{end_date}_{sequence}.{ext}
// Example: CAMT.053_54293249_2025-04-01_2025-04-30_1.xml
var camtFilenamePattern = regexp.MustCompile(`^CAMT\.053_([0-9]+)_\d{4}-\d{2}-\d{2}_\d{4}-\d{2}-\d{2}_\d+\.(xml|csv)$`)

// ExtractAccountFromCAMTFilename extracts the account number from CAMT filename patterns
// Supports patterns like: CAMT.053_54293249_2025-04-01_2025-04-30_1.xml
// Returns AccountIdentifier with the extracted account number or fallback to base filename
func ExtractAccountFromCAMTFilename(filename string) AccountIdentifier {
	// Get just the filename without path
	baseName := filepath.Base(filename)

	// Try to match the CAMT pattern
	matches := camtFilenamePattern.FindStringSubmatch(baseName)
	if len(matches) >= 2 {
		return AccountIdentifier{
			ID:     matches[1], // The account number from the first capture group
			Source: "filename",
		}
	}

	// Fallback: use the base filename without extension as account ID
	baseWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	return AccountIdentifier{
		ID:     SanitizeAccountID(baseWithoutExt),
		Source: "default",
	}
}

// SanitizeAccountID sanitizes an account identifier to be filesystem-safe
// Removes or replaces characters that are not safe for filenames
// Also removes path traversal sequences like ".." for security
func SanitizeAccountID(accountID string) string {
	// Remove leading/trailing whitespace
	sanitized := strings.TrimSpace(accountID)

	// Replace spaces with underscores
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Remove or replace problematic characters for filesystem safety
	// Keep alphanumeric, underscores, hyphens, and dots
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == '.' {
			result.WriteRune(r)
		} else {
			// Replace other characters with underscore
			result.WriteRune('_')
		}
	}

	sanitized = result.String()

	// Remove path traversal sequences ".." for security
	// This prevents directory traversal attacks
	for strings.Contains(sanitized, "..") {
		sanitized = strings.ReplaceAll(sanitized, "..", "_")
	}

	// Remove multiple consecutive underscores
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}

	// Remove leading/trailing underscores and dots
	sanitized = strings.Trim(sanitized, "_.")

	// Ensure we have at least something
	if sanitized == "" {
		sanitized = "UNKNOWN"
	}

	return sanitized
}

// ExtractAccountFromFilename is a generic function that tries to extract account information
// from various filename patterns. It delegates to specific extraction functions based on
// the filename pattern detected.
func ExtractAccountFromFilename(filename string) AccountIdentifier {
	baseName := filepath.Base(filename)

	// Check if it's a CAMT file
	if strings.HasPrefix(strings.ToUpper(baseName), "CAMT.053_") {
		return ExtractAccountFromCAMTFilename(filename)
	}

	// For other file types, use the base filename as fallback
	baseWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	return AccountIdentifier{
		ID:     SanitizeAccountID(baseWithoutExt),
		Source: "default",
	}
}

// Account numbers in this bank's exports appear in one of two places: after
// the CAMT.053_ prefix of an ISO 20022 statement, or at the very start of a
// PDF statement's name. Both are matched against the base name only.
//
// The length floors reject the two numbers that sit near an account number
// without being one: the 053 of the format prefix, and a single-digit
// sequence number leading a file name.
// camtAccountKeyPattern is deliberately looser than camtFilenamePattern above,
// which additionally requires the date range, sequence number, and a known
// extension: a CAMT export that names those parts differently still belongs to
// an account, and grouping it as "unknown" would be worse than reading the
// number it plainly carries. Both encode the same CAMT.053_<account>_ prefix,
// so a change to that convention has to be made in both.
// minAccountKeyDigits is the shortest digit run treated as an account number
// rather than a sequence number or a fragment of something else. It is the
// same floor the file-name patterns use, so both sources agree on what counts.
const minAccountKeyDigits = 6

var (
	camtAccountKeyPattern    = regexp.MustCompile(`(?i)^camt\.053_(\d{4,})_`)
	leadingAccountKeyPattern = regexp.MustCompile(`^(\d{6,})[_.]`)
)

// AccountKeyFromFilename returns the number of the account a statement file
// belongs to, or "" when its name carries none.
//
// The name is the only source available: CAMT.053 carries the statement's own
// account in <Stmt><Acct>, but camtparser's schema models only counterparty
// accounts (internal/camtparser/camt053_schema.go), and parser.Parser.Parse
// takes an io.Reader with no filename channel — so reading identity out of the
// statement instead would mean changing that schema, models.Transaction, and
// the Parser interface every format implements.
//
// This is deliberately not ExtractAccountFromFilename: that helper always
// answers with something, falling back to the whole base name, which is the
// right behaviour for labelling but the wrong one for grouping — every
// unrecognized file would become its own account. Callers grouping by account
// need to be able to tell "account 54293249" from "no account here".
func AccountKeyFromFilename(path string) string {
	baseName := filepath.Base(path)

	if matches := camtAccountKeyPattern.FindStringSubmatch(baseName); len(matches) >= 2 {
		return matches[1]
	}

	if matches := leadingAccountKeyPattern.FindStringSubmatch(baseName); len(matches) >= 2 {
		if isCompactDate(matches[1]) {
			return ""
		}
		return matches[1]
	}

	return ""
}

// isCompactDate reports whether a digit run is a YYYYMMDD date rather than an
// account number.
//
// Exports are routinely named by date, and a compact date occupies exactly the
// position an account number does: 20260401_releve.pdf. Reading it as an
// account writes one CSV per month for a single account — the mirror image of
// the mixing this split exists to prevent, and just as invisible. A digit run
// that only resembles a date (20261301) is not one, and stays an account.
func isCompactDate(digits string) bool {
	if len(digits) != 8 {
		return false
	}
	_, err := time.Parse("20060102", digits)
	return err == nil
}

// AccountKeyFromIBAN reduces an account identifier read from a statement to
// the key used to name that account's output.
//
// It takes the trailing run of digits when there is a long enough one, because
// that is the number the same bank puts in its file names: the statement says
// CH17 0076 7000 K542 9324 9 and the file is called CAMT.053_54293249_…, and
// both must land in one CSV rather than two. The letter that separates the
// account number from the branch part of a Swiss IBAN is what makes the run
// end where it does; an identifier with no such run keeps its whole sanitized
// form, which is still stable, just longer.
func AccountKeyFromIBAN(iban string) string {
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, strings.TrimSpace(iban))

	if compact == "" {
		return ""
	}

	digits := 0
	for i := len(compact) - 1; i >= 0 && compact[i] >= '0' && compact[i] <= '9'; i-- {
		digits++
	}

	if digits >= minAccountKeyDigits {
		return compact[len(compact)-digits:]
	}

	return SanitizeAccountID(compact)
}
