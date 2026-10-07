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

// countingStore wraps MockCategoryStore and counts saves.
type countingStore struct {
	*store.MockCategoryStore
	creditorSaves, debtorSaves int
}

func (s *countingStore) SaveCreditorMappings(m map[string]string) error {
	s.creditorSaves++
	return s.MockCategoryStore.SaveCreditorMappings(m)
}

func (s *countingStore) SaveDebtorMappings(m map[string]string) error {
	s.debtorSaves++
	return s.MockCategoryStore.SaveDebtorMappings(m)
}

func TestCategorize_AutoLearnOnlyFromAIAndNoSavePerTransaction(t *testing.T) {
	ai := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	st := &countingStore{MockCategoryStore: &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Courses", Keywords: []string{"MIGROS"}}, {Name: "Abonnements"}},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{"coop": "Courses"},
	}}
	c := NewCategorizer(ai, nil, st, testLogger(), true, 0.70)
	t.Cleanup(c.Shutdown)
	ctx := context.Background()

	_, _ = c.Categorize(ctx, "Coop", true, "-5", "2026-01-01", "")            // direct
	_, _ = c.Categorize(ctx, "MIGROS Lausanne", true, "-5", "2026-01-01", "") // keyword
	_, _ = c.Categorize(ctx, "Kiro", true, "-19", "2026-01-01", "")           // ai

	assert.Zero(t, st.debtorSaves, "nothing is written during the run")
	require.NoError(t, c.SaveDebitorsToYAML())
	assert.Equal(t, 1, st.debtorSaves, "one write at the end")

	saved := st.MockCategoryStore.DebtorMappings
	assert.Equal(t, "Abonnements", saved["kiro"], "AI answers are learned")
	assert.NotContains(t, saved, "migros lausanne", "keyword hits are not learned")
}

func TestCategorize_AIAnswerMustBeAKnownCategory(t *testing.T) {
	ai := &countingAI{answers: map[string]string{
		"evil":  `=HYPERLINK("http://x")`,
		"lunch": "restaurants", // wrong case, still known
	}}
	c, _ := newCostCategorizer(t, ai, false)
	ctx := context.Background()

	got, err := c.Categorize(ctx, "Evil", true, "-1", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, models.CategoryUncategorized, got.Name)

	got, err = c.Categorize(ctx, "Lunch", true, "-1", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Restaurants", got.Name, "canonical spelling from categories.yaml")
}

func TestCategorize_NoCategoriesLoadedSkipsValidation(t *testing.T) {
	ai := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	st := &store.MockCategoryStore{CreditorMappings: map[string]string{}, DebtorMappings: map[string]string{}}
	c := NewCategorizer(ai, nil, st, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	got, err := c.Categorize(context.Background(), "Kiro", true, "-1", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got.Name, "an empty categories.yaml must not reject every answer")
}
