package categorizer

import (
	"context"
	"sync"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingEmbedder records every embedding request and can block, so tests can
// observe a warm-up while it is still in flight.
type countingEmbedder struct {
	mu      sync.Mutex
	calls   int
	release chan struct{} // when non-nil, each call waits on it
}

func (c *countingEmbedder) Categorize(_ context.Context, tx models.Transaction) (models.Transaction, error) {
	return tx, nil
}

func (c *countingEmbedder) GetEmbedding(ctx context.Context, _ string) ([]float32, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()

	if c.release != nil {
		select {
		case <-c.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return []float32{1, 0, 0}, nil
}

func (c *countingEmbedder) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// newLifecycleStore returns a store with a few categories so the warm-up has
// something to do.
func newLifecycleStore() *store.MockCategoryStore {
	return &store.MockCategoryStore{
		Categories:       manyCategories(5),
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}
}

func manyCategories(n int) []models.CategoryConfig {
	cats := make([]models.CategoryConfig, n)
	for i := range cats {
		cats[i] = models.CategoryConfig{Name: "cat", Keywords: []string{"keyword"}}
	}
	return cats
}

// A client that cannot produce embeddings must not be handed to the semantic
// tier: it would fail on every category, then mark itself initialized with an
// empty map, silently disabling tier 3 while the logs claim it is active.
// Passing nil disables the tier honestly instead.
func TestNewCategorizer_NilEmbeddingClientDisablesSemanticTier(t *testing.T) {
	chat := &countingEmbedder{}

	c := NewCategorizer(chat, nil, newLifecycleStore(), testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	var semantic *SemanticStrategy
	for _, s := range c.strategies {
		if sem, ok := s.(*SemanticStrategy); ok {
			semantic = sem
		}
	}
	require.NotNil(t, semantic, "the semantic strategy must still be in the chain")

	assert.Nil(t, semantic.client, "semantic tier must not borrow the chat client")
	assert.Zero(t, chat.callCount(), "no warm-up may run without an embedding client")

	_, found, err := semantic.Categorize(context.Background(), Transaction{PartyName: "Coop"})
	require.NoError(t, err)
	assert.False(t, found, "a disabled semantic tier must decline, not guess")
}

// The chat client and the embedding client are distinct roles. Wiring them at
// construction is what removes the need to swap a client afterwards — the swap
// was a data race against the running warm-up goroutine.
func TestNewCategorizer_UsesDistinctChatAndEmbeddingClients(t *testing.T) {
	// Isolate HOME so a real embedding cache on the developer's machine cannot
	// satisfy the warm-up before it issues a single request.
	t.Setenv("HOME", t.TempDir())

	chat := &countingEmbedder{}
	embedder := &countingEmbedder{}

	c := NewCategorizer(chat, embedder, newLifecycleStore(), testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	require.Eventually(t, func() bool { return embedder.callCount() > 0 },
		2*time.Second, 5*time.Millisecond,
		"the embedding client must warm the semantic tier")

	assert.Zero(t, chat.callCount(), "the chat client must never be asked for embeddings")
}

// Shutdown must stop an in-flight warm-up rather than let it work through every
// remaining category; each one is a rate-limited network round trip.
func TestSemanticStrategy_ShutdownCancelsWarmup(t *testing.T) {
	embedder := &countingEmbedder{release: make(chan struct{})}

	s := NewSemanticStrategyWithCache(embedder, testLogger(), manyCategories(500), 0.70, nil)

	// Let the warm-up reach its first blocking call.
	require.Eventually(t, func() bool { return embedder.callCount() > 0 },
		2*time.Second, 5*time.Millisecond, "warm-up should have started")

	done := make(chan struct{})
	go func() {
		close(embedder.release) // unblock the in-flight call so cancellation can be observed
		s.Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not return: the warm-up is not cancellable")
	}

	assert.Less(t, embedder.callCount(), 500, "warm-up must stop early, not embed every category")
}

// Shutdown is safe when no warm-up ever started, and safe to call twice.
func TestSemanticStrategy_ShutdownIsSafeWithoutWarmup(t *testing.T) {
	s := NewSemanticStrategyWithCache(nil, testLogger(), manyCategories(3), 0.70, nil)

	assert.NotPanics(t, s.Shutdown)
	assert.NotPanics(t, s.Shutdown)
}

func TestCategorizer_ShutdownIsIdempotent(t *testing.T) {
	c := NewCategorizer(nil, &countingEmbedder{}, newLifecycleStore(), testLogger(), false, 0.70)

	assert.NotPanics(t, c.Shutdown)
	assert.NotPanics(t, c.Shutdown)
}
