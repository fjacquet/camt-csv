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

// CommonFormats is a list of standard formats to try when parsing dates
var CommonFormats = []string{
	DateLayoutISO,
	DateLayoutEuropean,
	DateLayoutSwiss,
	DateLayoutUS,
	DateLayoutFull,
	DateLayoutWithMonth,
	"02-01-2006",
	"02/01/2006",
	"2006/01/02",
	"Jan 2, 2006",
	"January 2, 2006",
}

// ParseDate attempts to parse a date string using multiple common formats
// Returns the parsed time and the detected format
func ParseDate(dateStr string) (time.Time, string, error) {
	// Clean up the date string
	dateStr = CleanDateString(dateStr)

	// Try each format until one works
	for _, format := range CommonFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, format, nil
		}
	}

	return time.Time{}, "", fmt.Errorf("unable to parse date: %s", dateStr)
}

// FormatDate formats a time.Time value according to the specified layout
// If no layout is provided, DateLayoutISO is used
func FormatDate(date time.Time, layout string) string {
	if layout == "" {
		layout = DateLayoutISO
	}
	return date.Format(layout)
}

// ToISODate formats a time.Time value as an ISO date (YYYY-MM-DD)
func ToISODate(date time.Time) string {
	return date.Format(DateLayoutISO)
}

// CleanDateString removes unwanted characters and normalizes a date string
func CleanDateString(dateStr string) string {
	// Trim whitespace
	dateStr = strings.TrimSpace(dateStr)

	// Replace multiple spaces with a single space
	re := regexp.MustCompile(`\s+`)
	dateStr = re.ReplaceAllString(dateStr, " ")

	return dateStr
}

// IsWeekend checks if a date falls on a weekend (Saturday or Sunday)
func IsWeekend(date time.Time) bool {
	day := date.Weekday()
	return day == time.Saturday || day == time.Sunday
}

// IsBusinessDay checks if a date is a business day (not a weekend)
// Does not account for holidays
func IsBusinessDay(date time.Time) bool {
	return !IsWeekend(date)
}

// StartOfMonth returns the first day of the month for a given date
func StartOfMonth(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}

// EndOfMonth returns the last day of the month for a given date
func EndOfMonth(date time.Time) time.Time {
	return StartOfMonth(date).AddDate(0, 1, -1)
}

// NextBusinessDay returns the next business day after a given date
// If the date is a weekday, it returns the next day
// If the date is a Friday, it returns the following Monday
// If the date is a Saturday, it returns the following Monday
func NextBusinessDay(date time.Time) time.Time {
	day := date.Weekday()
	switch day {
	case time.Friday:
		return date.AddDate(0, 0, 3)
	case time.Saturday:
		return date.AddDate(0, 0, 2)
	default:
		return date.AddDate(0, 0, 1)
	}
}

// PreviousBusinessDay returns the previous business day before a given date
// If the date is a weekday other than Monday, it returns the previous day
// If the date is a Monday, it returns the previous Friday
// If the date is a Sunday, it returns the previous Friday
func PreviousBusinessDay(date time.Time) time.Time {
	day := date.Weekday()
	switch day {
	case time.Monday:
		return date.AddDate(0, 0, -3)
	case time.Sunday:
		return date.AddDate(0, 0, -2)
	default:
		return date.AddDate(0, 0, -1)
	}
}

// ExtractYear extracts the year from a date string
// Returns the current year if the year cannot be extracted
func ExtractYear(dateStr string) int {
	t, _, err := ParseDate(dateStr)
	if err != nil {
		return time.Now().Year()
	}
	return t.Year()
}

// CompareDates compares two dates and returns:
//
//	-1 if date1 is before date2
//	 0 if date1 is equal to date2
//	 1 if date1 is after date2
func CompareDates(date1, date2 time.Time) int {
	// Normalize dates to remove time component
	date1 = time.Date(date1.Year(), date1.Month(), date1.Day(), 0, 0, 0, 0, time.UTC)
	date2 = time.Date(date2.Year(), date2.Month(), date2.Day(), 0, 0, 0, 0, time.UTC)

	if date1.Before(date2) {
		return -1
	} else if date1.After(date2) {
		return 1
	} else {
		return 0
	}
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

// ToSwissFormat formats a time.Time as DD.MM.YYYY (Swiss format)
func ToSwissFormat(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	return date.Format(DateLayoutSwiss)
}
