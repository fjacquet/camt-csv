// Package common provides shared functionality across different parsers.
package common

import (
	"path/filepath"
	"regexp"
	"strings"
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
