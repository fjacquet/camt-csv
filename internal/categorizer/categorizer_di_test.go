package categorizer

import (
	"context"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAIClient for testing dependency injection
type MockAIClient struct {
	CategorizeFunc   func(ctx context.Context, transaction models.Transaction) (models.Transaction, error)
	GetEmbeddingFunc func(ctx context.Context, text string) ([]float32, error)
}

func (m *MockAIClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	if m.CategorizeFunc != nil {
		return m.CategorizeFunc(ctx, transaction)
	}
	// Default behavior: return transaction with a mock category
	transaction.Category = "MockCategory"
	return transaction, nil
}

func (m *MockAIClient) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.GetEmbeddingFunc != nil {
		return m.GetEmbeddingFunc(ctx, text)
	}
	return []float32{0.0, 0.0}, nil
}

func TestCategorizer_DependencyInjection(t *testing.T) {
	// Create mock store and logger
	testStore := &store.CategoryStore{
		CategoriesFile: "testdata/categories.yaml",
		CreditorsFile:  "testdata/creditors.yaml",
		DebtorsFile:    "testdata/debtors.yaml",
	}
	testLogger := logging.NewLogrusAdapter("debug", "text")

	// Test with nil AI client
	cat1 := NewCategorizer(nil, nil, testStore, testLogger, true, 0.70)
	assert.NotNil(t, cat1)
	assert.Nil(t, cat1.aiClient)

	// Test with mock AI client
	mockAI := &MockAIClient{}
	cat2 := NewCategorizer(mockAI, mockAI, testStore, testLogger, true, 0.70)
	assert.NotNil(t, cat2)
	assert.NotNil(t, cat2.aiClient)

	// Test that both categorizers work independently
	transaction := Transaction{PartyName: "Unknown", IsDebtor: true}

	// First categorizer should return uncategorized (no AI)
	category1, err := cat1.CategorizeTransaction(context.Background(), transaction)
	require.NoError(t, err)
	assert.Equal(t, models.CategoryUncategorized, category1.Name)

	// Second categorizer should use AI
	category2, err := cat2.CategorizeTransaction(context.Background(), transaction)
	require.NoError(t, err)
	assert.Equal(t, "MockCategory", category2.Name)
}
