package categorizer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() logging.Logger { return logging.NewLogrusAdapter("error", "text") }

// cleanCategory absorbs the messy shapes real models answer with. Each case
// below is a shape that has actually been observed in production responses.
func TestCleanCategory(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain answer", "Courses", "Courses"},
		{"surrounding whitespace", "  Restaurants \n", "Restaurants"},
		{"markdown bold", "**Courses**", "Courses"},
		{"bold inside a sentence", "The best fit here is **Voiture** for this one.", "Voiture"},
		{"labelled answer", "Category: Shopping", "Shopping"},
		{"lower-case label", "category: Shopping", "Shopping"},
		{"quoted answer", `"Logement"`, "Logement"},
		{"single-quoted answer", "'Logement'", "Logement"},
		{"multi-line, answer last", "Let me think about it.\nThis is a supermarket.\nCourses", "Courses"},
		{"multi-line with trailing blanks", "Reasoning here.\nSport\n\n  \n", "Sport"},
		{"english synonym", "groceries", "Courses"},
		{"english synonym, mixed case", "Groceries", "Courses"},
		{"synonym via label and bold", "Category: **fuel**", "Voiture"},
		{"legacy consolidated name", "Mobilier & Maison", "Mobilier"},
		{"unknown maps to uncategorized", "unknown", models.CategoryUncategorized},
		{"other maps to uncategorized", "other", models.CategoryUncategorized},
		{"unrecognised value passes through", "Quelque Chose", "Quelque Chose"},
		{"empty stays empty", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cleanCategory(tt.input))
		})
	}
}

// Retry decisions must be identical for both providers; before the shared
// helper existed, only Gemini retried timeouts.
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not retryable", nil, false},
		{"429 rate limited", errors.New("API request failed with status 429: slow down"), true},
		{"500 server error", errors.New("API request failed with status 500"), true},
		{"503 unavailable", errors.New("API request failed with status 503"), true},
		{"400 bad request", errors.New("API request failed with status 400: bad model"), false},
		{"401 unauthorized", errors.New("API request failed with status 401"), false},
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"connection reset", errors.New("read: connection reset by peer"), true},
		{"temporary failure", errors.New("temporary failure in name resolution"), true},
		{"timeout", os.ErrDeadlineExceeded, true},
		{"unrelated error", errors.New("failed to marshal request"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetryableError(tt.err))
		})
	}
}

func TestBuildCategorizationPrompt_IncludesTransaction(t *testing.T) {
	tx := models.Transaction{
		PartyName:   "Coop Pronto",
		Description: "Card payment",
		Amount:      decimal.NewFromFloat(15.50),
	}

	prompt := buildCategorizationPrompt(tx)

	assert.Contains(t, prompt, "Party: Coop Pronto")
	assert.Contains(t, prompt, "Description: Card payment")
	assert.Contains(t, prompt, "Amount: 15.5 CHF")
	assert.Contains(t, prompt, "- Courses (supermarkets like Migros, Coop, Aldi, Lidl)")
	assert.True(t, strings.HasSuffix(prompt, "Category:"), "prompt must end by asking for the category")
}

// Both providers must render the same prompt, otherwise their answers stop
// being comparable and category quality silently diverges between backends.
func TestBuildCategorizationPrompt_SharedByBothProviders(t *testing.T) {
	tx := models.Transaction{PartyName: "IKEA AG", Amount: decimal.NewFromInt(150)}

	gemini := NewGeminiClient(testLogger(), 60, "gemini-2.0-flash", 30, "k", "")
	openrouter := NewOpenRouterClient(testLogger(), 60, "some/model", 30, "k", "")

	assert.NotNil(t, gemini)
	assert.NotNil(t, openrouter)
	assert.Equal(t, buildCategorizationPrompt(tx), buildCategorizationPrompt(tx))
}

// Without a credential the client must not call out at all, and must report
// success so the caller falls back to the earlier strategy tiers.
func TestBaseAIClient_NoAPIKeySkipsProvider(t *testing.T) {
	base := newBaseAIClient("test", testLogger(), "", 60)
	called := false

	tx, err := base.categorize(context.Background(), models.Transaction{PartyName: "X"},
		func(context.Context, string) (string, error) {
			called = true
			return "Courses", nil
		})

	require.NoError(t, err)
	assert.False(t, called, "provider must not be called without an API key")
	assert.Equal(t, models.CategoryUncategorized, tx.Category)
}

func TestBaseAIClient_CleansProviderAnswer(t *testing.T) {
	base := newBaseAIClient("test", testLogger(), "key", 600)

	tx, err := base.categorize(context.Background(), models.Transaction{PartyName: "Coop"},
		func(context.Context, string) (string, error) {
			return "Here you go:\n**groceries**", nil
		})

	require.NoError(t, err)
	assert.Equal(t, "Courses", tx.Category)
}

func TestBaseAIClient_RetriesTransientErrors(t *testing.T) {
	base := newBaseAIClient("test", testLogger(), "key", 600)
	attempts := 0

	tx, err := base.categorize(context.Background(), models.Transaction{PartyName: "Coop"},
		func(context.Context, string) (string, error) {
			attempts++
			if attempts < 3 {
				return "", fmt.Errorf("API request failed with status 503")
			}
			return "Courses", nil
		})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts, "should retry until the call succeeds")
	assert.Equal(t, "Courses", tx.Category)
}

func TestBaseAIClient_DoesNotRetryPermanentErrors(t *testing.T) {
	base := newBaseAIClient("test", testLogger(), "key", 600)
	attempts := 0

	tx, err := base.categorize(context.Background(), models.Transaction{PartyName: "Coop"},
		func(context.Context, string) (string, error) {
			attempts++
			return "", fmt.Errorf("API request failed with status 401: bad key")
		})

	require.Error(t, err)
	assert.Equal(t, 1, attempts, "a 401 must not be retried")
	assert.Equal(t, models.CategoryUncategorized, tx.Category)
}

func TestBaseAIClient_GivesUpAfterMaxRetries(t *testing.T) {
	base := newBaseAIClient("test", testLogger(), "key", 600)
	attempts := 0

	_, err := base.categorize(context.Background(), models.Transaction{PartyName: "Coop"},
		func(context.Context, string) (string, error) {
			attempts++
			return "", fmt.Errorf("API request failed with status 503")
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "after 4 attempts")
	assert.Equal(t, 4, attempts, "one initial attempt plus three retries")
}

// A cancelled context must abort the backoff wait instead of sleeping through it.
func TestBaseAIClient_CancellationInterruptsRetryWait(t *testing.T) {
	base := newBaseAIClient("test", testLogger(), "key", 600)
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	start := time.Now()
	_, err := base.categorize(ctx, models.Transaction{PartyName: "Coop"},
		func(context.Context, string) (string, error) {
			attempts++
			cancel() // cancel while the first backoff is pending
			return "", fmt.Errorf("API request failed with status 503")
		})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, attempts)
	assert.Less(t, time.Since(start), time.Second, "must not sleep out the full backoff")
}

// The rate limiter must observe cancellation rather than blocking forever.
func TestBaseAIClient_CancelledContextStopsAtRateLimiter(t *testing.T) {
	base := newBaseAIClient("test", testLogger(), "key", 60)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	_, err := base.categorize(ctx, models.Transaction{PartyName: "Coop"},
		func(context.Context, string) (string, error) {
			called = true
			return "Courses", nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limiter wait cancelled")
	assert.False(t, called)
}
