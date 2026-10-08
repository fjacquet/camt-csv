package categorizer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"fjacquet/camt-csv/internal/logging"
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

// "Non Classé" is a real entry in categories.yaml, so the AI may pick it. It
// still means "no decision": learning it would pin the party to it for good,
// and the AI would never be asked again.
func TestCategorize_UnknownAIAnswerIsNotLearned(t *testing.T) {
	ai := &countingAI{answers: map[string]string{"sdds sarl": "Non Classé", "kiro": "Abonnements"}}
	st := &countingStore{MockCategoryStore: &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Non Classé"}, {Name: "Abonnements"}},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}}
	c := NewCategorizer(ai, nil, st, testLogger(), true, 0.70)
	t.Cleanup(c.Shutdown)
	ctx := context.Background()

	got, err := c.Categorize(ctx, "SDDS Sarl", true, "-5", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Non Classé", got.Name, "the answer is still used for this run")
	_, _ = c.Categorize(ctx, "Kiro", true, "-19", "2026-01-01", "")

	require.NoError(t, c.SaveDebitorsToYAML())
	saved := st.MockCategoryStore.DebtorMappings
	assert.NotContains(t, saved, "sdds sarl", "an unknown category is not learned")
	assert.Equal(t, "Abonnements", saved["kiro"])
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

// memStaging is an in-memory StagingStoreInterface that counts merges.
type memStaging struct {
	creditors, debtors map[string]string
	merges             int
}

func (m *memStaging) LoadSuggestions() (map[string]string, map[string]string, error) {
	return m.creditors, m.debtors, nil
}

func (m *memStaging) MergeSuggestions(cr, db map[string]string) error {
	m.merges++
	for k, v := range cr {
		m.creditors[strings.ToLower(k)] = v
	}
	for k, v := range db {
		m.debtors[strings.ToLower(k)] = v
	}
	return nil
}

func TestCategorize_StagedSuggestionsAnswerTheNextRun(t *testing.T) {
	staging := &memStaging{creditors: map[string]string{}, debtors: map[string]string{}}
	ctx := context.Background()

	// Run 1: the AI answers; the suggestion is staged once, at the end.
	ai1 := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	c1, _ := newCostCategorizer(t, ai1, false)
	c1.SetStagingStore(staging)
	_, _ = c1.Categorize(ctx, "Kiro", true, "-19", "2026-01-01", "")
	_, _ = c1.Categorize(ctx, "MIGROS Lausanne", true, "-5", "2026-01-01", "")
	assert.Zero(t, staging.merges, "nothing written during the run")
	require.NoError(t, c1.FlushStaging())
	assert.Equal(t, 1, staging.merges)
	assert.Equal(t, map[string]string{"kiro": "Abonnements"}, staging.debtors, "only AI answers are staged")

	// Run 2: a fresh categorizer finds Kiro in staging and never calls the AI.
	ai2 := &countingAI{answers: map[string]string{}}
	c2, _ := newCostCategorizer(t, ai2, false)
	c2.SetStagingStore(staging)
	got, err := c2.Categorize(ctx, "KIRO", true, "-19", "2026-02-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got.Name)
	assert.Equal(t, "staged", got.Source)
	assert.Zero(t, ai2.callCount())
}

type failingAI struct {
	mu    sync.Mutex
	calls int
}

func (f *failingAI) Categorize(context.Context, models.Transaction) (models.Transaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return models.Transaction{}, fmt.Errorf("status 429")
}

func (f *failingAI) GetEmbedding(context.Context, string) ([]float32, error) { return nil, nil }

func TestCategorize_AIErrorIsCachedForTheRun(t *testing.T) {
	ai := &failingAI{}
	c := NewCategorizer(ai, nil, &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Courses"}},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	for i := 0; i < 3; i++ {
		got, err := c.Categorize(context.Background(), "Flaky Shop", true, "-10", "2026-01-01", "")
		require.NoError(t, err)
		assert.Equal(t, models.CategoryUncategorized, got.Name)
	}
	assert.Equal(t, 1, ai.calls, "a failing AI is not retried for the same party")
}

func TestCategorize_RejectedAnswerIsCachedAndNeverStaged(t *testing.T) {
	staging := &memStaging{creditors: map[string]string{}, debtors: map[string]string{}}
	ai := &countingAI{answers: map[string]string{"evil": `=HYPERLINK("http://x")`}}
	c, _ := newCostCategorizer(t, ai, false)
	c.SetStagingStore(staging)

	for _, name := range []string{"evil", "Evil"} {
		got, err := c.Categorize(context.Background(), name, true, "-10", "2026-01-01", "")
		require.NoError(t, err)
		assert.Equal(t, models.CategoryUncategorized, got.Name)
	}
	assert.Equal(t, 1, ai.callCount())
	require.NoError(t, c.FlushStaging())
	assert.Zero(t, staging.merges)
	assert.Empty(t, staging.debtors)
	assert.Empty(t, staging.creditors)
}

func TestStaged_PaddedNameRoundTrips(t *testing.T) {
	staging := &memStaging{creditors: map[string]string{}, debtors: map[string]string{}}
	ctx := context.Background()
	c1, _ := newCostCategorizer(t, &countingAI{answers: map[string]string{"kiro": "Abonnements"}}, false)
	c1.SetStagingStore(staging)
	_, _ = c1.Categorize(ctx, " Kiro ", true, "-19", "2026-01-01", "")
	require.NoError(t, c1.FlushStaging())
	assert.Equal(t, map[string]string{"kiro": "Abonnements"}, staging.debtors)

	ai2 := &countingAI{answers: map[string]string{}}
	c2, _ := newCostCategorizer(t, ai2, false)
	c2.SetStagingStore(staging)
	got, err := c2.Categorize(ctx, "kiro", true, "-19", "2026-02-01", "")
	require.NoError(t, err)
	assert.Equal(t, "staged", got.Source)
	assert.Zero(t, ai2.callCount())
}

func TestStaged_LoadedKeysAreNormalized(t *testing.T) {
	staging := &memStaging{creditors: map[string]string{}, debtors: map[string]string{" KIRO ": "abonnements"}}
	ai := &countingAI{answers: map[string]string{}}
	c, _ := newCostCategorizer(t, ai, false)
	c.SetStagingStore(staging)
	got, err := c.Categorize(context.Background(), "Kiro", true, "-19", "2026-02-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got.Name, "canonical spelling")
	assert.Equal(t, "staged", got.Source)
	assert.Zero(t, ai.callCount())
}

func TestStaged_UnknownCategoryIsDropped(t *testing.T) {
	staging := &memStaging{creditors: map[string]string{}, debtors: map[string]string{"kiro": "=x"}}
	ai := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	c, _ := newCostCategorizer(t, ai, false)
	c.SetStagingStore(staging)
	got, err := c.Categorize(context.Background(), "Kiro", true, "-19", "2026-02-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Abonnements", got.Name)
	assert.Equal(t, 1, ai.callCount())
}

func TestStaged_TierOrderAndDirection(t *testing.T) {
	ctx := context.Background()
	staging := &memStaging{creditors: map[string]string{}, debtors: map[string]string{"migros lausanne": "Abonnements", "kiro": "Abonnements"}}
	ai := &countingAI{answers: map[string]string{}}
	c, _ := newCostCategorizer(t, ai, false)
	c.SetStagingStore(staging)

	got, err := c.Categorize(ctx, "MIGROS Lausanne", true, "-5", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, "Courses", got.Name)
	assert.Equal(t, "keyword", got.Source, "keyword outranks staged")

	got, err = c.Categorize(ctx, "Kiro", false, "-5", "2026-01-01", "")
	require.NoError(t, err)
	assert.NotEqual(t, "staged", got.Source, "debtor-only entry does not answer a creditor")

	ai.calls = 0
	got, err = c.Categorize(ctx, "Kiro", true, "-5", "2026-01-01", "")
	require.NoError(t, err)
	assert.Equal(t, "staged", got.Source)
	assert.Zero(t, ai.callCount(), "staged beats the AI")
}

const noCategoriesWarning = "No categories loaded from categories.yaml; AI answers will not be validated"

func TestNewCategorizer_WarnsWhenAnswersCannotBeValidated(t *testing.T) {
	ai := &countingAI{answers: map[string]string{}}

	empty := logging.NewMockLogger()
	c := NewCategorizer(ai, nil, &store.MockCategoryStore{CreditorMappings: map[string]string{}, DebtorMappings: map[string]string{}}, empty, false, 0.70)
	t.Cleanup(c.Shutdown)
	assert.True(t, empty.HasEntry("WARN", noCategoriesWarning))

	full := logging.NewMockLogger()
	c2 := NewCategorizer(ai, nil, &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Courses"}},
		CreditorMappings: map[string]string{}, DebtorMappings: map[string]string{},
	}, full, false, 0.70)
	t.Cleanup(c2.Shutdown)
	assert.False(t, full.HasEntry("WARN", noCategoriesWarning))
}

// flakyStaging fails its first MergeSuggestions call.
type flakyStaging struct {
	memStaging
	failures int
}

func (f *flakyStaging) MergeSuggestions(cr, db map[string]string) error {
	if f.failures > 0 {
		f.failures--
		return fmt.Errorf("disk full")
	}
	return f.memStaging.MergeSuggestions(cr, db)
}

func TestFlushStaging_KeepsBatchOnWriteError(t *testing.T) {
	staging := &flakyStaging{memStaging: memStaging{creditors: map[string]string{}, debtors: map[string]string{}}, failures: 1}
	ai := &countingAI{answers: map[string]string{"kiro": "Abonnements"}}
	c, _ := newCostCategorizer(t, ai, false)
	c.SetStagingStore(staging)
	_, _ = c.Categorize(context.Background(), "Kiro", true, "-19", "2026-01-01", "")

	require.Error(t, c.FlushStaging())
	require.NoError(t, c.FlushStaging())
	assert.Equal(t, map[string]string{"kiro": "Abonnements"}, staging.debtors)
}
