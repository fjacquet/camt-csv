package categorizer

import (
	"context"
	"sync"
	"testing"

	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingChat fails the test if the AI tier is ever asked.
type recordingChat struct {
	mu    sync.Mutex
	calls int
}

func (r *recordingChat) Categorize(_ context.Context, tx models.Transaction) (models.Transaction, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return tx, nil
}

func (r *recordingChat) GetEmbedding(context.Context, string) ([]float32, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return []float32{1, 0, 0}, nil
}

func TestCategorizeLocal(t *testing.T) {
	chat := &recordingChat{}
	c := NewCategorizer(chat, nil, &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Voiture", Keywords: []string{"GARAGE"}}},
		CreditorMappings: map[string]string{"employeur sa": "Salaire"},
		DebtorMappings:   map[string]string{"coop city": "Alimentation"},
	}, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	tests := []struct {
		name       string
		tx         Transaction
		wantFound  bool
		wantName   string
		wantSource string
	}{
		{"debtor direct mapping", Transaction{PartyName: "COOP CITY", IsDebtor: true}, true, "Alimentation", "direct_mapping"},
		{"creditor direct mapping", Transaction{PartyName: "Employeur SA"}, true, "Salaire", "direct_mapping"},
		{"keyword", Transaction{PartyName: "Garage du Lac"}, true, "Voiture", "keyword"},
		{"keyword in info", Transaction{PartyName: "X", Info: "facture garage"}, true, "Voiture", "keyword"},
		{"no local match", Transaction{PartyName: "Inconnu SA"}, false, "", ""},
		{"empty party", Transaction{PartyName: "  "}, false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := c.CategorizeLocal(context.Background(), tt.tx)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantSource, got.Source)
		})
	}

	assert.Zero(t, chat.calls, "the AI client must never be consulted by CategorizeLocal")
}

// CategorizeLocal must not poison the batch cache: a later full categorization
// of the same party still has to run every tier.
func TestCategorizeLocal_DoesNotFillBatchCache(t *testing.T) {
	c := NewCategorizer(&recordingChat{}, nil, &store.MockCategoryStore{
		Categories:       []models.CategoryConfig{{Name: "Voiture", Keywords: []string{"GARAGE"}}},
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}, testLogger(), false, 0.70)
	t.Cleanup(c.Shutdown)

	_, found := c.CategorizeLocal(context.Background(), Transaction{PartyName: "Garage du Lac"})
	require.True(t, found)

	c.batchCacheMu.RLock()
	defer c.batchCacheMu.RUnlock()
	assert.Empty(t, c.batchCache)
}
