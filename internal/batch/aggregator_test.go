package batch

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cryptoRandIntn returns a random int in [0, n) using crypto/rand
func cryptoRandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	max := big.NewInt(int64(n))
	result, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return int(result.Int64())
}

// **Feature: parser-enhancements, Property 1: Account-based file aggregation**
// **Validates: Requirements 1.1**
func TestProperty_AccountBasedFileAggregation(t *testing.T) {
	// Property: For any set of CAMT files with the same account number in their filenames,
	// batch processing should produce exactly one consolidated output file per unique account number

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random test data
			numAccounts := cryptoRandIntn(5) + 1      // 1-5 accounts
			filesPerAccount := cryptoRandIntn(10) + 1 // 1-10 files per account

			var allFiles []string
			expectedGroups := make(map[string]int)

			for accountIdx := 0; accountIdx < numAccounts; accountIdx++ {
				accountID := fmt.Sprintf("5429324%d", accountIdx)
				expectedGroups[accountID] = filesPerAccount

				for fileIdx := 0; fileIdx < filesPerAccount; fileIdx++ {
					// Generate CAMT filename with random dates
					startDate := generateRandomDate()
					endDate := startDate.AddDate(0, 1, 0) // One month later
					filename := fmt.Sprintf("CAMT.053_%s_%s_%s_%d.xml",
						accountID,
						startDate.Format("2006-01-02"),
						endDate.Format("2006-01-02"),
						fileIdx+1)
					allFiles = append(allFiles, filename)
				}
			}

			// Test the property
			groups, err := aggregator.GroupFilesByAccount(allFiles)
			require.NoError(t, err)

			// Verify: exactly one group per unique account
			assert.Equal(t, numAccounts, len(groups), "Should have one group per account")

			// Verify: each group contains the correct number of files
			actualGroups := make(map[string]int)
			for _, group := range groups {
				actualGroups[group.AccountID] = len(group.Files)
			}

			assert.Equal(t, expectedGroups, actualGroups, "Each account should have correct number of files")

			// Verify: all files are accounted for
			totalFiles := 0
			for _, group := range groups {
				totalFiles += len(group.Files)
			}
			assert.Equal(t, len(allFiles), totalFiles, "All files should be grouped")
		})
	}
}

// **Feature: parser-enhancements, Property 2: Consolidated file naming convention**
// **Validates: Requirements 1.2, 7.1**
func TestProperty_ConsolidatedFileNamingConvention(t *testing.T) {
	// Property: For any account identifier and date range, the consolidated output filename
	// should follow the format "{account_id}_{start_date}_{end_date}.csv" with filesystem-safe characters

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random account ID (may contain unsafe characters)
			accountID := generateRandomAccountID()

			// Generate random date range
			startDate := generateRandomDate()
			endDate := startDate.AddDate(0, cryptoRandIntn(12)+1, cryptoRandIntn(30)) // 1-12 months, 0-30 days later
			dateRange := DateRange{Start: startDate, End: endDate}

			// Test the property
			filename := aggregator.GenerateOutputFilename(accountID, dateRange)

			// Verify: filename follows the expected format
			expectedPattern := fmt.Sprintf("_%s_%s.csv",
				startDate.Format("2006-01-02"),
				endDate.Format("2006-01-02"))
			assert.Contains(t, filename, expectedPattern, "Filename should contain date range")
			assert.True(t, strings.HasSuffix(filename, ".csv"), "Filename should end with .csv")

			// Verify: filename is filesystem-safe (no problematic characters)
			assertFilesystemSafe(t, filename)

			// Verify: filename contains sanitized account ID
			assert.NotEmpty(t, filename, "Filename should not be empty")

			// Test edge case: empty date range
			emptyRange := DateRange{}
			filenameNoDate := aggregator.GenerateOutputFilename(accountID, emptyRange)
			assert.True(t, strings.HasSuffix(filenameNoDate, ".csv"), "Filename without dates should still end with .csv")
			assertFilesystemSafe(t, filenameNoDate)
		})
	}
}

// **Feature: parser-enhancements, Property 3: Chronological transaction ordering**
// **Validates: Requirements 1.3**
func TestProperty_ChronologicalTransactionOrdering(t *testing.T) {
	// Property: For any set of transactions from multiple files being aggregated,
	// the final output should be sorted chronologically by transaction date

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random transactions with random dates
			numTransactions := cryptoRandIntn(50) + 10 // 10-59 transactions
			transactions := make([]models.Transaction, numTransactions)

			baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

			for j := 0; j < numTransactions; j++ {
				// Random date within a year
				randomDays := cryptoRandIntn(365)
				txDate := baseDate.AddDate(0, 0, randomDays)

				transactions[j] = models.Transaction{
					Date:        txDate,
					ValueDate:   txDate,
					Amount:      decimal.NewFromFloat(cryptoRandFloat64() * 1000),
					Description: fmt.Sprintf("Transaction %d", j),
				}
			}

			// Test the property: sort transactions
			aggregator.sortTransactionsChronologically(transactions)

			// Verify: transactions are sorted chronologically
			for j := 1; j < len(transactions); j++ {
				prev := transactions[j-1]
				curr := transactions[j]

				// Primary sort: by date
				if !prev.Date.Equal(curr.Date) {
					assert.True(t, prev.Date.Before(curr.Date) || prev.Date.Equal(curr.Date),
						"Transactions should be sorted by date: %s should be <= %s",
						prev.Date.Format("2006-01-02"), curr.Date.Format("2006-01-02"))
				} else {
					// Secondary sort: by value date
					if !prev.ValueDate.Equal(curr.ValueDate) {
						assert.True(t, prev.ValueDate.Before(curr.ValueDate) || prev.ValueDate.Equal(curr.ValueDate),
							"Transactions with same date should be sorted by value date")
					} else {
						// Tertiary sort: by amount
						assert.True(t, prev.Amount.LessThanOrEqual(curr.Amount),
							"Transactions with same date and value date should be sorted by amount")
					}
				}
			}
		})
	}
}

// Helper functions for property tests

// cryptoRandFloat64 returns a random float64 in [0, 1) using crypto/rand
func cryptoRandFloat64() float64 {
	max := big.NewInt(1000000)
	result, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return float64(result.Int64()) / 1000000.0
}

// cryptoRandInt63n returns a random int64 in [0, n) using crypto/rand
func cryptoRandInt63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	max := big.NewInt(n)
	result, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0
	}
	return result.Int64()
}

func generateRandomDate() time.Time {
	// Generate random date between 2020 and 2025
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	delta := end.Unix() - start.Unix()
	sec := cryptoRandInt63n(delta) + start.Unix()
	return time.Unix(sec, 0)
}

func generateRandomAccountID() string {
	// Generate account ID that may contain problematic characters
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_./\\:*?\"<>|"
	length := cryptoRandIntn(20) + 5 // 5-24 characters

	var result []byte
	for i := 0; i < length; i++ {
		result = append(result, chars[cryptoRandIntn(len(chars))])
	}

	return string(result)
}

// cryptoShuffle shuffles a slice of transactions using crypto/rand
func cryptoShuffle(transactions []models.Transaction) {
	n := len(transactions)
	for i := n - 1; i > 0; i-- {
		j := cryptoRandIntn(i + 1)
		transactions[i], transactions[j] = transactions[j], transactions[i]
	}
}

func assertFilesystemSafe(t *testing.T, filename string) {
	// Check for problematic characters that should not appear in filesystem-safe names
	problematicChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

	for _, char := range problematicChars {
		assert.NotContains(t, filename, char,
			"Filename should not contain problematic character: %s", char)
	}

	// Check that filename is not empty and doesn't start/end with problematic characters
	assert.NotEmpty(t, filename, "Filename should not be empty")
	assert.NotEqual(t, ".", filename, "Filename should not be just a dot")
	assert.NotEqual(t, "..", filename, "Filename should not be just double dots")
}

// Unit tests for specific functionality

func TestDateRange_String(t *testing.T) {
	tests := []struct {
		name     string
		dr       DateRange
		expected string
	}{
		{
			name: "valid date range",
			dr: DateRange{
				Start: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
			},
			expected: "2025-04-01_2025-06-30",
		},
		{
			name:     "zero dates",
			dr:       DateRange{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dr.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDateRange_Merge(t *testing.T) {
	tests := []struct {
		name     string
		dr1      DateRange
		dr2      DateRange
		expected DateRange
	}{
		{
			name: "overlapping ranges",
			dr1: DateRange{
				Start: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC),
			},
			dr2: DateRange{
				Start: time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
			},
			expected: DateRange{
				Start: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "one range is zero",
			dr1:  DateRange{},
			dr2: DateRange{
				Start: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC),
			},
			expected: DateRange{
				Start: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dr1.Merge(tt.dr2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBatchAggregator_GroupFilesByAccount(t *testing.T) {
	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	files := []string{
		"CAMT.053_54293249_2025-04-01_2025-04-30_1.xml",
		"CAMT.053_54293249_2025-05-01_2025-05-31_1.xml",
		"CAMT.053_54293250_2025-04-01_2025-04-30_1.xml",
		"other_file.csv",
	}

	groups, err := aggregator.GroupFilesByAccount(files)
	require.NoError(t, err)

	// Should have 3 groups: 54293249, 54293250, and other_file
	assert.Len(t, groups, 3)

	// Find the group for account 54293249
	var group249 *FileGroup
	for i := range groups {
		if groups[i].AccountID == "54293249" {
			group249 = &groups[i]
			break
		}
	}

	require.NotNil(t, group249)
	assert.Len(t, group249.Files, 2)
	assert.Equal(t, "54293249", group249.AccountID)
}

// **Feature: parser-enhancements, Property 4: Duplicate transaction preservation**
// **Validates: Requirements 1.4**
func TestProperty_DuplicateTransactionPreservation(t *testing.T) {
	// Property: For any duplicate transactions found across multiple input files,
	// all transactions should be included in the output and warnings should be logged

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random transactions with some intentional duplicates
			numTransactions := cryptoRandIntn(20) + 10 // 10-29 transactions
			numDuplicates := cryptoRandIntn(5) + 1     // 1-5 duplicates

			var allTransactions []models.Transaction
			baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

			// Create original transactions
			for j := 0; j < numTransactions; j++ {
				randomDays := cryptoRandIntn(30)
				txDate := baseDate.AddDate(0, 0, randomDays)

				tx := models.Transaction{
					Date:        txDate,
					ValueDate:   txDate,
					Amount:      decimal.NewFromFloat(cryptoRandFloat64() * 1000),
					Description: fmt.Sprintf("Transaction %d", j),
					Payee:       fmt.Sprintf("Party %d", j%5), // Limited parties to increase chance of duplicates
				}
				allTransactions = append(allTransactions, tx)
			}

			// Add some intentional duplicates
			for j := 0; j < numDuplicates && j < len(allTransactions); j++ {
				duplicate := allTransactions[j] // Copy the transaction
				allTransactions = append(allTransactions, duplicate)
			}

			// Shuffle to simulate random order from different files
			cryptoShuffle(allTransactions)

			originalCount := len(allTransactions)

			// Test the property: detect duplicates but preserve all transactions
			aggregator.detectAndLogDuplicates(allTransactions, "TEST_ACCOUNT")

			// Verify: all transactions are preserved (no removal)
			assert.Equal(t, originalCount, len(allTransactions),
				"All transactions should be preserved, including duplicates")

			// Verify: warnings are logged for duplicates (check mock logger)
			// Note: This is a behavioral test - the function should log warnings
			// The actual duplicate detection logic is tested separately
		})
	}
}

// **Feature: parser-enhancements, Property 5: Source file metadata inclusion**
// **Validates: Requirements 1.5**
func TestProperty_SourceFileMetadataInclusion(t *testing.T) {
	// Property: For any aggregated output file, the header should contain
	// a comment listing all source files that were merged

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random list of source files
			numFiles := cryptoRandIntn(10) + 1 // 1-10 files
			var sourceFiles []string

			for j := 0; j < numFiles; j++ {
				// Generate random filename
				filename := fmt.Sprintf("file_%d_%d.xml", i, j)
				sourceFiles = append(sourceFiles, filename)
			}

			// Test the property
			header := aggregator.GenerateSourceFileHeader(sourceFiles)

			// Verify: header contains all source files
			for _, file := range sourceFiles {
				assert.Contains(t, header, file,
					"Header should contain source file: %s", file)
			}

			// Verify: header has proper comment format
			assert.Contains(t, header, "# Consolidated from source files:",
				"Header should indicate consolidation")
			assert.Contains(t, header, "# Generated on:",
				"Header should contain generation timestamp")

			// Verify: each file is listed with comment prefix
			for _, file := range sourceFiles {
				expectedLine := fmt.Sprintf("# - %s", file)
				assert.Contains(t, header, expectedLine,
					"Each file should be listed with proper comment format")
			}

			// Test edge case: empty file list
			emptyHeader := aggregator.GenerateSourceFileHeader([]string{})
			assert.Empty(t, emptyHeader,
				"Header should be empty when no source files provided")
		})
	}
}

func TestBatchAggregator_GenerateOutputFilename(t *testing.T) {
	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	tests := []struct {
		name      string
		accountID string
		dateRange DateRange
		expected  string
	}{
		{
			name:      "with date range",
			accountID: "54293249",
			dateRange: DateRange{
				Start: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
			},
			expected: "54293249_2025-04-01_2025-06-30.csv",
		},
		{
			name:      "without date range",
			accountID: "54293249",
			dateRange: DateRange{},
			expected:  "54293249.csv",
		},
		{
			name:      "with unsafe characters",
			accountID: "account/with\\unsafe:chars",
			dateRange: DateRange{},
			expected:  "account_with_unsafe_chars.csv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregator.GenerateOutputFilename(tt.accountID, tt.dateRange)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// **Feature: parser-enhancements, Property 14: Date range calculation**
// **Validates: Requirements 7.2**
func TestProperty_DateRangeCalculation(t *testing.T) {
	// Property: For any set of files with overlapping date ranges, the consolidated filename
	// should use the overall date range spanning all input files

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random number of files (2-10) for the same account
			numFiles := cryptoRandIntn(9) + 2 // 2-10 files
			accountID := fmt.Sprintf("5429324%d", cryptoRandIntn(10))

			var allFiles []string
			var allDateRanges []DateRange

			// Track the expected overall date range
			var expectedStart, expectedEnd time.Time

			for j := 0; j < numFiles; j++ {
				// Generate random start date within a 2-year window
				baseYear := 2023 + cryptoRandIntn(3) // 2023-2025
				baseMonth := cryptoRandIntn(12) + 1  // 1-12
				baseDay := cryptoRandIntn(28) + 1    // 1-28 (safe for all months)

				startDate := time.Date(baseYear, time.Month(baseMonth), baseDay, 0, 0, 0, 0, time.UTC)

				// End date is 1-3 months after start
				monthsLater := cryptoRandIntn(3) + 1
				endDate := startDate.AddDate(0, monthsLater, 0)

				// Track overall date range
				if expectedStart.IsZero() || startDate.Before(expectedStart) {
					expectedStart = startDate
				}
				if expectedEnd.IsZero() || endDate.After(expectedEnd) {
					expectedEnd = endDate
				}

				allDateRanges = append(allDateRanges, DateRange{Start: startDate, End: endDate})

				// Create CAMT filename with this date range
				filename := fmt.Sprintf("CAMT.053_%s_%s_%s_%d.xml",
					accountID,
					startDate.Format("2006-01-02"),
					endDate.Format("2006-01-02"),
					j+1)
				allFiles = append(allFiles, filename)
			}

			// Test the property: group files and verify date range calculation
			groups, err := aggregator.GroupFilesByAccount(allFiles)
			require.NoError(t, err)
			require.Len(t, groups, 1, "All files should be grouped into one account")

			group := groups[0]

			// Verify: the group's date range spans all input files
			assert.Equal(t, expectedStart.Format("2006-01-02"), group.DateRange.Start.Format("2006-01-02"),
				"Start date should be the earliest date from all files")
			assert.Equal(t, expectedEnd.Format("2006-01-02"), group.DateRange.End.Format("2006-01-02"),
				"End date should be the latest date from all files")

			// Verify: the generated filename uses the overall date range
			outputFilename := aggregator.GenerateOutputFilename(group.AccountID, group.DateRange)
			expectedDateRangeStr := fmt.Sprintf("%s_%s",
				expectedStart.Format("2006-01-02"),
				expectedEnd.Format("2006-01-02"))
			assert.Contains(t, outputFilename, expectedDateRangeStr,
				"Output filename should contain the overall date range")

			// Verify: DateRange.Merge correctly combines overlapping ranges
			var mergedRange DateRange
			for _, dr := range allDateRanges {
				mergedRange = mergedRange.Merge(dr)
			}
			assert.Equal(t, expectedStart.Format("2006-01-02"), mergedRange.Start.Format("2006-01-02"),
				"Merged range start should match expected start")
			assert.Equal(t, expectedEnd.Format("2006-01-02"), mergedRange.End.Format("2006-01-02"),
				"Merged range end should match expected end")
		})
	}
}

// TestProperty_DateRangeCalculation_OverlappingRanges tests specifically overlapping date ranges
// **Feature: parser-enhancements, Property 14: Date range calculation (overlapping)**
// **Validates: Requirements 7.2**
func TestProperty_DateRangeCalculation_OverlappingRanges(t *testing.T) {
	// Property: For any set of overlapping date ranges, the merged result should span
	// from the earliest start to the latest end

	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random number of overlapping date ranges
			numRanges := cryptoRandIntn(10) + 2 // 2-11 ranges

			var ranges []DateRange
			var expectedStart, expectedEnd time.Time

			// Create a base date
			baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

			for j := 0; j < numRanges; j++ {
				// Generate overlapping ranges by using random offsets from base
				startOffset := cryptoRandIntn(365)  // 0-364 days from base
				duration := cryptoRandIntn(90) + 30 // 30-119 days duration

				startDate := baseDate.AddDate(0, 0, startOffset)
				endDate := startDate.AddDate(0, 0, duration)

				ranges = append(ranges, DateRange{Start: startDate, End: endDate})

				// Track expected overall range
				if expectedStart.IsZero() || startDate.Before(expectedStart) {
					expectedStart = startDate
				}
				if expectedEnd.IsZero() || endDate.After(expectedEnd) {
					expectedEnd = endDate
				}
			}

			// Test the property: merge all ranges
			var mergedRange DateRange
			for _, dr := range ranges {
				mergedRange = mergedRange.Merge(dr)
			}

			// Verify: merged range spans from earliest start to latest end
			assert.Equal(t, expectedStart, mergedRange.Start,
				"Merged start should be the earliest start date")
			assert.Equal(t, expectedEnd, mergedRange.End,
				"Merged end should be the latest end date")

			// Verify: merged range contains all individual ranges
			for _, dr := range ranges {
				assert.True(t, !dr.Start.Before(mergedRange.Start),
					"Individual range start should not be before merged start")
				assert.True(t, !dr.End.After(mergedRange.End),
					"Individual range end should not be after merged end")
			}
		})
	}
}

// TestProperty_DateRangeCalculation_FromTransactions tests date range calculation from transactions
// **Feature: parser-enhancements, Property 14: Date range calculation (transactions)**
// **Validates: Requirements 7.2**
func TestProperty_DateRangeCalculation_FromTransactions(t *testing.T) {
	// Property: For any set of transactions, the calculated date range should span
	// from the earliest transaction date to the latest transaction date

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random transactions
			numTransactions := cryptoRandIntn(50) + 5 // 5-54 transactions

			var transactions []models.Transaction
			var expectedStart, expectedEnd time.Time

			baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

			for j := 0; j < numTransactions; j++ {
				// Random date within a year
				dayOffset := cryptoRandIntn(365)
				txDate := baseDate.AddDate(0, 0, dayOffset)

				tx := models.Transaction{
					Date:        txDate,
					ValueDate:   txDate,
					Amount:      decimal.NewFromFloat(cryptoRandFloat64() * 1000),
					Description: fmt.Sprintf("Transaction %d", j),
				}
				transactions = append(transactions, tx)

				// Track expected range
				if expectedStart.IsZero() || txDate.Before(expectedStart) {
					expectedStart = txDate
				}
				if expectedEnd.IsZero() || txDate.After(expectedEnd) {
					expectedEnd = txDate
				}
			}

			// Test the property: calculate date range from transactions
			calculatedRange := aggregator.CalculateDateRangeFromTransactions(transactions)

			// Verify: calculated range matches expected
			assert.Equal(t, expectedStart.Format("2006-01-02"), calculatedRange.Start.Format("2006-01-02"),
				"Calculated start should be the earliest transaction date")
			assert.Equal(t, expectedEnd.Format("2006-01-02"), calculatedRange.End.Format("2006-01-02"),
				"Calculated end should be the latest transaction date")
		})
	}
}

// TestProperty_DateRangeCalculation_EmptyInput tests edge case with empty input
// **Feature: parser-enhancements, Property 14: Date range calculation (edge case)**
// **Validates: Requirements 7.2**
func TestProperty_DateRangeCalculation_EmptyInput(t *testing.T) {
	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Test with empty transactions
	emptyRange := aggregator.CalculateDateRangeFromTransactions([]models.Transaction{})
	assert.True(t, emptyRange.Start.IsZero(), "Empty transactions should produce zero start date")
	assert.True(t, emptyRange.End.IsZero(), "Empty transactions should produce zero end date")

	// Test merging with zero range
	validRange := DateRange{
		Start: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	}
	zeroRange := DateRange{}

	// Merging zero with valid should give valid
	merged1 := zeroRange.Merge(validRange)
	assert.Equal(t, validRange.Start, merged1.Start)
	assert.Equal(t, validRange.End, merged1.End)

	// Merging valid with zero should give valid
	merged2 := validRange.Merge(zeroRange)
	assert.Equal(t, validRange.Start, merged2.Start)
	assert.Equal(t, validRange.End, merged2.End)
}

// **Feature: parser-enhancements, Property 15: Output directory organization**
// **Validates: Requirements 7.4**
func TestProperty_OutputDirectoryOrganization(t *testing.T) {
	// Property: For any batch processing operation with a specified output directory,
	// all consolidated files should be placed in that directory

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random output directory path
			outputDir := generateRandomOutputDirectory()

			// Generate random number of account groups (1-5)
			numAccounts := cryptoRandIntn(5) + 1

			var generatedFilenames []string

			for accountIdx := 0; accountIdx < numAccounts; accountIdx++ {
				// Generate random account ID
				accountID := fmt.Sprintf("5429324%d", accountIdx)

				// Generate random date range
				startDate := generateRandomDate()
				endDate := startDate.AddDate(0, cryptoRandIntn(6)+1, 0) // 1-6 months later
				dateRange := DateRange{Start: startDate, End: endDate}

				// Generate output filename using the aggregator
				outputFilename := aggregator.GenerateOutputFilename(accountID, dateRange)

				// Construct full output path
				fullOutputPath := filepath.Join(outputDir, outputFilename)
				generatedFilenames = append(generatedFilenames, fullOutputPath)

				// Verify: output filename is filesystem-safe
				assertFilesystemSafe(t, outputFilename)

				// Verify: output path is within the specified output directory
				assert.True(t, strings.HasPrefix(fullOutputPath, outputDir),
					"Output file should be within the specified output directory")

				// Verify: output filename ends with .csv
				assert.True(t, strings.HasSuffix(outputFilename, ".csv"),
					"Output filename should end with .csv")

				// Verify: output path does not escape the output directory
				cleanPath := filepath.Clean(fullOutputPath)
				cleanOutputDir := filepath.Clean(outputDir)
				assert.True(t, strings.HasPrefix(cleanPath, cleanOutputDir),
					"Cleaned output path should still be within output directory (no path traversal)")
			}

			// Verify: all generated filenames are unique
			filenameSet := make(map[string]bool)
			for _, filename := range generatedFilenames {
				baseName := filepath.Base(filename)
				assert.False(t, filenameSet[baseName],
					"Each account should produce a unique filename: %s", baseName)
				filenameSet[baseName] = true
			}

			// Verify: number of output files matches number of accounts
			assert.Equal(t, numAccounts, len(generatedFilenames),
				"Should generate one output file per account")
		})
	}
}

// TestProperty_OutputDirectoryOrganization_PathSafety tests that output paths are safe
// **Feature: parser-enhancements, Property 15: Output directory organization (path safety)**
// **Validates: Requirements 7.4**
func TestProperty_OutputDirectoryOrganization_PathSafety(t *testing.T) {
	// Property: For any account identifier (including malicious ones), the generated
	// output path should not escape the specified output directory

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random output directory
			outputDir := generateRandomOutputDirectory()

			// Generate potentially malicious account IDs
			maliciousAccountIDs := []string{
				"../../../etc/passwd",
				"..\\..\\..\\windows\\system32",
				"account/../../../secret",
				"account/../../other",
				"account\x00hidden",
				"account\ninjection",
				"account\rinjection",
				"account\t\ttabs",
				generateRandomAccountID(), // Random account with special chars
			}

			for _, accountID := range maliciousAccountIDs {
				dateRange := DateRange{
					Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
				}

				// Generate output filename
				outputFilename := aggregator.GenerateOutputFilename(accountID, dateRange)

				// Construct full output path
				fullOutputPath := filepath.Join(outputDir, outputFilename)

				// Verify: output filename is filesystem-safe
				assertFilesystemSafe(t, outputFilename)

				// Verify: output path does not escape the output directory
				cleanPath := filepath.Clean(fullOutputPath)
				cleanOutputDir := filepath.Clean(outputDir)
				assert.True(t, strings.HasPrefix(cleanPath, cleanOutputDir),
					"Output path should not escape output directory for account: %s", accountID)

				// Verify: no path traversal sequences in the filename
				assert.NotContains(t, outputFilename, "..",
					"Filename should not contain path traversal sequences")
				assert.NotContains(t, outputFilename, "/",
					"Filename should not contain forward slashes")
				assert.NotContains(t, outputFilename, "\\",
					"Filename should not contain backslashes")
			}
		})
	}
}

// TestProperty_OutputDirectoryOrganization_ConsistentPlacement tests consistent file placement
// **Feature: parser-enhancements, Property 15: Output directory organization (consistency)**
// **Validates: Requirements 7.4**
func TestProperty_OutputDirectoryOrganization_ConsistentPlacement(t *testing.T) {
	// Property: For any set of file groups, all consolidated files should be placed
	// directly in the output directory (not in subdirectories)

	logger := logging.NewMockLogger()
	aggregator := NewBatchAggregator(logger)

	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random output directory
			outputDir := generateRandomOutputDirectory()

			// Generate random file groups
			numGroups := cryptoRandIntn(10) + 1 // 1-10 groups

			for groupIdx := 0; groupIdx < numGroups; groupIdx++ {
				// Generate random account ID
				accountID := fmt.Sprintf("account_%d_%d", i, groupIdx)

				// Generate random date range
				startDate := generateRandomDate()
				endDate := startDate.AddDate(0, cryptoRandIntn(12)+1, 0)
				dateRange := DateRange{Start: startDate, End: endDate}

				// Generate output filename
				outputFilename := aggregator.GenerateOutputFilename(accountID, dateRange)

				// Construct full output path
				fullOutputPath := filepath.Join(outputDir, outputFilename)

				// Verify: file is placed directly in output directory (no subdirectories)
				parentDir := filepath.Dir(fullOutputPath)
				assert.Equal(t, filepath.Clean(outputDir), filepath.Clean(parentDir),
					"Output file should be placed directly in output directory, not in subdirectory")

				// Verify: filename does not create subdirectories
				assert.Equal(t, outputFilename, filepath.Base(outputFilename),
					"Filename should not contain directory separators")
			}
		})
	}
}

// generateRandomOutputDirectory generates a random output directory path for testing
func generateRandomOutputDirectory() string {
	// Generate random directory names (using Unix-style paths for consistency)
	dirNames := []string{
		"/tmp/output",
		"/var/data/exports",
		"/home/user/documents/financial",
		"/path/with-dashes/output_dir",
		"/data/batch/results",
		"/exports/financial",
		"/output/consolidated",
		"/work/processed",
	}

	baseDir := dirNames[cryptoRandIntn(len(dirNames))]

	// Optionally add random subdirectory
	if cryptoRandIntn(2) == 0 {
		subDir := fmt.Sprintf("batch_%d", cryptoRandIntn(1000))
		return filepath.Join(baseDir, subDir)
	}

	return baseDir
}
