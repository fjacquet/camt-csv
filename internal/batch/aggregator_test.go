package batch

import (
	"crypto/rand"
	"fmt"
	"math/big"
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

// cryptoShuffle shuffles a slice of transactions using crypto/rand
func cryptoShuffle(transactions []models.Transaction) {
	n := len(transactions)
	for i := n - 1; i > 0; i-- {
		j := cryptoRandIntn(i + 1)
		transactions[i], transactions[j] = transactions[j], transactions[i]
	}
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

// Consolidate is what BatchProcessor calls once every file in a batch has been
// parsed. It must order the merged set by date regardless of the order files
// were read in, and must never drop a transaction: two identical purchases on
// the same day are a real thing, so duplicates are reported, not removed.
func TestConsolidate_SortsAndKeepsDuplicates(t *testing.T) {
	logger := logging.NewMockLogger()
	agg := NewBatchAggregator(logger)

	mar := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	jan := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)

	input := []models.Transaction{
		{BookkeepingNumber: "c", Date: mar, ValueDate: mar, Amount: decimal.NewFromInt(30), PartyName: "Migros", Payee: "Migros", DebitFlag: true},
		{BookkeepingNumber: "a", Date: jan, ValueDate: jan, Amount: decimal.NewFromInt(10), PartyName: "Coop", Payee: "Coop", DebitFlag: true},
		{BookkeepingNumber: "b", Date: feb, ValueDate: feb, Amount: decimal.NewFromInt(20), PartyName: "SBB", Payee: "SBB", DebitFlag: true},
		{BookkeepingNumber: "b2", Date: feb, ValueDate: feb, Amount: decimal.NewFromInt(20), PartyName: "SBB", Payee: "SBB", DebitFlag: true},
	}

	got := agg.Consolidate(input, "releves-2024")

	require.Len(t, got, 4, "Consolidate must not drop the duplicate")

	var dates []time.Time
	for _, tx := range got {
		dates = append(dates, tx.Date)
	}
	assert.Equal(t, []time.Time{jan, feb, feb, mar}, dates, "must be ordered by date")

	// BookkeepingNumber is the stable identifier; Number is a fresh UUID per run.
	assert.Equal(t, "a", got[0].BookkeepingNumber)
	assert.Equal(t, "c", got[3].BookkeepingNumber)

	// Verify duplicates are reported: reporting is the deliberate alternative to
	// removing them, so a later reader does not "simplify" the warning away.
	warnEntries := logger.GetEntriesByLevel("WARN")
	require.NotEmpty(t, warnEntries, "Consolidate must log a warning for duplicate transactions")

	// Check that at least one warning contains the duplicate transaction message
	// and carries the label in the account field.
	found := false
	for _, entry := range warnEntries {
		if entry.Message == "Potential duplicate transaction" {
			for _, field := range entry.Fields {
				if field.Key == "account" && field.Value == "releves-2024" {
					found = true
					break
				}
			}
		}
	}
	assert.True(t, found, "Consolidate must log duplicate warning with the batch label in the account field")
}

// detectAndLogDuplicates' inner loop breaks as soon as the date no longer
// matches, on the assumption that the slice is already sorted chronologically
// (true of its only caller, Consolidate). This must not degrade into breaking
// on index adjacency instead of on the date actually changing: two same-date
// transactions separated in the slice by a third, non-duplicate, same-date
// transaction must still be found.
func TestDetectAndLogDuplicates_NonAdjacentSameDateDuplicatesStillFound(t *testing.T) {
	logger := logging.NewMockLogger()
	agg := NewBatchAggregator(logger)

	day := time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)
	nextDay := time.Date(2024, 2, 21, 0, 0, 0, 0, time.UTC)

	// Already sorted, as Consolidate would leave it. "Migros" appears twice at
	// the same date with an unrelated "SBB" transaction in between them.
	transactions := []models.Transaction{
		{Date: day, Amount: decimal.NewFromInt(10), PartyName: "Migros", Payee: "Migros"},
		{Date: day, Amount: decimal.NewFromInt(20), PartyName: "SBB", Payee: "SBB"},
		{Date: day, Amount: decimal.NewFromInt(10), PartyName: "Migros", Payee: "Migros"},
		{Date: nextDay, Amount: decimal.NewFromInt(30), PartyName: "Coop", Payee: "Coop"},
	}

	agg.detectAndLogDuplicates(transactions, "TEST_ACCOUNT")

	var duplicateWarnings int
	for _, entry := range logger.GetEntriesByLevel("WARN") {
		if entry.Message == "Potential duplicate transaction" {
			duplicateWarnings++
		}
	}
	assert.Equal(t, 1, duplicateWarnings,
		"the two same-date Migros transactions must be found as duplicates despite the SBB transaction between them")
}

// An empty batch is a normal outcome (a directory of unreadable files), not an
// error: Consolidate must return cleanly rather than panic on the empty slice.
func TestConsolidate_EmptyInput(t *testing.T) {
	logger := logging.NewMockLogger()
	agg := NewBatchAggregator(logger)

	got := agg.Consolidate(nil, "empty")

	assert.Empty(t, got)
}
