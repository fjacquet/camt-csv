package dateutils

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetLogger(t *testing.T) {
	// Create a custom logger
	customLogger := logrus.New()
	customLogger.SetLevel(logrus.DebugLevel)
	
	// Save the original logger to restore after test
	originalLogger := log
	defer func() {
		log = originalLogger
	}()
	
	// Test with valid logger
	SetLogger(customLogger)
	assert.Equal(t, customLogger, log)
	
	// Test with nil logger (should not change the current logger)
	currentLogger := log
	SetLogger(nil)
	assert.Equal(t, currentLogger, log)
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		expectedOk  bool
		expectedY   int
		expectedM   time.Month
		expectedD   int
		expectedFmt string
	}{
		{"ISO format", "2023-01-15", true, 2023, time.January, 15, DateLayoutISO},
		{"European format", "15.01.2023", true, 2023, time.January, 15, DateLayoutEuropean},
		{"US format", "01/15/2023", true, 2023, time.January, 15, DateLayoutUS},
		{"Dash-separated EU", "15-01-2023", true, 2023, time.January, 15, "02-01-2006"},
		{"Full timestamp", "2023-01-15 10:30:45", true, 2023, time.January, 15, DateLayoutFull},
		{"With month name", "15-Jan-2023", true, 2023, time.January, 15, DateLayoutWithMonth},
		{"Empty string", "", false, 0, 0, 0, ""},
		{"Invalid format", "not a date", false, 0, 0, 0, ""},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			date, format, err := ParseDate(tc.dateStr)
			
			if tc.expectedOk {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedY, date.Year())
				assert.Equal(t, tc.expectedM, date.Month())
				assert.Equal(t, tc.expectedD, date.Day())
				assert.Equal(t, tc.expectedFmt, format)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	// Create a fixed test date (January 15, 2023)
	testDate := time.Date(2023, time.January, 15, 10, 30, 0, 0, time.UTC)
	
	tests := []struct {
		name     string
		layout   string
		expected string
	}{
		{"Default ISO layout", "", "2023-01-15"},
		{"Explicit ISO layout", DateLayoutISO, "2023-01-15"},
		{"European layout", DateLayoutEuropean, "15.01.2023"},
		{"US layout", DateLayoutUS, "01/15/2023"},
		{"Full layout", DateLayoutFull, "2023-01-15 10:30:00"},
		{"Custom layout", "Mon, 02 Jan 2006", "Sun, 15 Jan 2023"},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDate(testDate, tc.layout)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestToISODate(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{
			"Normal date",
			time.Date(2023, time.January, 15, 10, 30, 0, 0, time.UTC),
			"2023-01-15",
		},
		{
			"Zero date",
			time.Time{},
			"0001-01-01",
		},
		{
			"Future date",
			time.Date(2050, time.December, 31, 23, 59, 59, 0, time.UTC),
			"2050-12-31",
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToISODate(tc.date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCleanDateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Already clean", "2023-01-15", "2023-01-15"},
		{"With spaces", "  2023-01-15  ", "2023-01-15"},
		{"Multiple spaces", "2023  01  15", "2023 01 15"},
		{"Empty string", "", ""},
		{"Only whitespace", "   ", ""},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CleanDateString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsWeekend(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			"Monday (weekday)",
			time.Date(2023, time.January, 16, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Wednesday (weekday)",
			time.Date(2023, time.January, 18, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Saturday (weekend)",
			time.Date(2023, time.January, 14, 0, 0, 0, 0, time.UTC),
			true,
		},
		{
			"Sunday (weekend)",
			time.Date(2023, time.January, 15, 0, 0, 0, 0, time.UTC),
			true,
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsWeekend(tc.date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsBusinessDay(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			"Monday (business day)",
			time.Date(2023, time.January, 16, 0, 0, 0, 0, time.UTC),
			true,
		},
		{
			"Friday (business day)",
			time.Date(2023, time.January, 20, 0, 0, 0, 0, time.UTC),
			true,
		},
		{
			"Saturday (not business day)",
			time.Date(2023, time.January, 21, 0, 0, 0, 0, time.UTC),
			false,
		},
		{
			"Sunday (not business day)",
			time.Date(2023, time.January, 22, 0, 0, 0, 0, time.UTC),
			false,
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsBusinessDay(tc.date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStartOfMonth(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected time.Time
	}{
		{
			"Start of month already",
			time.Date(2023, time.January, 1, 10, 30, 0, 0, time.UTC),
			time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"Middle of month",
			time.Date(2023, time.February, 15, 10, 30, 0, 0, time.UTC),
			time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			"End of month",
			time.Date(2023, time.March, 31, 10, 30, 0, 0, time.UTC),
			time.Date(2023, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StartOfMonth(tc.date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEndOfMonth(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected time.Time
	}{
		{
			"January (31 days)",
			time.Date(2023, time.January, 15, 10, 30, 0, 0, time.UTC),
			time.Date(2023, time.January, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			"February 2023 (28 days)",
			time.Date(2023, time.February, 1, 10, 30, 0, 0, time.UTC),
			time.Date(2023, time.February, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			"February 2024 (leap year, 29 days)",
			time.Date(2024, time.February, 15, 10, 30, 0, 0, time.UTC),
			time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			"April (30 days)",
			time.Date(2023, time.April, 30, 10, 30, 0, 0, time.UTC),
			time.Date(2023, time.April, 30, 0, 0, 0, 0, time.UTC),
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := EndOfMonth(tc.date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNextBusinessDay(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected time.Time
	}{
		{
			"Monday → Tuesday",
			time.Date(2023, time.January, 16, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			"Friday → Monday (skip weekend)",
			time.Date(2023, time.January, 20, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			"Saturday → Monday (skip rest of weekend)",
			time.Date(2023, time.January, 21, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			"Sunday → Monday",
			time.Date(2023, time.January, 22, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 23, 0, 0, 0, 0, time.UTC),
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NextBusinessDay(tc.date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPreviousBusinessDay(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected time.Time
	}{
		{
			"Tuesday → Monday",
			time.Date(2023, time.January, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			"Monday → Friday (skip weekend)",
			time.Date(2023, time.January, 16, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			"Sunday → Friday (skip weekend)",
			time.Date(2023, time.January, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			"Saturday → Friday",
			time.Date(2023, time.January, 14, 0, 0, 0, 0, time.UTC),
			time.Date(2023, time.January, 13, 0, 0, 0, 0, time.UTC),
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := PreviousBusinessDay(tc.date)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractYear(t *testing.T) {
	currentYear := time.Now().Year()
	
	tests := []struct {
		name     string
		dateStr  string
		expected int
	}{
		{"ISO format", "2023-01-15", 2023},
		{"European format", "15.01.2023", 2023},
		{"US format", "01/15/2023", 2023},
		{"Invalid date", "not a date", currentYear},
		{"Empty string", "", currentYear},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractYear(tc.dateStr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCompareDates(t *testing.T) {
	date1 := time.Date(2023, time.January, 15, 10, 30, 0, 0, time.UTC)
	date2 := time.Date(2023, time.January, 15, 15, 45, 0, 0, time.UTC)
	date3 := time.Date(2023, time.January, 16, 10, 30, 0, 0, time.UTC)
	date4 := time.Date(2023, time.February, 15, 10, 30, 0, 0, time.UTC)
	date5 := time.Date(2022, time.January, 15, 10, 30, 0, 0, time.UTC)
	
	tests := []struct {
		name     string
		date1    time.Time
		date2    time.Time
		expected int
	}{
		{"Same day, different time", date1, date2, 0},
		{"Next day", date1, date3, -1},
		{"Previous day", date3, date1, 1},
		{"Next month", date1, date4, -1},
		{"Previous year", date5, date1, -1},
		{"Equal dates", date1, date1, 0},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CompareDates(tc.date1, tc.date2)
			assert.Equal(t, tc.expected, result)
		})
	}
}
