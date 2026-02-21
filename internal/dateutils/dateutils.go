// Package dateutils provides common date and time operations used throughout the application.
package dateutils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Common date format constants used throughout the application
const (
	DateLayoutISO       = "2006-01-02"
	DateLayoutEuropean  = "02.01.2006"
	DateLayoutSwiss     = "02.01.2006"
	DateLayoutUS        = "01/02/2006"
	DateLayoutFull      = "2006-01-02 15:04:05"
	DateLayoutWithMonth = "2-Jan-2006"
)

// CleanDateString removes unwanted characters and normalizes a date string
func CleanDateString(dateStr string) string {
	// Trim whitespace
	dateStr = strings.TrimSpace(dateStr)

	// Replace multiple spaces with a single space
	re := regexp.MustCompile(`\s+`)
	dateStr = re.ReplaceAllString(dateStr, " ")

	return dateStr
}

// ParseDateString attempts to parse a date string using multiple common formats
// This replaces the old FormatDate function with proper time.Time handling
// Returns the parsed time.Time or an error if parsing fails
func ParseDateString(dateStr string) (time.Time, error) {
	// Skip processing if empty
	if dateStr == "" {
		return time.Time{}, nil
	}

	// Clean the input string
	cleanDate := CleanDateString(dateStr)

	// Try various date formats commonly found in financial data
	formats := []string{
		DateLayoutEuropean,                // DD.MM.YYYY (Swiss/European)
		DateLayoutISO,                     // YYYY-MM-DD (ISO)
		DateLayoutFull,                    // YYYY-MM-DD HH:MM:SS
		DateLayoutISO + "T15:04:05Z",      // ISO 8601
		DateLayoutISO + "T15:04:05-07:00", // ISO 8601 with timezone
		"02/01/2006",                      // DD/MM/YYYY (European)
		DateLayoutUS,                      // MM/DD/YYYY (US format)
		"02-01-2006",                      // DD-MM-YYYY
		"01-02-2006",                      // MM-DD-YYYY
		"2.1.2006",                        // D.M.YYYY
		"January 2, 2006",                 // Month D, YYYY
		"2 January 2006",                  // D Month YYYY
		"02 Jan 2006",                     // DD MMM YYYY
		"Jan 02, 2006",                    // MMM DD, YYYY
		"January 2006",                    // Month YYYY (for monthly statements)
		"Jan 2006",                        // MMM YYYY (abbreviated month)
		"01/2006",                         // MM/YYYY
		"2006/01",                         // YYYY/MM
	}

	// Try each format until one works
	for _, format := range formats {
		if t, err := time.Parse(format, cleanDate); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
