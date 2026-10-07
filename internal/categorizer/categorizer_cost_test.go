package categorizer

import (
	"context"
	"strings"
	"sync"
	"testing"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingAI answers from a fixed map (lowercased party -> category) and counts calls.
type countingAI struct {
	mu      sync.Mutex
	answers map[string]string
	calls   int
}

func (c *countingAI) Categorize(_ context.Context, tx models.Transaction) (models.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	tx.Category = c.answers[strings.ToLower(strings.TrimSpace(tx.PartyName))]
	if tx.Category == "" {
		tx.Category = models.CategoryUncategorized
	}
	return tx, nil
}

func (c *countingAI) GetEmbedding(context.Context, string) ([]float32, error) { return nil, nil }

func (c *countingAI) callCount() int { c.mu.Lock(); defer c.mu.Unlock(); return c.calls }

func newCostCategorizer(t *testing.T, ai *countingAI, autoLearn bool) (*Categorizer, *store.MockCategoryStore) {
	t.Helper()
	st := &store.MockCategoryStore{
		Categories: []models.CategoryConfig{
			{Name: "Courses", Keywords: []string{"MIGROS"}},
			{Name: "Abonnements"},
			{Name: "Restaurants"},
		},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}
	c := NewCategorizer(ai, nil, st, testLogger(), autoLearn, 0.70)
	t.Cleanup(c.Shutdown)
	return c, st
}

func TestCategorize_UncategorizedIsCachedForTheRun(t *testing.T) {
	ai := &countingAI{answers: map[string]string{}}
	c, _ := newCostCategorizer(t, ai, false)

	for i := 0; i < 3; i++ {
		got, err := c.Categorize(context.Background(), "Mystery Shop", true, "-10", "2026-01-01", "")
		require.NoError(t, err)
		assert.Equal(t, models.CategoryUncategorized, got.Name)
	}
	assert.Equal(t, 1, ai.callCount(), "an unknown party reaches the AI once per run")
}
